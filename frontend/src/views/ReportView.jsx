import React, { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import ReportToolbar from '@/components/report/ReportToolbar'
import ReportNativeView from '@/components/report/ReportNativeView'
import { getReport, getCachedData } from '@/lib/api'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'

export default function ReportView() {
  const { vin } = useParams()
  const navigate = useNavigate()
  const location = useLocation()

  const cachedReport = vin ? getCachedData(`report:${String(vin).toUpperCase()}`) : null
  const [report, setReport] = useState(location.state?.report || cachedReport || null)
  const [loading, setLoading] = useState(!(location.state?.report || cachedReport))
  const [lang, setLang] = useState('en') // Official English layout by default, 1-click Arabic toggle

  useEffect(() => {
    if (vin) {
      const cached = getCachedData(`report:${String(vin).toUpperCase()}`)
      if (cached && (!report || report.vin?.toUpperCase() !== vin.toUpperCase())) {
        setReport(cached)
        setLoading(false)
        loadReport(vin, false)
      } else if (
        !report ||
        (report.vin && report.vin.toUpperCase() !== vin.toUpperCase())
      ) {
        loadReport(vin, true)
      }
    }
  }, [vin])

  const loadReport = async (vinCode, showSpinner = true) => {
    if (showSpinner) setLoading(true)
    try {
      const res = await getReport(vinCode)
      setReport(res?.report || res?.data?.report || res)
    } catch (err) {
      if (showSpinner) {
        const msg = err.response?.data?.message || 'تعذر تحميل التقرير المطلوب'
        toast.error(msg)
      }
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

  const handlePrint = () => {
    if (!vin) return
    const token = localStorage.getItem('carfix_token')
    const pdfUrl = `/api/reports/${encodeURIComponent(vin)}/pdf${token ? `?token=${encodeURIComponent(token)}` : ''}`
    
    // Direct official download / open in new tab
    const win = window.open(pdfUrl, '_blank')
    if (!win) {
      // Fallback if popup blocked: create temporary hidden anchor tag
      const a = document.createElement('a')
      a.href = pdfUrl
      a.download = `CARFAX_${vin}.pdf`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
    }
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
