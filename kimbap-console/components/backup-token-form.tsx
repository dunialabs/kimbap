'use client'

import type React from 'react'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Copy, CheckCheck, Download, Lock } from 'lucide-react'

interface BackupTokenFormProps {
  ownerToken: string
  onComplete: () => void
}

export function BackupTokenForm({ ownerToken, onComplete }: BackupTokenFormProps) {
  const [tokenCopied, setTokenCopied] = useState(false)


  const handleCopyToken = async () => {
    try {
      await navigator.clipboard.writeText(ownerToken)
      setTokenCopied(true)
      setTimeout(() => setTokenCopied(false), 2000)
    } catch (err) {
      // Clipboard copy failed silently
    }
  }

  const handleDownloadToken = () => {
    const tokenData = {
      server: 'My MCP Server',
      token: ownerToken,
      role: 'Owner',
      created: new Date().toISOString(),
      warning: 'Keep this token secure. This is the only time you will see it.'
    }

    const blob = new Blob([JSON.stringify(tokenData, null, 2)], {
      type: 'application/json'
    })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `owner-token-${Date.now()}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)


  }

  return (
    <div className="space-y-[12px] max-w-[600px] py-[32px] px-[24px] rounded-xl border border-border shadow-[0_0_12px_0_rgba(0,0,0,0.12)]">
      <div>
        <h2 className="text-[24px] font-bold mb-[4px]">
          Backup Your Owner Token
        </h2>
        <p className="text-muted-foreground text-[14px]">
          Save this token now. It will not be shown again.
        </p>
      </div>

      {/* Warning Alert */}
      <Alert className="border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20">
        <div className="flex items-start gap-2">
          <Lock className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5" />
          <div className="flex-1">
            <div className="font-bold text-sm text-red-800 dark:text-red-200 mb-1">
              Critical: Save Your Owner Token
            </div>
            <AlertDescription className="text-sm text-red-700 dark:text-red-300">
              This token grants full access to your server. If lost, other devices will not be able to reconnect.
            </AlertDescription>
          </div>
        </div>
      </Alert>

      {/* Token Display */}
      <div className="space-y-[8px]">
        <div className="flex justify-between items-center">
          <span className="text-[14px] font-[700] text-foreground">Owner Token:</span>
          <div className="flex gap-2">
            <Button
              onClick={handleCopyToken}
              size="sm"
              variant="outline"
              className="h-8 px-3 text-xs border-border"
            >
              {tokenCopied ? (
                <>
                  <CheckCheck className="h-3 w-3 mr-1" />
                  Copied
                </>
              ) : (
                <>
                  <Copy className="h-3 w-3 mr-1" />
                  Copy
                </>
              )}
            </Button>
            <Button
              onClick={handleDownloadToken}
              size="sm"
              variant="outline"
              className="h-8 px-3 text-xs border-border"
            >
              <Download className="h-3 w-3 mr-1" />
              Download
            </Button>
          </div>
        </div>
        <div className="p-4 bg-muted/50 rounded-lg border border-border">
          <code className="text-sm font-mono break-all text-foreground leading-relaxed">
            {ownerToken}
          </code>
        </div>
      </div>

      {/* Info Alert */}
      <Alert className="border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-950/20">
        <AlertDescription className="text-sm text-amber-800 dark:text-amber-200">
          <span className="font-[700]">Recommendation:</span> Store this token in a password manager or secure location.
        </AlertDescription>
      </Alert>

      {/* Complete Button */}
      <Button
        onClick={onComplete}
        className="w-full h-12 text-base bg-slate-900 hover:bg-slate-800 text-white rounded-[8px] dark:bg-slate-100 dark:hover:bg-slate-200 dark:text-slate-900"
        size="lg"
      >
        I've Backed Up My Token
      </Button>
    </div>
  )
}
