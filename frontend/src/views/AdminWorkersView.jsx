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
  Server,
  RefreshCw,
  Camera,
  PowerOff,
  RotateCcw,
  Trash2,
  CheckCircle2,
  AlertTriangle,
  Clock,
  Activity,
  Layers,
  Zap,
  Globe,
  ShieldCheck,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import {
  getAdminWorkerStatus,
  requestInspectNode,
  stopWorkerNode,
  resetWorkerNode,
  deleteWorkerNode,
  getAdminSearchLogs,
} from '@/lib/api'
import { toast } from 'sonner'

export default function AdminWorkersView() {
  const [status, setStatus] = useState(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState('')
  const [expandedScreenshot, setExpandedScreenshot] = useState({})
  const [nodeLogs, setNodeLogs] = useState([])
  const [loadingLogs, setLoadingLogs] = useState(false)

  const fetchStatus = async (showLoading = false) => {
    if (showLoading) setLoading(true)
    try {
      const res = await getAdminWorkerStatus()
      setStatus(res?.data || res)
    } catch {
      // background polling error
    } finally {
      setLoading(false)
    }
  }

  const fetchLogs = async () => {
    setLoadingLogs(true)
    try {
      const res = await getAdminSearchLogs({ limit: 50 })
      const list =
        res?.logs ||
        res?.data?.logs ||
        (Array.isArray(res?.data) ? res.data : Array.isArray(res) ? res : [])
      setNodeLogs(list)
    } catch {
      // ignore
    } finally {
      setLoadingLogs(false)
    }
  }

  useEffect(() => {
    fetchStatus(true)
    fetchLogs()
    const timer = setInterval(() => fetchStatus(false), 3000)
    return () => clearInterval(timer)
  }, [])

  const handleInspect = async (nodeId) => {
    setActionLoading(`inspect-${nodeId}`)
    try {
      await requestInspectNode(nodeId)
      toast.success(`تم إرسال أمر التقاط الشاشة المباشرة للعقدة (${nodeId})`)
      setTimeout(() => fetchStatus(false), 2500)
    } catch (err) {
      toast.error(err.message || 'فشل إرسال أمر الفحص')
    } finally {
      setActionLoading('')
    }
  }

  const handleReset = async (nodeId) => {
    setActionLoading(`reset-${nodeId}`)
    try {
      await resetWorkerNode(nodeId)
      toast.success(`تمت إعادة تعيين قاطع الحماية وتشغيل العقدة (${nodeId}) بنجاح`)
      await fetchStatus(false)
    } catch (err) {
      toast.error(err.message || 'فشل تشغيل العقدة')
    } finally {
      setActionLoading('')
    }
  }

  const handleStop = async (nodeId) => {
    if (!window.confirm(`هل أنت متأكد من تفعيل الإيقاف الاحترازي للعقدة (${nodeId})؟`)) return
    setActionLoading(`stop-${nodeId}`)
    try {
      await stopWorkerNode(nodeId)
      toast.success(`تم إيقاف العقدة (${nodeId}) احترازياً`)
      await fetchStatus(false)
    } catch (err) {
      toast.error(err.message || 'فشل إيقاف العقدة')
    } finally {
      setActionLoading('')
    }
  }

  const handleDelete = async (nodeId) => {
    if (!window.confirm(`هل تريد بالتأكيد إزالة العقدة (${nodeId}) من قائمة المراقبة؟`)) return
    setActionLoading(`delete-${nodeId}`)
    try {
      await deleteWorkerNode(nodeId)
      toast.success(`تمت إزالة العقدة (${nodeId}) من السجل`)
      await fetchStatus(false)
    } catch (err) {
      toast.error(err.message || 'فشل حذف العقدة')
    } finally {
      setActionLoading('')
    }
  }

  const toggleScreenshot = (nodeId) => {
    setExpandedScreenshot((prev) => ({
      ...prev,
      [nodeId]: !prev[nodeId],
    }))
  }

  const nodes = status?.nodes || []
  const totalNodes = status?.total_nodes || nodes.length || 0
  const onlineNodes = status?.online_nodes || nodes.filter((n) => n.connected).length || 0
  const offlineNodes = totalNodes - onlineNodes

  return (
    <AuthenticatedLayout title="شبكة حواسيب السحب (Cluster Farm)">
      <div className="space-y-6 text-right w-full" dir="rtl">
        {/* Top Header Card */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 bg-card p-5 rounded-lg border border-border shadow-xs">
          <div className="flex items-center gap-3">
            <div className="p-3 rounded-lg bg-primary/10 text-primary">
              <Server className="h-6 w-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-base font-bold text-foreground">
                  إدارة شبكة حواسيب كندا (Multi-Node Cluster)
                </h2>
                <Badge variant="outline" className="text-xs font-mono">
                  {onlineNodes}/{totalNodes} متصل
                </Badge>
              </div>
              <p className="text-xs text-muted-foreground mt-0.5">
                توزيع وتقسيم مهام الفحص اللحظية على عدة حواسيب بالتوازي ومراقبة أداء كل حاسبة
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                fetchStatus(true)
                fetchLogs()
              }}
              className="h-8 gap-1.5 text-xs cursor-pointer"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
              تحديث
            </Button>
          </div>
        </div>

        {/* Global Summary Stats */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card className="border-border bg-card shadow-xs">
            <CardContent className="p-4 flex items-center justify-between">
              <div>
                <p className="text-xs text-muted-foreground font-medium">إجمالي الحواسيب المسجلة</p>
                <p className="text-2xl font-black font-mono mt-1 text-foreground">{totalNodes}</p>
              </div>
              <div className="p-2.5 rounded-md bg-blue-500/10 text-blue-600 dark:text-blue-400">
                <Layers className="h-5 w-5" />
              </div>
            </CardContent>
          </Card>

          <Card className="border-border bg-card shadow-xs">
            <CardContent className="p-4 flex items-center justify-between">
              <div>
                <p className="text-xs text-muted-foreground font-medium">الحواسيب المتصلة والجاهزة</p>
                <p className="text-2xl font-black font-mono mt-1 text-emerald-600 dark:text-emerald-400">
                  {onlineNodes}
                </p>
              </div>
              <div className="p-2.5 rounded-md bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                <CheckCircle2 className="h-5 w-5" />
              </div>
            </CardContent>
          </Card>

          <Card className="border-border bg-card shadow-xs">
            <CardContent className="p-4 flex items-center justify-between">
              <div>
                <p className="text-xs text-muted-foreground font-medium">حواسيب متوقفة أو مفصولة</p>
                <p className="text-2xl font-black font-mono mt-1 text-rose-600 dark:text-rose-400">
                  {offlineNodes}
                </p>
              </div>
              <div className="p-2.5 rounded-md bg-rose-500/10 text-rose-600 dark:text-rose-400">
                <AlertTriangle className="h-5 w-5" />
              </div>
            </CardContent>
          </Card>

          <Card className="border-border bg-card shadow-xs">
            <CardContent className="p-4 flex items-center justify-between">
              <div>
                <p className="text-xs text-muted-foreground font-medium">إجمالي السحب اليومي للمزرعة</p>
                <p className="text-2xl font-black font-mono mt-1 text-foreground">
                  {status?.daily_pulls || 0}
                </p>
              </div>
              <div className="p-2.5 rounded-md bg-amber-500/10 text-amber-600 dark:text-amber-400">
                <Activity className="h-5 w-5" />
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Nodes Grid Cards */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-bold text-foreground flex items-center gap-2">
              <Globe className="h-4 w-4 text-primary" />
              حالة الحواسيب المتصلة (Nodes Status)
            </h3>
            <span className="text-xs text-muted-foreground">
              يتم تحديث الحالة واللقطات الحية كل 3 ثوانٍ
            </span>
          </div>

          {nodes.length === 0 ? (
            <Card className="p-8 text-center border-border bg-card">
              <div className="flex flex-col items-center justify-center gap-2 text-muted-foreground text-xs">
                <Server className="h-8 w-8 text-muted-foreground/60" />
                <p className="font-semibold text-foreground">لم يتم تسجيل أي حاسبة في الشبكة بعد</p>
                <p>
                  بمجرد تشغيل برنامج كندا على أي حاسبة وربطها بنفس المفتاح السري، ستظهر هنا تلقائياً.
                </p>
              </div>
            </Card>
          ) : (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
              {nodes.map((node) => {
                const isOnline = node.connected
                const hasBreaker = node.circuit_breaker_active
                const hasActiveVin = Boolean(node.active_vin)
                const isExpanded = expandedScreenshot[node.worker_id]
                const isBusyAction = actionLoading.endsWith(node.worker_id)

                return (
                  <Card
                    key={node.worker_id}
                    className={`border-2 transition-all shadow-xs bg-card ${
                      hasBreaker
                        ? 'border-rose-500/60 bg-rose-500/5'
                        : isOnline
                        ? 'border-emerald-500/40 hover:border-emerald-500'
                        : 'border-slate-300 dark:border-border opacity-90'
                    }`}
                  >
                    <CardHeader className="py-3 px-4 border-b border-border flex flex-row items-center justify-between gap-2">
                      <div className="flex items-center gap-2.5 min-w-0">
                        <span
                          className={`relative flex h-3 w-3 shrink-0 ${
                            isOnline ? '' : 'opacity-60'
                          }`}
                        >
                          {isOnline && (
                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                          )}
                          <span
                            className={`relative inline-flex rounded-full h-3 w-3 ${
                              isOnline ? 'bg-emerald-500' : 'bg-slate-400'
                            }`}
                          />
                        </span>
                        <div className="min-w-0">
                          <CardTitle className="text-sm font-bold truncate text-foreground">
                            {node.worker_name || node.worker_id}
                          </CardTitle>
                          <p className="text-[11px] font-mono text-muted-foreground truncate" dir="ltr">
                            {node.worker_id}
                          </p>
                        </div>
                      </div>

                      <div className="flex items-center gap-1.5 shrink-0">
                        {hasBreaker ? (
                          <Badge variant="destructive" className="text-[10px] gap-1">
                            <AlertTriangle className="h-3 w-3" />
                            قاطع الأمان مفعل
                          </Badge>
                        ) : isOnline ? (
                          <Badge
                            variant="outline"
                            className="border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 text-[10px] gap-1"
                          >
                            <CheckCircle2 className="h-3 w-3" />
                            متصل ونشط
                          </Badge>
                        ) : (
                          <Badge variant="secondary" className="text-[10px] text-muted-foreground">
                            غير متصل ({node.last_heartbeat_sec_ago > 9999 ? 'غير معروف' : `${node.last_heartbeat_sec_ago}s`})
                          </Badge>
                        )}
                      </div>
                    </CardHeader>

                    <CardContent className="p-4 space-y-3.5 text-xs">
                      {/* Critical Warning if circuit breaker is tripped on this node */}
                      {hasBreaker && (
                        <div className="p-2.5 rounded bg-rose-500/15 border border-rose-500/30 text-rose-800 dark:text-rose-200 text-xs">
                          <strong className="block font-bold">تنبيه حماية:</strong>
                          <span>{node.circuit_breaker_reason || 'تم إيقاف هذه العقدة لحماية الحساب'}</span>
                        </div>
                      )}

                      {/* Active In-Flight VIN on this node */}
                      {hasActiveVin ? (
                        <div className="p-3 rounded-md bg-amber-500/10 border border-amber-500/25 flex items-center justify-between gap-2">
                          <div className="flex items-center gap-2">
                            <Zap className="h-4 w-4 text-amber-500 animate-pulse shrink-0" />
                            <div>
                              <div className="flex items-center gap-1.5">
                                <span className="font-mono font-bold text-foreground" dir="ltr">
                                  {node.active_vin}
                                </span>
                                <Badge className="bg-amber-600 text-white text-[9px] px-1.5 py-0">
                                  قيد السحب الآن
                                </Badge>
                              </div>
                              <p className="text-[11px] text-muted-foreground mt-0.5">
                                {node.active_progress?.step_ar || node.active_progress?.step || 'معالجة...'}
                              </p>
                            </div>
                          </div>
                        </div>
                      ) : (
                        <div className="flex items-center justify-between py-1 text-muted-foreground">
                          <span className="flex items-center gap-1">
                            <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500" />
                            في وضع الاستعداد (جاهز للمهمة التالية)
                          </span>
                          {node.cooldown_remaining_sec > 0 && (
                            <span className="font-mono text-[11px] text-amber-600 dark:text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded">
                              تهدئة: {node.cooldown_remaining_sec}s
                            </span>
                          )}
                        </div>
                      )}

                      {/* Metrics Row */}
                      <div className="grid grid-cols-3 gap-2 py-2 border-y border-border/60 text-center font-mono">
                        <div className="bg-muted/40 p-2 rounded">
                          <span className="text-[10px] text-muted-foreground block font-sans">
                            سحب اليوم
                          </span>
                          <span className="text-sm font-bold text-foreground">
                            {node.daily_pulls || 0}
                          </span>
                        </div>
                        <div className="bg-muted/40 p-2 rounded">
                          <span className="text-[10px] text-muted-foreground block font-sans">
                            حالة المتصفح
                          </span>
                          <span className="text-xs font-bold text-foreground font-sans">
                            {node.telemetry?.session_active ? 'مسجل دخول' : node.telemetry ? 'يحتاج تسجيل' : '—'}
                          </span>
                        </div>
                        <div className="bg-muted/40 p-2 rounded">
                          <span className="text-[10px] text-muted-foreground block font-sans">
                            آخر نبض
                          </span>
                          <span className="text-xs font-bold text-foreground">
                            {node.last_heartbeat_sec_ago < 9999 ? `${node.last_heartbeat_sec_ago}s ago` : 'منذ فترة'}
                          </span>
                        </div>
                      </div>

                      {/* Live Screenshot toggle & view */}
                      {node.last_screenshot_base64 && (
                        <div className="space-y-2">
                          <button
                            type="button"
                            onClick={() => toggleScreenshot(node.worker_id)}
                            className="w-full flex items-center justify-between p-2 rounded bg-muted/30 hover:bg-muted/60 text-xs font-medium text-foreground transition-colors cursor-pointer"
                          >
                            <span className="flex items-center gap-1.5">
                              <Camera className="h-3.5 w-3.5 text-primary" />
                              لقطة الشاشة الحية لمتصفح هذه الحاسبة
                            </span>
                            {isExpanded ? (
                              <ChevronUp className="h-3.5 w-3.5" />
                            ) : (
                              <ChevronDown className="h-3.5 w-3.5" />
                            )}
                          </button>

                          {isExpanded && (
                            <div className="rounded-lg overflow-hidden border border-border bg-black/90 p-1">
                              <img
                                src={`data:image/jpeg;base64,${node.last_screenshot_base64}`}
                                alt={`Live Screen ${node.worker_id}`}
                                className="w-full h-auto rounded object-contain max-h-80"
                              />
                            </div>
                          )}
                        </div>
                      )}

                      {/* Action Buttons for this node */}
                      <div className="flex flex-wrap items-center justify-between gap-2 pt-1">
                        <div className="flex items-center gap-1.5">
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleInspect(node.worker_id)}
                            disabled={isBusyAction}
                            className="h-7 text-[11px] gap-1 cursor-pointer"
                          >
                            <Camera className="h-3 w-3" />
                            تصوير الشاشة
                          </Button>

                          {hasBreaker ? (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => handleReset(node.worker_id)}
                              disabled={isBusyAction}
                              className="h-7 text-[11px] gap-1 text-emerald-600 border-emerald-500/40 hover:bg-emerald-500/10 cursor-pointer"
                            >
                              <RotateCcw className="h-3 w-3" />
                              إعادة التشغيل
                            </Button>
                          ) : (
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => handleStop(node.worker_id)}
                              disabled={isBusyAction}
                              className="h-7 text-[11px] gap-1 text-rose-600 hover:bg-rose-500/10 cursor-pointer"
                            >
                              <PowerOff className="h-3 w-3" />
                              إيقاف احترازي
                            </Button>
                          )}
                        </div>

                        {!isOnline && (
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => handleDelete(node.worker_id)}
                            disabled={isBusyAction}
                            title="حذف هذه العقدة من سجل الشبكة"
                            className="h-7 px-2 text-[11px] text-muted-foreground hover:text-destructive cursor-pointer"
                          >
                            <Trash2 className="h-3 w-3" />
                          </Button>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          )}
        </div>

        {/* Node Task Execution Audit Table */}
        <Card className="border-border bg-card shadow-xs overflow-hidden">
          <CardHeader className="py-3.5 px-4 sm:px-5 border-b border-border flex flex-row items-center justify-between">
            <div className="flex items-center gap-2">
              <Activity className="h-4 w-4 text-primary" />
              <CardTitle className="text-sm font-bold text-foreground">
                سجل إسناد وتوزيع المهام على الحواسيب (Node Execution Audit)
              </CardTitle>
            </div>
            <span className="text-xs text-muted-foreground">آخر 50 عملية</span>
          </CardHeader>
          <CardContent className="p-0">
            {loadingLogs ? (
              <div className="p-8 text-center text-xs text-muted-foreground flex items-center justify-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin text-primary" />
                <span>جاري تحميل السجل...</span>
              </div>
            ) : nodeLogs.length === 0 ? (
              <div className="p-8 text-center text-xs text-muted-foreground">
                لا توجد عمليات مسجلة في السجل حالياً.
              </div>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow className="bg-muted/40 hover:bg-muted/40 text-xs">
                      <TableHead className="text-right text-xs">#</TableHead>
                      <TableHead className="text-right text-xs">رقم الشاصي</TableHead>
                      <TableHead className="text-right text-xs">الحاسبة المنفذة (Node)</TableHead>
                      <TableHead className="text-right text-xs">المستخدم</TableHead>
                      <TableHead className="text-right text-xs">النتيجة</TableHead>
                      <TableHead className="text-right text-xs">الاستغراق</TableHead>
                      <TableHead className="text-right text-xs">التاريخ والوقت</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {nodeLogs.slice(0, 30).map((log) => (
                      <TableRow key={log.id} className="hover:bg-muted/30 text-xs">
                        <TableCell className="font-mono text-muted-foreground">
                          {log.id}
                        </TableCell>
                        <TableCell className="font-mono font-bold text-foreground" dir="ltr">
                          {log.vin}
                        </TableCell>
                        <TableCell>
                          {log.worker_id ? (
                            <span className="inline-flex items-center gap-1 font-mono text-[11px] bg-primary/10 text-primary px-2 py-0.5 rounded">
                              <Server className="h-3 w-3" />
                              {log.worker_name || log.worker_id}
                            </span>
                          ) : (
                            <span className="text-muted-foreground font-mono text-[11px]">
                              {log.fetch_source === 'cache' ? 'ذاكرة التخزين (Cache)' : 'حاسبة 1'}
                            </span>
                          )}
                        </TableCell>
                        <TableCell className="text-foreground">
                          {log.user?.username ? `@${log.user.username}` : `UID: ${log.user_id}`}
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant="outline"
                            className={`text-[10px] ${
                              log.result_status === 'success'
                                ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
                                : 'border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-400'
                            }`}
                          >
                            {log.result_status === 'success' ? 'نجاح' : 'خطأ / فشل'}
                          </Badge>
                        </TableCell>
                        <TableCell className="font-mono text-muted-foreground">
                          {log.duration_ms ? `${(log.duration_ms / 1000).toFixed(1)}s` : '—'}
                        </TableCell>
                        <TableCell className="font-mono text-muted-foreground" dir="ltr">
                          {new Date(log.created_at).toLocaleString('en-US', {
                            month: 'short',
                            day: 'numeric',
                            hour: '2-digit',
                            minute: '2-digit',
                            hour12: true,
                          })}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </AuthenticatedLayout>
  )
}
