import React from 'react'
import { Card } from '@/components/ui/card'
import { Users, UserCheck, Shield } from 'lucide-react'

export default function AdminStatsCards({ users = [] }) {
  const totalUsers = users.length
  const activeUsers = users.filter((u) => u.is_active).length
  const adminUsers = users.filter((u) => u.role === 'admin').length

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 sm:gap-4 w-full">
      <Card className="p-4 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-sm">
        <div className="flex items-center justify-between text-xs font-semibold text-slate-500">
          <span>إجمالي الحسابات المسجلة</span>
          <Users className="h-4 w-4 text-[#003478]" />
        </div>
        <p className="text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100 mt-1">
          {totalUsers}
        </p>
      </Card>
      <Card className="p-4 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-sm">
        <div className="flex items-center justify-between text-xs font-semibold text-slate-500">
          <span>الحسابات النشطة والمفعلة</span>
          <UserCheck className="h-4 w-4 text-[#1C8237]" />
        </div>
        <p className="text-2xl font-bold tracking-tight text-[#1C8237] mt-1">
          {activeUsers}
        </p>
      </Card>
      <Card className="p-4 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-sm col-span-2 sm:col-span-1">
        <div className="flex items-center justify-between text-xs font-semibold text-slate-500">
          <span>المسؤولون والمدراء (Admins)</span>
          <Shield className="h-4 w-4 text-[#00599C]" />
        </div>
        <p className="text-2xl font-bold tracking-tight text-[#003478] dark:text-blue-300 mt-1">
          {adminUsers}
        </p>
      </Card>
    </div>
  )
}
