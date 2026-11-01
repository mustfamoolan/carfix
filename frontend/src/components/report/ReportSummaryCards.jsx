import React from 'react'
import { Card } from '@/components/ui/card'
import { Award, Users, AlertTriangle, Gauge, CheckCircle2 } from 'lucide-react'

export default function ReportSummaryCards({ summary }) {
  if (!summary) return null

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-3 w-full">
      {summary.carfax_score !== undefined && (
        <Card className="p-3.5 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-sm">
          <div className="flex items-center justify-between text-xs text-slate-500 mb-1">
            <span>تقييم كارفاكس</span>
            <Award className="h-4 w-4 text-[#00599C]" />
          </div>
          <p className="text-xl font-bold text-slate-900 dark:text-slate-100 font-mono">
            {summary.carfax_score}
          </p>
        </Card>
      )}
      {summary.total_owners !== undefined && (
        <Card className="p-3.5 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-sm">
          <div className="flex items-center justify-between text-xs text-slate-500 mb-1">
            <span>عدد الملاك السابقين</span>
            <Users className="h-4 w-4 text-slate-400" />
          </div>
          <p className="text-xl font-bold text-slate-900 dark:text-slate-100">
            {summary.total_owners} {summary.total_owners === 1 ? 'مالك واحد' : 'ملاك'}
          </p>
        </Card>
      )}
      {summary.accidents_reported !== undefined && (
        <Card className="p-3.5 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-sm">
          <div className="flex items-center justify-between text-xs text-slate-500 mb-1">
            <span>سجل الحوادث</span>
            {summary.accidents_reported > 0 ? (
              <AlertTriangle className="h-4 w-4 text-[#D0202E]" />
            ) : (
              <CheckCircle2 className="h-4 w-4 text-[#1C8237]" />
            )}
          </div>
          <p
            className={`text-xl font-bold ${
              summary.accidents_reported > 0 ? 'text-[#D0202E]' : 'text-[#1C8237]'
            }`}
          >
            {summary.accidents_reported > 0
              ? `${summary.accidents_reported} حادث مسجل`
              : 'خالية من الحوادث'}
          </p>
        </Card>
      )}
      {summary.last_reported_odometer && (
        <Card className="p-3.5 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-sm">
          <div className="flex items-center justify-between text-xs text-slate-500 mb-1">
            <span>آخر قراءة للعداد</span>
            <Gauge className="h-4 w-4 text-slate-400" />
          </div>
          <p className="text-xl font-bold text-slate-900 dark:text-slate-100 font-mono">
            {summary.last_reported_odometer}
          </p>
        </Card>
      )}
    </div>
  )
}
