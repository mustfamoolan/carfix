import React, { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { UserCog } from 'lucide-react'
import { toast } from 'sonner'
import { updateAdminUser } from '@/lib/api'

export default function EditUserDialog({ user, open, onOpenChange, onUserUpdated }) {
  const [submitting, setSubmitting] = useState(false)
  const [form, setForm] = useState({
    full_name: '',
    username: '',
    phone: '',
    password: '',
    role: 'user',
    is_active: true,
  })

  useEffect(() => {
    if (user) {
      setForm({
        full_name: user.full_name || '',
        username: user.username || '',
        phone: user.phone || '',
        password: '',
        role: user.role || 'user',
        is_active: user.is_active ?? true,
      })
    }
  }, [user])

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!user) return

    if (!form.username.trim()) {
      toast.error('اسم المستخدم مطلوب')
      return
    }

    setSubmitting(true)
    try {
      const payload = {
        full_name: form.full_name.trim() || form.username.trim(),
        username: form.username.trim(),
        phone: form.phone.trim(),
        role: form.role,
        is_active: form.is_active,
      }
      if (form.password.trim()) {
        payload.password = form.password.trim()
      }

      await updateAdminUser(user.id, payload)
      toast.success(`تم تحديث بيانات المستخدم ${form.username} بنجاح`)
      onOpenChange(false)
      if (onUserUpdated) onUserUpdated()
    } catch (err) {
      const msg =
        err.response?.data?.message ||
        err.data?.message ||
        err.message ||
        'فشل تحديث بيانات المستخدم'
      toast.error(msg)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md text-right" dir="rtl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-foreground font-bold">
            <UserCog className="h-5 w-5 text-primary" />
            تعديل بيانات الحساب: {user?.username}
          </DialogTitle>
          <DialogDescription className="text-xs text-muted-foreground">
            تحديث الاسم، رقم الهاتف، الصلاحية، أو تعيين كلمة مرور جديدة
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-3.5 mt-2">
          <div className="space-y-1.5">
            <Label className="text-xs font-semibold">الاسم الكامل / اسم المعرض</Label>
            <Input
              placeholder="مثال: معرض الأمل للسيارات"
              value={form.full_name}
              onChange={(e) => setForm({ ...form, full_name: e.target.value })}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-xs font-semibold">اسم المستخدم *</Label>
              <Input
                dir="ltr"
                value={form.username}
                onChange={(e) => setForm({ ...form, username: e.target.value })}
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-semibold">رقم الهاتف</Label>
              <Input
                dir="ltr"
                placeholder="0770XXXXXXX"
                value={form.phone}
                onChange={(e) => setForm({ ...form, phone: e.target.value })}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-xs font-semibold">كلمة مرور جديدة (اختياري)</Label>
              <Input
                type="password"
                dir="ltr"
                placeholder="اتركه فارغاً لعدم التغيير"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-semibold">الصلاحية (الدور)</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-xs shadow-xs"
                value={form.role}
                onChange={(e) => setForm({ ...form, role: e.target.value })}
              >
                <option value="user">مستخدم / تاجر (Dealer)</option>
                <option value="admin">مدير نظام (Admin)</option>
              </select>
            </div>
          </div>

          <div className="flex items-center gap-2 pt-2 border-t border-border">
            <input
              id="editIsActive"
              type="checkbox"
              checked={form.is_active}
              onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
              className="h-4 w-4 rounded border-input text-primary"
            />
            <Label htmlFor="editIsActive" className="text-xs font-semibold cursor-pointer">
              الحساب نشط ومسموح له بتسجيل الدخول
            </Label>
          </div>

          <DialogFooter className="mt-4 gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onOpenChange(false)}
            >
              إلغاء
            </Button>
            <Button
              type="submit"
              size="sm"
              disabled={submitting}
              className="font-bold cursor-pointer"
            >
              {submitting ? 'جاري الحفظ...' : 'حفظ التعديلات'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

