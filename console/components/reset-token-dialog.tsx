"use client"

import { Key, AlertTriangle, CheckCircle, Eye, EyeOff, Loader2 } from "lucide-react"
import { useEffect, useRef, useState } from "react"

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
  const [showToken, setShowToken] = useState(false)
  const [isResetting, setIsResetting] = useState(false)
  const [error, setError] = useState("")
  const tokenInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!open || isResetting) {
      return
    }

    const frame = window.requestAnimationFrame(() => {
      tokenInputRef.current?.focus()
    })

    return () => window.cancelAnimationFrame(frame)
  }, [open, isResetting])

  const handleReset = async () => {
    const trimmedToken = newToken.trim()

    if (!trimmedToken) {
      setError("Enter a new access token.")
      return
    }

    setIsResetting(true)
    setError("")

    try {
      await onResetToken(trimmedToken)
      setNewToken("")
      setShowToken(false)
      onOpenChange(false)
    } catch (err) {
      setError("Could not reconnect with that access token. Check the token and try again.")
    } finally {
      setIsResetting(false)
    }
  }

  const handleCancel = () => {
    setNewToken("")
    setShowToken(false)
    setError("")
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={(isOpen) => { if (!isOpen) handleCancel(); else onOpenChange(true); }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Key className="h-5 w-5" />
            Reconnect access token
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
                  <p className="break-all font-mono text-xs text-muted-foreground">
                    <strong className="font-semibold text-foreground">Address:</strong> {serverAddress}
                  </p>
                  <p>
                    <strong>Issue:</strong> The current access token is invalid or expired.
                  </p>
                </div>
              </AlertDescription>
            </Alert>

            <div className="space-y-2">
              <Label htmlFor="new-token">
                Access token
                <span className="ml-1 text-xs font-normal text-muted-foreground">(required)</span>
              </Label>
              <div className="relative">
                <Input
                  id="new-token"
                  type={showToken ? "text" : "password"}
                  placeholder="kimbap_..."
                  ref={tokenInputRef}
                  value={newToken}
                  onChange={(e) => {
                    setNewToken(e.target.value)
                    setError("")
                  }}
                  disabled={isResetting}
                  aria-invalid={Boolean(error)}
                  aria-describedby={error ? 'reset-token-error reset-token-note' : 'reset-token-note'}
                  autoCapitalize="none"
                  autoCorrect="off"
                  spellCheck={false}
                  autoComplete="off"
                  className="pr-10"
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowToken((prev) => !prev)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 rounded text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                  aria-label={showToken ? "Hide access token" : "Show access token"}
                  disabled={isResetting}
                >
                  {showToken ? <EyeOff className="h-4 w-4" aria-hidden="true" /> : <Eye className="h-4 w-4" aria-hidden="true" />}
                </button>
              </div>
              <p id="reset-token-note" className="text-xs text-muted-foreground">
                Use a token with access to this server.
              </p>
            </div>

            {error && (
              <Alert id="reset-token-error" variant="destructive">
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <div className="bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-md p-3">
              <div className="flex items-start gap-2">
                <CheckCircle className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
                <div className="text-xs text-blue-800 dark:text-blue-200">
                  <p className="font-medium mb-1">To get a new access token:</p>
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
            <Button type="submit" disabled={!newToken.trim() || isResetting}>
              {isResetting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Reconnecting...
                </>
              ) : (
                <>
                  <Key className="mr-2 h-4 w-4" />
                  Reconnect
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
