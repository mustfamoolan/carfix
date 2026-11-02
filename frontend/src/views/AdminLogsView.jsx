import React, { useState, useEffect, useMemo } from 'react'
import AuthenticatedLayout from '@/components/layout/AuthenticatedLayout'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import {
  Loader2,
  ExternalLink,
  RefreshCw,
  Search,
  ChevronRight,
  ChevronLeft,
  Clock,
  Car,
  User,
} from 'lucide-react'
import { getAdminSearchLogs, getCachedData } from '@/lib/api'
import { useNavigate } from 'react-router-dom'

export default function AdminLogsView() {
  const cachedRes = getCachedData('admin:logs:limit=150') || getCachedData('admin:logs:')
  const initialList =
    cachedRes?.logs ||
    cachedRes?.data?.logs ||
    (Array.isArray(cachedRes?.data) ? cachedRes.data : Array.isArray(cachedRes) ? cachedRes : null)

  const [logs, setLogs] = useState(initialList || [])
  const [loading, setLoading] = useState(!initialList)
  const [searchQuery, setSearchQuery] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(15)
  const navigate = useNavigate()

  const fetchLogs = async (showSpinner = false) => {
    if (showSpinner || logs.length === 0) setLoading(true)
    try {
      const res = await getAdminSearchLogs({ limit: 150 })
      const list =
        res?.logs ||
        res?.data?.logs ||
        (Array.isArray(res?.data) ? res.data : Array.isArray(res) ? res : [])
      setLogs(list)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchLogs(false)
  }, [])

  const filteredLogs = useMemo(() => {
    const q = searchQuery.trim().toLowerCase()
    if (!q) return logs
    return logs.filter(
      (log) =>
        (log.vin || '').toLowerCase().includes(q) ||
        (log.vehicle_title || '').toLowerCase().includes(q) ||
        (log.user?.username || '').toLowerCase().includes(q) ||
        (log.user?.full_name || '').toLowerCase().includes(q)
    )
  }, [logs, searchQuery])

  const totalPages = Math.max(1, Math.ceil(filteredLogs.length / pageSize))
  const currentPage = Math.min(page, totalPages)
  const paginatedLogs = useMemo(() => {
    const start = (currentPage - 1) * pageSize
    return filteredLogs.slice(start, start + pageSize)
  }, [filteredLogs, currentPage, pageSize])

  return (
    <AuthenticatedLayout title="سجل العمليات">
      <div className="space-y-5 text-right w-full" dir="rtl">
        <Card className="border-border shadow-xs bg-card overflow-hidden">
          <CardHeader className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 py-3.5 px-4 sm:px-5 border-b border-border">
            <div className="flex items-center justify-between sm:justify-start gap-2.5">
              <CardTitle className="text-sm font-bold text-foreground">سجل العمليات</CardTitle>
              <Badge variant="secondary" className="text-[11px] font-mono">
                {filteredLogs.length}
              </Badge>
            </div>

            <div className="flex items-center gap-2 w-full sm:w-auto">
              <div className="relative flex-1 sm:w-64">
                <Search className="h-3.5 w-3.5 text-muted-foreground absolute right-2.5 top-2.5" />
                <Input
                  value={searchQuery}
                  onChange={(e) => {
                    setSearchQuery(e.target.value)
                    setPage(1)
                  }}
                  placeholder="بحث برقم الشاصي، المركبة، أو المستخدم..."
                  className="h-8 pr-8 text-xs"
                />
              </div>

              <Button
                variant="outline"
                size="sm"
                onClick={() => fetchLogs(true)}
                className="text-xs h-8 gap-1.5 text-muted-foreground hover:text-foreground shrink-0 cursor-pointer"
              >
                <RefreshCw className="h-3.5 w-3.5" />
                <span className="hidden sm:inline">تحديث</span>
              </Button>
            </div>
          </CardHeader>

          <CardContent className="p-0">
            {loading && logs.length === 0 ? (
              <div className="flex items-center justify-center py-12 text-muted-foreground gap-2 text-xs">
                <Loader2 className="h-4 w-4 animate-spin text-primary" />
                <span>جاري التحميل...</span>
              </div>
            ) : filteredLogs.length === 0 ? (
              <div className="text-center py-12 text-muted-foreground text-xs">
                {searchQuery ? 'لا توجد نتائج مطابقة للبحث' : 'لا توجد عمليات مسجلة بعد'}
              </div>
            ) : (
              <>
                {/* Desktop Table View (md and up) */}
                <div className="hidden md:block overflow-x-auto">
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
                      {paginatedLogs.map((log) => (
                        <TableRow key={log.id} className="hover:bg-muted/30">
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
                              {log.result_status === 'success' ? 'مكتمل' : 'غير مكتمل'}
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

                {/* Mobile Responsive Cards View (below md) */}
                <div className="md:hidden divide-y divide-border">
                  {paginatedLogs.map((log) => (
                    <div
                      key={log.id}
                      onClick={() => log.result_status === 'success' && navigate(`/report/${log.vin}`)}
                      className="p-3.5 hover:bg-muted/30 transition-colors flex flex-col gap-2"
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div className="space-y-1 min-w-0">
                          <div className="flex items-center gap-1.5 text-xs font-bold text-foreground truncate">
                            <Car className="h-3.5 w-3.5 text-primary shrink-0" />
                            <span className="truncate">{log.vehicle_title || 'تقرير مركبة'}</span>
                          </div>
                          <div
                            className="inline-block font-mono text-xs font-bold text-primary bg-primary/10 px-2 py-0.5 rounded tracking-wider"
                            dir="ltr"
                          >
                            {log.vin}
                          </div>
                        </div>
                        <Badge
                          variant="outline"
                          className={`text-[10px] shrink-0 ${
                            log.result_status === 'success'
                              ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
                              : 'border-destructive/30 bg-destructive/10 text-destructive'
                          }`}
                        >
                          {log.result_status === 'success' ? 'مكتمل' : 'غير مكتمل'}
                        </Badge>
                      </div>

                      <div className="flex items-center justify-between text-[11px] text-muted-foreground pt-1 border-t border-border/50">
                        <span className="flex items-center gap-1 font-medium text-foreground">
                          <User className="h-3 w-3 text-muted-foreground" />
                          {log.user?.username ? `@${log.user.username}` : `UID: ${log.user_id}`}
                        </span>
                        <span className="flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          {new Date(log.created_at).toLocaleString('ar-EG', {
                            dateStyle: 'short',
                            timeStyle: 'short',
                          })}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>

                {/* Pagination Footer */}
                <div className="flex flex-col sm:flex-row items-center justify-between gap-3 px-4 py-3 border-t border-border bg-muted/20 text-xs">
                  <div className="flex items-center gap-2 text-muted-foreground">
                    <span>
                      عرض {(currentPage - 1) * pageSize + 1} -{' '}
                      {Math.min(currentPage * pageSize, filteredLogs.length)} من إجمالي{' '}
                      <strong className="text-foreground">{filteredLogs.length}</strong> عملية
                    </span>
                    <select
                      value={pageSize}
                      onChange={(e) => {
                        setPageSize(Number(e.target.value))
                        setPage(1)
                      }}
                      className="h-7 rounded border border-border bg-background px-1.5 text-[11px] font-medium text-foreground"
                    >
                      <option value={15}>15 / صفحة</option>
                      <option value={30}>30 / صفحة</option>
                      <option value={50}>50 / صفحة</option>
                    </select>
                  </div>

                  {totalPages > 1 && (
                    <div className="flex items-center gap-1.5">
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={currentPage <= 1}
                        onClick={() => setPage((p) => Math.max(1, p - 1))}
                        className="h-7 px-2.5 text-xs gap-1 cursor-pointer"
                      >
                        <ChevronRight className="h-3.5 w-3.5" />
                        السابق
                      </Button>
                      <span className="px-2 font-mono text-xs font-semibold">
                        {currentPage} / {totalPages}
                      </span>
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={currentPage >= totalPages}
                        onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                        className="h-7 px-2.5 text-xs gap-1 cursor-pointer"
                      >
                        التالي
                        <ChevronLeft className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  )}
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </AuthenticatedLayout>
  )
}
