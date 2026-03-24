'use client';

import {
  Calendar,
  CheckCircle,
  Clock,
  Crown,
  Fingerprint,
  Key,
  Shield,
  Upload,
  ArrowUp,
  Copy,
  Loader2,
} from 'lucide-react';
import { useState, useEffect, useRef, useCallback } from 'react';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
import { Button, buttonVariants } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  ScrollableDialogContent,
} from '@/components/ui/dialog';
import { Progress } from '@/components/ui/progress';
import { PLAN_DISPLAY, getPricingUrl, resolvePlanId } from '@/lib/plan-config';
import type { License, LicenseLimits, LicensePageState } from '@/lib/types/license';
import { cn } from '@/lib/utils';

const FREE_PLAN_MAX_TOOLS = 30;
const FREE_PLAN_MAX_TOKENS = 30;

export default function LicensePage() {
  const [licenseData, setLicenseData] = useState<LicensePageState>({
    licenseHistory: [],
    isLoading: true,
  });
  const [licenseLimits, setLicenseLimits] = useState<LicenseLimits | null>(null);
  const [newLicenseKey, setNewLicenseKey] = useState('');
  const [isActivating, setIsActivating] = useState(false);
  const [proxyKey, setProxyKey] = useState<string>('');
  const [fingerprint, setFingerprint] = useState<string>('');
  const [showActivateInput, setShowActivateInput] = useState(false);
  const [showHistoryDialog, setShowHistoryDialog] = useState(false);
  const [isLoadingFingerprint, setIsLoadingFingerprint] = useState(true);
  const [inputMode, setInputMode] = useState<'key' | 'file'>('key');
  const [isDragging, setIsDragging] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const fetchServerInfo = useCallback(async () => {
    try {
      setIsLoadingFingerprint(true);
      const { api } = await import('@/lib/api-client');
      const response = await api.servers.getInfo();

      if (response.data?.data) {
        if (response.data.data.proxyKey) {
          setProxyKey(response.data.data.proxyKey);
        }
        if (response.data.data.fingerprint) {
          setFingerprint(response.data.data.fingerprint);
        }
      }
    } catch (error) {
      console.warn('Failed to fetch server info:', error);
    } finally {
      setIsLoadingFingerprint(false);
    }
  }, []);

  const loadLicenseData = useCallback(async () => {
    try {
      setLicenseData((prev) => ({ ...prev, isLoading: true }));
      const { api } = await import('@/lib/api-client');

      const response = await api.license.getHistory();
      const licenseList = response.data?.data?.licenseList || [];
      let limitsData: LicenseLimits | null = null;
      try {
        const limitsResponse = await api.license.getLimits();
        limitsData = limitsResponse.data?.data || null;
      } catch (limitsError) {
        console.warn('Failed to load license limits:', limitsError);
      }

      // Find active license (status = 100)
      const activeLicense = licenseList.find((license: License) => license.status === 100);
      setLicenseLimits(limitsData);

      setLicenseData({
        activeLicense,
        licenseHistory: licenseList,
        isLoading: false,
        error: undefined,
      });
    } catch (error) {
      setLicenseData({
        licenseHistory: [],
        isLoading: false,
        error: 'Could not load license data. Try again.',
      });
    }
  }, [fingerprint]);

  // Load license data and server info on component mount
  useEffect(() => {
    loadLicenseData();
    fetchServerInfo();
  }, [loadLicenseData, fetchServerInfo]);

  const activateWithLicenseStr = async (licenseStr: string) => {
    if (!licenseStr.trim()) {
      toast.error('Please enter a license key');
      return;
    }

    try {
      setIsActivating(true);
      const { api } = await import('@/lib/api-client');

      // Get proxyKey - use cached value or fetch if not available
      let currentProxyKey = proxyKey;
      if (!currentProxyKey) {
        const serverResponse = await api.servers.getInfo();
        if (serverResponse.data?.data?.proxyKey) {
          currentProxyKey = serverResponse.data.data.proxyKey;
          setProxyKey(currentProxyKey);
        } else {
          toast.error('Could not get server information. Try again.');
          return;
        }
      }

      await api.license.importLicense({
        licenseStr: licenseStr.trim(),
        proxyKey: currentProxyKey,
      });

      toast.success('License activated.');
      setNewLicenseKey('');
      setShowActivateInput(false);
      setInputMode('key');

      // Reload license data
      await loadLicenseData();
    } catch (error: unknown) {
      toast.error('Could not activate license. Check the license key and try again.');
    } finally {
      setIsActivating(false);
    }
  };

  const handleActivateLicense = async () => {
    await activateWithLicenseStr(newLicenseKey);
  };

  const handleLicenseFile = async (file: File) => {
    try {
      const text = (await file.text()).trim();
      if (!text) {
        toast.error('License file is empty');
        return;
      }
      // Activate directly without requiring an extra click
      await activateWithLicenseStr(text);
    } catch (error) {
      // File read failed, toast shown below
      toast.error('Could not read license file');
    } finally {
      // reset input value so selecting same file again works
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const getStatusBadge = (status: number) => {
    switch (status) {
      case 100:
        return (
          <Badge className="bg-green-100 text-green-800 border-green-200 dark:bg-green-900 dark:text-green-300 dark:border-green-800">
            Active
          </Badge>
        );
      case 1:
        return (
          <Badge className="bg-gray-100 text-gray-800 border-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700">
            Inactive
          </Badge>
        );
      case 2:
        return (
          <Badge className="bg-red-100 text-red-800 border-red-200 dark:bg-red-900 dark:text-red-300 dark:border-red-800">
            Expired
          </Badge>
        );
      default:
        return <Badge>Unknown</Badge>;
    }
  };

  const formatDate = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleDateString();
  };

  const currentPlan = licenseData.activeLicense
    ? resolvePlanId(licenseData.activeLicense.plan)
    : 'free';
  const plan = PLAN_DISPLAY[currentPlan] || PLAN_DISPLAY.free;

  // Get upgrade button text based on license status
  const getUpgradeButtonText = () => {
    if (!licenseData.activeLicense) {
      return 'Get License';
    }
    const expiryDate =
      licenseData.activeLicense.expiresAt > 0
        ? new Date(licenseData.activeLicense.expiresAt * 1000)
        : null;
    if (!expiryDate) {
      return 'Upgrade License';
    }
    const today = new Date();
    const daysLeft = Math.ceil((expiryDate.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));
    if (daysLeft < 30) {
      return 'Renew License';
    }
    return 'Upgrade License';
  };

  if (licenseData.isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div aria-live="polite" className="text-center">
          <div
            className="w-8 h-8 border-2 border-muted-foreground/30 border-t-foreground rounded-full animate-spin mx-auto mb-4"
            aria-hidden="true"
          />
          <h3 className="text-lg font-semibold mb-2">Loading License Data</h3>
          <p className="text-muted-foreground">Fetching license information...</p>
        </div>
      </div>
    );
  }

  if (licenseData.error) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <h3 className="text-lg font-semibold mb-2">Could not load data</h3>
          <p className="text-muted-foreground mb-4">{licenseData.error}</p>
          <Button onClick={() => loadLicenseData()}>Retry</Button>
        </div>
      </div>
    );
  }

  const activeLicense = licenseData.activeLicense;
  const expiryDate =
    activeLicense && activeLicense.expiresAt > 0 ? new Date(activeLicense.expiresAt * 1000) : null;
  const daysLeft = expiryDate
    ? Math.ceil((expiryDate.getTime() - new Date().getTime()) / (1000 * 60 * 60 * 24))
    : null;
  const toolUsageCount =
    licenseLimits?.currentToolCount ?? activeLicense?.currentToolCreations ?? 0;
  const tokenUsageCount =
    licenseLimits?.currentAccessTokenCount ?? activeLicense?.currentAccessTokens ?? 0;
  const toolUsageLimit =
    licenseLimits?.maxToolCreations ??
    (activeLicense ? activeLicense.maxToolCreations : FREE_PLAN_MAX_TOOLS);
  const tokenUsageLimit =
    licenseLimits?.maxAccessTokens ??
    (activeLicense ? activeLicense.maxAccessTokens : FREE_PLAN_MAX_TOKENS);
  const hasToolUsageLimit = toolUsageLimit > 0;
  const hasTokenUsageLimit = tokenUsageLimit > 0;

  return (
    <div className="space-y-6">
      <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-4 mb-2">
        <p className="text-sm text-amber-800 dark:text-amber-300">
          <strong>Deprecated:</strong> This page will be removed in a future release. Navigate to
          the{' '}
          <a href="/dashboard" className="underline font-medium">
            Dashboard
          </a>{' '}
          for the updated experience.
        </p>
      </div>
      <div className="flex flex-col sm:flex-row items-start sm:justify-between gap-4">
        <div className="space-y-0">
          <h1 className="text-[30px] font-bold">License</h1>
          <p className="text-base text-muted-foreground">
            Manage your license and view usage statistics.
          </p>
        </div>
        <Button variant="outline" onClick={() => setShowHistoryDialog(true)}>
          <Clock className="h-4 w-4 mr-2" />
          View History
        </Button>
      </div>

      {/* Current License Card */}
      <Card>
        <CardHeader>
          <div className="flex flex-col sm:flex-row items-start sm:justify-between gap-4">
            <div className="flex items-start gap-3">
              <div className="h-10 w-10 rounded-full bg-green-100 dark:bg-green-900 flex items-center justify-center">
                <Shield className="h-5 w-5 text-green-600 dark:text-green-400" />
              </div>
              <div>
                <CardTitle>Current License</CardTitle>
                <CardDescription>Your license status and information</CardDescription>
              </div>
            </div>
            <div className="flex gap-2">
              <a
                href={getPricingUrl(fingerprint)}
                target="_blank"
                rel="noopener noreferrer"
                className={cn(buttonVariants({ variant: 'outline', size: 'sm' }))}
              >
                <ArrowUp className="h-4 w-4 mr-2" />
                {getUpgradeButtonText()}
              </a>
              <Button size="sm" onClick={() => setShowActivateInput(!showActivateInput)}>
                <Key className="h-4 w-4 mr-2" />
                Activate License
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* License Details */}
          <div className="grid md:grid-cols-4 gap-4">
            <div>
              <div className="flex items-center gap-2 mb-2">
                <Badge className={plan.color}>{plan.name} Plan</Badge>
              </div>
              <p className="text-2xl font-bold">{activeLicense ? 'Licensed' : 'Community Plan'}</p>
              <p className="text-sm text-muted-foreground">
                {activeLicense ? `${plan.name} License` : 'Up to 30 tools and 30 access tokens'}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground mb-1">Status</p>
              <div className="flex items-center gap-2">
                {activeLicense ? (
                  <>
                    <CheckCircle className="h-4 w-4 text-green-500 dark:text-green-400" />
                    <span className="font-semibold text-green-600 dark:text-green-400">Active</span>
                  </>
                ) : (
                  <>
                    <CheckCircle className="h-4 w-4 text-green-500 dark:text-green-400" />
                    <span className="font-semibold text-green-600 dark:text-green-400">
                      Community Tier
                    </span>
                  </>
                )}
              </div>
              <p className="text-sm text-muted-foreground mt-1">
                {activeLicense ? `Valid ${plan.name} license` : 'No license needed'}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground mb-1">Expires</p>
              <p className="font-semibold flex items-center gap-1">
                <Calendar className="h-4 w-4" />
                {expiryDate ? formatDate(activeLicense!.expiresAt) : 'N/A'}
              </p>
              <p className="text-sm text-muted-foreground mt-1">
                {!activeLicense
                  ? ''
                  : !expiryDate
                    ? 'Lifetime license'
                    : daysLeft && daysLeft > 0
                      ? `${daysLeft} days remaining`
                      : 'Expired'}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground mb-1">Days Remaining</p>
              <p className="font-semibold flex items-center gap-1">
                <Clock className="h-4 w-4" />
                {daysLeft && daysLeft > 0 ? `${daysLeft} days` : activeLicense ? 'Expired' : 'N/A'}
              </p>
            </div>
          </div>

          {/* Machine Fingerprint */}
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <Fingerprint className="h-4 w-4 text-muted-foreground" />
              <p className="text-sm font-medium">Machine Fingerprint</p>
            </div>
            <p className="text-xs text-muted-foreground">Required when purchasing a new license</p>
            {isLoadingFingerprint ? (
              <output
                aria-live="polite"
                className="flex items-center gap-2 p-3 bg-muted rounded-lg"
              >
                <Loader2
                  className="h-4 w-4 animate-spin text-muted-foreground"
                  aria-hidden="true"
                />
                <span className="text-sm text-muted-foreground">Loading fingerprint...</span>
              </output>
            ) : fingerprint ? (
              <div className="flex items-center gap-2 p-3 bg-muted rounded-lg">
                <code className="flex-1 text-sm font-mono break-all">{fingerprint}</code>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-8 w-8 p-0 shrink-0"
                  onClick={() => {
                    navigator.clipboard.writeText(fingerprint);
                    toast.success('Machine fingerprint copied to clipboard');
                  }}
                  title="Copy fingerprint"
                  aria-label="Copy machine fingerprint"
                >
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
            ) : (
              <div className="text-sm text-muted-foreground p-3 bg-muted rounded-lg">
                Unable to retrieve machine fingerprint
              </div>
            )}
          </div>

          {/* Activate License Input (shown when Activate is clicked) */}
          {showActivateInput && (
            <div className="space-y-4 pt-4 border-t">
              {/* Mode switch */}
              <div className="inline-flex items-center rounded-full bg-muted p-1">
                <button
                  type="button"
                  onClick={() => setInputMode('key')}
                  aria-pressed={inputMode === 'key'}
                  className={[
                    'h-10 px-4 rounded-full text-sm font-medium',
                    'inline-flex items-center gap-2 transition-colors',
                    inputMode === 'key'
                      ? 'bg-background shadow-sm text-foreground'
                      : 'text-muted-foreground hover:text-foreground',
                  ].join(' ')}
                >
                  <Key className="h-4 w-4" />
                  License Key
                </button>
                <button
                  type="button"
                  onClick={() => setInputMode('file')}
                  aria-pressed={inputMode === 'file'}
                  className={[
                    'h-10 px-4 rounded-full text-sm font-medium',
                    'inline-flex items-center gap-2 transition-colors',
                    inputMode === 'file'
                      ? 'bg-background shadow-sm text-foreground'
                      : 'text-muted-foreground hover:text-foreground',
                  ].join(' ')}
                >
                  <Upload className="h-4 w-4" />
                  Upload File
                </button>
              </div>

              {inputMode === 'key' ? (
                <div className="flex gap-2">
                  <Input
                    aria-label="License key"
                    placeholder="Enter your license key"
                    className="flex-1"
                    value={newLicenseKey}
                    onChange={(e) => setNewLicenseKey(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' && newLicenseKey.trim() && !isActivating) {
                        handleActivateLicense();
                      }
                    }}
                  />
                  <Button
                    onClick={handleActivateLicense}
                    disabled={isActivating || !newLicenseKey.trim()}
                  >
                    {isActivating ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" aria-hidden="true" />
                        Activating...
                      </>
                    ) : (
                      <>
                        <Key className="h-4 w-4 mr-2" />
                        Activate License
                      </>
                    )}
                  </Button>
                </div>
              ) : (
                <div className="space-y-2">
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".lic,.license,.key,.txt"
                    className="hidden"
                    onChange={async (e) => {
                      const file = e.target.files?.[0];
                      if (file) {
                        await handleLicenseFile(file);
                      }
                    }}
                  />
                  <button
                    type="button"
                    className={[
                      'border-2 border-dashed rounded-lg p-10',
                      'flex items-center justify-center text-center',
                      'cursor-pointer select-none',
                      isDragging ? 'border-primary bg-primary/5' : 'border-muted-foreground/25',
                    ].join(' ')}
                    onClick={() => fileInputRef.current?.click()}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        fileInputRef.current?.click();
                      }
                    }}
                    onDragEnter={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      setIsDragging(true);
                    }}
                    onDragOver={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      setIsDragging(true);
                    }}
                    onDragLeave={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      setIsDragging(false);
                    }}
                    onDrop={async (e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      setIsDragging(false);
                      const file = e.dataTransfer.files?.[0];
                      if (file) {
                        await handleLicenseFile(file);
                      }
                    }}
                  >
                    <div className="space-y-2">
                      <div className="mx-auto h-12 w-12 rounded-full bg-muted flex items-center justify-center">
                        <Upload className="h-5 w-5 text-muted-foreground" />
                      </div>
                      <div className="text-base font-medium">Drop your license file here</div>
                      <div className="text-sm text-muted-foreground">
                        or click to browse (.lic, .license, .key)
                      </div>
                    </div>
                  </button>
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Usage Statistics Card */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <div className="h-10 w-10 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center">
              <Upload className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            </div>
            <div>
              <CardTitle>Usage Statistics</CardTitle>
              <CardDescription>Usage against license limits</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="space-y-4">
            <div>
              <div className="flex items-center justify-between mb-2">
                <p className="font-medium">Tools Created</p>
                <p className="text-sm text-muted-foreground">
                  {toolUsageCount}
                  {hasToolUsageLimit ? ` / ${toolUsageLimit}` : ' / Unlimited'}
                </p>
              </div>
              {hasToolUsageLimit && (
                <>
                  <Progress
                    value={Math.min((toolUsageCount / toolUsageLimit) * 100, 100)}
                    className="h-2"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    {Math.max(toolUsageLimit - toolUsageCount, 0)} tools remaining
                  </p>
                </>
              )}
              {!hasToolUsageLimit && (
                <p className="text-xs text-muted-foreground">
                  {`No limits with ${plan.name} license`}
                </p>
              )}
            </div>

            <div>
              <div className="flex items-center justify-between mb-2">
                <p className="font-medium">Access Tokens</p>
                <p className="text-sm text-muted-foreground">
                  {tokenUsageCount}
                  {hasTokenUsageLimit ? ` / ${tokenUsageLimit}` : ' / Unlimited'}
                </p>
              </div>
              {hasTokenUsageLimit && (
                <>
                  <Progress
                    value={Math.min((tokenUsageCount / tokenUsageLimit) * 100, 100)}
                    className="h-2"
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    {Math.max(tokenUsageLimit - tokenUsageCount, 0)} tokens remaining
                  </p>
                </>
              )}
              {!hasTokenUsageLimit && (
                <p className="text-xs text-muted-foreground">
                  {`No limits with ${plan.name} license`}
                </p>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* License History Dialog */}
      <Dialog open={showHistoryDialog} onOpenChange={setShowHistoryDialog}>
        <ScrollableDialogContent className="max-w-4xl">
          <DialogHeader>
            <DialogTitle>License History</DialogTitle>
            <DialogDescription>Your license activation and renewal history</DialogDescription>
          </DialogHeader>
          <div>
            <div className="mt-4">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Date</TableHead>
                    <TableHead>License Type</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Expires</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {licenseData.licenseHistory.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={4} className="text-center py-8">
                        <div className="text-center">
                          <Shield className="h-12 w-12 text-muted-foreground/40 mx-auto mb-2" />
                          <p className="text-sm text-muted-foreground">
                            No license history found. Activate your first license to see history
                            here.
                          </p>
                        </div>
                      </TableCell>
                    </TableRow>
                  ) : (
                    licenseData.licenseHistory.map((license) => (
                      <TableRow
                        key={`${license.createdAt}-${license.plan}-${license.status}-${license.expiresAt}`}
                      >
                        <TableCell className="font-medium">
                          {formatDate(license.createdAt)}
                        </TableCell>
                        <TableCell>
                          {(() => {
                            const planKey = resolvePlanId(license.plan);
                            return PLAN_DISPLAY[planKey]?.name || license.plan || 'Unknown';
                          })()}
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            {license.status === 100 ? (
                              <CheckCircle className="h-4 w-4 text-green-500 dark:text-green-400" />
                            ) : license.status === 2 ? (
                              <Clock className="h-4 w-4 text-red-500 dark:text-red-400" />
                            ) : (
                              <Clock className="h-4 w-4 text-gray-500 dark:text-gray-400" />
                            )}
                            {getStatusBadge(license.status)}
                          </div>
                        </TableCell>
                        <TableCell>
                          {license.expiresAt > 0 ? formatDate(license.expiresAt) : 'Never'}
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          </div>
          <DialogFooter className="border-t pt-4">
            <Button variant="outline" onClick={() => setShowHistoryDialog(false)}>
              Close
            </Button>
          </DialogFooter>
        </ScrollableDialogContent>
      </Dialog>
    </div>
  );
}
