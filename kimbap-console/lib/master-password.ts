/**
 * Secure master password storage utility
 *
 * Uses session storage for temporary caching during the browser session.
 * The raw password is never stored directly — it is XOR-masked with an
 * ephemeral key held only in module-scope memory.  When the page unloads
 * or the JS context is torn down the ephemeral key is lost, making the
 * stored blob unrecoverable.  This prevents trivial extraction from
 * sessionStorage inspection (DevTools, extensions, XSS reading storage).
 *
 * Threat-model note: this does NOT protect against a same-origin attacker
 * that can execute arbitrary JS in the same page context, because such an
 * attacker could call getMasterPassword() directly.  Full protection would
 * require a Service Worker credential broker or native secure enclave.
 */

const MASTER_PASSWORD_KEY = 'mcp_master_pwd'
const SESSION_TIMEOUT = 30 * 60 * 1000 // 30 minutes in milliseconds

let ephemeralKey: Uint8Array | null = null

function getOrCreateEphemeralKey(): Uint8Array {
  if (!ephemeralKey) {
    ephemeralKey = new Uint8Array(64)
    crypto.getRandomValues(ephemeralKey)
  }
  return ephemeralKey
}

function xorMask(input: string, key: Uint8Array): string {
  const encoder = new TextEncoder()
  const bytes = encoder.encode(input)
  let chars = ''
  for (let i = 0; i < bytes.length; i++) {
    chars += String.fromCharCode(bytes[i] ^ key[i % key.length])
  }
  return btoa(chars)
}

function xorUnmask(encoded: string, key: Uint8Array): string {
  const raw = atob(encoded)
  const bytes = new Uint8Array(raw.length)
  for (let i = 0; i < raw.length; i++) {
    bytes[i] = raw.charCodeAt(i) ^ key[i % key.length]
  }
  return new TextDecoder().decode(bytes)
}

interface StoredMasterPassword {
  masked: string
  timestamp: number
}

/**
 * Store master password in session storage, XOR-masked with ephemeral key.
 */
export function storeMasterPassword(password: string): void {
  if (typeof window === 'undefined') return

  const key = getOrCreateEphemeralKey()
  const data: StoredMasterPassword = {
    masked: xorMask(password, key),
    timestamp: Date.now()
  }

  try {
    sessionStorage.setItem(MASTER_PASSWORD_KEY, JSON.stringify(data))
  } catch (error) {
    console.error('Failed to store master password:', error)
  }
}

/**
 * Retrieve master password from session storage.
 * Returns null if not found, expired, or ephemeral key was lost (page reload).
 */
export function getMasterPassword(): string | null {
  if (typeof window === 'undefined') return null
  if (!ephemeralKey) {
    clearMasterPassword()
    return null
  }

  try {
    const stored = sessionStorage.getItem(MASTER_PASSWORD_KEY)
    if (!stored) return null

    const data: StoredMasterPassword = JSON.parse(stored)

    if (Date.now() - data.timestamp > SESSION_TIMEOUT) {
      clearMasterPassword()
      return null
    }

    return xorUnmask(data.masked, ephemeralKey)
  } catch (error) {
    console.error('Failed to retrieve master password:', error)
    clearMasterPassword()
    return null
  }
}

/**
 * Clear stored master password and wipe the ephemeral key.
 */
export function clearMasterPassword(): void {
  if (typeof window === 'undefined') return

  try {
    sessionStorage.removeItem(MASTER_PASSWORD_KEY)
  } catch (error) {
    console.error('Failed to clear master password:', error)
  }
  if (ephemeralKey) {
    ephemeralKey.fill(0)
    ephemeralKey = null
  }
}

/**
 * Check if master password is currently stored and valid.
 */
export function hasMasterPassword(): boolean {
  return getMasterPassword() !== null
}

/**
 * Update the timestamp of stored master password to extend session.
 */
export function refreshMasterPasswordSession(): void {
  const password = getMasterPassword()
  if (password) {
    storeMasterPassword(password)
  }
}