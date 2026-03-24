import { Shield, Crown } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

// ─── Canonical plan identifiers ─────────────────────────────────────────────
export type PlanId = 'free' | 'pro' | 'enterprise'

// ─── Display configuration used across all license/billing surfaces ─────────
export interface PlanDisplayConfig {
  name: string
  icon: LucideIcon
  color: string
}

export const PLAN_DISPLAY: Record<PlanId, PlanDisplayConfig> = {
  free: {
    name: 'Community',
    icon: Shield,
    color:
      'bg-gray-100 text-gray-800 border-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700',
  },
  pro: {
    name: 'Business',
    icon: Crown,
    color:
      'bg-purple-100 text-purple-800 border-purple-200 dark:bg-purple-900 dark:text-purple-300 dark:border-purple-800',
  },
  enterprise: {
    name: 'Enterprise',
    icon: Crown,
    color:
      'bg-amber-100 text-amber-800 border-amber-200 dark:bg-amber-900 dark:text-amber-300 dark:border-amber-800',
  },
}

/**
 * Map backend planLevel identifiers to canonical PlanId.
 *
 * The encrypted license stores `planLevel` as defined by kimbap.io (e.g. "lv1",
 * "lv2") or descriptive names like "pro", "enterprise".  This function
 * normalises every known variant to a canonical PlanId.
 */
export function resolvePlanId(planLevel: string | undefined): PlanId {
  if (!planLevel) return 'free'
  const v = planLevel.toLowerCase().trim()

  // Descriptive names
  if (v === 'free' || v === 'community') return 'free'
  if (v === 'plus' || v === 'standard' || v === 'pro' || v === 'professional' || v === 'business') return 'pro'
  if (v === 'enterprise') return 'enterprise'

  // Backend level identifiers (defined by kimbap.io website)
  if (v === 'lv1' || v === 'lv2' || v === 'lv3') return 'pro'
  if (v === 'lv4') return 'enterprise'

  return 'free'
}

/** Build kimbap.io pricing URL, optionally pre-filling the machine fingerprint. */
export function getPricingUrl(fingerprint?: string): string {
  const baseUrl = 'https://kimbap.io/pricing'
  if (fingerprint) {
    return `${baseUrl}?fingerprint=${encodeURIComponent(fingerprint)}`
  }
  return baseUrl
}
