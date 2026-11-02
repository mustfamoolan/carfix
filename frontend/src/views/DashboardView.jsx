import React, { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import CanadaWorkerCard from '@/components/admin/CanadaWorkerCard'
import VinSearchBox from '@/components/dashboard/VinSearchBox'
import RecentSearchesTable from '@/components/dashboard/RecentSearchesTable'
import { useAuth } from '@/context/AuthContext'
import { getUserSearches, getCachedData, api } from '@/lib/api'
import { MapPin, CheckCircle2, Circle, Info, ChevronRight } from 'lucide-react'

export default function DashboardView() {
  const navigate = useNavigate()
  const { user, isAdmin, refreshUser } = useAuth()
  const cachedSearches = getCachedData('user:searches:')
  const initialLogs = cachedSearches?.logs || cachedSearches?.data?.logs || null
  const [searches, setSearches] = useState(initialLogs || [])
  const [loadingSearches, setLoadingSearches] = useState(!initialLogs)
  const [workerStatus, setWorkerStatus] = useState(getCachedData('worker:status') || null)

  const vinInputRef = useRef(null)
  const historySectionRef = useRef(null)

  const loadSearches = async (showSpinner = false) => {
    if (showSpinner || searches.length === 0) setLoadingSearches(true)
    try {
      const res = await getUserSearches({ limit: 100 })
      setSearches(res?.logs || res?.data?.logs || [])
    } catch (err) {
      console.error('Failed to load searches:', err)
    } finally {
      setLoadingSearches(false)
    }
  }

  const fetchWorkerStatus = async () => {
    try {
      const res = await api.get('/worker-status')
      setWorkerStatus(res?.data || res)
    } catch {
      // Ignore background poll error
    }
  }

  useEffect(() => {
    refreshUser()
    loadSearches()
    fetchWorkerStatus()
    const timer = setInterval(fetchWorkerStatus, 4000)
    return () => clearInterval(timer)
  }, [])

  const handleSuccessSearch = async (reportData, vin) => {
    await Promise.all([refreshUser(), fetchWorkerStatus(), loadSearches()])
    navigate(`/report/${vin}`, { state: { report: reportData } })
  }

  const focusVinInput = () => {
    if (vinInputRef.current) {
      vinInputRef.current.focus()
      vinInputRef.current.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  }

  const scrollToHistory = () => {
    if (historySectionRef.current) {
      historySectionRef.current.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  }

  const quota = user?.quota || {}
  const totalLimit = quota.total_limit ?? 0
  const usedTotal = quota.used_total ?? 0
  const remainingTotal =
    quota.remaining_total !== undefined
      ? quota.remaining_total
      : Math.max(0, totalLimit - usedTotal)

  const dailyLimit = quota.daily_limit ?? 0
  const usedToday = quota.used_today ?? 0
  const remainingDaily =
    quota.remaining_daily !== undefined
      ? quota.remaining_daily
      : Math.max(0, dailyLimit - usedToday)

  const displayName = user?.full_name || user?.username || 'Mohammed'

  return (
    <AuthenticatedLayout title="بوابة فحص CARFAX">
      <div className="space-y-6 w-full">
        {/* Admin Only: Canada Worker Status & Controls Bar */}
        {isAdmin && (
          <div dir="rtl">
            <CanadaWorkerCard status={workerStatus} onRefresh={fetchWorkerStatus} />
          </div>
        )}

        {/* Official CARFAX Advantage Dealership Portal Hero & Cards */}
        <div className="rounded-lg overflow-hidden border border-slate-200 dark:border-border bg-[#f4f6f8] dark:bg-background shadow-xs">
          {/* 1. Top Dealership Sub-Header Strip */}
          <div
            className="bg-[#1b75bc] text-white px-4 sm:px-8 py-2.5 flex flex-wrap items-center justify-between gap-3 text-xs font-semibold border-b border-white/15"
            dir="ltr"
          >
            <div className="flex items-center gap-2">
              <MapPin className="h-4 w-4 text-white shrink-0 fill-white/20" />
              <span className="tracking-wide font-bold">
                {user?.full_name || 'Authorized Dealership Portal'} (@{user?.username})
              </span>
            </div>

            <div className="flex items-center gap-4 text-[11px] sm:text-xs">
              <span className="flex items-center gap-1.5 font-bold">
                <CheckCircle2 className="h-3.5 w-3.5 fill-white text-[#1b75bc]" />
                Advantage
              </span>
              <span className="flex items-center gap-1.5 opacity-95">
                <Circle className="h-3 w-3 text-white" />
                CARFAX Car Listings
              </span>
            </div>
          </div>

          {/* 2. Curved Blue Hero Section */}
          <div className="relative bg-[#00599c] overflow-hidden pt-7 pb-32 px-4 sm:px-10">
            {/* Decorative Darker Blue Curved Wave (matches CARFAX portal hero curve) */}
            <div
              className="pointer-events-none absolute -right-24 -top-20 bottom-0 w-[55%] rounded-l-[100%] bg-[#004b87]/75 hidden md:block"
              aria-hidden="true"
            />
            <div
              className="pointer-events-none absolute -right-40 top-10 bottom-0 w-[45%] rounded-l-[100%] bg-[#003f73]/50 hidden md:block"
              aria-hidden="true"
            />

            <div className="relative z-10 max-w-5xl mx-auto" dir="ltr">
              {/* Welcome Header with CARFAX Advantage Dealer Badge */}
              <div className="flex flex-wrap items-center gap-4 mb-7">
                <img
                  src="/carfax-assets/advantage-dealer.png"
                  alt="CARFAX Advantage Dealer"
                  className="h-14 w-auto object-contain drop-shadow-sm"
                />
                <div>
                  <h1 className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight">
                    Welcome Back {displayName}!
                  </h1>
                  <p className="text-xs text-sky-100/90 mt-0.5 font-medium" dir="rtl">
                    مرحباً بك في بوابة وكلاء CARFAX المعتمدة — جاهز لسحب التقارير الفورية
                  </p>
                </div>
              </div>

              {/* Two Floating Metric Cards inside the Hero */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-12 gap-5">
                {/* Card 1: Reports Run / Quota Balance */}
                <div className="lg:col-span-4 bg-white dark:bg-card rounded shadow-md border-t-4 border-[#3898ec] flex flex-col justify-between overflow-hidden">
                  <div className="p-4 pb-2 text-center">
                    <h4 className="text-xs font-extrabold text-slate-800 dark:text-foreground">
                      Reports Run & Quota Balance
                    </h4>
                    <div className="mt-2.5 flex items-baseline justify-center gap-1.5">
                      <span className="text-3xl font-black text-slate-900 dark:text-foreground font-mono">
                        {remainingTotal}
                      </span>
                      <span className="text-xs font-bold text-slate-500">
                        / {totalLimit} متبقي
                      </span>
                    </div>
                    <p className="text-[11px] text-slate-500 mt-1 font-medium" dir="rtl">
                      المتبقي اليوم: <strong className="text-[#00599c]">{remainingDaily}</strong> من{' '}
                      {dailyLimit} • المستهلك الكلي: {usedTotal}
                    </p>
                  </div>

                  {/* CARFAX Style Trend Line Chart */}
                  <div className="px-2 pt-2">
                    <svg viewBox="0 0 240 50" className="w-full h-11 overflow-visible">
                      <polygon
                        points="10,12 120,38 230,38 230,46 10,46"
                        fill="rgba(0, 118, 190, 0.12)"
                      />
                      <polyline
                        fill="none"
                        stroke="#0076be"
                        strokeWidth="2.2"
                        points="10,12 120,38 230,38"
                      />
                      <circle cx="10" cy="12" r="3.5" fill="#0076be" />
                      <circle cx="120" cy="38" r="3.5" fill="#0076be" />
                      <circle cx="230" cy="38" r="3.5" fill="#0076be" />
                    </svg>
                  </div>

                  <div className="p-4 pt-2 space-y-2.5 border-t border-slate-100 dark:border-border">
                    <div className="text-xs font-extrabold text-slate-900 dark:text-foreground">
                      CARFAX Advantage
                    </div>
                    <button
                      type="button"
                      onClick={scrollToHistory}
                      className="w-full py-1.5 px-3 rounded border border-[#00599c] text-[#00599c] hover:bg-[#00599c] hover:text-white text-xs font-bold transition-colors cursor-pointer"
                    >
                      عرض سجل الفحوصات ({searches.length})
                    </button>
                  </div>
                </div>

                {/* Card 2: Sell Faster / Instant Report Card */}
                <div className="lg:col-span-4 bg-white dark:bg-card rounded shadow-md border-t-4 border-[#3b9c3f] flex flex-col justify-between overflow-hidden">
                  <div className="p-4 pb-2">
                    <h4 className="text-xs font-extrabold text-slate-900 dark:text-foreground mb-2">
                      Sell Faster
                    </h4>
                    <div className="flex items-center justify-between gap-2">
                      <p className="text-[11px] leading-relaxed text-slate-600 dark:text-muted-foreground flex-1" dir="rtl">
                        تقارير <strong>CARFAX</strong> الأصلية جاهزة للعرض والطباعة الفورية مع{' '}
                        <strong>{searches.length} تقرير</strong> محفوظ في أرشيفك.
                      </p>
                      <div className="w-20 h-20 rounded-lg border-2 border-slate-800 bg-slate-50 p-1 flex flex-col items-center justify-center shrink-0 shadow-xs">
                        <img
                          src="/carfax-assets/carfox-main.avif"
                          alt="Carfox"
                          className="h-12 w-auto object-contain"
                        />
                        <span className="text-[8px] font-bold text-[#00599c] mt-0.5">
                          CARFAX 4K
                        </span>
                      </div>
                    </div>
                  </div>

                  <div className="p-4 pt-2 space-y-2.5 border-t border-slate-100 dark:border-border">
                    <div className="flex items-center gap-1 text-xs font-extrabold text-slate-900 dark:text-foreground">
                      <span>CARFAX Car Listings</span>
                      <Info className="h-3.5 w-3.5 text-[#00599c]" />
                    </div>
                    <button
                      type="button"
                      onClick={focusVinInput}
                      className="w-full py-1.5 px-3 rounded bg-[#1f5c20] hover:bg-[#164417] text-white text-xs font-bold transition-colors cursor-pointer shadow-xs"
                    >
                      فحص شاصي جديد الآن
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* 3. Lower Overlapping Cards Section: Run CARFAX Report & Take Action */}
          <div className="relative z-20 -mt-20 px-4 sm:px-10 pb-8">
            <div className="max-w-5xl mx-auto grid grid-cols-1 lg:grid-cols-12 gap-6" dir="ltr">
              {/* Left Card (7 cols): Run CARFAX Report */}
              <div className="lg:col-span-7">
                <VinSearchBox onSuccessSearch={handleSuccessSearch} inputRef={vinInputRef} />
              </div>

              {/* Right Card (5 cols): Take Action */}
              <div className="lg:col-span-5">
                <div className="bg-white dark:bg-card rounded-md shadow-md border border-slate-200/90 dark:border-border p-6 flex flex-col justify-between h-full">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="text-xl font-extrabold text-slate-900 dark:text-foreground tracking-tight">
                      Take Action
                    </h3>
                    <span className="text-xs font-semibold text-slate-400">1 / 1</span>
                  </div>

                  <div className="bg-[#f2f8fd] dark:bg-primary/5 border border-[#d5e8f7] dark:border-border rounded-md p-4 space-y-2.5">
                    <h4 className="text-sm font-extrabold text-slate-900 dark:text-foreground">
                      How Can We Help? / الدعم الفني
                    </h4>
                    <p className="text-xs text-slate-600 dark:text-muted-foreground leading-relaxed" dir="rtl">
                      هل لديك استفسار أو تحتاج إلى شحن رصيد باقتك أو مساعدة في سحب التقارير؟ فريق
                      الدعم الفني جاهز لخدمتك على مدار الساعة!
                    </p>
                    <div className="pt-1 flex flex-wrap items-center justify-between gap-2">
                      <button
                        type="button"
                        onClick={() => navigate('/history')}
                        className="inline-flex items-center gap-1 text-xs font-extrabold text-[#00599c] hover:underline cursor-pointer"
                      >
                        <span>Contact Customer Support & History</span>
                        <ChevronRight className="h-3.5 w-3.5" />
                      </button>
                      <span
                        className="text-[11px] font-semibold text-emerald-700 bg-emerald-500/10 px-2 py-0.5 rounded"
                        dir="rtl"
                      >
                        الحساب نشط • متاح اليوم: {remainingDaily}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* 4. User's Recent Searches History Table */}
        <div ref={historySectionRef} className="text-right" dir="rtl">
          <RecentSearchesTable
            searches={searches}
            loading={loadingSearches}
            onRefresh={loadSearches}
            onSelectReport={(vin) => navigate(`/report/${vin}`)}
          />
        </div>
      </div>
    </AuthenticatedLayout>
  )
}
