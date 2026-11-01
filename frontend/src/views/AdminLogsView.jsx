import React, { useState, useEffect } from 'react'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
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
import { Activity, Loader2, ExternalLink } from 'lucide-react'
import { getAdminSearchLogs } from '@/lib/api'
import { useNavigate } from 'react-router-dom'

export default function AdminLogsView() {
  const [logs, setLogs] = useState([])
  const [loading, setLoading] = useState(true)
  const navigate = useNavigate()

  const fetchLogs = async () => {
    setLoading(true)
    try {
      const res = await getAdminSearchLogs()
      setLogs(res?.logs || res?.data?.logs || [])
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchLogs()
  }, [])

  return (
    <AuthenticatedLayout
      title="سجل العمليات الحية"
      subtitle="مراقبة حية لجميع طلبات فحص الشاصي عبر النظام"
    >
      <div className="space-y-6 text-right w-full" dir="rtl">
        <Card className="border-slate-200 dark:border-slate-800 shadow-sm w-full bg-white dark:bg-slate-900 rounded-xl overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between pb-3 text-right bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-800">
            <div>
              <CardTitle className="text-base font-bold flex items-center gap-2 text-[#003478] dark:text-blue-300">
                <Activity className="h-4.5 w-4.5 text-[#003478] dark:text-blue-300" />
                سجل الفحوصات والعمليات الشاملة في المنصة
              </CardTitle>
              <CardDescription className="text-xs text-slate-500">
                متابعة مباشرة لجميع طلبات فحص الشاصي عبر كافة التجار والمستخدمين مع مصدر الجلب وزمن الاستجابة
              </CardDescription>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={fetchLogs}
              className="text-xs h-8 hover:bg-slate-100 dark:hover:bg-slate-800"
            >
              تحديث
            </Button>
          </CardHeader>
          <CardContent className="p-0">
            {loading ? (
              <div className="flex flex-col items-center justify-center py-12 text-slate-400 gap-2">
                <Loader2 className="h-6 w-6 animate-spin text-[#003478]" />
                <span className="text-xs">جاري جلب السجلات...</span>
              </div>
            ) : logs.length === 0 ? (
              <div className="text-center py-12 text-slate-400 text-xs">
                لا توجد عمليات مسجلة بعد في النظام
              </div>
            ) : (
              <div className="overflow-x-auto w-full">
                <Table className="w-full text-right">
                  <TableHeader className="bg-[#E8F1F8] dark:bg-slate-800">
                    <TableRow className="border-b border-slate-200 dark:border-slate-700">
                      <TableHead className="w-[60px] text-right text-xs font-bold text-[#003478] dark:text-blue-200">#</TableHead>
                      <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">المستخدم</TableHead>
                      <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">رقم الشاصي (VIN)</TableHead>
                      <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">المركبة</TableHead>
                      <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">المصدر</TableHead>
                      <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">الحالة</TableHead>
                      <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">المدة</TableHead>
                      <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">التاريخ والوقت</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {logs.map((log) => (
                      <TableRow key={log.id} className="hover:bg-blue-50/40 dark:hover:bg-slate-800/40 border-b border-slate-100 dark:border-slate-800 transition-colors">
                        <TableCell className="font-mono text-xs text-slate-400">
                          {log.id}
                        </TableCell>
                        <TableCell className="text-xs font-bold text-slate-800 dark:text-slate-100">
                          {log.user?.username ? `@${log.user.username}` : `UID: ${log.user_id}`}
                        </TableCell>
                        <TableCell className="font-mono font-bold text-xs" dir="ltr">
                          <button
                            onClick={() => navigate(`/report/${log.vin}`)}
                            className="hover:underline text-[#00599C] dark:text-blue-400 flex items-center gap-1"
                          >
                            <span>{log.vin}</span>
                            <ExternalLink className="h-3 w-3 inline" />
                          </button>
                        </TableCell>
                        <TableCell className="text-xs text-slate-700 dark:text-slate-300">
                          {log.vehicle_title || '—'}
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline" className="text-[10px] border-slate-300">
                            {log.fetch_source || 'محلي'}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Badge
                            className={`text-[10px] font-bold border-none ${
                              log.result_status === 'success'
                                ? 'bg-emerald-100 text-[#1C8237]'
                                : 'bg-rose-100 text-[#D0202E]'
                            }`}
                          >
                            {log.result_status}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-xs font-mono text-slate-500">
                          {log.duration_ms} ms
                        </TableCell>
                        <TableCell className="text-xs text-slate-500">
                          {new Date(log.created_at).toLocaleString('ar-EG')}
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
