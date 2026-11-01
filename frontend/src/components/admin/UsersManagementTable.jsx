import React from 'react'
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
import { Users, Sliders, Loader2 } from 'lucide-react'

export default function UsersManagementTable({
  users = [],
  loading = false,
  onRefresh,
  onEditQuota,
  onToggleStatus,
}) {
  return (
    <Card className="border-slate-200 dark:border-slate-800 shadow-sm w-full bg-white dark:bg-slate-900 rounded-xl overflow-hidden">
      <CardHeader className="flex flex-row items-center justify-between pb-3 text-right bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-800">
        <div>
          <CardTitle className="text-base font-bold flex items-center gap-2 text-[#003478] dark:text-blue-300">
            <Users className="h-4.5 w-4.5 text-[#003478] dark:text-blue-300" />
            قائمة التجار وحصص الأرصدة
          </CardTitle>
          <CardDescription className="text-xs text-slate-500">
            جميع الحسابات المسجلة في المنصة مع تفاصيل الرصيد الكلي واليومي وصلاحيات الدخول
          </CardDescription>
        </div>
        {onRefresh && (
          <Button
            variant="outline"
            size="sm"
            onClick={onRefresh}
            className="text-xs h-8 hover:bg-slate-100 dark:hover:bg-slate-800"
          >
            تحديث القائمة
          </Button>
        )}
      </CardHeader>
      <CardContent className="p-0">
        {loading ? (
          <div className="flex flex-col items-center justify-center py-12 text-slate-400 gap-2">
            <Loader2 className="h-6 w-6 animate-spin text-[#003478]" />
            <span className="text-xs">جاري جلب حسابات المستخدمين...</span>
          </div>
        ) : (
          <div className="overflow-x-auto w-full">
            <Table className="w-full text-right">
              <TableHeader className="bg-[#E8F1F8] dark:bg-slate-800">
                <TableRow className="border-b border-slate-200 dark:border-slate-700">
                  <TableHead className="w-[60px] text-right text-xs font-bold text-[#003478] dark:text-blue-200">#</TableHead>
                  <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">المستخدم / التاجر</TableHead>
                  <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">الهاتف</TableHead>
                  <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">الدور</TableHead>
                  <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">الحالة</TableHead>
                  <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">الرصيد الكلي</TableHead>
                  <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">المتبقي اليوم</TableHead>
                  <TableHead className="text-center text-xs font-bold text-[#003478] dark:text-blue-200">الإجراءات والتحكم</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {users.map((u) => {
                  const q = u.quota || {}
                  return (
                    <TableRow key={u.id} className="hover:bg-blue-50/40 dark:hover:bg-slate-800/40 border-b border-slate-100 dark:border-slate-800 transition-colors">
                      <TableCell className="font-mono text-xs text-slate-400">
                        {u.id}
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-col">
                          <span className="font-bold text-xs text-slate-800 dark:text-slate-100">
                            {u.full_name || u.username}
                          </span>
                          <span className="text-[11px] text-slate-400 font-mono" dir="ltr">
                            @{u.username}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell className="text-xs font-mono text-slate-500" dir="ltr">
                        {u.phone || '—'}
                      </TableCell>
                      <TableCell>
                        {u.role === 'admin' ? (
                          <Badge className="bg-[#003478] text-white text-[10px] border-none font-bold">
                            مدير عام
                          </Badge>
                        ) : (
                          <Badge variant="outline" className="text-[10px] border-slate-300 text-slate-700 bg-slate-50">
                            تاجر (Dealer)
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        {u.is_active ? (
                          <Badge className="bg-emerald-100 text-[#1C8237] text-[10px] border-none font-bold">
                            نشط ومفعل
                          </Badge>
                        ) : (
                          <Badge className="bg-rose-100 text-[#D0202E] text-[10px] border-none font-bold">
                            معطل
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-xs">
                        <span className="font-bold text-slate-800 dark:text-slate-200">
                          {q.remaining_total ?? 0}
                        </span>{' '}
                        <span className="text-slate-400 text-[11px]">
                          / {q.total_limit ?? 0}
                        </span>
                      </TableCell>
                      <TableCell className="text-xs">
                        <span className="font-bold text-slate-800 dark:text-slate-200">
                          {q.remaining_daily ?? 0}
                        </span>{' '}
                        <span className="text-slate-400 text-[11px]">
                          / {q.daily_limit ?? 0}
                        </span>
                      </TableCell>
                      <TableCell className="text-center">
                        <div className="flex items-center justify-center gap-1.5">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => onEditQuota(u)}
                            className="text-xs h-7 px-2.5 gap-1 font-semibold border-slate-300 text-[#003478] hover:bg-blue-50"
                            title="تعديل الحصة والرصيد"
                          >
                            <Sliders className="h-3 w-3" />
                            تعديل الباقة
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => onToggleStatus(u)}
                            className={`text-xs h-7 px-2 font-bold ${
                              u.is_active
                                ? 'text-amber-600 hover:text-amber-700 hover:bg-amber-50'
                                : 'text-[#1C8237] hover:text-emerald-700 hover:bg-emerald-50'
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
