import React from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from '@/context/AuthContext'
import ProtectedRoute from '@/components/auth/ProtectedRoute'
import LoginView from '@/views/LoginView'
import DashboardView from '@/views/DashboardView'
import HistoryView from '@/views/HistoryView'
import ReportView from '@/views/ReportView'
import AdminLogsView from '@/views/AdminLogsView'

export default function AppRoutes() {
  const { token } = useAuth()

  return (
    <Routes>
      {/* Public Route: Login */}
      <Route
        path="/login"
        element={
          token ? (
            <Navigate to="/dashboard" replace />
          ) : (
            <LoginView />
          )
        }
      />

      {/* Main Canada Command & Control Dashboard */}
      <Route
        path="/dashboard"
        element={
          <ProtectedRoute>
            <DashboardView />
          </ProtectedRoute>
        }
      />

      <Route
        path="/history"
        element={
          <ProtectedRoute>
            <HistoryView />
          </ProtectedRoute>
        }
      />

      <Route
        path="/report/:vin"
        element={
          <ProtectedRoute>
            <ReportView />
          </ProtectedRoute>
        }
      />

      <Route
        path="/admin/logs"
        element={
          <ProtectedRoute>
            <AdminLogsView />
          </ProtectedRoute>
        }
      />

      {/* Root & Catch-all Redirect to Command Center */}
      <Route
        path="/"
        element={<Navigate to={token ? '/dashboard' : '/login'} replace />}
      />

      <Route
        path="*"
        element={<Navigate to={token ? '/dashboard' : '/login'} replace />}
      />
    </Routes>
  )
}
