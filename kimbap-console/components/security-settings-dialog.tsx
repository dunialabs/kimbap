"use client"

import { Shield, CheckCircle, Info } from "lucide-react"
import { useState } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"
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
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

interface SecuritySettings {
  requirePasswordOnStartup: boolean
  autoLockTimeout: number // in minutes, 0 means never
  requirePasswordForSensitiveOps: boolean
  enableFingerprintUnlock: boolean
}

interface SecuritySettingsDialogProps {
  onApplySettings?: (settings: SecuritySettings) => void
}

export function SecuritySettingsDialog({ onApplySettings }: SecuritySettingsDialogProps) {
  const [open, setOpen] = useState(false)
  const [settings, setSettings] = useState<SecuritySettings>({
    requirePasswordOnStartup: true,
    autoLockTimeout: 30, // 30 minutes default
    requirePasswordForSensitiveOps: true,
    enableFingerprintUnlock: false,
  })

  const [fingerprintAvailable] = useState(true) // Simulate fingerprint availability

  const handleApply = () => {
    if (onApplySettings) {
      onApplySettings(settings)
    }
    setOpen(false)
  }

  const timeoutOptions = [
    { value: 0, label: "Never" },
    { value: 5, label: "5 minutes" },
    { value: 15, label: "15 minutes" },
    { value: 30, label: "30 minutes" },
    { value: 60, label: "1 hour" },
    { value: 120, label: "2 hours" },
    { value: 240, label: "4 hours" },
  ]

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm" className="bg-transparent">
          <Shield className="w-3 h-3 mr-1.5" />
          Security
        </Button>
      </DialogTrigger>
      <ScrollableDialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Security Settings
          </DialogTitle>
          <DialogDescription>Manage security settings for this server.</DialogDescription>
        </DialogHeader>

        <div>
        <Tabs defaultValue="auth" className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="auth">Authentication</TabsTrigger>
            <TabsTrigger value="session">Session Control</TabsTrigger>
          </TabsList>

          <TabsContent value="auth" className="space-y-4">
            {/* Startup Authentication */}
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">
                  Startup Authentication
                </CardTitle>
                <CardDescription>Set when master password is required.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="space-y-1">
                    <Label htmlFor="startup-password">Require password on startup</Label>
                    <p className="text-xs text-muted-foreground">
                      Require master password at startup.
                    </p>
                  </div>
                  <Switch
                    id="startup-password"
                    checked={settings.requirePasswordOnStartup}
                    onCheckedChange={(checked) =>
                      setSettings((prev) => ({ ...prev, requirePasswordOnStartup: checked }))
                    }
                  />
                </div>

                <Separator />

                <div className="flex items-center justify-between">
                  <div className="space-y-1">
                    <Label htmlFor="sensitive-ops">Require password for sensitive operations</Label>
                    <p className="text-xs text-muted-foreground">
                      Require master password for sensitive operations.
                    </p>
                  </div>
                  <Switch
                    id="sensitive-ops"
                    checked={settings.requirePasswordForSensitiveOps}
                    onCheckedChange={(checked) =>
                      setSettings((prev) => ({ ...prev, requirePasswordForSensitiveOps: checked }))
                    }
                  />
                </div>
              </CardContent>
            </Card>

            {/* Biometric Authentication */}
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">
                  Biometric Authentication
                </CardTitle>
                <CardDescription>Use biometric unlock for routine access.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="space-y-1">
                    <Label htmlFor="fingerprint-unlock">Enable fingerprint unlock</Label>
                    <p className="text-xs text-muted-foreground">
                      Use fingerprint to unlock instead of typing master password
                    </p>
                  </div>
                  <Switch
                    id="fingerprint-unlock"
                    checked={settings.enableFingerprintUnlock}
                    onCheckedChange={(checked) =>
                      setSettings((prev) => ({ ...prev, enableFingerprintUnlock: checked }))
                    }
                    disabled={!fingerprintAvailable}
                  />
                </div>

                {fingerprintAvailable ? (
                  <Alert>
                    <CheckCircle className="h-4 w-4" />
                    <AlertDescription>
                      <div className="space-y-1">
                        <p className="font-medium text-green-900 dark:text-green-100">Fingerprint Available</p>
                        <p className="text-sm text-green-800 dark:text-green-200">
                          This device supports fingerprint unlock.
                        </p>
                      </div>
                    </AlertDescription>
                  </Alert>
                ) : (
                  <Alert>
                    <Info className="h-4 w-4" />
                    <AlertDescription>
                      <div className="space-y-1">
                        <p className="font-medium">Fingerprint Not Available</p>
                        <p className="text-sm text-muted-foreground">
                          Fingerprint unlock is unavailable on this device.
                        </p>
                      </div>
                    </AlertDescription>
                  </Alert>
                )}

                {settings.enableFingerprintUnlock && (
                  <div className="p-3 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-md">
                    <p className="text-xs text-blue-800 dark:text-blue-200">
                      <strong>Note:</strong> Sensitive operations may still require a password.
                    </p>
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="session" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">
                  Session Management
                </CardTitle>
                <CardDescription>Set lock behavior and timeout.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="auto-lock">Auto-lock timeout</Label>
                  <Select
                    value={settings.autoLockTimeout.toString()}
                    onValueChange={(value) =>
                      setSettings((prev) => ({ ...prev, autoLockTimeout: Number.parseInt(value) }))
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {timeoutOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value.toString()}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <p className="text-xs text-muted-foreground">
                    Lock after inactivity.
                  </p>
                </div>

                {settings.autoLockTimeout > 0 && (
                  <div className="p-3 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-md">
                    <p className="text-xs text-blue-800 dark:text-blue-200">
                      <strong>Note:</strong> Session locks after {settings.autoLockTimeout} minutes.
                    </p>
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>

        <div className="bg-slate-50 dark:bg-slate-800/50 p-4 rounded-lg">
          <h4 className="text-sm font-medium mb-2 text-slate-900 dark:text-slate-100">Settings Summary</h4>
          <div className="space-y-1 text-xs text-slate-600 dark:text-slate-400">
            <div className="flex justify-between">
              <span>Startup password:</span>
              <span className="font-medium">{settings.requirePasswordOnStartup ? "Required" : "Not required"}</span>
            </div>
            <div className="flex justify-between">
              <span>Fingerprint unlock:</span>
              <span className="font-medium">
                {settings.enableFingerprintUnlock ? "Enabled" : "Disabled"}
                {!fingerprintAvailable && settings.enableFingerprintUnlock && " (Not available)"}
              </span>
            </div>
            <div className="flex justify-between">
              <span>Auto-lock:</span>
              <span className="font-medium">
                {settings.autoLockTimeout === 0 ? "Never" : `${settings.autoLockTimeout} minutes`}
              </span>
            </div>
            <div className="flex justify-between">
              <span>Sensitive operations:</span>
              <span className="font-medium">
                {settings.requirePasswordForSensitiveOps ? "Password required" : "No password"}
              </span>
            </div>
          </div>
        </div>

        </div>
        <DialogFooter className="border-t pt-4">
          <Button variant="outline" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button onClick={handleApply} className="bg-slate-900 hover:bg-slate-800 dark:bg-slate-100 dark:hover:bg-slate-200 dark:text-slate-900">
            Save
          </Button>
        </DialogFooter>
      </ScrollableDialogContent>
    </Dialog>
  )
}
