import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import CanadaWorkerCard from '@/components/admin/CanadaWorkerCard'
import VinSearchBox from '@/components/dashboard/VinSearchBox'
import RecentSearchesTable from '@/components/dashboard/RecentSearchesTable'
import { getUserSearches, api } from '@/lib/api'

export default function DashboardView() {
  const navigate = useNavigate()
  const [searches, setSearches] = useState([])
  const [loadingSearches, setLoadingSearches] = useState(true)
  const [workerStatus, setWorkerStatus] = useState(null)

  const loadSearches = async () => {
    setLoadingSearches(true)
    try {
      const res = await getUserSearches()
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
    loadSearches()
    fetchWorkerStatus()
    const timer = setInterval(fetchWorkerStatus, 2500)
    return () => clearInterval(timer)
  }, [])

  const handleSuccessSearch = async (reportData, vin) => {
    await fetchWorkerStatus()
    navigate(`/report/${vin}`, { state: { report: reportData } })
  }

  return (
    <AuthenticatedLayout
      title="غرفة عمليات حاسبة كندا"
      subtitle="الاتصال المباشر مع carfaxonline.com وفحص وسحب التقارير"
    >
      <div className="space-y-6 text-right w-full" dir="rtl">
        {/* 1. Canada Stealth Node Live Control & Screenshot Inspector */}
        <CanadaWorkerCard status={workerStatus} onRefresh={fetchWorkerStatus} />

        {/* 2. Global 17-Char VIN Dispatcher */}
        <VinSearchBox onSuccessSearch={handleSuccessSearch} />

        {/* 3. Pulled & Cached Reports Archive */}
        <RecentSearchesTable
          searches={searches}
          loading={loadingSearches}
          onRefresh={loadSearches}
          onSelectReport={(vin) => navigate(`/report/${vin}`)}
        />
      </div>
    </AuthenticatedLayout>
  )
}
