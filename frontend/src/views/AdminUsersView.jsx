import React, { useState, useEffect } from 'react'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import UsersManagementTable from '@/components/admin/UsersManagementTable'
import CreateUserDialog from '@/components/admin/CreateUserDialog'
import EditUserDialog from '@/components/admin/EditUserDialog'
import EditQuotaDialog from '@/components/admin/EditQuotaDialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent } from '@/components/ui/card'
import { UserPlus, Search, RefreshCw, Users, CheckCircle2, FileText } from 'lucide-react'
import { toast } from 'sonner'
import { getAdminUsers, updateAdminUser, deleteAdminUser, getCachedData } from '@/lib/api'

export default function AdminUsersView() {
  const cachedUsers = getCachedData('admin:users:')
  const [users, setUsers] = useState(cachedUsers || [])
  const [loading, setLoading] = useState(!cachedUsers)
  const [searchQuery, setSearchQuery] = useState('')
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [selectedUserForEdit, setSelectedUserForEdit] = useState(null)
  const [selectedUserForQuota, setSelectedUserForQuota] = useState(null)

  const fetchUsers = async (query = searchQuery, showSpinner = false) => {
    if (showSpinner || users.length === 0) setLoading(true)
    try {
      const res = await getAdminUsers(query)
      const list = Array.isArray(res)
        ? res
        : res?.users || res?.data?.users || (Array.isArray(res?.data) ? res.data : [])
      setUsers(list)
    } catch {
      toast.error('فشل تحميل قائمة المستخدمين')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchUsers('', false)
  }, [])

  const handleSearch = (e) => {
    e.preventDefault()
    fetchUsers(searchQuery)
  }

  const handleToggleStatus = async (u) => {
    const newStatus = !u.is_active
    try {
      await updateAdminUser(u.id, { is_active: newStatus })
      toast.success(newStatus ? `تم تفعيل حساب ${u.username}` : `تم تعطيل حساب ${u.username}`)
      fetchUsers()
    } catch (err) {
      toast.error(err?.message || 'فشل تحديث حالة الحساب')
    }
  }

  const handleDeleteUser = async (u) => {
    if (!window.confirm(`هل أنت متأكد من حذف حساب "${u.full_name || u.username}" (@${u.username}) نهائياً؟`)) {
      return
    }
    try {
      await deleteAdminUser(u.id)
      toast.success(`تم حذف الحساب @${u.username} بنجاح`)
      fetchUsers()
    } catch (err) {
      toast.error(err?.message || 'تعذر حذف المستخدم')
    }
  }

  const activeCount = users.filter((u) => u.is_active).length
  const totalPulls = users.reduce(
    (acc, u) => acc + (Number(u.searches_count ?? u.quota?.used_total ?? 0) || 0),
    0
  )

  return (
    <AuthenticatedLayout title="إدارة المستخدمين">
      <div className="space-y-5 text-right w-full" dir="rtl">
        {/* Header & Add User */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div>
            <h2 className="text-lg font-bold text-foreground">إدارة المستخدمين والتجار</h2>
            <p className="text-xs text-muted-foreground mt-0.5">
              إضافة وتعديل حسابات المستخدمين، تغيير كلمات المرور، وشحن الأرصدة اليومية والكلية
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => fetchUsers()}
              className="gap-1.5 text-xs h-9 cursor-pointer"
            >
              <RefreshCw className="h-3.5 w-3.5" />
              تحديث
            </Button>
            <Button
              onClick={() => setIsCreateOpen(true)}
              className="gap-2 text-xs font-bold h-9 cursor-pointer"
            >
              <UserPlus className="h-4 w-4" />
              إضافة مستخدم جديد
            </Button>
          </div>
        </div>

        {/* Quick Stats Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3.5">
          <Card className="border-border shadow-xs bg-card">
            <CardContent className="p-4 flex items-center justify-between">
              <div>
                <p className="text-xs text-muted-foreground font-medium">إجمالي الحسابات</p>
                <p className="text-2xl font-bold font-mono text-foreground mt-1">{users.length}</p>
              </div>
              <div className="h-10 w-10 rounded-lg bg-primary/10 text-primary flex items-center justify-center">
                <Users className="h-5 w-5" />
              </div>
            </CardContent>
          </Card>

          <Card className="border-border shadow-xs bg-card">
            <CardContent className="p-4 flex items-center justify-between">
              <div>
                <p className="text-xs text-muted-foreground font-medium">الحسابات النشطة</p>
                <p className="text-2xl font-bold font-mono text-emerald-600 mt-1">{activeCount}</p>
              </div>
              <div className="h-10 w-10 rounded-lg bg-emerald-500/10 text-emerald-600 flex items-center justify-center">
                <CheckCircle2 className="h-5 w-5" />
              </div>
            </CardContent>
          </Card>

          <Card className="border-border shadow-xs bg-card">
            <CardContent className="p-4 flex items-center justify-between">
              <div>
                <p className="text-xs text-muted-foreground font-medium">إجمالي التقارير المسحوبة</p>
                <p className="text-2xl font-bold font-mono text-primary mt-1">{totalPulls}</p>
              </div>
              <div className="h-10 w-10 rounded-lg bg-primary/10 text-primary flex items-center justify-center">
                <FileText className="h-5 w-5" />
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Search Bar */}
        <form onSubmit={handleSearch} className="flex items-center gap-2 max-w-md">
          <div className="relative flex-1">
            <Search className="h-4 w-4 text-muted-foreground absolute right-3 top-2.5" />
            <Input
              placeholder="بحث بالاسم، اسم المستخدم، أو الهاتف..."
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value)
                if (e.target.value === '') fetchUsers('')
              }}
              className="pr-9 h-9 text-xs"
            />
          </div>
          <Button type="submit" variant="secondary" size="sm" className="h-9 text-xs font-semibold cursor-pointer">
            بحث
          </Button>
        </form>

        {/* Users Table */}
        <UsersManagementTable
          users={users}
          loading={loading}
          onRefresh={() => fetchUsers()}
          onEditUser={(u) => setSelectedUserForEdit(u)}
          onEditQuota={(u) => setSelectedUserForQuota(u)}
          onToggleStatus={handleToggleStatus}
          onDeleteUser={handleDeleteUser}
        />

        {/* Dialogs */}
        <CreateUserDialog
          open={isCreateOpen}
          onOpenChange={setIsCreateOpen}
          onUserCreated={() => fetchUsers()}
        />

        <EditUserDialog
          user={selectedUserForEdit}
          open={!!selectedUserForEdit}
          onOpenChange={(open) => !open && setSelectedUserForEdit(null)}
          onUserUpdated={() => fetchUsers()}
        />

        <EditQuotaDialog
          user={selectedUserForQuota}
          open={!!selectedUserForQuota}
          onOpenChange={(open) => !open && setSelectedUserForQuota(null)}
          onQuotaUpdated={() => fetchUsers()}
        />
      </div>
    </AuthenticatedLayout>
  )
}
