"use client"

import { useState, useCallback, useRef, useEffect } from "react"

interface UseAsyncState<T> {
  data: T | null
  error: Error | null
  loading: boolean
}

interface UseAsyncReturn<T, Args extends any[]> extends UseAsyncState<T> {
  execute: (...args: Args) => Promise<T | null>
  reset: () => void
}

export function useAsync<T = any, Args extends any[] = any[]>(
  asyncFunction: (...args: Args) => Promise<T>
): UseAsyncReturn<T, Args> {
  const [state, setState] = useState<UseAsyncState<T>>({
    data: null,
    error: null,
    loading: false,
  })

  // Track if component is mounted to prevent state updates after unmount
  const isMountedRef = useRef(true)

  useEffect(() => {
    return () => {
      isMountedRef.current = false
    }
  }, [])

  const execute = useCallback(
    async (...args: Args) => {
      setState({ data: null, error: null, loading: true })

      try {
        const data = await asyncFunction(...args)
        
        if (isMountedRef.current) {
          setState({ data, error: null, loading: false })
        }
        
        return data
      } catch (error) {
        const errorObj = error instanceof Error ? error : new Error(String(error))
        
        if (isMountedRef.current) {
          setState({ data: null, error: errorObj, loading: false })
        }
        
        return null
      }
    },
    [asyncFunction]
  )

  const reset = useCallback(() => {
    setState({ data: null, error: null, loading: false })
  }, [])

  return {
    ...state,
    execute,
    reset,
  }
}