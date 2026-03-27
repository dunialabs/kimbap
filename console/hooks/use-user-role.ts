'use client'

import { useState, useEffect } from 'react'

const safeStorageGet = (key: string): string | null => {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

export function useUserRole() {
  const [userRole, setUserRole] = useState<string | null>(null)

  useEffect(() => {
    const getUserRole = (): string => {
      if (typeof window !== 'undefined') {
        const storedServer = safeStorageGet('selectedServer')
        if (storedServer) {
          try {
            const parsedServer = JSON.parse(storedServer)
            if (parsedServer.role) return parsedServer.role
          } catch {
          }
        }
      }
      return 'Member'
    }

    setUserRole(getUserRole())
  }, [])

  const isOwner = userRole === 'Owner'
  const isAdmin = userRole === 'Admin'
  const isMember = userRole === 'Member' || userRole === null

  return {
    userRole,
    isOwner,
    isAdmin,
    isMember
  }
}
