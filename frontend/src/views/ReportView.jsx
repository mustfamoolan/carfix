import React, { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import ReportToolbar from '@/components/report/ReportToolbar'
import ReportSummaryCards from '@/components/report/ReportSummaryCards'
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
  const [lang, setLang] = useState('ar') // 'ar' or 'en'
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

  const handlePrint = () => {
    if (iframeRef.current && iframeRef.current.contentWindow) {
      iframeRef.current.contentWindow.focus()
      iframeRef.current.contentWindow.print()
    }
  }

  // Parse summary JSON
  let summary = null
  try {
    if (typeof report?.summary_json === 'string') {
      summary = JSON.parse(report.summary_json)
    } else if (report?.summary_json) {
      summary = report.summary_json
    }
  } catch {
    summary = null
  }

  const activeHtml =
    lang === 'ar'
      ? report?.translated_html_ar || report?.raw_html_en
      : report?.raw_html_en || report?.translated_html_ar

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
              onBack={() => navigate('/dashboard')}
              onPrint={handlePrint}
            />

            <ReportSummaryCards summary={summary} />

            <ReportFrame htmlContent={activeHtml} iframeRef={iframeRef} />
          </>
        )}
      </div>
    </AuthenticatedLayout>
  )
}
