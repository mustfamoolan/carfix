import React from 'react'
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
import { ExternalLink, Loader2, FileText, RefreshCw } from 'lucide-react'

export default function RecentSearchesTable({
  searches = [],
  loading = false,
  onRefresh,
  onSelectReport,
}) {
  return (
    <Card className="border-border shadow-xs bg-card overflow-hidden">
      <CardHeader className="flex flex-row items-center justify-between py-3.5 px-5 border-b border-border">
        <CardTitle className="text-sm font-bold text-foreground">التقارير المسحوبة</CardTitle>
        {onRefresh && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onRefresh}
            className="text-xs h-8 gap-1.5 text-muted-foreground hover:text-foreground"
          >
            <RefreshCw className="h-3.5 w-3.5" />
            <span>تحديث</span>
          </Button>
        )}
      </CardHeader>

      <CardContent className="p-0">
        {loading ? (
          <div className="flex items-center justify-center py-12 text-muted-foreground gap-2 text-xs">
            <Loader2 className="h-4 w-4 animate-spin text-primary" />
            <span>جاري التحميل...</span>
          </div>
        ) : searches.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-center text-muted-foreground">
            <FileText className="h-8 w-8 stroke-1 mb-2 opacity-40" />
            <p className="text-xs">لا توجد تقارير مسحوبة حتى الآن</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/40 hover:bg-muted/40">
                  <TableHead className="text-right text-xs font-semibold">رقم الشاصي</TableHead>
                  <TableHead className="text-right text-xs font-semibold">المركبة</TableHead>
                  <TableHead className="text-right text-xs font-semibold">الحالة</TableHead>
                  <TableHead className="text-right text-xs font-semibold">التاريخ</TableHead>
                  <TableHead className="text-left text-xs font-semibold">التقرير</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {searches.map((item) => {
                  const isOk =
                    item.result_status === 'success' || item.status === 'success'
                  return (
                    <TableRow key={item.id}>
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
                        {isOk ? (
                          <Badge
                            variant="outline"
                            className="border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 text-[10px]"
                          >
                            مكتمل
                          </Badge>
                        ) : (
                          <Badge
                            variant="outline"
                            className="border-destructive/30 bg-destructive/10 text-destructive text-[10px]"
                          >
                            فشل
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {new Date(item.created_at).toLocaleString('ar-EG')}
                      </TableCell>
                      <TableCell className="text-left">
                        {isOk && (
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => onSelectReport(item.vin)}
                            className="text-xs h-7 px-2.5 gap-1 font-semibold text-primary border-primary/20 hover:bg-primary/5 cursor-pointer"
                          >
                            <span>فتح التقرير</span>
                            <ExternalLink className="h-3 w-3" />
                          </Button>
                        )}
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
