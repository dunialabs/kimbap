"use client"

import { Server, Plus, CheckCircle, AlertCircle, Globe, Wifi, Wrench } from "lucide-react"
import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"

import { Button } from "@/components/ui/button"

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  ScrollableDialogContent,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"



interface ConnectKimbapServerDialogProps {
  onConnect: (servers: any[]) => void
}

export function ConnectKimbapServerDialog({ onConnect }: ConnectKimbapServerDialogProps) {
  const [open, setOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  // Kimbap server form
  const [kimbapName, setKimbapName] = useState("")
  const [kimbapAddress, setKimbapAddress] = useState("")
  const [kimbapToken, setKimbapToken] = useState("")
  const [testingKimbap, setTestingKimbap] = useState(false)
  const [kimbapTestResult, setKimbapTestResult] = useState<boolean | null>(null)

  // Other server form (JSON)
  const [otherName, setOtherName] = useState("")
  const [otherJson, setOtherJson] = useState("")
  const [testingOther, setTestingOther] = useState(false)
  const [otherTestResult, setOtherTestResult] = useState<boolean | null>(null)

  const handleTestKimbapServer = async () => {
    if (!kimbapAddress || !kimbapToken) return
    
    setTestingKimbap(true)
    
    // Simulate test - random success/failure
    await new Promise(resolve => setTimeout(resolve, 2000))
    const success = Math.random() > 0.3
    
    setKimbapTestResult(success)
    setTestingKimbap(false)
  }

  const handleTestOtherServer = async () => {
    if (!otherJson) return
    
    try {
      JSON.parse(otherJson) // Validate JSON
    } catch {
      setOtherTestResult(false)
      return
    }
    
    setTestingOther(true)
    
    // Simulate test - random success/failure
    await new Promise(resolve => setTimeout(resolve, 2000))
    const success = Math.random() > 0.3
    
    setOtherTestResult(success)
    setTestingOther(false)
  }


  const canConnect = () => {
    // Check kimbap server
    const kimbapValid = kimbapName && kimbapAddress && kimbapToken
    
    // Check other server
    const otherValid = otherName && otherJson
    try {
      if (otherJson) JSON.parse(otherJson)
    } catch {
      return kimbapValid
    }

    return kimbapValid || otherValid
  }

  const handleConnect = async () => {
    if (!canConnect()) return

    setIsLoading(true)

    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 2000))

    const serversToAdd = []

    // Add kimbap server
    if (kimbapName && kimbapAddress && kimbapToken) {
      serversToAdd.push({
        id: `kimbap-${Date.now()}`,
        name: kimbapName,
        type: "kimbap",
        status: "connected",
        address: kimbapAddress,
        userRole: "Member",
        autoDiscovered: false,
        hasUpdates: false,
        updates: [],
        tools: [
          {
            id: `web-search-kimbap-${Date.now()}`,
            name: "Web Search",
            icon: Globe,
            enabled: true,
            subFunctions: [{ name: "Google Search", enabled: true }],
          },
        ],
      })
    }

    // Add other server
    if (otherName && otherJson) {
      try {
        const config = JSON.parse(otherJson)
        serversToAdd.push({
          id: `other-${Date.now()}`,
          name: otherName,
          type: "other",
          status: "connected",
          address: config.address || "Custom Server",
          autoDiscovered: false,
          hasUpdates: false,
          updates: [],
          tools: [
            {
              id: `custom-tools-${Date.now()}`,
              name: "Custom Tools",
              icon: Wrench,
              enabled: true,
              subFunctions: [{ name: "Custom Function", enabled: true }],
            },
          ],
        })
      } catch (e) {
        // JSON parse failed - skip this server
      }
    }

    onConnect(serversToAdd)

    // Reset form
    setKimbapName("")
    setKimbapAddress("")
    setKimbapToken("")
    setTestingKimbap(false)
    setKimbapTestResult(null)
    setOtherName("")
    setOtherJson("")
    setTestingOther(false)
    setOtherTestResult(null)
    setIsLoading(false)
    setOpen(false)
  }

  const serverCount = (kimbapName ? 1 : 0) + (otherName ? 1 : 0)
  const [activeTab, setActiveTab] = useState("kimbap")

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <Plus className="mr-2 h-4 w-4" />
          Add MCP Server
        </Button>
      </DialogTrigger>
      <ScrollableDialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Server className="h-5 w-5" />
            Add MCP Server
          </DialogTitle>
          <DialogDescription>
            Add a Kimbap Server with an address and token, or add another server with JSON.
          </DialogDescription>
        </DialogHeader>

        <form
          onSubmit={(e) => {
            e.preventDefault()
            handleConnect()
          }}
        >
          <div>
          <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="kimbap">
              <Server className="w-4 h-4 mr-2" />
              Kimbap Server
            </TabsTrigger>
            <TabsTrigger value="other">
              <Wrench className="w-4 h-4 mr-2" />
              Other Server
            </TabsTrigger>
          </TabsList>

          <TabsContent value="kimbap" className="space-y-4">
            <Alert>
              <Server className="h-4 w-4" />
              <AlertDescription>
                Enter the server address and token.
              </AlertDescription>
            </Alert>

            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="kimbap-name">Server Name *</Label>
                <Input
                  id="kimbap-name"
                  placeholder="My Kimbap Server"
                  value={kimbapName}
                  onChange={(e) => setKimbapName(e.target.value)}
                  autoFocus
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="kimbap-address">Server Address *</Label>
                <Input
                  id="kimbap-address"
                  placeholder="https://mcp.example.com"
                  value={kimbapAddress}
                  onChange={(e) => setKimbapAddress(e.target.value)}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="kimbap-token">Access Token *</Label>
                <div className="flex gap-2">
                  <Input
                    id="kimbap-token"
                    type="password"
                  placeholder="kimbap_..."
                    value={kimbapToken}
                    onChange={(e) => setKimbapToken(e.target.value)}
                    className="flex-1"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={handleTestKimbapServer}
                    disabled={!kimbapAddress || !kimbapToken || testingKimbap}
                    className={cn(
                      "px-3",
                      kimbapTestResult === true && "border-green-500 dark:border-green-400 text-green-700 dark:text-green-300",
                      kimbapTestResult === false && "border-red-500 dark:border-red-400 text-red-700 dark:text-red-300"
                    )}
                  >
                    {testingKimbap ? (
                      <>
                        <div className="w-3 h-3 border-2 border-slate-400 border-t-transparent rounded-full animate-spin mr-1" />
                        Testing...
                      </>
                    ) : (
                      <>
                        <Wifi className="w-3 h-3 mr-1" />
                        Test
                      </>
                    )}
                  </Button>
                </div>
                {kimbapTestResult !== null && (
                  <div className={cn(
                    "text-xs mt-1 flex items-center gap-1",
                    kimbapTestResult ? "text-green-600 dark:text-green-400" : "text-red-600 dark:text-red-400"
                  )}>
                    {kimbapTestResult ? (
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
          </TabsContent>

          <TabsContent value="other" className="space-y-4">
            <Alert>
              <Wrench className="h-4 w-4" />
              <AlertDescription>
                Add another server with JSON configuration.
              </AlertDescription>
            </Alert>

            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="other-name">Server Name *</Label>
                <Input
                  id="other-name"
                  placeholder="My Custom Server"
                  value={otherName}
                  onChange={(e) => setOtherName(e.target.value)}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="other-json">Server Configuration (JSON) *</Label>
                <Textarea
                  id="other-json"
                  placeholder={`{
  "command": "node",
  "args": ["path/to/server.js"],
  "env": {
    "API_KEY": "your-api-key"
  },
  "address": "https://server.example.com"
}`}
                  value={otherJson}
                  onChange={(e) => setOtherJson(e.target.value)}
                  className="font-mono text-sm min-h-[120px]"
                />
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={handleTestOtherServer}
                    disabled={!otherJson || testingOther}
                    className={cn(
                      "px-3",
                      otherTestResult === true && "border-green-500 dark:border-green-400 text-green-700 dark:text-green-300",
                      otherTestResult === false && "border-red-500 dark:border-red-400 text-red-700 dark:text-red-300"
                    )}
                  >
                    {testingOther ? (
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
                {otherTestResult !== null && (
                  <div className={cn(
                    "text-xs mt-1 flex items-center gap-1",
                    otherTestResult ? "text-green-600 dark:text-green-400" : "text-red-600 dark:text-red-400"
                  )}>
                    {otherTestResult ? (
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
          </TabsContent>
          </Tabs>

          {serverCount > 0 && (
            <>
              <Separator />
              <div className="bg-blue-50 dark:bg-blue-950/20 p-3 rounded-lg">
                <div className="text-sm font-medium text-blue-900 dark:text-blue-100">Summary</div>
                <div className="text-xs text-blue-700 dark:text-blue-300 mt-1">
                  {serverCount} server{serverCount > 1 ? "s" : ""} will be added to MCP clients
                </div>
              </div>
            </>
          )}
          </div>

          <DialogFooter className="border-t pt-4">
            <Button type="button" variant="outline" onClick={() => setOpen(false)} disabled={isLoading}>
              Cancel
            </Button>
            <Button type="submit" disabled={!canConnect() || isLoading}>
              {isLoading ? "Connecting..." : serverCount > 0 ? `Connect (${serverCount})` : "Connect"}
            </Button>
          </DialogFooter>
        </form>
      </ScrollableDialogContent>
    </Dialog>
  )
}
