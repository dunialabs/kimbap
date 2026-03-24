'use client'

import React from 'react'
import { RefreshCw, Server, Activity } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card'
import { Button } from './ui/button'
import { Badge } from './ui/badge'
import { cn } from '@/lib/utils'
import type { Host } from '@/lib/types'

interface HostListProps {
  hosts: Host[]
  loading: boolean
  selectedHost: Host | null
  onSelectHost: (host: Host) => void
  onRefresh: () => void
}

export function HostList({
  hosts,
  loading,
  selectedHost,
  onSelectHost,
  onRefresh
}: HostListProps) {
  if (loading && hosts.length === 0) {
    return (
      <Card>
        <CardContent className="p-6">
          <div className="animate-pulse space-y-4">
            <div className="h-4 bg-muted rounded w-3/4"></div>
            <div className="h-4 bg-muted rounded w-1/2"></div>
            <div className="h-4 bg-muted rounded w-2/3"></div>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (hosts.length === 0) {
    return (
      <Card>
        <CardContent className="p-6 text-center">
          <Server className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
          <p className="text-muted-foreground mb-4">No hosts registered</p>
          <Button onClick={onRefresh} variant="outline" size="sm">
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">
            Active Hosts ({hosts.length})
          </CardTitle>
          <Button
            onClick={onRefresh}
            variant="ghost"
            size="sm"
            disabled={loading}
            aria-label="Refresh hosts"
          >
            <RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} />
            <span className="sr-only">Refresh hosts</span>
          </Button>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        <div className="divide-y">
          {hosts.map((host) => (
            <button
              type="button"
              key={host.hostId}
              onClick={() => onSelectHost(host)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  onSelectHost(host)
                }
              }}
              aria-pressed={selectedHost?.hostId === host.hostId}
              className={cn(
                'w-full text-left p-4 cursor-pointer transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset',
                selectedHost?.hostId === host.hostId &&
                  'bg-muted border-l-4 border-primary'
              )}
            >
              <div className="flex justify-between items-start mb-2">
                <div className="flex items-center gap-2">
                  <Server className="h-4 w-4 text-muted-foreground" />
                  <h4 className="font-medium">{host.hostId}</h4>
                </div>
                <Badge
                  variant={host.status === 'online' ? 'default' : 'secondary'}
                  className="text-xs"
                >
                  <Activity className="mr-1 h-3 w-3" />
                  {host.status || 'online'}
                </Badge>
              </div>

              <p className="text-sm text-muted-foreground mb-3">
                {host.hostUrl}
              </p>

              <div className="flex flex-wrap gap-1 mb-3">
                {host.tools.map((tool) => (
                  <Badge key={tool.name} variant="outline" className="text-xs">
                    {tool.name}
                  </Badge>
                ))}
              </div>

              <p className="text-xs text-muted-foreground">
                Last heartbeat: {new Date(host.lastHeartbeat).toLocaleString()}
              </p>
            </button>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
