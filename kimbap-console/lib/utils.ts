import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
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