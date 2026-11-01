import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/context/AuthContext'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { AlertCircle, Loader2, Car } from 'lucide-react'
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
      setErrorMsg('يرجى إدخال اسم المستخدم وكلمة المرور')
      return
    }

    setLoading(true)
    setErrorMsg('')
    try {
      await login(username.trim(), password)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      const msg =
        err.response?.data?.message ||
        err.response?.data?.error ||
        err.message ||
        'بيانات الدخول غير صحيحة'
      setErrorMsg(msg)
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-muted/30 px-4 py-8" dir="rtl">
      <Card className="w-full max-w-sm shadow-sm border-border bg-card">
        <CardHeader className="text-center space-y-2 pb-4">
          <div className="mx-auto flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Car className="h-5 w-5" />
          </div>
          <CardTitle className="text-lg font-bold">تسجيل الدخول</CardTitle>
          <CardDescription className="text-xs text-muted-foreground">
            أدخل بيانات حسابك للمتابعة إلى نظام كارفاكس
          </CardDescription>
        </CardHeader>

        <CardContent>
          {errorMsg && (
            <div className="mb-4 flex items-start gap-2 rounded-md bg-destructive/10 p-3 text-xs text-destructive border border-destructive/20 text-right">
              <AlertCircle className="h-4 w-4 shrink-0 mt-0.5" />
              <span>{errorMsg}</span>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-1.5 text-right">
              <Label htmlFor="username" className="text-xs font-semibold">
                اسم المستخدم
              </Label>
              <Input
                id="username"
                type="text"
                placeholder="اسم المستخدم"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                dir="ltr"
                autoComplete="username"
                className="font-mono"
                required
              />
            </div>

            <div className="space-y-1.5 text-right">
              <Label htmlFor="password" className="text-xs font-semibold">
                كلمة المرور
              </Label>
              <Input
                id="password"
                type="password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                dir="ltr"
                autoComplete="current-password"
                required
              />
            </div>

            <Button type="submit" className="w-full text-xs font-bold h-9 cursor-pointer" disabled={loading}>
              {loading ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin ml-2" />
                  جاري الدخول...
                </>
              ) : (
                'دخول'
              )}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Toaster position="top-center" dir="rtl" />
    </div>
  )
}
