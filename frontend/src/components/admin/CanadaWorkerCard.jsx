import React, { useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  ShieldAlert,
  ShieldCheck,
  RefreshCw,
  Camera,
  Clock,
  Zap,
  Ban,
  CheckCircle2,
  XCircle,
  Monitor,
  Loader2,
  Eye,
  EyeOff,
  Radio,
} from 'lucide-react'
import { toast } from 'sonner'
import { api } from '@/lib/api'

export default function CanadaWorkerCard({ status, onRefresh }) {
  const [inspecting, setInspecting] = useState(false)
  const [resetting, setResetting] = useState(false)
  const [stopping, setStopping] = useState(false)
  const [togglingMode, setTogglingMode] = useState(false)
  const [showScreenshot, setShowScreenshot] = useState(true)

  if (!status) {
    return (
      <Card className="border-slate-200 dark:border-slate-800 shadow-sm w-full bg-white dark:bg-slate-900 rounded-xl p-6 text-center">
        <div className="flex items-center justify-center gap-2 text-slate-500 text-xs">
          <Loader2 className="h-4 w-4 animate-spin text-[#003478]" />
          <span>جاري الاتصال بجسر حاسبة كندا...</span>
        </div>
      </Card>
    )
  }

  const {
    connected,
    circuit_breaker_active,
    circuit_breaker_reason,
    daily_pulls = 0,
    max_daily_pulls = 45,
    cooldown_remaining_sec = 0,
    queue_length = 0,
    use_mock_fallback = true,
    inspect_pending = false,
    telemetry,
    active_progress,
    last_screenshot_base64,
    last_screenshot_at,
  } = status

  const pullPercentage = Math.min(100, Math.round((daily_pulls / max_daily_pulls) * 100))

  const handleRequestInspect = async () => {
    setInspecting(true)
    try {
      const res = await api.post('/admin/worker-inspect', {})
      toast.success(res?.message || 'تم إرسال طلب فحص الشاشة والتقاط Screenshot لحاسبة كندا')
      setShowScreenshot(true)
      setTimeout(() => {
        if (onRefresh) onRefresh()
      }, 2200)
    } catch (err) {
      toast.error(err.message || 'تعذر إرسال طلب الفحص')
    } finally {
      setTimeout(() => setInspecting(false), 2200)
    }
  }

  const handleEmergencyStop = async () => {
    setStopping(true)
    try {
      const res = await api.post('/admin/worker-stop', {})
      toast.warning(res?.message || 'تم تفعيل الإيقاف الاحترازي لحاسبة كندا')
      if (onRefresh) onRefresh()
    } catch (err) {
      toast.error('فشل إرسال أمر الإيقاف')
    } finally {
      setStopping(false)
    }
  }

  const handleResetBreaker = async () => {
    setResetting(true)
    try {
      await api.post('/admin/worker-reset', {})
      toast.success('تمت إعادة تفعيل قاطع الأمان واستئناف العمل بنجاح')
      if (onRefresh) onRefresh()
    } catch (err) {
      toast.error('فشل إعادة تفعيل القاطع')
    } finally {
      setResetting(false)
    }
  }

  const handleToggleMode = async () => {
    setTogglingMode(true)
    try {
      const nextVal = !use_mock_fallback
      const res = await api.post('/admin/worker-mode', { use_mock_fallback: nextVal })
      toast.info(res?.message || 'تم تحديث وضع التشغيل')
      if (onRefresh) onRefresh()
    } catch (err) {
      toast.error('تعذر تغيير وضع التشغيل')
    } finally {
      setTogglingMode(false)
    }
  }

  return (
    <Card className="border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden w-full bg-white dark:bg-slate-900 rounded-xl">
      {/* Top Header Bar */}
      <div className="bg-[#003478] text-white px-5 py-3.5 flex flex-wrap items-center justify-between gap-3 text-right">
        <div className="flex items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-white/10 text-white font-bold text-base shadow-xs">
            🍁
          </div>
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="text-sm sm:text-base font-bold tracking-wide">
                غرفة عمليات حاسبة كندا (Canada CARFAXOnline Node)
              </h3>
              {circuit_breaker_active ? (
                <Badge className="bg-[#D0202E] hover:bg-[#D0202E] text-white text-[11px] border-none font-bold">
                  🛑 متوقف احترازياً لحماية الحساب (Frozen)
                </Badge>
              ) : connected ? (
                <Badge className="bg-[#1C8237] hover:bg-[#1C8237] text-white text-[11px] gap-1.5 border-none font-bold">
                  <span className="h-2 w-2 rounded-full bg-white animate-ping"></span>
                  متصل حياً بحاسبة كندا
                </Badge>
              ) : (
                <Badge className="bg-amber-500 hover:bg-amber-500 text-white text-[11px] border-none font-bold">
                  غير متصل ببرنامج كندا حالياً
                </Badge>
              )}
            </div>
            <p className="text-[11px] text-blue-200 mt-0.5">
              نظام المحاولة الواحدة أو التجميد الفوري (One-Shot Or Freeze) — البوابة المستهدفة: http://carfaxonline.com
            </p>
          </div>
        </div>

        {/* Action Controls */}
        <div className="flex flex-wrap items-center gap-2">
          <Button
            size="sm"
            onClick={handleRequestInspect}
            disabled={inspecting || inspect_pending}
            className="bg-[#00599C] hover:bg-[#004880] text-white text-xs h-8 gap-1.5 font-bold border border-white/20"
          >
            {inspecting || inspect_pending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Camera className="h-3.5 w-3.5" />
            )}
            <span>فحص شاشة كندا + Screenshot</span>
          </Button>

          {circuit_breaker_active ? (
            <Button
              size="sm"
              onClick={handleResetBreaker}
              disabled={resetting}
              className="bg-[#1C8237] hover:bg-emerald-700 text-white text-xs h-8 gap-1.5 font-bold"
            >
              <Zap className="h-3.5 w-3.5" />
              <span>إعادة تفعيل القاطع بعد التأكد</span>
            </Button>
          ) : (
            <Button
              variant="destructive"
              size="sm"
              onClick={handleEmergencyStop}
              disabled={stopping}
              className="bg-[#D0202E] hover:bg-red-700 text-white text-xs h-8 gap-1.5 font-bold"
            >
              <Ban className="h-3.5 w-3.5" />
              <span>إيقاف طوارئ</span>
            </Button>
          )}

          {onRefresh && (
            <Button
              variant="outline"
              size="sm"
              onClick={onRefresh}
              className="bg-white/10 hover:bg-white/20 text-white border-white/20 text-xs h-8 gap-1"
            >
              <RefreshCw className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>
      </div>

      <CardContent className="p-5 space-y-4">
        {/* Circuit Breaker Alert Banner */}
        {circuit_breaker_active && (
          <div className="rounded-xl bg-[#FDF2F2] dark:bg-rose-950/30 border-2 border-[#D0202E] p-4 text-xs text-[#D0202E] flex items-start justify-between gap-3 text-right">
            <div className="flex items-start gap-2.5">
              <ShieldAlert className="h-5 w-5 shrink-0 mt-0.5" />
              <div>
                <p className="font-bold text-sm">
                  توقف احترازي لحماية حساب كارفاكس كندا من الحظر (One-Shot Safety Freeze)
                </p>
                <p className="text-xs mt-1 font-semibold text-slate-800 dark:text-slate-200">
                  السبب: {circuit_breaker_reason}
                </p>
                <p className="text-[11px] mt-1 text-slate-600 dark:text-slate-400">
                  لم يقم البرنامج بأي محاولة متكررة. راجع لقطة الشاشة أدناه وتأكد من أن متصفح كندا مفتوح على صفحة البحث في carfaxonline.com ثم اضغط "إعادة تفعيل القاطع".
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Active Live Harvest Progress Step */}
        {active_progress && (
          <div className="rounded-xl bg-blue-50 dark:bg-blue-950/40 border border-blue-300 dark:border-blue-800 p-3.5 flex items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <Loader2 className="h-5 w-5 animate-spin text-[#003478]" />
              <div>
                <div className="flex items-center gap-2">
                  <span className="text-xs font-bold text-[#003478] dark:text-blue-300">
                    جاري التنفيذ الحي في حاسبة كندا للشاصي:
                  </span>
                  <span className="font-mono font-bold text-xs bg-white px-2 py-0.5 rounded border border-blue-200" dir="ltr">
                    {active_progress.vin}
                  </span>
                </div>
                <p className="text-xs font-semibold text-slate-700 dark:text-slate-200 mt-1">
                  {active_progress.step_ar}
                </p>
              </div>
            </div>
            <Badge className="bg-[#003478] text-white text-[10px] font-mono">
              {active_progress.step}
            </Badge>
          </div>
        )}

        {/* 4-Pillar Readiness Checklist for carfaxonline.com */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 text-right">
          {/* 1. Chrome Connection */}
          <div className="p-3.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50/70 dark:bg-slate-800/40">
            <div className="flex items-center justify-between mb-1">
              <span className="text-[11px] font-bold text-slate-500">1. متصفح كروم كندا (CDP 9222)</span>
              {telemetry?.chrome_connected ? (
                <CheckCircle2 className="h-4 w-4 text-[#1C8237]" />
              ) : (
                <XCircle className="h-4 w-4 text-slate-400" />
              )}
            </div>
            <p className={`text-xs font-bold ${telemetry?.chrome_connected ? 'text-[#1C8237]' : 'text-slate-500'}`}>
              {telemetry?.chrome_connected ? 'متصل وجاهز' : 'بانتظار اتصال متصفح كندا'}
            </p>
          </div>

          {/* 2. Target Portal Tab */}
          <div className="p-3.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50/70 dark:bg-slate-800/40">
            <div className="flex items-center justify-between mb-1">
              <span className="text-[11px] font-bold text-slate-500">2. تبويب carfaxonline.com</span>
              {telemetry?.carfax_tab_found ? (
                <CheckCircle2 className="h-4 w-4 text-[#1C8237]" />
              ) : (
                <XCircle className="h-4 w-4 text-amber-500" />
              )}
            </div>
            <p className="text-xs font-bold text-slate-800 dark:text-slate-200 truncate" dir="ltr" title={telemetry?.current_url}>
              {telemetry?.current_url || 'لم يتم رصد رابط بعد'}
            </p>
          </div>

          {/* 3. VIN Input Detection */}
          <div className="p-3.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50/70 dark:bg-slate-800/40">
            <div className="flex items-center justify-between mb-1">
              <span className="text-[11px] font-bold text-slate-500">3. حقل إدخال الشاصي (VIN Box)</span>
              {telemetry?.vin_input_found ? (
                <CheckCircle2 className="h-4 w-4 text-[#1C8237]" />
              ) : (
                <XCircle className="h-4 w-4 text-amber-500" />
              )}
            </div>
            <p className={`text-xs font-bold ${telemetry?.vin_input_found ? 'text-[#1C8237]' : 'text-amber-600'}`}>
              {telemetry?.vin_input_found
                ? `متاح وجاهز (${telemetry.vin_input_selector})`
                : 'غير ظاهر في الشاشة الحالية'}
            </p>
          </div>

          {/* 4. Session & Anti-Ban Cooldown */}
          <div className="p-3.5 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50/70 dark:bg-slate-800/40">
            <div className="flex items-center justify-between mb-1">
              <span className="text-[11px] font-bold text-slate-500">4. السحب اليومي والتهدئة</span>
              <Clock className="h-4 w-4 text-[#003478]" />
            </div>
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold font-mono text-slate-800 dark:text-slate-200">
                {daily_pulls} / {max_daily_pulls} اليوم
              </span>
              {cooldown_remaining_sec > 0 ? (
                <span className="text-[11px] font-bold text-amber-600">
                  تهدئة: {cooldown_remaining_sec} ث
                </span>
              ) : (
                <span className="text-[11px] font-bold text-[#1C8237]">جاهز فوراً</span>
              )}
            </div>
            <div className="mt-1.5 h-1.5 w-full bg-slate-200 rounded-full overflow-hidden">
              <div
                className={`h-full ${pullPercentage > 85 ? 'bg-amber-500' : 'bg-[#003478]'}`}
                style={{ width: `${pullPercentage}%` }}
              />
            </div>
          </div>
        </div>

        {/* Status Message & Mode Switch Bar */}
        <div className="flex flex-wrap items-center justify-between gap-3 pt-2 border-t border-slate-100 dark:border-slate-800 text-xs">
          <div className="flex items-center gap-2 text-slate-600 dark:text-slate-300">
            <ShieldCheck className="h-4 w-4 text-[#003478] shrink-0" />
            <span className="font-semibold">
              {telemetry?.status_message || 'بانتظار تشغيل برنامج canada_worker.exe على حاسبة كندا...'}
            </span>
          </div>

          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={handleToggleMode}
              disabled={togglingMode}
              className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[11px] font-bold border transition-colors ${
                !use_mock_fallback
                  ? 'bg-[#EBF7EE] text-[#1C8237] border-emerald-300'
                  : 'bg-amber-50 text-amber-800 border-amber-300'
              }`}
            >
              <Radio className="h-3 w-3" />
              {!use_mock_fallback
                ? 'وضع التشغيل: حاسبة كندا الحقيقية فقط (Strict Live)'
                : 'وضع التشغيل: مسموح بالمحاكاة عند إغلاق كندا (اضغط للتبديل)'}
            </button>

            {last_screenshot_base64 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowScreenshot(!showScreenshot)}
                className="text-xs h-7 gap-1 text-[#00599C]"
              >
                {showScreenshot ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                <span>{showScreenshot ? 'إخفاء شاشة كندا' : 'عرض شاشة كندا'}</span>
              </Button>
            )}
          </div>
        </div>

        {/* Live Compressed JPEG Screenshot Viewer */}
        {showScreenshot && last_screenshot_base64 && (
          <div className="rounded-xl border-2 border-slate-200 dark:border-slate-700 overflow-hidden bg-slate-900">
            <div className="bg-slate-800 text-slate-200 px-4 py-2 flex items-center justify-between text-xs">
              <div className="flex items-center gap-2">
                <Monitor className="h-4 w-4 text-blue-400" />
                <span className="font-bold">لقطة شاشة حية مضغوطة من متصفح كندا (Optimized JPEG 60%)</span>
              </div>
              {last_screenshot_at && (
                <span className="text-[11px] text-slate-400 font-mono">
                  آخر تحديث: {new Date(last_screenshot_at).toLocaleTimeString('ar-EG')}
                </span>
              )}
            </div>
            <div className="p-2 flex justify-center bg-slate-950">
              <img
                src={last_screenshot_base64}
                alt="Canada Chrome Live View"
                className="max-h-[420px] w-auto object-contain rounded border border-slate-800"
              />
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
