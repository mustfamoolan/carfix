import React, { useMemo } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  ShieldCheck,
  AlertTriangle,
  Gauge,
  Users,
  Wrench,
  MapPin,
  Car,
  DollarSign,
  FileText,
  Calendar,
  CheckCircle2,
  ExternalLink,
} from 'lucide-react'

const AR_DICT = {
  'No accidents or damage reported to CARFAX': 'لم يتم الإبلاغ عن أي حوادث أو أضرار لكارفاكس',
  'Severe Damage Reported': 'تم الإبلاغ عن أضرار شديدة',
  'Moderate Damage Reported': 'تم الإبلاغ عن أضرار متوسطة',
  'Minor Damage Reported': 'تم الإبلاغ عن أضرار طفيفة',
  'Vandalism Damage Reported': 'تم الإبلاغ عن أضرار تخريبية (Vandalism)',
  'Accident / Damage Reported': 'تم الإبلاغ عن حادث أو ضرر',
  'Personal Vehicle': 'استخدام شخصي',
  'Rental Vehicle': 'مركبة تأجير (Rental)',
  'Commercial Vehicle': 'استخدام تجاري',
  'Personal Lease': 'تأجير شخصي',
  'Year purchased': 'سنة الشراء',
  'Type of owner': 'نوع المالك',
  'Estimated length of ownership': 'المدة التقديرية للملكية',
  'Owned in the following states/provinces': 'الولايات / المقاطعات المسجل بها',
  'Estimated miles driven per year': 'المعدل السنوي للممشى',
  'Last reported odometer reading': 'آخر قراءة مسجلة للعداد',
  'Damage Brands': 'علامات الأضرار بالوثيقة (Title)',
  'Salvage | Junk | Rebuilt | Fire | Flood | Hail | Lemon':
    'سالفج | سكراب | معاد بناؤها | حريق | غرق | برد | عيب مصنعي',
  'Odometer Brands': 'علامات عداد المسافات',
  'Not Actual Mileage | Exceeds Mechanical Limits': 'عداد غير حقيقي | تجاوز الحدود الميكانيكية',
  'Guaranteed No Problem': '✓ مضمون - لا توجد مشكلة',
  'No Problem': '✓ لا توجد مشكلة',
  'Total Loss': 'خسارة كلية (شطب)',
  'No Total Loss Reported to CARFAX': '✓ لم تسجل خسارة كلية لدى كارفاكس',
  'Structural Damage': 'أضرار الشاصي والهيكل الإنشائي',
  'No Structural Damage Reported to CARFAX': '✓ لم تسجل أضرار هيكلية لدى كارفاكس',
  'Airbag Deployment': 'انفتاح الوسائد الهوائية (الإيرباج)',
  'No Airbag Deployment Reported to CARFAX': '✓ لم يسجل انفتاح للإيرباج لدى كارفاكس',
  'Odometer Check': 'فحص سلامة العداد',
  'No Indication of an Odometer Rollback': '✓ لا يوجد مؤشر تلاعب بالعداد',
  'Accident / Damage': 'الحوادث والأضرار',
  'Manufacturer Recall': 'استدعاءات المصنع (Recall)',
  'No Open Recalls Reported to CARFAX': '✓ لا توجد استدعاءات مفتوحة',
  'Basic Warranty': 'الضمان الأساسي للمصنع',
  'No Issues Reported': '✓ سليم - لا توجد مشاكل مسجلة',
  'Owner 1': 'المالك 1',
  'Owner 2': 'المالك 2',
  'Owner 3': 'المالك 3',
  'Owner 4': 'المالك 4',
}

function translateCell(text, lang) {
  if (!text) return ''
  if (lang !== 'ar') return text
  let out = String(text).trim()
  if (AR_DICT[out]) return AR_DICT[out]
  for (const [en, ar] of Object.entries(AR_DICT)) {
    if (out.includes(en)) {
      out = out.replaceAll(en, ar)
    }
  }
  out = out.replace(/Last Owned in\s+/i, 'آخر ملكية في ')
  out = out.replace(/(\d+)\s+Service History Records?/i, '$1 سجل صيانة مسجل')
  out = out.replace(/(\d+)\s+Detailed Records? Available/i, '$1 سجل تفصيلي متاح')
  out = out.replace(/\b(\d[\d,]*)\s*mi\b/gi, '$1 ميل')
  return out
}

// Fallback parser in case an older cached report doesn't have summary.tables populated yet
function parseFallbackFromHtml(rawHtml) {
  if (!rawHtml || typeof DOMParser === 'undefined') {
    return { specs: [], tables: [], detailedRecords: [], accidentEvents: [], carfaxValue: '' }
  }
  try {
    const cleanHtml = rawHtml.replace(
      /<script id="carfix-official-pdf"[\s\S]*?<\/script>/i,
      ''
    )
    const doc = new DOMParser().parseFromString(cleanHtml, 'text/html')
    const bodyText = doc.body ? doc.body.innerText || doc.body.textContent || '' : ''

    let carfaxValue = ''
    const valMatch =
      bodyText.match(
        /(?:CARFAX\s*VALUE|Retail\s*Value|History-Based\s*Value)[^\$\n]{0,40}(\$[0-9,]+)/i
      ) || bodyText.match(/(\$[0-9]{1,3}(?:,[0-9]{3})+)/)
    if (valMatch && valMatch[1]) {
      carfaxValue = valMatch[1].trim()
    }

    const specs = []
    const lines = bodyText
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean)
    for (const line of lines) {
      if (
        /^(Sedan|SUV|Coupe|Convertible|Hatchback|Pickup|Wagon|Van|Minivan|Truck|\d\.\dL\s|Gasoline|Diesel|Hybrid|Electric|Front wheel drive|Rear wheel drive|All wheel drive|4WD|AWD|FWD|RWD)/i.test(
          line
        ) &&
        line.length < 65
      ) {
        if (!specs.includes(line)) specs.push(line)
      }
    }

    const tables = []
    const detailedRecords = []
    const allTables = Array.from(doc.querySelectorAll('table'))
    for (const tbl of allTables) {
      const headers = Array.from(tbl.querySelectorAll('thead th, thead td, tr:first-child th'))
        .map((th) => (th.textContent || '').trim().replace(/\s+/g, ' '))
        .filter(Boolean)
      const rows = []
      const trs = Array.from(tbl.querySelectorAll('tbody tr, tr'))
      for (const tr of trs) {
        if (tr.querySelector('th') && rows.length === 0) continue
        const cells = Array.from(tr.querySelectorAll('td')).map((td) =>
          (td.textContent || '').trim().replace(/\s+/g, ' ')
        )
        if (cells.length === 0 || cells.every((c) => !c)) continue
        const firstCell = cells[0] || ''
        if (
          cells.length >= 3 &&
          (/^\d{2}\/\d{2}\/\d{4}$/.test(firstCell) ||
            /^\d{1,2}\/\d{1,2}\/\d{2,4}$/.test(firstCell) ||
            /^Not Reported$/i.test(firstCell))
        ) {
          detailedRecords.push({
            date: cells[0] || '',
            mileage: cells.length >= 4 ? cells[1] || '' : '',
            source: cells.length >= 4 ? cells[2] || '' : cells[1] || '',
            comments: cells.length >= 4 ? cells.slice(3).join(' | ') : cells[2] || '',
          })
        } else {
          rows.push(cells)
        }
      }
      if (rows.length > 0) {
        let sectionTitle = ''
        const prev = tbl.previousElementSibling
        if (prev && prev.textContent) {
          sectionTitle = prev.textContent.trim().split('\n')[0]
        }
        tables.push({
          title: sectionTitle || headers[0] || 'CARFAX Section',
          headers,
          rows,
        })
      }
    }

    return { specs, tables, detailedRecords, accidentEvents: [], carfaxValue }
  } catch {
    return { specs: [], tables: [], detailedRecords: [], accidentEvents: [], carfaxValue: '' }
  }
}

export default function ReportNativeView({
  report,
  summary,
  lang,
  onSwitchToOriginal,
  hasFullOriginal,
}) {
  const isAr = lang === 'ar'
  const vin = summary?.vin || report?.vin || ''
  const title = summary?.vehicle_title || report?.vehicle_title || `CARFAX Report (${vin})`
  const hasAccident = Number(summary?.accidents_reported || 0) > 0
  const cleanTitle = summary?.clean_title !== false

  const fallback = useMemo(() => {
    if (Array.isArray(summary?.tables) && summary.tables.length > 0) {
      return null
    }
    return parseFallbackFromHtml(report?.raw_html_en)
  }, [summary?.tables, report?.raw_html_en])

  const specs =
    Array.isArray(summary?.specs) && summary.specs.length > 0
      ? summary.specs
      : fallback?.specs || []
  const tables =
    Array.isArray(summary?.tables) && summary.tables.length > 0
      ? summary.tables
      : fallback?.tables || []
  const detailedRecords =
    Array.isArray(summary?.detailed_records) && summary.detailed_records.length > 0
      ? summary.detailed_records
      : fallback?.detailedRecords || []
  const accidentEvents =
    Array.isArray(summary?.accident_events) && summary.accident_events.length > 0
      ? summary.accident_events
      : fallback?.accidentEvents || []
  const carfaxValue = summary?.carfax_value || fallback?.carfaxValue || ''

  return (
    <div className="space-y-4 w-full" dir={isAr ? 'rtl' : 'ltr'}>
      {/* 1. Hero Vehicle Identification & Value Banner */}
      <Card className="border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden bg-white dark:bg-slate-900 rounded-xl">
        <div className="bg-gradient-to-l from-[#003478] to-[#00599C] px-5 py-4 text-white flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div className="space-y-1">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="bg-white text-[#003478] font-black tracking-widest px-2.5 py-0.5 rounded text-xs">
                C A R F A X
              </span>
              <span className="text-xs text-blue-100 font-semibold">
                {isAr ? 'ملخص الفحص الذكي السريع' : 'Fast Structured Vehicle History Report'}
              </span>
              {cleanTitle ? (
                <Badge className="bg-emerald-500/20 text-emerald-100 border border-emerald-400/40 text-[11px]">
                  {isAr ? '✓ وثيقة ملكية سليمة (Clean Title)' : '✓ Guaranteed Clean Title'}
                </Badge>
              ) : (
                <Badge className="bg-red-500/30 text-red-100 border border-red-400/40 text-[11px]">
                  {isAr ? '⚠️ تنبيه في وثيقة الملكية' : '⚠️ Title Brand Alert'}
                </Badge>
              )}
            </div>
            <h2 className="text-xl sm:text-2xl font-extrabold tracking-tight">{title}</h2>
            <p className="text-xs sm:text-sm font-mono text-blue-100">
              VIN: <span className="font-bold text-white select-all">{vin}</span>
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            {carfaxValue && (
              <div className="bg-white/10 backdrop-blur-xs border border-white/20 rounded-lg px-4 py-2 text-center">
                <div className="text-[11px] text-blue-100 font-medium flex items-center justify-center gap-1">
                  <DollarSign className="h-3.5 w-3.5" />
                  <span>{isAr ? 'قيمة كارفاكس التقديرية' : 'CARFAX Value'}</span>
                </div>
                <div className="text-lg font-black text-emerald-300 font-mono">{carfaxValue}</div>
              </div>
            )}

            <button
              type="button"
              onClick={onSwitchToOriginal}
              className="inline-flex items-center gap-2 bg-white text-[#003478] hover:bg-blue-50 font-bold text-xs px-4 py-2.5 rounded-lg shadow-xs transition-all cursor-pointer"
            >
              <ExternalLink className="h-4 w-4" />
              <span>
                {hasFullOriginal
                  ? isAr
                    ? 'فتح صفحة كارفاكس الأصلية الكاملة'
                    : 'Open Full Original Carfax Page'
                  : isAr
                    ? 'معاينة صفحة كارفاكس (جاري استكمال التصميم بصمت...)'
                    : 'View Carfax Page (Completing assets silently...)'}
              </span>
            </button>
          </div>
        </div>

        {/* Vehicle Specifications Bar */}
        {specs.length > 0 && (
          <div className="px-5 py-3 bg-slate-50 dark:bg-slate-800/60 border-t border-slate-200 dark:border-slate-800 flex flex-wrap items-center gap-2">
            <span className="text-xs font-bold text-slate-500 flex items-center gap-1">
              <Car className="h-3.5 w-3.5 text-[#003478]" />
              {isAr ? 'مواصفات المركبة:' : 'Vehicle Specs:'}
            </span>
            {specs.map((sp, i) => (
              <Badge
                key={i}
                variant="outline"
                className="bg-white dark:bg-slate-900 text-slate-700 dark:text-slate-200 border-slate-300 dark:border-slate-700 text-xs font-semibold"
              >
                {translateCell(sp, lang)}
              </Badge>
            ))}
          </div>
        )}
      </Card>

      {/* 2. Key Highlights Grid (6 Core Carfax Indicators) */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        {/* Accident / Damage Card */}
        <Card
          className={`p-4 border shadow-xs rounded-xl ${
            hasAccident
              ? 'bg-red-50/60 dark:bg-red-950/20 border-red-200 dark:border-red-900'
              : 'bg-emerald-50/60 dark:bg-emerald-950/20 border-emerald-200 dark:border-emerald-900'
          }`}
        >
          <div className="flex items-start gap-3">
            <div
              className={`p-2.5 rounded-lg ${
                hasAccident ? 'bg-red-100 text-red-700' : 'bg-emerald-100 text-emerald-700'
              }`}
            >
              {hasAccident ? (
                <AlertTriangle className="h-5 w-5" />
              ) : (
                <ShieldCheck className="h-5 w-5" />
              )}
            </div>
            <div>
              <p className="text-xs font-bold text-slate-500">
                {isAr ? 'حالة الحوادث والأضرار' : 'Accident & Damage Status'}
              </p>
              <p
                className={`text-sm font-extrabold mt-0.5 ${
                  hasAccident ? 'text-red-700 dark:text-red-400' : 'text-emerald-700 dark:text-emerald-400'
                }`}
              >
                {translateCell(
                  summary?.accident_summary ||
                    (hasAccident
                      ? 'Accident / Damage Reported'
                      : 'No accidents or damage reported to CARFAX'),
                  lang
                )}
              </p>
            </div>
          </div>
        </Card>

        {/* Odometer Card */}
        <Card className="p-4 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-xs rounded-xl">
          <div className="flex items-start gap-3">
            <div className="p-2.5 rounded-lg bg-blue-50 text-[#003478]">
              <Gauge className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs font-bold text-slate-500">
                {isAr ? 'آخر قراءة لعداد المسافات' : 'Last Reported Odometer'}
              </p>
              <p className="text-base font-extrabold text-slate-900 dark:text-slate-100 font-mono mt-0.5">
                {translateCell(summary?.last_reported_odometer || 'غير مسجل', lang)}
              </p>
            </div>
          </div>
        </Card>

        {/* Owners Card */}
        <Card className="p-4 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-xs rounded-xl">
          <div className="flex items-start gap-3">
            <div className="p-2.5 rounded-lg bg-blue-50 text-[#003478]">
              <Users className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs font-bold text-slate-500">
                {isAr ? 'عدد الملاك ونوع الاستخدام' : 'Ownership & Vehicle Use'}
              </p>
              <p className="text-sm font-extrabold text-slate-900 dark:text-slate-100 mt-0.5">
                {summary?.total_owners || 1}{' '}
                {isAr
                  ? summary?.total_owners === 1
                    ? 'مالك واحد'
                    : 'ملاك'
                  : summary?.total_owners === 1
                    ? 'Owner'
                    : 'Owners'}{' '}
                • {translateCell(summary?.vehicle_use || 'Personal Vehicle', lang)}
              </p>
            </div>
          </div>
        </Card>

        {/* Service History Card */}
        <Card className="p-4 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-xs rounded-xl">
          <div className="flex items-start gap-3">
            <div className="p-2.5 rounded-lg bg-blue-50 text-[#003478]">
              <Wrench className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs font-bold text-slate-500">
                {isAr ? 'سجلات الصيانة والفحص' : 'Service History Records'}
              </p>
              <p className="text-sm font-extrabold text-slate-900 dark:text-slate-100 mt-0.5">
                {translateCell(
                  summary?.service_records_count ||
                    `${detailedRecords.length || 'عدة'} سجلات متاحة`,
                  lang
                )}
              </p>
            </div>
          </div>
        </Card>

        {/* Last Location Card */}
        <Card className="p-4 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-xs rounded-xl">
          <div className="flex items-start gap-3">
            <div className="p-2.5 rounded-lg bg-blue-50 text-[#003478]">
              <MapPin className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs font-bold text-slate-500">
                {isAr ? 'موقع الملكية المسجل' : 'Last Owned Location'}
              </p>
              <p className="text-sm font-extrabold text-slate-900 dark:text-slate-100 mt-0.5">
                {translateCell(summary?.last_owned_location || 'North America (US / Canada)', lang)}
              </p>
            </div>
          </div>
        </Card>

        {/* Detailed Records Count Card */}
        <Card className="p-4 bg-white dark:bg-slate-900 border-slate-200 dark:border-slate-800 shadow-xs rounded-xl">
          <div className="flex items-start gap-3">
            <div className="p-2.5 rounded-lg bg-blue-50 text-[#003478]">
              <FileText className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs font-bold text-slate-500">
                {isAr ? 'إجمالي السجلات التفصيلية' : 'Detailed History Records'}
              </p>
              <p className="text-sm font-extrabold text-slate-900 dark:text-slate-100 mt-0.5">
                {translateCell(
                  summary?.detailed_records_count ||
                    `${detailedRecords.length} Detailed Records Available`,
                  lang
                )}
              </p>
            </div>
          </div>
        </Card>
      </div>

      {/* 3. Accident Events Alert Box (if any) */}
      {accidentEvents.length > 0 && (
        <Card className="border-red-300 dark:border-red-900 bg-red-50/40 dark:bg-red-950/20 shadow-xs rounded-xl overflow-hidden">
          <div className="bg-red-600 text-white px-4 py-2.5 text-sm font-bold flex items-center gap-2">
            <AlertTriangle className="h-4 w-4" />
            <span>
              {isAr ? 'تفاصيل الحوادث والأضرار المسجلة' : 'Reported Accident / Damage Events'}
            </span>
          </div>
          <CardContent className="p-4 space-y-2">
            {accidentEvents.map((ev, idx) => (
              <div
                key={idx}
                className="p-3 bg-white dark:bg-slate-900 rounded-lg border border-red-200 dark:border-red-900 text-xs sm:text-sm font-semibold text-slate-800 dark:text-slate-200"
              >
                {translateCell(ev, lang)}
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      {/* 4. Extracted Carfax Structured Tables (Ownership, Title Guarantee, Additional History) */}
      {tables.map((tbl, tIdx) => (
        <Card
          key={tIdx}
          className="border-slate-200 dark:border-slate-800 shadow-xs bg-white dark:bg-slate-900 rounded-xl overflow-hidden"
        >
          <div className="bg-[#003478] text-white px-4 py-2.5 text-xs sm:text-sm font-bold flex items-center justify-between">
            <span>{translateCell(tbl.title, lang)}</span>
            <CheckCircle2 className="h-4 w-4 text-blue-200" />
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-xs sm:text-sm border-collapse">
              {Array.isArray(tbl.headers) && tbl.headers.length > 0 && (
                <thead>
                  <tr className="bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-200 border-b border-slate-200 dark:border-slate-700">
                    {tbl.headers.map((h, hIdx) => (
                      <th key={hIdx} className="py-2.5 px-4 font-bold text-start">
                        {translateCell(h, lang)}
                      </th>
                    ))}
                  </tr>
                </thead>
              )}
              <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                {Array.isArray(tbl.rows) &&
                  tbl.rows.map((row, rIdx) => (
                    <tr
                      key={rIdx}
                      className="hover:bg-slate-50/80 dark:hover:bg-slate-800/40 transition-colors"
                    >
                      {Array.isArray(row) &&
                        row.map((cell, cIdx) => (
                          <td
                            key={cIdx}
                            className={`py-2.5 px-4 ${
                              cIdx === 0
                                ? 'font-bold text-slate-700 dark:text-slate-200 bg-slate-50/50 dark:bg-slate-800/20 w-1/3'
                                : 'text-slate-800 dark:text-slate-100'
                            }`}
                          >
                            {translateCell(cell, lang)}
                          </td>
                        ))}
                    </tr>
                  ))}
              </tbody>
            </table>
          </div>
        </Card>
      ))}

      {/* 5. Chronological Detailed History Timeline */}
      {detailedRecords.length > 0 && (
        <Card className="border-slate-200 dark:border-slate-800 shadow-xs bg-white dark:bg-slate-900 rounded-xl overflow-hidden">
          <div className="bg-[#00599C] text-white px-4 py-3 text-sm font-bold flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Calendar className="h-4 w-4" />
              <span>
                {isAr
                  ? `السجل التاريخي الزمني المفصل (${detailedRecords.length} سجل)`
                  : `Detailed Chronological History (${detailedRecords.length} Records)`}
              </span>
            </div>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-xs sm:text-sm border-collapse">
              <thead>
                <tr className="bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-200 border-b border-slate-200 dark:border-slate-700">
                  <th className="py-2.5 px-4 font-bold text-start w-28">
                    {isAr ? 'التاريخ' : 'Date'}
                  </th>
                  <th className="py-2.5 px-4 font-bold text-start w-32">
                    {isAr ? 'قراءة العداد' : 'Mileage'}
                  </th>
                  <th className="py-2.5 px-4 font-bold text-start w-48">
                    {isAr ? 'المصدر / الجهة' : 'Source'}
                  </th>
                  <th className="py-2.5 px-4 font-bold text-start">
                    {isAr ? 'التفاصيل والملاحظات' : 'Comments & Service Details'}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                {detailedRecords.map((rec, idx) => (
                  <tr
                    key={idx}
                    className="hover:bg-blue-50/30 dark:hover:bg-slate-800/40 transition-colors"
                  >
                    <td className="py-2.5 px-4 font-mono font-bold text-[#003478] dark:text-blue-300 whitespace-nowrap">
                      {rec.date || '—'}
                    </td>
                    <td className="py-2.5 px-4 font-mono font-semibold text-slate-800 dark:text-slate-200 whitespace-nowrap">
                      {translateCell(rec.mileage || '—', lang)}
                    </td>
                    <td className="py-2.5 px-4 font-semibold text-slate-700 dark:text-slate-300">
                      {translateCell(rec.source || '—', lang)}
                    </td>
                    <td className="py-2.5 px-4 text-slate-800 dark:text-slate-100 leading-relaxed">
                      {translateCell(rec.comments || '', lang)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}
    </div>
  )
}
