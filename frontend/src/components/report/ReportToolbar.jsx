import React from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  ArrowRight,
  Globe,
  Printer,
  FileDown,
  Sparkles,
  ExternalLink,
  Loader2,
  CheckCircle2,
} from 'lucide-react'

export default function ReportToolbar({
  report,
  lang,
  onLangChange,
  viewMode,
  onViewModeChange,
  onOpenOriginalNewTab,
  onBack,
  onPrint,
  hasOfficialPdf,
  isBackgroundLoading,
  onDownloadOfficialPdf,
}) {
  return (
    <Card className="border-slate-200 dark:border-slate-800 shadow-sm w-full bg-white dark:bg-slate-900 rounded-xl">
      <CardContent className="p-4 sm:p-5">
        <div className="flex flex-col xl:flex-row xl:items-center justify-between gap-4">
          {/* Vehicle & Title */}
          <div className="flex items-center gap-3">
            <Button
              variant="outline"
              size="sm"
              onClick={onBack}
              className="gap-1.5 text-xs font-semibold h-9 border-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer"
            >
              <ArrowRight className="h-4 w-4" />
              <span>العودة للبحث</span>
            </Button>
            <div>
              <div className="flex items-center gap-2 flex-wrap">
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
                {isBackgroundLoading ? (
                  <Badge className="text-[10px] font-bold bg-amber-100 text-amber-800 border-none flex items-center gap-1">
                    <Loader2 className="h-3 w-3 animate-spin" />
                    <span>جاري تجهيز الصفحة الأصلية والطباعة بصمت...</span>
                  </Badge>
                ) : (
                  hasOfficialPdf && (
                    <Badge className="text-[10px] font-bold bg-emerald-100 text-emerald-800 border-none flex items-center gap-1">
                      <CheckCircle2 className="h-3 w-3" />
                      <span>النسخة الكاملة والطباعة جاهزة</span>
                    </Badge>
                  )
                )}
              </div>
              <p className="text-xs text-slate-500 font-mono tracking-wider mt-0.5">
                VIN:{' '}
                <span className="text-[#003478] dark:text-blue-300 font-bold">{report?.vin}</span>
              </p>
            </div>
          </div>

          {/* Actions: View Mode Switch, Language Switch & Official Print PDF */}
          <div className="flex flex-wrap items-center gap-2">
            {/* 1. View Mode Toggle: Smart Design vs Full Carfax Page */}
            <div className="flex items-center bg-slate-100 dark:bg-slate-800 p-1 rounded-lg border border-slate-200 dark:border-slate-700 text-xs">
              <button
                type="button"
                onClick={() => onViewModeChange('smart')}
                className={`px-3 py-1.5 rounded-md transition-all font-bold flex items-center gap-1.5 cursor-pointer ${
                  viewMode === 'smart'
                    ? 'bg-[#003478] text-white shadow-xs'
                    : 'text-slate-600 dark:text-slate-300 hover:text-slate-900'
                }`}
              >
                <Sparkles className="h-3.5 w-3.5" />
                <span>تصميمنا السريع</span>
              </button>
              <button
                type="button"
                onClick={() => onViewModeChange('original')}
                className={`px-3 py-1.5 rounded-md transition-all font-bold flex items-center gap-1.5 cursor-pointer ${
                  viewMode === 'original'
                    ? 'bg-[#003478] text-white shadow-xs'
                    : 'text-slate-600 dark:text-slate-300 hover:text-slate-900'
                }`}
              >
                <ExternalLink className="h-3.5 w-3.5" />
                <span>صفحة كارفاكس الأصلية</span>
              </button>
            </div>

            {/* Open Full Original Carfax Page in New Browser Tab */}
            <Button
              variant="outline"
              size="sm"
              onClick={onOpenOriginalNewTab}
              className="gap-1.5 text-xs font-bold h-9 border-[#003478]/30 text-[#003478] hover:bg-blue-50 dark:text-blue-300 cursor-pointer"
            >
              <ExternalLink className="h-3.5 w-3.5" />
              <span>فتح الصفحة بنافذة مستقلة</span>
            </Button>

            {/* 2. Language Toggle */}
            <div className="flex items-center bg-slate-100 dark:bg-slate-800 p-1 rounded-lg border border-slate-200 dark:border-slate-700 text-xs">
              <button
                type="button"
                onClick={() => onLangChange('ar')}
                className={`px-3 py-1.5 rounded-md transition-all font-bold flex items-center gap-1.5 cursor-pointer ${
                  lang === 'ar'
                    ? 'bg-[#003478] text-white shadow-xs'
                    : 'text-slate-600 dark:text-slate-300 hover:text-slate-900'
                }`}
              >
                <Globe className="h-3.5 w-3.5" />
                <span>العربية</span>
              </button>
              <button
                type="button"
                onClick={() => onLangChange('en')}
                className={`px-3 py-1.5 rounded-md transition-all font-bold cursor-pointer ${
                  lang === 'en'
                    ? 'bg-[#003478] text-white shadow-xs'
                    : 'text-slate-600 dark:text-slate-300 hover:text-slate-900'
                }`}
              >
                English
              </button>
            </div>

            {/* 3. Official PDF Download / Print Button */}
            <Button
              onClick={hasOfficialPdf ? onDownloadOfficialPdf : onPrint}
              disabled={isBackgroundLoading && !hasOfficialPdf}
              className={`gap-1.5 text-xs font-bold h-9 text-white shadow-sm cursor-pointer ${
                hasOfficialPdf
                  ? 'bg-[#1C8237] hover:bg-[#15682b]'
                  : 'bg-[#003478] hover:bg-[#002657]'
              }`}
            >
              {isBackgroundLoading && !hasOfficialPdf ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span>جاري تجهيز الطباعة بصمت...</span>
                </>
              ) : hasOfficialPdf ? (
                <>
                  <FileDown className="h-4 w-4" />
                  <span>تحميل الطباعة الرسمية PDF (كندا)</span>
                </>
              ) : (
                <>
                  <Printer className="h-4 w-4" />
                  <span>طباعة التقرير ({lang === 'ar' ? 'عربي' : 'EN'})</span>
                </>
              )}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
