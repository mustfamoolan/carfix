import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import RecentSearchesTable from '@/components/dashboard/RecentSearchesTable'
import { getUserSearches, getCachedData } from '@/lib/api'

export default function HistoryView() {
  const navigate = useNavigate()
  const cachedSearches = getCachedData('user:searches:')
  const initialLogs = cachedSearches?.logs || cachedSearches?.data?.logs || null
  const [searches, setSearches] = useState(initialLogs || [])
  const [loading, setLoading] = useState(!initialLogs)

  const loadSearches = async (showSpinner = false) => {
    if (showSpinner || searches.length === 0) setLoading(true)
    try {
      const res = await getUserSearches({ limit: 150 })
      setSearches(res?.logs || res?.data?.logs || [])
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadSearches(false)
  }, [])

  return (
    <AuthenticatedLayout
      title="سجل التقارير المسحوبة"
      subtitle="أرشيف كامل لجميع عمليات الفحص السابقة"
    >
      <div className="space-y-6 text-right w-full" dir="rtl">
        <RecentSearchesTable
          searches={searches}
          loading={loading}
          onRefresh={() => loadSearches(true)}
          onSelectReport={(vin) => navigate(`/report/${vin}`)}
        />
      </div>
    </AuthenticatedLayout>
  )
}
