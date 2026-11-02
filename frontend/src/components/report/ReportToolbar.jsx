import React from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { ArrowRight, Globe, Printer } from 'lucide-react'

export default function ReportToolbar({
  report,
  lang,
  onLangChange,
  onBack,
  onPrint,
}) {
  return (
    <Card className="shadow-xs w-full print:hidden">
      <CardContent className="p-3.5 sm:px-5">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          {/* Back button & Vehicle Title */}
          <div className="flex items-center gap-3">
            <Button
              variant="outline"
              size="sm"
              onClick={onBack}
              className="gap-1.5 text-xs font-medium h-8 cursor-pointer"
            >
              <ArrowRight className="h-3.5 w-3.5" />
              <span>العودة للبحث</span>
            </Button>
            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-sm font-bold text-foreground">
                {report?.vehicle_title || 'CARFAX Report'}
              </span>
              <span className="text-xs font-mono text-muted-foreground">
                ({report?.vin})
              </span>
            </div>
          </div>

          {/* Language Switch & Instant Print */}
          <div className="flex flex-wrap items-center gap-2">
            <div className="flex items-center bg-muted p-0.5 rounded-md border text-xs">
              <button
                type="button"
                onClick={() => onLangChange('en')}
                className={`px-2.5 py-1 rounded transition-colors font-semibold cursor-pointer ${
                  lang === 'en'
                    ? 'bg-primary text-primary-foreground shadow-xs'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                English
              </button>
              <button
                type="button"
                onClick={() => onLangChange('ar')}
                className={`px-2.5 py-1 rounded transition-colors font-semibold flex items-center gap-1 cursor-pointer ${
                  lang === 'ar'
                    ? 'bg-primary text-primary-foreground shadow-xs'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                <Globe className="h-3 w-3" />
                <span>العربية</span>
              </button>
            </div>

            <Button
              size="sm"
              variant="default"
              onClick={onPrint}
              className="gap-1.5 text-xs font-semibold h-8 cursor-pointer"
            >
              <Printer className="h-3.5 w-3.5" />
              <span>طباعة التقرير</span>
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
