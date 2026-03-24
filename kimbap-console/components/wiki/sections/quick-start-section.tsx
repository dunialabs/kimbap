import { AlertCircle, CheckCircle, Clock, Globe, Key, Server, Shield, Zap, ArrowRight, Eye, Download, Copy, Lock, Crown, UserCheck, AlertTriangle, PlayCircle } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

export function QuickStartSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-2xl mb-4">
          <Zap className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">5-Minute Quick Start</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Get your Kimbap.io MCP platform up and running in minutes. This guide covers everything from initial access to configuring your first AI tool integration.
        </p>
      </div>

      {/* Prerequisites */}
      <Alert className="border-amber-200 bg-amber-50 dark:bg-amber-950/30 dark:border-amber-800">
        <AlertTriangle className="h-4 w-4" />
        <AlertDescription>
          <div className="space-y-3">
            <p><strong>Before You Start:</strong> Ensure you have the following ready:</p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <div className="flex items-start gap-2">
                <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                <div>
                  <p className="text-sm font-medium">Admin Access</p>
                  <p className="text-xs text-muted-foreground">Owner or Admin role required for setup</p>
                </div>
              </div>
              <div className="flex items-start gap-2">
                <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                <div>
                  <p className="text-sm font-medium">Master Password</p>
                  <p className="text-xs text-muted-foreground">Provided by your organization admin</p>
                </div>
              </div>
              <div className="flex items-start gap-2">
                <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                <div>
                  <p className="text-sm font-medium">Modern Browser</p>
                  <p className="text-xs text-muted-foreground">Chrome, Firefox, Safari, or Edge</p>
                </div>
              </div>
              <div className="flex items-start gap-2">
                <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                <div>
                  <p className="text-sm font-medium">AI Assistant</p>
                  <p className="text-xs text-muted-foreground">Claude Desktop, Cursor, or VS Code</p>
                </div>
              </div>
            </div>
          </div>
        </AlertDescription>
      </Alert>

      {/* Time Estimate */}
      <div className="flex items-center justify-center gap-6 py-4 bg-gradient-to-r from-blue-50 dark:from-blue-950 to-indigo-50 dark:to-indigo-950 rounded-lg border border-blue-200 dark:border-blue-800">
        <div className="flex items-center gap-2">
          <Clock className="h-5 w-5 text-blue-600 dark:text-blue-400" />
          <span className="font-medium">Estimated Time: 3-5 minutes</span>
        </div>
        <div className="flex items-center gap-2">
          <Shield className="h-5 w-5 text-green-600 dark:text-green-400" />
          <span className="font-medium">Difficulty: Beginner</span>
        </div>
        <div className="flex items-center gap-2">
          <Zap className="h-5 w-5 text-orange-600 dark:text-orange-400" />
          <span className="font-medium">Result: Fully functional MCP server</span>
        </div>
      </div>

      {/* Step 1: Console Access */}
      <Card className="border-l-4 border-l-blue-500">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-3">
              <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center">
                <span className="font-bold text-blue-600 dark:text-blue-400">1</span>
              </div>
              Access Kimbap.io Console
            </CardTitle>
            <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300">2 minutes</Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-6 border border-slate-200 dark:border-slate-700">
            <div className="text-center mb-4">
              <p className="text-lg font-semibold text-slate-700 dark:text-slate-300 mb-2">Console Access Flow</p>
              <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border-2 border-dashed border-slate-300 dark:border-slate-600">
                <p className="text-muted-foreground mb-2">[Screenshot Placeholder]</p>
                <p className="text-sm text-muted-foreground">
                  Add screenshot showing the Kimbap.io Console login page with master password field
                </p>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="space-y-4">
              <h4 className="font-semibold flex items-center gap-2">
                <Globe className="h-4 w-4" />
                Access Steps
              </h4>
              <ol className="space-y-3">
                <li className="flex items-start gap-3">
                  <div className="w-6 h-6 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                    1
                  </div>
                  <div>
                    <p className="font-medium text-sm">Navigate to Console</p>
                    <p className="text-sm text-muted-foreground">Open your browser and go to your Kimbap.io instance URL</p>
                    <div className="mt-2 p-2 bg-slate-100 dark:bg-slate-800 rounded border dark:border-slate-700 font-mono text-xs">
                      https://your-domain.kimbap.io
                    </div>
                  </div>
                </li>
                <li className="flex items-start gap-3">
                  <div className="w-6 h-6 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                    2
                  </div>
                  <div>
                    <p className="font-medium text-sm">Enter Master Password</p>
                    <p className="text-sm text-muted-foreground">Use the master password provided by your administrator</p>
                    <Alert className="mt-2 bg-amber-50 dark:bg-amber-950/30 border-amber-200 dark:border-amber-800">
                      <Eye className="h-4 w-4" />
                      <AlertDescription className="text-xs">
                        <strong>Security Tip:</strong> Master passwords are case-sensitive and session-based for security
                      </AlertDescription>
                    </Alert>
                  </div>
                </li>
              </ol>
            </div>

            <div className="space-y-4">
              <h4 className="font-semibold flex items-center gap-2">
                <AlertTriangle className="h-4 w-4 text-amber-600 dark:text-amber-400" />
                Common Issues
              </h4>
              <div className="space-y-3">
                <div className="p-3 bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded-lg">
                  <p className="text-sm font-medium text-red-900 dark:text-red-200 mb-1">Password Not Accepted</p>
                  <ul className="text-xs text-red-800 dark:text-red-300 space-y-1">
                    <li>• Check for typos or case sensitivity</li>
                    <li>• Verify with your administrator</li>
                    <li>• Clear browser cache if needed</li>
                  </ul>
                </div>
                <div className="p-3 bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded-lg">
                  <p className="text-sm font-medium text-red-900 dark:text-red-200 mb-1">Page Won't Load</p>
                  <ul className="text-xs text-red-800 dark:text-red-300 space-y-1">
                    <li>• Check your internet connection</li>
                    <li>• Verify the URL is correct</li>
                    <li>• Try a different browser</li>
                  </ul>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Step 2: Server Initialization */}
      <Card className="border-l-4 border-l-green-500">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-3">
              <div className="w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center">
                <span className="font-bold text-green-600 dark:text-green-400">2</span>
              </div>
              Initialize Your First Server
            </CardTitle>
            <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300">2 minutes</Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <Alert className="border-orange-200 bg-orange-50 dark:bg-orange-950/30 dark:border-orange-800">
            <Crown className="h-4 w-4" />
            <AlertDescription>
              <strong>Critical Step:</strong> Server initialization creates your Owner Token. This token provides full administrative access and cannot be recovered if lost!
            </AlertDescription>
          </Alert>

          <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-6 border border-slate-200 dark:border-slate-700">
            <div className="text-center mb-4">
              <p className="text-lg font-semibold text-slate-700 dark:text-slate-300 mb-2">Server Initialization Process</p>
              <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border-2 border-dashed border-slate-300 dark:border-slate-600 mb-4">
                <p className="text-muted-foreground mb-2">[Video Placeholder]</p>
                <p className="text-sm text-muted-foreground">
                  Add step-by-step video showing the server initialization process
                </p>
              </div>
              <Button size="sm" variant="outline">
                <PlayCircle className="h-4 w-4 mr-2" />
                Watch Setup Video
              </Button>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="space-y-4">
              <h4 className="font-semibold">Initialization Process</h4>
              <ol className="space-y-3">
                <li className="flex items-start gap-3">
                  <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="font-medium text-sm">Click "Initialize New Server"</p>
                    <p className="text-xs text-muted-foreground">Located on the main welcome page</p>
                  </div>
                </li>
                <li className="flex items-start gap-3">
                  <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="font-medium text-sm">Verify Master Password</p>
                    <p className="text-xs text-muted-foreground">Same password used to access the console</p>
                  </div>
                </li>
                <li className="flex items-start gap-3">
                  <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="font-medium text-sm">Server Creation</p>
                    <p className="text-xs text-muted-foreground">System creates your MCP server instance</p>
                  </div>
                </li>
                <li className="flex items-start gap-3">
                  <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="font-medium text-sm">Owner Token Generation</p>
                    <p className="text-xs text-muted-foreground">One-time display of your admin token</p>
                  </div>
                </li>
              </ol>
            </div>

            <div className="space-y-4">
              <div className="p-4 bg-red-50 dark:bg-red-950/30 border-2 border-red-200 dark:border-red-800 rounded-lg">
                <h4 className="font-semibold text-red-900 dark:text-red-200 mb-3 flex items-center gap-2">
                  <Lock className="h-4 w-4" />
                  Token Security Guide
                </h4>
                <div className="space-y-2">
                  <div className="flex items-start gap-2">
                    <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium">✓ DO: Save Immediately</p>
                      <p className="text-xs text-muted-foreground">Copy to secure password manager</p>
                    </div>
                  </div>
                  <div className="flex items-start gap-2">
                    <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium">✓ DO: Download Backup</p>
                      <p className="text-xs text-muted-foreground">Use the download button provided</p>
                    </div>
                  </div>
                  <div className="flex items-start gap-2">
                    <AlertTriangle className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium">✗ DON'T: Share Publicly</p>
                      <p className="text-xs text-muted-foreground">Never paste in chat or email</p>
                    </div>
                  </div>
                  <div className="flex items-start gap-2">
                    <AlertTriangle className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium">✗ DON'T: Skip Backup</p>
                      <p className="text-xs text-muted-foreground">Token cannot be recovered</p>
                    </div>
                  </div>
                </div>
                <div className="mt-3 p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                  <p className="text-xs font-mono">owner_abc123xyz789_admin_token</p>
                  <div className="flex gap-2 mt-2">
                    <Button size="sm" variant="outline" className="text-xs h-6">
                      <Copy className="h-3 w-3 mr-1" />
                      Copy
                    </Button>
                    <Button size="sm" variant="outline" className="text-xs h-6">
                      <Download className="h-3 w-3 mr-1" />
                      Download
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Step 3: First Tool Configuration */}
      <Card className="border-l-4 border-l-purple-500">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-3">
              <div className="w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-full flex items-center justify-center">
                <span className="font-bold text-purple-600 dark:text-purple-400">3</span>
              </div>
              Configure Your First Tool
            </CardTitle>
            <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300">1 minute</Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-muted-foreground">
            After server initialization, you'll be redirected to the dashboard where you can add your first MCP tool integration.
          </p>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
              <CardHeader>
                <CardTitle className="text-base">Popular First Tools</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="space-y-2">
                  <div className="flex items-center gap-3 p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                    <div className="w-8 h-8 bg-gray-900 rounded flex items-center justify-center">
                      <span className="text-white text-xs font-bold">GH</span>
                    </div>
                    <div>
                      <p className="font-medium text-sm">GitHub Integration</p>
                      <p className="text-xs text-muted-foreground">Repository management and code operations</p>
                    </div>
                    <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Recommended</Badge>
                  </div>
                  <div className="flex items-center gap-3 p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                    <div className="w-8 h-8 bg-black rounded flex items-center justify-center">
                      <span className="text-white text-xs font-bold">N</span>
                    </div>
                    <div>
                      <p className="font-medium text-sm">Notion Workspace</p>
                      <p className="text-xs text-muted-foreground">Document and database management</p>
                    </div>
                    <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Easy Setup</Badge>
                  </div>
                  <div className="flex items-center gap-3 p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                    <div className="w-8 h-8 bg-blue-600 rounded flex items-center justify-center">
                      <span className="text-white text-xs font-bold">PG</span>
                    </div>
                    <div>
                      <p className="font-medium text-sm">PostgreSQL</p>
                      <p className="text-xs text-muted-foreground">Database queries and management</p>
                    </div>
                    <Badge className="bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 text-xs">Advanced</Badge>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
              <CardHeader>
                <CardTitle className="text-base">Configuration Steps</CardTitle>
              </CardHeader>
              <CardContent>
                <ol className="space-y-3">
                  <li className="flex items-start gap-3">
                    <div className="w-6 h-6 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                      1
                    </div>
                    <div>
                      <p className="font-medium text-sm">Navigate to Tools</p>
                      <p className="text-xs text-muted-foreground">Click "Tool Configuration" in sidebar</p>
                    </div>
                  </li>
                  <li className="flex items-start gap-3">
                    <div className="w-6 h-6 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                      2
                    </div>
                    <div>
                      <p className="font-medium text-sm">Select Tool Type</p>
                      <p className="text-xs text-muted-foreground">Choose from preset integrations</p>
                    </div>
                  </li>
                  <li className="flex items-start gap-3">
                    <div className="w-6 h-6 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                      3
                    </div>
                    <div>
                      <p className="font-medium text-sm">Add Credentials</p>
                      <p className="text-xs text-muted-foreground">Enter API keys or tokens</p>
                    </div>
                  </li>
                  <li className="flex items-start gap-3">
                    <div className="w-6 h-6 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                      4
                    </div>
                    <div>
                      <p className="font-medium text-sm">Test & Save</p>
                      <p className="text-xs text-muted-foreground">Verify connection works</p>
                    </div>
                  </li>
                </ol>
                <Alert className="mt-4 bg-white dark:bg-slate-800 border-green-200 dark:border-green-800">
                  <Shield className="h-4 w-4" />
                  <AlertDescription className="text-xs">
                    All credentials are encrypted with AES-256 before storage
                  </AlertDescription>
                </Alert>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>

    </div>
  )
}