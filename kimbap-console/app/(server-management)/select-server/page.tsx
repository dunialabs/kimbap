"use client"

import {
  ServerIcon,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Crown,
  UserCheck,
  User,
  Clock,
  Globe,
  Eye,
  EyeOff,
  ArrowRight,
  MoreVertical,
  Trash2,
  Settings,
  ExternalLink,
} from "lucide-react"
import { useRouter } from "next/navigation"
import type React from "react"
import { useState } from "react"

import { MCPPageHeader } from "@/components/layouts/mcp-page-header"
import { MCPPageLayout } from "@/components/layouts/mcp-page-layout"
import { Alert, AlertDescription } from "@/components/ui/alert"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"

// Server status types
type ServerStatus = "running" | "stopped" | "unknown" | "owner" | "auth-failed" | "address-error" | "not-joined"

interface ServerDetails {
  id: string
  name: string
  address: string
  status: ServerStatus
  userRole?: "Admin" | "Member" | null
  description?: string
  lastAccessed?: string
}

export default function SelectServerPage() {
  const [servers, setServers] = useState<ServerDetails[]>([])
  const [selectedServer, setSelectedServer] = useState<ServerDetails | null>(null)
  const [showAuthDialog, setShowAuthDialog] = useState(false)
  const [showMemberAccess, setShowMemberAccess] = useState(false)
  const [showRemoveDialog, setShowRemoveDialog] = useState(false)
  const [showAddressErrorDialog, setShowAddressErrorDialog] = useState(false)
  const [authToken, setAuthToken] = useState("")
  const [showToken, setShowToken] = useState(false)
  const [authError, setAuthError] = useState("")
  const [isAuthenticating, setIsAuthenticating] = useState(false)
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [confirmText, setConfirmText] = useState("")
  const [isRemoving, setIsRemoving] = useState(false)
  const router = useRouter()

  const getStatusInfo = (status: ServerStatus) => {
    switch (status) {
      case "running":
        return {
          label: "Running",
          color: "text-emerald-700 dark:text-emerald-300 bg-emerald-50 dark:bg-emerald-950/20 border-emerald-200 dark:border-emerald-800",
          icon: CheckCircle,
          iconColor: "text-emerald-600 dark:text-emerald-400",
          indicator: "bg-emerald-500",
        }
      case "stopped":
        return {
          label: "Stopped",
          color: "text-red-700 dark:text-red-300 bg-red-50 dark:bg-red-950/20 border-red-200 dark:border-red-800",
          icon: XCircle,
          iconColor: "text-red-600 dark:text-red-400",
          indicator: "bg-red-500",
        }
      case "owner":
        return {
          label: "Owner",
          color: "text-purple-700 dark:text-purple-300 bg-purple-50 dark:bg-purple-950/20 border-purple-200 dark:border-purple-800",
          icon: Crown,
          iconColor: "text-purple-600 dark:text-purple-400",
          indicator: "bg-purple-500",
        }
      case "auth-failed":
        return {
          label: "Auth Failed",
          color: "text-red-700 dark:text-red-300 bg-red-50 dark:bg-red-950/20 border-red-200 dark:border-red-800",
          icon: AlertTriangle,
          iconColor: "text-red-600 dark:text-red-400",
          indicator: "bg-red-500",
        }
      case "address-error":
        return {
          label: "Address Error",
          color: "text-red-700 dark:text-red-300 bg-red-50 dark:bg-red-950/20 border-red-200 dark:border-red-800",
          icon: XCircle,
          iconColor: "text-red-600 dark:text-red-400",
          indicator: "bg-red-500",
        }
      case "not-joined":
        return {
          label: "Not Joined",
          color: "text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-slate-900 border-slate-200 dark:border-slate-700",
          icon: User,
          iconColor: "text-slate-500 dark:text-slate-400",
          indicator: "bg-slate-400 dark:bg-slate-500",
        }
      default:
        return {
          label: "Unknown",
          color: "text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-950/20 border-amber-200 dark:border-amber-800",
          icon: AlertTriangle,
          iconColor: "text-amber-600 dark:text-amber-400",
          indicator: "bg-amber-500",
        }
    }
  }

  const handleServerClick = (server: ServerDetails) => {
    if (server.status === "address-error") {
      setSelectedServer(server)
      setShowAddressErrorDialog(true)
      return
    }

    if (server.status === "not-joined" || server.status === "auth-failed") {
      setSelectedServer(server)
      setShowAuthDialog(true)
      setAuthToken("")
      setAuthError("")
    } else if (server.userRole === "Member") {
      setSelectedServer(server)
      setShowMemberAccess(true)
    } else {
      // Admin access - go directly to dashboard
      router.push("/dashboard")
    }
  }

  const handleRemoveServer = (server: ServerDetails, e: React.MouseEvent) => {
    e.stopPropagation()
    setSelectedServer(server)
    setShowRemoveDialog(true)
    setConfirmText("")
  }

  const handleConfirmRemove = async () => {
    if (!selectedServer) return

    const isOwner = selectedServer.status === "owner"
    const requiredText = isOwner ? selectedServer.name : "remove"

    if (confirmText.toLowerCase() !== requiredText.toLowerCase()) {
      return
    }

    setIsRemoving(true)

    await new Promise((resolve) => setTimeout(resolve, 1500))

    setServers((prev) => prev.filter((s) => s.id !== selectedServer.id))

    setIsRemoving(false)
    setShowRemoveDialog(false)
    setSelectedServer(null)
    setConfirmText("")
  }

  const handleAuthSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedServer || !authToken.trim()) return

    setIsAuthenticating(true)
    setAuthError("")

    await new Promise((resolve) => setTimeout(resolve, 1500))

    setIsAuthenticating(false)
    setShowAuthDialog(false)

    // Update server status and redirect
    if (selectedServer.status === "auth-failed") {
      // Redirect to dashboard for re-authenticated servers
      router.push("/dashboard")
    } else {
      // For new joins, show member access first
      setShowMemberAccess(true)
    }
  }


  // Member access removed - redirect to dashboard
  if (showMemberAccess && selectedServer) {
    // Instead of showing client management card, redirect to dashboard
    window.location.href = '/dashboard'
    return null
  }

  return (
    <MCPPageLayout>
      <MCPPageHeader
        title="Select Your Server"
        description="Select a server to manage, or add a new one."
        badge={{
          text: "MCP Server Management",
          icon: <ServerIcon className="w-4 h-4" />
        }}
        actions={
          <div className="flex gap-3">
            <Button
              onClick={() => setShowCreateDialog(true)}
              className="bg-primary hover:bg-primary/90 shadow-lg"
            >
              <ServerIcon className="w-4 h-4 mr-2" />
              Create New Server
            </Button>
            <Button variant="outline" className="backdrop-blur-sm">
              <Globe className="w-4 h-4 mr-2" />
              Join Server by URL
            </Button>
          </div>
        }
        centered
      />

        {servers.length === 0 ? (
          <Card className="border-dashed bg-white/60 dark:bg-card/60">
            <CardHeader>
              <CardTitle>No servers yet</CardTitle>
              <CardDescription>
                Create a server or join an existing one.
              </CardDescription>
            </CardHeader>
            <CardContent className="text-sm text-muted-foreground">
              Your servers will appear here.
            </CardContent>
          </Card>
        ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {servers.map((server) => {
            const statusInfo = getStatusInfo(server.status)
            const StatusIcon = statusInfo.icon
            const needsAuth =
              server.status === "not-joined" || server.status === "auth-failed" || server.status === "address-error"

            return (
              <Card
                key={server.id}
                className={cn(
                  "cursor-pointer transition-all duration-200 hover:shadow-lg bg-white/60 dark:bg-card/60 backdrop-blur-sm border-white/20 dark:border-border/20 group",
                  needsAuth && "border-yellow-200 dark:border-yellow-800 hover:border-yellow-300 dark:hover:border-yellow-700 hover:bg-yellow-50/50 dark:hover:bg-yellow-950/30",
                  server.status === "running" && "hover:border-emerald-300 dark:hover:border-emerald-700 hover:bg-emerald-50/50 dark:hover:bg-emerald-950/30",
                  server.status === "owner" && "hover:border-purple-300 dark:hover:border-purple-700 hover:bg-purple-50/50 dark:hover:bg-purple-950/30",
                )}
                onClick={() => handleServerClick(server)}
              >
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3 flex-1 min-w-0">
                      <div className="relative">
                        <div className="w-10 h-10 rounded-lg bg-slate-100 dark:bg-slate-800 flex items-center justify-center">
                          <ServerIcon className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                        </div>
                        <div
                          className={cn(
                            "absolute -bottom-1 -right-1 w-3 h-3 rounded-full border-2 border-white dark:border-card",
                            statusInfo.indicator,
                          )}
                        />
                      </div>
                      <div className="flex-1 min-w-0">
                        <CardTitle className="text-base font-semibold text-slate-900 dark:text-slate-100 truncate">{server.name}</CardTitle>
                        <CardDescription className="text-xs text-slate-500 dark:text-slate-400 truncate">{server.address}</CardDescription>
                      </div>
                    </div>

                    {/* Server Actions Dropdown */}
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 w-8 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
                          onClick={(e) => e.stopPropagation()}
                          aria-label="Server actions"
                        >
                          <MoreVertical className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-48">
                        <DropdownMenuItem onClick={(e) => e.stopPropagation()}>
                          <Settings className="mr-2 h-4 w-4" />
                          Server Settings
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={(e) => e.stopPropagation()}>
                          <ExternalLink className="mr-2 h-4 w-4" />
                          Open in Browser
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={(e) => handleRemoveServer(server, e)}
                          className="text-red-600 dark:text-red-400 focus:text-red-600 dark:focus:text-red-400"
                        >
                          <Trash2 className="mr-2 h-4 w-4" />
                          {server.status === "owner" ? "Delete Server" : "Remove Server"}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </CardHeader>

                <CardContent className="pt-0">
                  <div className="space-y-3">
                    {/* Status Badge */}
                    <div className="flex items-center justify-between">
                      <Badge variant="outline" className={cn("text-xs font-medium gap-1.5", statusInfo.color)}>
                        <StatusIcon className={cn("w-3 h-3", statusInfo.iconColor)} />
                        {statusInfo.label}
                      </Badge>
                      {server.userRole && (
                        <Badge variant="secondary" className="text-xs font-medium bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400">
                          {server.userRole === "Admin" ? (
                            <UserCheck className="w-3 h-3 mr-1" />
                          ) : (
                            <User className="w-3 h-3 mr-1" />
                          )}
                          {server.userRole}
                        </Badge>
                      )}
                    </div>

                    {/* Description */}
                    {server.description && <p className="text-xs text-slate-600 dark:text-slate-400 line-clamp-2">{server.description}</p>}

                    {/* Last Accessed */}
                    {server.lastAccessed && (
                      <div className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400">
                        <Clock className="w-3 h-3" />
                        Last accessed {server.lastAccessed}
                      </div>
                    )}

                    {/* Action Hint */}
                    <div className="flex items-center justify-between pt-2 border-t border-slate-100 dark:border-slate-800">
                      <span className="text-xs text-slate-500 dark:text-slate-400">
                        {server.status === "address-error" && "Click to view connection issue"}
                        {server.status === "not-joined" && "Click to authenticate"}
                        {server.status === "auth-failed" && "Click to re-authenticate"}
                        {server.status === "running" && server.userRole === "Admin" && "Click to manage"}
                        {server.status === "running" && server.userRole === "Member" && "Click to access"}
                        {server.status === "owner" && "Click to manage"}
                        {server.status === "stopped" && "Server offline"}
                      </span>
                      {(server.status === "running" || server.status === "owner" || needsAuth) && (
                        <ArrowRight className="w-3 h-3 text-slate-400 dark:text-slate-500" />
                      )}
                    </div>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
        )}

        {/* Authentication Dialog */}
        <Dialog open={showAuthDialog} onOpenChange={setShowAuthDialog}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <ServerIcon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                Connect to Server
              </DialogTitle>
              <DialogDescription>Enter your access token to connect to {selectedServer?.name}</DialogDescription>
            </DialogHeader>

            <div className="space-y-4 py-4">
              {/* Server Info */}
              <div className="p-3 bg-slate-50 dark:bg-slate-900 rounded-lg border">
                <div className="flex items-center gap-2 mb-1">
                  <ServerIcon className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                  <span className="font-medium text-slate-900 dark:text-slate-100">{selectedServer?.name}</span>
                </div>
                <p className="text-sm text-slate-600 dark:text-slate-400">{selectedServer?.address}</p>
              </div>

              {/* Token Input */}
              <form onSubmit={handleAuthSubmit} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="token">Access Token</Label>
                  <div className="relative">
                    <Input
                      id="token"
                      type={showToken ? "text" : "password"}
                      placeholder="Enter your access token"
                      value={authToken}
                      onChange={(e) => setAuthToken(e.target.value)}
                      className="pr-10"
                      disabled={isAuthenticating}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                      onClick={() => setShowToken(!showToken)}
                      disabled={isAuthenticating}
                      aria-label={showToken ? "Hide access token" : "Show access token"}
                    >
                      {showToken ? (
                        <EyeOff className="w-4 h-4 text-muted-foreground" />
                      ) : (
                        <Eye className="w-4 h-4 text-muted-foreground" />
                      )}
                    </Button>
                  </div>
                </div>

                {authError && (
                  <div className="p-3 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800 rounded-lg">
                    <div className="flex items-center gap-2">
                      <AlertTriangle className="w-4 h-4 text-red-600 dark:text-red-400" />
                      <span className="text-sm text-red-700 dark:text-red-300">{authError}</span>
                    </div>
                  </div>
                )}

                <DialogFooter className="gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setShowAuthDialog(false)}
                    disabled={isAuthenticating}
                  >
                    Cancel
                  </Button>
                  <Button
                    type="submit"
                    disabled={!authToken.trim() || isAuthenticating}
                    className="bg-blue-600 hover:bg-blue-700"
                  >
                    {isAuthenticating ? (
                      <>
                        <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2" />
                        Connecting...
                      </>
                    ) : (
                      "Connect"
                    )}
                  </Button>
                </DialogFooter>
              </form>
            </div>
          </DialogContent>
        </Dialog>

        {/* Address Error Dialog */}
        <AlertDialog open={showAddressErrorDialog} onOpenChange={setShowAddressErrorDialog}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle className="flex items-center gap-2">
                <XCircle className="w-5 h-5 text-red-600 dark:text-red-400" />
                Connection Failed
              </AlertDialogTitle>
              <AlertDialogDescription asChild>
                <div className="space-y-3">
                  <p>The current server address cannot be reached:</p>
                  <div className="p-3 bg-slate-50 dark:bg-slate-900 rounded-lg border font-mono text-sm">{selectedServer?.address}</div>
                  <Alert>
                    <AlertTriangle className="h-4 w-4" />
                    <AlertDescription>
                      Please contact the server provider to verify the correct address and ensure the server is running.
                    </AlertDescription>
                  </Alert>
                </div>
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Close</AlertDialogCancel>
              <AlertDialogAction
                onClick={() => {
                  setShowAddressErrorDialog(false)
                  // Could open a contact form or support page
                }}
                className="bg-blue-600 hover:bg-blue-700"
              >
                Contact Support
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        {/* Remove/Delete Server Dialog */}
        <AlertDialog open={showRemoveDialog} onOpenChange={setShowRemoveDialog}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle className="flex items-center gap-2">
                <Trash2 className="w-5 h-5 text-red-600 dark:text-red-400" />
                {selectedServer?.status === "owner" ? "Delete Server" : "Remove Server"}
              </AlertDialogTitle>
              <AlertDialogDescription asChild>
                <div className="space-y-4">
                  {selectedServer?.status === "owner" ? (
                    <div className="space-y-3">
                      <p>You are about to permanently delete this server:</p>
                      <div className="p-3 bg-slate-50 dark:bg-slate-900 rounded-lg border">
                        <p className="font-medium">{selectedServer?.name}</p>
                        <p className="text-sm text-slate-600 dark:text-slate-400">{selectedServer?.address}</p>
                      </div>
                      <Alert variant="destructive">
                        <AlertTriangle className="h-4 w-4" />
                        <AlertDescription>
                          <strong>Warning:</strong> This action cannot be undone. The server will be permanently deleted
                          and all users connected to this server will lose access immediately.
                        </AlertDescription>
                      </Alert>
                      <div className="space-y-2">
                        <Label htmlFor="confirm-delete">
                          Type the server name{" "}
                          <code className="bg-muted px-1 py-0.5 rounded text-sm">{selectedServer?.name}</code> to
                          confirm deletion:
                        </Label>
                        <Input
                          id="confirm-delete"
                          placeholder={selectedServer?.name || ""}
                          value={confirmText}
                          onChange={(e) => setConfirmText(e.target.value)}
                          disabled={isRemoving}
                        />
                      </div>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      <p>You are about to remove this server from your list:</p>
                      <div className="p-3 bg-slate-50 dark:bg-slate-900 rounded-lg border">
                        <p className="font-medium">{selectedServer?.name}</p>
                        <p className="text-sm text-slate-600 dark:text-slate-400">{selectedServer?.address}</p>
                      </div>
                      <Alert>
                        <AlertTriangle className="h-4 w-4" />
                        <AlertDescription>
                          This will only remove the server from your personal list. The server itself will continue to
                          run and you can reconnect later if needed.
                        </AlertDescription>
                      </Alert>
                      <div className="space-y-2">
                        <Label htmlFor="confirm-remove">
                          Type <code className="bg-muted px-1 py-0.5 rounded text-sm">remove</code> to confirm:
                        </Label>
                        <Input
                          id="confirm-remove"
                          placeholder="remove"
                          value={confirmText}
                          onChange={(e) => setConfirmText(e.target.value)}
                          disabled={isRemoving}
                        />
                      </div>
                    </div>
                  )}
                </div>
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={isRemoving}>Cancel</AlertDialogCancel>
              <AlertDialogAction
                onClick={handleConfirmRemove}
                disabled={
                  isRemoving ||
                  (selectedServer?.status === "owner"
                    ? confirmText.toLowerCase() !== selectedServer?.name.toLowerCase()
                    : confirmText.toLowerCase() !== "remove")
                }
                className="bg-red-600 hover:bg-red-700"
              >
                {isRemoving ? (
                  <>
                    <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2" />
                    {selectedServer?.status === "owner" ? "Deleting..." : "Removing..."}
                  </>
                ) : (
                  <>
                    <Trash2 className="w-4 h-4 mr-2" />
                    {selectedServer?.status === "owner" ? "Delete Server" : "Remove Server"}
                  </>
                )}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        {/* Create Server Dialog */}
        <AlertDialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Create New Server</AlertDialogTitle>
              <AlertDialogDescription>
                This will redirect you to the server creation page where you can set up a new MCP server.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction
                onClick={() => router.push("/create-server")}
                className="bg-blue-600 hover:bg-blue-700"
              >
                Continue
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </MCPPageLayout>
  )
}
