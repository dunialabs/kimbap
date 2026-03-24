"use client"

import { Wrench, CheckCircle, AlertCircle, Wifi } from "lucide-react"
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
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"

interface AddOtherServerDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onAdd: (server: any) => void
}

export function AddOtherServerDialog({ open, onOpenChange, onAdd }: AddOtherServerDialogProps) {
  const [isLoading, setIsLoading] = useState(false)
  const [serverName, setServerName] = useState("")
  const [serverJson, setServerJson] = useState("")
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<boolean | null>(null)

  const handleTest = async () => {
    if (!serverJson) return
    
    try {
      JSON.parse(serverJson) // Validate JSON
    } catch {
      setTestResult(false)
      return
    }
    
    setTesting(true)
    
    // Simulate test - random success/failure
    await new Promise(resolve => setTimeout(resolve, 2000))
    const success = Math.random() > 0.3
    
    setTestResult(success)
    setTesting(false)
  }

  const canAdd = () => {
    if (!serverName || !serverJson) return false
    try {
      JSON.parse(serverJson)
      return true
    } catch {
      return false
    }
  }

  const handleAdd = async () => {
    if (!canAdd()) return

    setIsLoading(true)

    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 1500))

    try {
      const config = JSON.parse(serverJson)
      const newServer = {
        id: `other-${Date.now()}`,
        name: serverName,
        type: "other",
        status: "connected",
        address: config.address || "Custom Server",
        hasUpdates: false,
        updates: [],
        tools: [
          {
            id: `custom-tools-${Date.now()}`,
            name: "Custom Tools",
            icon: Wrench,
            enabled: true,
            subFunctions: [],
          },
        ],
      }

      onAdd(newServer)

      // Reset form
      setServerName("")
      setServerJson("")
      setTestResult(null)
      setIsLoading(false)
      onOpenChange(false)
    } catch (e) {
      // JSON parse failed - ignore invalid input
      setIsLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Wrench className="h-5 w-5" />
            Add Other Server
          </DialogTitle>
          <DialogDescription>
            Add a server with JSON configuration.
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
              <Wrench className="h-4 w-4" />
              <AlertDescription>
                Paste a valid MCP server JSON config.
              </AlertDescription>
            </Alert>

            <div className="space-y-2">
              <Label htmlFor="server-name">Server Name *</Label>
              <Input
                id="server-name"
                placeholder="My Custom Server"
                value={serverName}
                onChange={(e) => setServerName(e.target.value)}
                autoFocus
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="server-json">Server Configuration (JSON) *</Label>
              <Textarea
                id="server-json"
                placeholder={`{
  "command": "node",
  "args": ["path/to/server.js"],
  "env": {
    "API_KEY": "your-api-key"
  },
  "address": "https://server.example.com"
}`}
                value={serverJson}
                onChange={(e) => setServerJson(e.target.value)}
                className="font-mono text-sm min-h-[120px]"
              />
              <div className="flex gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={handleTest}
                  disabled={!serverJson || testing}
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
                      Validate JSON
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
                      JSON valid
                    </>
                  ) : (
                    <>
                      <AlertCircle className="w-3 h-3" />
                      JSON invalid
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
