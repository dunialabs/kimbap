"use client"

import {
  Cloud,
  HardDrive,
  Download,
  Upload,
  FileText,
} from "lucide-react"
import { useState } from "react"

import { AuthLoginDialog } from "@/components/auth-login-dialog"
import { AuthRegistrationDialog } from "@/components/auth-registration-dialog"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { cn } from "@/lib/utils"

interface BackupDialogProps {
  children: React.ReactNode
  isLoggedIn?: boolean
  onLoginRequired?: () => void
}

interface BackupItem {
  id: string
  name: string
  date: string
  size: string
  type: "cloud" | "local"
  clientCount: number
  serverCount: number
}

export function BackupDialog({ children }: BackupDialogProps) {
  const [open, setOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [isLoggedIn, setIsLoggedIn] = useState(false)
  const [activeTab, setActiveTab] = useState("cloud")
  const [selectedBackup, setSelectedBackup] = useState<string | null>(null)
  const [showLoginDialog, setShowLoginDialog] = useState(false)
  const [showRegistrationDialog, setShowRegistrationDialog] = useState(false)

  // Mock backup data
  const [cloudBackups] = useState<BackupItem[]>([
    {
      id: "cloud-1",
      name: "Full System Backup",
      date: "2024-01-15",
      size: "2.3 MB",
      type: "cloud",
      clientCount: 3,
      serverCount: 5
    },
    {
      id: "cloud-2", 
      name: "Weekly Backup",
      date: "2024-01-08",
      size: "1.8 MB",
      type: "cloud",
      clientCount: 2,
      serverCount: 4
    }
  ])
  
  const [localBackups] = useState<BackupItem[]>([
    {
      id: "local-1",
      name: "Manual Backup",
      date: "2024-01-10",
      size: "1.5 MB", 
      type: "local",
      clientCount: 2,
      serverCount: 3
    }
  ])

  const handleLoginSuccess = (_user: { email: string; name?: string }) => {
    setIsLoggedIn(true)
    setShowLoginDialog(false)
  }

  const handleRegistrationSuccess = (_user: { email: string; name?: string }) => {
    setIsLoggedIn(true)
    setShowRegistrationDialog(false)
  }

  const handleBackup = async (_type: "cloud" | "local") => {
    setIsLoading(true)
    
    // Simulate backup creation
    await new Promise(resolve => setTimeout(resolve, 3000))
    
    setIsLoading(false)
    // In real app, would refresh backup list
  }

  const handleRestore = async (_backupId: string) => {
    setIsLoading(true)
    
    // Simulate restore
    await new Promise(resolve => setTimeout(resolve, 3000))
    
    setIsLoading(false)
    setOpen(false)
    // In real app, would reload the page or refresh data
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short', 
      day: 'numeric'
    })
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {children}
      </DialogTrigger>
      <ScrollableDialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <HardDrive className="h-5 w-5" />
            Backup & Restore
          </DialogTitle>
          <DialogDescription>
            Create or restore backups.
          </DialogDescription>
        </DialogHeader>

        <div>
        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="cloud">
              <Cloud className="w-4 h-4 mr-2" />
              Cloud
            </TabsTrigger>
            <TabsTrigger value="local">
              <HardDrive className="w-4 h-4 mr-2" />
              Local
            </TabsTrigger>
          </TabsList>

          <TabsContent value="cloud" className="space-y-4">
            {!isLoggedIn ? (
              <Card className="p-6">
                <div className="text-center space-y-4">
                  <div className="w-16 h-16 mx-auto bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center">
                    <Cloud className="w-8 h-8 text-blue-600 dark:text-blue-400" />
                  </div>
                  <div>
                    <h3 className="text-lg font-semibold">Sign In Required</h3>
                    <p className="text-sm text-muted-foreground">
                      Sign in to use cloud backups.
                    </p>
                  </div>
                  <div className="space-y-3 max-w-sm mx-auto">
                    <Button 
                      onClick={() => setShowLoginDialog(true)}
                      className="w-full"
                    >
                      Sign In
                    </Button>
                    <div className="text-center">
                      <p className="text-xs text-muted-foreground">
                        Don't have an account?{" "}
                        <button
                          type="button"
                          onClick={() => setShowRegistrationDialog(true)}
                          className="text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 font-medium"
                        >
                          Create account
                        </button>
                      </p>
                    </div>
                  </div>
                </div>
              </Card>
            ) : (
              <div className="space-y-4">
                <Alert>
                  <Cloud className="h-4 w-4" />
                  <AlertDescription>
                    Cloud backups are encrypted and available on any signed-in device.
                  </AlertDescription>
                </Alert>

                <div className="flex justify-between items-center">
                  <h3 className="text-sm font-medium">Cloud Backups</h3>
                  <Button 
                    size="sm" 
                    onClick={() => handleBackup("cloud")}
                    disabled={isLoading}
                  >
                    {isLoading ? "Creating..." : "Create"}
                  </Button>
                </div>

                <div className="space-y-2 max-h-60 overflow-y-auto">
                  {cloudBackups.map((backup) => (
                    <Card
                      key={backup.id}
                      className={cn(
                        "cursor-pointer transition-colors",
                        selectedBackup === backup.id ? "ring-2 ring-blue-500 bg-blue-50 dark:bg-blue-950/30" : "hover:bg-gray-50 dark:hover:bg-gray-800/50"
                      )}
                      onClick={() => setSelectedBackup(selectedBackup === backup.id ? null : backup.id)}
                    >
                      <CardContent className="p-4">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-3 flex-1">
                            <FileText className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                            <div>
                              <div className="font-medium text-sm">{backup.name}</div>
                              <div className="text-xs text-muted-foreground">
                                {formatDate(backup.date)} • {backup.size}
                              </div>
                            </div>
                          </div>
                          <div className="flex items-center gap-2">
                            <Badge variant="outline" className="text-xs">
                              {backup.clientCount} clients
                            </Badge>
                            <Badge variant="outline" className="text-xs">
                              {backup.serverCount} servers
                            </Badge>
                          </div>
                        </div>
                        {selectedBackup === backup.id && (
                          <div className="mt-3 pt-3 border-t">
                            <Button 
                              size="sm" 
                              onClick={(e) => {
                                e.stopPropagation()
                                handleRestore(backup.id)
                              }}
                              disabled={isLoading}
                              className="w-full"
                            >
                              {isLoading ? "Restoring..." : "Restore"}
                            </Button>
                          </div>
                        )}
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </div>
            )}
          </TabsContent>

          <TabsContent value="local" className="space-y-4">
            <Alert>
              <HardDrive className="h-4 w-4" />
              <AlertDescription>
                Local backups stay on this device.
              </AlertDescription>
            </Alert>

            <div className="flex justify-between items-center">
              <h3 className="text-sm font-medium">Local Backups</h3>
              <div className="flex gap-2">
                <Button 
                  size="sm" 
                  variant="outline"
                  onClick={() => handleBackup("local")}
                  disabled={isLoading}
                >
                  <Download className="w-3 h-3 mr-1" />
                  Export
                </Button>
                <Button 
                  size="sm"
                  onClick={() => {
                    const input = document.createElement('input')
                    input.type = 'file'
                    input.accept = '.json'
                    input.onchange = (e) => {
                      const file = (e.target as HTMLInputElement).files?.[0]
                      if (file) {
                        // TODO: Handle file import
                      }
                    }
                    input.click()
                  }}
                  disabled={isLoading}
                >
                  <Upload className="w-3 h-3 mr-1" />
                  Import
                </Button>
              </div>
            </div>

            <div className="space-y-2 max-h-60 overflow-y-auto">
              {localBackups.map((backup) => (
                <Card
                  key={backup.id}
                  className={cn(
                    "cursor-pointer transition-colors",
                    selectedBackup === backup.id ? "ring-2 ring-blue-500 bg-blue-50 dark:bg-blue-950/30" : "hover:bg-gray-50 dark:hover:bg-gray-800/50"
                  )}
                  onClick={() => setSelectedBackup(selectedBackup === backup.id ? null : backup.id)}
                >
                  <CardContent className="p-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3 flex-1">
                        <FileText className="w-4 h-4 text-slate-600 dark:text-slate-400" />
                        <div>
                          <div className="font-medium text-sm">{backup.name}</div>
                          <div className="text-xs text-muted-foreground">
                            {formatDate(backup.date)} • {backup.size}
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge variant="outline" className="text-xs">
                          {backup.clientCount} clients
                        </Badge>
                        <Badge variant="outline" className="text-xs">
                          {backup.serverCount} servers
                        </Badge>
                      </div>
                    </div>
                    {selectedBackup === backup.id && (
                      <div className="mt-3 pt-3 border-t">
                        <Button 
                          size="sm" 
                          onClick={(e) => {
                            e.stopPropagation()
                            handleRestore(backup.id)
                          }}
                          disabled={isLoading}
                          className="w-full"
                        >
                          {isLoading ? "Restoring..." : "Restore"}
                        </Button>
                      </div>
                    )}
                  </CardContent>
                </Card>
              ))}
              
              {localBackups.length === 0 && (
                <div className="text-center py-8">
                  <HardDrive className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                  <p className="text-muted-foreground">No local backups found</p>
                  <p className="text-xs text-muted-foreground mt-1">
                    Create or import a backup to get started
                  </p>
                </div>
              )}
            </div>
          </TabsContent>
        </Tabs>
        </div>

        <DialogFooter className="border-t pt-4">
          <Button variant="outline" onClick={() => setOpen(false)} disabled={isLoading}>
            Close
          </Button>
        </DialogFooter>
      </ScrollableDialogContent>

      {/* Auth Dialogs */}
      <AuthLoginDialog
        open={showLoginDialog}
        onOpenChange={setShowLoginDialog}
        onShowRegistrationDialog={() => {
          setShowLoginDialog(false)
          setShowRegistrationDialog(true)
        }}
        onLoginSuccess={handleLoginSuccess}
      />

      <AuthRegistrationDialog
        open={showRegistrationDialog}
        onOpenChange={setShowRegistrationDialog}
        onShowLoginDialog={() => {
          setShowRegistrationDialog(false)
          setShowLoginDialog(true)
        }}
        onRegistrationSuccess={handleRegistrationSuccess}
      />
    </Dialog>
  )
}
