import React, { useState, useEffect } from 'react'
import { Badge } from '@/components/ui/badge'
import { Loader2, CheckCircle2, Sparkles, AlertTriangle, Car, Zap } from 'lucide-react'
import { toast } from 'sonner'
import {
  api,
  getCachedVinSpecs,
  setCachedVinSpecs,
} from '@/lib/api'

// Instant 0ms ISO 3779 / NHTSA VIN Check-Digit & Structure Validator
function validateVinLocally(vinStr) {
  const clean = String(vinStr || '').trim().toUpperCase()
  if (clean.length !== 17) {
    return { valid: false, error: 'رقم الشاصي (VIN) يجب أن يتكون من 17 رمزاً' }
  }
  if (!/^[A-HJ-NPR-Z0-9]{17}$/.test(clean)) {
    return { valid: false, error: 'رقم الشاصي يحتوي على رموز غير مسموحة' }
  }
  // Reject trivial fake patterns (e.g. 11111111111111111)
  if (/^(.)\1{10,}$/.test(clean) || /^.{3}(.)\1{11,}$/.test(clean)) {
    return { valid: false, error: 'رقم الشاصي غير صحيح (نمط متكرر غير صالح)' }
  }
  const translit = {
    A: 1, B: 2, C: 3, D: 4, E: 5, F: 6, G: 7, H: 8,
    J: 1, K: 2, L: 3, M: 4, N: 5, P: 7, R: 9,
    S: 2, T: 3, U: 4, V: 5, W: 6, X: 7, Y: 8, Z: 9,
    0: 0, 1: 1, 2: 2, 3: 3, 4: 4, 5: 5, 6: 6, 7: 7, 8: 8, 9: 9,
  }
  const weights = [8, 7, 6, 5, 4, 3, 2, 10, 0, 9, 8, 7, 6, 5, 4, 3, 2]
  let sum = 0
  for (let i = 0; i < 17; i++) {
    const val = translit[clean[i]]
    if (val === undefined) {
      return { valid: false, error: 'رقم الشاصي غير صحيح' }
    }
    sum += val * weights[i]
  }
  const rem = sum % 11
  const expectedCheck = rem === 10 ? 'X' : String(rem)
  if (clean[8] !== expectedCheck) {
    return {
      valid: false,
      error: 'رقم الشاصي غير صحيح: يوجد خطأ في أحد الأرقام أو الحروف (Check Digit غير مطابق)',
    }
  }
  return { valid: true, error: '' }
}

export default function VinSearchBox({ onQueueSearch, activeSearchCount = 0, inputRef }) {
  const [vin, setVin] = useState('')
  const [activeTab, setActiveTab] = useState('vin')
  const [specsLoading, setSpecsLoading] = useState(false)
  const [vehicleSpecs, setVehicleSpecs] = useState(null)
  const [vinCheckError, setVinCheckError] = useState('')
  const [lastQueuedVin, setLastQueuedVin] = useState('')

  // Instant pre-search VIN validation + Factory Specs lookup as soon as 17 chars are entered
  useEffect(() => {
    if (vin.length === 17) {
      const localCheck = validateVinLocally(vin)
      if (!localCheck.valid) {
        setVehicleSpecs(null)
        setVinCheckError(localCheck.error)
        setSpecsLoading(false)
        return
      }
      setVinCheckError('')
      verifyAndDecodeVin(vin)
    } else {
      setVehicleSpecs(null)
      setVinCheckError('')
      setSpecsLoading(false)
    }
  }, [vin])

  const verifyAndDecodeVin = async (vinCode) => {
    const cached = getCachedVinSpecs(vinCode)
    if (cached) {
      setVehicleSpecs(cached.specs)
      setVinCheckError(cached.error || '')
      setSpecsLoading(false)
      return
    }

    setSpecsLoading(true)
    setVinCheckError('')
    try {
      // 1. Strict Backend VIN Check Digit & Structure Verification
      try {
        const valRes = await api.post('/tools/vin-validate', { vin: vinCode })
        if (valRes && valRes.is_valid === false) {
          const msg =
            valRes.error_msg ||
            'رقم الشاصي غير صحيح: يوجد خطأ في أحد الأرقام أو الحروف، يرجى مراجعة الرقم بدقة'
          setVehicleSpecs(null)
          setVinCheckError(msg)
          setCachedVinSpecs(vinCode, { specs: null, error: msg })
          setSpecsLoading(false)
          return
        }
      } catch {
        // Continue to factory lookup if local check endpoint had a transient network blip
      }

      // 2. Vehicle Factory Specifications Lookup
      const res = await fetch(
        `https://vpic.nhtsa.dot.gov/api/vehicles/DecodeVinValues/${vinCode}?format=json`
      )
      const data = await res.json()
      if (data?.Results?.[0]) {
        const item = data.Results[0]
        const errCodes = String(item.ErrorCode || '')
          .split(',')
          .map((c) => c.trim())
        const hasFatalVinError =
          errCodes.includes('1') || errCodes.includes('6') || errCodes.includes('7')
        const realMake = (item.Make || '').trim()
        const realYear = (item.ModelYear || '').trim()
        const realModel = (item.Model || item.Series || '').trim()

        if (realMake && realYear && !hasFatalVinError) {
          const engineParts = [
            item.DisplacementL ? `${Number(item.DisplacementL).toFixed(1)}L` : '',
            item.EngineConfiguration || '',
            item.EngineCylinders ? `${item.EngineCylinders}-Cyl` : '',
          ]
            .filter(Boolean)
            .join(' ')

          const decoded = {
            year: realYear,
            make: realMake,
            model: realModel,
            trim: (item.Trim || '').trim(),
            title: `${realYear} ${realMake} ${realModel} ${item.Trim || ''}`.replace(/\s+/g, ' ').trim(),
            body: (item.BodyClass || item.VehicleType || 'غير محدد').trim(),
            engine: (engineParts || item.EngineModel || item.FuelTypePrimary || 'غير محدد').trim(),
            drive: (item.DriveType || '').trim(),
            fuel: (item.FuelTypePrimary || 'غير محدد').trim(),
            country: (item.PlantCountry || item.PlantState || 'غير محدد').trim(),
            manufacturer: (item.Manufacturer || '').trim(),
            isCheckDigitValid: true,
          }
          setVehicleSpecs(decoded)
          setVinCheckError('')
          setCachedVinSpecs(vinCode, { specs: decoded, error: '' })
        } else {
          const errMsg =
            'رقم الشاصي غير صحيح أو غير مسجل لأي مركبة حقيقية — يرجى التأكد من صحة جميع الأرقام والحروف'
          setVehicleSpecs(null)
          setVinCheckError(errMsg)
          setCachedVinSpecs(vinCode, { specs: null, error: errMsg })
        }
      }
    } catch {
      // Ignore network error
    } finally {
      setSpecsLoading(false)
    }
  }

  const handleVinChange = (e) => {
    let val = e.target.value.toUpperCase().replace(/[\s\-_]/g, '')
    val = val.replace(/[IOQ]/g, '')
    if (val.length <= 17) {
      setVin(val)
    }
  }

  const handleSearchSubmit = async (e) => {
    e.preventDefault()
    const cleanVin = vin.trim().toUpperCase()
    if (cleanVin.length !== 17) {
      toast.error('رقم الشاصي (VIN) يجب أن يتكون من 17 رمزاً')
      return
    }

    const localCheck = validateVinLocally(cleanVin)
    if (!localCheck.valid || vinCheckError) {
      toast.error(
        vinCheckError || localCheck.error || 'رقم الشاصي غير صحيح، لا يمكن إرسال طلب فحص لرقم خاطئ'
      )
      return
    }

    const submittedTitle = vehicleSpecs?.title || ''

    // Immediately clear input field & re-focus so user can paste the next VIN without waiting!
    setVin('')
    setVehicleSpecs(null)
    setVinCheckError('')
    setSpecsLoading(false)
    setLastQueuedVin(cleanVin)

    if (inputRef?.current) {
      inputRef.current.focus()
    }

    if (onQueueSearch) {
      onQueueSearch(cleanVin, submittedTitle)
    }
  }

  const localValid = vin.length === 17 && validateVinLocally(vin).valid && !vinCheckError
  const isReady = localValid

  return (
    <div className="bg-white dark:bg-card rounded-md shadow-md border border-slate-200/90 dark:border-border p-5 sm:p-6 flex flex-col justify-between h-full">
      <div>
        {/* Card Header */}
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-xl font-extrabold text-slate-900 dark:text-foreground tracking-tight">
            Run CARFAX Report
          </h3>
          <div className="flex items-center gap-2">
            {activeSearchCount > 0 && (
              <Badge className="bg-amber-500/15 text-amber-700 dark:text-amber-300 border border-amber-500/30 text-[11px] gap-1 font-bold" dir="rtl">
                <Loader2 className="h-3 w-3 animate-spin" />
                جاري سحب ({activeSearchCount}) في السجل
              </Badge>
            )}
            <span className="text-xs font-semibold text-slate-500">فحص الشاصي الفوري المتعدد</span>
          </div>
        </div>

        {/* CARFAX Tabs */}
        <div className="grid grid-cols-2 border-b border-slate-200 dark:border-border mb-5">
          <button
            type="button"
            onClick={() => setActiveTab('vin')}
            className={`pb-2.5 text-sm font-bold text-center transition-colors cursor-pointer ${
              activeTab === 'vin'
                ? 'border-b-[3px] border-[#00599c] text-[#00599c]'
                : 'text-slate-500 hover:text-slate-800'
            }`}
          >
            Search by VIN
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('plate')}
            className={`pb-2.5 text-sm font-semibold text-center transition-colors cursor-pointer ${
              activeTab === 'plate'
                ? 'border-b-[3px] border-[#00599c] text-[#00599c]'
                : 'text-slate-500 hover:text-slate-800'
            }`}
          >
            Search by License Plate
          </button>
        </div>

        {/* Input Section */}
        <form onSubmit={handleSearchSubmit} className="space-y-3">
          <div className="flex items-center justify-between flex-wrap gap-1.5">
            <label className="text-xs font-bold text-slate-800 dark:text-slate-200 uppercase tracking-wider">
              VIN (ألصق الشاصي واضغط Run Report لإرساله للسجل فوراً)
            </label>
            <div className="flex items-center gap-2">
              {vehicleSpecs?.title && (
                <Badge
                  variant="secondary"
                  className="text-[11px] font-mono bg-[#e8f2fa] text-[#00599c] border border-[#b9d8f2]"
                >
                  {vehicleSpecs.title}
                </Badge>
              )}
              <span
                className={`text-[11px] font-mono font-bold px-1.5 py-0.5 rounded ${
                  isReady
                    ? 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400'
                    : 'bg-slate-100 text-slate-500 dark:bg-muted dark:text-muted-foreground'
                }`}
              >
                {vin.length}/17
              </span>
            </div>
          </div>

          <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2.5" dir="ltr">
            <input
              ref={inputRef}
              type="text"
              placeholder="Enter VIN (e.g. 1GKS1DRLXSR219688)"
              value={vin}
              onChange={handleVinChange}
              required
              className="flex-1 h-11 px-3.5 rounded border border-slate-400 dark:border-input bg-white dark:bg-background text-sm font-mono tracking-wider uppercase text-slate-900 dark:text-foreground placeholder:text-slate-400 placeholder:tracking-normal placeholder:font-sans focus:outline-none focus:ring-2 focus:ring-[#00599c] focus:border-[#00599c]"
            />

            <button
              type="submit"
              disabled={!isReady}
              className={`h-11 px-6 rounded text-sm font-bold transition-all flex items-center justify-center gap-2 shrink-0 ${
                !isReady
                  ? 'bg-[#e0e0e0] text-[#757575] cursor-not-allowed'
                  : 'bg-[#00599c] hover:bg-[#004478] text-white shadow-sm cursor-pointer'
              }`}
            >
              <span>Run Report</span>
            </button>
          </div>
        </form>
      </div>

      {/* 1. Loading Instant Check */}
      {specsLoading && (
        <div
          className="mt-4 flex items-center gap-2 p-3 bg-sky-50/80 dark:bg-sky-950/30 rounded-lg border border-sky-200 text-xs text-[#003478] dark:text-sky-300"
          dir="rtl"
        >
          <Loader2 className="h-4 w-4 animate-spin shrink-0 text-[#00599c]" />
          <span className="font-semibold">
            الشاصي مطابق قياسياً — يمكنك الضغط على Run Report فوراً أو انتظار مواصفات المركبة...
          </span>
        </div>
      )}

      {/* 2. VIN Validation Error Warning */}
      {!specsLoading && vinCheckError && !vehicleSpecs && (
        <div
          className="mt-4 flex items-center gap-2 p-3 bg-rose-50 dark:bg-rose-950/30 rounded-lg border border-rose-200 text-xs text-rose-800 dark:text-rose-300"
          dir="rtl"
        >
          <AlertTriangle className="h-4 w-4 shrink-0 text-rose-600" />
          <span className="font-bold">{vinCheckError}</span>
        </div>
      )}

      {/* 3. Factory Specs Card */}
      {vehicleSpecs && !specsLoading && (
        <div
          className="mt-4 rounded-lg border border-[#b9d8f2] bg-[#f0f7fc]/90 dark:bg-sky-950/20 p-3.5 transition-all"
          dir="rtl"
        >
          <div className="flex flex-wrap items-center justify-between gap-2 pb-2.5 mb-2.5 border-b border-[#cfe3f5] dark:border-border">
            <div className="flex items-center gap-1.5">
              <Sparkles className="h-4 w-4 text-[#00599c] shrink-0" />
              <span className="text-xs font-extrabold text-[#003478] dark:text-sky-300">
                بيانات ومواصفات المركبة الأساسية:
              </span>
            </div>
            {vehicleSpecs.isCheckDigitValid ? (
              <Badge className="bg-[#1c8237] hover:bg-[#1c8237] text-white text-[10px] gap-1 border-none">
                <CheckCircle2 className="h-3 w-3" />
                شاصي صحيح ومطابق
              </Badge>
            ) : (
              <Badge className="bg-amber-600 hover:bg-amber-600 text-white text-[10px] gap-1 border-none">
                <Car className="h-3 w-3" />
                بيانات المركبة الأساسية
              </Badge>
            )}
          </div>

          <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5 text-xs">
            <div className="bg-white dark:bg-card p-2.5 rounded border border-slate-200/80 dark:border-border">
              <span className="text-slate-500 block text-[10px] font-semibold">الموديل والسنة:</span>
              <span className="font-extrabold text-slate-900 dark:text-foreground block mt-0.5" dir="ltr">
                {vehicleSpecs.year} {vehicleSpecs.make} {vehicleSpecs.model}
              </span>
            </div>
            <div className="bg-white dark:bg-card p-2.5 rounded border border-slate-200/80 dark:border-border">
              <span className="text-slate-500 block text-[10px] font-semibold">هيكل المركبة:</span>
              <span className="font-bold text-slate-800 dark:text-foreground block mt-0.5" dir="ltr">
                {vehicleSpecs.body}
              </span>
            </div>
            <div className="bg-white dark:bg-card p-2.5 rounded border border-slate-200/80 dark:border-border">
              <span className="text-slate-500 block text-[10px] font-semibold">المحرك والدفع:</span>
              <span className="font-bold text-slate-800 dark:text-foreground block mt-0.5" dir="ltr">
                {vehicleSpecs.engine}
                {vehicleSpecs.drive ? ` (${vehicleSpecs.drive})` : ''}
              </span>
            </div>
            <div className="bg-white dark:bg-card p-2.5 rounded border border-slate-200/80 dark:border-border">
              <span className="text-slate-500 block text-[10px] font-semibold">بلد التصنيع والوقود:</span>
              <span className="font-bold text-slate-800 dark:text-foreground block mt-0.5" dir="ltr">
                {vehicleSpecs.country} • {vehicleSpecs.fuel}
              </span>
            </div>
          </div>
        </div>
      )}

      {/* 4. Instant Queue Confirmation Strip */}
      {!vin && lastQueuedVin && (
        <div
          className="mt-3 rounded-md bg-emerald-50 dark:bg-emerald-950/30 border border-emerald-200 dark:border-emerald-800/50 text-emerald-900 dark:text-emerald-200 px-3.5 py-2.5 text-right flex items-center justify-between gap-2 text-xs"
          dir="rtl"
        >
          <div className="flex items-center gap-2">
            <Zap className="h-4 w-4 text-emerald-600 dark:text-emerald-400 shrink-0" />
            <span className="font-semibold">
              تمت إضافة الشاصي <strong className="font-mono" dir="ltr">{lastQueuedVin}</strong> إلى جدول التقارير أدناه — الحقل جاهز لإدخال شاصي جديد فوراً!
            </span>
          </div>
        </div>
      )}
    </div>
  )
}
