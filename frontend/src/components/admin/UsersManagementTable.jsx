import React from 'react'
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
import { Sliders, Loader2 } from 'lucide-react'

export default function UsersManagementTable({
  users = [],
  loading = false,
  onEditQuota,
  onToggleStatus,
}) {
  return (
    <Card className="border-border shadow-xs bg-card overflow-hidden">
      <CardContent className="p-0">
        {loading ? (
          <div className="flex items-center justify-center py-12 text-muted-foreground gap-2 text-xs">
            <Loader2 className="h-4 w-4 animate-spin text-primary" />
            <span>جاري التحميل...</span>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/40 hover:bg-muted/40">
                  <TableHead className="w-[50px] text-right text-xs font-semibold">#</TableHead>
                  <TableHead className="text-right text-xs font-semibold">المستخدم</TableHead>
                  <TableHead className="text-right text-xs font-semibold">الهاتف</TableHead>
                  <TableHead className="text-right text-xs font-semibold">الدور</TableHead>
                  <TableHead className="text-right text-xs font-semibold">الحالة</TableHead>
                  <TableHead className="text-right text-xs font-semibold">الرصيد الكلي</TableHead>
                  <TableHead className="text-right text-xs font-semibold">المتبقي اليوم</TableHead>
                  <TableHead className="text-left text-xs font-semibold">الإجراءات</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {users.map((u) => {
                  const q = u.quota || {}
                  return (
                    <TableRow key={u.id}>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {u.id}
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-col">
                          <span className="font-medium text-xs text-foreground">
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
                            مستخدم
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
                        <span className="font-semibold text-foreground">
                          {q.remaining_total ?? 0}
                        </span>{' '}
                        <span className="text-muted-foreground">/ {q.total_limit ?? 0}</span>
                      </TableCell>
                      <TableCell className="text-xs font-mono">
                        <span className="font-semibold text-foreground">
                          {q.remaining_daily ?? 0}
                        </span>{' '}
                        <span className="text-muted-foreground">/ {q.daily_limit ?? 0}</span>
                      </TableCell>
                      <TableCell className="text-left">
                        <div className="flex items-center justify-end gap-1.5">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => onEditQuota(u)}
                            className="text-xs h-7 px-2.5 gap-1 cursor-pointer"
                          >
                            <Sliders className="h-3 w-3" />
                            الرصيد
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => onToggleStatus(u)}
                            className={`text-xs h-7 px-2 cursor-pointer ${
                              u.is_active
                                ? 'text-muted-foreground hover:text-destructive'
                                : 'text-emerald-600 hover:text-emerald-700'
                            }`}
                          >
                            {u.is_active ? 'تعطيل' : 'تفعيل'}
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
