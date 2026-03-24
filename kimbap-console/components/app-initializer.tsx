"use client"

export function AppInitializer({ children }: { children: React.ReactNode }) {
  // Backend now handles automatic reconnection after login
  // No need for client-side reconnection logic
  return <>{children}</>
}