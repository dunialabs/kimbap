import { Server, Shield, Key, Settings, CheckCircle, AlertCircle, Crown, Globe, Lock, Eye, EyeOff, Copy, Download, Upload, RefreshCw, Zap, AlertTriangle, PlayCircle, ArrowRight, Clock, Terminal, Code, Network, Database, Monitor, Users, Plus } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function FirstServerSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-green-500 to-emerald-600 rounded-2xl mb-4">
          <Server className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">Initial Configuration</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Quick and simple server initialization. Get your Kimbap.io MCP server up and running in just a few clicks.
        </p>
      </div>

      {/* Simple Prerequisites */}
      <Alert className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
        <Globe className="h-4 w-4" />
        <AlertDescription>
          <strong>Prerequisites:</strong> Ensure Kimbap.io Console is installed and accessible via HTTPS. You'll need your master password to proceed.
        </AlertDescription>
      </Alert>

      {/* Simple Process Overview */}
      <Card className="border-2 border-green-200 dark:border-green-800 bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950/30 dark:to-emerald-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Zap className="h-5 w-5 text-green-600 dark:text-green-400" />
            Simple 3-Step Process
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="text-center space-y-3">
              <div className="w-12 h-12 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center mx-auto">
                <Globe className="h-6 w-6 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <h4 className="font-semibold text-sm">1. Access Console</h4>
                <p className="text-xs text-muted-foreground">Login with master password</p>
              </div>
            </div>
            <div className="text-center space-y-3">
              <div className="w-12 h-12 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto">
                <Server className="h-6 w-6 text-green-600 dark:text-green-400" />
              </div>
              <div>
                <h4 className="font-semibold text-sm">2. Initialize Server</h4>
                <p className="text-xs text-muted-foreground">Click "Initialize New Server"</p>
              </div>
            </div>
            <div className="text-center space-y-3">
              <div className="w-12 h-12 bg-orange-100 dark:bg-orange-900/30 rounded-full flex items-center justify-center mx-auto">
                <Key className="h-6 w-6 text-orange-600 dark:text-orange-400" />
              </div>
              <div>
                <h4 className="font-semibold text-sm">3. Save Owner Token</h4>
                <p className="text-xs text-muted-foreground">Securely store your admin token</p>
              </div>
            </div>
          </div>
          <p className="text-center text-sm text-green-700 dark:text-green-300 mt-4">
            <strong>Total time: ~2 minutes</strong>
          </p>
        </CardContent>
      </Card>

      {/* Step-by-Step Guide */}
      <div className="space-y-6">
        {/* Step 1: Access Console */}
        <Card className="border-l-4 border-l-blue-500">
          <CardHeader>
            <CardTitle className="flex items-center gap-3">
              <div className="w-8 h-8 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center">
                <span className="font-bold text-blue-600 dark:text-blue-400">1</span>
              </div>
              Access Console
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-start gap-3">
              <Globe className="h-5 w-5 text-blue-600 dark:text-blue-400 mt-1" />
              <div>
                <p className="text-sm font-medium mb-1">Navigate to your Kimbap.io Console URL and login</p>
                <p className="text-sm text-muted-foreground">Enter your master password to access the dashboard</p>
                <div className="mt-2 p-2 bg-slate-100 dark:bg-slate-800 rounded border dark:border-slate-700 font-mono text-xs">
                  https://your-domain.com
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Step 2: Initialize Server */}
        <Card className="border-l-4 border-l-green-500">
          <CardHeader>
            <CardTitle className="flex items-center gap-3">
              <div className="w-8 h-8 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center">
                <span className="font-bold text-green-600 dark:text-green-400">2</span>
              </div>
              Initialize New Server
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-start gap-3">
              <Server className="h-5 w-5 text-green-600 dark:text-green-400 mt-1" />
              <div>
                <p className="text-sm font-medium mb-1">Click the "Initialize New Server" button</p>
                <p className="text-sm text-muted-foreground">The system will automatically set up your MCP server instance</p>
              </div>
            </div>
            
            <Alert className="border-amber-200 bg-amber-50 dark:bg-amber-950/30 dark:border-amber-800">
              <Crown className="h-4 w-4" />
              <AlertDescription className="text-sm">
                <strong>One-time process:</strong> Server initialization creates your Owner Token with full administrative control.
              </AlertDescription>
            </Alert>
          </CardContent>
        </Card>

        {/* Step 3: Save Owner Token */}
        <Card className="border-l-4 border-l-orange-500">
          <CardHeader>
            <CardTitle className="flex items-center gap-3">
              <div className="w-8 h-8 bg-orange-100 dark:bg-orange-900/30 rounded-full flex items-center justify-center">
                <span className="font-bold text-orange-600 dark:text-orange-400">3</span>
              </div>
              Save Your Owner Token
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-start gap-3">
              <Key className="h-5 w-5 text-orange-600 dark:text-orange-400 mt-1" />
              <div>
                <p className="text-sm font-medium mb-1">Immediately copy and securely store your Owner Token</p>
                <p className="text-sm text-muted-foreground">This token cannot be recovered if lost and provides complete admin access</p>
              </div>
            </div>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Card className="border-red-200 bg-red-50 dark:bg-red-950/30 dark:border-red-800">
                <CardContent className="pt-4">
                  <h4 className="font-semibold text-sm text-red-900 dark:text-red-200 mb-2">⚠️ Critical Actions</h4>
                  <ul className="text-xs space-y-1">
                    <li>• Copy token immediately using the copy button</li>
                    <li>• Download backup file</li>
                    <li>• Store in password manager</li>
                    <li>• Never share publicly</li>
                  </ul>
                </CardContent>
              </Card>
              
              <div className="p-3 bg-slate-100 dark:bg-slate-800 rounded border dark:border-slate-700">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs font-medium">Example Token Format:</span>
                  <Button size="sm" variant="outline" className="h-5 text-xs px-2">
                    <Copy className="h-2 w-2 mr-1" />
                    Copy
                  </Button>
                </div>
                <div className="font-mono text-xs text-slate-700 dark:text-slate-300 break-all">
                  owner_abc123...xyz789
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

      </div>

    </div>
  )
}