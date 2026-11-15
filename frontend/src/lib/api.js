import { toast } from 'sonner'

// Lightweight API client with JWT bearer token + Fast In-Memory Stale-While-Revalidate Cache

const memoryCache = new Map()
const vinSpecsCache = new Map()

export function getCachedData(key) {
  const entry = memoryCache.get(key)
  if (!entry) return null
  return entry.data
}

export function setCachedData(key, data) {
  memoryCache.set(key, {
    data,
    timestamp: Date.now(),
  })
}

export function invalidateCache(prefix = '') {
  if (!prefix) {
    memoryCache.clear()
    return
  }
  for (const k of memoryCache.keys()) {
    if (k.startsWith(prefix)) {
      memoryCache.delete(k)
    }
  }
}

export function getCachedVinSpecs(vin) {
  return vinSpecsCache.get(vin) || null
}

export function setCachedVinSpecs(vin, specs) {
  vinSpecsCache.set(vin, specs)
}

export async function apiRequest(endpoint, options = {}) {
  const token = localStorage.getItem('carfix_token')
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const url = endpoint.startsWith('http')
    ? endpoint
    : `/api${endpoint.startsWith('/') ? endpoint : `/${endpoint}`}`

  const response = await fetch(url, {
    ...options,
    headers,
  })

  // Handle unauthorized gracefully
  if (response.status === 401 && !endpoint.includes('/auth/login')) {
    localStorage.removeItem('carfix_token')
    localStorage.removeItem('carfix_user')
    memoryCache.clear()
    window.dispatchEvent(new Event('auth:unauthorized'))
  }

  const contentType = response.headers.get('content-type')
  let data = null
  if (contentType && contentType.includes('application/json')) {
    data = await response.json()
  } else {
    data = await response.text()
  }

  if (!response.ok) {
    const errorMsg =
      data?.message ||
      data?.error ||
      (typeof data === 'string' ? data : 'حدث خطأ في الاتصال بالخادم')
    const error = new Error(errorMsg)
    error.status = response.status
    error.response = { data, status: response.status }
    error.data = data
    throw error
  }

  // Support both direct object access and .data access
  if (data && typeof data === 'object') {
    if (!('data' in data)) {
      data.data = data
    }
  }

  return data
}

export const api = {
  get: (url, options) => apiRequest(url, { ...options, method: 'GET' }),
  post: (url, body, options) =>
    apiRequest(url, { ...options, method: 'POST', body: JSON.stringify(body) }),
  put: (url, body, options) =>
    apiRequest(url, { ...options, method: 'PUT', body: JSON.stringify(body) }),
  delete: (url, options) => apiRequest(url, { ...options, method: 'DELETE' }),
}

// Viewed / Unread Report Tracking (persisted in localStorage)
const VIEWED_VINS_KEY = 'carfix_viewed_vins'
const UNVIEWED_VINS_KEY = 'carfix_unviewed_vins'

function readJsonStorage(key) {
  try {
    const raw = localStorage.getItem(key)
    return raw ? JSON.parse(raw) : {}
  } catch {
    return {}
  }
}

function writeJsonStorage(key, val) {
  try {
    localStorage.setItem(key, JSON.stringify(val))
  } catch {
    // Ignore storage quota errors
  }
}

export function markVinUnviewed(vin) {
  if (!vin) return
  const clean = String(vin).trim().toUpperCase()
  const unviewed = readJsonStorage(UNVIEWED_VINS_KEY)
  unviewed[clean] = Date.now()
  writeJsonStorage(UNVIEWED_VINS_KEY, unviewed)

  const viewed = readJsonStorage(VIEWED_VINS_KEY)
  if (viewed[clean]) {
    delete viewed[clean]
    writeJsonStorage(VIEWED_VINS_KEY, viewed)
  }
}

export function markVinViewed(vin) {
  if (!vin) return
  const clean = String(vin).trim().toUpperCase()
  const viewed = readJsonStorage(VIEWED_VINS_KEY)
  viewed[clean] = Date.now()
  writeJsonStorage(VIEWED_VINS_KEY, viewed)

  const unviewed = readJsonStorage(UNVIEWED_VINS_KEY)
  if (unviewed[clean]) {
    delete unviewed[clean]
    writeJsonStorage(UNVIEWED_VINS_KEY, unviewed)
  }
}

export function isVinUnread(itemOrVin) {
  const vin = typeof itemOrVin === 'string' ? itemOrVin : itemOrVin?.vin
  if (!vin) return false
  const clean = String(vin).trim().toUpperCase()
  const unviewed = readJsonStorage(UNVIEWED_VINS_KEY)
  if (unviewed[clean]) return true

  const viewed = readJsonStorage(VIEWED_VINS_KEY)
  if (viewed[clean]) return false

  // If item has a recent created_at (within last 12 hours) and was never opened in this browser, treat as new
  if (typeof itemOrVin === 'object' && itemOrVin?.created_at) {
    const createdMs = new Date(itemOrVin.created_at).getTime()
    if (!Number.isNaN(createdMs) && Date.now() - createdMs < 12 * 60 * 60 * 1000) {
      return true
    }
  }
  return false
}

// User / Search APIs with automatic cache population & invalidation
export const searchVin = async (vin) => {
  const cleanVin = String(vin).trim().toUpperCase()
  let res = null

  // Adaptive Polling: never give up while task is in queue or actively being fetched!
  // Checks up to 100 cycles (~15 minutes capacity for large queues)
  for (let cycle = 1; cycle <= 100; cycle++) {
    try {
      res = await api.post('/search', { vin: cleanVin })
      if (res?.status === 'processing') {
        await new Promise((r) => setTimeout(r, 1500))
        continue
      }
      break
    } catch (err) {
      if (err?.status === 499) {
        return { status: 'cancelled' }
      }
      // Do not retry on explicit client/auth/quota/validation errors (400, 401, 403, 404)
      if (err?.status === 400 || err?.status === 401 || err?.status === 403 || err?.status === 404) {
        throw err
      }
      // If a proxy/network blip or gateway timeout happened while the VIN is still queued or active, keep waiting smoothly
      try {
        const qStatus = await api.get(`/search/queue-status/${cleanVin}`)
        const data = qStatus?.data || qStatus
        if (
          data?.is_ready ||
          data?.status === 'ready' ||
          data?.is_active ||
          (data?.status === 'queued' && (data?.queue_position > 0 || data?.worker_online))
        ) {
          if (data?.is_ready) {
            // Re-run POST /search on the next loop iteration so quota is properly deducted and SearchLog is recorded!
            continue
          }
          await new Promise((r) => setTimeout(r, 2000))
          continue
        }
      } catch {
        // Status check had network blip, continue waiting if within safe limit
      }
      throw err
    }
  }

  if (res?.status === 'processing') {
    throw new Error('الطلب لا يزال في قائمة الانتظار، يرجى الاستمرار في الانتظار أو مراجعة السجل بعد قليل')
  }

  markVinUnviewed(cleanVin)
  invalidateCache('user:searches')
  invalidateCache('admin:logs')
  invalidateCache('admin:users')
  if (res?.report || res?.vin) {
    setCachedData(`report:${cleanVin}`, res?.report || res)
  }
  return res
}

export const getQueueStatus = (vin) => api.get(`/search/queue-status/${vin}`)

export const cancelQueuedSearches = (vins = []) => {
  const cleanVins = (Array.isArray(vins) ? vins : [vins])
    .map((v) => String(v || '').trim().toUpperCase())
    .filter(Boolean)
  if (cleanVins.length === 0) return

  const payload = JSON.stringify({ vins: cleanVins })
  try {
    if (typeof navigator !== 'undefined' && typeof navigator.sendBeacon === 'function') {
      const blob = new Blob([payload], { type: 'application/json' })
      if (navigator.sendBeacon('/api/search/cancel-queued', blob)) {
        return
      }
    }
  } catch {
    // Fallback to fetch with keepalive
  }
  try {
    fetch('/api/search/cancel-queued', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: payload,
      keepalive: true,
    }).catch(() => {})
  } catch {
    // Ignore unload network error
  }
}

// ============================================================================
// Global Persistent Pending Searches Store & Background Queue Runner
// Keeps yellow in-progress rows ("جاري السحب...") alive across SPA route navigation!
// ============================================================================
const PENDING_SEARCHES_KEY = 'carfix_pending_searches'
let globalPendingSearches = []
const pendingSearchListeners = new Set()
const activeSearchPromises = new Map()
let globalTickerInterval = null
let globalStatusPollInterval = null
let unloadListenerRegistered = false

function loadPersistedPendingSearches() {
  try {
    const raw = sessionStorage.getItem(PENDING_SEARCHES_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    // Discard stale items older than 15 minutes
    const now = Date.now()
    return parsed.filter((item) => {
      if (!item?.vin) return false
      const createdMs = item.created_at ? new Date(item.created_at).getTime() : now
      return Number.isNaN(createdMs) || now - createdMs < 15 * 60 * 1000
    })
  } catch {
    return []
  }
}

function savePersistedPendingSearches() {
  try {
    sessionStorage.setItem(PENDING_SEARCHES_KEY, JSON.stringify(globalPendingSearches))
  } catch {
    // ignore storage errors
  }
}

function notifyPendingListeners(event = null) {
  savePersistedPendingSearches()
  const snapshot = [...globalPendingSearches]
  for (const listener of pendingSearchListeners) {
    try {
      listener(snapshot, event)
    } catch {
      // ignore subscriber error
    }
  }
}

function ensureGlobalUnloadHandler() {
  if (unloadListenerRegistered || typeof window === 'undefined') return
  unloadListenerRegistered = true

  const handleBrowserClose = () => {
    const activeVins = globalPendingSearches.map((p) => p.vin).filter(Boolean)
    if (activeVins.length > 0) {
      cancelQueuedSearches(activeVins)
    }
  }

  window.addEventListener('pagehide', handleBrowserClose)
  window.addEventListener('beforeunload', handleBrowserClose)
}

function ensureGlobalPendingTimers() {
  ensureGlobalUnloadHandler()

  if (globalPendingSearches.length === 0) {
    if (globalTickerInterval) {
      clearInterval(globalTickerInterval)
      globalTickerInterval = null
    }
    if (globalStatusPollInterval) {
      clearInterval(globalStatusPollInterval)
      globalStatusPollInterval = null
    }
    return
  }

  if (!globalTickerInterval) {
    globalTickerInterval = setInterval(() => {
      if (globalPendingSearches.length === 0) {
        ensureGlobalPendingTimers()
        return
      }
      globalPendingSearches = globalPendingSearches.map((item) => ({
        ...item,
        estimated_seconds:
          item.estimated_seconds > 2 ? item.estimated_seconds - 1 : item.estimated_seconds,
      }))
      notifyPendingListeners()
    }, 1000)
  }

  if (!globalStatusPollInterval) {
    globalStatusPollInterval = setInterval(async () => {
      if (globalPendingSearches.length === 0) {
        ensureGlobalPendingTimers()
        return
      }
      const itemsSnapshot = [...globalPendingSearches]
      for (const item of itemsSnapshot) {
        try {
          const qRes = await getQueueStatus(item.vin)
          const info = qRes?.data || qRes
          if (!info) continue

          if (info.is_ready || info.status === 'ready') {
            markVinUnviewed(item.vin)
            invalidateCache('user:searches')
            invalidateCache('admin:logs')
            invalidateCache('admin:users')
            globalPendingSearches = globalPendingSearches.filter((c) => c.vin !== item.vin)
            ensureGlobalPendingTimers()
            notifyPendingListeners({ type: 'completed', vin: item.vin })
            continue
          }

          let changed = false
          globalPendingSearches = globalPendingSearches.map((c) => {
            if (c.vin !== item.vin) return c
            let stepAr = c.step_ar
            if (info.is_active) {
              stepAr = 'جاري سحب التقرير الآن...'
            } else if (info.queue_position > 0) {
              stepAr = `في الطابور (#${info.queue_position})`
            }
            const est =
              info.estimated_seconds > 0 &&
              (c.estimated_seconds === 0 ||
                Math.abs(c.estimated_seconds - info.estimated_seconds) > 4)
                ? info.estimated_seconds
                : c.estimated_seconds
            if (stepAr !== c.step_ar || est !== c.estimated_seconds) {
              changed = true
            }
            return {
              ...c,
              step_ar: stepAr,
              estimated_seconds: est,
            }
          })
          if (changed) {
            notifyPendingListeners()
          }
        } catch {
          // ignore poll blip
        }
      }
    }, 2000)
  }
}

export function getPendingSearches() {
  if (globalPendingSearches.length === 0) {
    const persisted = loadPersistedPendingSearches()
    if (persisted.length > 0) {
      globalPendingSearches = persisted
      ensureGlobalPendingTimers()
      // Resume background search promises if page was soft-reloaded
      for (const item of globalPendingSearches) {
        if (!activeSearchPromises.has(item.vin)) {
          runBackgroundSearchWorker(item.vin)
        }
      }
    }
  }
  return [...globalPendingSearches]
}

export function subscribePendingSearches(listener) {
  pendingSearchListeners.add(listener)
  // Immediately invoke with current state
  try {
    listener(getPendingSearches(), null)
  } catch {
    // ignore
  }
  return () => {
    pendingSearchListeners.delete(listener)
  }
}

async function runBackgroundSearchWorker(cleanVin) {
  if (activeSearchPromises.has(cleanVin)) {
    return activeSearchPromises.get(cleanVin)
  }

  const taskPromise = (async () => {
    try {
      const res = await searchVin(cleanVin)
      if (res?.status === 'cancelled') return res
      markVinUnviewed(cleanVin)
      invalidateCache('user:searches')
      invalidateCache('admin:logs')
      invalidateCache('admin:users')
      toast.success(
        res?.message || `اكتمل جلب تقرير (${cleanVin}) بنجاح — اضغط "عرض التقرير" من الجدول`
      )
      notifyPendingListeners({ type: 'completed', vin: cleanVin, response: res })
      return res
    } catch (err) {
      const msg =
        err?.response?.data?.message ||
        err?.data?.message ||
        err?.response?.data?.error ||
        err?.message ||
        `تعذر استخراج التقرير للشاصي (${cleanVin})`
      toast.error(msg)
      notifyPendingListeners({ type: 'error', vin: cleanVin, error: err })
      throw err
    } finally {
      activeSearchPromises.delete(cleanVin)
      globalPendingSearches = globalPendingSearches.filter((p) => p.vin !== cleanVin)
      ensureGlobalPendingTimers()
      notifyPendingListeners({ type: 'settled', vin: cleanVin })
    }
  })()

  activeSearchPromises.set(cleanVin, taskPromise)
  return taskPromise
}

export function enqueueGlobalVinSearch(vin, initialTitle = '') {
  const cleanVin = String(vin || '').trim().toUpperCase()
  if (!cleanVin) return

  // Ensure loaded
  getPendingSearches()

  if (globalPendingSearches.some((p) => p.vin === cleanVin)) {
    toast.info(`الشاصي (${cleanVin}) قيد الجلب حالياً في الجدول أدناه`)
    return
  }

  const pendingItem = {
    id: `pending-${cleanVin}-${Date.now()}`,
    vin: cleanVin,
    vehicle_title: initialTitle || 'جاري جلب بيانات المركبة...',
    result_status: 'processing',
    status: 'processing',
    step_ar: 'جاري طلب وسحب التقرير...',
    estimated_seconds: 12 + globalPendingSearches.length * 6,
    created_at: new Date().toISOString(),
  }

  globalPendingSearches = [pendingItem, ...globalPendingSearches]
  ensureGlobalPendingTimers()
  notifyPendingListeners({ type: 'added', vin: cleanVin })

  if (!initialTitle) {
    const cachedSpecs = getCachedVinSpecs(cleanVin)
    if (cachedSpecs?.title) {
      globalPendingSearches = globalPendingSearches.map((c) =>
        c.vin === cleanVin ? { ...c, vehicle_title: cachedSpecs.title } : c
      )
      notifyPendingListeners()
    } else {
      fetch(`https://vpic.nhtsa.dot.gov/api/vehicles/DecodeVinValues/${cleanVin}?format=json`)
        .then((r) => r.json())
        .then((data) => {
          const item = data?.Results?.[0]
          if (item?.Make && item?.ModelYear) {
            const decodedTitle = `${item.ModelYear} ${item.Make} ${item.Model || item.Series || ''}`
              .replace(/\s+/g, ' ')
              .trim()
            globalPendingSearches = globalPendingSearches.map((c) =>
              c.vin === cleanVin ? { ...c, vehicle_title: decodedTitle } : c
            )
            notifyPendingListeners()
          }
        })
        .catch(() => {})
    }
  }

  runBackgroundSearchWorker(cleanVin).catch(() => {})
}

export const getReport = async (vin) => {
  const key = `report:${String(vin).toUpperCase()}`
  const res = await api.get(`/reports/${vin}`)
  const reportObj = res?.report || res?.data?.report || res
  if (reportObj) {
    setCachedData(key, reportObj)
  }
  return res
}

export const getReportHtml = (vin, lang = 'en') =>
  api.get(`/reports/${vin}/html?lang=${lang}`)

export const getUserSearches = async (params = {}) => {
  const qs = new URLSearchParams()
  if (params.page) qs.set('page', params.page)
  if (params.limit) qs.set('limit', params.limit)
  if (params.q) qs.set('q', params.q)
  const queryStr = qs.toString()
  const key = `user:searches:${queryStr}`
  const res = await api.get(`/user/searches${queryStr ? `?${queryStr}` : ''}`)
  setCachedData(key, res)
  if (!queryStr) {
    setCachedData('user:searches:', res)
  }
  return res
}

// Admin APIs with automatic cache population & invalidation
export const getAdminUsers = async (search = '') => {
  const key = `admin:users:${search}`
  const res = await api.get(
    `/admin/users${search ? `?q=${encodeURIComponent(search)}` : ''}`
  )
  const list = Array.isArray(res)
    ? res
    : res?.users || res?.data?.users || (Array.isArray(res?.data) ? res.data : [])
  setCachedData(key, list)
  return res
}

export const createAdminUser = async (userData) => {
  const res = await api.post('/admin/users', userData)
  invalidateCache('admin:users')
  return res
}

export const updateAdminUser = async (id, userData) => {
  const res = await api.put(`/admin/users/${id}`, userData)
  invalidateCache('admin:users')
  return res
}

export const deleteAdminUser = async (id) => {
  const res = await api.delete(`/admin/users/${id}`)
  invalidateCache('admin:users')
  return res
}

export const updateUserQuota = async (id, quotaData) => {
  const res = await api.put(`/admin/users/${id}/quota`, quotaData)
  invalidateCache('admin:users')
  return res
}

export const getAdminSearchLogs = async (params = {}) => {
  const qs = new URLSearchParams()
  if (params.page) qs.set('page', params.page)
  if (params.limit) qs.set('limit', params.limit)
  if (params.vin) qs.set('vin', params.vin)
  if (params.result_status) qs.set('result_status', params.result_status)
  const queryStr = qs.toString()
  const key = `admin:logs:${queryStr}`
  const res = await api.get(`/admin/search-logs${queryStr ? `?${queryStr}` : ''}`)
  setCachedData(key, res)
  return res
}

// Queue Management APIs
export const getAdminQueue = () => api.get('/admin/worker/queue-details')
export const removeAdminQueueTask = (vin) => api.delete(`/admin/worker/queue/${vin}`)
export const promoteAdminQueueTask = (vin) => api.post(`/admin/worker/queue/${vin}/promote`, {})
export const purgeAdminQueue = () => api.post('/admin/worker/queue/purge', {})

// Multi-Node Workers APIs
export const getAdminWorkerStatus = () => api.get('/admin/worker-status')
export const requestInspectNode = (nodeId = '') =>
  api.post('/admin/worker-inspect', { node_id: nodeId })
export const stopWorkerNode = (nodeId = '') =>
  api.post('/admin/worker-stop', { node_id: nodeId })
export const resetWorkerNode = (nodeId = '') =>
  api.post('/admin/worker-reset', { node_id: nodeId })
export const deleteWorkerNode = (nodeId) =>
  api.delete(`/admin/worker/nodes/${nodeId}`)
export const setWorkerFallbackMode = (enabled) =>
  api.post('/admin/worker-mode', { use_mock_fallback: enabled })
