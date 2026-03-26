# Contexts Directory Status

This directory no longer contains runtime React context providers.

## What changed

The previous global `MasterPasswordProvider` / `useMasterPassword` flow and
`MasterPasswordDialog` component were removed during auth-flow simplification.

Current behavior:

- Login and master-password setup are handled by page-local components
  (`components/login-form.tsx`, `components/master-password-form.tsx`)
- Dashboard layout does **not** wrap children with a master-password context
- No context hook from `@/contexts/*` is required for current auth flows

## Guidance for contributors

- Do not import `@/contexts/master-password-context` (file removed)
- Do not rely on a global master-password dialog singleton
- If a feature requires password confirmation, wire it explicitly in the feature
  component and use current API contracts

If a new shared context is introduced in the future, add it here and document it
with file path, usage example, and lifecycle constraints.
