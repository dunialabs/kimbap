import { Monitor, Shield, Users, Settings, BarChart, Lock, Network, Server, Database, Eye, Zap, CheckCircle, ArrowRight, Globe, Crown, Key, Bell, RefreshCw, Download, Upload, Code } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"

export function ConsoleOverviewSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-blue-500 to-cyan-600 rounded-2xl mb-4">
          <Monitor className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-2xl lg:text-3xl font-bold mb-2">Console Overview</h1>
        <p className="text-base lg:text-lg text-muted-foreground max-w-2xl mx-auto">
          The web-based management interface for your Kimbap.io MCP infrastructure. 
          Control servers, configure tools, manage teams, and monitor usage from a single dashboard.
        </p>
      </div>

      {/* Console Purpose */}
      <Card className="border-2 border-blue-200 dark:border-blue-800 bg-gradient-to-br from-blue-50 to-cyan-50 dark:from-blue-950/30 dark:to-cyan-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            What is MCP Console?
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            MCP Console is the administrative control center for your Kimbap.io platform. It provides a comprehensive web interface 
            for managing your entire MCP infrastructure, from server configuration to team collaboration and usage monitoring.
          </p>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-blue-200 dark:border-blue-800">
              <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Server className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              </div>
              <p className="font-medium text-sm mb-1">Server Management</p>
              <p className="text-xs text-muted-foreground">Configure and monitor MCP servers</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-green-200 dark:border-green-800">
              <div className="w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Settings className="h-5 w-5 text-green-600 dark:text-green-400" />
              </div>
              <p className="font-medium text-sm mb-1">Tool Configuration</p>
              <p className="text-xs text-muted-foreground">Set up and test tool integrations</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-purple-200 dark:border-purple-800">
              <div className="w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Users className="h-5 w-5 text-purple-600 dark:text-purple-400" />
              </div>
              <p className="font-medium text-sm mb-1">Team Management</p>
              <p className="text-xs text-muted-foreground">Invite users and manage permissions</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-orange-200 dark:border-orange-800">
              <div className="w-10 h-10 bg-orange-100 dark:bg-orange-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <BarChart className="h-5 w-5 text-orange-600 dark:text-orange-400" />
              </div>
              <p className="font-medium text-sm mb-1">Analytics</p>
              <p className="text-xs text-muted-foreground">Monitor usage and performance</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Key Features */}
      <div>
        <h2 className="text-lg lg:text-xl font-semibold mb-4">Core Management Features</h2>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Server & Infrastructure */}
          <Card className="border-blue-200 dark:border-blue-800">
            <CardHeader>
              <CardTitle className="text-base lg:text-lg flex items-center gap-2">
                <Server className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                Server & Infrastructure
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-3">
                <div className="flex items-start gap-3">
                  <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="font-medium text-sm">Server Initialization</p>
                    <p className="text-xs text-muted-foreground">One-click setup of new MCP server instances</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="font-medium text-sm">Health Monitoring</p>
                    <p className="text-xs text-muted-foreground">Real-time status and performance metrics</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="font-medium text-sm">Configuration Management</p>
                    <p className="text-xs text-muted-foreground">Centralized server settings and environment variables</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="font-medium text-sm">Backup & Recovery</p>
                    <p className="text-xs text-muted-foreground">Automated backups and disaster recovery</p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Tools & Integrations */}
          <Card className="border-green-200 dark:border-green-800">
            <CardHeader>
              <CardTitle className="text-base lg:text-lg flex items-center gap-2">
                <Settings className="h-5 w-5 text-green-600 dark:text-green-400" />
                Tools & Integrations
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-3">
                <div className="flex items-start gap-3">
                  <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="font-medium text-sm">Tool Catalog</p>
                    <p className="text-xs text-muted-foreground">Browse and install from 50+ pre-built integrations</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="font-medium text-sm">Configuration Wizard</p>
                    <p className="text-xs text-muted-foreground">Step-by-step setup with validation and testing</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="font-medium text-sm">Credential Management</p>
                    <p className="text-xs text-muted-foreground">Secure storage and rotation of API keys</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="font-medium text-sm">Custom Tools</p>
                    <p className="text-xs text-muted-foreground">Upload and configure custom MCP server implementations</p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* User & Access Management */}
          <Card className="group relative overflow-hidden border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm hover:shadow-2xl hover:shadow-purple-500/10 transition-all duration-500 hover:scale-[1.02] dark:border-slate-700/60 h-full">
            <div className="absolute inset-0 bg-gradient-to-br from-purple-50/50 to-violet-50/20 dark:from-purple-950/20 dark:to-violet-950/10" />
            <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-purple-500/10 to-violet-500/10 rounded-full -mr-16 -mt-16 transition-all duration-500 group-hover:scale-150" />
            <CardHeader className="relative pb-4">
              <div className="flex items-center space-x-4 mb-4">
                <div className="h-12 w-12 rounded-xl bg-gradient-to-br from-purple-600 to-violet-600 flex items-center justify-center shadow-lg">
                  <Users className="h-6 w-6 text-white" />
                </div>
                <div>
                  <CardTitle className="text-xl font-bold text-slate-900 dark:text-slate-100">User & Access Management</CardTitle>
                </div>
              </div>
            </CardHeader>
            <CardContent className="relative space-y-6">
              <div className="space-y-4">
                <div className="flex items-start gap-3">
                  <div className="h-2 w-2 bg-purple-600 rounded-full flex-shrink-0 mt-2" />
                  <div>
                    <p className="font-medium text-sm text-slate-900 dark:text-slate-100">Role-Based Access</p>
                    <p className="text-sm text-slate-600 dark:text-slate-300">Owner, Admin, and Member permission levels</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <div className="h-2 w-2 bg-purple-600 rounded-full flex-shrink-0 mt-2" />
                  <div>
                    <p className="font-medium text-sm text-slate-900 dark:text-slate-100">Token Management</p>
                    <p className="text-sm text-slate-600 dark:text-slate-300">Generate and revoke access tokens for users and agents</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <div className="h-2 w-2 bg-purple-600 rounded-full flex-shrink-0 mt-2" />
                  <div>
                    <p className="font-medium text-sm text-slate-900 dark:text-slate-100">Multi-Tenancy</p>
                    <p className="text-sm text-slate-600 dark:text-slate-300">Organize users into teams and projects</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <div className="h-2 w-2 bg-purple-600 rounded-full flex-shrink-0 mt-2" />
                  <div>
                    <p className="font-medium text-sm text-slate-900 dark:text-slate-100">Activity Auditing</p>
                    <p className="text-sm text-slate-600 dark:text-slate-300">Track user actions and access patterns</p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Security & Monitoring */}
          <Card className="group relative overflow-hidden border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm hover:shadow-2xl hover:shadow-orange-500/10 transition-all duration-500 hover:scale-[1.02] dark:border-slate-700/60 h-full">
            <div className="absolute inset-0 bg-gradient-to-br from-orange-50/50 to-amber-50/20 dark:from-orange-950/20 dark:to-amber-950/10" />
            <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-orange-500/10 to-amber-500/10 rounded-full -mr-16 -mt-16 transition-all duration-500 group-hover:scale-150" />
            <CardHeader className="relative pb-4">
              <div className="flex items-center space-x-4 mb-4">
                <div className="h-12 w-12 rounded-xl bg-gradient-to-br from-orange-600 to-amber-600 flex items-center justify-center shadow-lg">
                  <Shield className="h-6 w-6 text-white" />
                </div>
                <div>
                  <CardTitle className="text-xl font-bold text-slate-900 dark:text-slate-100">Security & Monitoring</CardTitle>
                </div>
              </div>
            </CardHeader>
            <CardContent className="relative space-y-6">
              <div className="space-y-4">
                <div className="flex items-start gap-3">
                  <div className="h-2 w-2 bg-orange-600 rounded-full flex-shrink-0 mt-2" />
                  <div>
                    <p className="font-medium text-sm text-slate-900 dark:text-slate-100">Network Security</p>
                    <p className="text-sm text-slate-600 dark:text-slate-300">IP whitelisting and firewall configuration</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <div className="h-2 w-2 bg-orange-600 rounded-full flex-shrink-0 mt-2" />
                  <div>
                    <p className="font-medium text-sm text-slate-900 dark:text-slate-100">Usage Analytics</p>
                    <p className="text-sm text-slate-600 dark:text-slate-300">Detailed metrics on tool usage and performance</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <div className="h-2 w-2 bg-orange-600 rounded-full flex-shrink-0 mt-2" />
                  <div>
                    <p className="font-medium text-sm text-slate-900 dark:text-slate-100">Log Management</p>
                    <p className="text-sm text-slate-600 dark:text-slate-300">Centralized logging with search and filtering</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <div className="h-2 w-2 bg-orange-600 rounded-full flex-shrink-0 mt-2" />
                  <div>
                    <p className="font-medium text-sm text-slate-900 dark:text-slate-100">Alerts & Notifications</p>
                    <p className="text-sm text-slate-600 dark:text-slate-300">Automated alerts for issues and thresholds</p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Interface Walkthrough */}
      <Card className="border-2 border-slate-200 dark:border-slate-700">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Eye className="h-5 w-5 text-slate-600 dark:text-slate-400" />
            Console Interface Overview
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="bg-gradient-to-br from-slate-50 dark:from-slate-950 to-slate-100 dark:to-slate-900 rounded-lg p-6 border border-slate-200 dark:border-slate-700">
            <h3 className="text-base lg:text-lg font-semibold text-slate-700 dark:text-slate-300 mb-4 text-center">Main Dashboard Layout</h3>
            
            {/* Dashboard Layout */}
            <div className="bg-white dark:bg-slate-800 rounded-lg border-2 border-slate-300 dark:border-slate-600 p-4 space-y-4">
              {/* Header */}
              <div className="flex items-center justify-between p-3 bg-blue-50 dark:bg-blue-950/30 rounded border border-blue-200 dark:border-blue-800">
                <div className="flex items-center gap-2">
                  <Monitor className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                  <span className="font-medium text-sm">Kimbap.io Console</span>
                </div>
                <div className="flex items-center gap-2">
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Server Online</Badge>
                  <div className="w-8 h-8 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center">
                    <Crown className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                  </div>
                </div>
              </div>
              
              {/* Sidebar + Main Content */}
              <div className="grid grid-cols-4 gap-4">
                {/* Sidebar */}
                <div className="bg-slate-50 dark:bg-slate-800/50 rounded p-3 space-y-2">
                  <div className="text-xs font-medium text-slate-600 dark:text-slate-400">NAVIGATION</div>
                  <div className="space-y-1">
                    <div className="flex items-center gap-2 p-1 bg-blue-100 dark:bg-blue-900/30 rounded text-xs">
                      <BarChart className="h-3 w-3" />
                      <span>Dashboard</span>
                    </div>
                    <div className="flex items-center gap-2 p-1 text-xs">
                      <Settings className="h-3 w-3" />
                      <span>Tools</span>
                    </div>
                    <div className="flex items-center gap-2 p-1 text-xs">
                      <Users className="h-3 w-3" />
                      <span>Tokens</span>
                    </div>
                    <div className="flex items-center gap-2 p-1 text-xs">
                      <Shield className="h-3 w-3" />
                      <span>Security</span>
                    </div>
                  </div>
                </div>
                
                {/* Main Content Area */}
                <div className="col-span-3 space-y-3">
                  <div className="grid grid-cols-3 gap-2">
                    <div className="p-2 bg-green-50 dark:bg-green-950/30 rounded border text-center">
                      <div className="text-lg font-bold text-green-600 dark:text-green-400">12</div>
                      <div className="text-xs text-muted-foreground">Active Tools</div>
                    </div>
                    <div className="p-2 bg-blue-50 dark:bg-blue-950/30 rounded border text-center">
                      <div className="text-lg font-bold text-blue-600 dark:text-blue-400">8</div>
                      <div className="text-xs text-muted-foreground">Active Tokens</div>
                    </div>
                    <div className="p-2 bg-purple-50 dark:bg-purple-950/30 rounded border text-center">
                      <div className="text-lg font-bold text-purple-600 dark:text-purple-400">324</div>
                      <div className="text-xs text-muted-foreground">API Calls Today</div>
                    </div>
                  </div>
                  <div className="p-3 border rounded">
                    <div className="text-xs font-medium mb-2">Recent Activity</div>
                    <div className="space-y-1 text-xs text-muted-foreground">
                      <div>• GitHub tool used by john@company.com</div>
                      <div>• New access token generated for user</div>
                      <div>• Notion integration updated</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          
          {/* Key Interface Sections */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-3">
              <h4 className="text-sm font-semibold">Primary Navigation</h4>
              <div className="space-y-2">
                <div className="flex items-center gap-3 p-2 bg-slate-50 dark:bg-slate-800/50 rounded">
                  <BarChart className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                  <div>
                    <p className="text-sm font-medium">Dashboard</p>
                    <p className="text-xs text-muted-foreground">Overview and key metrics</p>
                  </div>
                </div>
                <div className="flex items-center gap-3 p-2 bg-slate-50 dark:bg-slate-800/50 rounded">
                  <Settings className="h-4 w-4 text-green-600 dark:text-green-400" />
                  <div>
                    <p className="text-sm font-medium">Tool Management</p>
                    <p className="text-xs text-muted-foreground">Configure and test integrations</p>
                  </div>
                </div>
                <div className="flex items-center gap-3 p-2 bg-slate-50 dark:bg-slate-800/50 rounded">
                  <Users className="h-4 w-4 text-purple-600 dark:text-purple-400" />
                  <div>
                    <p className="text-sm font-medium">Token Management</p>
                    <p className="text-xs text-muted-foreground">Invite and manage team access</p>
                  </div>
                </div>
              </div>
            </div>
            
            <div className="space-y-3">
              <h4 className="text-sm font-semibold">Advanced Features</h4>
              <div className="space-y-2">
                <div className="flex items-center gap-3 p-2 bg-slate-50 dark:bg-slate-800/50 rounded">
                  <Eye className="h-4 w-4 text-orange-600 dark:text-orange-400" />
                  <div>
                    <p className="text-sm font-medium">Usage Analytics</p>
                    <p className="text-xs text-muted-foreground">Detailed usage reports and trends</p>
                  </div>
                </div>
                <div className="flex items-center gap-3 p-2 bg-slate-50 dark:bg-slate-800/50 rounded">
                  <Lock className="h-4 w-4 text-red-600 dark:text-red-400" />
                  <div>
                    <p className="text-sm font-medium">Security Settings</p>
                    <p className="text-xs text-muted-foreground">Network access and policies</p>
                  </div>
                </div>
                <div className="flex items-center gap-3 p-2 bg-slate-50 dark:bg-slate-800/50 rounded">
                  <Database className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
                  <div>
                    <p className="text-sm font-medium">Server Control</p>
                    <p className="text-xs text-muted-foreground">Advanced server management</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Access & Requirements */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card className="border-amber-200 bg-amber-50 dark:bg-amber-950/30 dark:border-amber-800">
          <CardHeader>
            <CardTitle className="text-base lg:text-lg flex items-center gap-2">
              <Key className="h-5 w-5 text-amber-600 dark:text-amber-400" />
              Access Requirements
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-2">
              <div className="flex items-start gap-2">
                <Crown className="h-4 w-4 text-purple-600 dark:text-purple-400 mt-0.5" />
                <div>
                  <p className="font-medium text-sm">Owner Access</p>
                  <p className="text-xs text-muted-foreground">Full administrative control and server management</p>
                </div>
              </div>
              <div className="flex items-start gap-2">
                <Shield className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                <div>
                  <p className="font-medium text-sm">Admin Access</p>
                  <p className="text-xs text-muted-foreground">Tool management and token administration</p>
                </div>
              </div>
              <div className="flex items-start gap-2">
                <Eye className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                <div>
                  <p className="font-medium text-sm">View-Only Access</p>
                  <p className="text-xs text-muted-foreground">Read-only access to analytics and logs</p>
                </div>
              </div>
            </div>
            
            <Alert className="bg-white dark:bg-slate-800 border-amber-200 dark:border-amber-800">
              <Globe className="h-4 w-4" />
              <AlertDescription className="text-xs">
                <strong>Browser Access:</strong> Console works in any modern web browser with HTTPS support
              </AlertDescription>
            </Alert>
          </CardContent>
        </Card>

        <Card className="border-indigo-200 bg-indigo-50 dark:bg-indigo-950/30 dark:border-indigo-800">
          <CardHeader>
            <CardTitle className="text-base lg:text-lg flex items-center gap-2">
              <Zap className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
              Getting Started
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <ol className="space-y-2">
              <li className="flex items-start gap-2">
                <span className="w-5 h-5 bg-indigo-100 dark:bg-indigo-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">1</span>
                <p className="text-sm">Access Console via your server URL</p>
              </li>
              <li className="flex items-start gap-2">
                <span className="w-5 h-5 bg-indigo-100 dark:bg-indigo-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">2</span>
                <p className="text-sm">Enter master password or use owner token</p>
              </li>
              <li className="flex items-start gap-2">
                <span className="w-5 h-5 bg-indigo-100 dark:bg-indigo-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">3</span>
                <p className="text-sm">Initialize server if first time</p>
              </li>
              <li className="flex items-start gap-2">
                <span className="w-5 h-5 bg-indigo-100 dark:bg-indigo-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">4</span>
                <p className="text-sm">Start configuring tools and inviting team</p>
              </li>
            </ol>
            
            <div className="pt-3">
              <Button size="sm" className="w-full bg-indigo-600 hover:bg-indigo-700">
                <ArrowRight className="h-4 w-4 mr-2" />
                View Installation Guide
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

    </div>
  )
}