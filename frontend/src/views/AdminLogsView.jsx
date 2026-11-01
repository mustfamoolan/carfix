import React, { useState, useEffect } from 'react'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
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
import { Loader2, ExternalLink, RefreshCw } from 'lucide-react'
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
    <AuthenticatedLayout title="سجل العمليات">
      <div className="space-y-5 text-right w-full" dir="rtl">
        <Card className="border-border shadow-xs bg-card overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between py-3.5 px-5 border-b border-border">
            <CardTitle className="text-sm font-bold text-foreground">سجل العمليات</CardTitle>
            <Button
              variant="ghost"
              size="sm"
              onClick={fetchLogs}
              className="text-xs h-8 gap-1.5 text-muted-foreground hover:text-foreground"
            >
              <RefreshCw className="h-3.5 w-3.5" />
              <span>تحديث</span>
            </Button>
          </CardHeader>
          <CardContent className="p-0">
            {loading ? (
              <div className="flex items-center justify-center py-12 text-muted-foreground gap-2 text-xs">
                <Loader2 className="h-4 w-4 animate-spin text-primary" />
                <span>جاري التحميل...</span>
              </div>
            ) : logs.length === 0 ? (
              <div className="text-center py-12 text-muted-foreground text-xs">
                لا توجد عمليات مسجلة بعد
              </div>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow className="bg-muted/40 hover:bg-muted/40">
                      <TableHead className="w-[50px] text-right text-xs font-semibold">#</TableHead>
                      <TableHead className="text-right text-xs font-semibold">المستخدم</TableHead>
                      <TableHead className="text-right text-xs font-semibold">رقم الشاصي</TableHead>
                      <TableHead className="text-right text-xs font-semibold">المركبة</TableHead>
                      <TableHead className="text-right text-xs font-semibold">الحالة</TableHead>
                      <TableHead className="text-right text-xs font-semibold">التاريخ</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {logs.map((log) => (
                      <TableRow key={log.id}>
                        <TableCell className="font-mono text-xs text-muted-foreground">
                          {log.id}
                        </TableCell>
                        <TableCell className="text-xs font-medium text-foreground">
                          {log.user?.username ? `@${log.user.username}` : `UID: ${log.user_id}`}
                        </TableCell>
                        <TableCell className="font-mono font-semibold text-xs" dir="ltr">
                          <button
                            onClick={() => navigate(`/report/${log.vin}`)}
                            className="hover:underline text-primary flex items-center gap-1 cursor-pointer"
                          >
                            <span>{log.vin}</span>
                            <ExternalLink className="h-3 w-3 inline" />
                          </button>
                        </TableCell>
                        <TableCell className="text-xs text-foreground">
                          {log.vehicle_title || '—'}
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant="outline"
                            className={`text-[10px] ${
                              log.result_status === 'success'
                                ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
                                : 'border-destructive/30 bg-destructive/10 text-destructive'
                            }`}
                          >
                            {log.result_status === 'success' ? 'مكتمل' : log.result_status}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
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
