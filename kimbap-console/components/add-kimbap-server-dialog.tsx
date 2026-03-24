"use client"

import { Server, CheckCircle, AlertCircle, Wifi } from "lucide-react"
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
import { cn } from "@/lib/utils"

interface AddKimbapServerDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onAdd: (server: any) => void
}

export function AddKimbapServerDialog({ open, onOpenChange, onAdd }: AddKimbapServerDialogProps) {
  const [isLoading, setIsLoading] = useState(false)
  const [kimbapUrl, setKimbapUrl] = useState("")
  const [kimbapApiKey, setKimbapApiKey] = useState("")
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<boolean | null>(null)

  const handleTest = async () => {
    if (!kimbapUrl || !kimbapApiKey) return
    
    setTesting(true)
    
    // Simulate test - random success/failure
    await new Promise(resolve => setTimeout(resolve, 2000))
    const success = Math.random() > 0.3
    
    setTestResult(success)
    setTesting(false)
  }

  const canAdd = () => {
    return kimbapUrl && kimbapApiKey
  }

  const handleAdd = async () => {
    if (!canAdd()) return

    setIsLoading(true)

    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 1500))

    const newServer = {
      id: `kimbap-${Date.now()}`,
      name: new URL(kimbapUrl).hostname,
      type: "kimbap",
      status: "connected",
      address: kimbapUrl,
      hasUpdates: false,
      updates: [],
      tools: [
        {
          id: `tools-${Date.now()}`,
          name: "Kimbap Tools",
          icon: Server,
          enabled: true,
          subFunctions: [],
        },
      ],
    }

    onAdd(newServer)

    // Reset form
    setKimbapUrl("")
    setKimbapApiKey("")
    setTestResult(null)
    setIsLoading(false)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Server className="h-5 w-5" />
            Add Kimbap Server
          </DialogTitle>
          <DialogDescription>
            Enter the server URL and API key.
          </DialogDescription>
        </DialogHeader>

        <form
          onSubmit={(e) => {
            e.preventDefault()
            handleAdd()
          }}
        >
          <div className="space-y-4">
            <Alert>
              <Server className="h-4 w-4" />
              <AlertDescription>
                Add the server URL and API key.
              </AlertDescription>
            </Alert>

            <div className="space-y-2">
              <Label htmlFor="kimbap-url">Kimbap Server URL *</Label>
              <Input
                id="kimbap-url"
                placeholder="https://mcp.example.com"
                value={kimbapUrl}
                onChange={(e) => setKimbapUrl(e.target.value)}
                autoFocus
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="kimbap-api-key">API Key *</Label>
              <div className="flex gap-2">
                <Input
                  id="kimbap-api-key"
                  type="password"
                  placeholder="kimbap_..."
                  value={kimbapApiKey}
                  onChange={(e) => setKimbapApiKey(e.target.value)}
                  className="flex-1"
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={handleTest}
                  disabled={!kimbapUrl || !kimbapApiKey || testing}
                  className={cn(
                    "px-3",
                    testResult === true && "border-green-500 dark:border-green-400 text-green-700 dark:text-green-300",
                    testResult === false && "border-red-500 dark:border-red-400 text-red-700 dark:text-red-300"
                  )}
                >
                  {testing ? (
                    <>
                      <div className="w-3 h-3 border-2 border-slate-400 border-t-transparent rounded-full animate-spin mr-1" />
                      Testing...
                    </>
                  ) : (
                    <>
                      <Wifi className="w-3 h-3 mr-1" />
                      Test Connection
                    </>
                  )}
                </Button>
              </div>
              {testResult !== null && (
                <div className={cn(
                  "text-xs mt-1 flex items-center gap-1",
                  testResult ? "text-green-600 dark:text-green-400" : "text-red-600 dark:text-red-400"
                )}>
                  {testResult ? (
                    <>
                      <CheckCircle className="w-3 h-3" />
                      Connected
                    </>
                  ) : (
                    <>
                      <AlertCircle className="w-3 h-3" />
                      Could not connect
                    </>
                  )}
                </div>
              )}
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={isLoading}>
              Cancel
            </Button>
            <Button type="submit" disabled={!canAdd() || isLoading}>
              {isLoading ? "Adding..." : "Add Server"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
