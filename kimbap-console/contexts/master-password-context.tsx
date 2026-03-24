'use client'

import { createContext, useContext, useState, ReactNode } from 'react'
import { MasterPasswordDialog } from '@/components/master-password-dialog'
import { useUserRole } from '@/hooks/use-user-role'

interface MasterPasswordContextType {
  requestMasterPassword: (options: {
    title?: string
    description?: string
    onConfirm: (password: string) => void | Promise<void>
  }) => void
  closeMasterPasswordDialog: () => void
}

const MasterPasswordContext = createContext<MasterPasswordContextType | undefined>(undefined)

export function MasterPasswordProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false)
  const [title, setTitle] = useState('Master Password Required')
  const [description, setDescription] = useState('Please enter your master password to continue.')
  const [onConfirmCallback, setOnConfirmCallback] = useState<((password: string) => void | Promise<void>) | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const { isOwner } = useUserRole()

  const requestMasterPassword = (options: {
    title?: string
    description?: string
    onConfirm: (password: string) => void | Promise<void>
  }) => {
    setTitle(options.title || 'Master Password Required')
    setDescription(options.description || 'Please enter your master password to continue.')
    setOnConfirmCallback(() => options.onConfirm)
    setOpen(true)
  }

  const closeMasterPasswordDialog = () => {
    setOpen(false)
    setIsLoading(false)
    setOnConfirmCallback(null)
  }

  const handleConfirm = async (password: string) => {
    if (onConfirmCallback) {
      try {
        setIsLoading(true)
        await onConfirmCallback(password)
        closeMasterPasswordDialog()
      } catch (error) {
        console.error('Master password confirmation error:', error)
        // Don't close dialog on error so user can retry
        setIsLoading(false)
      }
    }
  }

  return (
    <MasterPasswordContext.Provider value={{ requestMasterPassword, closeMasterPasswordDialog }}>
      {children}
      <MasterPasswordDialog
        open={open}
        onOpenChange={(open) => {
          if (!open) {
            closeMasterPasswordDialog()
          }
        }}
        onConfirm={handleConfirm}
        title={title}
        description={description}
        isLoading={isLoading}
        showForgotPassword={isOwner}
      />
    </MasterPasswordContext.Provider>
  )
}

export function useMasterPassword() {
  const context = useContext(MasterPasswordContext)
  if (!context) {
    throw new Error('useMasterPassword must be used within MasterPasswordProvider')
  }
  return context
}
