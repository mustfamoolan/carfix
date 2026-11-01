import React, { useState, useEffect } from 'react'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import UsersManagementTable from '@/components/admin/UsersManagementTable'
import CreateUserDialog from '@/components/admin/CreateUserDialog'
import EditQuotaDialog from '@/components/admin/EditQuotaDialog'
import { Button } from '@/components/ui/button'
import { UserPlus } from 'lucide-react'
import { toast } from 'sonner'
import { getAdminUsers, updateAdminUser } from '@/lib/api'

export default function AdminUsersView() {
  const [users, setUsers] = useState([])
  const [loading, setLoading] = useState(true)
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [selectedUserForQuota, setSelectedUserForQuota] = useState(null)

  const fetchUsers = async () => {
    setLoading(true)
    try {
      const res = await getAdminUsers()
      setUsers(res?.users || res?.data?.users || [])
    } catch {
      toast.error('فشل تحميل قائمة المستخدمين')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchUsers()
  }, [])

  const handleToggleStatus = async (u) => {
    const newStatus = !u.is_active
    try {
      await updateAdminUser(u.id, { is_active: newStatus })
      toast.success(newStatus ? 'تم تفعيل الحساب' : 'تم تعطيل الحساب')
      fetchUsers()
    } catch {
      toast.error('فشل تحديث حالة الحساب')
    }
  }

  return (
    <AuthenticatedLayout title="إدارة المستخدمين">
      <div className="space-y-5 text-right w-full" dir="rtl">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div>
            <h2 className="text-lg font-bold text-foreground">إدارة المستخدمين</h2>
            <p className="text-xs text-muted-foreground mt-0.5">
              إضافة وتحديث حسابات المستخدمين وتحديد أرصدتهم
            </p>
          </div>
          <Button
            onClick={() => setIsCreateOpen(true)}
            className="gap-2 text-xs font-bold h-9 cursor-pointer"
          >
            <UserPlus className="h-4 w-4" />
            إضافة مستخدم جديد
          </Button>
        </div>

        <UsersManagementTable
          users={users}
          loading={loading}
          onRefresh={fetchUsers}
          onEditQuota={(u) => setSelectedUserForQuota(u)}
          onToggleStatus={handleToggleStatus}
        />

        <CreateUserDialog
          open={isCreateOpen}
          onOpenChange={setIsCreateOpen}
          onUserCreated={fetchUsers}
        />

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
