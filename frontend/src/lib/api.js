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
          if (data?.is_ready && data?.report) {
            res = data.report
            break
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

  invalidateCache('user:searches')
  invalidateCache('admin:logs')
  invalidateCache('admin:users')
  if (res?.report || res?.vin) {
    setCachedData(`report:${cleanVin}`, res?.report || res)
  }
  return res
}

export const getQueueStatus = (vin) => api.get(`/search/queue-status/${vin}`)

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
