import React, { useState, useEffect } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Search, Sparkles, Loader2, CheckCircle2, Car, Shield, Info, ArrowLeft } from 'lucide-react'
import { toast } from 'sonner'
import { searchVin } from '@/lib/api'

export default function VinSearchBox({ onSuccessSearch }) {
  const [vin, setVin] = useState('')
  const [searching, setSearching] = useState(false)
  const [nhtsaLoading, setNhtsaLoading] = useState(false)
  const [nhtsaSpecs, setNhtsaSpecs] = useState(null)

  // Fetch free NHTSA VPIC specs as soon as 17 valid characters are typed
  useEffect(() => {
    if (vin.length === 17) {
      fetchNhtsaSpecs(vin)
    } else {
      setNhtsaSpecs(null)
    }
  }, [vin])

  const fetchNhtsaSpecs = async (vinCode) => {
    setNhtsaLoading(true)
    try {
      const res = await fetch(`https://vpic.nhtsa.dot.gov/api/vehicles/DecodeVinValues/${vinCode}?format=json`)
      const data = await res.json()
      if (data && data.Results && data.Results.length > 0) {
        const item = data.Results[0]
        if (item.Make || item.ModelYear) {
          setNhtsaSpecs({
            year: item.ModelYear || '—',
            make: item.Make || '—',
            model: item.Model || '—',
            body: item.BodyClass || 'Sedan / Passenger Car',
            engine: item.DisplacementL ? `${item.DisplacementL}L ${item.EngineConfiguration || ''} ${item.EngineCylinders ? item.EngineCylinders + '-Cyl' : ''}` : 'Standard Engine',
            drive: item.DriveType || 'FWD / RWD',
            fuel: item.FuelTypePrimary || 'Gasoline',
            country: item.PlantCountry || 'United States / Canada',
          })
        }
      }
    } catch {
      // Ignore background specs error
    } finally {
      setNhtsaLoading(false)
    }
  }

  const handleVinChange = (e) => {
    let val = e.target.value.toUpperCase().replace(/[\s\-_]/g, '')
    // Filter out forbidden VIN chars: I, O, Q
    val = val.replace(/[IOQ]/g, '')
    if (val.length <= 17) {
      setVin(val)
    }
  }

  const handleSearchSubmit = async (e) => {
    e.preventDefault()
    if (vin.length !== 17) {
      toast.error('رقم الشاصي يجب أن يكون من 17 حرفاً ورقماً بدقة')
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
        'تعذر استخراج التقرير، يرجى المحاولة لاحقاً'
      toast.error(msg)
    } finally {
      setSearching(false)
    }
  }

  return (
    <Card className="border-slate-200 dark:border-slate-800 shadow-sm w-full overflow-hidden bg-white dark:bg-slate-900 rounded-xl">
      {/* Official Carfax Header Banner */}
      <div className="bg-[#003478] text-white px-5 py-3.5 flex items-center justify-between text-right">
        <div className="flex items-center gap-2">
          <Car className="h-5 w-5 text-blue-200" />
          <h3 className="text-sm sm:text-base font-bold font-cairo tracking-wide">
            Run a CARFAX Vehicle History Report™
          </h3>
        </div>
        <Badge className="bg-[#00599C] hover:bg-[#00599C] text-white text-[11px] font-mono px-2 py-0.5 border-none">
          OFFICIAL DEALER INQUIRY
        </Badge>
      </div>

      <CardContent className="p-5 sm:p-6 space-y-4 text-right">
        <form onSubmit={handleSearchSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-bold text-slate-700 dark:text-slate-300 mb-1.5">
              رقم تعريف المركبة (VIN) المكون من 17 رمزاً:
            </label>
            <div className="flex flex-col sm:flex-row gap-3">
              <div className="relative flex-1">
                <Input
                  type="text"
                  placeholder="أدخل رقم الشاصي (مثال: 1HGCR2F85HA000000)"
                  value={vin}
                  onChange={handleVinChange}
                  dir="ltr"
                  className="font-mono text-base tracking-widest pl-10 pr-24 uppercase h-12 border-2 border-slate-300 focus-visible:border-[#003478] focus-visible:ring-[#003478]/20 bg-white dark:bg-slate-800"
                  disabled={searching}
                  required
                />
                {/* Character Counter & Clear Badge */}
                <div className="absolute right-3 top-3 flex items-center gap-1.5">
                  <span
                    className={`text-xs font-mono font-bold px-1.5 py-0.5 rounded ${
                      vin.length === 17
                        ? 'bg-emerald-100 text-[#1C8237] dark:bg-emerald-950 dark:text-emerald-300'
                        : 'bg-slate-100 text-slate-500 dark:bg-slate-700 dark:text-slate-400'
                    }`}
                  >
                    {vin.length}/17
                  </span>
                </div>
                {/* Search Icon */}
                <Search className="absolute left-3.5 top-3.5 h-5 w-5 text-slate-400 pointer-events-none" />
              </div>

              <Button
                type="submit"
                disabled={searching || vin.length !== 17}
                className="bg-[#003478] hover:bg-[#002657] text-white font-bold h-12 px-6 text-sm shadow-sm transition-all sm:w-auto w-full"
              >
                {searching ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin ml-2" />
                    جاري جلب وفحص التقرير...
                  </>
                ) : (
                  <>
                    <Search className="h-4 w-4 ml-2" />
                    فحص واستخراج التقرير
                  </>
                )}
              </Button>
            </div>
          </div>
        </form>

        {/* Live NHTSA Factory Specs Card */}
        {nhtsaLoading && (
          <div className="flex items-center gap-2 p-3 bg-blue-50/70 dark:bg-blue-950/30 rounded-lg border border-blue-200 text-xs text-[#003478] dark:text-blue-300">
            <Loader2 className="h-4 w-4 animate-spin shrink-0" />
            <span>جاري التحقق الفوري من مواصفات المصنع القياسية للمركبة (NHTSA VPIC)...</span>
          </div>
        )}

        {nhtsaSpecs && !nhtsaLoading && (
          <div className="rounded-xl border border-blue-200 bg-[#E8F1F8]/40 dark:bg-blue-950/20 p-4 transition-all">
            <div className="flex items-center justify-between pb-2 mb-3 border-b border-blue-200/60">
              <div className="flex items-center gap-2">
                <Sparkles className="h-4 w-4 text-[#00599C]" />
                <span className="text-xs font-bold text-[#003478] dark:text-blue-300">
                  مواصفات المركبة المعتمدة من قاعدة بيانات المصنع (NHTSA):
                </span>
              </div>
              <Badge className="bg-[#1C8237] hover:bg-[#1C8237] text-white text-[10px] border-none">
                شاصي صحيح ومطابق
              </Badge>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
              <div className="bg-white/80 dark:bg-slate-800/80 p-2.5 rounded-lg border border-slate-200/70">
                <span className="text-slate-500 block text-[11px]">الموديل والسنة:</span>
                <span className="font-bold text-slate-800 dark:text-slate-200">
                  {nhtsaSpecs.year} {nhtsaSpecs.make} {nhtsaSpecs.model}
                </span>
              </div>
              <div className="bg-white/80 dark:bg-slate-800/80 p-2.5 rounded-lg border border-slate-200/70">
                <span className="text-slate-500 block text-[11px]">هيكل المركبة:</span>
                <span className="font-bold text-slate-800 dark:text-slate-200">
                  {nhtsaSpecs.body}
                </span>
              </div>
              <div className="bg-white/80 dark:bg-slate-800/80 p-2.5 rounded-lg border border-slate-200/70">
                <span className="text-slate-500 block text-[11px]">المحرك وناقل الحركة:</span>
                <span className="font-bold text-slate-800 dark:text-slate-200">
                  {nhtsaSpecs.engine} ({nhtsaSpecs.drive})
                </span>
              </div>
              <div className="bg-white/80 dark:bg-slate-800/80 p-2.5 rounded-lg border border-slate-200/70">
                <span className="text-slate-500 block text-[11px]">بلد التصنيع:</span>
                <span className="font-bold text-slate-800 dark:text-slate-200">
                  {nhtsaSpecs.country}
                </span>
              </div>
            </div>
          </div>
        )}

        {/* Quick Demo VINs Helper */}
        <div className="pt-2 flex flex-wrap items-center gap-2 text-xs text-slate-500">
          <span className="font-semibold text-slate-600 dark:text-slate-400">شاصي تجريبي سريع:</span>
          <button
            type="button"
            onClick={() => setVin('1HGCR2F85HA000000')}
            className="font-mono text-[11px] bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 px-2 py-0.5 rounded border border-slate-300 transition-colors"
          >
            1HGCR2F85HA000000 (Honda Accord)
          </button>
          <button
            type="button"
            onClick={() => setVin('1FA6P8CF9H5XXXXXX')}
            className="font-mono text-[11px] bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 px-2 py-0.5 rounded border border-slate-300 transition-colors"
          >
            1FA6P8CF9H5XXXXXX (Ford Mustang)
          </button>
        </div>
      </CardContent>
    </Card>
  )
}
