import React, { useState, useEffect } from 'react'
import { Badge } from '@/components/ui/badge'
import { Loader2, CheckCircle2, Clock, Sparkles, AlertTriangle, Car } from 'lucide-react'
import { toast } from 'sonner'
import {
  searchVin,
  getQueueStatus,
  api,
  getCachedVinSpecs,
  setCachedVinSpecs,
} from '@/lib/api'

export default function VinSearchBox({ onSuccessSearch, inputRef }) {
  const [vin, setVin] = useState('')
  const [activeTab, setActiveTab] = useState('vin')
  const [searching, setSearching] = useState(false)
  const [queueInfo, setQueueInfo] = useState(null)
  const [displaySeconds, setDisplaySeconds] = useState(0)
  const [specsLoading, setSpecsLoading] = useState(false)
  const [vehicleSpecs, setVehicleSpecs] = useState(null)
  const [vinCheckError, setVinCheckError] = useState('')

  // Instant pre-search VIN validation + Factory Specs lookup as soon as 17 chars are entered
  useEffect(() => {
    if (vin.length === 17) {
      verifyAndDecodeVin(vin)
    } else {
      setVehicleSpecs(null)
      setVinCheckError('')
      setSpecsLoading(false)
    }
  }, [vin])

  // Poll live status while searching
  useEffect(() => {
    if (!searching || vin.length !== 17) {
      setQueueInfo(null)
      setDisplaySeconds(0)
      return
    }

    let active = true
    const poll = async () => {
      try {
        const qRes = await getQueueStatus(vin)
        const info = qRes?.data || qRes
        if (active && info) {
          setQueueInfo(info)
          if (info.estimated_seconds > 0) {
            setDisplaySeconds((prev) =>
              prev === 0 || Math.abs(prev - info.estimated_seconds) > 4
                ? info.estimated_seconds
                : prev
            )
          }
        }
      } catch {
        // Ignore poll error
      }
    }

    poll()
    const pollTimer = setInterval(poll, 2000)
    const tickTimer = setInterval(() => {
      setDisplaySeconds((prev) => (prev > 2 ? prev - 1 : prev))
    }, 1000)

    return () => {
      active = false
      clearInterval(pollTimer)
      clearInterval(tickTimer)
    }
  }, [searching, vin])

  const verifyAndDecodeVin = async (vinCode) => {
    // 0. Check In-Memory VIN Specs Cache first (0ms instant response)
    const cached = getCachedVinSpecs(vinCode)
    if (cached) {
      setVehicleSpecs(cached.specs)
      setVinCheckError(cached.error || '')
      setSpecsLoading(false)
      return
    }

    setSpecsLoading(true)
    setVinCheckError('')
    let localErr = ''
    try {
      // 1. Local VIN syntax check
      try {
        const valRes = await api.post('/tools/vin-validate', { vin: vinCode })
        if (valRes && valRes.is_valid === false && valRes.error_msg) {
          localErr = valRes.error_msg
          setVinCheckError(localErr)
        }
      } catch {
        // Ignore local tool check failure
      }

      // 2. Vehicle Factory Specifications Lookup
      const res = await fetch(
        `https://vpic.nhtsa.dot.gov/api/vehicles/DecodeVinValues/${vinCode}?format=json`
      )
      const data = await res.json()
      if (data?.Results?.[0]) {
        const item = data.Results[0]
        const errCode = String(item.ErrorCode || '')
        const hasValidMakeOrYear = Boolean(item.Make || item.ModelYear)

        if (hasValidMakeOrYear) {
          const engineParts = [
            item.DisplacementL ? `${Number(item.DisplacementL).toFixed(1)}L` : '',
            item.EngineConfiguration || '',
            item.EngineCylinders ? `${item.EngineCylinders}-Cyl` : '',
          ]
            .filter(Boolean)
            .join(' ')

          const decoded = {
            year: item.ModelYear || '—',
            make: item.Make || '—',
            model: item.Model || '—',
            trim: item.Trim || item.Series || '',
            title: `${item.ModelYear || ''} ${item.Make || ''} ${item.Model || ''} ${item.Trim || ''}`.trim(),
            body: item.BodyClass || item.VehicleType || 'Passenger Vehicle',
            engine: engineParts || item.FuelTypePrimary || 'Standard Engine',
            drive: item.DriveType || 'Standard Drive',
            fuel: item.FuelTypePrimary || 'Gasoline',
            country: item.PlantCountry || item.PlantState || 'North America',
            manufacturer: item.Manufacturer || '',
            isCheckDigitValid: errCode.startsWith('0'),
          }
          setVehicleSpecs(decoded)
          setCachedVinSpecs(vinCode, { specs: decoded, error: '' })
        } else {
          const errMsg = localErr || 'يرجى التأكد من صحة كتابة رقم الشاصي'
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
    // Filter out forbidden VIN characters: I, O, Q
    val = val.replace(/[IOQ]/g, '')
    if (val.length <= 17) {
      setVin(val)
    }
  }

  const handleSearchSubmit = async (e) => {
    e.preventDefault()
    if (vin.length !== 17) {
      toast.error('رقم الشاصي (VIN) يجب أن يتكون من 17 رمزاً')
      return
    }

    setSearching(true)
    try {
      const res = await searchVin(vin)
      toast.success(res?.message || 'تم استخراج التقرير بنجاح')
      if (onSuccessSearch) {
        onSuccessSearch(res?.report || res, vin)
      }
    } catch (err) {
      const msg =
        err.response?.data?.message ||
        err.data?.message ||
        err.response?.data?.error ||
        err.message ||
        'تعذر استخراج التقرير حالياً، يرجى المحاولة مرة أخرى'
      toast.error(msg)
    } finally {
      setSearching(false)
      setQueueInfo(null)
    }
  }

  const isReady = vin.length === 17

  return (
    <div className="bg-white dark:bg-card rounded-md shadow-md border border-slate-200/90 dark:border-border p-5 sm:p-6 flex flex-col justify-between h-full">
      <div>
        {/* Card Header */}
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-xl font-extrabold text-slate-900 dark:text-foreground tracking-tight">
            Run CARFAX Report
          </h3>
          <span className="text-xs font-semibold text-slate-500">فحص الشاصي الفوري</span>
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
              VIN (رقم الشاصي)
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
              disabled={searching}
              required
              className="flex-1 h-11 px-3.5 rounded border border-slate-400 dark:border-input bg-white dark:bg-background text-sm font-mono tracking-wider uppercase text-slate-900 dark:text-foreground placeholder:text-slate-400 placeholder:tracking-normal placeholder:font-sans focus:outline-none focus:ring-2 focus:ring-[#00599c] focus:border-[#00599c]"
            />

            <button
              type="submit"
              disabled={searching || !isReady}
              className={`h-11 px-6 rounded text-sm font-bold transition-all flex items-center justify-center gap-2 shrink-0 ${
                searching || !isReady
                  ? 'bg-[#e0e0e0] text-[#757575] cursor-not-allowed'
                  : 'bg-[#00599c] hover:bg-[#004478] text-white shadow-sm cursor-pointer'
              }`}
            >
              {searching ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span>Running...</span>
                </>
              ) : (
                <span>Run Report</span>
              )}
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
            جاري التحقق من رقم الشاصي وجلب مواصفات المركبة...
          </span>
        </div>
      )}

      {/* 2. VIN Validation Error Warning */}
      {!specsLoading && vinCheckError && !vehicleSpecs && (
        <div
          className="mt-4 flex items-center gap-2 p-3 bg-amber-50 dark:bg-amber-950/30 rounded-lg border border-amber-200 text-xs text-amber-800 dark:text-amber-300"
          dir="rtl"
        >
          <AlertTriangle className="h-4 w-4 shrink-0 text-amber-600" />
          <span className="font-semibold">{vinCheckError}</span>
        </div>
      )}

      {/* 3. Factory Specs Card (Appears immediately before sending & stays visible while report is being prepared) */}
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
                {vehicleSpecs.engine} ({vehicleSpecs.drive})
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

      {/* 4. Clean Waiting Banner (No technical or internal terms) */}
      {searching && (
        <div className="mt-3 rounded-md bg-[#003478] text-white p-3 text-right shadow-xs" dir="rtl">
          <div className="flex items-center justify-between gap-2 flex-wrap">
            <div className="flex items-center gap-2">
              <Loader2 className="h-4 w-4 animate-spin text-sky-300 shrink-0" />
              <span className="text-xs font-bold">
                {`جاري استخراج وتجهيز التقرير للمركبة (${vehicleSpecs?.title || vin}) يرجى الانتظار لحظات...`}
              </span>
            </div>
            {displaySeconds > 0 && (
              <span className="inline-flex items-center gap-1 text-[11px] font-mono font-semibold text-sky-200 bg-white/10 px-2 py-0.5 rounded border border-white/20">
                <Clock className="h-3 w-3" />~{displaySeconds} ثانية
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
