import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Search, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { searchVin } from '@/lib/api'

export default function VinSearchBox({ onSuccessSearch }) {
  const [vin, setVin] = useState('')
  const [searching, setSearching] = useState(false)
  const [nhtsaSpecs, setNhtsaSpecs] = useState(null)

  useEffect(() => {
    if (vin.length === 17) {
      fetchNhtsaSpecs(vin)
    } else {
      setNhtsaSpecs(null)
    }
  }, [vin])

  const fetchNhtsaSpecs = async (vinCode) => {
    try {
      const res = await fetch(
        `https://vpic.nhtsa.dot.gov/api/vehicles/DecodeVinValues/${vinCode}?format=json`
      )
      const data = await res.json()
      if (data?.Results?.[0]) {
        const item = data.Results[0]
        if (item.Make || item.ModelYear) {
          setNhtsaSpecs({
            title: `${item.ModelYear || ''} ${item.Make || ''} ${item.Model || ''}`.trim(),
            engine: item.DisplacementL ? `${item.DisplacementL}L` : '',
            body: item.BodyClass || '',
          })
        }
      }
    } catch {
      // Ignore background NHTSA lookup error
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
    if (vin.length !== 17) {
      toast.error('رقم الشاصي يجب أن يتكون من 17 رمزاً')
      return
    }

    setSearching(true)
    try {
      const res = await searchVin(vin)
      toast.success(res?.message || 'تم جلب التقرير بنجاح')
      if (onSuccessSearch) {
        onSuccessSearch(res?.report || res, vin)
      }
    } catch (err) {
      const msg =
        err.response?.data?.message ||
        err.data?.message ||
        err.response?.data?.error ||
        err.message ||
        'تعذر استخراج التقرير'
      toast.error(msg)
    } finally {
      setSearching(false)
    }
  }

  return (
    <Card className="border-border shadow-xs bg-card">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-base font-bold text-foreground">
              فحص رقم الشاصي (VIN)
            </CardTitle>
            <CardDescription className="text-xs text-muted-foreground mt-0.5">
              أدخل رقم الشاصي المكون من 17 رمزاً لسحب تقرير كارفاكس
            </CardDescription>
          </div>
          {nhtsaSpecs?.title && (
            <Badge variant="secondary" className="text-xs font-mono">
              {nhtsaSpecs.title} {nhtsaSpecs.engine && `• ${nhtsaSpecs.engine}`}
            </Badge>
          )}
        </div>
      </CardHeader>

      <CardContent>
        <form onSubmit={handleSearchSubmit} className="flex flex-col sm:flex-row gap-2.5">
          <div className="relative flex-1">
            <Input
              type="text"
              placeholder="مثال: 1GKS1DRLXSR219688"
              value={vin}
              onChange={handleVinChange}
              dir="ltr"
              className="font-mono text-sm tracking-widest pl-4 pr-16 uppercase h-10"
              disabled={searching}
              required
            />
            <div className="absolute right-2.5 top-2.5">
              <span
                className={`text-[11px] font-mono font-semibold px-1.5 py-0.5 rounded ${
                  vin.length === 17
                    ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
                    : 'bg-muted text-muted-foreground'
                }`}
              >
                {vin.length}/17
              </span>
            </div>
          </div>

          <Button
            type="submit"
            disabled={searching || vin.length !== 17}
            className="h-10 px-5 text-xs font-bold gap-2 cursor-pointer"
          >
            {searching ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                <span>جاري الفحص...</span>
              </>
            ) : (
              <>
                <Search className="h-4 w-4" />
                <span>فحص الشاصي</span>
              </>
            )}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
