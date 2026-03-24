import { Shield, CheckCircle, Settings, AlertCircle, Monitor, Key, Network, Globe, Server, Eye, Lock, Zap, ChevronRight, Copy, Play, Wifi, User, Clock, Fingerprint, RefreshCw, ToggleLeft, ToggleRight, Database } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function DeskSetupSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-green-500 to-emerald-600 rounded-2xl mb-4">
          <Settings className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">Connecting to Servers</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Complete guide to setting up MCP Desk, connecting to your organization's servers, and configuring secure access for optimal performance.
        </p>
      </div>

      {/* Quick Setup Overview */}
      <Card className="border-2 border-green-200 dark:border-green-800 bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950/30 dark:to-emerald-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-green-800 dark:text-green-300">
            <Zap className="h-6 w-6" />
            Quick Setup Overview
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-green-700 dark:text-green-300">
            Setting up MCP Desk takes just a few minutes. Follow this step-by-step process to connect to your organization's MCP servers and start using AI-powered tools.
          </p>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-green-200 dark:border-green-800">
              <div className="w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Key className="h-5 w-5 text-green-600 dark:text-green-400" />
              </div>
              <p className="font-medium text-sm mb-1">Get Credentials</p>
              <p className="text-xs text-muted-foreground">Obtain server URL and token</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-blue-200 dark:border-blue-800">
              <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Network className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              </div>
              <p className="font-medium text-sm mb-1">Add Connection</p>
              <p className="text-xs text-muted-foreground">Configure server connection</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-purple-200 dark:border-purple-800">
              <div className="w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <CheckCircle className="h-5 w-5 text-purple-600 dark:text-purple-400" />
              </div>
              <p className="font-medium text-sm mb-1">Test Connection</p>
              <p className="text-xs text-muted-foreground">Verify connectivity</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-orange-200 dark:border-orange-800">
              <div className="w-10 h-10 bg-orange-100 dark:bg-orange-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Monitor className="h-5 w-5 text-orange-600 dark:text-orange-400" />
              </div>
              <p className="font-medium text-sm mb-1">Start Using</p>
              <p className="text-xs text-muted-foreground">Access tools and features</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Prerequisites */}
      <Alert className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
        <Shield className="h-4 w-4" />
        <AlertDescription>
          <strong>Before You Start:</strong> Ensure MCP Desk is installed and you have received server connection details from your administrator. You'll need a server URL and personal access token.
        </AlertDescription>
      </Alert>

      {/* Setup Steps */}
      <Tabs defaultValue="security" className="w-full">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="security" className="flex items-center gap-2 text-xs">
            <Lock className="h-4 w-4" />
            Security Setup
          </TabsTrigger>
          <TabsTrigger value="connection" className="flex items-center gap-2 text-xs">
            <Network className="h-4 w-4" />
            Server Connection
          </TabsTrigger>
          <TabsTrigger value="verification" className="flex items-center gap-2 text-xs">
            <CheckCircle className="h-4 w-4" />
            Verification
          </TabsTrigger>
          <TabsTrigger value="controls" className="flex items-center gap-2 text-xs">
            <Settings className="h-4 w-4" />
            Access Controls
          </TabsTrigger>
        </TabsList>

        {/* Security Setup */}
        <TabsContent value="security" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Lock className="h-5 w-5 text-red-600 dark:text-red-400" />
                Master Password & Security Configuration
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="bg-red-50 dark:bg-red-950/30 border-red-200 dark:border-red-800">
                <Shield className="h-4 w-4" />
                <AlertDescription>
                  <strong>First Launch:</strong> MCP Desk will prompt you to create a Master Password on first launch. This is required to protect all your server connections and enable biometric authentication.
                </AlertDescription>
              </Alert>
              
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Master Password Setup</h4>
                  <ol className="space-y-3">
                    <li className="flex items-start gap-3">
                      <div className="w-6 h-6 bg-red-100 dark:bg-red-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                        1
                      </div>
                      <div>
                        <p className="font-medium text-sm">Create Strong Password</p>
                        <p className="text-xs text-muted-foreground">Use at least 12 characters with mixed case, numbers, and symbols</p>
                      </div>
                    </li>
                    <li className="flex items-start gap-3">
                      <div className="w-6 h-6 bg-red-100 dark:bg-red-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                        2
                      </div>
                      <div>
                        <p className="font-medium text-sm">Confirm Password</p>
                        <p className="text-xs text-muted-foreground">Re-enter to verify and prevent typos</p>
                      </div>
                    </li>
                    <li className="flex items-start gap-3">
                      <div className="w-6 h-6 bg-red-100 dark:bg-red-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                        3
                      </div>
                      <div>
                        <p className="font-medium text-sm">Set Auto-Lock Timer</p>
                        <p className="text-xs text-muted-foreground">Choose timeout: 5min, 15min, 30min, 1hr, or Never</p>
                      </div>
                    </li>
                    <li className="flex items-start gap-3">
                      <div className="w-6 h-6 bg-red-100 dark:bg-red-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                        4
                      </div>
                      <div>
                        <p className="font-medium text-sm">Enable Biometric Auth</p>
                        <p className="text-xs text-muted-foreground">Setup fingerprint, Face ID, or Windows Hello (optional)</p>
                      </div>
                    </li>
                  </ol>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Security Features</h4>
                  <div className="space-y-3">
                    <div className="flex items-start gap-3 p-3 bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded-lg">
                      <Clock className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5" />
                      <div>
                        <p className="text-sm font-medium text-red-800 dark:text-red-300">Auto-Lock Protection</p>
                        <p className="text-xs text-red-600 dark:text-red-400">Automatically locks after inactivity to protect connections</p>
                      </div>
                    </div>
                    
                    <div className="flex items-start gap-3 p-3 bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 rounded-lg">
                      <Fingerprint className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                      <div>
                        <p className="text-sm font-medium text-blue-800 dark:text-blue-300">Biometric Authentication</p>
                        <p className="text-xs text-blue-600 dark:text-blue-400">Use fingerprint or facial recognition for quick unlock</p>
                      </div>
                    </div>
                    
                    <div className="flex items-start gap-3 p-3 bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded-lg">
                      <Shield className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                      <div>
                        <p className="text-sm font-medium text-green-800 dark:text-green-300">Local Encryption</p>
                        <p className="text-xs text-green-600 dark:text-green-400">All credentials encrypted locally, never sent to cloud</p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              
              <Alert className="bg-amber-50 dark:bg-amber-950/30 border-amber-200 dark:border-amber-800">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription className="text-xs">
                  <strong>Important:</strong> The master password cannot be recovered if lost. MCP Desk doesn't store passwords on remote servers for security reasons.
                </AlertDescription>
              </Alert>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Server Connection Setup */}
        <TabsContent value="connection" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Server className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                Adding Your First Server Connection
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Step-by-Step Process</h4>
                  <ol className="space-y-3">
                    <li className="flex items-start gap-3">
                      <div className="w-6 h-6 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                        1
                      </div>
                      <div>
                        <p className="font-medium text-sm">Launch MCP Desk</p>
                        <p className="text-xs text-muted-foreground">Open the application from your desktop or start menu</p>
                      </div>
                    </li>
                    <li className="flex items-start gap-3">
                      <div className="w-6 h-6 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                        2
                      </div>
                      <div>
                        <p className="font-medium text-sm">Click "Add Server"</p>
                        <p className="text-xs text-muted-foreground">Look for the "+" button or "Add Server Connection" option</p>
                      </div>
                    </li>
                    <li className="flex items-start gap-3">
                      <div className="w-6 h-6 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                        3
                      </div>
                      <div>
                        <p className="font-medium text-sm">Enter Server Details</p>
                        <p className="text-xs text-muted-foreground">Input the server URL and your access token</p>
                      </div>
                    </li>
                    <li className="flex items-start gap-3">
                      <div className="w-6 h-6 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                        4
                      </div>
                      <div>
                        <p className="font-medium text-sm">Test & Save</p>
                        <p className="text-xs text-muted-foreground">Verify the connection works and save your settings</p>
                      </div>
                    </li>
                  </ol>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Connection Form</h4>
                  <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-4 border border-slate-200 dark:border-slate-700">
                    <div className="space-y-3">
                      <div>
                        <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Server Name</label>
                        <div className="mt-1 p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded text-sm font-mono text-slate-500 dark:text-slate-400">
                          My Organization Server
                        </div>
                      </div>
                      <div>
                        <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Server URL</label>
                        <div className="mt-1 p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded text-sm font-mono text-slate-500 dark:text-slate-400">
                          https://mcp.company.com
                        </div>
                      </div>
                      <div>
                        <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Access Token</label>
                        <div className="mt-1 p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded text-sm font-mono text-slate-500 dark:text-slate-400">
                          •••••••••••••••••••••••••••••••••••••••••
                        </div>
                      </div>
                      <div className="flex gap-2 pt-2">
                        <Button size="sm" className="flex-1 text-xs">Test Connection</Button>
                        <Button size="sm" variant="outline" className="flex-1 text-xs">Save</Button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <Alert className="bg-amber-50 dark:bg-amber-950/30 border-amber-200 dark:border-amber-800">
                <Key className="h-4 w-4" />
                <AlertDescription className="text-xs">
                  <strong>Token Security:</strong> Your access token is encrypted and stored securely on your device. Never share your personal access token with others.
                </AlertDescription>
              </Alert>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Connection Verification */}
        <TabsContent value="verification" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400" />
                Connection Verification & Testing
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Connection Tests</h4>
                  <div className="space-y-3">
                    <div className="flex items-start gap-3 p-3 bg-green-50 dark:bg-green-950/30 rounded-lg border border-green-200 dark:border-green-800">
                      <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                      <div>
                        <p className="font-medium text-sm">Network Connectivity</p>
                        <p className="text-xs text-muted-foreground">Can reach the server URL</p>
                      </div>
                    </div>
                    <div className="flex items-start gap-3 p-3 bg-green-50 dark:bg-green-950/30 rounded-lg border border-green-200 dark:border-green-800">
                      <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                      <div>
                        <p className="font-medium text-sm">Authentication</p>
                        <p className="text-xs text-muted-foreground">Access token is valid and active</p>
                      </div>
                    </div>
                    <div className="flex items-start gap-3 p-3 bg-green-50 dark:bg-green-950/30 rounded-lg border border-green-200 dark:border-green-800">
                      <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                      <div>
                        <p className="font-medium text-sm">API Access</p>
                        <p className="text-xs text-muted-foreground">Can retrieve available tools and permissions</p>
                      </div>
                    </div>
                    <div className="flex items-start gap-3 p-3 bg-green-50 dark:bg-green-950/30 rounded-lg border border-green-200 dark:border-green-800">
                      <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                      <div>
                        <p className="font-medium text-sm">Tool Discovery</p>
                        <p className="text-xs text-muted-foreground">Available MCP tools detected</p>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Troubleshooting Common Issues</h4>
                  <div className="space-y-3">
                    <Card className="border-red-200 bg-red-50 dark:bg-red-950/30 dark:border-red-800">
                      <CardContent className="p-3">
                        <div className="flex items-start gap-2">
                          <AlertCircle className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5" />
                          <div>
                            <p className="font-medium text-sm text-red-800 dark:text-red-300">Connection Timeout</p>
                            <p className="text-xs text-red-700 dark:text-red-300 mt-1">Check network connection and server URL</p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-red-200 bg-red-50 dark:bg-red-950/30 dark:border-red-800">
                      <CardContent className="p-3">
                        <div className="flex items-start gap-2">
                          <AlertCircle className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5" />
                          <div>
                            <p className="font-medium text-sm text-red-800 dark:text-red-300">Invalid Token</p>
                            <p className="text-xs text-red-700 dark:text-red-300 mt-1">Verify token with administrator or request new one</p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-red-200 bg-red-50 dark:bg-red-950/30 dark:border-red-800">
                      <CardContent className="p-3">
                        <div className="flex items-start gap-2">
                          <AlertCircle className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5" />
                          <div>
                            <p className="font-medium text-sm text-red-800 dark:text-red-300">SSL Certificate Error</p>
                            <p className="text-xs text-red-700 dark:text-red-300 mt-1">Check server certificate or contact IT support</p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Granular Access Controls */}
        <TabsContent value="controls" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Settings className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                Granular Access Controls
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="bg-purple-50 dark:bg-purple-950/30 border-purple-200 dark:border-purple-800">
                <Eye className="h-4 w-4" />
                <AlertDescription>
                  <strong>Fine-grained Control:</strong> MCP Desk provides individual toggle switches for clients, servers, tools, and data access, giving you complete control over what each AI assistant can access.
                </AlertDescription>
              </Alert>
              
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Control Categories</h4>
                  <div className="space-y-3">
                    <div className="p-3 bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 rounded-lg">
                      <div className="flex items-center justify-between mb-2">
                        <span className="font-medium text-sm flex items-center gap-2">
                          <Monitor className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                          Client Controls
                        </span>
                        <ToggleRight className="h-4 w-4 text-green-600 dark:text-green-400" />
                      </div>
                      <p className="text-xs text-blue-700 dark:text-blue-300">Enable/disable access for specific AI clients (Claude Desktop, Cursor, etc.)</p>
                    </div>
                    
                    <div className="p-3 bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded-lg">
                      <div className="flex items-center justify-between mb-2">
                        <span className="font-medium text-sm flex items-center gap-2">
                          <Server className="h-4 w-4 text-green-600 dark:text-green-400" />
                          Server Controls
                        </span>
                        <ToggleRight className="h-4 w-4 text-green-600 dark:text-green-400" />
                      </div>
                      <p className="text-xs text-green-700 dark:text-green-300">Control access to individual MCP servers and their connections</p>
                    </div>
                    
                    <div className="p-3 bg-orange-50 dark:bg-orange-950/30 border border-orange-200 dark:border-orange-800 rounded-lg">
                      <div className="flex items-center justify-between mb-2">
                        <span className="font-medium text-sm flex items-center gap-2">
                          <Settings className="h-4 w-4 text-orange-600 dark:text-orange-400" />
                          Tool Controls
                        </span>
                        <ToggleLeft className="h-4 w-4 text-red-600 dark:text-red-400" />
                      </div>
                      <p className="text-xs text-orange-700 dark:text-orange-300">Granular permissions for each tool (GitHub, Notion, Database, etc.)</p>
                    </div>
                    
                    <div className="p-3 bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded-lg">
                      <div className="flex items-center justify-between mb-2">
                        <span className="font-medium text-sm flex items-center gap-2">
                          <Database className="h-4 w-4 text-red-600 dark:text-red-400" />
                          Data Controls
                        </span>
                        <ToggleRight className="h-4 w-4 text-green-600 dark:text-green-400" />
                      </div>
                      <p className="text-xs text-red-700 dark:text-red-300">Control data access levels and sensitive information exposure</p>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Control Examples</h4>
                  <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-4 border border-slate-200 dark:border-slate-700">
                    <h5 className="font-medium text-sm mb-3 text-slate-800 dark:text-slate-200">Example Configuration:</h5>
                    <div className="space-y-2">
                      <div className="flex items-center justify-between text-xs">
                        <span className="flex items-center gap-2">
                          <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                          Claude Desktop MCP
                        </span>
                        <span className="text-green-600 dark:text-green-400 font-medium">Enabled</span>
                      </div>
                      <div className="flex items-center justify-between text-xs ml-4">
                        <span className="flex items-center gap-2">
                          <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                          GitHub Integration
                        </span>
                        <span className="text-green-600 dark:text-green-400 font-medium">Enabled</span>
                      </div>
                      <div className="flex items-center justify-between text-xs ml-4">
                        <span className="flex items-center gap-2">
                          <div className="w-2 h-2 bg-red-500 rounded-full"></div>
                          Database Access
                        </span>
                        <span className="text-red-600 dark:text-red-400 font-medium">Disabled</span>
                      </div>
                      <div className="flex items-center justify-between text-xs">
                        <span className="flex items-center gap-2">
                          <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
                          Cursor MCP
                        </span>
                        <span className="text-yellow-600 dark:text-yellow-400 font-medium">Limited</span>
                      </div>
                    </div>
                  </div>
                  
                  <Alert className="bg-green-50 dark:bg-green-950/30 border-green-200 dark:border-green-800">
                    <Shield className="h-4 w-4" />
                    <AlertDescription className="text-xs">
                      <strong>Security Benefit:</strong> These controls ensure each AI client only has access to the tools and data it needs, following the principle of least privilege.
                    </AlertDescription>
                  </Alert>
                </div>
              </div>
              
              <div className="space-y-4">
                <h4 className="font-semibold">Runtime Control</h4>
                <div className="p-4 bg-gradient-to-r from-purple-50 to-indigo-50 dark:from-purple-950/30 dark:to-indigo-950/30 border border-purple-200 dark:border-purple-800 rounded-lg">
                  <div className="flex items-start gap-3">
                    <Eye className="h-5 w-5 text-purple-600 dark:text-purple-400 mt-0.5" />
                    <div>
                      <p className="font-medium text-sm text-purple-800 dark:text-purple-300 mb-1">Real-time Permission Management</p>
                      <p className="text-xs text-purple-700 dark:text-purple-300 mb-3">
                        Access controls can be changed while MCP Desk is running. Changes take effect immediately without requiring client restart.
                      </p>
                      <div className="flex items-center gap-2">
                        <RefreshCw className="h-3 w-3 text-purple-600 dark:text-purple-400" />
                        <span className="text-xs text-purple-600 dark:text-purple-400">Controls apply instantly to active sessions</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Multiple Servers */}
      <Card className="border-2 border-indigo-200 dark:border-indigo-800 bg-gradient-to-br from-indigo-50 to-blue-50 dark:from-indigo-950/30 dark:to-blue-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-indigo-800 dark:text-indigo-300">
            <Network className="h-5 w-5" />
            Managing Multiple Server Connections
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-indigo-700 dark:text-indigo-300">
            MCP Desk supports connections to multiple MCP servers simultaneously, allowing you to access tools from different organizations or teams.
          </p>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-3">
              <h4 className="font-semibold">Benefits of Multiple Connections</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-indigo-600 dark:text-indigo-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Access Multiple Organizations</p>
                    <p className="text-xs text-muted-foreground">Connect to different company servers</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-indigo-600 dark:text-indigo-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Separate Development & Production</p>
                    <p className="text-xs text-muted-foreground">Different environments for testing</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-indigo-600 dark:text-indigo-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Team-Specific Tools</p>
                    <p className="text-xs text-muted-foreground">Access tools relevant to different projects</p>
                  </div>
                </li>
              </ul>
            </div>
            
            <div className="space-y-3">
              <h4 className="font-semibold">Connection Management</h4>
              <div className="bg-white dark:bg-slate-800 rounded-lg p-3 border border-indigo-200 dark:border-indigo-800">
                <div className="space-y-2">
                  <div className="flex items-center justify-between p-2 bg-green-50 dark:bg-green-950/30 rounded border">
                    <div>
                      <p className="text-sm font-medium">Production Server</p>
                      <p className="text-xs text-muted-foreground">company.com</p>
                    </div>
                    <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Connected</Badge>
                  </div>
                  <div className="flex items-center justify-between p-2 bg-blue-50 dark:bg-blue-950/30 rounded border">
                    <div>
                      <p className="text-sm font-medium">Configured Server</p>
                      <p className="text-xs text-muted-foreground">dev.company.com</p>
                    </div>
                    <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Connected</Badge>
                  </div>
                  <div className="flex items-center justify-between p-2 bg-gray-50 dark:bg-gray-950/50 rounded border">
                    <div>
                      <p className="text-sm font-medium">Partner Server</p>
                      <p className="text-xs text-muted-foreground">partner.com</p>
                    </div>
                    <Badge variant="outline" className="text-xs">Offline</Badge>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

    </div>
  )
}
