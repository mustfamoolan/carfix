import React, { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/context/AuthContext'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Toaster } from '@/components/ui/sonner'
import {
  LayoutDashboard,
  History,
  Users,
  Activity,
  Shield,
  User as UserIcon,
  LogOut,
  Menu,
  X,
  Car,
} from 'lucide-react'

export default function AuthenticatedLayout({ children, title }) {
  const { user, logout } = useAuth()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const location = useLocation()
  const navigate = useNavigate()

  const isAdmin = user?.role === 'admin'

  const navItems = [
    {
      path: '/dashboard',
      label: 'فحص الشاصي',
      icon: LayoutDashboard,
    },
    {
      path: '/history',
      label: 'التقارير المسحوبة',
      icon: History,
    },
    ...(isAdmin
      ? [
          {
            path: '/admin/users',
            label: 'إدارة المستخدمين',
            icon: Users,
          },
          {
            path: '/admin/logs',
            label: 'سجل العمليات',
            icon: Activity,
          },
        ]
      : []),
  ]

  const handleLogout = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  const SidebarContent = () => (
    <>
      {/* Logo */}
      <div className="flex h-14 items-center border-b border-border px-5">
        <Link to="/dashboard" className="flex items-center gap-2.5">
          <img
            src="/carfax-assets/carfax-logo.svg"
            alt="CARFAX"
            className="h-5 w-auto object-contain"
          />
          <span className="font-bold text-xs text-muted-foreground border-r border-border pr-2">
            بوابة التجار
          </span>
        </Link>
      </div>

      {/* Navigation */}
      <div className="flex flex-1 flex-col gap-1 p-3 overflow-y-auto">
        <p className="px-3 pb-1 pt-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider text-right">
          القائمة الرئيسية
        </p>
        {navItems.map((item) => {
          const Icon = item.icon
          const active =
            location.pathname === item.path ||
            (item.path !== '/dashboard' && location.pathname.startsWith(item.path))
          return (
            <Link
              key={item.path}
              to={item.path}
              onClick={() => setSidebarOpen(false)}
              className={`flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors ${
                active
                  ? 'bg-primary text-primary-foreground font-medium'
                  : 'text-muted-foreground hover:bg-secondary/70 hover:text-foreground'
              }`}
            >
              <Icon className="h-4 w-4 shrink-0" />
              {item.label}
            </Link>
          )
        })}
      </div>

      {/* User footer */}
      <div className="border-t border-border p-3">
        <div className="flex items-center gap-3 rounded-md px-3 py-2">
          <Avatar className="h-8 w-8">
            <AvatarFallback className="text-xs bg-primary/10 text-primary font-semibold">
              {user?.username?.charAt(0)?.toUpperCase() || 'A'}
            </AvatarFallback>
          </Avatar>
          <div className="flex-1 min-w-0 text-right">
            <p className="text-sm font-medium truncate">
              {user?.full_name || user?.username || 'المدير'}
            </p>
            <p className="text-xs text-muted-foreground truncate">@{user?.username || 'admin'}</p>
          </div>
          <button
            onClick={handleLogout}
            title="تسجيل الخروج"
            className="text-muted-foreground hover:text-foreground transition-colors p-1 cursor-pointer"
          >
            <LogOut className="h-4 w-4" />
          </button>
        </div>
      </div>
    </>
  )

  return (
    <div className="flex min-h-screen bg-background text-foreground" dir="rtl">
      {/* Desktop Sidebar */}
      <aside className="hidden lg:flex lg:flex-col lg:fixed lg:inset-y-0 lg:right-0 lg:w-60 border-l border-border bg-background z-20">
        <SidebarContent />
      </aside>

      {/* Mobile overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-30 bg-black/50 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Mobile Sidebar */}
      <aside
        className={`fixed inset-y-0 right-0 z-40 w-64 flex flex-col border-l border-border bg-background transform transition-transform duration-200 ease-in-out lg:hidden ${
          sidebarOpen ? 'translate-x-0' : 'translate-x-full'
        }`}
      >
        <div className="flex h-14 items-center justify-between border-b border-border px-4">
          <span className="font-semibold text-sm">القائمة</span>
          <button
            onClick={() => setSidebarOpen(false)}
            className="text-muted-foreground hover:text-foreground"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="flex flex-1 flex-col overflow-y-auto">
          <SidebarContent />
        </div>
      </aside>

      {/* Main content */}
      <div className="flex flex-1 flex-col min-w-0 lg:pr-60 w-full">
        {/* Top header */}
        <header className="sticky top-0 z-10 flex h-14 items-center gap-4 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 px-4 lg:px-6">
          <button
            className="lg:hidden text-muted-foreground hover:text-foreground"
            onClick={() => setSidebarOpen(true)}
          >
            <Menu className="h-5 w-5" />
          </button>

          {/* Breadcrumb */}
          <div className="flex-1">
            <nav className="flex items-center gap-1.5 text-sm text-muted-foreground justify-start">
              <span>الرئيسية</span>
              {title && (
                <>
                  <span>/</span>
                  <span className="font-medium text-foreground">{title}</span>
                </>
              )}
            </nav>
          </div>

          {/* Header actions */}
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-2">
              <div className="hidden sm:block text-right">
                <p className="text-sm font-medium leading-none">
                  {user?.full_name || user?.username}
                </p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  {isAdmin ? (
                    <span className="flex items-center gap-1 justify-end">
                      <Shield className="h-3 w-3 text-primary" /> أدمن
                    </span>
                  ) : (
                    <span className="flex items-center gap-1 justify-end">
                      <UserIcon className="h-3 w-3" /> مستخدم
                    </span>
                  )}
                </p>
              </div>
              <Avatar className="h-8 w-8">
                <AvatarFallback className="text-xs bg-primary/10 text-primary font-semibold">
                  {user?.username?.charAt(0)?.toUpperCase() || 'A'}
                </AvatarFallback>
              </Avatar>
            </div>

            <button
              onClick={handleLogout}
              title="تسجيل الخروج"
              className="text-muted-foreground hover:text-foreground transition-colors p-1 cursor-pointer"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 p-4 lg:p-6 space-y-6">{children}</main>
      </div>

      <Toaster position="top-center" dir="rtl" />
    </div>
  )
}
