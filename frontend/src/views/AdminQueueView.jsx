import React, { useState, useEffect } from 'react'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import {
  Loader2,
  ListOrdered,
  ArrowUpCircle,
  Trash2,
  RefreshCw,
  Clock,
  Car,
  AlertTriangle,
  CheckCircle2,
  Zap,
} from 'lucide-react'
import {
  getAdminQueue,
  promoteAdminQueueTask,
  removeAdminQueueTask,
  purgeAdminQueue,
} from '@/lib/api'
import { toast } from 'sonner'
import { useNavigate } from 'react-router-dom'

export default function AdminQueueView() {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [actionVin, setActionVin] = useState('')
  const [purging, setPurging] = useState(false)
  const navigate = useNavigate()

  const fetchQueue = async (showLoading = false) => {
    if (showLoading) setLoading(true)
    try {
      const res = await getAdminQueue()
      setData(res?.data || res)
    } catch {
      // background poll error
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchQueue(true)
    const interval = setInterval(() => fetchQueue(false), 2000)
    return () => clearInterval(interval)
  }, [])

  const handlePromote = async (vin) => {
    setActionVin(vin)
    try {
      await promoteAdminQueueTask(vin)
      toast.success(`تم تقديم الشاصي (${vin}) إلى رأس الطابور بنجاح`)
      await fetchQueue(false)
    } catch (err) {
      toast.error(err.message || 'فشل تقديم الشاصي')
    } finally {
      setActionVin('')
    }
  }

  const handleRemove = async (vin) => {
    if (!window.confirm(`هل أنت متأكد من إلغاء الشاصي (${vin}) من قائمة الانتظار؟`)) return
    setActionVin(vin)
    try {
      await removeAdminQueueTask(vin)
      toast.success(`تم حذف الشاصي (${vin}) من الطابور`)
      await fetchQueue(false)
    } catch (err) {
      toast.error(err.message || 'فشل حذف الشاصي')
    } finally {
      setActionVin('')
    }
  }

  const handlePurge = async () => {
    if (!window.confirm('تحذير: هل أنت متأكد من مسح وتفريغ قائمة الانتظار بالكامل؟')) return
    setPurging(true)
    try {
      await purgeAdminQueue()
      toast.success('تم تفريغ قائمة الانتظار بالكامل')
      await fetchQueue(false)
    } catch (err) {
      toast.error(err.message || 'فشل تفريغ الطابور')
    } finally {
      setPurging(false)
    }
  }

  const activeVin = data?.active_vin || ''
  const activeProg = data?.active_progress
  const queue = data?.queue || []
  const connected = data?.connected

  return (
    <AuthenticatedLayout title="إدارة طوابير السحب المباشرة">
      <div className="space-y-5 text-right w-full" dir="rtl">
        {/* Header Stats & Quick Actions */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 bg-card p-4 rounded-lg border border-border shadow-xs">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-md bg-primary/10 text-primary">
              <ListOrdered className="h-5 w-5" />
            </div>
            <div>
              <h2 className="text-sm font-bold text-foreground">طابور الفحص اللحظي (Queue State)</h2>
              <p className="text-xs text-muted-foreground mt-0.5">
                مراقبة الطلبات قيد السحب وإعادة ترتيب الأولويات فورياً
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => fetchQueue(true)}
              className="h-8 gap-1.5 text-xs cursor-pointer"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
              تحديث
            </Button>
            {queue.length > 0 && (
              <Button
                variant="destructive"
                size="sm"
                onClick={handlePurge}
                disabled={purging}
                className="h-8 gap-1.5 text-xs cursor-pointer"
              >
                {purging ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
                تفريغ الطابور
              </Button>
            )}
          </div>
        </div>

        {/* 0. Critical Alert Banner if Canada Node is Disconnected while Tasks are in Queue */}
        {!connected && (
          <div className="flex items-center gap-3 p-4 rounded-lg border-2 border-rose-500 bg-rose-50 dark:bg-rose-950/40 text-rose-900 dark:text-rose-200 shadow-sm animate-pulse">
            <AlertTriangle className="h-6 w-6 text-rose-600 dark:text-rose-400 shrink-0" />
            <div className="flex-1 text-xs">
              <strong className="block text-sm font-extrabold text-rose-700 dark:text-rose-300">
                تنبيه عاجل: انقطع اتصال حاسبة كندا بالسيرفر!
              </strong>
              <span>
                {queue.length > 0 || activeVin
                  ? `توجد (${queue.length + (activeVin ? 1 : 0)}) طلبات محفوظة في الطابور الآن تنتظر الاستئناف فور تشغيل برنامج كندا.`
                  : 'حاسبة كندا غير متصلة حالياً. يرجى التأكد من تشغيل الحاسبة وبرنامج السحب لمواصلة معالجة الطلبات.'}
              </span>
            </div>
            <Button
              size="sm"
              variant="outline"
              onClick={() => fetchQueue(true)}
              className="border-rose-400 bg-white dark:bg-rose-900/60 text-rose-800 dark:text-rose-200 hover:bg-rose-100 text-xs shrink-0 cursor-pointer"
            >
              فحص الاتصال
            </Button>
          </div>
        )}

        {/* 1. Active In-Flight Task (Being pulled by Canada right now) */}
        <Card className="border-border shadow-xs bg-card">
          <CardHeader className="py-3 px-4 sm:px-5 border-b border-border flex flex-row items-center justify-between">
            <div className="flex items-center gap-2">
              <Zap className="h-4 w-4 text-amber-500 animate-pulse" />
              <CardTitle className="text-sm font-bold text-foreground">الطلب قيد السحب حالياً في كندا</CardTitle>
            </div>
            {connected ? (
              <Badge variant="outline" className="border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 text-[11px] gap-1">
                <CheckCircle2 className="h-3 w-3" />
                حاسبة كندا متصلة
              </Badge>
            ) : (
              <Badge variant="secondary" className="text-muted-foreground text-[11px]">
                حاسبة كندا غير متصلة
              </Badge>
            )}
          </CardHeader>
          <CardContent className="p-4">
            {activeVin ? (
              <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 p-3.5 rounded-md bg-amber-500/10 border border-amber-500/20">
                <div className="flex items-center gap-3">
                  <div className="p-2 rounded bg-amber-500/20 text-amber-600 dark:text-amber-400">
                    <Loader2 className="h-5 w-5 animate-spin" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-sm font-extrabold text-foreground" dir="ltr">
                        {activeVin}
                      </span>
                      <Badge className="bg-amber-600 hover:bg-amber-600 text-white text-[10px]">
                        جاري السحب اللحظي
                      </Badge>
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      المرحلة: <span className="font-semibold text-foreground">{activeProg?.step_ar || activeProg?.step || 'قيد المعالجة'}</span>
                    </p>
                  </div>
                </div>
                {data?.cooldown_seconds > 0 && (
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground font-mono bg-background/80 px-2.5 py-1 rounded border border-border">
                    <Clock className="h-3.5 w-3.5 text-amber-500" />
                    فترة تهدئة لحماية الحساب: {data.cooldown_seconds}s
                  </div>
                )}
              </div>
            ) : (
              <div className="text-center py-6 text-muted-foreground text-xs flex flex-col items-center gap-2">
                <CheckCircle2 className="h-7 w-7 text-emerald-500/60" />
                <span>لا يوجد أي فحص جاري الآن (المتصفح في وضع الاستعداد)</span>
              </div>
            )}
          </CardContent>
        </Card>

        {/* 2. Pending FIFO Queue List */}
        <Card className="border-border shadow-xs bg-card">
          <CardHeader className="py-3 px-4 sm:px-5 border-b border-border flex flex-row items-center justify-between">
            <div className="flex items-center gap-2">
              <Clock className="h-4 w-4 text-primary" />
              <CardTitle className="text-sm font-bold text-foreground">قائمة الانتظار (Pending Queue)</CardTitle>
            </div>
            <Badge variant="secondary" className="font-mono text-xs">
              {queue.length} بانتظار السحب
            </Badge>
          </CardHeader>
          <CardContent className="p-0">
            {queue.length === 0 ? (
              <div className="text-center py-10 text-muted-foreground text-xs">
                الطابور فارغ تماماً حالياً — جميع الطلبات تم تنفيذها.
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow className="border-border hover:bg-transparent">
                    <TableHead className="text-right text-xs">الترتيب</TableHead>
                    <TableHead className="text-right text-xs">رقم الشاصي (VIN)</TableHead>
                    <TableHead className="text-right text-xs">الوقت المتوقع</TableHead>
                    <TableHead className="text-center text-xs">الإجراءات والتحكم</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {queue.map((vin, idx) => {
                    const isFirst = idx === 0
                    const isOperating = actionVin === vin
                    return (
                      <TableRow key={vin} className="border-border hover:bg-muted/40">
                        <TableCell className="text-xs font-bold text-muted-foreground">
                          #{idx + 1}
                        </TableCell>
                        <TableCell className="font-mono text-xs font-bold text-foreground" dir="ltr">
                          {vin}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground font-mono">
                          ~{(idx + (activeVin ? 1 : 0)) * 25 + (data?.cooldown_seconds || 0)} ثانية
                        </TableCell>
                        <TableCell className="text-center">
                          <div className="flex items-center justify-center gap-1.5">
                            {!isFirst && (
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => handlePromote(vin)}
                                disabled={isOperating}
                                className="h-7 px-2.5 text-[11px] gap-1 text-primary border-primary/30 hover:bg-primary/10 cursor-pointer"
                              >
                                {isOperating ? <Loader2 className="h-3 w-3 animate-spin" /> : <ArrowUpCircle className="h-3 w-3" />}
                                تقديم للأول
                              </Button>
                            )}
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => handleRemove(vin)}
                              disabled={isOperating}
                              className="h-7 px-2 text-[11px] text-destructive hover:bg-destructive/10 cursor-pointer"
                            >
                              <Trash2 className="h-3 w-3" />
                              إلغاء
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </AuthenticatedLayout>
  )
}

