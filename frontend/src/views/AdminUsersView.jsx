import React, { useState, useEffect } from 'react'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import CanadaWorkerCard from '@/components/admin/CanadaWorkerCard'
import AdminStatsCards from '@/components/admin/AdminStatsCards'
import UsersManagementTable from '@/components/admin/UsersManagementTable'
import CreateUserDialog from '@/components/admin/CreateUserDialog'
import EditQuotaDialog from '@/components/admin/EditQuotaDialog'
import { Button } from '@/components/ui/button'
import { UserPlus, Shield } from 'lucide-react'
import { toast } from 'sonner'
import { getAdminUsers, updateAdminUser, api } from '@/lib/api'

export default function AdminUsersView() {
  const [users, setUsers] = useState([])
  const [loading, setLoading] = useState(true)
  const [workerStatus, setWorkerStatus] = useState(null)

  // Dialogs state
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [selectedUserForQuota, setSelectedUserForQuota] = useState(null)

  const fetchUsers = async () => {
    setLoading(true)
    try {
      const res = await getAdminUsers()
      setUsers(res?.users || res?.data?.users || [])
    } catch (err) {
      toast.error('فشل تحميل قائمة المستخدمين')
    } finally {
      setLoading(false)
    }
  }

  const fetchWorkerStatus = async () => {
    try {
      const res = await api.get('/admin/worker-status')
      setWorkerStatus(res?.data || res)
    } catch {
      // Ignore background errors
    }
  }

  useEffect(() => {
    fetchUsers()
    fetchWorkerStatus()
    const timer = setInterval(fetchWorkerStatus, 5000)
    return () => clearInterval(timer)
  }, [])

  const handleToggleStatus = async (u) => {
    const newStatus = !u.is_active
    try {
      await updateAdminUser(u.id, { is_active: newStatus })
      toast.success(newStatus ? 'تم تفعيل الحساب بنجاح' : 'تم تعطيل الحساب بنجاح')
      fetchUsers()
    } catch (err) {
      toast.error('فشل تحديث حالة الحساب')
    }
  }

  return (
    <AuthenticatedLayout
      title="لوحة تحكم الإدارة"
      subtitle="إدارة حسابات التجار وباقات الفحص اليومية والكلية"
    >
      <div className="space-y-6 text-right w-full" dir="rtl">
        {/* Header & Quick Action */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div>
            <h2 className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100 flex items-center gap-2">
              <Shield className="h-6 w-6 text-[#003478] dark:text-blue-300" />
              إدارة التجار وباقات الاستخدام
            </h2>
            <p className="text-xs sm:text-sm text-slate-500 mt-0.5">
              إضافة وتعديل حسابات المعارض والوكلاء والتحكم في باقات الرصيد اليومي والكلي
            </p>
          </div>
          <Button
            onClick={() => setIsCreateOpen(true)}
            className="gap-2 text-xs font-bold h-10 bg-[#003478] hover:bg-[#002657] text-white shadow-sm"
          >
            <UserPlus className="h-4 w-4" />
            إضافة تاجر / مستخدم جديد
          </Button>
        </div>

        {/* Canada Live Stealth Worker Status & Anti-Ban Monitor */}
        <CanadaWorkerCard status={workerStatus} onRefresh={fetchWorkerStatus} />

        {/* Overview Statistics Cards */}
        <AdminStatsCards users={users} />

        {/* Users Management Data Table */}
        <UsersManagementTable
          users={users}
          loading={loading}
          onRefresh={fetchUsers}
          onEditQuota={(u) => setSelectedUserForQuota(u)}
          onToggleStatus={handleToggleStatus}
        />

        {/* Create User Dialog */}
        <CreateUserDialog
          open={isCreateOpen}
          onOpenChange={setIsCreateOpen}
          onUserCreated={fetchUsers}
        />

        {/* Edit Quota Dialog */}
        <EditQuotaDialog
          user={selectedUserForQuota}
          open={!!selectedUserForQuota}
          onOpenChange={(open) => !open && setSelectedUserForQuota(null)}
          onQuotaUpdated={fetchUsers}
        />
      </div>
    </AuthenticatedLayout>
  )
}
