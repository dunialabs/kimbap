'use client'

import { useState, useEffect, useCallback } from 'react'
import axios from 'axios'
import { getGatewayUrl } from '@/lib/utils'
import type { Host, ApiResponse } from '@/lib/types'

export function useHosts() {
  const [hosts, setHosts] = useState<Host[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchHosts = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await axios.get<ApiResponse<{ hosts: Host[] }>>(
        `${getGatewayUrl()}/api/hosts`
      )

      if (response.data.hosts) {
        // Add status based on heartbeat
        const hostsWithStatus = response.data.hosts.map(host => ({
          ...host,
          status: isHostOnline(host.lastHeartbeat) ? 'online' : 'offline'
        })) as Host[]

        setHosts(hostsWithStatus)
      } else {
        setHosts([])
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error'
      setError(`Failed to fetch hosts: ${errorMessage}`)
      setHosts([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchHosts()
    // Refresh hosts every 5 seconds
    const interval = setInterval(fetchHosts, 5000)
    return () => clearInterval(interval)
  }, [fetchHosts])

  return {
    hosts,
    loading,
    error,
    refetch: fetchHosts
  }
}

function isHostOnline(lastHeartbeat: string): boolean {
  const heartbeatTime = new Date(lastHeartbeat).getTime()
  const now = Date.now()
  const fiveMinutes = 5 * 60 * 1000
  return (now - heartbeatTime) < fiveMinutes
}