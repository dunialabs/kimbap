/*
 * @Author: xudada 1820064201@qq.com
 * @Date: 2025-07-24 11:40:59
 * @LastEditors: xudada 1820064201@qq.com
 * @LastEditTime: 2025-07-24 11:45:51
 * @FilePath: /mcp-kimbap/mcp-desktop-app/frontend/app/page.tsx
 * @Description: 这是默认设置,请设置`customMade`, 打开koroFileHeader查看配置 进行设置: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%AE
 */
'use client'

import { useState } from 'react'
import { Zap, Network, AlertTriangle } from 'lucide-react'
import { HostList } from '@/components/host-list'
import { ToolTester } from '@/components/tool-tester'
import { useHosts } from '@/hooks/use-hosts'
import { getGatewayUrl } from '@/lib/utils'
import type { Host } from '@/lib/types'

export default function HomePage() {
  const { hosts, loading, error, refetch } = useHosts()
  const [selectedHost, setSelectedHost] = useState<Host | null>(null)

  return (
    <div className="container mx-auto px-4 py-8 space-y-8">
      {/* Header */}
      <header className="space-y-2">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <Zap className="h-8 w-8 text-primary" />
            <h1 className="text-4xl font-bold">MCP Desktop Application</h1>
          </div>
        </div>
        <p className="text-muted-foreground text-lg">
          Model Context Protocol Gateway Management
        </p>
      </header>

      {/* Global Error */}
      {error && (
        <div className="flex items-start gap-3 p-4 bg-destructive/10 border border-destructive/20 rounded-lg">
          <AlertTriangle className="h-5 w-5 text-destructive mt-0.5 flex-shrink-0" />
          <div>
            <p className="font-medium text-destructive">Connection Error</p>
            <p className="text-sm text-destructive/90">{error}</p>
          </div>
        </div>
      )}

      {/* Main Content */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Hosts Panel */}
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <Network className="h-6 w-6 text-muted-foreground" />
            <h2 className="text-2xl font-semibold">Registered Hosts</h2>
          </div>
          <HostList
            hosts={hosts}
            loading={loading}
            selectedHost={selectedHost}
            onSelectHost={setSelectedHost}
            onRefresh={refetch}
          />
        </div>

        {/* Tool Tester Panel */}
        <div className="space-y-4">
          <h2 className="text-2xl font-semibold">Tool Tester</h2>
          {selectedHost ? (
            <ToolTester host={selectedHost} gatewayUrl={getGatewayUrl()} />
          ) : (
            <div className="bg-card border rounded-lg p-8 text-center">
              <Network className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
              <p className="text-muted-foreground">
                Select a host from the left panel to test its tools
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Footer */}
      <footer className="border-t pt-8 mt-12">
        <div className="flex items-center justify-between text-sm text-muted-foreground">
          <p>
            Gateway URL:{' '}
            <code className="bg-muted px-2 py-1 rounded text-xs">
              {getGatewayUrl()}
            </code>
          </p>
          <p>
            {hosts.length} host{hosts.length !== 1 ? 's' : ''} connected
          </p>
        </div>
      </footer>
    </div>
  )
}
