'use client'

import React, { useState } from 'react'
import { Play, AlertCircle, CheckCircle, Wrench } from 'lucide-react'
import axios from 'axios'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Badge } from './ui/badge'
import { cn } from '@/lib/utils'
import type { Host, Tool, ToolResult } from '@/lib/types'

interface ToolTesterProps {
  host: Host
  gatewayUrl: string
}

export function ToolTester({ host, gatewayUrl }: ToolTesterProps) {
  const [selectedTool, setSelectedTool] = useState<Tool | null>(null)
  const [params, setParams] = useState<Record<string, any>>({})
  const [response, setResponse] = useState<ToolResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleToolSelect = (tool: Tool) => {
    setSelectedTool(tool)
    setParams({})
    setResponse(null)
    setError(null)
  }

  const handleParamChange = (paramName: string, value: string) => {
    setParams((prev) => ({
      ...prev,
      [paramName]: value
    }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedTool) return

    setLoading(true)
    setError(null)
    setResponse(null)

    try {
      const result = await axios.post(
        `${gatewayUrl}/api/hosts/${host.hostId}/tools/${selectedTool.name}`,
        params
      )
      setResponse(result.data)
    } catch (err: any) {
      setError(err.response?.data?.error || err.message)
    } finally {
      setLoading(false)
    }
  }

  const getParameterEntries = (tool: Tool) => {
    if (tool.inputSchema?.properties) {
      return Object.entries(tool.inputSchema.properties)
    }
    // Fallback for old format
    if ((tool as any).parameters) {
      return Object.entries((tool as any).parameters)
    }
    return []
  }

  const isParameterRequired = (paramName: string, tool: Tool) => {
    return tool.inputSchema?.required?.includes(paramName) || false
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Wrench className="h-5 w-5" />
          Tool Tester
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Available Tools */}
        <div>
          <h3 className="text-sm font-medium mb-3">
            Available Tools ({host.tools.length})
          </h3>
          <div className="grid grid-cols-1 gap-2">
            {host.tools.map((tool) => (
              <button
                key={tool.name}
                onClick={() => handleToolSelect(tool)}
                className={cn(
                  'text-left p-3 rounded-md border transition-colors',
                  selectedTool?.name === tool.name
                    ? 'border-primary bg-primary/5'
                    : 'border-border hover:border-primary/50 hover:bg-muted/50'
                )}
              >
                <div className="font-medium flex items-center gap-2">
                  <Play className="h-4 w-4" />
                  {tool.name}
                </div>
                <div className="text-sm text-muted-foreground mt-1">
                  {tool.description}
                </div>
              </button>
            ))}
          </div>
        </div>

        {/* Tool Form */}
        {selectedTool && (
          <div>
            <div className="flex items-center gap-2 mb-4">
              <h4 className="text-lg font-medium">Test {selectedTool.name}</h4>
              <Badge variant="outline">
                {getParameterEntries(selectedTool).length} parameters
              </Badge>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              {getParameterEntries(selectedTool).map(
                ([paramName, paramInfo]: [string, any]) => (
                  <div key={paramName}>
                    <label className="block text-sm font-medium mb-2">
                      {paramName}
                      {isParameterRequired(paramName, selectedTool) && (
                        <span className="text-destructive ml-1">*</span>
                      )}
                    </label>
                    <Input
                      type="text"
                      value={params[paramName] || ''}
                      onChange={(e) =>
                        handleParamChange(paramName, e.target.value)
                      }
                      placeholder={
                        paramInfo.description || `Enter ${paramName}...`
                      }
                      required={isParameterRequired(paramName, selectedTool)}
                    />
                    {paramInfo.description && (
                      <p className="text-xs text-muted-foreground mt-1">
                        {paramInfo.description}
                      </p>
                    )}
                  </div>
                )
              )}

              <Button type="submit" disabled={loading} className="w-full">
                {loading ? (
                  <>
                    <div className="mr-2 h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
                    Executing...
                  </>
                ) : (
                  <>
                    <Play className="mr-2 h-4 w-4" />
                    Execute Tool
                  </>
                )}
              </Button>
            </form>
          </div>
        )}

        {/* Error Display */}
        {error && (
          <div className="flex items-start gap-3 p-4 bg-destructive/10 border border-destructive/20 rounded-md">
            <AlertCircle className="h-5 w-5 text-destructive mt-0.5 flex-shrink-0" />
            <div>
              <p className="font-medium text-destructive">Error</p>
              <p className="text-sm text-destructive/90">{error}</p>
            </div>
          </div>
        )}

        {/* Response Display */}
        {response && (
          <div>
            <div className="flex items-center gap-2 mb-3">
              <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400" />
              <h4 className="text-lg font-medium">Response</h4>
            </div>
            <div className="bg-muted p-4 rounded-md">
              <pre className="text-sm overflow-x-auto whitespace-pre-wrap break-words">
                {JSON.stringify(response, null, 2)}
              </pre>
            </div>
          </div>
        )}

        {/* No Tool Selected */}
        {!selectedTool && host.tools.length > 0 && (
          <div className="text-center py-8">
            <Wrench className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground">Select a tool to test it</p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
