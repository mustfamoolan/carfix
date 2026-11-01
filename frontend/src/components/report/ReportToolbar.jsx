import React from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ArrowRight, Globe, Printer, FileCheck } from 'lucide-react'

export default function ReportToolbar({
  report,
  lang,
  onLangChange,
  onBack,
  onPrint,
}) {
  return (
    <Card className="border-slate-200 dark:border-slate-800 shadow-sm w-full bg-white dark:bg-slate-900 rounded-xl">
      <CardContent className="p-4 sm:p-5">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          {/* Vehicle & Title */}
          <div className="flex items-center gap-3">
            <Button
              variant="outline"
              size="sm"
              onClick={onBack}
              className="gap-1.5 text-xs font-semibold h-9 border-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800"
            >
              <ArrowRight className="h-4 w-4" />
              <span>العودة للبحث</span>
            </Button>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-base sm:text-lg font-bold text-slate-800 dark:text-slate-100 truncate max-w-sm sm:max-w-md">
                  {report?.vehicle_title || 'تقرير فحص المركبة الرسمي'}
                </h1>
                {report?.fetch_source && (
                  <Badge
                    className={`text-[10px] font-bold border-none ${
                      report.fetch_source === 'cache'
                        ? 'bg-[#EBF7EE] text-[#1C8237] dark:bg-emerald-950 dark:text-emerald-300'
                        : 'bg-blue-100 text-[#003478] dark:bg-blue-950 dark:text-blue-300'
                    }`}
                  >
                    {report.fetch_source === 'cache' ? 'كاش فوري' : 'سحب مباشر'}
                  </Badge>
                )}
              </div>
              <p className="text-xs text-slate-500 font-mono tracking-wider mt-0.5">
                VIN: <span className="text-[#003478] dark:text-blue-300 font-bold">{report?.vin}</span>
              </p>
            </div>
          </div>

          {/* Actions: Language Switch & Print */}
          <div className="flex items-center gap-2">
            <div className="flex items-center bg-slate-100 dark:bg-slate-800 p-1 rounded-lg border border-slate-200 dark:border-slate-700 text-xs">
              <button
                type="button"
                onClick={() => onLangChange('ar')}
                className={`px-3 py-1 rounded-md transition-all font-bold flex items-center gap-1.5 ${
                  lang === 'ar'
                    ? 'bg-[#003478] text-white shadow-xs'
                    : 'text-slate-600 dark:text-slate-300 hover:text-slate-900'
                }`}
              >
                <Globe className="h-3.5 w-3.5" />
                العربية
              </button>
              <button
                type="button"
                onClick={() => onLangChange('en')}
                className={`px-3 py-1 rounded-md transition-all font-bold ${
                  lang === 'en'
                    ? 'bg-[#003478] text-white shadow-xs'
                    : 'text-slate-600 dark:text-slate-300 hover:text-slate-900'
                }`}
              >
                English (Original)
              </button>
            </div>

            <Button
              onClick={onPrint}
              className="gap-1.5 text-xs font-bold h-9 bg-[#003478] hover:bg-[#002657] text-white shadow-sm"
            >
              <Printer className="h-4 w-4" />
              <span>طباعة / حفظ PDF</span>
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
