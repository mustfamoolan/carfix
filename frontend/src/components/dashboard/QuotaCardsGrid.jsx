import React from 'react'
import { Card } from '@/components/ui/card'
import { Layers, Database, CheckCircle2, Calendar } from 'lucide-react'

export default function QuotaCardsGrid({ quota = {} }) {
  const totalLimit = quota.total_limit ?? 0
  const usedTotal = quota.used_total ?? 0
  const remainingTotal = quota.remaining_total ?? 0
  const remainingDaily = quota.remaining_daily ?? 0
  const dailyLimit = quota.daily_limit ?? 0

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4 w-full">
      {/* Total Limit */}
      <Card className="p-4 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-sm">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-slate-500">الرصيد الكلي للباقة</span>
          <div className="h-8 w-8 rounded-lg bg-blue-50 dark:bg-blue-950/60 flex items-center justify-center text-[#003478] dark:text-blue-300">
            <Layers className="h-4 w-4" />
          </div>
        </div>
        <div className="mt-2 flex items-baseline gap-1">
          <span className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            {totalLimit}
          </span>
          <span className="text-xs text-slate-500 font-medium">فحص</span>
        </div>
      </Card>

      {/* Used Total */}
      <Card className="p-4 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-sm">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-slate-500">إجمالي المستهلك</span>
          <div className="h-8 w-8 rounded-lg bg-slate-100 dark:bg-slate-800 flex items-center justify-center text-slate-600 dark:text-slate-400">
            <Database className="h-4 w-4" />
          </div>
        </div>
        <div className="mt-2 flex items-baseline gap-1">
          <span className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            {usedTotal}
          </span>
          <span className="text-xs text-slate-500 font-medium">فحص</span>
        </div>
      </Card>

      {/* Remaining Total */}
      <Card className="p-4 bg-emerald-50/50 dark:bg-emerald-950/20 border-emerald-200 dark:border-emerald-800/40 shadow-sm">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-emerald-800 dark:text-emerald-300">الرصيد المتبقي</span>
          <div className="h-8 w-8 rounded-lg bg-emerald-100 dark:bg-emerald-900/60 text-[#1C8237] dark:text-emerald-300 flex items-center justify-center">
            <CheckCircle2 className="h-4 w-4" />
          </div>
        </div>
        <div className="mt-2 flex items-baseline gap-1">
          <span className="text-xl sm:text-2xl font-bold tracking-tight text-[#1C8237] dark:text-emerald-300">
            {remainingTotal}
          </span>
          <span className="text-xs text-emerald-700/80 font-medium">فحص</span>
        </div>
      </Card>

      {/* Remaining Today */}
      <Card className="p-4 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-sm">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-slate-500">المتبقي لليوم</span>
          <div className="h-8 w-8 rounded-lg bg-amber-50 dark:bg-amber-950/60 flex items-center justify-center text-amber-600 dark:text-amber-400">
            <Calendar className="h-4 w-4" />
          </div>
        </div>
        <div className="mt-2 flex items-baseline gap-1">
          <span className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            {remainingDaily}
          </span>
          <span className="text-xs text-slate-500 font-medium">
            من أصل {dailyLimit}
          </span>
        </div>
      </Card>
    </div>
  )
}
