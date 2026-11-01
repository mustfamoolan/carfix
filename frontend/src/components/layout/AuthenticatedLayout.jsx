import React, { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/context/AuthContext'
import CarfaxBadgeLogo from '@/components/common/CarfaxBadgeLogo'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Toaster } from '@/components/ui/sonner'
import {
  History,
  Shield,
  LogOut,
  Menu,
  X,
  Activity,
  ChevronLeft,
  Radio,
} from 'lucide-react'

export default function AuthenticatedLayout({ children, title, subtitle }) {
  const { user, logout } = useAuth()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const location = useLocation()
  const navigate = useNavigate()

  // Focused Command Center Navigation (Users/Sales postponed per roadmap)
  const navItems = [
    {
      path: '/dashboard',
      label: 'غرفة عمليات كندا والفحص',
      icon: Radio,
      badge: 'Live',
    },
    {
      path: '/history',
      label: 'أرشيف التقارير المسحوبة',
      icon: History,
      badge: null,
    },
    {
      path: '/admin/logs',
      label: 'سجل عمليات السحب الحي',
      icon: Activity,
      badge: null,
    },
  ]

  const handleLogout = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  const SidebarContent = () => (
    <div className="flex h-full flex-col bg-white dark:bg-slate-900 border-l border-slate-200 dark:border-slate-800">
      {/* Brand Header */}
      <div className="flex h-16 items-center justify-between border-b border-slate-200 dark:border-slate-800 px-5 bg-slate-50/60 dark:bg-slate-900/60">
        <CarfaxBadgeLogo size="sm" subtitle="CANADA COMMAND CENTER" />
      </div>

      {/* Navigation Links */}
      <div className="flex-1 overflow-y-auto p-3 space-y-1">
        <p className="px-3 pb-2 pt-2 text-[11px] font-bold text-slate-400 dark:text-slate-500 uppercase tracking-wider text-right">
          غرفة التحكم والعمليات
        </p>
        {navItems.map((item) => {
          const Icon = item.icon
          const isActive =
            location.pathname === item.path ||
            (item.path !== '/dashboard' && location.pathname.startsWith(item.path))
          return (
            <Link
              key={item.path}
              to={item.path}
              onClick={() => setMobileMenuOpen(false)}
              className={`flex items-center justify-between rounded-lg px-3.5 py-2.5 text-xs font-semibold transition-all ${
                isActive
                  ? 'bg-[#003478] text-white shadow-sm'
                  : 'text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 hover:text-slate-900'
              }`}
            >
              <div className="flex items-center gap-2.5">
                <Icon className={`h-4 w-4 shrink-0 ${isActive ? 'text-white' : 'text-slate-500'}`} />
                <span>{item.label}</span>
              </div>
              {item.badge && (
                <span
                  className={`text-[10px] px-1.5 py-0.5 rounded font-bold ${
                    isActive
                      ? 'bg-white/20 text-white'
                      : 'bg-emerald-50 text-[#1C8237] dark:bg-emerald-950 dark:text-emerald-300'
                  }`}
                >
                  {item.badge}
                </span>
              )}
            </Link>
          )
        })}
      </div>

      {/* User Info / Logout Footer */}
      <div className="border-t border-slate-200 dark:border-slate-800 p-3 bg-slate-50/40 dark:bg-slate-900/40">
        <div className="flex items-center gap-3 rounded-lg p-2 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors">
          <Avatar className="h-8 w-8 border border-slate-200 dark:border-slate-700">
            <AvatarFallback className="bg-[#003478] text-white font-bold text-xs">
              {user?.username?.charAt(0)?.toUpperCase() || 'A'}
            </AvatarFallback>
          </Avatar>
          <div className="flex-1 min-w-0 text-right">
            <p className="text-xs font-bold text-slate-800 dark:text-slate-200 truncate">
              {user?.username || 'مدير العمليات'}
            </p>
            <div className="flex items-center gap-1 text-[11px] text-slate-500">
              <span className="inline-flex items-center text-[#003478] dark:text-blue-300 font-medium">
                <Shield className="h-3 w-3 ml-0.5" /> تحكم مباشر بكندا
              </span>
            </div>
          </div>
          <button
            onClick={handleLogout}
            title="تسجيل الخروج"
            className="text-slate-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-950/40 rounded p-1.5 transition-colors"
          >
            <LogOut className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  )

  return (
    <div className="flex min-h-screen bg-slate-50 dark:bg-slate-950 font-cairo text-slate-900 dark:text-slate-100" dir="rtl">
      {/* Desktop Sidebar (Fixed Right) */}
      <aside className="hidden lg:flex lg:flex-col lg:fixed lg:inset-y-0 lg:right-0 lg:w-64 z-30">
        <SidebarContent />
      </aside>

      {/* Mobile Drawer Backdrop */}
      {mobileMenuOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm lg:hidden transition-opacity"
          onClick={() => setMobileMenuOpen(false)}
        />
      )}

      {/* Mobile Drawer Sidebar */}
      <aside
        className={`fixed inset-y-0 right-0 z-50 w-72 flex flex-col bg-white dark:bg-slate-900 border-l border-slate-200 dark:border-slate-800 shadow-2xl transform transition-transform duration-200 ease-in-out lg:hidden ${
          mobileMenuOpen ? 'translate-x-0' : 'translate-x-full'
        }`}
      >
        <div className="flex h-14 items-center justify-between border-b border-slate-200 dark:border-slate-800 px-4 bg-slate-50 dark:bg-slate-900">
          <CarfaxBadgeLogo size="sm" subtitle="غرفة العمليات" />
          <button
            onClick={() => setMobileMenuOpen(false)}
            className="text-slate-500 hover:text-slate-800 dark:hover:text-slate-200 p-1"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto">
          <SidebarContent />
        </div>
      </aside>

      {/* Main Content Area */}
      <div className="flex flex-1 flex-col min-w-0 lg:pr-64 w-full">
        {/* Top Sticky Navbar */}
        <header className="sticky top-0 z-20 flex h-14 items-center justify-between border-b border-slate-200 dark:border-slate-800 bg-white/95 dark:bg-slate-900/95 backdrop-blur px-4 lg:px-8">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setMobileMenuOpen(true)}
              className="lg:hidden text-slate-600 hover:text-slate-900 dark:text-slate-300 dark:hover:text-white p-1 rounded-md hover:bg-slate-100 dark:hover:bg-slate-800"
            >
              <Menu className="h-5 w-5" />
            </button>
            <div className="flex items-center gap-2 text-xs text-slate-500">
              <span className="font-bold text-slate-800 dark:text-slate-200">{title || 'غرفة عمليات كندا'}</span>
              {subtitle && (
                <>
                  <ChevronLeft className="h-3 w-3 text-slate-400" />
                  <span className="text-slate-500 hidden sm:inline">{subtitle}</span>
                </>
              )}
            </div>
          </div>

          <div className="flex items-center gap-3">
            <Badge
              variant="outline"
              className="hidden sm:inline-flex border-slate-300 text-[#003478] bg-blue-50/60 dark:bg-slate-800 dark:text-blue-300 text-[11px] font-bold"
            >
              🍁 بوابة التحكم بحاسبة كندا
            </Badge>
            <button
              onClick={handleLogout}
              className="flex items-center gap-1.5 text-xs text-slate-500 hover:text-red-600 transition-colors px-2 py-1 rounded hover:bg-red-50 dark:hover:bg-red-950/40"
            >
              <LogOut className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">تسجيل الخروج</span>
            </button>
          </div>
        </header>

        {/* Page Body */}
        <main className="flex-1 w-full p-4 lg:p-8 space-y-6">
          {children}
        </main>
      </div>

      <Toaster position="top-center" dir="rtl" />
    </div>
  )
}
