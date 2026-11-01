import React, { useState, useEffect, useRef, useMemo } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import ReportToolbar from '@/components/report/ReportToolbar'
import ReportSummaryCards from '@/components/report/ReportSummaryCards'
import ReportNativeView from '@/components/report/ReportNativeView'
import ReportFrame from '@/components/report/ReportFrame'
import { getReport } from '@/lib/api'
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
  const iframeRef = useRef(null)

  useEffect(() => {
    if (!report && vin) {
      loadReport(vin)
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

  // Extract embedded Official Native Chrome PDF if present inside raw_html_en
  const officialPdfBase64 = useMemo(() => {
    const raw = report?.raw_html_en || ''
    const markerStart = '<script id="carfix-official-pdf" type="application/pdf-base64">'
    const startIdx = raw.indexOf(markerStart)
    if (startIdx === -1) return null
    const contentStart = startIdx + markerStart.length
    const endIdx = raw.indexOf('</script>', contentStart)
    if (endIdx === -1) return null
    return raw.substring(contentStart, endIdx).trim()
  }, [report?.raw_html_en])

  const isBackgroundPending = Boolean(summary?.background_pending && !officialPdfBase64)

  // Silent Background Polling: while Stage 2 (Full Carfax Page + 4-Section Print PDF) is finishing in Canada,
  // poll silently every 2.5 seconds until the full assets & PDF arrive.
  useEffect(() => {
    const targetVin = report?.vin || vin
    if (!targetVin || !isBackgroundPending) return

    let attempts = 0
    const timer = setInterval(async () => {
      attempts++
      if (attempts > 15) {
        clearInterval(timer)
        return
      }
      try {
        const res = await getReport(targetVin)
        const updated = res?.report || res?.data?.report || res
        if (updated && updated.raw_html_en && updated.raw_html_en.includes('carfix-official-pdf')) {
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

  const handleDownloadOfficialPdf = () => {
    if (!officialPdfBase64) {
      handlePrint()
      return
    }
    try {
      const byteCharacters = atob(officialPdfBase64)
      const byteNumbers = new Array(byteCharacters.length)
      for (let i = 0; i < byteCharacters.length; i++) {
        byteNumbers[i] = byteCharacters.charCodeAt(i)
      }
      const byteArray = new Uint8Array(byteNumbers)
      const blob = new Blob([byteArray], { type: 'application/pdf' })
      const blobUrl = URL.createObjectURL(blob)

      // Open in new tab for native PDF viewing & printing + trigger download
      window.open(blobUrl, '_blank')
      const link = document.createElement('a')
      link.href = blobUrl
      link.download = `CARFAX_${report?.vin || vin || 'Report'}.pdf`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      toast.success('تم فتح وتحميل ملف الطباعة الرسمي PDF من حاسبة كندا')
    } catch (e) {
      console.error(e)
      handlePrint()
    }
  }

  const activeHtml =
    lang === 'ar'
      ? report?.translated_html_ar || report?.raw_html_en
      : report?.raw_html_en || report?.translated_html_ar

  const handleOpenOriginalNewTab = () => {
    const htmlToOpen = report?.raw_html_en || activeHtml
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

  return (
    <AuthenticatedLayout
      title={`تقرير تاريخ المركبة: ${vin || ''}`}
      subtitle="CARFAX Official Vehicle History Report™"
    >
      <div className="space-y-4 text-right w-full" dir="rtl">
        {loading ? (
          <div className="flex flex-col items-center justify-center py-24 text-slate-400 gap-3">
            <Loader2 className="h-8 w-8 animate-spin text-[#003478]" />
            <span className="text-sm font-semibold">جاري جلب تفاصيل تقرير كارفاكس...</span>
          </div>
        ) : !report ? (
          <div className="text-center py-16 text-slate-400 text-sm">
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
              hasOfficialPdf={Boolean(officialPdfBase64)}
              isBackgroundLoading={isBackgroundPending}
              onDownloadOfficialPdf={handleDownloadOfficialPdf}
            />

            <ReportSummaryCards summary={summary} />

            {viewMode === 'smart' ? (
              <ReportNativeView
                report={report}
                summary={summary}
                lang={lang}
                hasFullOriginal={Boolean(officialPdfBase64)}
                onSwitchToOriginal={() => setViewMode('original')}
              />
            ) : (
              <ReportFrame
                htmlContent={activeHtml}
                iframeRef={iframeRef}
                onPrintRequest={officialPdfBase64 ? handleDownloadOfficialPdf : handlePrint}
              />
            )}
          </>
        )}
      </div>
    </AuthenticatedLayout>
  )
}
