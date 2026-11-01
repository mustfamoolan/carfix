import React, { createContext, useContext, useState, useEffect } from 'react'
import { api } from '../lib/api'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => localStorage.getItem('carfix_token'))
  const [user, setUser] = useState(() => {
    try {
      const saved = localStorage.getItem('carfix_user')
      return saved ? JSON.parse(saved) : null
    } catch {
      return null
    }
  })
  // If we already have token and user in localStorage, don't show full-page loading spinner
  const [loading, setLoading] = useState(() => {
    const savedToken = localStorage.getItem('carfix_token')
    const savedUser = localStorage.getItem('carfix_user')
    return !!savedToken && !savedUser
  })

  const refreshUser = async () => {
    const currentToken = localStorage.getItem('carfix_token')
    if (!currentToken) {
      setUser(null)
      setLoading(false)
      return null
    }
    try {
      const res = await api.get('/auth/me')
      const userData = res?.user || res?.data?.user || res
      setUser(userData)
      localStorage.setItem('carfix_user', JSON.stringify(userData))
      return userData
    } catch (err) {
      console.warn('Failed to refresh user:', err)
      if (err.status === 401 || err.status === 403) {
        logout()
      }
      return null
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    refreshUser()

    const handleUnauthorized = () => {
      setToken(null)
      setUser(null)
    }

    window.addEventListener('auth:unauthorized', handleUnauthorized)
    return () => window.removeEventListener('auth:unauthorized', handleUnauthorized)
  }, [])

  const login = async (username, password) => {
    const res = await api.post('/auth/login', { username, password })
    const authToken = res?.token || res?.data?.token
    const authUser = res?.user || res?.data?.user
    if (authToken) {
      setToken(authToken)
      localStorage.setItem('carfix_token', authToken)
    }
    if (authUser) {
      setUser(authUser)
      localStorage.setItem('carfix_user', JSON.stringify(authUser))
    }
    return { token: authToken, user: authUser, raw: res }
  }

  const logout = async () => {
    try {
      await api.post('/auth/logout', {})
    } catch {
      // Ignore network failures on logout
    } finally {
      setToken(null)
      setUser(null)
      localStorage.removeItem('carfix_token')
      localStorage.removeItem('carfix_user')
    }
  }

  const isAdmin = user?.role === 'admin'

  return (
    <AuthContext.Provider
      value={{
        token,
        user,
        quota: user?.quota,
        loading,
        isAdmin,
        login,
        logout,
        refreshUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
