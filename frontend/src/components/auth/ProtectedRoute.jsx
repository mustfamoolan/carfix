import React from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/context/AuthContext'
import { Car } from 'lucide-react'

export default function ProtectedRoute({ children }) {
  const { token, loading } = useAuth()
  const location = useLocation()

  if (loading) {
    return (
      <div className="min-h-screen bg-background flex flex-col items-center justify-center text-foreground font-cairo" dir="rtl">
        <div className="w-12 h-12 rounded-xl bg-[#003478] text-white flex items-center justify-center shadow-md animate-pulse mb-3">
          <Car className="w-6 h-6" />
        </div>
        <p className="text-muted-foreground text-xs font-semibold">
          جاري التحقق من الجلسة...
        </p>
      </div>
    )
  }

  if (!token) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return children
}
