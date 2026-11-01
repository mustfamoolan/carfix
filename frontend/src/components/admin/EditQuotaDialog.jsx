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
import { Sliders } from 'lucide-react'
import { toast } from 'sonner'
import { updateUserQuota } from '@/lib/api'

export default function EditQuotaDialog({ user, open, onOpenChange, onQuotaUpdated }) {
  const [submitting, setSubmitting] = useState(false)
  const [form, setForm] = useState({
    total_limit: 0,
    daily_limit: 0,
    reset_today: false,
  })

  useEffect(() => {
    if (user) {
      setForm({
        total_limit: user.quota?.total_limit ?? 0,
        daily_limit: user.quota?.daily_limit ?? 0,
        reset_today: false,
      })
    }
  }, [user])

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!user) return

    setSubmitting(true)
    try {
      await updateUserQuota(user.id, {
        total_limit: Number(form.total_limit),
        daily_limit: Number(form.daily_limit),
        reset_today: form.reset_today,
      })
      toast.success(`تم تحديث حصة المستخدم ${user.username} بنجاح`)
      onOpenChange(false)
      if (onQuotaUpdated) onQuotaUpdated()
    } catch (err) {
      const msg = err.response?.data?.message || err.message || 'فشل تحديث الحصة'
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
            <Sliders className="h-5 w-5 text-[#003478]" />
            تعديل حصة ورصيد: {user?.username}
          </DialogTitle>
          <DialogDescription className="text-xs text-slate-500">
            تعديل سقف الرصيد الكلي أو الحد اليومي المسموح به لهذا الحساب
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 mt-2">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-xs font-semibold">الرصيد الكلي المتاح</Label>
              <Input
                type="number"
                value={form.total_limit}
                onChange={(e) => setForm({ ...form, total_limit: e.target.value })}
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-semibold">الحد الأقصى اليومي</Label>
              <Input
                type="number"
                value={form.daily_limit}
                onChange={(e) => setForm({ ...form, daily_limit: e.target.value })}
                required
              />
            </div>
          </div>

          <div className="flex items-center gap-2 pt-2 border-t border-slate-200 dark:border-slate-800">
            <input
              id="resetToday"
              type="checkbox"
              checked={form.reset_today}
              onChange={(e) => setForm({ ...form, reset_today: e.target.checked })}
              className="h-4 w-4 rounded border-slate-300 text-[#003478] focus:ring-[#003478]"
            />
            <Label htmlFor="resetToday" className="text-xs font-semibold cursor-pointer text-slate-700 dark:text-slate-300">
              تصفير عداد اليوم وإعادة تعيين الرصيد اليومي فوراً
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
              className="bg-[#003478] hover:bg-[#002657] text-white font-bold"
            >
              {submitting ? 'جاري الحفظ...' : 'حفظ التعديلات'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
