# Global Master Password Dialog Guide

## Overview
A single Master Password dialog is provided via Context/Provider so every page reuses the same modal by passing different titles, descriptions, and callbacks.

## Features
- ✅ Global singleton dialog to avoid duplicates
- ✅ Automatically shows "Forgot Password" for Owners only
- ✅ Supports async actions and loading state
- ✅ Error handling keeps the dialog open for retries

## Usage

### 1. Ensure the Provider is mounted
`app/(dashboard)/dashboard/layout.tsx` already wraps children with `MasterPasswordProvider`:

```tsx
import { MasterPasswordProvider } from '@/contexts/master-password-context'

export default function DashboardLayout({ children }) {
  return (
    <MasterPasswordProvider>
      {children}
    </MasterPasswordProvider>
  )
}
```

### 2. Use in a component

```tsx
import { useMasterPassword } from '@/contexts/master-password-context'

export default function YourPage() {
  const { requestMasterPassword } = useMasterPassword()

  const handleSomeAction = async () => {
    // Check cached master password first
    const cachedPassword = MasterPasswordManager.getCachedMasterPassword()
    if (cachedPassword) {
      await performAction(cachedPassword)
      return
    }

    // Show the master password dialog
    requestMasterPassword({
      title: 'Your Action - Master Password Required',
      description: 'Please enter your master password to perform this action.',
      onConfirm: async (password) => {
        await performAction(password)
      }
    })
  }

  return <button onClick={handleSomeAction}>Do Something</button>
}
```

### 3. API

#### `requestMasterPassword(options)`
Display the master password dialog.

**Options:**
- `options.title`: Dialog title (optional, default "Master Password Required")
- `options.description`: Dialog description (optional, default "Please enter your master password to continue.")
- `options.onConfirm`: Confirm callback that receives the password, supports async

**Examples:**

```tsx
// Simple example
requestMasterPassword({
  title: 'Delete Tool',
  description: 'Enter your master password to delete this tool.',
  onConfirm: async (password) => {
    await api.tools.delete({ toolId, masterPwd: password })
    toast.success('Tool deleted')
  }
})

// With error handling
requestMasterPassword({
  title: 'Update Settings',
  description: 'Enter your master password to update settings.',
  onConfirm: async (password) => {
    try {
      const response = await api.settings.update({ masterPwd: password, ...data })
      if (!response.success) {
        throw new Error('Update failed')
      }
      toast.success('Settings updated')
    } catch (error) {
      toast.error('Failed to update settings')
      throw error // Re-throw to keep dialog open for retry
    }
  }
})
```

## Migration Guide

### From local MasterPasswordDialog

#### Previous code:
```tsx
const [showMasterPasswordDialog, setShowMasterPasswordDialog] = useState(false)
const [isProcessingWithPassword, setIsProcessingWithPassword] = useState(false)

const handleAction = () => {
  setShowMasterPasswordDialog(true)
}

const handleConfirm = async (password: string) => {
  try {
    setIsProcessingWithPassword(true)
    await performAction(password)
    setShowMasterPasswordDialog(false)
  } catch (error) {
    // handle error
  } finally {
    setIsProcessingWithPassword(false)
  }
}

return (
  <>
    <button onClick={handleAction}>Action</button>
    <MasterPasswordDialog
      open={showMasterPasswordDialog}
      onOpenChange={setShowMasterPasswordDialog}
      onConfirm={handleConfirm}
      title="Action Title"
      description="Action description"
      isLoading={isProcessingWithPassword}
      showForgotPassword={isOwner}
    />
  </>
)
```

#### Updated code:
```tsx
const { requestMasterPassword } = useMasterPassword()

const handleAction = () => {
  requestMasterPassword({
    title: 'Action Title',
    description: 'Action description',
    onConfirm: async (password) => {
      await performAction(password)
    }
  })
}

return <button onClick={handleAction}>Action</button>
```

## Best Practices
1. **Always check the cache first:** call `MasterPasswordManager.getCachedMasterPassword()` before opening the dialog.
2. **Handle errors:** catch and rethrow inside `onConfirm` to keep the dialog open for retries.
3. **Clear titles/descriptions:** explain why the password is needed.
4. **Async ready:** `onConfirm` supports async/await so API calls can run directly.

## Notes
- ⚠️ Only use `useMasterPassword` inside components wrapped by `MasterPasswordProvider`.
- ⚠️ Errors thrown from `onConfirm` keep the dialog open for retry.
- ⚠️ "Forgot Password" visibility is handled automatically by user role.
