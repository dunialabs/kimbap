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
  const [userRole, setUserRole] = useState<string>('Member')

  useEffect(() => {
    const getUserRole = (): string => {
      if (typeof window !== 'undefined') {
        const storedServer = safeStorageGet('selectedServer')
        if (storedServer) {
          try {
            const parsedServer = JSON.parse(storedServer)
            return parsedServer.role || 'Member'
          } catch {
          }
        }
      }
      return 'Member' // Default to Member if not found
    }

    setUserRole(getUserRole())
  }, [])

  const isOwner = userRole === 'Owner'
  const isAdmin = userRole === 'Admin'
  const isMember = userRole === 'Member'

  return {
    userRole,
    isOwner,
    isAdmin,
    isMember
  }
}
