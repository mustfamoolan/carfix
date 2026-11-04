import React, { useState } from 'react'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { api } from '@/lib/api'
import {
  Gavel,
  Search,
  Loader2,
  ExternalLink,
  Car,
  Image as ImageIcon,
  DollarSign,
  MapPin,
  Calendar,
  AlertCircle,
  CheckCircle2,
} from 'lucide-react'
import { toast } from 'sonner'

export default function AdminCopartView() {
  const [searchQuery, setSearchQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const [lotData, setLotData] = useState(null)

  // Bidding test state
  const [bidAmount, setBidAmount] = useState('')
  const [bidding, setBidding] = useState(false)
  const [recentBids, setRecentBids] = useState([])

  const fetchLot = async (e) => {
    if (e) e.preventDefault()
    const q = searchQuery.trim()
    if (!q) {
      toast.error('يرجى إدخال رقم اللوت أو رقم الشاصي')
      return
    }

    setLoading(true)
    setLotData(null)
    try {
      const res = await api.get(`/admin/copart/lot/${encodeURIComponent(q)}`)
      const data = res?.lot || res?.data?.lot || res
      setLotData(data)
      if (data?.current_bid) {
        setBidAmount(String(data.current_bid + 100))
      }
      toast.success('تم جلب تفاصيل اللوت بنجاح')
      loadBids(data?.lot_number)
    } catch (err) {
      const msg = err.response?.data?.message || err.message || 'تعذر جلب بيانات اللوت من كوبارت'
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  const loadBids = async (lotNum) => {
    try {
      const res = await api.get(`/admin/copart/bids${lotNum ? `?lot_number=${lotNum}` : ''}`)
      setRecentBids(res?.bids || res?.data?.bids || [])
    } catch {
      // Ignore background load error
    }
  }

  const handlePlaceBid = async (e) => {
    e.preventDefault()
    const amt = parseFloat(bidAmount)
    if (!amt || amt <= 0) {
      toast.error('يرجى تحديد مبلغ مزايدة صحيح')
      return
    }

    setBidding(true)
    try {
      const res = await api.post('/admin/copart/bid', {
        lot_number: lotData.lot_number,
        vin: lotData.vin,
        vehicle_title: lotData.vehicle_title,
        max_bid_amount: amt,
        notes: 'مزايدة تجريبية من لوحة الإدارة',
      })
      toast.success('تم تسجيل وتثبيت المزايدة بنجاح تحت حسابك الموحد!')
      loadBids(lotData.lot_number)
    } catch (err) {
      const msg = err.response?.data?.message || err.message || 'فشل إرسال المزايدة'
      toast.error(msg)
    } finally {
      setBidding(false)
    }
  }

  return (
    <AuthenticatedLayout
      title="بوابة مزادات Copart (تجريبي خاص بالإدارة)"
      subtitle="نظام فحص سيارات كوبارت وإدارة المزايدة الموحدة للزبائن"
    >
      <div className="space-y-6 text-right w-full" dir="rtl">
        {/* Search Header Bar */}
        <Card className="border-border shadow-xs bg-card">
          <CardHeader className="p-4 sm:p-5 border-b border-border">
            <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
              <div>
                <CardTitle className="text-base font-bold flex items-center gap-2">
                  <Gavel className="h-5 w-5 text-primary" />
                  <span>بوابة مزادات Copart (إدارة مركزية)</span>
                  <Badge variant="outline" className="text-[10px] text-amber-600 border-amber-500/30 bg-amber-500/10">
                    ميزة خاصة بالأدمن
                  </Badge>
                </CardTitle>
                <p className="text-xs text-muted-foreground mt-1">
                  البحث برقم اللوت (Lot Number) أو رقم الشاصي (VIN) لاستخراج الصور الدقيقة والمواصفات واختبار المزايدة
                </p>
              </div>

              <form onSubmit={fetchLot} className="flex items-center gap-2 w-full sm:w-auto">
                <div className="relative flex-1 sm:w-72">
                  <Search className="h-3.5 w-3.5 text-muted-foreground absolute right-2.5 top-3" />
                  <Input
                    placeholder="رقم اللوت (مثل: 58941234) أو VIN..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    dir="ltr"
                    className="h-9 pr-8 text-xs font-mono"
                  />
                </div>
                <Button type="submit" size="sm" disabled={loading} className="h-9 text-xs font-bold gap-1.5 cursor-pointer">
                  {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Search className="h-3.5 w-3.5" />}
                  <span>فحص اللوت</span>
                </Button>
              </form>
            </div>
          </CardHeader>
        </Card>

        {/* Lot Details & Image Gallery */}
        {lotData && (
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
            {/* Right/Top details card */}
            <div className="lg:col-span-7 space-y-4">
              <Card className="border-border shadow-xs bg-card overflow-hidden">
                <CardHeader className="p-4 bg-muted/40 border-b border-border flex flex-row items-center justify-between">
                  <div>
                    <h3 className="text-base font-bold text-foreground">{lotData.vehicle_title}</h3>
                    <p className="text-xs font-mono text-muted-foreground mt-0.5" dir="ltr">
                      Lot #: <strong className="text-foreground">{lotData.lot_number}</strong> | VIN: {lotData.vin || '—'}
                    </p>
                  </div>
                  <Badge variant="outline" className="text-xs font-bold border-emerald-500/30 bg-emerald-500/10 text-emerald-600">
                    {lotData.item_status || 'Active'}
                  </Badge>
                </CardHeader>

                <CardContent className="p-4 space-y-4">
                  {/* Key Stats Grid */}
                  <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 text-xs">
                    <div className="p-2.5 rounded bg-muted/30 border border-border">
                      <span className="text-muted-foreground block text-[11px]">الضرر الأساسي:</span>
                      <strong className="text-foreground mt-0.5 block">{lotData.damage_primary || '—'}</strong>
                    </div>

                    <div className="p-2.5 rounded bg-muted/30 border border-border">
                      <span className="text-muted-foreground block text-[11px]">الضرر الثانوي:</span>
                      <strong className="text-foreground mt-0.5 block">{lotData.damage_secondary || '—'}</strong>
                    </div>

                    <div className="p-2.5 rounded bg-muted/30 border border-border">
                      <span className="text-muted-foreground block text-[11px]">العداد المسجل:</span>
                      <strong className="text-foreground mt-0.5 block font-mono">{lotData.odometer || '—'}</strong>
                    </div>

                    <div className="p-2.5 rounded bg-muted/30 border border-border">
                      <span className="text-muted-foreground block text-[11px]">موقع المزاد (Yard):</span>
                      <strong className="text-foreground mt-0.5 block flex items-center gap-1">
                        <MapPin className="h-3 w-3 text-primary shrink-0" />
                        <span className="truncate">{lotData.sale_location || '—'}</span>
                      </strong>
                    </div>

                    <div className="p-2.5 rounded bg-muted/30 border border-border">
                      <span className="text-muted-foreground block text-[11px]">المزايدة الحالية:</span>
                      <strong className="text-emerald-600 mt-0.5 block font-mono text-sm">
                        ${lotData.current_bid || 0} {lotData.currency}
                      </strong>
                    </div>

                    <div className="p-2.5 rounded bg-muted/30 border border-border">
                      <span className="text-muted-foreground block text-[11px]">الشراء الفوري:</span>
                      <strong className="text-foreground mt-0.5 block font-mono">
                        {lotData.buy_now_price > 0 ? `$${lotData.buy_now_price}` : 'غير متاح'}
                      </strong>
                    </div>
                  </div>

                  {/* Images Gallery */}
                  <div>
                    <h4 className="text-xs font-bold text-foreground mb-2.5 flex items-center gap-1.5">
                      <ImageIcon className="h-4 w-4 text-primary" />
                      <span>صور المركبة في المزاد ({lotData.images?.length || 0} صورة عالية الدقة)</span>
                    </h4>

                    {lotData.images && lotData.images.length > 0 ? (
                      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                        {lotData.images.map((img, i) => (
                          <a
                            key={i}
                            href={img}
                            target="_blank"
                            rel="noreferrer"
                            className="group block relative aspect-4/3 rounded overflow-hidden border border-border bg-black/5 hover:border-primary transition-colors"
                          >
                            <img src={img} alt={`Lot ${i + 1}`} className="w-full h-full object-cover group-hover:scale-105 transition-transform" />
                            <span className="absolute bottom-1 right-1 bg-black/60 text-white text-[9px] px-1 rounded font-mono">
                              #{i + 1}
                            </span>
                          </a>
                        ))}
                      </div>
                    ) : (
                      <div className="p-6 text-center text-xs text-muted-foreground bg-muted/20 rounded border border-dashed border-border">
                        لا توجد صور متوفرة في استجابة المزاد حالياً لهذا اللوت.
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Left side: Place Proxy Bid Card */}
            <div className="lg:col-span-5 space-y-4">
              <Card className="border-border shadow-xs bg-card">
                <CardHeader className="p-4 border-b border-border bg-primary/5">
                  <CardTitle className="text-sm font-bold flex items-center gap-2">
                    <DollarSign className="h-4 w-4 text-primary" />
                    <span>تقديم مزايدة مبكرة (Pre-Bid Gateway)</span>
                  </CardTitle>
                  <p className="text-xs text-muted-foreground">
                    تتم المزايدة تحت حساب الشركة الموحد، ويتم تقييد اسم المستخدم تلقائياً
                  </p>
                </CardHeader>

                <CardContent className="p-4">
                  <form onSubmit={handlePlaceBid} className="space-y-3.5">
                    <div>
                      <label className="text-xs font-semibold block mb-1">الحد الأقصى للمزايدة (USD $)</label>
                      <Input
                        type="number"
                        placeholder="مثال: 5500"
                        value={bidAmount}
                        onChange={(e) => setBidAmount(e.target.value)}
                        className="font-mono text-sm h-9"
                        dir="ltr"
                        required
                      />
                      <span className="text-[11px] text-muted-foreground mt-1 block">
                        سيقوم النظام بالمزايدة التلقائية حتى هذا السقف كحد أقصى
                      </span>
                    </div>

                    <Button type="submit" disabled={bidding} className="w-full h-9 text-xs font-bold gap-1.5 cursor-pointer">
                      {bidding ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Gavel className="h-3.5 w-3.5" />}
                      <span>تثبيت المزايدة على اللوت</span>
                    </Button>
                  </form>

                  {/* Internal Ledger Info */}
                  <div className="mt-4 pt-4 border-t border-border">
                    <h5 className="text-xs font-bold mb-2">سجل المزايدات لهذا اللوت:</h5>
                    {recentBids.length === 0 ? (
                      <p className="text-[11px] text-muted-foreground">لا توجد مزايدات مسجلة من نظامنا على هذا اللوت حتى الآن.</p>
                    ) : (
                      <div className="space-y-2">
                        {recentBids.map((b) => (
                          <div key={b.id} className="p-2 rounded bg-muted/40 border border-border text-xs flex items-center justify-between">
                            <div>
                              <span className="font-semibold block">@{b.user?.username || 'admin'}</span>
                              <span className="text-[10px] text-muted-foreground font-mono">{new Date(b.created_at).toLocaleTimeString('ar-EG')}</span>
                            </div>
                            <div className="text-left font-mono">
                              <span className="font-bold text-emerald-600 block">${b.max_bid_amount}</span>
                              <Badge variant="outline" className="text-[9px] px-1 py-0 border-emerald-500/30 text-emerald-700 bg-emerald-500/10">
                                {b.status}
                              </Badge>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        )}
      </div>
    </AuthenticatedLayout>
  )
}
