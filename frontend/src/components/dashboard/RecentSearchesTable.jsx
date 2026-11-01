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
import { Clock, ExternalLink, Loader2, FileText, CheckCircle2, AlertCircle } from 'lucide-react'

export default function RecentSearchesTable({
  searches = [],
  loading = false,
  onRefresh,
  onSelectReport,
}) {
  return (
    <Card className="border-slate-200 dark:border-slate-800 shadow-sm w-full overflow-hidden bg-white dark:bg-slate-900 rounded-xl">
      <CardHeader className="flex flex-row items-center justify-between pb-3 text-right bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-800">
        <div>
          <CardTitle className="text-sm sm:text-base font-bold flex items-center gap-2 text-[#003478] dark:text-blue-300">
            <FileText className="h-4.5 w-4.5 text-[#003478] dark:text-blue-300" />
            Recently Run Reports | التقارير المسحوبة مؤخراً
          </CardTitle>
          <CardDescription className="text-xs text-slate-500">
            سجل تقارير كارفاكس التي تم شراؤها واستخراجها مع إمكانية إعادة فتحها أو طباعتها فوراً بدون خصم رصيد إضافي
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
            <span className="text-xs">جاري تحميل سجل التقارير...</span>
          </div>
        ) : searches.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-center text-slate-400">
            <FileText className="h-10 w-10 stroke-1 mb-2 text-slate-300 dark:text-slate-600" />
            <p className="text-sm font-semibold text-slate-600 dark:text-slate-300">لا توجد تقارير مسحوبة حتى الآن</p>
            <p className="text-xs text-slate-400 mt-1">
              أدخل رقم شاصي من 17 رمزاً في الصندوق أعلاه لاستخراج أول تقرير لك
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto w-full">
            <Table className="w-full text-right">
              <TableHeader className="bg-[#E8F1F8] dark:bg-slate-800">
                <TableRow className="border-b border-slate-200 dark:border-slate-700">
                  <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">
                    رقم الشاصي (VIN)
                  </TableHead>
                  <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">
                    حالة الفحص
                  </TableHead>
                  <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">
                    مصدر الجلب
                  </TableHead>
                  <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">
                    زمن الاستجابة
                  </TableHead>
                  <TableHead className="text-right text-xs font-bold text-[#003478] dark:text-blue-200">
                    تاريخ العملية
                  </TableHead>
                  <TableHead className="text-center text-xs font-bold text-[#003478] dark:text-blue-200">
                    الإجراء
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {searches.map((item) => (
                  <TableRow
                    key={item.id}
                    className="hover:bg-blue-50/40 dark:hover:bg-slate-800/40 border-b border-slate-100 dark:border-slate-800 transition-colors"
                  >
                    <TableCell className="font-mono font-bold text-xs tracking-wider text-[#003478] dark:text-blue-300" dir="ltr">
                      {item.vin}
                    </TableCell>
                    <TableCell>
                      {item.status === 'success' ? (
                        <span className="inline-flex items-center gap-1 text-[11px] font-bold text-[#1C8237] bg-[#EBF7EE] dark:bg-emerald-950/40 px-2 py-0.5 rounded-full border border-emerald-200 dark:border-emerald-800">
                          <CheckCircle2 className="h-3 w-3" /> مكتمل بنجاح
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 text-[11px] font-bold text-[#D0202E] bg-[#FDF2F2] dark:bg-rose-950/40 px-2 py-0.5 rounded-full border border-rose-200 dark:border-rose-800">
                          <AlertCircle className="h-3 w-3" /> {item.status}
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="text-xs text-slate-600 dark:text-slate-400">
                      {item.fetch_source === 'cache' ? (
                        <Badge variant="outline" className="text-[10px] bg-slate-50 dark:bg-slate-800 text-slate-600 border-slate-300">
                          ذاكرة الكاش (فوري)
                        </Badge>
                      ) : item.fetch_source === 'worker_canada' ? (
                        <Badge className="text-[10px] bg-blue-100 text-[#003478] border-none">
                          سيرفر كندا (مباشر)
                        </Badge>
                      ) : (
                        <Badge variant="outline" className="text-[10px] text-slate-500">
                          {item.fetch_source || 'محلي'}
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-xs font-mono text-slate-500">
                      {item.duration_ms ? `${item.duration_ms} ms` : '—'}
                    </TableCell>
                    <TableCell className="text-xs text-slate-500">
                      <div className="flex items-center gap-1">
                        <Clock className="h-3 w-3 text-slate-400" />
                        <span>{new Date(item.created_at).toLocaleString('ar-EG')}</span>
                      </div>
                    </TableCell>
                    <TableCell className="text-center">
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => onSelectReport(item.vin)}
                        className="text-xs font-bold text-[#00599C] hover:text-[#003478] hover:bg-blue-50 dark:hover:bg-blue-950/40 h-8 gap-1"
                      >
                        <ExternalLink className="h-3.5 w-3.5" />
                        عرض التقرير
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
