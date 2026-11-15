import React, { useState, useMemo } from 'react'
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
  ExternalLink,
  Loader2,
  FileText,
  RefreshCw,
  Search,
  ChevronRight,
  ChevronLeft,
  Calendar,
  Clock,
  Sparkles,
  CheckCircle2,
  Eye,
} from 'lucide-react'
import { isVinUnread, markVinViewed } from '@/lib/api'

export default function RecentSearchesTable({
  searches = [],
  pendingSearches = [],
  loading = false,
  onRefresh,
  onSelectReport,
  defaultPageSize = 10,
}) {
  const [filterText, setFilterText] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(defaultPageSize)
  const [, setReadTick] = useState(0)

  const handleOpenReport = (vin) => {
    markVinViewed(vin)
    setReadTick((t) => t + 1)
    if (onSelectReport) {
      onSelectReport(vin)
    }
  }

  const combinedSearches = useMemo(() => {
    const activeList = Array.isArray(pendingSearches) ? pendingSearches : []
    const historyList = Array.isArray(searches) ? searches : []
    return [...activeList, ...historyList]
  }, [pendingSearches, searches])

  const filteredSearches = useMemo(() => {
    const q = filterText.trim().toLowerCase()
    if (!q) return combinedSearches
    return combinedSearches.filter(
      (item) =>
        (item.vin || '').toLowerCase().includes(q) ||
        (item.vehicle_title || '').toLowerCase().includes(q)
    )
  }, [combinedSearches, filterText])

  const totalPages = Math.max(1, Math.ceil(filteredSearches.length / pageSize))
  const safePage = Math.min(page, totalPages)

  const paginatedItems = useMemo(() => {
    const start = (safePage - 1) * pageSize
    return filteredSearches.slice(start, start + pageSize)
  }, [filteredSearches, safePage, pageSize])

  const handleFilterChange = (e) => {
    setFilterText(e.target.value)
    setPage(1)
  }

  return (
    <Card className="border-border shadow-xs bg-card overflow-hidden">
      <CardHeader className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 py-3.5 px-4 sm:px-5 border-b border-border">
        <div className="flex items-center justify-between sm:justify-start gap-2.5 flex-wrap">
          <CardTitle className="text-sm font-bold text-foreground">
            التقارير المسحوبة وقائمة الجلب المباشر
          </CardTitle>
          <Badge variant="secondary" className="text-[11px] font-mono">
            {filteredSearches.length}
          </Badge>
          {pendingSearches.length > 0 && (
            <Badge className="bg-amber-500/15 text-amber-700 dark:text-amber-300 border border-amber-500/30 text-[11px] gap-1">
              <Loader2 className="h-3 w-3 animate-spin" />
              {pendingSearches.length} قيد الجلب الآن
            </Badge>
          )}
          {onRefresh && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onRefresh}
              className="sm:hidden text-xs h-8 gap-1.5 text-muted-foreground hover:text-foreground cursor-pointer"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
              <span>تحديث</span>
            </Button>
          )}
        </div>

        <div className="flex items-center gap-2 w-full sm:w-auto">
          <div className="relative flex-1 sm:w-64">
            <Search className="h-3.5 w-3.5 text-muted-foreground absolute right-2.5 top-2.5" />
            <Input
              placeholder="بحث برقم الشاصي أو المركبة..."
              value={filterText}
              onChange={handleFilterChange}
              className="pr-8 h-8 text-xs"
            />
          </div>
          {onRefresh && (
            <Button
              variant="outline"
              size="sm"
              onClick={onRefresh}
              className="hidden sm:inline-flex text-xs h-8 gap-1.5 text-muted-foreground hover:text-foreground cursor-pointer shrink-0"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
              <span>تحديث</span>
            </Button>
          )}
        </div>
      </CardHeader>

      <CardContent className="p-0">
        {loading && combinedSearches.length === 0 ? (
          <div className="flex items-center justify-center py-12 text-muted-foreground gap-2 text-xs">
            <Loader2 className="h-4 w-4 animate-spin text-primary" />
            <span>جاري التحميل...</span>
          </div>
        ) : filteredSearches.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-center text-muted-foreground">
            <FileText className="h-8 w-8 stroke-1 mb-2 opacity-40" />
            <p className="text-xs">
              {filterText ? 'لا توجد نتائج مطابقة للبحث' : 'لا توجد تقارير مسحوبة حتى الآن'}
            </p>
          </div>
        ) : (
          <>
            {/* 1. Desktop Data Grid View (md and up) */}
            <div className="hidden md:block overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow className="bg-muted/40 hover:bg-muted/40">
                    <TableHead className="text-right text-xs font-semibold">رقم الشاصي</TableHead>
                    <TableHead className="text-right text-xs font-semibold">المركبة</TableHead>
                    <TableHead className="text-right text-xs font-semibold">حالة الجلب والفتح</TableHead>
                    <TableHead className="text-right text-xs font-semibold">الوقت / التاريخ</TableHead>
                    <TableHead className="text-left text-xs font-semibold">عرض التقرير</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paginatedItems.map((item) => {
                    const isProcessing =
                      item.result_status === 'processing' || item.status === 'processing'
                    const isOk =
                      !isProcessing &&
                      (item.result_status === 'success' || item.status === 'success')
                    const isUnread = isOk && isVinUnread(item)

                    return (
                      <TableRow
                        key={item.id || item.vin}
                        className={
                          isProcessing
                            ? 'bg-amber-500/5 hover:bg-amber-500/10 transition-colors'
                            : isUnread
                              ? 'bg-sky-500/5 hover:bg-sky-500/10 transition-colors'
                              : ''
                        }
                      >
                        <TableCell
                          className="font-mono font-semibold text-xs text-primary"
                          dir="ltr"
                        >
                          {item.vin}
                        </TableCell>
                        <TableCell className="text-xs font-medium text-foreground">
                          {item.vehicle_title || '—'}
                        </TableCell>
                        <TableCell>
                          {isProcessing ? (
                            <div className="flex flex-wrap items-center gap-1.5">
                              <Badge
                                variant="outline"
                                className="border-amber-500/40 bg-amber-500/15 text-amber-700 dark:text-amber-300 text-[10px] gap-1 font-bold"
                              >
                                <Loader2 className="h-3 w-3 animate-spin" />
                                {item.step_ar || 'جاري الجلب...'}
                              </Badge>
                              {item.estimated_seconds > 0 && (
                                <span className="inline-flex items-center gap-1 text-[11px] font-mono font-bold text-amber-700 dark:text-amber-300 bg-amber-500/10 px-2 py-0.5 rounded border border-amber-500/25">
                                  <Clock className="h-3 w-3" />~{item.estimated_seconds} ثانية
                                </span>
                              )}
                            </div>
                          ) : isOk ? (
                            <div className="flex flex-wrap items-center gap-1.5">
                              <Badge
                                variant="outline"
                                className="border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 text-[10px] gap-1"
                              >
                                <CheckCircle2 className="h-3 w-3" />
                                تم الجلب
                              </Badge>
                              {isUnread ? (
                                <Badge className="bg-[#00599c] hover:bg-[#00599c] text-white text-[10px] gap-1 font-bold shadow-2xs">
                                  <Sparkles className="h-2.5 w-2.5" />
                                  جديد (غير مفتوح)
                                </Badge>
                              ) : (
                                <Badge
                                  variant="secondary"
                                  className="text-[10px] text-muted-foreground gap-1 font-normal"
                                >
                                  <Eye className="h-2.5 w-2.5" />
                                  تم الفتح
                                </Badge>
                              )}
                            </div>
                          ) : (
                            <Badge
                              variant="outline"
                              className="border-destructive/30 bg-destructive/10 text-destructive text-[10px]"
                            >
                              {item.error_message || 'غير مكتمل'}
                            </Badge>
                          )}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {isProcessing
                            ? 'الآن (قيد السحب المباشر)'
                            : new Date(item.created_at).toLocaleString('ar-EG')}
                        </TableCell>
                        <TableCell className="text-left">
                          {isProcessing ? (
                            <span className="inline-flex items-center gap-1.5 text-xs text-amber-600 dark:text-amber-400 font-semibold">
                              <Loader2 className="h-3.5 w-3.5 animate-spin" />
                              بانتظار الجلب...
                            </span>
                          ) : isOk ? (
                            <Button
                              size="sm"
                              variant={isUnread ? 'default' : 'outline'}
                              onClick={() => handleOpenReport(item.vin)}
                              className={`text-xs h-7 px-3 gap-1.5 font-bold cursor-pointer ${
                                isUnread
                                  ? 'bg-[#00599c] hover:bg-[#004478] text-white shadow-xs'
                                  : 'text-primary border-primary/20 hover:bg-primary/5'
                              }`}
                            >
                              <span>{isUnread ? 'عرض التقرير (جديد)' : 'عرض التقرير'}</span>
                              <ExternalLink className="h-3 w-3" />
                            </Button>
                          ) : null}
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>

            {/* 2. Mobile Responsive Card List View (< md) */}
            <div className="md:hidden divide-y divide-border">
              {paginatedItems.map((item) => {
                const isProcessing =
                  item.result_status === 'processing' || item.status === 'processing'
                const isOk =
                  !isProcessing &&
                  (item.result_status === 'success' || item.status === 'success')
                const isUnread = isOk && isVinUnread(item)

                return (
                  <div
                    key={item.id || item.vin}
                    className={`p-3.5 space-y-2.5 transition-colors ${
                      isProcessing
                        ? 'bg-amber-500/5'
                        : isUnread
                          ? 'bg-sky-500/5'
                          : 'hover:bg-muted/20'
                    }`}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <span
                          className="font-mono font-bold text-xs text-primary block tracking-wider"
                          dir="ltr"
                        >
                          {item.vin}
                        </span>
                        <span className="text-xs font-semibold text-foreground block mt-0.5">
                          {item.vehicle_title || '—'}
                        </span>
                      </div>

                      <div className="flex flex-wrap items-center gap-1 shrink-0">
                        {isProcessing ? (
                          <Badge
                            variant="outline"
                            className="border-amber-500/40 bg-amber-500/15 text-amber-700 dark:text-amber-300 text-[10px] gap-1"
                          >
                            <Loader2 className="h-3 w-3 animate-spin" />
                            {item.estimated_seconds > 0
                              ? `جاري الجلب (~${item.estimated_seconds}s)`
                              : 'جاري الجلب...'}
                          </Badge>
                        ) : isOk ? (
                          <>
                            <Badge
                              variant="outline"
                              className="border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 text-[10px]"
                            >
                              تم الجلب
                            </Badge>
                            {isUnread ? (
                              <Badge className="bg-[#00599c] text-white text-[10px] gap-1">
                                جديد
                              </Badge>
                            ) : (
                              <Badge variant="secondary" className="text-[10px]">
                                تم الفتح
                              </Badge>
                            )}
                          </>
                        ) : (
                          <Badge
                            variant="outline"
                            className="border-destructive/30 bg-destructive/10 text-destructive text-[10px] shrink-0"
                          >
                            غير مكتمل
                          </Badge>
                        )}
                      </div>
                    </div>

                    <div className="flex items-center justify-between gap-2 pt-1">
                      <span className="inline-flex items-center gap-1 text-[11px] text-muted-foreground">
                        <Calendar className="h-3 w-3" />
                        {isProcessing
                          ? item.step_ar || 'الآن (قيد السحب)'
                          : new Date(item.created_at).toLocaleString('ar-EG')}
                      </span>

                      {isProcessing ? (
                        <span className="inline-flex items-center gap-1 text-xs text-amber-600 font-semibold">
                          <Loader2 className="h-3.5 w-3.5 animate-spin" />
                          بانتظار الجلب...
                        </span>
                      ) : isOk ? (
                        <Button
                          size="sm"
                          variant={isUnread ? 'default' : 'outline'}
                          onClick={() => handleOpenReport(item.vin)}
                          className="text-xs h-8 px-3 gap-1.5 font-semibold cursor-pointer"
                        >
                          <span>{isUnread ? 'عرض التقرير (جديد)' : 'عرض التقرير'}</span>
                          <ExternalLink className="h-3 w-3" />
                        </Button>
                      ) : null}
                    </div>
                  </div>
                )
              })}
            </div>

            {/* 3. Pagination Footer */}
            {filteredSearches.length > 5 && (
              <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 border-t border-border bg-muted/20 text-xs">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <span>
                    عرض {(safePage - 1) * pageSize + 1} –{' '}
                    {Math.min(safePage * pageSize, filteredSearches.length)} من{' '}
                    {filteredSearches.length}
                  </span>
                  <select
                    value={pageSize}
                    onChange={(e) => {
                      setPageSize(Number(e.target.value))
                      setPage(1)
                    }}
                    className="h-7 rounded border border-border bg-background px-2 text-xs text-foreground"
                  >
                    <option value={10}>10 / صفحة</option>
                    <option value={20}>20 / صفحة</option>
                    <option value={50}>50 / صفحة</option>
                  </select>
                </div>

                <div className="flex items-center gap-1.5">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={safePage <= 1}
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    className="h-7 px-2.5 text-xs gap-1 cursor-pointer"
                  >
                    <ChevronRight className="h-3.5 w-3.5" />
                    <span>السابق</span>
                  </Button>
                  <span className="px-2 font-mono font-semibold text-foreground">
                    {safePage} / {totalPages}
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={safePage >= totalPages}
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    className="h-7 px-2.5 text-xs gap-1 cursor-pointer"
                  >
                    <span>التالي</span>
                    <ChevronLeft className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}
