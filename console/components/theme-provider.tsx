'use client'

import { useEffect } from 'react'

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    // Function to update theme based on system preference
    const updateTheme = (e?: MediaQueryListEvent | MediaQueryList) => {
      const isDark = e
        ? e.matches
        : window.matchMedia('(prefers-color-scheme: dark)').matches

      if (isDark) {
        document.documentElement.classList.add('dark')
      } else {
        document.documentElement.classList.remove('dark')
      }
    }

    // Check initial system preference
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    updateTheme(mediaQuery)

    // Listen for changes to system preference
    const listener = (e: MediaQueryListEvent) => updateTheme(e)
    mediaQuery.addEventListener('change', listener)

    return () => {
      mediaQuery.removeEventListener('change', listener)
    }
  }, [])

  return <>{children}</>
}
