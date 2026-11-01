import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/context/AuthContext'
import CarfaxBadgeLogo from '@/components/common/CarfaxBadgeLogo'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Lock, User, AlertCircle, Loader2, ShieldCheck, Car } from 'lucide-react'
import { toast, Toaster } from 'sonner'

export default function LoginView() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [errorMsg, setErrorMsg] = useState('')

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!username.trim() || !password.trim()) {
      setErrorMsg('يرجى ملء جميع الحقول المطلوبة')
      return
    }

    setLoading(true)
    setErrorMsg('')
    try {
      await login(username.trim(), password)
      toast.success('تم الدخول إلى غرفة عمليات كندا بنجاح')
      navigate('/dashboard', { replace: true })
    } catch (err) {
      const msg = err.response?.data?.message || err.response?.data?.error || err.message || 'بيانات الدخول غير صحيحة، يرجى المحاولة ثانية'
      setErrorMsg(msg)
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  const fillDemoAdmin = () => {
    setUsername('admin')
    setPassword('admin123456')
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-slate-100/70 px-4 py-8 font-cairo" dir="rtl">
      {/* Brand Header */}
      <div className="mb-6 flex flex-col items-center gap-2 text-center">
        <CarfaxBadgeLogo size="lg" subtitle="DEALER PORTAL | بوابة تقارير المركبات" />
        <p className="text-xs text-slate-500 max-w-sm mt-1">
          بوابة معارض السيارات المعتمدة لاستخراج تقارير CARFAX Vehicle History Report™ الفورية
        </p>
      </div>

      {/* Login Card */}
      <Card className="w-full max-w-md shadow-lg border-slate-200 bg-white rounded-xl overflow-hidden">
        <div className="bg-[#003478] text-white px-6 py-4 text-center">
          <div className="inline-flex items-center justify-center w-9 h-9 rounded-full bg-white/10 mb-2">
            <Car className="h-5 w-5 text-white" />
          </div>
          <h2 className="text-base font-bold tracking-wide">تسجيل دخول التجار والوكلاء</h2>
          <p className="text-xs text-blue-200 mt-0.5">CARFAXOnline Authorized Dealer Login</p>
        </div>

        <CardContent className="p-6">
          {errorMsg && (
            <div className="mb-4 flex items-start gap-2 rounded-md bg-[#FDF2F2] p-3 text-xs text-[#D0202E] border border-[#F8B4B4] text-right">
              <AlertCircle className="h-4 w-4 shrink-0 mt-0.5" />
              <span>{errorMsg}</span>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            {/* Username Input */}
            <div className="space-y-1.5 text-right">
              <Label htmlFor="username" className="text-xs font-bold text-slate-700">
                اسم المستخدم / Dealer Username
              </Label>
              <div className="relative">
                <Input
                  id="username"
                  type="text"
                  placeholder="admin"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  dir="ltr"
                  autoComplete="username"
                  className="pr-9 text-right border-slate-300 focus-visible:border-[#003478] focus-visible:ring-[#003478]/20"
                  required
                />
                <User className="absolute right-3 top-2.5 h-4 w-4 text-muted-foreground pointer-events-none" />
              </div>
            </div>

            {/* Password Input */}
            <div className="space-y-1.5 text-right">
              <Label htmlFor="password" className="text-xs font-bold text-slate-700">
                كلمة المرور / Password
              </Label>
              <div className="relative">
                <Input
                  id="password"
                  type="password"
                  placeholder="••••••••"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  dir="ltr"
                  autoComplete="current-password"
                  className="pr-9 text-right border-slate-300 focus-visible:border-[#003478] focus-visible:ring-[#003478]/20"
                  required
                />
                <Lock className="absolute right-3 top-2.5 h-4 w-4 text-muted-foreground pointer-events-none" />
              </div>
            </div>

            {/* Submit Button */}
            <Button
              type="submit"
              className="w-full bg-[#003478] hover:bg-[#002657] text-white text-xs font-bold h-10 mt-2 shadow-sm transition-all"
              disabled={loading}
            >
              {loading ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin ml-2" />
                  جاري التحقق والدخول...
                </>
              ) : (
                'تسجيل الدخول إلى البوابة'
              )}
            </Button>
          </form>

          {/* Quick Demo Fill */}
          <div className="mt-5 border-t border-slate-200 pt-3 text-center">
            <button
              type="button"
              onClick={fillDemoAdmin}
              className="text-[11px] text-[#00599C] hover:text-[#003478] transition-colors font-semibold"
            >
              تجربة سريعة بحساب الأدمن (اضغط للتعبئة التلقائية)
            </button>
          </div>
        </CardContent>
      </Card>

      <div className="mt-6 flex items-center gap-2 text-[11px] text-slate-500">
        <ShieldCheck className="h-4 w-4 text-[#1C8237]" />
        <span>بوابة مشفرة ومحمية بنظام أمان CARFAX Dealer Protocol</span>
      </div>

      <Toaster position="top-center" dir="rtl" />
    </div>
  )
}
