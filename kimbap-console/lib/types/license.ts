/**
 * License record returned by protocol-10020 (getHistory).
 * Shared between the API handler and the billing UI.
 */
export interface License {
  plan: string
  status: number // 100: active, 1: inactive, 2: expired
  expiresAt: number
  createdAt: number
  customerEmail: string
  licenseStr: string
  maxToolCreations: number
  maxAccessTokens: number
  currentToolCreations: number
  currentAccessTokens: number
}

/** Real-time usage + limits returned by protocol 10025. */
export interface LicenseLimits {
  isFreeTier: boolean
  expiresAt: number
  maxToolCreations: number
  currentToolCount: number
  remainingToolCount: number
  maxAccessTokens: number
  currentAccessTokenCount: number
  remainingAccessTokenCount: number
  licenseKey: string
  fingerprintHash: string
}

/** Client-side state for the license page. */
export interface LicensePageState {
  activeLicense?: License
  licenseHistory: License[]
  isLoading: boolean
  error?: string
}
