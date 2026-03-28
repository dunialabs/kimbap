'use client'

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { ComponentType } from 'react'
import {
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  Clock,
  Loader2,
  PlugZap,
  Wrench
} from 'lucide-react'

import { useDebounce } from '@/hooks/use-debounce'
import type { RestApiValidationReport, ToolDefinition } from '@/lib/rest-api-utils'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Textarea } from '@/components/ui/textarea'

interface RestApiValidationPanelProps {
  configText: string
  disabled?: boolean
  className?: string
}

interface ConnectionTestResponse {
  success: boolean
  message?: string
  statusCode?: number
  method?: string
  responseTime?: number
  request?: {
    method?: string
    url?: string
    headers?: Record<string, string>
  }
  response?: {
    statusCode?: number
    statusText?: string
    headers?: Record<string, string>
    bodyPreview?: string
  }
  auth?: {
    type: string
    passed: boolean
  }
  error?: string
}

interface ToolTestResponse {
  success: boolean
  message?: string
  statusCode: number
  statusText?: string
  responseTime: number
  request: {
    method: string
    url: string
    headers: Record<string, string>
    body?: string
  }
  response: {
    statusCode: number
    statusText?: string
    headers: Record<string, string>
    body: any
    bodyPreview?: string
  }
  transformedResponse?: any
  warnings: string[]
  missingParams: string[]
  invalidParams: string[]
}


function getToolParamPlaceholder(param: ToolDefinition['parameters'][number]): string {
  switch (param.type) {
    case 'number':
      return 'e.g., 10'
    case 'boolean':
      return 'e.g., true'
    case 'array':
      return 'e.g., value1,value2'
    case 'object':
      return 'e.g., {"key":"value"}'
    default:
      return `e.g., ${param.name}`
  }
}

const validationStatusConfig: Record<
  RestApiValidationReport['summary'],
  { label: string; badgeClass: string; icon: ComponentType<{ className?: string }> }
> = {
  idle: {
    label: 'Pending',
    badgeClass: 'bg-gray-100 text-gray-700 border-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700',
    icon: Clock
  },
  valid: {
    label: 'Valid',
    badgeClass: 'bg-green-100 text-green-800 border-green-300 dark:bg-green-900 dark:text-green-300 dark:border-green-800',
    icon: CheckCircle2
  },
  warning: {
    label: 'Warnings',
    badgeClass: 'bg-amber-100 text-amber-800 border-amber-300 dark:bg-amber-900 dark:text-amber-300 dark:border-amber-800',
    icon: AlertTriangle
  },
  error: {
    label: 'Errors',
    badgeClass: 'bg-red-100 text-red-800 border-red-300 dark:bg-red-900 dark:text-red-300 dark:border-red-800',
    icon: AlertCircle
  }
}

export function RestApiValidationPanel({
  configText,
  disabled,
  className
}: RestApiValidationPanelProps) {
  const debouncedConfig = useDebounce(configText, 600)
  const [report, setReport] = useState<RestApiValidationReport | null>(null)
  const [validationError, setValidationError] = useState<string | null>(null)
  const [isValidating, setIsValidating] = useState(false)
  const validationAbortRef = useRef<AbortController | null>(null)

  const [connectionResult, setConnectionResult] = useState<ConnectionTestResponse | null>(null)
  const [connectionError, setConnectionError] = useState<string | null>(null)
  const [isTestingConnection, setIsTestingConnection] = useState(false)

  const [toolDialogOpen, setToolDialogOpen] = useState(false)
  const [activeTool, setActiveTool] = useState<ToolDefinition | null>(null)
  const [toolParamInputs, setToolParamInputs] = useState<Record<string, string>>({})
  const [toolTestResult, setToolTestResult] = useState<ToolTestResponse | null>(null)
  const [toolTestError, setToolTestError] = useState<string | null>(null)
  const [isTestingTool, setIsTestingTool] = useState(false)

  const parsedTools = report?.parsedConfig?.tools || []
  const parsedConfig = report?.parsedConfig

  useEffect(() => {
    if (!debouncedConfig || !debouncedConfig.trim()) {
      validationAbortRef.current?.abort()
      setReport(null)
      setValidationError(null)
      return
    }

    let mounted = true
    const controller = new AbortController()
    validationAbortRef.current = controller

    const runValidation = async () => {
      setIsValidating(true)
      setValidationError(null)

      try {
        const response = await fetch('/api/validate-config', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({ config: debouncedConfig }),
          signal: controller.signal
        })
        const data = await response.json()

        if (!mounted) return

        if (!response.ok || !data.success) {
          throw new Error(data.error || 'Validation failed')
        }

        setReport(data.report as RestApiValidationReport)
      } catch (error: any) {
        if (error?.name === 'AbortError') {
          return
        }
        setReport(null)
        setValidationError(error?.message || 'Unable to validate configuration.')
      } finally {
        if (mounted) {
          setIsValidating(false)
        }
      }
    }

    runValidation()

    return () => {
      mounted = false
      controller.abort()
    }
  }, [debouncedConfig])

  const statusConfig = validationStatusConfig[report?.summary || 'idle']
  const StatusIcon = statusConfig.icon

  const handleTestConnection = useCallback(async () => {
    if (!parsedConfig || disabled) return

    setIsTestingConnection(true)
    setConnectionError(null)
    setConnectionResult(null)

    try {
      const response = await fetch('/api/test-connection', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ apiConfig: parsedConfig })
      })
      const data = await response.json()

      if (!response.ok) {
        throw new Error(data.error || 'Connection test failed.')
      }

      setConnectionResult(data as ConnectionTestResponse)
      if (!data.success) {
        setConnectionError(data.error || data.message || 'Connection test failed.')
      } else {
        setConnectionError(null)
      }
    } catch (error: any) {
      setConnectionResult(null)
      setConnectionError(error?.message || 'Unable to test API connection.')
    } finally {
      setIsTestingConnection(false)
    }
  }, [parsedConfig, disabled])

  const openToolDialog = useCallback(
    (tool: ToolDefinition) => {
      if (!parsedConfig) return
      setActiveTool(tool)

      const initialValues: Record<string, string> = {}
      tool.parameters.forEach((param) => {
        if (param.default !== undefined && param.default !== null) {
          if (typeof param.default === 'object') {
            initialValues[param.name] = JSON.stringify(param.default, null, 2)
          } else {
            initialValues[param.name] = String(param.default)
          }
        } else {
          initialValues[param.name] = ''
        }
      })
      setToolParamInputs(initialValues)
      setToolTestResult(null)
      setToolTestError(null)
      setToolDialogOpen(true)
    },
    [parsedConfig]
  )

  const handleToolParamChange = useCallback((name: string, value: string) => {
    setToolParamInputs((prev) => ({
      ...prev,
      [name]: value
    }))
  }, [])

  const handleTestTool = useCallback(async () => {
    if (!parsedConfig || !activeTool) return
    setIsTestingTool(true)
    setToolTestError(null)
    setToolTestResult(null)

    try {
      const sanitizedParams: Record<string, string> = {}
      for (const [key, value] of Object.entries(toolParamInputs)) {
        if (value !== undefined && value !== null && value !== '') {
          sanitizedParams[key] = value
        }
      }

      const response = await fetch('/api/test-tool', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          apiConfig: parsedConfig,
          tool: activeTool,
          testParams: sanitizedParams
        })
      })
      const data = await response.json()

      if (!response.ok) {
        throw new Error(data.error || data.message || 'Tool test failed.')
      }

      setToolTestResult(data as ToolTestResponse)
      if (!data.success) {
        setToolTestError(data.error || data.message || 'Tool test failed.')
      } else {
        setToolTestError(null)
      }
    } catch (error: any) {
      setToolTestResult(null)
      setToolTestError(error?.message || 'Unable to execute tool test.')
    } finally {
      setIsTestingTool(false)
    }
  }, [parsedConfig, activeTool, toolParamInputs])

  useEffect(() => {
    if (!toolDialogOpen) {
      setActiveTool(null)
      setToolParamInputs({})
      setToolTestResult(null)
      setToolTestError(null)
      setIsTestingTool(false)
    }
  }, [toolDialogOpen])

  const canTestConnection = !!parsedConfig && !report?.openApiDetected && !disabled && !isValidating
  const hasTools = parsedTools.length > 0

  return (
    <div className={cn('space-y-4', className)}>
      <Card>
        <CardHeader className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <CardTitle className="flex items-center gap-2 text-base">
              <PlugZap className="h-4 w-4 text-primary" />
              Configuration Validation
            </CardTitle>
            <p className="text-sm text-muted-foreground">
              Automatic checks for JSON/YAML parsing, schema validation, and required fields.
            </p>
          </div>
          <Badge
            variant="outline"
            className={cn(
              'text-[11px] font-semibold px-3 py-1 rounded-full flex items-center gap-1.5 shadow-sm',
              statusConfig.badgeClass
            )}
          >
            {isValidating ? (
              <>
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                Validating...
              </>
            ) : (
              <>
                <StatusIcon className="h-3.5 w-3.5" />
                {statusConfig.label}
              </>
            )}
          </Badge>
        </CardHeader>
        <CardContent className="space-y-4">
          {validationError && (
            <div className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 dark:bg-red-950 dark:border-red-800 p-3 text-sm text-red-800 dark:text-red-200">
              <AlertTriangle className="h-4 w-4 mt-0.5" />
              <span>{validationError}</span>
            </div>
          )}

          {!validationError && report && (
            <div className="space-y-3">
              {report.openApiDetected && (
                <div className="flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 dark:bg-amber-950 dark:border-amber-800 p-3 text-sm text-amber-800 dark:text-amber-200">
                  <AlertCircle className="h-4 w-4 mt-0.5" />
                  <span>
                    Detected an OpenAPI specification. Convert it to the REST API configuration format before running tests
                    or saving.
                  </span>
                </div>
              )}

              <div className="space-y-2">
                {report.checks.map((check) => (
                  <div
                    key={check.id}
                    className={cn(
                      'rounded-md border p-3 text-sm',
                      check.status === 'pass' && 'border-green-200 bg-green-50 text-green-800 dark:border-green-800 dark:bg-green-950 dark:text-green-200',
                      check.status === 'warn' && 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200',
                      check.status === 'error' && 'border-red-200 bg-red-50 text-red-800 dark:border-red-800 dark:bg-red-950 dark:text-red-200'
                    )}
                  >
                    <div className="flex items-center gap-2 font-medium">
                      {check.status === 'pass' && <CheckCircle2 className="h-4 w-4" />}
                      {check.status === 'warn' && <AlertTriangle className="h-4 w-4" />}
                      {check.status === 'error' && <AlertCircle className="h-4 w-4" />}
                      <span>{check.label}</span>
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">{check.message}</p>
                    {check.details && (
                      <ul className="mt-2 list-inside list-disc text-xs text-muted-foreground">
                        {check.details.map((detail, index) => (
                          <li key={index}>{detail}</li>
                        ))}
                      </ul>
                    )}
                  </div>
                ))}
              </div>

              {report.environmentPlaceholders.length > 0 && (
                <div className="rounded-md border border-blue-200 bg-blue-50 dark:bg-blue-950 dark:border-blue-800 p-3 text-sm text-blue-900 dark:text-blue-200">
                  <p className="font-semibold">Detected placeholders:</p>
                  <p className="text-xs">
                    Replace these values with actual secrets before saving: {report.environmentPlaceholders.join(', ')}
                  </p>
                </div>
              )}

              <div className="flex flex-wrap items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleTestConnection()}
                  disabled={!canTestConnection || isTestingConnection}
                >
                  {isTestingConnection ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Test connection
                    </>
                  ) : (
                    <>
                      <PlugZap className="mr-2 h-4 w-4" />
                      Test connection
                    </>
                  )}
                </Button>
                {report.checkedAt && (
                  <p className="text-xs text-muted-foreground">
                    {`Last checked ${new Date(report.checkedAt).toLocaleTimeString()}`}
                  </p>
                )}
              </div>

              {connectionResult && (
                <div
                  className={cn(
                    'rounded-md border p-4 text-sm',
                    connectionResult.success
                      ? 'border-green-200 bg-green-50 text-green-900 dark:border-green-800 dark:bg-green-950 dark:text-green-200'
                      : 'border-red-200 bg-red-50 text-red-900 dark:border-red-800 dark:bg-red-950 dark:text-red-200'
                  )}
                >
                  <div className="flex items-center gap-2 font-semibold">
                    {connectionResult.success ? (
                      <CheckCircle2 className="h-4 w-4" />
                    ) : (
                      <AlertTriangle className="h-4 w-4" />
                    )}
                    <span>
                      {connectionResult.message || (connectionResult.success ? 'Connection successful.' : 'Connection failed.')}
                    </span>
                  </div>
                  <div className="mt-2 grid gap-1 text-xs">
                    {connectionResult.method && (
                      <div>
                        Method: <span className="font-mono uppercase">{connectionResult.method}</span>
                      </div>
                    )}
                    {connectionResult.statusCode !== undefined && <div>Status Code: {connectionResult.statusCode}</div>}
                    {connectionResult.responseTime !== undefined && <div>Response Time: {connectionResult.responseTime} ms</div>}
                    {connectionResult.auth && (
                      <div>
                        Auth ({connectionResult.auth.type}):{' '}
                        <span className={connectionResult.auth.passed ? 'text-green-700 dark:text-green-400' : 'text-red-700 dark:text-red-400'}>
                          {connectionResult.auth.passed ? 'Valid' : 'Failed'}
                        </span>
                      </div>
                    )}
                  </div>
                  {connectionResult.response?.bodyPreview && (
                    <Textarea
                      readOnly
                      value={connectionResult.response.bodyPreview}
                      className="mt-3 h-32 resize-none font-mono text-xs"
                    />
                  )}
                </div>
              )}
              {connectionError && <p className="text-xs text-red-600 dark:text-red-400">{connectionError}</p>}
            </div>
          )}

          {!validationError && !report && !isValidating && (
            <p className="text-sm text-muted-foreground">Start typing your REST API configuration to see validation results.</p>
          )}
        </CardContent>
      </Card>

      {hasTools && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Wrench className="h-4 w-4 text-primary" />
              Tools ({parsedTools.length})
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {parsedTools.map((tool) => (
              <div key={tool.name} className="rounded-md border border-border p-3">
                <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <div className="flex items-center gap-2 text-sm font-semibold">
                      <Badge variant="outline" className="text-[0.65rem] uppercase">
                        {tool.method}
                      </Badge>
                      <span>{tool.name}</span>
                    </div>
                    <p className="font-mono text-xs text-muted-foreground">{tool.endpoint}</p>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={!parsedConfig || disabled}
                    onClick={() => openToolDialog(tool)}
                  >
                    Test tool
                  </Button>
                </div>
                {tool.description && <p className="mt-2 text-xs text-muted-foreground">{tool.description}</p>}
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      <Dialog open={toolDialogOpen} onOpenChange={setToolDialogOpen}>
        <DialogContent className="max-h-[90vh] overflow-hidden sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-base">
              <Wrench className="h-4 w-4 text-primary" />
              Test tool
            </DialogTitle>
            <DialogDescription>
              Provide sample parameters and run the tool against your API to verify the response.
            </DialogDescription>
          </DialogHeader>

          <ScrollArea className="max-h-[60vh] pr-4">
            {activeTool ? (
              <div className="space-y-4 pb-4">
                <div className="rounded-md border border-border p-3 text-sm">
                  <div className="flex items-center gap-2 font-semibold">
                    <Badge variant="outline" className="text-[0.65rem] uppercase">
                      {activeTool.method}
                    </Badge>
                  <span>{activeTool.name}</span>
                </div>
                <p className="font-mono text-xs text-muted-foreground">{activeTool.endpoint}</p>
                {activeTool.description && <p className="mt-1 text-xs text-muted-foreground">{activeTool.description}</p>}
              </div>

              <div className="max-h-64 overflow-y-auto rounded-md border border-dashed border-border p-3">
                <div className="space-y-3">
                  {activeTool.parameters.length === 0 && (
                    <p className="text-xs text-muted-foreground">This tool does not require any parameters.</p>
                  )}
                  {activeTool.parameters.map((param) => (
                    <div key={param.name} className="space-y-1">
                      <Label className="text-xs font-semibold">
                        {param.name}{' '}
                        {param.required && <span className="text-red-500 dark:text-red-400">*</span>}
                        <span className="ml-1 rounded bg-muted px-1 text-[0.65rem] uppercase">{param.location}</span>
                      </Label>
                      <Input
                        value={toolParamInputs[param.name] ?? ''}
                        onChange={(event) => handleToolParamChange(param.name, event.target.value)}
                        placeholder={getToolParamPlaceholder(param)}
                      />
                      <p className="text-[0.65rem] text-muted-foreground">Type: {param.type}</p>
                    </div>
                  ))}
                </div>
              </div>

              {toolTestError && (
                <div className="flex items-start gap-2 rounded-md border border-red-200 bg-red-50 dark:bg-red-950 dark:border-red-800 p-3 text-sm text-red-800 dark:text-red-200">
                  <AlertTriangle className="h-4 w-4 mt-0.5" />
                  <span>{toolTestError}</span>
                </div>
              )}

              {toolTestResult && (
                <div
                  className={cn(
                    'rounded-md border p-3 text-sm',
                    toolTestResult.success
                      ? 'border-green-200 bg-green-50 text-green-900 dark:border-green-800 dark:bg-green-950 dark:text-green-200'
                      : 'border-red-200 bg-red-50 text-red-900 dark:border-red-800 dark:bg-red-950 dark:text-red-200'
                  )}
                >
                  <div className="flex items-center gap-2 font-semibold">
                    {toolTestResult.success ? (
                      <CheckCircle2 className="h-4 w-4" />
                    ) : (
                      <AlertTriangle className="h-4 w-4" />
                    )}
                    <span>
                      {toolTestResult.message ||
                        (toolTestResult.success ? 'Tool executed.' : 'Tool execution failed.')}
                    </span>
                  </div>
                  <div className="mt-2 grid gap-1 text-xs">
                    <div>Status: {toolTestResult.statusCode}</div>
                    <div>Response Time: {toolTestResult.responseTime} ms</div>
                  </div>
                  {toolTestResult.response?.body && (
                    <Textarea
                      readOnly
                      value={
                        typeof toolTestResult.response.body === 'string'
                          ? toolTestResult.response.body
                          : JSON.stringify(toolTestResult.response.body, null, 2)
                      }
                      className="mt-3 h-40 resize-none font-mono text-xs"
                    />
                  )}
                  {toolTestResult.transformedResponse !== undefined && (
                    <div className="mt-2 rounded-md border border-dashed border-green-300 dark:border-green-700 bg-background p-2 text-xs text-green-900 dark:text-green-200">
                      <p className="font-semibold">Transformed Response</p>
                      <pre className="mt-1 whitespace-pre-wrap break-all">
                        {JSON.stringify(toolTestResult.transformedResponse, null, 2)}
                      </pre>
                    </div>
                  )}
                </div>
              )}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground pb-4">Select a tool from the list to start testing.</p>
          )}
          </ScrollArea>

          <DialogFooter className="mt-4">
            <Button onClick={handleTestTool} disabled={!activeTool || isTestingTool}>
              {isTestingTool ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Execute tool
                </>
              ) : (
                'Execute tool'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
