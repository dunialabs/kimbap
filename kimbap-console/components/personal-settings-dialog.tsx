"use client"

import {
  LogIn,
  LogOut,
  Trash2,
  Key,
} from "lucide-react"
import { useState, useEffect } from "react"
import { toast } from "sonner"

import { AuthLoginDialog } from "@/components/auth-login-dialog"
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
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
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

interface PersonalSettingsDialogProps {
  children: React.ReactNode
}

export function PersonalSettingsDialog({ children }: PersonalSettingsDialogProps) {
  const [open, setOpen] = useState(false)
  const [isLoggedIn, setIsLoggedIn] = useState(false)
  const [oldMasterPassword, setOldMasterPassword] = useState("")
  const [newMasterPassword, setNewMasterPassword] = useState("")
  const [masterPasswordError, setMasterPasswordError] = useState("")

  const [showClearDataDialog, setShowClearDataDialog] = useState(false)
  const [showMasterPasswordDialog, setShowMasterPasswordDialog] = useState(false)
  const [showLoginDialog, setShowLoginDialog] = useState(false)
  const [confirmationText, setConfirmationText] = useState("")
  const [clearDataError, setClearDataError] = useState("")

  useEffect(() => {
    // Check authentication status
    const authStatus = localStorage.getItem("clientManagementAuth")
    setIsLoggedIn(authStatus === "true")
  }, [])

  const handleMasterPasswordUpdate = async () => {
    setMasterPasswordError("")

    if (!oldMasterPassword || !newMasterPassword) {
      setMasterPasswordError("Both old and new passwords are required")
      return
    }

    if (newMasterPassword.length < 10) {
      setMasterPasswordError("New master password must be at least 10 characters long")
      return
    }

    const storedDataStr = localStorage.getItem("clientManagementMasterPassword")
    if (storedDataStr) {
      try {
        const { CryptoUtils } = await import("@/lib/crypto")
        const storedData = JSON.parse(storedDataStr)
        if (storedData.hash && storedData.salt) {
          const salt = new Uint8Array(atob(storedData.salt).split("").map(c => c.charCodeAt(0)))
          const isValid = await CryptoUtils.verifyPasswordWithSalt(oldMasterPassword, storedData.hash, salt)
          if (!isValid) {
            setMasterPasswordError("Old master password is incorrect")
            return
          }
        } else if (oldMasterPassword !== storedDataStr) {
          setMasterPasswordError("Old master password is incorrect")
          return
        }
      } catch {
        if (oldMasterPassword !== storedDataStr) {
          setMasterPasswordError("Old master password is incorrect")
          return
        }
      }
    }

    const { CryptoUtils } = await import("@/lib/crypto")
    const salt = CryptoUtils.generateSalt()
    const hash = await CryptoUtils.hashPasswordWithSalt(newMasterPassword, salt)
    const hashedData = {
      hash,
      salt: btoa(String.fromCharCode.apply(null, Array.from(salt)))
    }
    localStorage.setItem("clientManagementMasterPassword", JSON.stringify(hashedData))
    localStorage.setItem("clientManagementMasterAuth", "true")

    setOldMasterPassword("")
    setNewMasterPassword("")
    setMasterPasswordError("")
    setShowMasterPasswordDialog(false)

    toast.success("Master password updated.")
  }

  const handleLoginSuccess = (_user: { email: string; name?: string }) => {
    // Set logged in state
    localStorage.setItem("clientManagementAuth", "true")
    setIsLoggedIn(true)
    setShowLoginDialog(false)
  }

  const handleLogout = () => {
    localStorage.setItem("clientManagementAuth", "guest")
    setIsLoggedIn(false)
  }

  const handleClearLocalData = () => {
    if (confirmationText !== "DELETE MY DATA") {
      setClearDataError('Please type "DELETE MY DATA" to confirm')
      return
    }

    const masterAuth = localStorage.getItem("clientManagementMasterAuth")
    const masterPasswordData = localStorage.getItem("clientManagementMasterPassword")

    localStorage.clear()

    if (masterAuth && masterPasswordData) {
      localStorage.setItem("clientManagementMasterAuth", masterAuth)
      localStorage.setItem("clientManagementMasterPassword", masterPasswordData)
    }

    localStorage.setItem("clientManagementAuth", "guest")
    setIsLoggedIn(false)
    setShowClearDataDialog(false)
    setConfirmationText("")
    setClearDataError("")

    toast.success("Local data cleared.")
  }

  return (
    <>
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {children}
      </DialogTrigger>
      <ScrollableDialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="text-base">Personal Settings</DialogTitle>
          <DialogDescription className="text-xs">
            Manage account and local data.
          </DialogDescription>
        </DialogHeader>

        <div>
        {/* Settings Cards */}
        <div className="space-y-3">
          {/* Account Section */}
          <Card className="shadow-sm">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">
                Account
              </CardTitle>
              <CardDescription className="text-xs">
                {isLoggedIn ? "Logged in as user@example.com" : "Not logged in"}
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-0">
              {!isLoggedIn ? (
                <Button size="sm" className="w-full text-xs" onClick={() => setShowLoginDialog(true)}>
                  <LogIn className="w-3 h-3 mr-2" />
                  Sign In
                </Button>
              ) : (
                <Button
                  onClick={handleLogout}
                  variant="outline"
                  size="sm"
                  className="w-full text-xs"
                >
                  <LogOut className="w-3 h-3 mr-2" />
                  Logout
                </Button>
              )}
            </CardContent>
          </Card>

          {/* Security Section */}
          <Card className="shadow-sm">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">
                Security
              </CardTitle>
              <CardDescription className="text-xs">
                Master password protection
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-0">
              <Dialog open={showMasterPasswordDialog} onOpenChange={(v) => { setShowMasterPasswordDialog(v); if (!v) setMasterPasswordError("") }}>
                <DialogTrigger asChild>
                  <Button variant="outline" size="sm" className="w-full text-xs">
                    <Key className="w-3 h-3 mr-2" />
                    Change Master Password
                  </Button>
                </DialogTrigger>
                <DialogContent className="sm:max-w-md">
                  <DialogHeader>
                    <DialogTitle className="text-base">Change Master Password</DialogTitle>
                    <DialogDescription className="text-xs">
                      Enter your current and new password.
                    </DialogDescription>
                  </DialogHeader>
                  <div className="space-y-3">
                    <div className="space-y-1">
                      <Label htmlFor="old-password" className="text-xs">Current Master Password</Label>
                      <Input
                        id="old-password"
                        type="password"
                        placeholder="Current password"
                        value={oldMasterPassword}
                        onChange={(e) => { setOldMasterPassword(e.target.value); setMasterPasswordError("") }}
                        className="text-xs"
                      />
                    </div>
                    <div className="space-y-1">
                      <Label htmlFor="new-password" className="text-xs">New Master Password</Label>
                      <Input
                        id="new-password"
                        type="password"
                        placeholder="New password"
                        value={newMasterPassword}
                        onChange={(e) => {
                          setNewMasterPassword(e.target.value)
                          setMasterPasswordError("")
                        }}
                        className="text-xs"
                      />
                    </div>
                    {masterPasswordError && (
                      <p className="text-xs text-destructive">{masterPasswordError}</p>
                    )}
                  </div>
                  <DialogFooter>
                    <Button
                      onClick={handleMasterPasswordUpdate}
                      disabled={!oldMasterPassword || !newMasterPassword}
                      size="sm"
                      className="text-xs"
                    >
                      <Key className="w-3 h-3 mr-2" />
                      Save Password
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </CardContent>
          </Card>

          {/* Data Management */}
          <Card className="shadow-sm">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm">
                Data Management
              </CardTitle>
              <CardDescription className="text-xs">
                Local data and preferences
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-0">
              <AlertDialog open={showClearDataDialog} onOpenChange={setShowClearDataDialog}>
                <Button
                  variant="destructive"
                  size="sm"
                  className="w-full text-xs"
                  onClick={() => setShowClearDataDialog(true)}
                >
                  <Trash2 className="w-3 h-3 mr-2" />
                  Clear Local Data
                </Button>
                <AlertDialogContent className="sm:max-w-md">
                  <AlertDialogHeader>
                    <AlertDialogTitle className="text-base">Clear Local Data</AlertDialogTitle>
                    <AlertDialogDescription className="text-xs">
                      This will clear all locally stored configurations and settings. Your master password will be preserved.
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <div className="space-y-2">
                    <Label htmlFor="confirmation" className="text-xs font-medium">
                      Type "DELETE MY DATA" to confirm:
                    </Label>
                    <Input
                      id="confirmation"
                      placeholder="DELETE MY DATA"
                      value={confirmationText}
                      onChange={(e) => { setConfirmationText(e.target.value); setClearDataError("") }}
                      className="text-xs"
                    />
                    {clearDataError && (
                      <p className="text-xs text-destructive">{clearDataError}</p>
                    )}
                  </div>
                  <AlertDialogFooter>
                    <AlertDialogCancel
                      className="text-xs"
                      onClick={() => {
                        setConfirmationText("")
                        setClearDataError("")
                        setShowClearDataDialog(false)
                      }}
                    >
                      Cancel
                    </AlertDialogCancel>
                    <AlertDialogAction
                      onClick={handleClearLocalData}
                      disabled={confirmationText !== "DELETE MY DATA"}
                      className="bg-red-600 hover:bg-red-700 text-xs"
                    >
                      Clear Data
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </CardContent>
          </Card>
        </div>
        </div>
      </ScrollableDialogContent>
    </Dialog>

    <AuthLoginDialog
      open={showLoginDialog}
      onOpenChange={setShowLoginDialog}
      onLoginSuccess={handleLoginSuccess}
    />
    </>
  )
}
