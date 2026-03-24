"use client"

import { Trash2, AlertTriangle, Loader2 } from "lucide-react"
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
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

interface DestroyTokenDialogProps {
  tokenName: string
  tokenId: string
  isOwnerToken?: boolean
  onDestroy: (tokenId: string) => Promise<void>
}

export function DestroyTokenDialog({ tokenName, tokenId, isOwnerToken = false, onDestroy }: DestroyTokenDialogProps) {
  const [open, setOpen] = useState(false)
  const [confirmText, setConfirmText] = useState("")
  const [isDestroying, setIsDestroying] = useState(false)
  const [error, setError] = useState("")

  const requiredText = "destroy token"
  const canDestroy = confirmText.toLowerCase() === requiredText && !isOwnerToken

  const handleDestroy = async () => {
    if (!canDestroy) return

    setIsDestroying(true)
    setError("")

    try {
      await onDestroy(tokenId)
      setOpen(false)
      setConfirmText("")
    } catch (err) {
      setError("Could not delete token.")
    } finally {
      setIsDestroying(false)
    }
  }

  const handleCancel = () => {
    setConfirmText("")
    setError("")
    setOpen(false)
  }

  if (isOwnerToken) {
    return (
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          <Button variant="outline" size="sm" disabled className="text-xs opacity-50 bg-transparent" aria-label="Cannot destroy owner token">
            <Trash2 className="h-3 w-3" />
          </Button>
        </DialogTrigger>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-amber-500 dark:text-amber-400" />
              Owner Token Protected
            </DialogTitle>
            <DialogDescription>Owner tokens cannot be deleted.</DialogDescription>
          </DialogHeader>
          <Alert>
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>
              Owner token <strong>{tokenName}</strong> is protected and cannot be deleted.
            </AlertDescription>
          </Alert>
          <DialogFooter>
            <Button onClick={() => setOpen(false)}>Close</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm" className="text-xs text-destructive hover:text-destructive hover:bg-destructive/10 bg-transparent" aria-label="Destroy access token">
          <Trash2 className="h-3 w-3" />
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Trash2 className="h-5 w-5 text-red-500 dark:text-red-400" />
            Delete Access Token
          </DialogTitle>
          <DialogDescription>This action permanently deletes the token and cannot be undone.</DialogDescription>
        </DialogHeader>

        <form
          onSubmit={(e) => {
            e.preventDefault()
            handleDestroy()
          }}
        >
          <div className="space-y-4">
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                <div className="space-y-1">
                  <p>
                    <strong>Token:</strong> {tokenName}
                  </p>
                  <p>
                    <strong>ID:</strong> {tokenId}
                  </p>
                  <p>
                    <strong>Warning:</strong> Clients using this token lose access immediately.
                  </p>
                </div>
              </AlertDescription>
            </Alert>

            <div className="space-y-2">
              <Label htmlFor="confirm-text">
                Type <code className="bg-muted px-1 py-0.5 rounded text-sm">{requiredText}</code>
              </Label>
              <Input
                id="confirm-text"
                placeholder={requiredText}
                value={confirmText}
                onChange={(e) => {
                  setConfirmText(e.target.value)
                  setError("")
                }}
                disabled={isDestroying}
                autoFocus
              />
            </div>

            {error && (
              <Alert variant="destructive">
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleCancel} disabled={isDestroying}>
              Cancel
            </Button>
            <Button type="submit" variant="destructive" disabled={!canDestroy || isDestroying}>
              {isDestroying ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Destroying...
                </>
              ) : (
                <>
                  <Trash2 className="mr-2 h-4 w-4" />
                  Destroy Token
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
