import React from 'react'
import { Card } from '@/components/ui/card'

export default function ReportFrame({ htmlContent, iframeRef }) {
  return (
    <Card className="border-border shadow-xs overflow-hidden w-full">
      <div className="p-0 bg-white">
        <iframe
          ref={iframeRef}
          srcDoc={htmlContent}
          title="Carfax Official Report"
          className="w-full h-[78vh] border-0 bg-white"
          sandbox="allow-same-origin allow-scripts allow-popups allow-modals"
        />
      </div>
    </Card>
  )
}

