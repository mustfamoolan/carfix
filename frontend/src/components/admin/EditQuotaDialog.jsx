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
    reset_daily: false,
    reset_total: false,
  })

  useEffect(() => {
    if (user) {
      setForm({
        total_limit: user.quota?.total_limit ?? 0,
        daily_limit: user.quota?.daily_limit ?? 0,
        reset_daily: false,
        reset_total: false,
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
        reset_daily: form.reset_daily,
        reset_today: form.reset_daily,
        reset_total: form.reset_total,
      })
      toast.success(`تم تحديث حصة ورصيد المستخدم ${user.username} بنجاح`)
      onOpenChange(false)
      if (onQuotaUpdated) onQuotaUpdated()
    } catch (err) {
      const msg = err.response?.data?.message || err.message || 'فشل تحديث الحصة'
      toast.error(msg)
    } finally {
      setSubmitting(false)
    }
  }

  const q = user?.quota || {}

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md text-right" dir="rtl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-foreground font-bold">
            <Sliders className="h-5 w-5 text-primary" />
            تعديل حصة ورصيد: {user?.full_name || user?.username}
          </DialogTitle>
          <DialogDescription className="text-xs text-muted-foreground">
            المستهلك الكلي حالياً: {q.used_total ?? 0} • المستهلك اليوم: {q.used_today ?? 0}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 mt-2">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-xs font-semibold">سقف الرصيد الكلي</Label>
              <Input
                type="number"
                min="0"
                value={form.total_limit}
                onChange={(e) => setForm({ ...form, total_limit: e.target.value })}
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-semibold">الحد الأقصى اليومي</Label>
              <Input
                type="number"
                min="0"
                value={form.daily_limit}
                onChange={(e) => setForm({ ...form, daily_limit: e.target.value })}
                required
              />
            </div>
          </div>

          <div className="space-y-2.5 pt-2 border-t border-border">
            <div className="flex items-center gap-2">
              <input
                id="resetDaily"
                type="checkbox"
                checked={form.reset_daily}
                onChange={(e) => setForm({ ...form, reset_daily: e.target.checked })}
                className="h-4 w-4 rounded border-input text-primary"
              />
              <Label htmlFor="resetDaily" className="text-xs font-semibold cursor-pointer">
                تصفير استهلاك اليوم ({q.used_today ?? 0} ⬅️ 0)
              </Label>
            </div>

            <div className="flex items-center gap-2">
              <input
                id="resetTotal"
                type="checkbox"
                checked={form.reset_total}
                onChange={(e) => setForm({ ...form, reset_total: e.target.checked })}
                className="h-4 w-4 rounded border-input text-primary"
              />
              <Label htmlFor="resetTotal" className="text-xs font-semibold cursor-pointer">
                تصفير الاستهلاك الكلي وإعادة شحن الباقة بالكامل ({q.used_total ?? 0} ⬅️ 0)
              </Label>
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
              className="font-bold cursor-pointer"
            >
              {submitting ? 'جاري الحفظ...' : 'حفظ الرصيد'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
