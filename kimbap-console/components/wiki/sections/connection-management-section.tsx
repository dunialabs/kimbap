import { Network, Shield, CheckCircle, AlertCircle, Settings, Monitor, Globe, Server, Key, Eye, Lock, Zap, ChevronRight, Copy, Play, Wifi, RefreshCw, X, Plus, Edit, ToggleLeft, ToggleRight, ChevronDown, ChevronUp, Search, Github, Database, Wrench } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function ConnectionManagementSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-cyan-500 to-blue-600 rounded-2xl mb-4">
          <Network className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">Managing Connections</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Manage multiple server connections, monitor connection health, troubleshoot issues, and optimize your MCP Desk configuration for the best experience.
        </p>
      </div>

      {/* Live Connection Status */}
      <Card className="border-2 border-blue-200 dark:border-blue-800 bg-gradient-to-br from-blue-50 to-cyan-50 dark:from-blue-950/30 dark:to-cyan-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-blue-800 dark:text-blue-300">
            <Monitor className="h-6 w-6" />
            Client Connection Status
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <Alert className="bg-blue-50 dark:bg-blue-950/30 border-blue-200 dark:border-blue-800">
            <Monitor className="h-4 w-4" />
            <AlertDescription>
              <strong>Real-time Monitoring:</strong> MCP Desk shows the live status of all your AI client connections, allowing you to monitor health, manage tools, and control access at a granular level.
            </AlertDescription>
          </Alert>
          
          {/* Claude Desktop Connection */}
          <div className="bg-white dark:bg-slate-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-orange-100 dark:bg-orange-900/30 rounded-lg flex items-center justify-center">
                  <Monitor className="h-5 w-5 text-orange-600 dark:text-orange-400" />
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <p className="font-semibold text-sm">Claude Desktop MCP</p>
                    <Badge className="bg-gray-100 dark:bg-gray-900/50 text-gray-600 dark:text-gray-400 text-xs">Checking...</Badge>
                    <Button size="sm" variant="destructive" className="h-6 text-xs px-2">
                      Disconnect
                    </Button>
                  </div>
                  <p className="text-xs text-muted-foreground">MCP Disabled/ 3 mcp server available</p>
                </div>
              </div>
              <ToggleRight className="h-6 w-6 text-blue-500 dark:text-blue-400" />
            </div>
            <Alert className="bg-amber-50 dark:bg-amber-950/30 border-amber-200 dark:border-amber-800">
              <RefreshCw className="h-4 w-4" />
              <AlertDescription className="text-xs">
                <strong>Restart Required</strong> - Configuration changes require restarting Claude Desktop to take effect.
              </AlertDescription>
            </Alert>
            <Button size="sm" variant="outline" className="mt-3 text-xs">
              Check
            </Button>
          </div>

          {/* Cursor MCP Connection */}
          <div className="bg-white dark:bg-slate-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center">
                  <Monitor className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                </div>
                <div>
                  <p className="font-semibold text-sm">Cursor MCP</p>
                  <p className="text-xs text-muted-foreground">2/3 servers are connected</p>
                </div>
              </div>
              <ToggleRight className="h-6 w-6 text-blue-500 dark:text-blue-400" />
            </div>
            
            {/* Staging Environment */}
            <div className="ml-4 border-l-2 border-gray-200 dark:border-gray-800 pl-4 space-y-4">
              <div className="flex items-center gap-2">
                <ChevronDown className="h-4 w-4 text-gray-600 dark:text-gray-400" />
                <span className="font-medium text-sm">Staging Environment</span>
                <div className="flex gap-1">
                  <div className="w-2 h-2 rounded-full bg-red-500"></div>
                  <div className="w-2 h-2 rounded-full bg-gray-400"></div>
                  <div className="w-2 h-2 rounded-full bg-green-500"></div>
                </div>
                <ToggleRight className="h-5 w-5 text-blue-500 dark:text-blue-400 ml-auto" />
              </div>
              
              {/* WebSearch Tool */}
              <div className="ml-4 space-y-3">
                <div className="flex items-center gap-2">
                  <CheckCircle className="h-4 w-4 text-blue-500 dark:text-blue-400" />
                  <Search className="h-4 w-4 text-gray-600 dark:text-gray-400" />
                  <span className="text-sm">WebSearch</span>
                </div>
                
                {/* Functions */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-medium text-gray-700 dark:text-gray-300">Functions - 8/8 enable</span>
                  </div>
                  <div className="space-y-1 max-h-32 overflow-y-auto">
                    {[1,2,3,4,5,6].map((i) => (
                      <div key={i} className="flex items-center gap-2 text-xs">
                        <CheckCircle className="h-3 w-3 text-blue-500 dark:text-blue-400" />
                        <span className="text-gray-600 dark:text-gray-400">Function name nameFunction name nameFunction name...</span>
                      </div>
                    ))}
                  </div>
                </div>
                
                {/* Data */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-medium text-gray-700 dark:text-gray-300">Data - 7/N enabled (N = number of files manager can access)</span>
                  </div>
                  <div className="space-y-1 max-h-32 overflow-y-auto">
                    {[1,2,3,4,5].map((i) => (
                      <div key={i} className="flex items-center gap-2 text-xs">
                        <CheckCircle className="h-3 w-3 text-blue-500 dark:text-blue-400" />
                        <span className="text-gray-600 dark:text-gray-400">Function name nameFunction name nameFunction name...</span>
                      </div>
                    ))}
                  </div>
                </div>
                
                {/* GitHub */}
                <div className="flex items-center gap-2">
                  <CheckCircle className="h-4 w-4 text-blue-500 dark:text-blue-400" />
                  <Github className="h-4 w-4 text-gray-600 dark:text-gray-400" />
                  <span className="text-sm">GitHub</span>
                </div>
              </div>
            </div>
            
            {/* Another Staging Environment (collapsed) */}
            <div className="ml-4 border-l-2 border-gray-200 dark:border-gray-800 pl-4 mt-4">
              <div className="flex items-center gap-2">
                <ChevronUp className="h-4 w-4 text-gray-600 dark:text-gray-400" />
                <span className="font-medium text-sm">Staging Environment</span>
                <div className="w-2 h-2 rounded-full bg-green-500"></div>
                <ToggleLeft className="h-5 w-5 text-gray-400 ml-auto" />
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Connection Management Features */}
      <Card className="border-2 border-slate-200 dark:border-slate-800 bg-gradient-to-br from-slate-50 to-gray-50 dark:from-slate-900/50 dark:to-gray-900/50">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-slate-800 dark:text-slate-200">
            <Settings className="h-6 w-6" />
            Connection Management Features
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-slate-700 dark:text-slate-300">
            MCP Desk provides comprehensive tools for managing connections, monitoring status, and controlling access to ensure optimal performance.
          </p>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-blue-200 dark:border-blue-800">
              <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Monitor className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              </div>
              <p className="font-medium text-sm mb-1">Real-time Status</p>
              <p className="text-xs text-muted-foreground">Live connection monitoring</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-purple-200 dark:border-purple-800">
              <div className="w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <RefreshCw className="h-5 w-5 text-purple-600 dark:text-purple-400" />
              </div>
              <p className="font-medium text-sm mb-1">Auto-Recovery</p>
              <p className="text-xs text-muted-foreground">Automatic reconnection</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-orange-200 dark:border-orange-800">
              <div className="w-10 h-10 bg-orange-100 dark:bg-orange-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Settings className="h-5 w-5 text-orange-600 dark:text-orange-400" />
              </div>
              <p className="font-medium text-sm mb-1">Configuration</p>
              <p className="text-xs text-muted-foreground">Flexible connection settings</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Connection Management Tabs */}
      <Tabs defaultValue="overview" className="w-full">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="overview" className="flex items-center gap-2 text-xs">
            <Eye className="h-4 w-4" />
            Overview
          </TabsTrigger>
          <TabsTrigger value="multiple" className="flex items-center gap-2 text-xs">
            <Server className="h-4 w-4" />
            Multi-Server
          </TabsTrigger>
          <TabsTrigger value="monitoring" className="flex items-center gap-2 text-xs">
            <Monitor className="h-4 w-4" />
            Monitoring
          </TabsTrigger>
          <TabsTrigger value="troubleshooting" className="flex items-center gap-2 text-xs">
            <AlertCircle className="h-4 w-4" />
            Troubleshooting
          </TabsTrigger>
        </TabsList>

        {/* Connection Overview */}
        <TabsContent value="overview" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Network className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                Current Connections
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-4">
                <h4 className="font-semibold">Active Server Connections</h4>
                <div className="space-y-3">
                  {/* Production Server */}
                  <div className="flex items-center justify-between p-4 border rounded-lg bg-green-50 dark:bg-green-950/30 border-green-200 dark:border-green-800">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-lg flex items-center justify-center">
                        <Server className="h-5 w-5 text-green-600 dark:text-green-400" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <p className="font-medium text-sm">Production Server</p>
                          <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Connected</Badge>
                        </div>
                        <p className="text-xs text-muted-foreground">https://mcp.company.com</p>
                        <p className="text-xs text-green-600 dark:text-green-400">12 tools available • Last sync: 2 minutes ago</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button size="sm" variant="outline" className="h-7 text-xs">
                        <Eye className="h-3 w-3 mr-1" />
                        View
                      </Button>
                      <Button size="sm" variant="outline" className="h-7 text-xs">
                        <Settings className="h-3 w-3 mr-1" />
                        Edit
                      </Button>
                    </div>
                  </div>

                  {/* Connected Server */}
                  <div className="flex items-center justify-between p-4 border rounded-lg bg-blue-50 dark:bg-blue-950/30 border-blue-200 dark:border-blue-800">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center">
                        <Server className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <p className="font-medium text-sm">Connected Server</p>
                          <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Connected</Badge>
                        </div>
                        <p className="text-xs text-muted-foreground">https://dev.company.com</p>
                        <p className="text-xs text-blue-600 dark:text-blue-400">8 tools available • Last sync: 5 minutes ago</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button size="sm" variant="outline" className="h-7 text-xs">
                        <Eye className="h-3 w-3 mr-1" />
                        View
                      </Button>
                      <Button size="sm" variant="outline" className="h-7 text-xs">
                        <Settings className="h-3 w-3 mr-1" />
                        Edit
                      </Button>
                    </div>
                  </div>

                  {/* Offline Server */}
                  <div className="flex items-center justify-between p-4 border rounded-lg bg-gray-50 dark:bg-gray-950/50 border-gray-200 dark:border-gray-800">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-gray-100 dark:bg-gray-900/50 rounded-lg flex items-center justify-center">
                        <Server className="h-5 w-5 text-gray-600 dark:text-gray-400" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <p className="font-medium text-sm">Partner Server</p>
                          <Badge variant="outline" className="text-xs">Offline</Badge>
                        </div>
                        <p className="text-xs text-muted-foreground">https://partner.example.com</p>
                        <p className="text-xs text-red-600 dark:text-red-400">Connection timeout • Last seen: 2 hours ago</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button size="sm" variant="outline" className="h-7 text-xs">
                        <RefreshCw className="h-3 w-3 mr-1" />
                        Retry
                      </Button>
                      <Button size="sm" variant="outline" className="h-7 text-xs">
                        <Settings className="h-3 w-3 mr-1" />
                        Edit
                      </Button>
                    </div>
                  </div>
                </div>
              </div>

              <div className="flex gap-3">
                <Button className="flex-1">
                  <Plus className="h-4 w-4 mr-2" />
                  Add New Server
                </Button>
                <Button variant="outline" className="flex-1">
                  <RefreshCw className="h-4 w-4 mr-2" />
                  Refresh All
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Multiple Servers */}
        <TabsContent value="multiple" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Server className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                Managing Multiple Server Connections
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="border-purple-200 bg-purple-50 dark:bg-purple-950/30 dark:border-purple-800">
                <Server className="h-4 w-4" />
                <AlertDescription>
                  <strong>Multi-Server Support:</strong> MCP Desk can maintain connections to multiple MCP servers simultaneously, 
                  allowing you to access tools from different organizations, environments, or projects.
                </AlertDescription>
              </Alert>

              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Use Cases for Multiple Connections</h4>
                  <div className="space-y-3">
                    <div className="p-3 border rounded-lg">
                      <div className="flex items-center gap-2 mb-1">
                        <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                        <span className="font-medium text-sm">Multi-Organization Access</span>
                      </div>
                      <p className="text-xs text-muted-foreground">
                        Connect to different companies or clients while maintaining separate tool access
                      </p>
                    </div>
                    
                    <div className="p-3 border rounded-lg">
                      <div className="flex items-center gap-2 mb-1">
                        <CheckCircle className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                        <span className="font-medium text-sm">Environment Separation</span>
                      </div>
                      <p className="text-xs text-muted-foreground">
                        Separate development, staging, and production environments
                      </p>
                    </div>
                    
                    <div className="p-3 border rounded-lg">
                      <div className="flex items-center gap-2 mb-1">
                        <CheckCircle className="h-4 w-4 text-purple-600 dark:text-purple-400" />
                        <span className="font-medium text-sm">Project-Based Tools</span>
                      </div>
                      <p className="text-xs text-muted-foreground">
                        Access different tool sets for different projects or teams
                      </p>
                    </div>
                    
                    <div className="p-3 border rounded-lg">
                      <div className="flex items-center gap-2 mb-1">
                        <CheckCircle className="h-4 w-4 text-orange-600 dark:text-orange-400" />
                        <span className="font-medium text-sm">Backup Redundancy</span>
                      </div>
                      <p className="text-xs text-muted-foreground">
                        Maintain backup connections for critical workflows
                      </p>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Connection Priority & Routing</h4>
                  <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-4 border dark:border-slate-700">
                    <div className="space-y-3">
                      <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                        <div className="flex items-center gap-2">
                          <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                          <span className="text-sm font-medium">Primary (Production)</span>
                        </div>
                        <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Priority 1</Badge>
                      </div>
                      
                      <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                        <div className="flex items-center gap-2">
                          <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                          <span className="text-sm font-medium">Secondary (Development)</span>
                        </div>
                        <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Priority 2</Badge>
                      </div>
                      
                      <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                        <div className="flex items-center gap-2">
                          <div className="w-2 h-2 bg-orange-500 rounded-full"></div>
                          <span className="text-sm font-medium">Backup (Partner)</span>
                        </div>
                        <Badge className="bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 text-xs">Priority 3</Badge>
                      </div>
                    </div>
                  </div>
                  
                  <Alert className="bg-blue-50 dark:bg-blue-950/30 border-blue-200 dark:border-blue-800">
                    <Zap className="h-4 w-4" />
                    <AlertDescription className="text-xs">
                      <strong>Smart Routing:</strong> MCP Desk automatically routes tool requests to the appropriate server based on tool availability and connection priority.
                    </AlertDescription>
                  </Alert>
                </div>
              </div>

              <div className="space-y-4">
                <h4 className="font-semibold">Adding a New Server Connection</h4>
                <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-4 border dark:border-slate-700">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="space-y-3">
                      <div>
                        <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Connection Name</label>
                        <div className="mt-1 p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded text-sm">
                          <input type="text" placeholder="e.g., Staging Environment" className="w-full bg-transparent border-none outline-none text-sm" />
                        </div>
                      </div>
                      <div>
                        <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Server URL</label>
                        <div className="mt-1 p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded text-sm">
                          <input type="text" placeholder="https://staging.company.com" className="w-full bg-transparent border-none outline-none text-sm font-mono" />
                        </div>
                      </div>
                      <div>
                        <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Access Token</label>
                        <div className="mt-1 p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded text-sm">
                          <input type="password" placeholder="Enter your access token" className="w-full bg-transparent border-none outline-none text-sm font-mono" />
                        </div>
                      </div>
                    </div>
                    
                    <div className="space-y-3">
                      <div>
                        <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Priority Level</label>
                        <div className="mt-1 p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded text-sm">
                          <select className="w-full bg-transparent border-none outline-none text-sm">
                            <option>High Priority</option>
                            <option>Medium Priority</option>
                            <option>Low Priority</option>
                          </select>
                        </div>
                      </div>
                      <div>
                        <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Connection Settings</label>
                        <div className="mt-1 space-y-2">
                          <div className="flex items-center gap-2">
                            <input type="checkbox" className="w-3 h-3" defaultChecked />
                            <span className="text-xs">Auto-connect on startup</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <input type="checkbox" className="w-3 h-3" defaultChecked />
                            <span className="text-xs">Enable auto-reconnect</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <input type="checkbox" className="w-3 h-3" />
                            <span className="text-xs">Use as backup only</span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                  
                  <div className="flex gap-2 mt-4">
                    <Button size="sm" className="flex-1">
                      <CheckCircle className="h-4 w-4 mr-2" />
                      Test & Save
                    </Button>
                    <Button size="sm" variant="outline" className="flex-1">
                      <X className="h-4 w-4 mr-2" />
                      Cancel
                    </Button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Health Monitoring */}
        <TabsContent value="monitoring" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Monitor className="h-5 w-5 text-green-600 dark:text-green-400" />
                Connection Health Monitoring
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
                <Eye className="h-4 w-4" />
                <AlertDescription>
                  <strong>Real-Time Monitoring:</strong> MCP Desk continuously monitors all server connections and provides 
                  detailed health metrics to ensure optimal performance.
                </AlertDescription>
              </Alert>

              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Connection Metrics</h4>
                  <div className="space-y-3">
                    <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
                      <CardContent className="p-4">
                        <div className="flex items-center justify-between mb-2">
                          <span className="font-medium text-sm">Response Time</span>
                          <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Excellent</Badge>
                        </div>
                        <div className="flex items-center gap-2">
                          <div className="flex-1 bg-green-200 rounded-full h-2">
                            <div className="bg-green-600 h-2 rounded-full" style={{width: '85%'}}></div>
                          </div>
                          <span className="text-xs text-green-700 dark:text-green-300">142ms avg</span>
                        </div>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
                      <CardContent className="p-4">
                        <div className="flex items-center justify-between mb-2">
                          <span className="font-medium text-sm">Success Rate</span>
                          <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">99.8%</Badge>
                        </div>
                        <div className="flex items-center gap-2">
                          <div className="flex-1 bg-blue-200 rounded-full h-2">
                            <div className="bg-blue-600 h-2 rounded-full" style={{width: '99%'}}></div>
                          </div>
                          <span className="text-xs text-blue-700 dark:text-blue-300">2,847/2,853 requests</span>
                        </div>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-purple-200 bg-purple-50 dark:bg-purple-950/30 dark:border-purple-800">
                      <CardContent className="p-4">
                        <div className="flex items-center justify-between mb-2">
                          <span className="font-medium text-sm">Tool Availability</span>
                          <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 text-xs">12/12 Online</Badge>
                        </div>
                        <div className="flex items-center gap-2">
                          <div className="flex-1 bg-purple-200 rounded-full h-2">
                            <div className="bg-purple-600 h-2 rounded-full" style={{width: '100%'}}></div>
                          </div>
                          <span className="text-xs text-purple-700 dark:text-purple-300">All tools responsive</span>
                        </div>
                      </CardContent>
                    </Card>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Connection Events</h4>
                  <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-4 border max-h-64 overflow-y-auto">
                    <div className="space-y-2">
                      <div className="flex items-start gap-2 p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                        <div className="w-2 h-2 bg-green-500 rounded-full mt-1.5"></div>
                        <div className="flex-1">
                          <p className="text-xs font-medium">Connection established</p>
                          <p className="text-xs text-muted-foreground">Production Server • 2 minutes ago</p>
                        </div>
                      </div>
                      
                      <div className="flex items-start gap-2 p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                        <div className="w-2 h-2 bg-blue-500 rounded-full mt-1.5"></div>
                        <div className="flex-1">
                          <p className="text-xs font-medium">Tool sync completed</p>
                          <p className="text-xs text-muted-foreground">Connected Server • 5 minutes ago</p>
                        </div>
                      </div>
                      
                      <div className="flex items-start gap-2 p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                        <div className="w-2 h-2 bg-yellow-500 rounded-full mt-1.5"></div>
                        <div className="flex-1">
                          <p className="text-xs font-medium">Slow response detected</p>
                          <p className="text-xs text-muted-foreground">Production Server • 8 minutes ago</p>
                        </div>
                      </div>
                      
                      <div className="flex items-start gap-2 p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                        <div className="w-2 h-2 bg-red-500 rounded-full mt-1.5"></div>
                        <div className="flex-1">
                          <p className="text-xs font-medium">Connection timeout</p>
                          <p className="text-xs text-muted-foreground">Partner Server • 2 hours ago</p>
                        </div>
                      </div>
                      
                      <div className="flex items-start gap-2 p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                        <div className="w-2 h-2 bg-purple-500 rounded-full mt-1.5"></div>
                        <div className="flex-1">
                          <p className="text-xs font-medium">New tools discovered</p>
                          <p className="text-xs text-muted-foreground">Production Server • 1 day ago</p>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Card className="text-center">
                  <CardContent className="p-4">
                    <div className="text-2xl font-bold text-green-600 dark:text-green-400 mb-1">99.2%</div>
                    <p className="text-xs text-muted-foreground">Uptime (30 days)</p>
                  </CardContent>
                </Card>
                
                <Card className="text-center">
                  <CardContent className="p-4">
                    <div className="text-2xl font-bold text-blue-600 dark:text-blue-400 mb-1">156ms</div>
                    <p className="text-xs text-muted-foreground">Avg Response Time</p>
                  </CardContent>
                </Card>
                
                <Card className="text-center">
                  <CardContent className="p-4">
                    <div className="text-2xl font-bold text-purple-600 dark:text-purple-400 mb-1">3,247</div>
                    <p className="text-xs text-muted-foreground">Tool Requests Today</p>
                  </CardContent>
                </Card>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Troubleshooting */}
        <TabsContent value="troubleshooting" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
                Connection Troubleshooting
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="border-red-200 bg-red-50 dark:bg-red-950/30 dark:border-red-800">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  <strong>Troubleshooting Guide:</strong> Common connection issues and their solutions. 
                  Most problems can be resolved quickly with these diagnostic steps.
                </AlertDescription>
              </Alert>

              <div className="space-y-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Common Connection Issues</h4>
                  
                  <Card className="border-red-200 dark:border-red-800">
                    <CardHeader className="pb-3">
                      <CardTitle className="text-base flex items-center gap-2">
                        <AlertCircle className="h-4 w-4 text-red-600 dark:text-red-400" />
                        Connection Timeout
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <div className="bg-red-50 dark:bg-red-950/30 rounded p-3 border border-red-200 dark:border-red-800">
                        <p className="text-sm font-medium text-red-900 dark:text-red-200 mb-2">Symptoms:</p>
                        <ul className="text-sm text-red-800 dark:text-red-300 space-y-1">
                          <li>• Server shows as "Connecting..." indefinitely</li>
                          <li>• "Connection timeout" error messages</li>
                          <li>• Tools become unavailable suddenly</li>
                        </ul>
                      </div>
                      
                      <div className="bg-green-50 dark:bg-green-950/30 rounded p-3 border border-green-200 dark:border-green-800">
                        <p className="text-sm font-medium text-green-900 dark:text-green-200 mb-2">Solutions:</p>
                        <ol className="text-sm text-green-800 dark:text-green-300 space-y-1">
                          <li>1. Check your internet connection</li>
                          <li>2. Verify the server URL is correct</li>
                          <li>3. Ensure the server is running and accessible</li>
                          <li>4. Check firewall settings (ports 80, 443)</li>
                          <li>5. Try connecting from a different network</li>
                        </ol>
                      </div>
                    </CardContent>
                  </Card>
                  
                  <Card className="border-orange-200 dark:border-orange-800">
                    <CardHeader className="pb-3">
                      <CardTitle className="text-base flex items-center gap-2">
                        <Key className="h-4 w-4 text-orange-600 dark:text-orange-400" />
                        Authentication Failed
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <div className="bg-orange-50 dark:bg-orange-950/30 rounded p-3 border border-orange-200 dark:border-orange-800">
                        <p className="text-sm font-medium text-orange-900 dark:text-orange-200 mb-2">Symptoms:</p>
                        <ul className="text-sm text-orange-800 dark:text-orange-300 space-y-1">
                          <li>• "Invalid token" or "Authentication failed" errors</li>
                          <li>• Connection established but no tools visible</li>
                          <li>• Intermittent access to tools</li>
                        </ul>
                      </div>
                      
                      <div className="bg-green-50 dark:bg-green-950/30 rounded p-3 border border-green-200 dark:border-green-800">
                        <p className="text-sm font-medium text-green-900 dark:text-green-200 mb-2">Solutions:</p>
                        <ol className="text-sm text-green-800 dark:text-green-300 space-y-1">
                          <li>1. Verify your access token is correct</li>
                          <li>2. Check if your token has expired</li>
                          <li>3. Contact administrator for a new token</li>
                          <li>4. Ensure token permissions are sufficient</li>
                          <li>5. Try logging out and back in</li>
                        </ol>
                      </div>
                    </CardContent>
                  </Card>
                  
                  <Card className="border-blue-200 dark:border-blue-800">
                    <CardHeader className="pb-3">
                      <CardTitle className="text-base flex items-center gap-2">
                        <Shield className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                        SSL Certificate Issues
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <div className="bg-blue-50 dark:bg-blue-950/30 rounded p-3 border border-blue-200 dark:border-blue-800">
                        <p className="text-sm font-medium text-blue-900 dark:text-blue-200 mb-2">Symptoms:</p>
                        <ul className="text-sm text-blue-800 dark:text-blue-300 space-y-1">
                          <li>• "SSL certificate error" warnings</li>
                          <li>• "Insecure connection" notifications</li>
                          <li>• Browser security warnings</li>
                        </ul>
                      </div>
                      
                      <div className="bg-green-50 dark:bg-green-950/30 rounded p-3 border border-green-200 dark:border-green-800">
                        <p className="text-sm font-medium text-green-900 dark:text-green-200 mb-2">Solutions:</p>
                        <ol className="text-sm text-green-800 dark:text-green-300 space-y-1">
                          <li>1. Verify the server URL uses HTTPS</li>
                          <li>2. Check if certificate is valid and not expired</li>
                          <li>3. Contact IT support for certificate renewal</li>
                          <li>4. Try accessing from a different device</li>
                          <li>5. Clear browser cache and cookies</li>
                        </ol>
                      </div>
                    </CardContent>
                  </Card>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Diagnostic Tools</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <Card className="border-indigo-200 bg-indigo-50 dark:bg-indigo-950/30 dark:border-indigo-800">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-2">
                          <Monitor className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
                          <span className="font-medium text-sm">Connection Test</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          Test connectivity to a specific server
                        </p>
                        <Button size="sm" className="w-full">
                          <Play className="h-3 w-3 mr-1" />
                          Run Test
                        </Button>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-cyan-200 bg-cyan-50 dark:bg-cyan-950/30 dark:border-cyan-800">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-2">
                          <Eye className="h-4 w-4 text-cyan-600 dark:text-cyan-400" />
                          <span className="font-medium text-sm">Network Diagnostics</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          Check network paths and latency
                        </p>
                        <Button size="sm" variant="outline" className="w-full">
                          <RefreshCw className="h-3 w-3 mr-1" />
                          Diagnose
                        </Button>
                      </CardContent>
                    </Card>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Getting Help</h4>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                    <Button variant="outline" className="flex-1">
                      <Copy className="h-4 w-4 mr-2" />
                      Export Logs
                    </Button>
                    <Button variant="outline" className="flex-1">
                      <Monitor className="h-4 w-4 mr-2" />
                      System Info
                    </Button>
                    <Button variant="outline" className="flex-1">
                      <Globe className="h-4 w-4 mr-2" />
                      Contact Support
                    </Button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Best Practices */}
      <Card className="bg-gradient-to-r from-cyan-50 dark:from-cyan-950 to-blue-50 dark:to-blue-950 border-cyan-200 dark:border-cyan-800">
        <CardHeader>
          <CardTitle className="text-cyan-800 dark:text-cyan-300">Connection Management Best Practices</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-3">
              <h4 className="font-semibold">Performance Optimization</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-cyan-600 dark:text-cyan-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Prioritize Active Connections</p>
                    <p className="text-xs text-muted-foreground">Set frequently used servers to high priority</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-cyan-600 dark:text-cyan-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Monitor Connection Health</p>
                    <p className="text-xs text-muted-foreground">Regularly check metrics and resolve issues</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-cyan-600 dark:text-cyan-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Clean Up Unused Connections</p>
                    <p className="text-xs text-muted-foreground">Remove servers you no longer need</p>
                  </div>
                </li>
              </ul>
            </div>
            
            <div className="space-y-3">
              <h4 className="font-semibold">Security & Maintenance</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-cyan-600 dark:text-cyan-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Rotate Access Tokens</p>
                    <p className="text-xs text-muted-foreground">Update tokens regularly for security</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-cyan-600 dark:text-cyan-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Use Secure Connections</p>
                    <p className="text-xs text-muted-foreground">Always connect via HTTPS with valid certificates</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-cyan-600 dark:text-cyan-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Review Permissions</p>
                    <p className="text-xs text-muted-foreground">Ensure you have appropriate tool access</p>
                  </div>
                </li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
