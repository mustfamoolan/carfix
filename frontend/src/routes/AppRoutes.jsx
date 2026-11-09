import React from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from '@/context/AuthContext'
import ProtectedRoute from '@/components/auth/ProtectedRoute'
import LoginView from '@/views/LoginView'
import DashboardView from '@/views/DashboardView'
import HistoryView from '@/views/HistoryView'
import ReportView from '@/views/ReportView'
import AdminUsersView from '@/views/AdminUsersView'
import AdminLogsView from '@/views/AdminLogsView'
import AdminQueueView from '@/views/AdminQueueView'
import AdminCopartView from '@/views/AdminCopartView'

export default function AppRoutes() {
  const { token } = useAuth()

  return (
    <Routes>
      <Route
        path="/login"
        element={token ? <Navigate to="/dashboard" replace /> : <LoginView />}
      />

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
        path="/admin/users"
        element={
          <ProtectedRoute>
            <AdminUsersView />
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

      <Route
        path="/admin/queue"
        element={
          <ProtectedRoute>
            <AdminQueueView />
          </ProtectedRoute>
        }
      />

      <Route
        path="/admin/copart"
        element={
          <ProtectedRoute>
            <AdminCopartView />
          </ProtectedRoute>
        }
      />

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
