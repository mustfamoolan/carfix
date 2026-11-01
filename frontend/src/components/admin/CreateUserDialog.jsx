import React, { useState } from 'react'
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
import { UserPlus } from 'lucide-react'
import { toast } from 'sonner'
import { createAdminUser } from '@/lib/api'

export default function CreateUserDialog({ open, onOpenChange, onUserCreated }) {
  const [submitting, setSubmitting] = useState(false)
  const [form, setForm] = useState({
    full_name: '',
    username: '',
    phone: '',
    password: '',
    role: 'user',
    total_limit: 50,
    daily_limit: 10,
  })

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!form.username.trim() || !form.password.trim()) {
      toast.error('يرجى ملء اسم المستخدم وكلمة المرور')
      return
    }

    setSubmitting(true)
    try {
      await createAdminUser({
        full_name: form.full_name,
        username: form.username.trim(),
        phone: form.phone,
        password: form.password,
        role: form.role,
        total_limit: Number(form.total_limit) || 0,
        daily_limit: Number(form.daily_limit) || 0,
      })
      toast.success('تم إنشاء حساب المستخدم بنجاح')
      onOpenChange(false)
      setForm({
        full_name: '',
        username: '',
        phone: '',
        password: '',
        role: 'user',
        total_limit: 50,
        daily_limit: 10,
      })
      if (onUserCreated) onUserCreated()
    } catch (err) {
      const msg = err.response?.data?.message || err.response?.data?.error || err.message || 'فشل إنشاء المستخدم'
      toast.error(msg)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md text-right font-cairo" dir="rtl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-slate-900 dark:text-slate-100 font-bold">
            <UserPlus className="h-5 w-5 text-[#003478]" />
            إضافة حساب مستخدم / تاجر جديد
          </DialogTitle>
          <DialogDescription className="text-xs text-slate-500">
            قم بتعبئة بيانات الحساب وتعيين الحصة الابتدائية المخصصة له
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
                placeholder="alamal_cars"
                dir="ltr"
                value={form.username}
                onChange={(e) => setForm({ ...form, username: e.target.value })}
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-semibold">رقم الهاتف</Label>
              <Input
                placeholder="0770XXXXXXX"
                dir="ltr"
                value={form.phone}
                onChange={(e) => setForm({ ...form, phone: e.target.value })}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-xs font-semibold">كلمة المرور *</Label>
              <Input
                type="password"
                placeholder="••••••••"
                dir="ltr"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-semibold">الصلاحية (الدور)</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm"
                value={form.role}
                onChange={(e) => setForm({ ...form, role: e.target.value })}
              >
                <option value="user">تاجر (Dealer)</option>
                <option value="admin">مدير نظام (Admin)</option>
              </select>
            </div>
          </div>

          <div className="border-t border-slate-200 dark:border-slate-800 pt-3">
            <p className="text-xs font-bold text-slate-700 dark:text-slate-300 mb-2">إعدادات الحصة الابتدائية:</p>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label className="text-xs text-slate-500">الرصيد الكلي للباقة</Label>
                <Input
                  type="number"
                  value={form.total_limit}
                  onChange={(e) => setForm({ ...form, total_limit: e.target.value })}
                />
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs text-slate-500">الحد الأقصى اليومي</Label>
                <Input
                  type="number"
                  value={form.daily_limit}
                  onChange={(e) => setForm({ ...form, daily_limit: e.target.value })}
                />
              </div>
            </div>
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
              className="bg-[#003478] hover:bg-[#002657] text-white font-bold"
            >
              {submitting ? 'جاري الحفظ...' : 'إنشاء الحساب'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
