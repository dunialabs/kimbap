"use client"

import { AppInitializer } from '@/components/app-initializer'

export function ConditionalLayout({ children }: { children: React.ReactNode }) {
  // Wrap all content with AppInitializer to handle service reconnection on app restart
  return (
    <AppInitializer>
      {children}
    </AppInitializer>
  )
}