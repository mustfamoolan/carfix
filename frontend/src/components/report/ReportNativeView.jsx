import React, { useMemo, useState } from 'react'

// Dictionary for Arabic mode when user toggles to Arabic
const AR_DICT = {
  'Additional History': 'السجل الإضافي والفحص الفني',
  'Not all accidents / issues are reported to CARFAX': 'لا يتم الإبلاغ عن جميع الحوادث أو المشاكل إلى كارفاكس',
  'Title History': 'تاريخ وثيقة الملكية (Title)',
  'CARFAX guarantees the information in this section': 'تضمن كارفاكس صحة المعلومات الواردة في هذا القسم',
  'Ownership History': 'تاريخ الملكية وعدد الملاك',
  'The number of owners is estimated': 'عدد الملاك تقديري بناءً على سجلات التسجيل',
  'Detailed History': 'السجل التفصيلي الزمني للمركبة',
  'Accident / Damage History': 'سجل الحوادث والأضرار',
  'Total Loss': 'خسارة كلية (شطب)',
  'No total loss reported to CARFAX.': 'لم يتم تسجيل خسارة كلية لدى كارفاكس.',
  'Structural Damage': 'أضرار هيكلية (الشاصي)',
  'CARFAX recommends that you have this vehicle inspected by a collision repair specialist.': 'توصي كارفاكس بفحص المركبة لدى مختص هياكل.',
  'Airbag Deployment': 'انفتاح الوسائد الهوائية (Airbag)',
  'No airbag deployment reported to CARFAX.': 'لم يتم تسجيل انفتاح للوسائد الهوائية لدى كارفاكس.',
  'Odometer Check': 'فحص عداد المسافات',
  'No indication of an odometer rollback.': 'لا توجد مؤشرات على التلاعب بعداد المسافات.',
  'Accident / Damage': 'الحوادث / الأضرار',
  'No accidents or damage reported to CARFAX.': 'لم يتم الإبلاغ عن أي حوادث أو أضرار لدى كارفاكس.',
  'Manufacturer Recall': 'استدعاءات المصنع (Recalls)',
  'Basic Warranty': 'ضمان المصنع الأساسي',
  'Damage Brands': 'حالة وثيقة الأضرار (Damage Brands)',
  'Salvage | Junk | Rebuilt | Fire | Flood | Hail | Lemon': 'سالفج (غرق/حريق/حطام/معاد بناؤه/خلل مصنعي)',
  'Odometer Brands': 'حالة وثيقة العداد (Odometer Brands)',
  'Not Actual Mileage | Exceeds Mechanical Limits': 'قراءة غير حقيقية | تجاوز الحدود الميكانيكية',
  'Year purchased': 'سنة الشراء',
  'Type of owner': 'نوع الملكية / الاستخدام',
  'Estimated length of ownership': 'مدة الملكية التقديرية',
  'Owned in the following states/provinces': 'الولايات / المقاطعات المسجلة بها',
  'Estimated miles driven per year': 'معدل المشي السنوي التقديري',
  'Last reported odometer reading': 'آخر قراءة مسجلة للعداد',
  'No Issues Reported': 'لا توجد مشاكل مسجلة',
  'No Issues Indicated': 'سليم - لا توجد ملاحظات',
  'No Recalls Reported': 'لا توجد استدعاءات نشطة',
  'Warranty Active': 'الضمان ساري المفعول',
  'Warranty Expired': 'انتهت مدة الضمان',
  'Guaranteed': 'مضمون 100%',
  'No Problem': 'لا توجد أي مشكلة',
  'Damage Reported': 'يوجد ضرر مسجل',
  'Accident Reported': 'يوجد حادث مسجل',
  'Personal': 'شخصي',
  'Personal vehicle': 'استخدام شخصي',
  'Rental': 'شركة تأجير (Rental)',
  'Rental Vehicle': 'مركبة تأجير',
  'Personal Vehicle': 'مركبة استخدام شخصي',
  'Commercial': 'تجاري',
  'Lease': 'تأجير تمويلي (Lease)',
  'Not Reported': 'غير مسجل',
}

function trText(text, lang) {
  if (!text) return ''
  const str = String(text).trim()
  if (lang !== 'ar') return str
  if (AR_DICT[str]) return AR_DICT[str]
  let out = str
  for (const [en, ar] of Object.entries(AR_DICT)) {
    if (out.includes(en)) {
      out = out.replaceAll(en, ar)
    }
  }
  return out
}

export default function ReportNativeView({
  report,
  summary,
  lang = 'en',
  onLangChange,
  onPrint,
}) {
  const [showSpecs, setShowSpecs] = useState(true)
  const [priceMode, setPriceMode] = useState('Retail')

  const isAr = lang === 'ar'
  const vin = report?.vin || summary?.vin || ''
  const vehicleTitle =
    summary?.vehicle_title || report?.vehicle_title || `CARFAX Vehicle Report`
  const odometer = summary?.last_reported_odometer || 'Not Reported'
  const owners = summary?.total_owners ?? 1
  const accidents = summary?.accidents_reported ?? 0
  const accidentSummary =
    summary?.accident_summary ||
    (accidents > 0
      ? 'Accident / Damage Reported'
      : 'No accidents or damage reported to CARFAX')
  const damageSeverity = summary?.damage_severity || (accidents > 0 ? 'Moderate' : 'None')
  const carfaxValue = summary?.carfax_value || ''
  const serviceRecordsText =
    summary?.service_records_count || 'Service History Records'
  const vehicleUse = summary?.vehicle_use || 'Personal Vehicle'
  const lastLocation = summary?.last_owned_location || 'United States'
  const detailedCountText =
    summary?.detailed_records_count ||
    `${summary?.detailed_records?.length || 0} Detailed Records Available`
  const specs = Array.isArray(summary?.specs) && summary.specs.length > 0
    ? summary.specs
    : ['Sedan / SUV 4 DR', 'Gasoline', 'Automatic']

  // Categorize tables into Additional History, Title History, Ownership History
  const { additionalTable, titleTable, ownershipTable, otherTables } = useMemo(() => {
    const rawTables = Array.isArray(summary?.tables) ? summary.tables : []
    let additional = null
    let titleTbl = null
    let ownership = null
    const others = []

    for (const t of rawTables) {
      const h0 = (t.title || t.headers?.[0] || '').toLowerCase()
      const firstRowText = (t.rows?.[0]?.[0] || '').toLowerCase()
      if (h0.includes('additional history') || firstRowText.includes('total loss')) {
        additional = t
      } else if (h0.includes('title history') || firstRowText.includes('damage brands') || firstRowText.includes('salvage')) {
        titleTbl = t
      } else if (h0.includes('ownership history') || firstRowText.includes('year purchased')) {
        ownership = t
      } else {
        others.push(t)
      }
    }

    return {
      additionalTable: additional,
      titleTable: titleTbl,
      ownershipTable: ownership,
      otherTables: others,
    }
  }, [summary?.tables])

  const detailedRecords = useMemo(() => {
    return Array.isArray(summary?.detailed_records) ? summary.detailed_records : []
  }, [summary?.detailed_records])

  const accidentEvents = useMemo(() => {
    return Array.isArray(summary?.accident_events) ? summary.accident_events : []
  }, [summary?.accident_events])

  // Split row heading cell into main bold title + subtitle description
  const splitRowHeading = (rawCell) => {
    if (!rawCell) return { main: '', sub: '' }
    const lines = String(rawCell)
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean)
    if (lines.length >= 2) {
      return { main: lines[0], sub: lines.slice(1).join(' ') }
    }
    const text = lines[0] || ''
    const knownPrefixes = [
      'Total Loss',
      'Structural Damage',
      'Airbag Deployment',
      'Odometer Check',
      'Accident / Damage',
      'Manufacturer Recall',
      'Basic Warranty',
      'Damage Brands',
      'Odometer Brands',
    ]
    for (const kp of knownPrefixes) {
      if (text.startsWith(kp) && text.length > kp.length) {
        return {
          main: kp,
          sub: text.slice(kp.length).trim(),
        }
      }
    }
    return { main: text, sub: '' }
  }

  return (
    <div
      id="carfax-official-clone"
      dir="ltr"
      className="w-full bg-[#f5f7f8] text-[#212529] rounded-lg overflow-hidden border border-[#d6dce2] shadow-sm font-sans select-text"
      style={{ fontFamily: "'Roboto', 'Cairo', 'Helvetica Neue', Arial, sans-serif" }}
    >
      {/* 1. OFFICIAL TOP CARFAX BLUE BAR (#0076be) */}
      <header className="bg-[#0076be] h-[54px] px-4 md:px-8 flex items-center justify-between border-b border-[#005a92] print:hidden">
        <div className="w-24 hidden sm:block" />
        <div className="flex items-center justify-center">
          <img
            src="/carfax-assets/carfax-logo.svg"
            alt="CARFAX"
            className="h-[26px] w-auto object-contain"
          />
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={onPrint}
            className="bg-white hover:bg-slate-50 text-[#0076be] font-bold text-[13px] px-3.5 py-1.5 rounded flex items-center gap-1.5 shadow-sm transition-colors"
          >
            <svg className="w-4 h-4 fill-current" viewBox="0 0 24 24">
              <path d="M19 8H5c-1.66 0-3 1.34-3 3v6h4v4h12v-4h4v-6c0-1.66-1.34-3-3-3zm-3 11H8v-5h8v5zm3-7c-.55 0-1-.45-1-1s.45-1 1-1 1 .45 1 1-.45 1-1 1zm-1-9H6v4h12V3z" />
            </svg>
            <span>Print</span>
          </button>
        </div>
      </header>

      {/* 2. TWO-COLUMN CARFAX VHR CONTAINER (Left Dealer Sidebar + Main Report) */}
      <div className="max-w-[1220px] mx-auto flex flex-col lg:flex-row items-start">
        {/* LEFT SIDEBAR: Vehicle Info + CARFAX Advantage Dealer Card */}
        <aside className="w-full lg:w-[235px] shrink-0 p-4 lg:pt-6 lg:px-4 border-b lg:border-b-0 lg:border-r border-[#e1e5ea] bg-[#f5f7f8] print:hidden">
          <div className="mb-6">
            <p className="text-[14px] font-bold text-[#212529] uppercase leading-snug">
              {vehicleTitle}
            </p>
            <p className="text-[12.5px] text-[#495057] mt-0.5 font-mono">
              <span className="font-semibold">VIN:</span> {vin}
            </p>
          </div>

          {/* Advantage Dealer White Box with Overlapping Badge */}
          <div className="relative bg-white border border-[#d0d7de] rounded-[4px] pt-12 pb-4 px-3.5 text-center mt-10 shadow-[0_1px_3px_rgba(0,0,0,0.04)]">
            <img
              src="/carfax-assets/advantage-dealer.png"
              alt="CARFAX Advantage Dealer"
              className="w-[116px] h-auto mx-auto -mt-[74px] mb-2 select-none"
            />
            <span
              className="absolute top-2.5 right-2.5 text-[#0076be] cursor-pointer"
              title="CARFAX Advantage Dealer"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
                <circle cx="12" cy="12" r="10" />
                <path d="M12 16v-4M12 8h.01" />
              </svg>
            </span>
            <p className="text-[11px] text-[#6c757d] leading-tight mt-1">
              This report provided free of charge by:
            </p>
            <p className="text-[13px] font-bold text-[#212529] mt-2 leading-snug">
              Discount Motor Sales Llc
            </p>
            <div className="text-[12px] text-[#495057] mt-1.5 leading-snug">
              <div>601 S Broadway</div>
              <div>Wichita, KS 67202</div>
            </div>
            <p className="text-[12.5px] font-bold text-[#212529] mt-2">
              316-409-4354
            </p>
          </div>
        </aside>

        {/* MAIN REPORT BODY */}
        <main className="flex-1 min-w-0 p-3 sm:p-5 lg:p-6 w-full">
          {/* Top Hat Row: Display CARFAX toggle + English/Arabic switch */}
          <div className="flex flex-wrap items-center justify-between gap-3 mb-3 px-1 print:hidden">
            <label className="inline-flex items-center gap-2.5 text-[13px] text-[#495057] cursor-pointer select-none">
              <span>Display CARFAX in online listings:</span>
              <span className="w-9 h-5 bg-[#6c757d] rounded-full relative inline-flex items-center px-0.5">
                <span className="w-4 h-4 bg-white rounded-full shadow-sm" />
              </span>
            </label>

            <div className="inline-flex rounded border border-[#0076be] overflow-hidden text-[12px] font-semibold">
              <button
                type="button"
                onClick={() => onLangChange && onLangChange('en')}
                className={`px-3 py-1 transition-colors ${
                  !isAr
                    ? 'bg-[#0076be] text-white'
                    : 'bg-white text-[#0076be] hover:bg-blue-50'
                }`}
              >
                English
              </button>
              <button
                type="button"
                onClick={() => onLangChange && onLangChange('ar')}
                className={`px-3 py-1 transition-colors ${
                  isAr
                    ? 'bg-[#0076be] text-white'
                    : 'bg-white text-[#0076be] hover:bg-blue-50'
                }`}
              >
                العربية
              </button>
            </div>
          </div>

          {/* SUMMARY SECTION CARD */}
          <div className="bg-white border border-[#d0d7de] rounded-[6px] overflow-hidden shadow-[0_1px_3px_rgba(0,0,0,0.05)]">
            {/* Blue Header Bar: [C][A][R][F][A][X] Report + US $49.99 */}
            <div className="bg-[#0076be] px-4 py-2.5 flex items-center justify-between text-white">
              <div className="flex items-center gap-2.5">
                <img
                  src="/carfax-assets/carfax-logo.svg"
                  alt="CARFAX"
                  className="h-[24px] w-auto"
                />
                <span className="text-[20px] font-bold tracking-tight">
                  {isAr ? 'تقرير المركبة' : 'Report'}
                </span>
              </div>
              <div className="bg-[#005a92] text-white text-[12px] font-bold px-3 py-1 rounded-full border border-white/15">
                US $49.99
              </div>
            </div>

            {/* Vehicle Title, Mileage, VIN, Window Sticker & Specs */}
            <div className="px-5 py-4 border-b border-[#e9ecef] bg-white">
              <h1 className="text-[22px] font-bold text-[#212529] leading-tight">
                {vehicleTitle}
              </h1>

              <div className="flex flex-wrap items-center gap-y-1 text-[13.5px] text-[#212529] mt-2">
                {odometer && odometer !== 'غير مسجل' && (
                  <>
                    <span className="font-bold">{odometer}</span>
                    <span className="mx-2.5 text-[#ced4da]">|</span>
                  </>
                )}
                <span>
                  <strong className="font-bold">VIN:</strong>{' '}
                  <span className="font-mono">{vin}</span>
                </span>
                <span className="mx-2.5 text-[#ced4da] print:hidden">|</span>
                <a
                  href="#detailed-history-section"
                  className="text-[#0076be] hover:underline font-medium print:hidden"
                >
                  Original Window Sticker
                </a>
                <span className="mx-2.5 text-[#ced4da] print:hidden">|</span>
                <button
                  type="button"
                  onClick={() => setShowSpecs((s) => !s)}
                  className="text-[#0076be] hover:underline font-medium inline-flex items-center gap-1 print:hidden"
                >
                  <span>Vehicle Details</span>
                  <svg
                    className={`w-3.5 h-3.5 transition-transform ${
                      showSpecs ? 'rotate-180' : ''
                    }`}
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2.5"
                    viewBox="0 0 24 24"
                  >
                    <path d="M19 9l-7 7-7-7" />
                  </svg>
                </button>
              </div>

              {showSpecs && specs.length > 0 && (
                <div className="flex flex-wrap items-center text-[12.5px] text-[#495057] mt-2 pt-2 border-t border-dashed border-[#e9ecef]">
                  {specs.map((sp, i) => (
                    <React.Fragment key={i}>
                      <span>{sp}</span>
                      {i < specs.length - 1 && (
                        <span className="mx-2.5 text-[#ced4da]">|</span>
                      )}
                    </React.Fragment>
                  ))}
                </div>
              )}
            </div>

            {/* SHOWCASE GRID: Left (Car Fox + Cards) | Right (5 Highlights Rows) */}
            <div className="grid grid-cols-1 md:grid-cols-12 bg-[#f5f7f8]">
              {/* Left 7 Columns: Car Fox + Floating Cards */}
              <div className="md:col-span-7 p-4 sm:p-5 flex flex-col sm:flex-row items-center sm:items-start gap-4 border-b md:border-b-0 md:border-r border-[#e1e5ea]">
                {/* Official Car Fox Mascot */}
                <div className="w-[138px] sm:w-[152px] shrink-0 flex flex-col items-center relative select-none">
                  <div className="bg-white border border-[#bcd0e5] text-[#00599c] text-[10px] font-extrabold uppercase tracking-tight text-center px-2 py-1 rounded shadow-sm mb-1 leading-tight">
                    CARFAX HAS THE MOST ACCIDENT &amp; DAMAGE INFORMATION
                  </div>
                  <img
                    src="/carfax-assets/carfox-main.avif"
                    alt="CAR FOX"
                    className="w-[136px] h-auto object-contain drop-shadow-sm"
                  />
                </div>

                {/* Stacked Showcase Cards */}
                <div className="flex-1 w-full space-y-3 my-auto">
                  {/* Card 1: Accident / Damage Card */}
                  <a
                    href="#accident-damage-section"
                    className="block bg-white rounded-[8px] border border-[#dce1e7] p-3.5 shadow-[0_2px_6px_rgba(0,0,0,0.06)] hover:border-[#0076be] transition-colors"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <span
                        className={`text-[10.5px] font-extrabold uppercase px-2 py-0.5 rounded tracking-wide text-white ${
                          accidents > 0 ? 'bg-[#00599c]' : 'bg-[#1c8237]'
                        }`}
                      >
                        {accidents > 0
                          ? trText(accidentSummary.replace(/Reported/i, '').trim(), lang)
                          : isAr
                          ? 'سجل الحوادث'
                          : 'NO ACCIDENTS'}
                      </span>
                      <svg className="w-4 h-4 text-[#0076be]" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                        <path d="M9 5l7 7-7 7" />
                      </svg>
                    </div>

                    <div className="flex items-center gap-2.5 text-[13.5px] font-bold text-[#212529]">
                      {accidents > 0 ? (
                        <span className="w-5 h-5 rounded-full bg-[#d93025] text-white flex items-center justify-center text-xs font-black shrink-0">
                          !
                        </span>
                      ) : (
                        <span className="w-5 h-5 rounded-full bg-[#1c8237] text-white flex items-center justify-center text-xs font-black shrink-0">
                          ✓
                        </span>
                      )}
                      <span>{trText(accidentSummary, lang)}</span>
                    </div>

                    {/* Damage Severity Scale Bar when Minor/Moderate/Severe */}
                    {accidents > 0 &&
                      ['Minor', 'Moderate', 'Severe'].includes(damageSeverity) && (
                        <div className="mt-3 pt-2 border-t border-[#f1f3f5]">
                          <div className="grid grid-cols-3 gap-1 h-2.5 bg-[#e9ecef] rounded overflow-hidden p-0.5">
                            <div
                              className={`rounded-sm ${
                                damageSeverity === 'Minor'
                                  ? 'bg-[#0076be]'
                                  : 'bg-transparent'
                              }`}
                            />
                            <div
                              className={`rounded-sm ${
                                damageSeverity === 'Moderate'
                                  ? 'bg-[#0076be]'
                                  : 'bg-transparent'
                              }`}
                            />
                            <div
                              className={`rounded-sm ${
                                damageSeverity === 'Severe'
                                  ? 'bg-[#0076be]'
                                  : 'bg-transparent'
                              }`}
                            />
                          </div>
                          <div className="flex justify-between text-[10.5px] text-[#6c757d] font-medium mt-1">
                            <span>Minor</span>
                            <span>Moderate</span>
                            <span>Severe</span>
                          </div>
                        </div>
                      )}
                  </a>

                  {/* Card 2: CARFAX Value Card (if available) */}
                  {carfaxValue && (
                    <div className="bg-white rounded-[8px] border border-[#dce1e7] p-3.5 shadow-[0_2px_6px_rgba(0,0,0,0.06)]">
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center gap-2">
                          <span className="bg-[#1b7734] text-white text-[10.5px] font-extrabold uppercase px-2 py-0.5 rounded tracking-wide">
                            CARFAX VALUE
                          </span>
                          <span className="text-[15px] font-extrabold text-[#212529]">
                            {carfaxValue}
                          </span>
                        </div>
                        <svg className="w-4 h-4 text-[#0076be]" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                          <path d="M9 5l7 7-7 7" />
                        </svg>
                      </div>

                      <div className="flex items-center gap-5 text-[12.5px] text-[#212529] my-2">
                        <label className="inline-flex items-center gap-1.5 cursor-pointer">
                          <input
                            type="radio"
                            checked={priceMode === 'Retail'}
                            onChange={() => setPriceMode('Retail')}
                            className="accent-[#0076be]"
                          />
                          <span>Retail</span>
                        </label>
                        <label className="inline-flex items-center gap-1.5 cursor-pointer">
                          <input
                            type="radio"
                            checked={priceMode === 'Wholesale'}
                            onChange={() => setPriceMode('Wholesale')}
                            className="accent-[#0076be]"
                          />
                          <span>Wholesale</span>
                        </label>
                      </div>

                      <div className="pt-2 mt-2 border-t border-[#e9ecef] flex items-center justify-between text-[11.5px] text-[#0076be] font-medium">
                        <span>Update Trim &amp; Options</span>
                        <span>Hide</span>
                      </div>
                    </div>
                  )}
                </div>
              </div>

              {/* Right 5 Columns: 5 Official Overview Rows */}
              <div className="md:col-span-5 bg-white divide-y divide-[#e9ecef] flex flex-col justify-center">
                {/* Row 1: Service History Records */}
                <a
                  href="#detailed-history-section"
                  className="px-4 py-3.5 flex items-center justify-between hover:bg-[#f8f9fa] transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <span className="w-7 h-7 rounded-full bg-[#eef4f8] text-[#495057] flex items-center justify-center shrink-0">
                      <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M22.7 19l-9.1-9.1c.9-2.3.4-5-1.5-6.9-2-2-5-2.4-7.4-1.3L9 6 6 9 1.6 4.7C.4 7.1.9 10.1 2.9 12.1c1.9 1.9 4.6 2.4 6.9 1.5l9.1 9.1c.4.4 1 .4 1.4 0l2.3-2.3c.5-.4.5-1.1.1-1.4z" />
                      </svg>
                    </span>
                    <span className="text-[13.5px] text-[#212529] font-medium">
                      {trText(serviceRecordsText, lang)}
                    </span>
                  </div>
                  <svg className="w-4 h-4 text-[#0076be] shrink-0" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                    <path d="M9 5l7 7-7 7" />
                  </svg>
                </a>

                {/* Row 2: Owners Count */}
                <a
                  href="#ownership-history-section"
                  className="px-4 py-3.5 flex items-center justify-between hover:bg-[#f8f9fa] transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <span className="w-7 h-7 rounded-full bg-[#0076be] text-white font-extrabold text-[12px] flex items-center justify-center shrink-0">
                      {owners}
                    </span>
                    <span className="text-[13.5px] text-[#212529] font-medium">
                      {owners === 1
                        ? isAr
                          ? 'مركبة مالك واحد (CARFAX 1-Owner)'
                          : 'CARFAX 1-Owner Vehicle'
                        : isAr
                        ? `${owners} ملاك سابقين`
                        : `${owners}-Owner Vehicle`}
                    </span>
                  </div>
                  <svg className="w-4 h-4 text-[#0076be] shrink-0" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                    <path d="M9 5l7 7-7 7" />
                  </svg>
                </a>

                {/* Row 3: Vehicle Use (Personal / Rental / Lease) */}
                <a
                  href="#ownership-history-section"
                  className="px-4 py-3.5 flex items-center justify-between hover:bg-[#f8f9fa] transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <span className="w-7 h-7 rounded-full bg-[#eef4f8] text-[#495057] flex items-center justify-center shrink-0">
                      <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M18.92 6.01C18.72 5.42 18.16 5 17.5 5h-11c-.66 0-1.21.42-1.42 1.01L3 12v8c0 .55.45 1 1 1h1c.55 0 1-.45 1-1v-1h12v1c0 .55.45 1 1 1h1c.55 0 1-.45 1-1v-8l-2.08-5.99zM6.5 16c-.83 0-1.5-.67-1.5-1.5S5.67 13 6.5 13s1.5.67 1.5 1.5S7.33 16 6.5 16zm11 0c-.83 0-1.5-.67-1.5-1.5s.67-1.5 1.5-1.5 1.5.67 1.5 1.5-.67 1.5-1.5 1.5zM5 11l1.5-4.5h11L19 11H5z" />
                      </svg>
                    </span>
                    <span className="text-[13.5px] text-[#212529] font-medium">
                      {trText(vehicleUse, lang)}
                    </span>
                  </div>
                  <svg className="w-4 h-4 text-[#0076be] shrink-0" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                    <path d="M9 5l7 7-7 7" />
                  </svg>
                </a>

                {/* Row 4: Last Owned Location */}
                <a
                  href="#ownership-history-section"
                  className="px-4 py-3.5 flex items-center justify-between hover:bg-[#f8f9fa] transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <span className="w-7 h-7 rounded-full bg-[#e6f4ea] text-[#1c8237] flex items-center justify-center shrink-0">
                      <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z" />
                      </svg>
                    </span>
                    <span className="text-[13.5px] text-[#212529] font-medium">
                      {trText(lastLocation, lang)}
                    </span>
                  </div>
                  <svg className="w-4 h-4 text-[#0076be] shrink-0" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                    <path d="M9 5l7 7-7 7" />
                  </svg>
                </a>

                {/* Row 5: Detailed Records Available */}
                <a
                  href="#detailed-history-section"
                  className="px-4 py-3.5 flex items-center justify-between hover:bg-[#f8f9fa] transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <span className="w-7 h-7 rounded-full bg-[#fff8e1] text-[#b28900] flex items-center justify-center shrink-0">
                      <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z" />
                      </svg>
                    </span>
                    <span className="text-[13.5px] text-[#212529] font-medium">
                      {trText(detailedCountText, lang)}
                    </span>
                  </div>
                  <svg className="w-4 h-4 text-[#0076be] shrink-0" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                    <path d="M9 5l7 7-7 7" />
                  </svg>
                </a>
              </div>
            </div>
          </div>

          {/* LEAD-IN DISCLAIMER PARAGRAPH */}
          <p className="text-[12px] text-[#495057] leading-relaxed my-4 px-1">
            This CARFAX Vehicle History Report is based only on{' '}
            <span className="text-[#0076be] underline">information</span>{' '}
            supplied to CARFAX. Other information about this vehicle, including
            problems, may not have been reported to CARFAX. Use this report as
            one important tool, along with a vehicle inspection and test drive,
            to make a better decision about your next used car.
          </p>

          {/* SECTION: ACCIDENT / DAMAGE HISTORY */}
          <section
            id="accident-damage-section"
            className="bg-white border border-[#d0d7de] rounded-[6px] overflow-hidden mb-6 shadow-[0_1px_3px_rgba(0,0,0,0.04)]"
          >
            <div className="bg-[#0076be] px-4 py-3 text-white">
              <div className="flex items-center gap-2.5">
                <img
                  src="/carfax-assets/carfax-logo.svg"
                  alt="CARFAX"
                  className="h-[22px] w-auto"
                />
                <h2 className="text-[18px] font-bold">
                  {trText('Accident / Damage History', lang)}
                </h2>
              </div>
              <p className="text-[12px] text-white/90 mt-0.5">
                {trText('Not all accidents / issues are reported to CARFAX', lang)}
              </p>
            </div>

            <div className="p-4">
              {accidents > 0 || accidentEvents.length > 0 ? (
                <div className="space-y-3">
                  {(accidentEvents.length > 0
                    ? accidentEvents
                    : [{ title: 'Event 1', details: accidentSummary }]
                  ).map((ev, idx) => (
                    <div
                      key={idx}
                      className="border border-[#d0d7de] rounded overflow-hidden"
                    >
                      <div className="bg-[#e9ecef] px-4 py-2 text-center font-bold text-[13.5px] text-[#212529] border-b border-[#d0d7de]">
                        {ev.title || `Event ${idx + 1}`}
                      </div>
                      <div className="p-4 text-[13.5px] text-[#212529] whitespace-pre-line leading-relaxed font-medium">
                        {trText(ev.details?.replace(/^Event\s+\d+\s*/i, ''), lang)}
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="flex items-center gap-3 py-2 text-[#1c8237] font-bold text-[14px]">
                  <span className="w-6 h-6 rounded-full bg-[#1c8237] text-white flex items-center justify-center text-xs">
                    ✓
                  </span>
                  <span>
                    {trText('No accidents or damage reported to CARFAX.', lang)}
                  </span>
                </div>
              )}
            </div>
          </section>

          {/* SECTION: ADDITIONAL HISTORY TABLE */}
          {additionalTable && (
            <section
              id="additional-history-section"
              className="bg-white border border-[#d0d7de] rounded-[6px] overflow-hidden mb-6 shadow-[0_1px_3px_rgba(0,0,0,0.04)]"
            >
              <table className="w-full border-collapse text-left">
                <thead>
                  <tr>
                    <th className="bg-[#0076be] px-4 py-3 text-white w-[62%] align-middle">
                      <div className="flex items-center gap-2.5">
                        <img
                          src="/carfax-assets/carfax-logo.svg"
                          alt="CARFAX"
                          className="h-[22px] w-auto"
                        />
                        <span className="text-[18px] font-bold">
                          {trText('Additional History', lang)}
                        </span>
                      </div>
                      <div className="text-[12px] text-white/90 font-normal mt-0.5">
                        {trText(
                          'Not all accidents / issues are reported to CARFAX',
                          lang
                        )}
                      </div>
                    </th>
                    {(additionalTable.headers?.length > 1
                      ? additionalTable.headers.slice(1)
                      : ['Owner 1']
                    ).map((ownerCol, ci) => (
                      <th
                        key={ci}
                        className="bg-[#003478] px-4 py-3 text-white text-center text-[13.5px] font-bold border-l border-white/20 align-middle"
                      >
                        <div className="inline-flex items-center gap-1.5">
                          <svg className="w-4 h-4 fill-current" viewBox="0 0 24 24">
                            <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z" />
                          </svg>
                          <span>{ownerCol}</span>
                        </div>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#dee2e6] text-[13px]">
                  {additionalTable.rows.map((row, ri) => {
                    const { main, sub } = splitRowHeading(row[0])
                    const valCols = row.slice(1)
                    return (
                      <tr
                        key={ri}
                        className="hover:bg-[#f8f9fa] transition-colors"
                      >
                        <th
                          scope="row"
                          className="px-4 py-3.5 font-normal text-[#212529] align-middle border-r border-[#dee2e6]"
                        >
                          <div className="font-bold text-[13.5px] text-[#212529]">
                            {trText(main, lang)}
                          </div>
                          {sub && (
                            <div className="text-[12px] text-[#495057] mt-0.5 leading-snug">
                              {trText(sub, lang)}
                            </div>
                          )}
                        </th>
                        {valCols.map((cell, ci) => {
                          const cellLower = String(cell).toLowerCase()
                          const isClean =
                            cellLower.includes('no issues') ||
                            cellLower.includes('no recalls') ||
                            cellLower.includes('warranty active') ||
                            cellLower.includes('no problem')
                          const isAlert =
                            cellLower.includes('damage') ||
                            cellLower.includes('accident') ||
                            cellLower.includes('reported') && !isClean
                          return (
                            <td
                              key={ci}
                              className="px-4 py-3.5 text-center align-middle"
                            >
                              <div className="inline-flex items-center justify-center gap-2 font-medium text-[#212529]">
                                {isClean && (
                                  <span className="w-5 h-5 rounded-full bg-[#1c8237] text-white flex items-center justify-center text-[11px] font-bold shrink-0">
                                    ✓
                                  </span>
                                )}
                                {isAlert && (
                                  <span className="w-5 h-5 rounded-full bg-[#d93025] text-white flex items-center justify-center text-[11px] font-bold shrink-0">
                                    !
                                  </span>
                                )}
                                <span
                                  className={
                                    isAlert
                                      ? 'text-[#0076be] underline font-semibold'
                                      : ''
                                  }
                                >
                                  {trText(cell, lang)}
                                </span>
                              </div>
                            </td>
                          )
                        })}
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </section>
          )}

          {/* SECTION: TITLE HISTORY TABLE + BUYBACK GUARANTEE */}
          {titleTable && (
            <section
              id="title-history-section"
              className="bg-white border border-[#d0d7de] rounded-[6px] overflow-hidden mb-6 shadow-[0_1px_3px_rgba(0,0,0,0.04)]"
            >
              <table className="w-full border-collapse text-left">
                <thead>
                  <tr>
                    <th className="bg-[#0076be] px-4 py-3 text-white w-[62%] align-middle">
                      <div className="flex items-center gap-2.5">
                        <img
                          src="/carfax-assets/carfax-logo.svg"
                          alt="CARFAX"
                          className="h-[22px] w-auto"
                        />
                        <span className="text-[18px] font-bold">
                          {trText('Title History', lang)}
                        </span>
                      </div>
                      <div className="text-[12px] text-white/90 font-normal mt-0.5">
                        {trText(
                          'CARFAX guarantees the information in this section',
                          lang
                        )}
                      </div>
                    </th>
                    {(titleTable.headers?.length > 1
                      ? titleTable.headers.slice(1)
                      : ['Owner 1']
                    ).map((ownerCol, ci) => (
                      <th
                        key={ci}
                        className="bg-[#003478] px-4 py-3 text-white text-center text-[13.5px] font-bold border-l border-white/20 align-middle"
                      >
                        <div className="inline-flex items-center gap-1.5">
                          <svg className="w-4 h-4 fill-current" viewBox="0 0 24 24">
                            <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z" />
                          </svg>
                          <span>{ownerCol}</span>
                        </div>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#dee2e6] text-[13px]">
                  {titleTable.rows
                    .filter(
                      (r) =>
                        !String(r[0])
                          .toLowerCase()
                          .includes('none of these title problems')
                    )
                    .map((row, ri) => {
                      const { main, sub } = splitRowHeading(row[0])
                      const valCols = row.slice(1)
                      return (
                        <tr
                          key={ri}
                          className="hover:bg-[#f8f9fa] transition-colors"
                        >
                          <th
                            scope="row"
                            className="px-4 py-3.5 font-normal text-[#212529] align-middle border-r border-[#dee2e6]"
                          >
                            <div className="font-bold text-[13.5px] text-[#212529]">
                              {trText(main, lang)}
                            </div>
                            {sub && (
                              <div className="text-[12px] text-[#495057] mt-0.5 leading-snug">
                                {trText(sub, lang)}
                              </div>
                            )}
                          </th>
                          {valCols.map((cell, ci) => {
                            const hasGuaranteed = String(cell)
                              .toLowerCase()
                              .includes('guaranteed')
                            return (
                              <td
                                key={ci}
                                className="px-4 py-3.5 text-center align-middle"
                              >
                                <div className="inline-flex items-center justify-center gap-2">
                                  <span className="w-5 h-5 rounded-full bg-[#1c8237] text-white flex items-center justify-center text-[11px] font-bold shrink-0">
                                    ✓
                                  </span>
                                  <div className="text-left leading-tight">
                                    {hasGuaranteed ? (
                                      <>
                                        <div className="font-bold text-[#212529]">
                                          {trText('Guaranteed', lang)}
                                        </div>
                                        <div className="text-[12px] text-[#495057]">
                                          {trText('No Problem', lang)}
                                        </div>
                                      </>
                                    ) : (
                                      <span>{trText(cell, lang)}</span>
                                    )}
                                  </div>
                                </div>
                              </td>
                            )
                          })}
                        </tr>
                      )
                    })}

                  {/* Official CARFAX Buyback Guarantee Footer Row */}
                  <tr>
                    <td
                      colSpan={Math.max(2, titleTable.headers?.length || 2)}
                      className="px-5 py-4 bg-[#fafbfc]"
                    >
                      <div className="flex flex-col sm:flex-row items-center gap-4">
                        <img
                          src="/carfax-assets/buyback-guarantee.gif"
                          alt="CARFAX Buyback Guarantee"
                          className="h-[52px] w-auto shrink-0"
                        />
                        <div className="text-[12px] text-[#212529] leading-relaxed">
                          <p>
                            <strong className="font-bold">Guaranteed</strong> -
                            None of these title problems were reported by a U.S.
                            state Department of Motor Vehicles (DMV). If you
                            find that any of these title problems were reported
                            by a DMV and not included in this report, you may
                            qualify.
                          </p>
                          <p className="mt-1 text-[#0076be] font-medium">
                            <span className="underline cursor-pointer">
                              View Terms
                            </span>{' '}
                            |{' '}
                            <span className="underline cursor-pointer">
                              View Certificate
                            </span>
                          </p>
                        </div>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </section>
          )}

          {/* SECTION: OWNERSHIP HISTORY TABLE + 1-OWNER BADGE */}
          {ownershipTable && (
            <section
              id="ownership-history-section"
              className="bg-white border border-[#d0d7de] rounded-[6px] overflow-hidden mb-6 shadow-[0_1px_3px_rgba(0,0,0,0.04)]"
            >
              <table className="w-full border-collapse text-left">
                <thead>
                  <tr>
                    <th className="bg-[#0076be] px-4 py-3 text-white w-[62%] align-middle">
                      <div className="flex items-center gap-2.5">
                        <img
                          src="/carfax-assets/carfax-logo.svg"
                          alt="CARFAX"
                          className="h-[22px] w-auto"
                        />
                        <span className="text-[18px] font-bold">
                          {trText('Ownership History', lang)}
                        </span>
                      </div>
                      <div className="text-[12px] text-white/90 font-normal mt-0.5">
                        {trText('The number of owners is estimated', lang)}
                      </div>
                    </th>
                    {(ownershipTable.headers?.length > 1
                      ? ownershipTable.headers.slice(1)
                      : ['Owner 1']
                    ).map((ownerCol, ci) => (
                      <th
                        key={ci}
                        className="bg-[#003478] px-4 py-3 text-white text-center text-[13.5px] font-bold border-l border-white/20 align-middle"
                      >
                        <div className="inline-flex items-center gap-1.5">
                          <svg className="w-4 h-4 fill-current" viewBox="0 0 24 24">
                            <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z" />
                          </svg>
                          <span>{ownerCol}</span>
                        </div>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#dee2e6] text-[13px]">
                  {ownershipTable.rows.map((row, ri) => {
                    const label = row[0] || ''
                    const valCols = row.slice(1)
                    return (
                      <tr
                        key={ri}
                        className="hover:bg-[#f8f9fa] transition-colors"
                      >
                        <th
                          scope="row"
                          className="px-4 py-3 font-medium text-[#212529] align-middle border-r border-[#dee2e6]"
                        >
                          {trText(label, lang)}
                        </th>
                        {valCols.map((cell, ci) => (
                          <td
                            key={ci}
                            className="px-4 py-3 text-center font-medium text-[#212529] align-middle"
                          >
                            {trText(cell, lang)}
                          </td>
                        ))}
                      </tr>
                    )
                  })}
                </tbody>
              </table>

              {owners === 1 && (
                <div className="py-3 px-4 bg-[#f8f9fa] border-t border-[#dee2e6] flex justify-center">
                  <img
                    src="/carfax-assets/one-owner-badge.svg"
                    alt="CARFAX 1-Owner"
                    className="h-[46px] w-auto"
                  />
                </div>
              )}
            </section>
          )}

          {/* ANY OTHER TABLES */}
          {otherTables.map((tbl, ti) => (
            <section
              key={ti}
              className="bg-white border border-[#d0d7de] rounded-[6px] overflow-hidden mb-6 shadow-[0_1px_3px_rgba(0,0,0,0.04)]"
            >
              <div className="bg-[#0076be] px-4 py-3 text-white flex items-center gap-2.5">
                <img
                  src="/carfax-assets/carfax-logo.svg"
                  alt="CARFAX"
                  className="h-[22px] w-auto"
                />
                <h2 className="text-[18px] font-bold">
                  {trText(tbl.title, lang)}
                </h2>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full border-collapse text-left text-[13px]">
                  <tbody className="divide-y divide-[#dee2e6]">
                    {tbl.rows.map((r, ri) => (
                      <tr key={ri} className="hover:bg-[#f8f9fa]">
                        {r.map((c, ci) => (
                          <td
                            key={ci}
                            className={`px-4 py-3 ${
                              ci === 0 ? 'font-bold bg-[#fafbfc] w-1/3' : ''
                            }`}
                          >
                            {trText(c, lang)}
                          </td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          ))}

          {/* SECTION: DETAILED HISTORY CHRONOLOGICAL TIMELINE */}
          {detailedRecords.length > 0 && (
            <section
              id="detailed-history-section"
              className="bg-white border border-[#d0d7de] rounded-[6px] overflow-hidden mb-6 shadow-[0_1px_3px_rgba(0,0,0,0.04)]"
            >
              {/* Blue Section Header */}
              <div className="bg-[#0076be] px-4 py-3 text-white flex items-center justify-between">
                <div className="flex items-center gap-2.5">
                  <img
                    src="/carfax-assets/carfax-logo.svg"
                    alt="CARFAX"
                    className="h-[22px] w-auto"
                  />
                  <h2 className="text-[18px] font-bold">
                    {trText('Detailed History', lang)}
                  </h2>
                </div>
                <span className="text-[12.5px] underline cursor-pointer text-white/90">
                  Glossary
                </span>
              </div>

              <div className="p-3 sm:p-4">
                {/* Dark Navy Owner Tab Banner */}
                <div className="bg-[#003478] text-white px-4 py-2.5 rounded-t-[4px] flex items-center justify-between">
                  <div className="flex items-center gap-2.5">
                    <span className="w-7 h-7 rounded-full bg-white text-[#003478] flex items-center justify-center font-bold">
                      <svg className="w-4 h-4 fill-current" viewBox="0 0 24 24">
                        <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z" />
                      </svg>
                    </span>
                    <div>
                      <div className="font-bold text-[14.5px] leading-tight">
                        Owner 1
                      </div>
                      <div className="text-[11.5px] text-white/85">
                        {trText(vehicleUse, lang)}
                      </div>
                    </div>
                  </div>
                  <div className="text-[13px] font-bold">
                    {trText(vehicleUse, lang)}
                  </div>
                </div>

                {/* Detailed Records Table */}
                <div className="overflow-x-auto border-x border-b border-[#d0d7de]">
                  <table className="w-full border-collapse text-left text-[13px]">
                    <thead>
                      <tr className="bg-[#e8f1f8] text-[#212529] border-b border-[#d0d7de] font-bold text-[12.5px]">
                        <th className="py-2.5 px-3.5 w-[115px]">Date</th>
                        <th className="py-2.5 px-3.5 w-[105px]">Mileage</th>
                        <th className="py-2.5 px-3.5 w-[220px]">Source</th>
                        <th className="py-2.5 px-4">Comments</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-[#e1e6eb]">
                      {detailedRecords.map((rec, idx) => {
                        const commentsLines = String(rec.comments || '')
                          .split('\n')
                          .map((l) => l.trim())
                          .filter(Boolean)
                        const sourceLines = String(rec.source || '')
                          .split('\n')
                          .map((l) => l.trim())
                          .filter(Boolean)
                        const commLower = String(rec.comments || '').toLowerCase()
                        const isWindowSticker = commLower.includes('window sticker')
                        const isService =
                          commLower.includes('serviced') ||
                          commLower.includes('oil') ||
                          commLower.includes('inspection')
                        const isDamage =
                          commLower.includes('damage') ||
                          commLower.includes('accident')

                        return (
                          <tr
                            key={idx}
                            className={`${
                              idx % 2 === 1 ? 'bg-[#f4f8fb]' : 'bg-white'
                            } hover:bg-[#edf4fa] transition-colors align-top`}
                          >
                            {/* Date */}
                            <td className="py-3.5 px-3.5 font-medium text-[#212529] whitespace-nowrap">
                              {trText(rec.date, lang)}
                            </td>

                            {/* Mileage */}
                            <td className="py-3.5 px-3.5 font-mono text-[#212529] whitespace-nowrap">
                              {rec.mileage || ''}
                            </td>

                            {/* Source */}
                            <td className="py-3.5 px-3.5 text-[#212529] leading-snug">
                              {sourceLines.map((sln, si) => (
                                <div
                                  key={si}
                                  className={
                                    si === 0
                                      ? 'font-medium text-[#212529]'
                                      : 'text-[12px] text-[#495057]'
                                  }
                                >
                                  {sln}
                                </div>
                              ))}
                            </td>

                            {/* Comments */}
                            <td className="py-3.5 px-4 text-[#212529]">
                              <div className="flex items-start gap-3">
                                {/* Row Icon (Service Wrench / Damage Alert / Window Sticker) */}
                                {isService && (
                                  <span className="w-6 h-6 rounded-full bg-[#e8f1f8] text-[#00599c] flex items-center justify-center shrink-0 mt-0.5">
                                    <svg className="w-3.5 h-3.5 fill-current" viewBox="0 0 24 24">
                                      <path d="M22.7 19l-9.1-9.1c.9-2.3.4-5-1.5-6.9-2-2-5-2.4-7.4-1.3L9 6 6 9 1.6 4.7C.4 7.1.9 10.1 2.9 12.1c1.9 1.9 4.6 2.4 6.9 1.5l9.1 9.1c.4.4 1 .4 1.4 0l2.3-2.3c.5-.4.5-1.1.1-1.4z" />
                                    </svg>
                                  </span>
                                )}
                                {isDamage && (
                                  <span className="w-6 h-6 rounded-full bg-[#d93025] text-white flex items-center justify-center text-xs font-black shrink-0 mt-0.5">
                                    !
                                  </span>
                                )}

                                <div className="flex-1 space-y-1">
                                  {isWindowSticker ? (
                                    <div className="inline-flex items-center gap-2 text-[#0076be] font-bold hover:underline cursor-pointer">
                                      <span>Original Window Sticker</span>
                                      <img
                                        src="/carfax-assets/window-sticker-icon.svg"
                                        alt=""
                                        className="h-5 w-auto"
                                      />
                                    </div>
                                  ) : (
                                    commentsLines.map((cln, ci) => (
                                      <div
                                        key={ci}
                                        className={
                                          ci === 0
                                            ? 'font-bold text-[#212529]'
                                            : 'text-[12.5px] text-[#212529] pl-3 relative before:content-["•"] before:absolute before:left-0 before:text-[#495057]'
                                        }
                                      >
                                        {trText(cln, lang)}
                                      </div>
                                    ))
                                  )}
                                </div>
                              </div>
                            </td>
                          </tr>
                        )
                      })}
                    </tbody>
                  </table>
                </div>

                {/* Car Fox Expert Footer Callout */}
                <div className="mt-4 p-3 bg-[#f4f8fb] border border-[#d5e3f0] rounded flex items-center gap-3">
                  <img
                    src="/carfax-assets/carfox-expert.avif"
                    alt="Car Fox"
                    className="w-[54px] h-auto shrink-0"
                  />
                  <div className="bg-white border border-[#bcd0e5] rounded px-3.5 py-2 text-[12.5px] text-[#212529] font-medium shadow-sm">
                    Have Questions? Consumers, please visit our Help Center at{' '}
                    <span className="text-[#0076be] underline">www.carfax.com</span>.
                    Dealers or Subscribers, please visit{' '}
                    <span className="text-[#0076be] underline">
                      www.carfaxonline.com
                    </span>
                    .
                  </div>
                </div>
              </div>
            </section>
          )}
        </main>
      </div>
    </div>
  )
}
