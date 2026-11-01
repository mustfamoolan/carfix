// Lightweight API client with automatic JWT bearer token attachment

export async function apiRequest(endpoint, options = {}) {
  const token = localStorage.getItem('carfix_token')
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const url = endpoint.startsWith('http') ? endpoint : `/api${endpoint.startsWith('/') ? endpoint : `/${endpoint}`}`

  const response = await fetch(url, {
    ...options,
    headers,
  })

  // Handle unauthorized gracefully
  if (response.status === 401 && !endpoint.includes('/auth/login')) {
    localStorage.removeItem('carfix_token')
    localStorage.removeItem('carfix_user')
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
    const errorMsg = data?.message || data?.error || (typeof data === 'string' ? data : 'حدث خطأ في الاتصال بالخادم')
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
  post: (url, body, options) => apiRequest(url, { ...options, method: 'POST', body: JSON.stringify(body) }),
  put: (url, body, options) => apiRequest(url, { ...options, method: 'PUT', body: JSON.stringify(body) }),
  delete: (url, options) => apiRequest(url, { ...options, method: 'DELETE' }),
}

// User / Search APIs
export const searchVin = (vin) => api.post('/search', { vin })
export const getReport = (vin) => api.get(`/reports/${vin}`)
export const getReportHtml = (vin, lang = 'en') => api.get(`/reports/${vin}/html?lang=${lang}`)
export const getUserSearches = () => api.get('/user/searches')

// Admin APIs
export const getAdminUsers = () => api.get('/admin/users')
export const createAdminUser = (userData) => api.post('/admin/users', userData)
export const updateAdminUser = (id, userData) => api.put(`/admin/users/${id}`, userData)
export const updateUserQuota = (id, quotaData) => api.put(`/admin/users/${id}/quota`, quotaData)
export const getAdminSearchLogs = () => api.get('/admin/search-logs')
