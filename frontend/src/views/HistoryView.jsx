import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import RecentSearchesTable from '@/components/dashboard/RecentSearchesTable'
import { getUserSearches } from '@/lib/api'

export default function HistoryView() {
  const navigate = useNavigate()
  const [searches, setSearches] = useState([])
  const [loading, setLoading] = useState(true)

  const loadSearches = async () => {
    setLoading(true)
    try {
      const res = await getUserSearches()
      setSearches(res?.logs || res?.data?.logs || [])
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadSearches()
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
          onRefresh={loadSearches}
          onSelectReport={(vin) => navigate(`/report/${vin}`)}
        />
      </div>
    </AuthenticatedLayout>
  )
}
