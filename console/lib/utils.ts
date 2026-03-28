import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

const compactNumberFormatter = new Intl.NumberFormat(undefined, {
  notation: 'compact',
  maximumFractionDigits: 1
})

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDisplayNumber(
  value: number | null | undefined,
  options: { compact?: boolean } = {}
): string {
  if (value == null || !Number.isFinite(value)) return '—'
  if (options.compact && Math.abs(value) >= 1_000_000) {
    return compactNumberFormatter.format(value)
  }
  return value.toLocaleString()
}

export function formatPercentage(
  value: number | null | undefined,
  digits = 1
): string {
  if (value == null || !Number.isFinite(value)) return '—'
  return `${value.toFixed(digits)}%`
}

export function formatNullableText(value: string | null | undefined): string {
  const normalized = value?.trim()
  return normalized ? normalized : '—'
}

export function formatResponseTime(ms: number | null | undefined): string {
  if (ms == null || !Number.isFinite(ms)) return '—'
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`
  return `${Math.round(ms)}ms`
}

export function formatRelativeMinutes(totalMinutes: number | null | undefined): string {
  if (totalMinutes == null || !Number.isFinite(totalMinutes)) return '—'

  const minutes = Math.max(0, Math.floor(totalMinutes))

  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes} minute${minutes === 1 ? '' : 's'} ago`

  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} hour${hours === 1 ? '' : 's'} ago`

  const days = Math.floor(hours / 24)
  return `${days} day${days === 1 ? '' : 's'} ago`
}

export function formatDateTime(
  value: string | number | Date | null | undefined,
  options: Intl.DateTimeFormatOptions = {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }
): string {
  if (value == null) return '—'

  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) {
    return typeof value === 'string' ? formatNullableText(value) : '—'
  }

  return date.toLocaleString(undefined, options)
}

// Get gateway URL from Electron or use default
export function getGatewayUrl(): string {
  if (typeof window !== 'undefined' && (window as any).electronAPI) {
    return (window as any).electronAPI.getGatewayURL()
  }
  if (typeof window !== 'undefined' && (window as any).GATEWAY_URL) {
    return (window as any).GATEWAY_URL
  }
  return 'http://localhost:3002'
}
