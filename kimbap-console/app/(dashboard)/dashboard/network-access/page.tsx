'use client';

import { Plus, Trash2, AlertTriangle, CheckCircle, Copy, Shield, Check } from 'lucide-react';
import Link from 'next/link';
import { useState, useEffect, useCallback } from 'react';
import { toast } from 'sonner';

import { AuthLoginDialog } from '@/components/auth-login-dialog';
import { api } from '@/lib/api-client';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
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
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Textarea } from '@/components/ui/textarea';

export default function NetworkAccessPage() {
  const [isRemoteAccessEnabled, setIsRemoteAccessEnabled] = useState(false);
  const [allowAllIPs, setAllowAllIPs] = useState(false);
  const [isConfirmDialogOpen, setIsConfirmDialogOpen] = useState(false);
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const [ipToDelete, setIpToDelete] = useState<{
    id: number;
    ip: string;
  } | null>(null);
  const [isAuthDialogOpen, setIsAuthDialogOpen] = useState(false);
  const [loginPurpose, setLoginPurpose] = useState('');
  const [copiedMcpUrl, setCopiedMcpUrl] = useState(false);
  const [activeKimbapSubdomain, setActiveKimbapSubdomain] = useState<string | null>(null);
  const [enablingRemoteAccess, setEnablingRemoteAccess] = useState(false);
  const [proxyKey, setProxyKey] = useState<string>('');
  const [manualConnection, setManualConnection] = useState<string>('');

  // DNS Configuration States
  const [dnsStatus, setDnsStatus] = useState<'inactive' | 'active' | 'error'>('inactive');
  const [lastIpUpdate, setLastIpUpdate] = useState<string | null>(null);
  const [ipWhitelist, setIpWhitelist] = useState<
    Array<{ id: number; ip: string; description?: string; createdAt?: string }>
  >([]);
  const [newIp, setNewIp] = useState('');
  const [newIpDescription, setNewIpDescription] = useState('');
  const [isAddIpDialogOpen, setIsAddIpDialogOpen] = useState(false);

  // Fetch server info to get proxyKey
  const fetchServerInfo = useCallback(async () => {
    try {
      const response = await api.servers.getInfo();
      if (response.data?.data?.proxyKey) {
        setProxyKey(response.data.data.proxyKey);
      }
    } catch (error) {
      // Failed to fetch server info:
    }
  }, []);

  // API: Get DNS configuration (10014)
  const fetchDnsConfig = useCallback(async () => {
    try {
      const response = await api.networkAccess.getDnsConfig();
      const data = response.data;
      if (data?.data) {
        // If Kimbap.io subdomain exists, remote access is enabled
        if (data.data.kimbapSubdomain) {
          setActiveKimbapSubdomain(`https://${data.data.kimbapSubdomain}/mcp`);
          setIsRemoteAccessEnabled(true);
          setDnsStatus('active');
        } else {
          setActiveKimbapSubdomain('');
          setIsRemoteAccessEnabled(false);
        }

        setManualConnection(data.data.manualConnection || '');
      }
    } catch (error) {
      // Failed to fetch DNS config:
    }
  }, []);

  // API: Get IP whitelist (10012)
  const fetchIpWhitelist = useCallback(async () => {
    try {
      const response = await api.networkAccess.getIpWhitelist();
      const data = response.data;
      if (data?.data?.ipList) {
        setIpWhitelist(data.data.ipList);
        // Check if 0.0.0.0/0 exists (allow all)
        const hasAllowAll = data.data.ipList.some((item: any) => item.ip === '0.0.0.0/0');
        setAllowAllIPs(hasAllowAll);
      }
    } catch (error) {
      // Failed to fetch IP whitelist:
    }
  }, []);

  // Load DNS configuration on mount
  useEffect(() => {
    fetchServerInfo();
    fetchDnsConfig();
    fetchIpWhitelist();
  }, [fetchServerInfo, fetchDnsConfig, fetchIpWhitelist]);

  const handleAllowAllIPsChange = async (checked: boolean) => {
    if (checked) {
      setIsConfirmDialogOpen(true);
    } else {
      try {
        const response = await api.networkAccess.operateIpWhitelist({
          handleType: 3, // not allow all ip
          ipList: [],
          idList: [],
        });
        if (response.status === 200) {
          setAllowAllIPs(false);
          toast.success('IP whitelist protection enabled');
        }
      } catch (error) {
        toast.error('Could not update IP whitelist settings');
      }
    }
  };

  const confirmAllowAllIPs = async () => {
    try {
      const response = await api.networkAccess.operateIpWhitelist({
        handleType: 2, // allow all ip
        ipList: [],
        idList: [],
      });
      if (response.status === 200) {
        setAllowAllIPs(true);
        setIsConfirmDialogOpen(false);
        toast.success('All IPs are now allowed to connect');
      }
    } catch (error) {
      toast.error('Could not update IP whitelist settings');
    }
  };

  const handleLoginRequired = (purpose: string) => {
    setLoginPurpose(purpose);
    setIsAuthDialogOpen(true);
  };

  const handleLoginSuccess = (user: { email: string; name?: string }) => {
    // Proceed with the original action after successful login
    if (loginPurpose === 'ddns') {
      toast.success(`DDNS remote connection enabled for ${user.email}`);
    }
    setIsAuthDialogOpen(false);
  };

  const handleAddIp = async () => {
    if (!newIp.trim()) {
      toast.error('Please enter an IP address or CIDR block');
      return;
    }

    // Validate IP address or CIDR notation
    const ipRegex = /^(\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?$/;
    if (!ipRegex.test(newIp.trim())) {
      toast.error(
        'Please enter a valid IP address (e.g., 192.168.1.1) or CIDR block (e.g., 192.168.1.0/24)',
      );
      return;
    }

    // Validate IP octets are in range 0-255
    const parts = newIp.split('/')[0].split('.');
    if (parts.some((part) => parseInt(part) > 255 || parseInt(part) < 0)) {
      toast.error('Invalid IP address: octets must be between 0 and 255');
      return;
    }

    try {
      const response = await api.networkAccess.operateIpWhitelist({
        handleType: 1, // add ip
        ipList: [newIp.trim()],
        idList: [],
      });
      if (response.status === 200) {
        const newId = Math.max(...ipWhitelist.map((ip) => ip.id), 0) + 1;
        const newEntry = {
          id: newId,
          ip: newIp.trim(),
          description: newIpDescription.trim() || undefined,
          createdAt: new Date().toISOString(),
        };
        setIpWhitelist([...ipWhitelist, newEntry]);
        setNewIp('');
        setNewIpDescription('');
        setIsAddIpDialogOpen(false);
        toast.success('IP rule added');
      }
    } catch (error) {
      toast.error('Could not add IP rule');
    }
  };

  const handleDeleteClick = (item: { id: number; ip: string }) => {
    setIpToDelete(item);
    setIsDeleteConfirmOpen(true);
  };

  const handleConfirmDelete = async () => {
    if (!ipToDelete) return;

    try {
      const response = await api.networkAccess.operateIpWhitelist({
        handleType: 4, // delete ip
        ipList: [],
        idList: [ipToDelete.id],
      });
      if (response.status === 200) {
        setIpWhitelist(ipWhitelist.filter((item) => item.id !== ipToDelete.id));
        toast.success(`IP address ${ipToDelete.ip} removed from whitelist`);
      }
    } catch (error) {
      toast.error('Could not remove IP from whitelist');
    } finally {
      setIsDeleteConfirmOpen(false);
      setIpToDelete(null);
    }
  };

  const handleCopyMcpUrl = () => {
    if (!manualConnection.trim()) {
      toast.error('MCP server address is not configured');
      return;
    }
    const mcpUrl = manualConnection;
    navigator.clipboard
      .writeText(mcpUrl)
      .then(() => {
        setCopiedMcpUrl(true);
        toast.success('MCP server address copied to clipboard');
        setTimeout(() => setCopiedMcpUrl(false), 2000);
      })
      .catch(() => {
        toast.error('Could not copy to clipboard');
      });
  };

  const handleRemoteAccessToggle = async (checked: boolean) => {
    if (checked) {
      // Enable Remote Access - use protocol 10011 to enable Kimbap.io DDNS
      setEnablingRemoteAccess(true);
      try {
        const response = await api.networkAccess.operateDnsRecord({
          handleType: 1, // start remote access
          proxyKey: proxyKey,
        });
        const data = response.data;
        if (response.status === 200 && data?.data?.subdomain) {
          setActiveKimbapSubdomain('https://' + data.data.subdomain + '/mcp');
          setIsRemoteAccessEnabled(true);
          setDnsStatus('active');
          setLastIpUpdate(new Date().toLocaleString());
          toast.success(`Remote access enabled: ${data.data.subdomain}`);
        }
      } catch (error: any) {
        // Failed to enable remote access:
        const errorMessage =
          error?.response?.data?.error?.message ||
          error.message ||
          'Could not enable remote access';
        toast.error(errorMessage);
        setIsRemoteAccessEnabled(false);
      } finally {
        setEnablingRemoteAccess(false);
      }
    } else {
      // Disable Remote Access - call protocol 10011 with handleType=2
      setEnablingRemoteAccess(true);
      try {
        const response = await api.networkAccess.operateDnsRecord({
          handleType: 2, // stop remote access
          proxyKey: proxyKey,
        });
        if (response.status === 200) {
          setIsRemoteAccessEnabled(false);
          setActiveKimbapSubdomain(null);
          setDnsStatus('inactive');
          toast.success('Remote access disabled');
        }
      } catch (error: any) {
        // Failed to disable remote access:
        const errorMessage =
          error?.response?.data?.error?.message ||
          error.message ||
          'Could not disable remote access';
        toast.error(errorMessage);
        // Revert the state if API call failed
        setIsRemoteAccessEnabled(true);
      } finally {
        setEnablingRemoteAccess(false);
      }
    }
  };

  return (
    <>
      <div className="space-y-4">
        <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-4 mb-2">
          <p className="text-sm text-amber-800 dark:text-amber-300">
            <strong>Deprecated:</strong> Network Access has been superseded by{' '}
            <Link href="/dashboard/integrations" className="underline font-medium">
              Integrations
            </Link>
            . This page will be removed in a future release.
          </p>
        </div>
        <div className="space-y-0">
          <h1 className="text-[30px] font-bold">Network Access</h1>
          <p className="text-base text-muted-foreground">
            Configure remote access, DDNS, and IP whitelist.
          </p>
        </div>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
            <div>
              <CardTitle>Current Server Address</CardTitle>
              <CardDescription>Your MCP server address configuration.</CardDescription>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="group flex items-center justify-between p-3 bg-muted rounded-md hover:bg-muted/80 transition-colors gap-3">
              <div className="flex items-center gap-3 min-w-0 flex-1">
                <div className="min-w-0">
                  <p className="text-sm font-mono break-words">
                    {manualConnection || 'Not configured'}
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">MCP Server endpoint address</p>
                </div>
              </div>
              <Button variant="ghost" size="sm" onClick={handleCopyMcpUrl} className="h-8 px-2">
                {copiedMcpUrl ? (
                  <>
                    <Check className="h-4 w-4 mr-1 text-green-600 dark:text-green-400" />
                    <span className="text-green-600 dark:text-green-400">Copied</span>
                  </>
                ) : (
                  <>
                    <Copy className="h-4 w-4 mr-1" />
                    Copy
                  </>
                )}
              </Button>
            </div>

            <div className="p-3 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-md">
              <p className="text-xs text-blue-800 dark:text-blue-200">
                Your MCP endpoint. Point your clients to this address.
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="space-y-0 pb-4">
            <CardTitle>Kimbap DDNS (Optional)</CardTitle>
            <CardDescription>
              Use a *.p-mcp.com domain to reach your server without managing IP changes.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            {/* DNS Status Overview */}
            <div className="space-y-2">
              <Label className="text-sm font-medium">Status</Label>
              <div className="flex items-center justify-between p-4 bg-muted rounded-lg">
                <div className="flex items-center gap-3">
                  <div
                    className={`h-3 w-3 rounded-full ${
                      dnsStatus === 'active'
                        ? 'bg-green-500'
                        : dnsStatus === 'error'
                          ? 'bg-red-500'
                          : 'bg-gray-400 dark:bg-gray-500'
                    }`}
                  />
                  <div>
                    <p className="font-medium">
                      {dnsStatus === 'active'
                        ? 'Kimbap DDNS Active'
                        : dnsStatus === 'error'
                          ? 'Configuration Error'
                          : 'Kimbap DDNS not enabled'}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {dnsStatus === 'active' && lastIpUpdate && `Last updated: ${lastIpUpdate}`}
                      {dnsStatus === 'inactive' &&
                        'Enable this to get an automatically assigned domain name for your server. Enabling Kimbap DDNS does not make your server publicly accessible.'}
                    </p>
                  </div>
                </div>
                {dnsStatus === 'active' && (
                  <Badge
                    variant="outline"
                    className="text-green-600 dark:text-green-400 border-green-600 dark:border-green-400"
                  >
                    <CheckCircle className="w-3 h-3 mr-1" />
                    Active
                  </Badge>
                )}
              </div>
            </div>

            {/* Toggle Control */}
            <div className="flex items-center space-x-2">
              <Switch
                id="remote-access-toggle"
                checked={isRemoteAccessEnabled}
                onCheckedChange={handleRemoteAccessToggle}
                disabled={enablingRemoteAccess}
              />
              <Label htmlFor="remote-access-toggle" className="text-sm font-medium">
                {enablingRemoteAccess
                  ? isRemoteAccessEnabled
                    ? 'Disabling...'
                    : 'Enabling...'
                  : 'Enable Kimbap DDNS'}
              </Label>
            </div>

            {/* p-mcp.com DDNS Configuration */}
            <div className="space-y-4">
              <div className="space-y-2">
                <Label className="text-sm font-medium">Assigned domain</Label>
                {activeKimbapSubdomain ? (
                  <div className="p-4 bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-800 rounded-lg max-w-[480px]">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-mono text-blue-600 dark:text-blue-400">
                        {activeKimbapSubdomain.replace('https://', '').replace('/mcp', '')}
                      </span>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-6 w-6 p-0"
                        aria-label="Copy assigned domain"
                        onClick={() => {
                          navigator.clipboard.writeText(`${activeKimbapSubdomain}`);
                          toast.success('Domain copied to clipboard');
                        }}
                      >
                        <Copy className="w-3 h-3" />
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="grid gap-3 max-w-md">
                    <div className="flex items-center gap-2">
                      <Input
                        id="kimbap-subdomain"
                        value="Not assigned"
                        readOnly
                        aria-label="Assigned domain"
                        className="flex-1 bg-muted"
                      />
                      <span className="text-sm text-muted-foreground">.p-mcp.com</span>
                    </div>
                    <p className="text-xs text-muted-foreground">
                      Your domain will be generated automatically after enabling.
                    </p>
                  </div>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div>
                <CardTitle>IP Whitelist Rules</CardTitle>
                <CardDescription className="mt-1">
                  Control access to your server by IP address. Add IP addresses or CIDR blocks to
                  allow connections.
                </CardDescription>
              </div>
              <div className="flex items-center gap-3">
                <div className="flex items-center space-x-2">
                  <Switch
                    id="allow-all-ips"
                    checked={allowAllIPs}
                    onCheckedChange={handleAllowAllIPsChange}
                  />
                  <Label htmlFor="allow-all-ips" className="text-sm font-medium">
                    Allow All IPs
                  </Label>
                </div>
                <Button onClick={() => setIsAddIpDialogOpen(true)} disabled={allowAllIPs} size="sm">
                  <Plus className="mr-2 h-4 w-4" />
                  Add Rule
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {allowAllIPs ? (
              <div className="flex items-center justify-center py-12 border-2 border-dashed rounded-lg bg-muted/50">
                <div className="text-center space-y-2">
                  <Shield className="h-12 w-12 mx-auto text-muted-foreground/50" />
                  <p className="text-sm font-medium text-muted-foreground">
                    All IP Addresses Allowed
                  </p>
                  <p className="text-xs text-muted-foreground max-w-sm">
                    IP whitelist is disabled. Any IP address can connect to your server. Consider
                    enabling the whitelist for better security.
                  </p>
                </div>
              </div>
            ) : (
              <>
                {ipWhitelist.filter((item) => item.ip !== '0.0.0.0/0').length === 0 ? (
                  <div className="flex items-center justify-center py-12 border-2 border-dashed rounded-lg">
                    <div className="text-center space-y-3">
                      <Shield className="h-12 w-12 mx-auto text-muted-foreground/50" />
                      <div>
                        <p className="text-sm font-medium">No IP Rules Configured</p>
                        <p className="text-xs text-muted-foreground mt-1">
                          Add your first IP rule to control access to your server
                        </p>
                      </div>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setIsAddIpDialogOpen(true)}
                      >
                        <Plus className="mr-2 h-4 w-4" />
                        Add IP Rule
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="border rounded-lg">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead className="w-[200px]">IP Address / CIDR</TableHead>
                          <TableHead>Description</TableHead>
                          <TableHead className="w-[180px]">Created</TableHead>
                          <TableHead className="w-[100px] text-right">Actions</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {ipWhitelist
                          .filter((item) => item.ip !== '0.0.0.0/0')
                          .map((item) => (
                            <TableRow key={item.id}>
                              <TableCell className="font-mono text-sm">{item.ip}</TableCell>
                              <TableCell className="text-sm text-muted-foreground">
                                {item.description || <span className="italic">No description</span>}
                              </TableCell>
                              <TableCell className="text-sm text-muted-foreground">
                                {item.createdAt
                                  ? new Date(item.createdAt).toLocaleDateString('en-US', {
                                      year: 'numeric',
                                      month: 'short',
                                      day: 'numeric',
                                      hour: '2-digit',
                                      minute: '2-digit',
                                    })
                                  : '-'}
                              </TableCell>
                              <TableCell className="text-right">
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleDeleteClick(item)}
                                  aria-label={`Delete IP rule ${item.ip}`}
                                  className="text-destructive hover:text-destructive hover:bg-destructive/10"
                                >
                                  <Trash2 className="h-4 w-4" />
                                </Button>
                              </TableCell>
                            </TableRow>
                          ))}
                      </TableBody>
                    </Table>
                  </div>
                )}
              </>
            )}

            {!allowAllIPs && ipWhitelist.filter((item) => item.ip !== '0.0.0.0/0').length > 0 && (
              <div className="mt-4 p-3 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-md">
                <p className="text-xs text-blue-800 dark:text-blue-200">
                  Only connections from listed IPs will be allowed. Include your current IP to avoid
                  lockout.
                </p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <AlertDialog open={isConfirmDialogOpen} onOpenChange={setIsConfirmDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="text-destructive" /> Are you absolutely sure?
            </AlertDialogTitle>
            <AlertDialogDescription>
              Allowing all IP addresses will expose your server to the public internet. This is a
              security risk and could make your server vulnerable to attacks. We strongly recommend
              using the IP whitelist for enhanced security.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setIsConfirmDialogOpen(false)}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction onClick={confirmAllowAllIPs}>Yes, allow all IPs</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Delete IP Confirmation Dialog */}
      <AlertDialog open={isDeleteConfirmOpen} onOpenChange={setIsDeleteConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete IP Address</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to remove{' '}
              <strong className="font-mono">{ipToDelete?.ip}</strong> from the whitelist? This IP
              address will no longer be able to access your server.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel
              onClick={() => {
                setIsDeleteConfirmOpen(false);
                setIpToDelete(null);
              }}
            >
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete IP
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Add IP Rule Dialog */}
      <Dialog open={isAddIpDialogOpen} onOpenChange={setIsAddIpDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Add IP Whitelist Rule</DialogTitle>
            <DialogDescription>
              Add an IP address or CIDR block to allow connections from specific sources.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="ip-address">
                IP Address or CIDR Block <span className="text-destructive">*</span>
              </Label>
              <Input
                id="ip-address"
                placeholder="e.g., 192.168.1.1 or 10.0.0.0/24"
                value={newIp}
                onChange={(e) => setNewIp(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && newIp.trim()) {
                    handleAddIp();
                  }
                }}
              />
              <p className="text-xs text-muted-foreground">
                Enter a single IP address (e.g., 203.0.113.25) or CIDR notation for a range (e.g.,
                203.0.113.0/24)
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">Description (Optional)</Label>
              <Textarea
                id="description"
                placeholder="e.g., Office network, Home IP, VPN gateway"
                value={newIpDescription}
                onChange={(e) => setNewIpDescription(e.target.value)}
                rows={3}
                className="resize-none"
              />
              <p className="text-xs text-muted-foreground">
                Add a description to help identify this IP rule later
              </p>
            </div>
            <div className="rounded-lg border border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-950/20 p-3">
              <div className="flex gap-2">
                <Shield className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
                <div className="space-y-1">
                  <p className="text-xs font-medium text-blue-900 dark:text-blue-100">
                    Common CIDR Examples:
                  </p>
                  <ul className="text-xs text-blue-800 dark:text-blue-200 space-y-0.5">
                    <li>• Single IP: 192.168.1.100/32</li>
                    <li>• Small network: 192.168.1.0/24 (256 addresses)</li>
                    <li>• Large network: 10.0.0.0/16 (65,536 addresses)</li>
                  </ul>
                </div>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setIsAddIpDialogOpen(false);
                setNewIp('');
                setNewIpDescription('');
              }}
            >
              Cancel
            </Button>
            <Button onClick={handleAddIp} disabled={!newIp.trim()}>
              Add Rule
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Auth Login Dialog for DDNS */}
      <AuthLoginDialog
        open={isAuthDialogOpen}
        onOpenChange={setIsAuthDialogOpen}
        onLoginSuccess={handleLoginSuccess}
      />
    </>
  );
}
