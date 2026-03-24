'use client';

import { Eye, EyeOff, Settings } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { useState, useEffect } from 'react';

import { Button } from '@/components/ui/button';
import { Dialog, DialogContent } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

import { RestoreDialog } from '@/components/restore-dialog';
import { api } from '@/lib/api-client';
import { ManualConnectForm } from '@/components/manual-connect-form';
import { InitializeServerForm } from '@/components/initialize-server-form';
import { BackupTokenForm } from '@/components/backup-token-form';
import { LoginForm } from '@/components/login-form';

export default function WelcomePage() {
  const router = useRouter();
  const [isCheckingAuth, setIsCheckingAuth] = useState(true);

  // View mode for auth flows: null | 'manual-connect' | 'initialize' | 'backup-token' | 'login' | 'reset-success'
  const [viewMode, setViewMode] = useState<
    'manual-connect' | 'initialize' | 'backup-token' | 'login' | 'reset-success' | null
  >(null);
  // previousViewMode tracks where we came from
  const [previousViewMode, setPreviousViewMode] = useState<
    'manual-connect' | 'login' | 'initialize' | null
  >(null);

  // Token login states
  const [token, setToken] = useState('');

  // Restore states
  const [showRestoreDialog, setShowRestoreDialog] = useState(false);

  // First-Time Setup states

  const [ownerToken, setOwnerToken] = useState('');

  // Server info states
  const [proxyServerInfo, setProxyServerInfo] = useState<any>(null);
  const [isLoadingServerInfo, setIsLoadingServerInfo] = useState(false);

  // Manual server connection states
  const [showManualConnectionDialog, setShowManualConnectionDialog] = useState(false);
  const [manualHost, setManualHost] = useState('localhost');
  const [manualPort, setManualPort] = useState('3002');
  const [manualToken, setManualToken] = useState('');
  const [showManualToken, setShowManualToken] = useState(false);
  const [manualConnectionError, setManualConnectionError] = useState('');
  const [previousDialog, setPreviousDialog] = useState<'login' | 'init' | null>(null);
  const [isValidatingConnection, setIsValidatingConnection] = useState(false);

  // KIMBAP Core detection states
  const [kimbapCoreStatus, setKimbapCoreStatus] = useState<{
    isAvailable: number; // 1-Available, 2-Not started, 3-Started but not KIMBAP Core, 4-Config saved
    host?: string;
    port?: number;
  } | null>(null);

  // Detect KIMBAP Core service
  const detectKimbapCore = async (): Promise<{
    isAvailable: number;
    host?: string;
    port?: number;
  }> => {
    try {
      const response = await api.servers.detectKimbapCore();

      if (response.data?.data) {
        const { isAvailable, kimbapCoreHost, kimbapCorePort } = response.data.data;
        const status = {
          isAvailable,
          host: kimbapCoreHost,
          port: kimbapCorePort,
        };
        setKimbapCoreStatus(status);
        return status;
      }

      return { isAvailable: 2 }; // Default: port not started
    } catch (error: any) {
      // Detection failed, return default status
      return { isAvailable: 2 };
    } finally {
    }
  };

  // Fetch proxy server info
  const fetchProxyServerInfo = async (): Promise<boolean> => {
    setIsLoadingServerInfo(true);
    try {
      const response = await api.servers.getInfo();

      if (response.data?.data) {
        setProxyServerInfo(response.data.data);

        return true;
      }

      return false;
    } catch (error: any) {
      // 404 means no server exists yet
      if (error.response?.status === 404) {
        return false;
      }
      // Non-404 error, assume no server exists

      return false;
    } finally {
      setIsLoadingServerInfo(false);
    }
  };

  // Check authentication status on component mount
  useEffect(() => {
    const checkAuth = async () => {
      try {
        // Check if userid + token exists in localStorage - if yes, go to dashboard
        const userid = localStorage.getItem('userid');
        const authToken = localStorage.getItem('auth_token');
        if (userid && authToken) {
          router.push('/dashboard');
          return;
        }
        if (userid && !authToken) {
          localStorage.removeItem('userid');
        }

        // Check if reset success dialog should be shown first
        const showResetSuccess = sessionStorage.getItem('showResetSuccess');
        if (showResetSuccess) {
          setViewMode('reset-success');
          setIsCheckingAuth(false);
          // Also fetch server info
          fetchProxyServerInfo();
          // Don't remove sessionStorage here - will be removed when user clicks continue
          return;
        }

        // Step 1: Detect KIMBAP Core service (protocol 10021)
        const kimbapCoreDetection = await detectKimbapCore();

        // If KIMBAP Core is not available (only 1 or 4 are considered available), show manual connection dialog
        // isAvailable: 1 = Available, 2 = Port not started, 3 = Port started but not KIMBAP Core, 4 = Config saved
        if (kimbapCoreDetection.isAvailable !== 1 && kimbapCoreDetection.isAvailable !== 4) {
          setViewMode('manual-connect');
          setIsCheckingAuth(false);
          return;
        }

        // Step 2: KIMBAP Core is available, check if local server exists (initialized)
        const serverExists = await fetchProxyServerInfo();

        // If server not initialized, show initialize page (which includes master password setup)
        if (!serverExists) {
          setViewMode('initialize');
          setIsCheckingAuth(false);
          return;
        }

        // Server is initialized, show login

        // Step 4: Check if user has previous successful connection
        const { getAuthToken } = await import('@/lib/api-client');
        const token = getAuthToken();
        const selectedServer = localStorage.getItem('selectedServer');

        // Case 0: Previously connected successfully
        if (token && selectedServer) {
          // Try to use existing connection
          try {
            // Test if connection still works by fetching server info
            await api.servers.getInfo();
            router.push('/dashboard');
            return;
          } catch (error) {
            // Still go to dashboard, show reconnection UI there
            router.push('/dashboard');
            return;
          }
        }

        // Case 1: No previous connection, but local server exists (initialized)
        // Server exists means it's already initialized, always show login regardless of master password
        if (serverExists) {
          setViewMode('login');
          setIsCheckingAuth(false);
          return;
        }
      } catch (error) {
        // Auth check failed, continue to login view
        setIsCheckingAuth(false);
      }
    };

    checkAuth();
  }, [router]);

  const handleManualConnection = async () => {
    setManualConnectionError('');

    if (!manualHost.trim()) {
      setManualConnectionError('Host is required');
      return;
    }

    const port = parseInt(manualPort.trim());
    if (!port || port <= 0 || port > 65535) {
      setManualConnectionError('Port must be between 1 and 65535');
      return;
    }

    setIsValidatingConnection(true);

    try {
      // Step 1: Call protocol 10022 to validate and save KIMBAP Core config
      const configResponse = await api.servers.configureKimbapCore({
        host: manualHost.trim(),
        port,
      });

      if (configResponse.data?.data) {
        const { isValid, message } = configResponse.data.data;

        if (isValid === 1) {
          // Success - configuration saved

          // Step 2: Call protocol 10002 to check if server is initialized
          try {
            const serverInfoResponse = await api.servers.getInfo();

            if (serverInfoResponse.data?.data) {
              // Server is initialized
              setShowManualConnectionDialog(false);
              setManualConnectionError('');

              // If user provided access token, use it to login
              if (manualToken.trim()) {
                setToken(manualToken);

                setViewMode('login');
              } else {
                // No token provided, show login
                setViewMode('login');
              }
            } else {
              // No server data but 10002 succeeded - should not happen
              setViewMode('initialize');
            }
          } catch (serverInfoError: any) {
            // 10002 failed - server not initialized yet
            setViewMode('initialize');
          }
        } else if (isValid === 2) {
          // Host/port invalid or cannot connect
          setManualConnectionError(message || 'Cannot connect to the specified host and port');
        } else if (isValid === 3) {
          // Port responding but not KIMBAP Core
          setManualConnectionError(
            message || 'The specified port is not running Kimbap Core service',
          );
        }
      }
    } catch (error: any) {
      // Connection error handled below with setManualConnectionError()
      setManualConnectionError(
        error.response?.data?.error ||
          error.message ||
          'Could not validate Kimbap Core configuration',
      );
    } finally {
      setIsValidatingConnection(false);
    }
  };

  // Show loading while checking authentication
  if (isCheckingAuth) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-muted">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-muted-foreground/30 border-t-foreground rounded-full animate-spin mx-auto mb-4" />
          <p className="text-muted-foreground">Loading...</p>
        </div>
      </div>
    );
  }

  // Show AuthLayout with appropriate form based on viewMode
  if (viewMode) {
    return (
      <div className="flex min-h-screen p-[24px] pb-0 bg-[#F7F7F3] dark:bg-background flex-col">
        <div className="flex flex-1">
          {/* Left Side - Branding */}
          <div className="hidden lg:flex lg:w-1/2 p-12 flex-col justify-center items-center">
            <div className="max-w-[780px]">
              <h1 className="text-[52px] font-bold text-[#F56711] leading-[60px] mb-[4px]">
                Kimbap Console
              </h1>
              <h2 className="text-[40px] font-bold text-[#26251E] dark:text-foreground leading-[48px] mb-[24px]">
                Operations Console
              </h2>
              <p className="text-muted-foreground leading-[24px] text-[16px]">
                Observe health, explore audit and logs, review approvals, monitor integration
                status, and triage operational incidents — all in one place.
              </p>
            </div>
          </div>

          {/* Right Side - Content */}
          <div className="flex-1 flex bg-white dark:bg-slate-900 rounded-[12px]">
            <div className="w-full flex flex-col">
              <div className="p-[14px]">
                <img src="/new_logo.svg" alt="Kimbap Logo" className="block dark:hidden" />
                <img src="/darklogo.svg" alt="Kimbap Logo" className="hidden dark:block" />
              </div>
              <div className="flex-1 flex flex-col justify-center items-center">
                {viewMode === 'manual-connect' && (
                  <ManualConnectForm
                    onSuccess={async () => {
                      // Check if server is initialized after successful connection
                      try {
                        const serverInfoResponse = await api.servers.getInfo();

                        if (serverInfoResponse.data?.data) {
                          // Server is initialized, show login

                          // Check if there's a saved access token
                          const savedToken = localStorage.getItem('manualAccessToken');
                          if (savedToken) {
                            setToken(savedToken);
                            localStorage.removeItem('manualAccessToken');
                          }
                          setViewMode('login');
                        } else {
                          // No server data, show initialize
                          setViewMode('initialize');
                        }
                      } catch (error) {
                        // Server not initialized yet, show initialize
                        setViewMode('initialize');
                      }
                    }}
                    onBack={
                      previousViewMode
                        ? () => {
                            setViewMode(previousViewMode);
                            setPreviousViewMode(null);
                          }
                        : undefined
                    }
                  />
                )}

                {viewMode === 'initialize' && (
                  <InitializeServerForm
                    onSuccess={(ownerToken) => {
                      setOwnerToken(ownerToken);
                      setViewMode('backup-token');
                    }}
                    onManualConnect={() => {
                      setPreviousViewMode('initialize');
                      setViewMode('manual-connect');
                    }}
                    onRestore={() => {
                      setViewMode(null);
                      setShowRestoreDialog(true);
                    }}
                    kimbapCoreStatus={kimbapCoreStatus}
                  />
                )}

                {viewMode === 'backup-token' && (
                  <BackupTokenForm
                    ownerToken={ownerToken}
                    onComplete={() => {
                      setViewMode(null);

                      router.push('/dashboard');
                    }}
                  />
                )}

                {viewMode === 'login' && (
                  <LoginForm
                    onSuccess={() => {
                      setViewMode(null);

                      router.push('/dashboard');
                    }}
                    onManualConnect={() => {
                      setPreviousViewMode('login');
                      setViewMode('manual-connect');
                    }}
                    defaultToken={token}
                  />
                )}

                {viewMode === 'reset-success' && (
                  <div className="space-y-[12px] max-w-[460px] py-[32px] px-[24px]">
                    <div>
                      <h2 className="text-[24px] font-bold mb-[4px]">Server Reset Successful</h2>
                      <p className="text-muted-foreground text-[14px]">
                        Your server has been reset to its initial state.
                      </p>
                    </div>

                    {/* Status Information */}
                    <div className="space-y-[8px] p-4 bg-emerald-50 dark:bg-emerald-950/20 rounded-lg border border-emerald-200 dark:border-emerald-800">
                      <div className="flex justify-between items-center">
                        <span className="text-[14px] text-foreground">Reset Status</span>
                        <span className="text-[14px] text-emerald-500 dark:text-emerald-400">
                          ✓ Completed
                        </span>
                      </div>
                      <div className="flex justify-between items-center">
                        <span className="text-[14px] text-foreground">Server Status</span>
                        <span
                          className={`text-[14px] ${
                            isLoadingServerInfo
                              ? 'text-muted-foreground'
                              : proxyServerInfo?.status === 1
                                ? 'text-emerald-500 dark:text-emerald-400'
                                : 'text-amber-500 dark:text-amber-400'
                          }`}
                        >
                          {isLoadingServerInfo
                            ? 'Loading...'
                            : proxyServerInfo?.status === 1
                              ? '✓ Running'
                              : '⚠ Stopped'}
                        </span>
                      </div>
                      <div className="flex justify-between items-center">
                        <span className="text-[14px] text-foreground">Configuration</span>
                        <span className="text-[14px] text-emerald-500 dark:text-emerald-400">
                          ✓ Reset to Default
                        </span>
                      </div>
                      <div className="flex justify-between items-center">
                        <span className="text-[14px] text-foreground">Ready to Use</span>
                        <span
                          className={`text-[14px] ${
                            proxyServerInfo?.status === 1
                              ? 'text-emerald-500 dark:text-emerald-400'
                              : 'text-amber-500 dark:text-amber-400'
                          }`}
                        >
                          {proxyServerInfo?.status === 1 ? '✓ Yes' : '⚠ Need to Start Server'}
                        </span>
                      </div>
                    </div>

                    <p className="text-[14px] text-muted-foreground">
                      The server has been completely reset and is ready for initial setup. Click
                      continue to proceed with first-time server initialization.
                    </p>

                    {/* Continue Button */}
                    <Button
                      onClick={() => {
                        sessionStorage.removeItem('showResetSuccess');
                        setViewMode('initialize');
                      }}
                      className="w-full h-12 text-base bg-foreground hover:bg-foreground/90 text-background rounded-[8px]"
                      size="lg"
                    >
                      <Settings className="w-4 h-4 mr-2" />
                      Continue to Setup
                    </Button>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Footer */}
        <footer className="w-full py-4 border-t border-slate-200 dark:border-slate-800">
          <div className="text-center text-xs text-slate-500 dark:text-slate-400">
            <span>© 2026 </span>
            <a
              href="https://kimbap.io"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:underline text-slate-600 dark:text-slate-300"
            >
              Dunia Labs, Inc.
            </a>
            <span>
              {' '}
              Secure action runtime for AI agents: vault, skills, policies, and approvals.
            </span>
          </div>
        </footer>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col bg-gradient-to-br from-slate-50 via-white to-blue-50/30 dark:from-slate-950 dark:via-slate-900 dark:to-blue-950/30">
      {/* Background Pattern */}
      <div
        className="absolute inset-0 opacity-50"
        style={{
          backgroundImage: `url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23e2e8f0' fill-opacity='0.1'%3E%3Ccircle cx='30' cy='30' r='1'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E")`,
          backgroundSize: '60px 60px',
        }}
      />
      <div
        className="absolute inset-0 opacity-50 dark:block hidden"
        style={{
          backgroundImage: `url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23475569' fill-opacity='0.05'%3E%3Ccircle cx='30' cy='30' r='1'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E")`,
          backgroundSize: '60px 60px',
        }}
      />

      {/* Restore Dialog */}
      <RestoreDialog
        open={showRestoreDialog}
        onOpenChange={setShowRestoreDialog}
        onRestoreSuccess={() => {
          // After successful restore, redirect to dashboard
          router.push('/dashboard');
        }}
      />

      {/* Manual Server Connection Dialog */}
      <Dialog
        open={showManualConnectionDialog}
        onOpenChange={(open) => {
          if (!open) {
            // Close manual connection dialog and return to previous dialog
            setShowManualConnectionDialog(false);
            setManualConnectionError('');

            // Reopen the previous dialog
            if (previousDialog === 'login') {
            } else if (previousDialog === 'init') {
              setViewMode('initialize');
            }
            setPreviousDialog(null);
          }
        }}
      >
        <DialogContent className="max-w-[500px] p-8">
          {/* Header */}
          <div className="mb-6">
            <h2 className="text-3xl font-bold text-gray-900 dark:text-white mb-3">
              Configure Kimbap Core
            </h2>
            <p className="text-gray-500 dark:text-gray-400 text-base">
              Enter the host and port where your Kimbap Core service is running.
            </p>
          </div>

          {/* Form Fields */}
          <div className="space-y-5">
            {/* Access Token (Optional) */}
            <div>
              <Label
                htmlFor="manual-token"
                className="text-base font-semibold text-gray-900 dark:text-white mb-3 block"
              >
                Access Token <span className="text-muted-foreground font-normal">(Optional)</span>
              </Label>
              <div className="relative">
                <Input
                  id="manual-token"
                  type={showManualToken ? 'text' : 'password'}
                  placeholder="Enter access token (optional)"
                  value={manualToken}
                  onChange={(e) => {
                    setManualToken(e.target.value);
                    setManualConnectionError('');
                  }}
                  disabled={isValidatingConnection}
                  className="h-14 px-4 text-base rounded-lg border-gray-300 dark:border-gray-600"
                />
                <button
                  type="button"
                  onClick={() => setShowManualToken(!showManualToken)}
                  className="absolute right-4 top-1/2 -translate-y-1/2 text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200"
                  tabIndex={-1}
                  aria-label="Toggle token visibility"
                >
                  {showManualToken ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
                </button>
              </div>
            </div>

            {/* Host and Port */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label
                  htmlFor="manual-host"
                  className="text-base font-semibold text-gray-900 dark:text-white mb-3 block"
                >
                  Host
                </Label>
                <Input
                  id="manual-host"
                  type="text"
                  placeholder="localhost"
                  value={manualHost}
                  onChange={(e) => {
                    setManualHost(e.target.value);
                    setManualConnectionError('');
                  }}
                  disabled={isValidatingConnection}
                  className="h-14 px-4 text-base rounded-lg border-gray-300 dark:border-gray-600"
                />
              </div>
              <div>
                <Label
                  htmlFor="manual-port"
                  className="text-base font-semibold text-gray-900 dark:text-white mb-3 block"
                >
                  Port
                </Label>
                <Input
                  id="manual-port"
                  type="text"
                  placeholder="e.g., 3002"
                  value={manualPort}
                  onChange={(e) => {
                    setManualPort(e.target.value);
                    setManualConnectionError('');
                  }}
                  disabled={isValidatingConnection}
                  className="h-14 px-4 text-base rounded-lg border-gray-300 dark:border-gray-600"
                />
              </div>
            </div>

            {/* Error Message */}
            {manualConnectionError && (
              <div className="p-4 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800 rounded-lg">
                <p className="text-base text-red-700 dark:text-red-300">
                  <strong>Error:</strong> {manualConnectionError}
                </p>
              </div>
            )}
          </div>

          {/* Action Buttons */}
          <div className="grid grid-cols-2 gap-4 mt-6">
            <Button
              variant="outline"
              onClick={() => {
                setShowManualConnectionDialog(false);
                setManualConnectionError('');

                // Reopen the previous dialog
                if (previousDialog === 'login') {
                } else if (previousDialog === 'init') {
                  setViewMode('initialize');
                }
                setPreviousDialog(null);
              }}
              disabled={isValidatingConnection}
              className="h-14 text-base font-medium rounded-lg"
            >
              Cancel
            </Button>
            <Button
              onClick={handleManualConnection}
              disabled={!manualHost.trim() || !manualPort.trim() || isValidatingConnection}
              className="h-14 text-base font-medium bg-blue-400 hover:bg-blue-500 text-white rounded-lg disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isValidatingConnection ? (
                <>
                  <div className="w-5 h-5 mr-2 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  Validating...
                </>
              ) : (
                <>
                  <Settings className="w-5 h-5 mr-2" />
                  Connect
                </>
              )}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
