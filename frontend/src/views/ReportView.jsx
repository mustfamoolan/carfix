import React, { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import ReportToolbar from '@/components/report/ReportToolbar'
import ReportNativeView from '@/components/report/ReportNativeView'
import { getReport } from '@/lib/api'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'

export default function ReportView() {
  const { vin } = useParams()
  const navigate = useNavigate()
  const location = useLocation()

  const [report, setReport] = useState(location.state?.report || null)
  const [loading, setLoading] = useState(!report)
  const [lang, setLang] = useState('en') // Official English layout by default, 1-click Arabic toggle

  useEffect(() => {
    if (vin) {
      if (
        !report ||
        (report.vin && report.vin.toUpperCase() !== vin.toUpperCase())
      ) {
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

  // Parse summary JSON (~5 KB structured data)
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
  const isBackgroundPending = Boolean(
    summary?.background_pending && !hasOfficialPdf
  )

  // Silent Background Polling while Stage 2 (4-Section Print PDF) finishes in Canada
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
          (updated.has_official_pdf || updatedSummary?.background_ready)
        ) {
          setReport((prev) => ({
            ...prev,
            ...updated,
          }))
          clearInterval(timer)
          toast.success('اكتمل تجهيز ملف الطباعة الرسمي PDF في الخلفية!')
        }
      } catch {
        // Ignore silent background poll errors
      }
    }, 2500)

    return () => clearInterval(timer)
  }, [report?.vin, vin, isBackgroundPending])

  const handleDownloadOfficialPdf = () => {
    const targetVin = report?.vin || vin
    if (!hasOfficialPdf || !targetVin) {
      window.print()
      return
    }
    window.open(`/api/reports/${targetVin}/pdf`, '_blank')
  }

  const handlePrint = () => {
    if (hasOfficialPdf) {
      handleDownloadOfficialPdf()
      return
    }
    window.print()
  }

  return (
    <AuthenticatedLayout
      title={`تقرير كارفاكس: ${vin || ''}`}
      subtitle="CARFAX Vehicle History Report™"
    >
      <div className="space-y-3 text-right w-full" dir="rtl">
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
              onBack={() => navigate('/dashboard')}
              onPrint={handlePrint}
              hasOfficialPdf={hasOfficialPdf}
              isBackgroundLoading={isBackgroundPending}
              onDownloadOfficialPdf={handleDownloadOfficialPdf}
            />

            <ReportNativeView
              report={report}
              summary={summary}
              lang={lang}
              onLangChange={setLang}
              onPrint={handlePrint}
            />
          </>
        )}
      </div>
    </AuthenticatedLayout>
  )
}
