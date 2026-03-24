'use client';

import {
  Download,
  PowerOff,
  Trash2,
  AlertTriangle,
  Lock,
  Cloud,
  Upload,
  Calendar,
  Crown,
  ExternalLink,
  Loader2,
  Shield,
  Key,
  Copy,
} from 'lucide-react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState, useEffect, useCallback } from 'react';
import { toast } from 'sonner';

import { AuthLoginDialog } from '@/components/auth-login-dialog';
import { RestoreDialog } from '@/components/restore-dialog';
import { AuthRegistrationDialog } from '@/components/auth-registration-dialog';
import { PlanSelectionDialog } from '@/components/plan-selection-dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { MasterPasswordManager } from '@/lib/crypto';
import { api } from '@/lib/api-client';
import type { PlanId } from '@/lib/plan-config';
import { useMasterPassword } from '@/contexts/master-password-context';

export default function ServerControlPage() {
  const { requestMasterPassword } = useMasterPassword();
  const [resetConfirmText, setResetConfirmText] = useState('');
  const [isResetDialogOpen, setIsResetDialogOpen] = useState(false);
  const [loginPurpose, setLoginPurpose] = useState('');
  const [serverInfo, setServerInfo] = useState<any>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [serverName, setServerName] = useState('');
  const [originalServerName, setOriginalServerName] = useState('');
  const [serverStatus, setServerStatus] = useState<number>(1); // 1-running, 2-stopped
  const [isStopDialogOpen, setIsStopDialogOpen] = useState(false);
  const [isTogglingServer, setIsTogglingServer] = useState(false);

  // Restore dialog states
  const [showRestoreDialog, setShowRestoreDialog] = useState(false);
  const [showPlanDialog, setShowPlanDialog] = useState(false);
  const [showRegistrationDialog, setShowRegistrationDialog] = useState(false);
  const [showAuthLoginDialog, setShowAuthLoginDialog] = useState(false);
  const [currentServerPlan] = useState<PlanId>('free'); // This would come from server context

  // Backup dialog states
  const [showBackupDialog, setShowBackupDialog] = useState(false);
  const [backupMasterPassword, setBackupMasterPassword] = useState('');
  const [isBackingUp, setIsBackingUp] = useState(false);
  const [passwordError, setPasswordError] = useState('');

  const router = useRouter();

  const fetchServerInfo = useCallback(async () => {
    try {
      setIsLoading(true);
      // Use protocol 10002 to get proxy server info
      const response = await api.servers.getInfo();

      if (response.data?.data) {
        const data = response.data.data;
        setServerInfo({
          id: data.proxyId, // Store proxyId as id for consistency
          proxyId: data.proxyId, // Also store as proxyId
          name: data.proxyName,
          proxyName: data.proxyName,
          status: data.status,
          proxyKey: data.proxyKey,
          createAt: data.createAt,
          fingerprint: data.fingerprint, // Store hardware fingerprint
        });
        setServerName(data.proxyName || '');
        setOriginalServerName(data.proxyName || '');
        setServerStatus(data.status || 1);
      }
    } catch (error: any) {
      // Failed to fetch server info:
      toast.error('Could not load server info');
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch server info on mount
  useEffect(() => {
    void fetchServerInfo();
  }, [fetchServerInfo]);

  const handleSaveChanges = async () => {
    try {
      setIsSaving(true);
      const { api } = await import('@/lib/api-client');

      await api.servers.editInfo({
        handleType: 1, // 1-edit base info
        proxyId: serverInfo?.proxyId,
        proxyName: serverName,
      });

      // Update localStorage cache with new server name
      const selectedServer = localStorage.getItem('selectedServer');
      if (selectedServer) {
        try {
          const parsedServer = JSON.parse(selectedServer);
          parsedServer.name = serverName;
          parsedServer.proxyName = serverName;
          localStorage.setItem('selectedServer', JSON.stringify(parsedServer));
        } catch (error) {
          // Failed to update localStorage:
        }
      }

      setOriginalServerName(serverName);
      toast.success('Server name updated');
      await fetchServerInfo(); // Refresh server info
    } catch (error: any) {
      // Failed to update server info:
      toast.error((error as any).userMessage || 'Could not update server name');
    } finally {
      setIsSaving(false);
    }
  };

  const executeServerToggle = async () => {
    try {
      setIsTogglingServer(true);
      const { api } = await import('@/lib/api-client');
      const isRunning = serverStatus === 1;

      await api.servers.editInfo({
        handleType: isRunning ? 3 : 2, // 2-start server, 3-stop server
        proxyId: serverInfo?.proxyId,
      });

      // Update local status immediately
      const newStatus = isRunning ? 2 : 1;
      setServerStatus(newStatus);
      toast.success(isRunning ? 'Server stopped' : 'Server started');

      // Refresh server info from API to ensure consistency
      await fetchServerInfo();

      // Close stop dialog if it was open
      setIsStopDialogOpen(false);
    } catch (error: any) {
      // Failed to toggle server:
      toast.error((error as any).userMessage || 'Could not change server status');
    } finally {
      setIsTogglingServer(false);
    }
  };

  const handleResetServer = async () => {
    if (resetConfirmText.toLowerCase() !== 'reset server') {
      return;
    }

    // Close the reset dialog first
    setIsResetDialogOpen(false);

    // Show master password dialog
    requestMasterPassword({
      title: 'Reset Server',
      description: 'Enter your master password to confirm.',
      onConfirm: async (password) => {
        try {
          // Call protocol 10003 with handleType 8 (delete server)
          const response = await api.servers.editInfo({
            handleType: 8, // 8-delete server (reset server)
            proxyId: serverInfo?.proxyId || 0,
            proxyName: serverInfo?.proxyName || '',
            proxyKey: serverInfo?.proxyKey || '',
            masterPwd: password, // Pass master password for validation
          });

          if (!response.data) {
            throw new Error('Could not reset server');
          }

          // Clear local storage
          localStorage.removeItem('selectedServer');

          // Clear master password verification since server is reset
          sessionStorage.removeItem('masterPasswordVerified');

          // Set flag for showing reset success dialog on welcome page
          sessionStorage.setItem('showResetSuccess', 'true');
          localStorage.removeItem('userid');

          // Redirect to home
          router.push('/');
        } catch (error: any) {
          // Failed to reset server:
          toast.error('Could not reset server. Try again.');
          throw error; // Re-throw to keep dialog open
        }
      },
    });

    // Reset confirm text for next time
    setResetConfirmText('');
  };

  const handleLoginRequired = (purpose: string) => {
    setLoginPurpose(purpose);
    setShowAuthLoginDialog(true);
  };

  const handleLoginSuccess = async () => {
    setShowAuthLoginDialog(false);

    // Proceed with the original action
    switch (loginPurpose) {
      case 'backup':
        await handleBackupToCloud();
        break;
      case 'restore':
        setShowRestoreDialog(true);
        break;
      default:
        break;
    }
  };

  const handleBackupToLocal = async (masterPassword: string) => {
    try {
      setIsBackingUp(true);
      setPasswordError(''); // Clear any previous error

      const { api } = await import('@/lib/api-client');

      const response = await api.servers.backupServerInfoToLocal({
        masterPwd: masterPassword,
      });

      // Handle the backup download
      if (response.data?.data) {
        const backupTypeLabel = 'full';
        const backupTimestamp = new Date().toISOString();

        // Create backup metadata with the actual data
        const backupData = {
          metadata: {
            backupTime: backupTimestamp,
            backupVersion: '0.1',
            backupType: backupTypeLabel,
            dataSize: JSON.stringify(response.data.data).length,
          },
          data: response.data.data,
        };

        const element = document.createElement('a');
        const file = new Blob([JSON.stringify(backupData, null, 2)], {
          type: 'application/json',
        });
        element.href = URL.createObjectURL(file);
        element.download = `Kimbap-backup-${backupTypeLabel}-${backupTimestamp
          .slice(0, 19)
          .replace(/:/g, '-')}.backup`;
        document.body.appendChild(element);
        element.click();
        document.body.removeChild(element);
        URL.revokeObjectURL(element.href);

        toast.success('Backup downloaded');

        // Close dialog and reset state on success
        setShowBackupDialog(false);
        setBackupMasterPassword('');
        setPasswordError('');
      }
    } catch (error: any) {
      // Failed to backup:

      // Check if it's a password error
      const errorMessage = (error as any).userMessage || 'Could not create backup';
      const statusCode = error.response?.status;

      if (statusCode === 401 || errorMessage.toLowerCase().includes('password')) {
        // Password error - show in dialog
        setPasswordError('Incorrect master password.');
      } else {
        // Other errors - show toast
        toast.error(errorMessage);
      }
    } finally {
      setIsBackingUp(false);
    }
  };

  const handleBackupToCloud = async () => {
    try {
      const { api } = await import('@/lib/api-client');

      await api.servers.editInfo({
        handleType: 4, // 4-backup to cloud
        proxyId: serverInfo?.proxyId,
      });

      toast.success('Cloud backup complete');
    } catch (error: any) {
      // Failed to backup to cloud:
      toast.error((error as any).userMessage || 'Could not create cloud backup');
    }
  };

  const handleRestoreFromBackup = () => {
    setShowRestoreDialog(true);
  };

  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-4 mb-2">
        <p className="text-sm text-amber-800 dark:text-amber-300">
          <strong>Deprecated:</strong> This page has been replaced by{' '}
          <a href="/dashboard/settings" className="underline font-medium">
            Settings
          </a>
          . It will be removed in a future release.
        </p>
      </div>
      <div className="space-y-0">
        <h1 className="text-[30px] font-bold tracking-tight">Server Control</h1>
        <p className="text-base text-muted-foreground">Manage server settings and backups.</p>
      </div>

      {/* Name and Server Status - Top Section */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3 items-stretch">
        <Card className="h-full">
          <CardContent className="p-4 flex flex-col gap-1 h-full justify-between">
            <CardTitle>Name</CardTitle>
            <Label htmlFor="server-name" className="sr-only">
              Name
            </Label>
            <div className="flex items-center gap-2">
              <Input
                id="server-name"
                value={serverName}
                onChange={(e) => setServerName(e.target.value)}
                onBlur={() => {
                  if (serverName !== originalServerName && serverName.trim()) {
                    handleSaveChanges();
                  } else if (!serverName.trim()) {
                    setServerName(originalServerName);
                  }
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.currentTarget.blur();
                    if (serverName !== originalServerName && serverName.trim()) {
                      handleSaveChanges();
                    } else if (!serverName.trim()) {
                      setServerName(originalServerName);
                    }
                  }
                }}
                disabled={isLoading || isSaving}
                placeholder={isLoading ? 'Loading...' : 'Enter server name'}
                className="flex-1"
              />
              {isSaving && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
            </div>
          </CardContent>
        </Card>

        <Card className="h-full">
          <CardContent className="p-4 flex flex-col gap-1 h-full justify-between">
            <CardTitle>Server Status</CardTitle>
            <div className="flex items-center min-h-[40px]">
              {isLoading ? (
                <div className="flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span className="text-muted-foreground">Loading server status...</span>
                </div>
              ) : (
                <div className="flex items-center flex-wrap gap-2">
                  <span className="relative flex h-3 w-3">
                    {serverStatus === 1 ? (
                      <>
                        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                        <span className="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
                      </>
                    ) : (
                      <span className="relative inline-flex rounded-full h-3 w-3 bg-gray-400 dark:bg-gray-500"></span>
                    )}
                  </span>
                  <span className="font-medium">
                    {serverStatus === 1 ? 'Kimbap Core Running' : 'Stopped'}
                  </span>
                  {serverInfo?.createAt && (
                    <span className="text-sm text-muted-foreground">
                      Created: {new Date(serverInfo.createAt * 1000).toLocaleDateString()}
                    </span>
                  )}
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="flex flex-col sm:flex-row items-start sm:justify-between space-y-4 sm:space-y-0 pb-4">
          <div>
            <CardTitle>Backups</CardTitle>
            <CardDescription>Back up and restore server configuration.</CardDescription>
          </div>
          <div className="flex flex-wrap gap-2">
            {/* <Button
              variant="outline"
              size="sm"
              onClick={() => handleLoginRequired("backup")}
              disabled={isLoading}
            >
              <Lock className="mr-2 h-3 w-3" />
              <Cloud className="ml-1 h-3 w-3" />
              Backup to Cloud
            </Button> */}
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowBackupDialog(true)}
              disabled={isLoading}
            >
              <Download className="mr-2 h-3 w-3" />
              Create Backup
            </Button>
            <Button size="sm" onClick={handleRestoreFromBackup} disabled={isLoading}>
              <Upload className="mr-2 h-3 w-3" />
              Restore Backup
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="p-3 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-md">
            <p className="text-xs text-blue-800 dark:text-blue-200">
              <strong>Privacy Notice:</strong> Backups contain server configuration only. Your data
              is not accessed or read. All backups are encrypted.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Server Management Actions */}
      <Card className="flex flex-col sm:flex-row justify-between sm:items-center">
        <CardHeader>
          <CardTitle>Danger Zone</CardTitle>
          <CardDescription>Irreversible operations</CardDescription>
        </CardHeader>
        <CardContent className="p-4">
          <div className="flex gap-3">
            {/* Reset Server */}
            <Dialog
              open={isResetDialogOpen}
              onOpenChange={(open) => {
                setIsResetDialogOpen(open);
                if (!open) {
                  setResetConfirmText('');
                }
              }}
            >
              <DialogTrigger asChild>
                <Button variant="destructive">
                  <Trash2 className="mr-2 h-4 w-4" />
                  Reset Server
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle className="flex items-center gap-2 text-red-600 dark:text-red-400">
                    <AlertTriangle className="h-5 w-5" />
                    Reset Server
                  </DialogTitle>
                  <DialogDescription>
                    This will reset the server to its initial state, clearing all data and returning
                    you to the welcome setup, including:
                  </DialogDescription>
                </DialogHeader>
                <form
                  onSubmit={(e) => {
                    e.preventDefault();
                    if (resetConfirmText.toLowerCase() === 'reset server') handleResetServer();
                  }}
                >
                  <div className="space-y-4">
                    <ul className="text-sm text-muted-foreground space-y-1 ml-4">
                      <li>• All tool configurations</li>
                      <li>• Access tokens and permissions</li>
                      <li>• Network access settings</li>
                      <li>• Usage logs and history</li>
                      <li>• All server backups</li>
                    </ul>
                    <div className="p-3 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800 rounded-md">
                      <p className="text-xs text-red-800 dark:text-red-200 font-medium">
                        Warning: This will reset all configurations and return you to the initial
                        setup page. You will be asked to enter your master password to confirm.
                      </p>
                    </div>
                    <div>
                      <Label htmlFor="reset-confirm">
                        Type <code className="bg-muted px-1 rounded">reset server</code> to confirm:
                      </Label>
                      <Input
                        id="reset-confirm"
                        placeholder="reset server"
                        value={resetConfirmText}
                        onChange={(e) => setResetConfirmText(e.target.value)}
                        className="mt-2"
                      />
                    </div>
                  </div>
                  <DialogFooter>
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => {
                        setIsResetDialogOpen(false);
                        setResetConfirmText('');
                      }}
                    >
                      Cancel
                    </Button>
                    <Button
                      variant="destructive"
                      type="submit"
                      disabled={resetConfirmText.toLowerCase() !== 'reset server'}
                    >
                      <Trash2 className="mr-2 h-4 w-4" />
                      Reset Server
                    </Button>
                  </DialogFooter>
                </form>
              </DialogContent>
            </Dialog>
          </div>
        </CardContent>
      </Card>

      {/* Auth Login Dialog */}
      <AuthLoginDialog
        open={showAuthLoginDialog}
        onOpenChange={setShowAuthLoginDialog}
        onShowRegistrationDialog={() => {
          setShowAuthLoginDialog(false);
          setShowRegistrationDialog(true);
        }}
        onLoginSuccess={handleLoginSuccess}
      />

      {/* Restore Dialog */}
      <RestoreDialog
        open={showRestoreDialog}
        onOpenChange={setShowRestoreDialog}
        onRestoreSuccess={() => {
          // Refresh server info after successful restore
          fetchServerInfo();
        }}
      />

      {/* Plan Selection Dialog */}
      <PlanSelectionDialog
        open={showPlanDialog}
        onOpenChange={setShowPlanDialog}
        currentPlan={currentServerPlan}
      />

      {/* Registration Dialog */}
      <AuthRegistrationDialog
        open={showRegistrationDialog}
        onOpenChange={setShowRegistrationDialog}
        onShowLoginDialog={() => {
          setShowRegistrationDialog(false);
          setShowAuthLoginDialog(true);
        }}
        onRegistrationSuccess={() => {
          // Handle registration success
        }}
      />

      {/* Backup to Local Files Dialog */}
      <Dialog
        open={showBackupDialog}
        onOpenChange={(open) => {
          setShowBackupDialog(open);
          if (!open) {
            setBackupMasterPassword('');
            setPasswordError('');
          }
        }}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Download className="h-5 w-5 text-green-600 dark:text-green-400" />
              Backup to Local Files
            </DialogTitle>
            <DialogDescription>
              Download a full backup of your server configuration and credentials.
            </DialogDescription>
          </DialogHeader>

          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (!backupMasterPassword.trim()) {
                toast.error('Please enter your master password');
                return;
              }
              handleBackupToLocal(backupMasterPassword);
            }}
          >
            <div className="space-y-4">
              {/* Full Backup Info */}
              <div className="p-4 border rounded-lg bg-amber-50 dark:bg-amber-950/20 border-amber-200 dark:border-amber-800">
                <div className="font-medium text-sm flex items-center gap-2 mb-2">
                  <Shield className="h-3.5 w-3.5 text-amber-600 dark:text-amber-400" />
                  Full Backup (Sensitive Data)
                </div>
                <div className="text-xs text-muted-foreground">
                  Includes everything: server settings, tool configurations, credentials, and access
                  tokens.
                  <span className="text-amber-700 dark:text-amber-400 font-medium">
                    {' '}
                    Master Password is required
                  </span>{' '}
                  for security.
                </div>
              </div>

              {/* Master Password Field */}
              <div className="space-y-2">
                <Label htmlFor="master-password" className="flex items-center gap-2">
                  <Key className="h-4 w-4" />
                  Master Password *
                </Label>
                <Input
                  id="master-password"
                  type="password"
                  placeholder="Enter master password"
                  value={backupMasterPassword}
                  onChange={(e) => {
                    setBackupMasterPassword(e.target.value);
                    // Clear error when user starts typing
                    if (passwordError) {
                      setPasswordError('');
                    }
                  }}
                  className={passwordError ? 'border-red-500' : ''}
                />
                <p className="text-xs text-muted-foreground">Used to encrypt the backup file</p>

                {/* Password Error Message */}
                {passwordError && (
                  <div className="p-3 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800 rounded-lg">
                    <p className="text-xs text-red-800 dark:text-red-200">{passwordError}</p>
                  </div>
                )}
              </div>

              {/* Tip Box */}
              <div className="p-3 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-lg">
                <p className="text-xs text-blue-800 dark:text-blue-200">
                  <span className="font-medium">Tip:</span> Full backups contain sensitive data and
                  should be stored securely. Never share backup files containing credentials.
                </p>
              </div>
            </div>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setShowBackupDialog(false);
                  setBackupMasterPassword('');
                  setPasswordError('');
                }}
                disabled={isBackingUp}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={isBackingUp || !backupMasterPassword.trim()}
                className="bg-green-600 hover:bg-green-700"
              >
                {isBackingUp ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Creating backup…
                  </>
                ) : (
                  <>
                    <Download className="mr-2 h-4 w-4" />
                    Create Backup
                  </>
                )}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Stop Server Confirmation Dialog */}
      <Dialog open={isStopDialogOpen} onOpenChange={setIsStopDialogOpen}>
        <DialogContent>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              executeServerToggle();
            }}
          >
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2 text-orange-600 dark:text-orange-400">
                <AlertTriangle className="h-5 w-5" />
                Stop Server
              </DialogTitle>
              <DialogDescription>
                Are you sure you want to stop the server? This will immediately disconnect all
                clients and stop all MCP tool connections.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="p-3 bg-orange-50 dark:bg-orange-950/20 border border-orange-200 dark:border-orange-800 rounded-md">
                <p className="text-sm text-orange-800 dark:text-orange-200">
                  <strong>Warning:</strong> All active AI assistant connections will be terminated.
                  Users will need to reconnect once the server is restarted.
                </p>
              </div>
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                type="button"
                onClick={() => setIsStopDialogOpen(false)}
                disabled={isTogglingServer}
              >
                Cancel
              </Button>
              <Button variant="destructive" type="submit" disabled={isTogglingServer}>
                {isTogglingServer ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Stopping...
                  </>
                ) : (
                  <>
                    <PowerOff className="mr-2 h-4 w-4" />
                    Stop Server
                  </>
                )}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
