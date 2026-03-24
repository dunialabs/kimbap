import { AlertTriangle } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

interface AnonymousAccessConfigurationProps {
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  rateLimit: number
  onRateLimitChange: (value: number) => void
  baseUrl?: string
}

export function AnonymousAccessConfiguration({
  checked,
  onCheckedChange,
  rateLimit,
  onRateLimitChange,
  baseUrl,
}: AnonymousAccessConfigurationProps) {
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">Anonymous Access</h3>
      </div>
      <p className="text-xs text-gray-500 dark:text-gray-400">
        Expose this server on the <code className="text-xs bg-gray-100 dark:bg-gray-800 px-1 py-0.5 rounded">/mcp/public</code> endpoint for clients without a Kimbap access token.
        Same tools, capabilities, and protocol as <code className="text-xs bg-gray-100 dark:bg-gray-800 px-1 py-0.5 rounded">/mcp</code> — only the endpoint and authentication differ.
        Rate limiting is per source IP (clients behind the same NAT/proxy share a single bucket).
      </p>
      <div className="border rounded-lg p-3 bg-gray-50 dark:bg-gray-900">
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    Enable Anonymous Access
                  </span>
                </div>
                <Switch checked={checked} onCheckedChange={onCheckedChange} size="sm" />
              </div>
            </TooltipTrigger>
            <TooltipContent>
            <p className="text-sm max-w-xs">
              Expose this server on <code className="text-xs bg-gray-100 dark:bg-gray-800 px-1 py-0.5 rounded">/mcp/public</code> for token-less access.
              Same capabilities as <code className="text-xs bg-gray-100 dark:bg-gray-800 px-1 py-0.5 rounded">/mcp</code>; rate-limited per source IP.
            </p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
      {checked && (
        <div className="space-y-3">
          <div className="border rounded-lg p-3 bg-gray-50 dark:bg-gray-900">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Anonymous Rate Limit (req/min per source IP)
            </label>
            <Input
              type="number"
              min={1}
              max={1000}
              value={rateLimit}
              onChange={(e) => {
                const val = parseInt(e.target.value, 10)
                if (!isNaN(val)) {
                  onRateLimitChange(Math.min(1000, Math.max(1, val)))
                }
              }}
              className="w-full"
            />
          </div>
          <div className="bg-amber-50 dark:bg-amber-950/20 border border-amber-200 dark:border-amber-800 rounded-md p-3">
            <div className="flex gap-2">
              <AlertTriangle className="h-4 w-4 text-amber-600 dark:text-amber-400 mt-0.5 flex-shrink-0" />
              <div className="text-xs text-amber-800 dark:text-amber-200">
                <p className="font-semibold">
                  When enabled, clients can access this server without authentication:
                </p>
                <p className="mt-1 font-mono text-amber-700 dark:text-amber-300">
                  POST {baseUrl || 'https://your-domain.com'}/mcp/public
                </p>
                <p className="mt-1">Authenticated clients can also use this endpoint. Rate limiting is per source IP — clients behind the same NAT or proxy share a single bucket.</p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
