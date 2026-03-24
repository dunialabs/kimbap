/**
 * Secure master password storage utility
 * Uses session storage for temporary storage during the session
 */

const MASTER_PASSWORD_KEY = 'mcp_master_pwd'
const SESSION_TIMEOUT = 30 * 60 * 1000 // 30 minutes in milliseconds

interface StoredMasterPassword {
  password: string
  timestamp: number
}

/**
 * Store master password securely in session storage with timestamp
 */
export function storeMasterPassword(password: string): void {
  if (typeof window === 'undefined') return

  const data: StoredMasterPassword = {
    password,
    timestamp: Date.now()
  }

  try {
    sessionStorage.setItem(MASTER_PASSWORD_KEY, JSON.stringify(data))
  } catch (error) {
    console.error('Failed to store master password:', error)
  }
}

/**
 * Retrieve master password from session storage
 * Returns null if not found or expired
 */
export function getMasterPassword(): string | null {
  if (typeof window === 'undefined') return null

  try {
    const stored = sessionStorage.getItem(MASTER_PASSWORD_KEY)
    if (!stored) return null

    const data: StoredMasterPassword = JSON.parse(stored)
    
    // Check if password has expired (30 minutes)
    if (Date.now() - data.timestamp > SESSION_TIMEOUT) {
      clearMasterPassword()
      return null
    }

    return data.password
  } catch (error) {
    console.error('Failed to retrieve master password:', error)
    clearMasterPassword()
    return null
  }
}

/**
 * Clear stored master password
 */
export function clearMasterPassword(): void {
  if (typeof window === 'undefined') return

  try {
    sessionStorage.removeItem(MASTER_PASSWORD_KEY)
  } catch (error) {
    console.error('Failed to clear master password:', error)
  }
}

/**
 * Check if master password is currently stored and valid
 */
export function hasMasterPassword(): boolean {
  return getMasterPassword() !== null
}

/**
 * Update the timestamp of stored master password to extend session
 */
export function refreshMasterPasswordSession(): void {
  const password = getMasterPassword()
  if (password) {
    storeMasterPassword(password)
  }
}