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
    const timer = setInterval(fetchWorkerStatus, 3000)
    return () => clearInterval(timer)
  }, [])

  const handleSuccessSearch = async (reportData, vin) => {
    await fetchWorkerStatus()
    navigate(`/report/${vin}`, { state: { report: reportData } })
  }

  return (
    <AuthenticatedLayout title="فحص الشاصي">
      <div className="space-y-5 text-right w-full" dir="rtl">
        {/* 1. Minimal Canada Connection Bar */}
        <CanadaWorkerCard status={workerStatus} onRefresh={fetchWorkerStatus} />

        {/* 2. Clean VIN Search Card */}
        <VinSearchBox onSuccessSearch={handleSuccessSearch} />

        {/* 3. Recent Reports Table */}
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
