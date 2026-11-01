import React from 'react'

export default function CarfaxBadgeLogo({ size = 'md', subtitle = 'DEALER PORTAL | بوابة تقارير المركبات' }) {
  const boxClass = size === 'sm'
    ? 'w-5 h-6 text-xs font-black'
    : size === 'lg'
    ? 'w-8 h-9 text-lg font-black'
    : 'w-6 h-7 text-sm font-black'

  const foxClass = size === 'sm'
    ? 'text-[10px] px-1.5 py-0.5'
    : size === 'lg'
    ? 'text-xs px-2.5 py-1'
    : 'text-[11px] px-2 py-0.5'

  return (
    <div className="flex flex-col items-start select-none" dir="ltr">
      <div className="flex items-center gap-[2px]">
        <span className={`${boxClass} flex items-center justify-center bg-[#003478] text-white rounded-[2px] tracking-tight`}>C</span>
        <span className={`${boxClass} flex items-center justify-center bg-[#003478] text-white rounded-[2px] tracking-tight`}>A</span>
        <span className={`${boxClass} flex items-center justify-center bg-[#003478] text-white rounded-[2px] tracking-tight`}>R</span>
        <span className={`${boxClass} flex items-center justify-center bg-[#003478] text-white rounded-[2px] tracking-tight`}>F</span>
        <span className={`${boxClass} flex items-center justify-center bg-[#003478] text-white rounded-[2px] tracking-tight`}>A</span>
        <span className={`${boxClass} flex items-center justify-center bg-[#003478] text-white rounded-[2px] tracking-tight`}>X</span>
        <span className={`${foxClass} flex items-center justify-center bg-[#e65100] text-white font-bold rounded-[2px] tracking-wider ml-0.5`}>FOX</span>
      </div>
      {subtitle && (
        <span className="text-[10px] font-bold text-[#003478] dark:text-blue-300 tracking-wider uppercase mt-1">
          {subtitle}
        </span>
      )}
    </div>
  )
}
