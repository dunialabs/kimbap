"use client"

import { useState, useRef } from "react"
import { Upload, Calendar, AlertTriangle, Shield, Loader2 } from "lucide-react"
import { toast } from "sonner"


import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

interface RestoreDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onRestoreSuccess?: () => void
}

export function RestoreDialog({ open, onOpenChange, onRestoreSuccess }: RestoreDialogProps) {
  const [restoreFile, setRestoreFile] = useState<File | null>(null)
  const [restoreFileType, setRestoreFileType] = useState<"config" | "full" | null>(null)
  const [masterPassword, setMasterPassword] = useState("")
  const [confirmationText, setConfirmationText] = useState("")
  const [isRestoring, setIsRestoring] = useState(false)
  const [restoreError, setRestoreError] = useState("")
  const [masterPasswordError, setMasterPasswordError] = useState(false)
  
  const masterPasswordRef = useRef<HTMLInputElement>(null)

  const handleOpenChange = (open: boolean) => {
    onOpenChange(open)
    if (!open) {
      // Reset form when closing
      setRestoreFile(null)
      setRestoreFileType(null)
      setMasterPassword("")
      setConfirmationText("")
      setRestoreError("")
      setIsRestoring(false)
      setMasterPasswordError(false)
    }
  }

  const handleRestoreExecute = async () => {
    // Check confirmation text
    if (confirmationText.toLowerCase() !== "restore and overwrite") {
      toast.error("Please type 'restore and overwrite' to confirm")
      return
    }
    
    setIsRestoring(true)
    
    try {
      let encryptedData = ""
      
      // Read file content for local file restore
      if (restoreFile) {
        const fileContent = await restoreFile.text()
        const backupData = JSON.parse(fileContent)
        encryptedData = backupData.data?.encryptedData
      }
      
      if (!encryptedData) {
        toast.error("No backup data found")
        return
      }
      
      const { api } = await import('@/lib/api-client')
      
      // Clear any previous errors
      setRestoreError("")
      setMasterPasswordError(false)
      
      // Call restore API
      await api.servers.restoreServerFromLocal({
        masterPwd: masterPassword,
        encryptedData: encryptedData
      })
      
      // If we reach here, the API call was successful
      toast.success("Server restored.")
      
      // Close dialog and reset state on success
      handleOpenChange(false)
      
      // Call success callback if provided
      if (onRestoreSuccess) {
        onRestoreSuccess()
      }
      
    } catch (error: any) {
      // Error handled below via UI state
      
      // Extract error message from the API error response
      const errorMessage = (error as any).userMessage || "Could not restore server"
      
      // Set error message to display in dialog
      setRestoreError(errorMessage)
      
      // If it's a password error, focus on the password input
      const statusCode = error.response?.status
      if (statusCode === 401 || errorMessage.toLowerCase().includes('password')) {
        setMasterPasswordError(true)
        setTimeout(() => {
          masterPasswordRef.current?.focus()
        }, 100)
      }
      
      // Keep dialog open on error so user can retry
    } finally {
      // Always reset the loading state
      setIsRestoring(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-2xl">
        <form onSubmit={(e) => { e.preventDefault(); handleRestoreExecute(); }}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Upload className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            Restore Backup
          </DialogTitle>
          <DialogDescription>
            Restore server data from a backup file.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* File upload section */}
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="restore-file">Backup File</Label>
              <Input
                id="restore-file"
                type="file"
                accept=".backup,.bak"
                onChange={async (e) => {
                  const file = e.target.files?.[0] || null
                  setRestoreFile(file)
                  setRestoreFileType(null)
                  
                  if (file) {
                    try {
                      const text = await file.text()
                      const data = JSON.parse(text)
                      
                      // Check if it has the expected structure
                      if (data.metadata && data.metadata.backupType) {
                        setRestoreFileType(data.metadata.backupType as "config" | "full")
                      }
                    } catch (error) {
                      // Parse failed - assume full backup needing password
                      // If we can't parse it, we'll assume it needs a password
                      setRestoreFileType("full")
                    }
                  }
                }}
              />
            </div>
            {restoreFile && (
              <div className="p-3 bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-800 rounded-md">
                <div className="flex items-center gap-2">
                  <Calendar className="h-4 w-4 text-green-600 dark:text-green-400" />
                  <span className="text-sm font-medium">{restoreFile.name}</span>
                </div>
                <div className="flex items-center mt-1">
                  <p className="text-xs text-green-800 dark:text-green-200">
                    Size: {(restoreFile.size / 1024).toFixed(1)} KB
                  </p>
                  {restoreFileType && (
                    <p className="text-xs text-green-800 dark:text-green-200 flex items-center">
                      <span className="mx-1">•</span>
                      <span className={restoreFileType === "full" ? "text-yellow-600 dark:text-yellow-500 flex items-center gap-1" : ""}>
                        {restoreFileType === "full" ? (
                          <>
                            Full Backup
                            <Shield className="h-3 w-3" />
                          </>
                        ) : (
                          "Configuration Only"
                        )}
                      </span>
                    </p>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Master Password and Confirmation - Show when backup/file is selected */}
          {restoreFile && (
            <div className="space-y-4 pt-4 border-t">
              <div className="p-4 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800 rounded-md">
                <div className="flex items-center gap-2 mb-1">
                  <AlertTriangle className="h-4 w-4 text-red-600 dark:text-red-400 flex-shrink-0" />
                  <p className="text-sm font-medium text-red-800 dark:text-red-200">
                    Overwrite Warning
                  </p>
                </div>
                <p className="text-xs text-red-700 dark:text-red-300">
                  This overwrites current server data. This cannot be undone.
                </p>
              </div>

              {/* Only show master password field for full backups */}
              {restoreFileType === "full" && (
                <div className="space-y-2">
                  <Label htmlFor="master-password">Master Password *</Label>
                  <Input
                    id="master-password"
                    ref={masterPasswordRef}
                    type="password"
                    placeholder="Master password"
                    value={masterPassword}
                    onChange={(e) => {
                      setMasterPassword(e.target.value)
                      setMasterPasswordError(false)
                    }}
                    className={masterPasswordError ? "border-red-500 focus:border-red-500 focus:ring-red-500" : ""}
                  />
                  <p className="text-xs text-muted-foreground">
                    Use the password used when this backup was created.
                  </p>
                </div>
              )}

              <div className="space-y-2">
                <Label htmlFor="confirmation">Confirmation Phrase *</Label>
                <Input
                  id="confirmation"
                  placeholder="restore and overwrite"
                  value={confirmationText}
                  onChange={(e) => setConfirmationText(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  Enter <code className="bg-muted px-1 rounded">restore and overwrite</code>.
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Error Display */}
        {restoreError && (
          <div className="p-4 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800 rounded-md">
            <div className="flex items-center gap-2 mb-1">
              <AlertTriangle className="h-5 w-5 text-red-600 dark:text-red-400 flex-shrink-0" />
              <p className="text-sm font-medium text-red-800 dark:text-red-200">
                Restore Error
              </p>
            </div>
            <p className="text-xs text-red-700 dark:text-red-300">
              {restoreError}
            </p>
          </div>
        )}

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => {
            handleOpenChange(false)
          }} disabled={isRestoring}>
            Cancel
          </Button>
          <Button 
            type="submit"
            disabled={
              isRestoring || 
              !restoreFile ||
              (restoreFileType === "full" && !masterPassword.trim()) ||
              confirmationText.toLowerCase() !== "restore and overwrite"
            }
            className="bg-blue-600 hover:bg-blue-700"
          >
            {isRestoring ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Restoring...
              </>
            ) : (
              <>
                <Upload className="w-4 h-4 mr-2" />
                Restore Server
              </>
            )}
          </Button>
        </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
