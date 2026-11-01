import React from 'react'
import { Card } from '@/components/ui/card'

export default function ReportFrame({ htmlContent, iframeRef, onPrintRequest }) {
  const handleIframeLoad = () => {
    try {
      const doc = iframeRef?.current?.contentDocument
      if (!doc) return
      // Wire up Carfax's internal Print button inside the iframe so clicking it works seamlessly
      const buttons = doc.querySelectorAll('button, a')
      buttons.forEach((btn) => {
        const txt = (btn.innerText || '').trim().toLowerCase()
        if (txt === 'print' || btn.className?.toString().toLowerCase().includes('print')) {
          btn.onclick = (e) => {
            e.preventDefault()
            if (onPrintRequest) onPrintRequest()
          }
        }
      })
    } catch {
      // ignore cross-frame errors if any
    }
  }

  return (
    <Card className="border-border shadow-xs overflow-hidden w-full">
      <div className="p-0 bg-white">
        <iframe
          ref={iframeRef}
          srcDoc={htmlContent}
          onLoad={handleIframeLoad}
          title="Carfax Official Report"
          className="w-full h-[80vh] border-0 bg-white"
          sandbox="allow-same-origin allow-scripts allow-popups allow-modals"
        />
      </div>
    </Card>
  )
}
