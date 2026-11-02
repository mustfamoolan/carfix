import React, { useState, useMemo } from 'react'
import { Card, CardContent } from '@/components/ui/card'
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
import {
  Sliders,
  Loader2,
  Pencil,
  Trash2,
  Users,
  ChevronRight,
  ChevronLeft,
  Phone,
} from 'lucide-react'

export default function UsersManagementTable({
  users = [],
  loading = false,
  onEditUser,
  onEditQuota,
  onToggleStatus,
  onDeleteUser,
}) {
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const totalPages = Math.max(1, Math.ceil(users.length / pageSize))
  const safePage = Math.min(page, totalPages)

  const paginatedUsers = useMemo(() => {
    const start = (safePage - 1) * pageSize
    return users.slice(start, start + pageSize)
  }, [users, safePage, pageSize])

  return (
    <Card className="border-border shadow-xs bg-card overflow-hidden">
      <CardContent className="p-0">
        {loading && users.length === 0 ? (
          <div className="flex items-center justify-center py-12 text-muted-foreground gap-2 text-xs">
            <Loader2 className="h-4 w-4 animate-spin text-primary" />
            <span>جاري التحميل...</span>
          </div>
        ) : users.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-2">
            <Users className="h-8 w-8 opacity-30" />
            <p className="text-xs">لا يوجد مستخدمون مطابقون للبحث</p>
          </div>
        ) : (
          <>
            {/* 1. Desktop Data Grid View (md and up) */}
            <div className="hidden md:block overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow className="bg-muted/40 hover:bg-muted/40">
                    <TableHead className="w-[50px] text-right text-xs font-semibold">#</TableHead>
                    <TableHead className="text-right text-xs font-semibold">المستخدم / المعرض</TableHead>
                    <TableHead className="text-right text-xs font-semibold">الهاتف</TableHead>
                    <TableHead className="text-right text-xs font-semibold">الدور</TableHead>
                    <TableHead className="text-right text-xs font-semibold">الحالة</TableHead>
                    <TableHead className="text-right text-xs font-semibold">الرصيد الكلي</TableHead>
                    <TableHead className="text-right text-xs font-semibold">المتبقي اليوم</TableHead>
                    <TableHead className="text-right text-xs font-semibold">الفحوصات</TableHead>
                    <TableHead className="text-left text-xs font-semibold">الإجراءات</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paginatedUsers.map((u) => {
                    const q = u.quota || {}
                    const remTotal =
                      q.remaining_total !== undefined
                        ? q.remaining_total
                        : Math.max(0, (q.total_limit || 0) - (q.used_total || 0))
                    const remDaily =
                      q.remaining_daily !== undefined
                        ? q.remaining_daily
                        : Math.max(0, (q.daily_limit || 0) - (q.used_today || 0))

                    return (
                      <TableRow key={u.id}>
                        <TableCell className="font-mono text-xs text-muted-foreground">
                          {u.id}
                        </TableCell>
                        <TableCell>
                          <div className="flex flex-col">
                            <span className="font-bold text-xs text-foreground">
                              {u.full_name || u.username}
                            </span>
                            <span className="text-[11px] text-muted-foreground font-mono" dir="ltr">
                              @{u.username}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell className="text-xs font-mono text-muted-foreground" dir="ltr">
                          {u.phone || '—'}
                        </TableCell>
                        <TableCell>
                          {u.role === 'admin' ? (
                            <Badge className="text-[10px] font-semibold">أدمن</Badge>
                          ) : (
                            <Badge variant="secondary" className="text-[10px]">
                              تاجر / مستخدم
                            </Badge>
                          )}
                        </TableCell>
                        <TableCell>
                          {u.is_active ? (
                            <Badge
                              variant="outline"
                              className="border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 text-[10px]"
                            >
                              نشط
                            </Badge>
                          ) : (
                            <Badge
                              variant="outline"
                              className="border-destructive/30 bg-destructive/10 text-destructive text-[10px]"
                            >
                              معطل
                            </Badge>
                          )}
                        </TableCell>
                        <TableCell className="text-xs font-mono">
                          <span className="font-bold text-primary">{remTotal}</span>{' '}
                          <span className="text-muted-foreground">/ {q.total_limit ?? 0}</span>
                        </TableCell>
                        <TableCell className="text-xs font-mono">
                          <span className="font-bold text-foreground">{remDaily}</span>{' '}
                          <span className="text-muted-foreground">/ {q.daily_limit ?? 0}</span>
                        </TableCell>
                        <TableCell className="text-xs font-mono font-semibold text-muted-foreground">
                          {u.searches_count ?? q.used_total ?? 0}
                        </TableCell>
                        <TableCell className="text-left">
                          <div className="flex items-center justify-end gap-1.5">
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => onEditUser && onEditUser(u)}
                              className="text-xs h-7 px-2.5 gap-1 cursor-pointer"
                              title="تعديل بيانات الحساب وكلمة المرور"
                            >
                              <Pencil className="h-3 w-3" />
                              تعديل
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => onEditQuota && onEditQuota(u)}
                              className="text-xs h-7 px-2.5 gap-1 cursor-pointer text-primary border-primary/25 hover:bg-primary/5"
                              title="تعديل وشحن الرصيد"
                            >
                              <Sliders className="h-3 w-3" />
                              الرصيد
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => onToggleStatus && onToggleStatus(u)}
                              className={`text-xs h-7 px-2 cursor-pointer ${
                                u.is_active
                                  ? 'text-amber-600 hover:text-amber-700 hover:bg-amber-50'
                                  : 'text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50'
                              }`}
                            >
                              {u.is_active ? 'تعطيل' : 'تفعيل'}
                            </Button>
                            {onDeleteUser && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => onDeleteUser(u)}
                                className="text-xs h-7 px-2 text-muted-foreground hover:text-destructive hover:bg-destructive/10 cursor-pointer"
                                title="حذف الحساب"
                              >
                                <Trash2 className="h-3.5 w-3.5" />
                              </Button>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>

            {/* 2. Mobile Responsive Card List View (< md) */}
            <div className="md:hidden divide-y divide-border">
              {paginatedUsers.map((u) => {
                const q = u.quota || {}
                const remTotal =
                  q.remaining_total !== undefined
                    ? q.remaining_total
                    : Math.max(0, (q.total_limit || 0) - (q.used_total || 0))
                const remDaily =
                  q.remaining_daily !== undefined
                    ? q.remaining_daily
                    : Math.max(0, (q.daily_limit || 0) - (q.used_today || 0))

                return (
                  <div key={u.id} className="p-4 space-y-3 hover:bg-muted/20 transition-colors">
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-bold text-sm text-foreground">
                            {u.full_name || u.username}
                          </span>
                          {u.role === 'admin' ? (
                            <Badge className="text-[10px]">أدمن</Badge>
                          ) : (
                            <Badge variant="secondary" className="text-[10px]">
                              تاجر
                            </Badge>
                          )}
                        </div>
                        <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                          <span className="font-mono" dir="ltr">
                            @{u.username}
                          </span>
                          {u.phone && (
                            <span className="inline-flex items-center gap-1 font-mono" dir="ltr">
                              <Phone className="h-3 w-3" />
                              {u.phone}
                            </span>
                          )}
                        </div>
                      </div>

                      {u.is_active ? (
                        <Badge
                          variant="outline"
                          className="border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 text-[10px]"
                        >
                          نشط
                        </Badge>
                      ) : (
                        <Badge
                          variant="outline"
                          className="border-destructive/30 bg-destructive/10 text-destructive text-[10px]"
                        >
                          معطل
                        </Badge>
                      )}
                    </div>

                    {/* Quota Mini Grid */}
                    <div className="grid grid-cols-3 gap-2 rounded-md bg-muted/40 p-2.5 text-center border border-border/60">
                      <div>
                        <span className="text-[10px] text-muted-foreground block">الرصيد الكلي</span>
                        <span className="text-xs font-mono font-bold text-primary">
                          {remTotal} / {q.total_limit ?? 0}
                        </span>
                      </div>
                      <div>
                        <span className="text-[10px] text-muted-foreground block">المتبقي اليوم</span>
                        <span className="text-xs font-mono font-bold text-foreground">
                          {remDaily} / {q.daily_limit ?? 0}
                        </span>
                      </div>
                      <div>
                        <span className="text-[10px] text-muted-foreground block">الفحوصات</span>
                        <span className="text-xs font-mono font-bold text-foreground">
                          {u.searches_count ?? q.used_total ?? 0}
                        </span>
                      </div>
                    </div>

                    {/* Action Buttons Row */}
                    <div className="flex flex-wrap items-center justify-end gap-1.5 pt-1">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => onEditUser && onEditUser(u)}
                        className="text-xs h-8 px-3 gap-1 cursor-pointer flex-1 sm:flex-none"
                      >
                        <Pencil className="h-3 w-3" />
                        تعديل
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => onEditQuota && onEditQuota(u)}
                        className="text-xs h-8 px-3 gap-1 cursor-pointer text-primary border-primary/25 hover:bg-primary/5 flex-1 sm:flex-none"
                      >
                        <Sliders className="h-3 w-3" />
                        الرصيد
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => onToggleStatus && onToggleStatus(u)}
                        className={`text-xs h-8 px-2.5 cursor-pointer ${
                          u.is_active ? 'text-amber-600' : 'text-emerald-600'
                        }`}
                      >
                        {u.is_active ? 'تعطيل' : 'تفعيل'}
                      </Button>
                      {onDeleteUser && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => onDeleteUser(u)}
                          className="text-xs h-8 px-2.5 text-muted-foreground hover:text-destructive hover:bg-destructive/10 cursor-pointer"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>

            {/* 3. Pagination Footer */}
            {users.length > 5 && (
              <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 border-t border-border bg-muted/20 text-xs">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <span>
                    عرض {(safePage - 1) * pageSize + 1} –{' '}
                    {Math.min(safePage * pageSize, users.length)} من {users.length}
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
