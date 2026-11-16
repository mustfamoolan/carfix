import React, { useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { RefreshCw, Camera, Loader2, CheckCircle2, AlertCircle, RotateCcw } from 'lucide-react'
import { toast } from 'sonner'
import { api } from '@/lib/api'

export default function CanadaWorkerCard({ status, userQuota, onRefresh }) {
  const [inspecting, setInspecting] = useState(false)
  const [resetting, setResetting] = useState(false)
  const [previewOpen, setPreviewOpen] = useState(false)
  const [selectedNodeId, setSelectedNodeId] = useState('')

  if (!status) {
    return null
  }

  const {
    connected,
    circuit_breaker_active,
    circuit_breaker_reason,
    daily_pulls = 0,
    max_daily_pulls,
    cooldown_remaining_sec = 0,
    inspect_pending = false,
    telemetry,
    active_progress,
    last_screenshot_base64,
    last_screenshot_at,
    nodes = [],
    online_nodes = 0,
    total_nodes = 0,
  } = status

  const activeNode =
    nodes.find((n) => n.worker_id === selectedNodeId) ||
    nodes.find((n) => n.connected) ||
    nodes[0] ||
    null

  const activePreviewSc = activeNode?.last_screenshot_base64 || last_screenshot_base64
  const activePreviewAt = activeNode?.last_screenshot_at || last_screenshot_at
  const activePreviewTelemetry = activeNode?.telemetry || telemetry

  const effectiveDailyLimit =
    userQuota?.daily_limit > 0
      ? userQuota.daily_limit
      : userQuota?.total_limit > 0
      ? userQuota.total_limit
      : max_daily_pulls || 0

  const handleRequestInspect = async (targetNodeId = '') => {
    const nodeToInspect = targetNodeId || activeNode?.worker_id || ''
    if (nodeToInspect) {
      setSelectedNodeId(nodeToInspect)
    }
    setInspecting(true)
    setPreviewOpen(true)
    try {
      await api.post('/admin/worker-inspect', nodeToInspect ? { node_id: nodeToInspect } : {})
      setTimeout(() => {
        if (onRefresh) onRefresh()
      }, 2000)
    } catch (err) {
      toast.error(err.message || 'تعذر طلب لقطة الشاشة')
    } finally {
      setTimeout(() => setInspecting(false), 2000)
    }
  }

  const handleResetBreaker = async () => {
    setResetting(true)
    try {
      await api.post('/admin/worker-reset', {})
      toast.success('تمت إعادة تفعيل الاتصال بنجاح')
      if (onRefresh) onRefresh()
    } catch {
      toast.error('فشل إعادة التفعيل')
    } finally {
      setResetting(false)
    }
  }

  return (
    <>
      <Card className="border-border shadow-xs bg-card">
        <CardContent className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          {/* Right Side: Clean Connection State & Daily Counter */}
          <div className="flex flex-wrap items-center gap-3 text-xs">
            {circuit_breaker_active ? (
              <Badge
                variant="outline"
                className="border-destructive/30 bg-destructive/10 text-destructive font-semibold gap-1.5 py-1 px-2.5"
              >
                <AlertCircle className="h-3.5 w-3.5" />
                متوقف احترازياً
              </Badge>
            ) : connected ? (
              <Badge
                variant="outline"
                className="border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 font-semibold gap-1.5 py-1 px-2.5"
              >
                <CheckCircle2 className="h-3.5 w-3.5" />
                حواسيب كندا متصلة ({online_nodes || 1}/{total_nodes || 1})
              </Badge>
            ) : (
              <Badge variant="destructive" className="bg-rose-500 hover:bg-rose-600 text-white font-semibold py-1 px-2.5 animate-pulse gap-1">
                <AlertCircle className="h-3.5 w-3.5" />
                حاسبة كندا غير متصلة!
              </Badge>
            )}

            <span className="text-muted-foreground">
              السحب اليومي:{' '}
              <strong className="text-foreground font-mono">
                {effectiveDailyLimit > 0 ? `${daily_pulls} / ${effectiveDailyLimit}` : `${daily_pulls} (مفتوح)`}
              </strong>
            </span>

            {cooldown_remaining_sec > 0 && (
              <span className="text-amber-600 font-medium">
                (تهدئة: {cooldown_remaining_sec} ثانية)
              </span>
            )}

            {telemetry?.vin_input_found && connected && !circuit_breaker_active && (
              <span className="text-muted-foreground hidden md:inline">
                • جاهز للفحص المباشر
              </span>
            )}
          </div>

          {/* Left Side: Minimal Controls */}
          <div className="flex items-center gap-2">
            {circuit_breaker_active && (
              <Button
                size="sm"
                variant="outline"
                onClick={handleResetBreaker}
                disabled={resetting}
                className="h-8 text-xs gap-1.5 border-primary/30 text-primary hover:bg-primary/5"
              >
                <RotateCcw className="h-3.5 w-3.5" />
                <span>إعادة تفعيل</span>
              </Button>
            )}

            <Button
              size="sm"
              variant="outline"
              onClick={() => handleRequestInspect('')}
              disabled={inspecting || inspect_pending}
              className="h-8 text-xs gap-1.5"
            >
              {inspecting || inspect_pending ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Camera className="h-3.5 w-3.5" />
              )}
              <span>معاينة شاشة كندا</span>
            </Button>

            {onRefresh && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onRefresh}
                className="h-8 w-8 p-0 text-muted-foreground hover:text-foreground"
                title="تحديث الحالة"
              >
                <RefreshCw className="h-3.5 w-3.5" />
              </Button>
            )}
          </div>
        </CardContent>

        {/* Active Pull Progress (Only visible while pulling a report) */}
        {active_progress && (
          <div className="border-t border-border bg-muted/40 px-4 py-2.5 flex items-center justify-between text-xs">
            <div className="flex items-center gap-2">
              <Loader2 className="h-3.5 w-3.5 animate-spin text-primary" />
              <span className="font-medium text-foreground">{active_progress.step_ar}</span>
            </div>
            <span className="font-mono text-muted-foreground" dir="ltr">
              {active_progress.vin}
            </span>
          </div>
        )}

        {/* Reason banner ONLY if paused */}
        {circuit_breaker_active && circuit_breaker_reason && (
          <div className="border-t border-destructive/20 bg-destructive/5 px-4 py-2 text-xs text-destructive">
            سبب التوقف: {circuit_breaker_reason}
          </div>
        )}
      </Card>

      {/* Clean Modal for Screenshot Preview (Only opens when requested, with per-PC selector) */}
      <Dialog open={previewOpen} onOpenChange={setPreviewOpen}>
        <DialogContent className="max-w-3xl" dir="rtl">
          <DialogHeader>
            <DialogTitle className="text-sm font-bold text-right">
              معاينة متصفح حواسيب كندا ({activeNode?.worker_name || activeNode?.worker_id || 'حاسبة كندا'})
            </DialogTitle>
            <DialogDescription className="text-xs text-right">
              {activePreviewTelemetry?.status_message || 'لقطة شاشة حية لمتصفح كروم في كندا'}
              {activePreviewAt &&
                ` — آخر تحديث: ${new Date(activePreviewAt).toLocaleTimeString('ar-EG')}`}
            </DialogDescription>
          </DialogHeader>

          {nodes.length > 1 && (
            <div className="flex flex-wrap items-center gap-2 pb-1">
              {nodes.map((n) => {
                const isSelected = (activeNode?.worker_id || '') === n.worker_id
                return (
                  <Button
                    key={n.worker_id}
                    type="button"
                    size="sm"
                    variant={isSelected ? 'default' : 'outline'}
                    onClick={() => handleRequestInspect(n.worker_id)}
                    className="h-8 text-xs gap-1.5 cursor-pointer"
                  >
                    <Camera className="h-3.5 w-3.5" />
                    <span>{n.worker_name || n.worker_id}</span>
                  </Button>
                )
              })}
            </div>
          )}

          <div className="rounded-lg border border-border overflow-hidden bg-muted/30 flex items-center justify-center min-h-[260px]">
            {activePreviewSc ? (
              <img
                src={
                  activePreviewSc.startsWith('data:image')
                    ? activePreviewSc
                    : `data:image/jpeg;base64,${activePreviewSc.replace(/^[\s\r\n]+/, '')}`
                }
                alt="Canada Browser"
                className="max-h-[460px] w-auto object-contain"
              />
            ) : (
              <div className="text-xs text-muted-foreground py-12 flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin text-primary" />
                <span>جاري جلب لقطة الشاشة من حاسبة كندا...</span>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
