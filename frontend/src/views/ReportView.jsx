import React, { useState, useEffect, useRef, useMemo } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import ReportToolbar from '@/components/report/ReportToolbar'
import ReportSummaryCards from '@/components/report/ReportSummaryCards'
import ReportNativeView from '@/components/report/ReportNativeView'
import ReportFrame from '@/components/report/ReportFrame'
import { getReport, getReportHtml } from '@/lib/api'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'

export default function ReportView() {
  const { vin } = useParams()
  const navigate = useNavigate()
  const location = useLocation()

  const [report, setReport] = useState(location.state?.report || null)
  const [loading, setLoading] = useState(!report)
  const [viewMode, setViewMode] = useState('smart') // 'smart' (Fast Native Design) | 'original' (Full Carfax Page)
  const [lang, setLang] = useState('ar') // Instant Arabic/English toggle
  const [htmlCache, setHtmlCache] = useState({})
  const [htmlLoading, setHtmlLoading] = useState(false)
  const iframeRef = useRef(null)

  useEffect(() => {
    if (vin) {
      // Always fetch or refresh lightweight report metadata (~12 KB) if not already loaded for this VIN
      if (!report || (report.vin && report.vin.toUpperCase() !== vin.toUpperCase())) {
        loadReport(vin)
      }
    }
  }, [vin])

  const loadReport = async (vinCode) => {
    setLoading(true)
    try {
      const res = await getReport(vinCode)
      setReport(res?.report || res?.data?.report || res)
    } catch (err) {
      const msg = err.response?.data?.message || 'تعذر تحميل التقرير المطلوب'
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  // Parse summary JSON
  const summary = useMemo(() => {
    try {
      if (typeof report?.summary_json === 'string') {
        return JSON.parse(report.summary_json)
      } else if (report?.summary_json) {
        return report.summary_json
      }
    } catch {
      return null
    }
    return null
  }, [report?.summary_json])

  const hasOfficialPdf = Boolean(
    report?.has_official_pdf || summary?.has_official_pdf
  )
  const hasFullOriginal = Boolean(
    report?.has_full_html || summary?.has_full_html || hasOfficialPdf || report?.raw_html_en
  )
  const isBackgroundPending = Boolean(summary?.background_pending && !hasOfficialPdf)

  // Silent Background Polling: while Stage 2 (Full Carfax Page + 4-Section Print PDF) is finishing in Canada,
  // poll lightweight metadata every 2.5 seconds until has_official_pdf / background_ready becomes true.
  useEffect(() => {
    const targetVin = report?.vin || vin
    if (!targetVin || !isBackgroundPending) return

    let attempts = 0
    const timer = setInterval(async () => {
      attempts++
      if (attempts > 18) {
        clearInterval(timer)
        return
      }
      try {
        const res = await getReport(targetVin)
        const updated = res?.report || res?.data?.report || res
        let updatedSummary = null
        try {
          updatedSummary =
            typeof updated?.summary_json === 'string'
              ? JSON.parse(updated.summary_json)
              : updated?.summary_json
        } catch {
          updatedSummary = null
        }

        if (
          updated &&
          (updated.has_official_pdf ||
            updated.has_full_html ||
            updatedSummary?.background_ready)
        ) {
          setReport((prev) => ({
            ...prev,
            ...updated,
          }))
          clearInterval(timer)
          toast.success('اكتمل تجهيز صفحة كارفاكس الأصلية وملف الطباعة الرسمي PDF في الخلفية!')
        }
      } catch {
        // Ignore silent background poll errors
      }
    }, 2500)

    return () => clearInterval(timer)
  }, [report?.vin, vin, isBackgroundPending])

  // Lazy-load full Carfax HTML on-demand ONLY when user switches to 'original' mode
  useEffect(() => {
    const targetVin = report?.vin || vin
    if (viewMode !== 'original' || !targetVin) return
    if (htmlCache[lang]) return

    let cancelled = false
    const fetchOnDemandHtml = async () => {
      setHtmlLoading(true)
      try {
        const res = await getReportHtml(targetVin, lang)
        const htmlStr = res?.html || res?.data?.html || ''
        if (!cancelled && htmlStr) {
          setHtmlCache((prev) => ({ ...prev, [lang]: htmlStr }))
        }
      } catch {
        if (!cancelled) {
          toast.error('نسخة صفحة كارفاكس الكاملة قيد التجهيز أو غير متوفرة')
        }
      } finally {
        if (!cancelled) setHtmlLoading(false)
      }
    }

    fetchOnDemandHtml()
    return () => {
      cancelled = true
    }
  }, [viewMode, lang, report?.vin, vin, htmlCache])

  const handleDownloadOfficialPdf = () => {
    const targetVin = report?.vin || vin
    if (!hasOfficialPdf || !targetVin) {
      handlePrint()
      return
    }
    window.open(`/api/reports/${targetVin}/pdf`, '_blank')
    toast.success('تم فتح ملف الطباعة الرسمي PDF المباشر')
  }

  const handleOpenOriginalNewTab = async () => {
    const targetVin = report?.vin || vin
    if (!targetVin) return

    let htmlToOpen = htmlCache.en || htmlCache[lang] || report?.raw_html_en
    if (!htmlToOpen) {
      const loadingToast = toast.loading('جاري فتح صفحة كارفاكس الأصلية...')
      try {
        const res = await getReportHtml(targetVin, 'en')
        htmlToOpen = res?.html || res?.data?.html || ''
        if (htmlToOpen) {
          setHtmlCache((prev) => ({ ...prev, en: htmlToOpen }))
        }
        toast.dismiss(loadingToast)
      } catch {
        toast.dismiss(loadingToast)
        toast.error('محتوى الصفحة الأصلية غير متوفر بعد')
        return
      }
    }

    if (!htmlToOpen) {
      toast.error('محتوى الصفحة غير متوفر بعد')
      return
    }
    const blob = new Blob([htmlToOpen], { type: 'text/html;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    window.open(url, '_blank')
  }

  const handlePrint = () => {
    if (viewMode === 'original' && iframeRef.current && iframeRef.current.contentWindow) {
      iframeRef.current.contentWindow.focus()
      iframeRef.current.contentWindow.print()
      return
    }
    window.print()
  }

  const activeHtml = htmlCache[lang] || report?.raw_html_en || ''

  return (
    <AuthenticatedLayout
      title={`تقرير تاريخ المركبة: ${vin || ''}`}
      subtitle="CARFAX Official Vehicle History Report™"
    >
      <div className="space-y-4 text-right w-full" dir="rtl">
        {loading ? (
          <div className="flex flex-col items-center justify-center py-20 text-muted-foreground gap-2.5">
            <Loader2 className="h-6 w-6 animate-spin text-primary" />
            <span className="text-xs font-medium">جاري فتح التقرير...</span>
          </div>
        ) : !report ? (
          <div className="text-center py-16 text-muted-foreground text-sm">
            لم يتم العثور على التقرير المطلوب برقم الشاصي {vin}.
          </div>
        ) : (
          <>
            <ReportToolbar
              report={report}
              lang={lang}
              onLangChange={setLang}
              viewMode={viewMode}
              onViewModeChange={setViewMode}
              onOpenOriginalNewTab={handleOpenOriginalNewTab}
              onBack={() => navigate('/dashboard')}
              onPrint={handlePrint}
              hasOfficialPdf={hasOfficialPdf}
              isBackgroundLoading={isBackgroundPending}
              onDownloadOfficialPdf={handleDownloadOfficialPdf}
            />

            <ReportSummaryCards summary={summary} />

            {viewMode === 'smart' ? (
              <ReportNativeView
                report={report}
                summary={summary}
                lang={lang}
                hasFullOriginal={hasFullOriginal}
                onSwitchToOriginal={() => setViewMode('original')}
              />
            ) : htmlLoading && !activeHtml ? (
              <div className="flex flex-col items-center justify-center py-20 bg-card border rounded-xl text-muted-foreground gap-2.5">
                <Loader2 className="h-6 w-6 animate-spin text-primary" />
                <span className="text-xs font-medium">جاري تحميل نسخة صفحة كارفاكس الأصلية...</span>
              </div>
            ) : (
              <ReportFrame
                htmlContent={activeHtml}
                iframeRef={iframeRef}
                onPrintRequest={hasOfficialPdf ? handleDownloadOfficialPdf : handlePrint}
              />
            )}
          </>
        )}
      </div>
    </AuthenticatedLayout>
  )
}
