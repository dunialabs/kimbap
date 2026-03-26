"use client"

import { Key, AlertTriangle, CheckCircle, Loader2 } from "lucide-react"
import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
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

interface ResetTokenDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  serverName: string
  serverAddress: string
  onResetToken: (newToken: string) => Promise<void>
}

export function ResetTokenDialog({
  open,
  onOpenChange,
  serverName,
  serverAddress,
  onResetToken,
}: ResetTokenDialogProps) {
  const [newToken, setNewToken] = useState("")
  const [isResetting, setIsResetting] = useState(false)
  const [error, setError] = useState("")

  const handleReset = async () => {
    if (!newToken.trim()) {
      setError("Token is required")
      return
    }

    setIsResetting(true)
    setError("")

    try {
      await onResetToken(newToken)
      setNewToken("")
      onOpenChange(false)
    } catch (err) {
      setError("Could not reset token.")
    } finally {
      setIsResetting(false)
    }
  }

  const handleCancel = () => {
    setNewToken("")
    setError("")
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={(isOpen) => { if (!isOpen) handleCancel(); else onOpenChange(true); }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Key className="h-5 w-5" />
            Reset token
          </DialogTitle>
          <DialogDescription>
            Connection to <strong>{serverName}</strong> failed. Enter a new access token to reconnect.
          </DialogDescription>
        </DialogHeader>

        <form
          onSubmit={(e) => {
            e.preventDefault()
            handleReset()
          }}
        >
          <div className="space-y-4">
            <Alert>
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                <div className="space-y-1">
                  <p>
                    <strong>Server:</strong> {serverName}
                  </p>
                  <p>
                    <strong>Address:</strong> {serverAddress}
                  </p>
                  <p>
                    <strong>Issue:</strong> Access token is invalid or expired
                  </p>
                </div>
              </AlertDescription>
            </Alert>

            <div className="space-y-2">
              <Label htmlFor="new-token">Access token</Label>
              <Input
                id="new-token"
                type="password"
                placeholder="kimbap_..."
                value={newToken}
                onChange={(e) => {
                  setNewToken(e.target.value)
                  setError("")
                }}
                disabled={isResetting}
                autoFocus
              />
              <p className="text-xs text-muted-foreground">
                Use a token with access to this server.
              </p>
            </div>

            {error && (
              <Alert variant="destructive">
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <div className="bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-md p-3">
              <div className="flex items-start gap-2">
                <CheckCircle className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
                <div className="text-xs text-blue-800 dark:text-blue-200">
                  <p className="font-medium mb-1">Get a new access token:</p>
                  <ul className="space-y-1">
                    <li>• Open your server admin panel</li>
                    <li>• Go to Access Tokens</li>
                    <li>• Create or copy a token</li>
                    <li>• Confirm it has required access</li>
                  </ul>
                </div>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleCancel} disabled={isResetting}>
              Cancel
            </Button>
            <Button type="submit" variant="destructive" disabled={!newToken.trim() || isResetting}>
              {isResetting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Resetting...
                </>
              ) : (
                <>
                  <Key className="mr-2 h-4 w-4" />
                  Reset Token
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
