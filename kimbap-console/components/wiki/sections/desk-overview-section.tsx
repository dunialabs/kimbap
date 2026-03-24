import { Monitor, Download, Settings, Users, Activity, Bell, RefreshCw, Upload, Search, Filter, Calendar, Crown, UserCheck, User, Clock, Zap, Plus, Copy, Terminal, Database, Code, CheckCircle, XCircle, AlertTriangle, Shield, Key, Wifi, Globe, Server, Network, Laptop, Smartphone, Lock, Eye, Fingerprint, ArrowRight } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function DeskOverviewSection() {
  return (
    <div className="space-y-8 lg:space-y-12">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-purple-500 to-violet-600 rounded-2xl mb-4 shadow-lg shadow-purple-500/25">
          <Monitor className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-2xl lg:text-3xl font-bold mb-2 text-slate-900 dark:text-slate-100">MCP Desk Overview</h1>
        <p className="text-base lg:text-lg text-slate-600 dark:text-slate-400 max-w-2xl mx-auto">
          MCP Desk is the desktop client that enables seamless integration between AI assistants like Claude 
          and your organization's MCP servers. Install once, connect everywhere.
        </p>
      </div>

      {/* What is MCP Desk */}
      <Card className="border-slate-200 dark:border-slate-800/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-lg hover:shadow-xl transition-all duration-300">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-slate-900 dark:text-slate-100">
            <div className="w-8 h-8 bg-gradient-to-br from-purple-500 to-violet-600 rounded-lg flex items-center justify-center shadow-md">
              <Monitor className="h-4 w-4 text-white" />
            </div>
            What is MCP Desk?
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="bg-gradient-to-br from-purple-50 to-violet-100 dark:from-purple-950/30 dark:to-violet-900/30 rounded-lg p-6 border border-purple-200 dark:border-purple-800">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <div className="text-center">
                <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-blue-600 rounded-2xl flex items-center justify-center mx-auto mb-3 shadow-lg shadow-blue-500/25">
                  <Laptop className="h-6 w-6 text-white" />
                </div>
                <h3 className="font-semibold text-slate-900 dark:text-slate-100 mb-2">Desktop Client</h3>
                <p className="text-sm text-slate-700 dark:text-slate-300">Native application for Windows, macOS, and Linux</p>
              </div>
              <div className="text-center">
                <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-purple-600 rounded-2xl flex items-center justify-center mx-auto mb-3 shadow-lg shadow-purple-500/25">
                  <Network className="h-6 w-6 text-white" />
                </div>
                <h3 className="font-semibold text-slate-900 dark:text-slate-100 mb-2">Bridge Technology</h3>
                <p className="text-sm text-slate-700 dark:text-slate-300">Connects AI assistants to your MCP servers securely</p>
              </div>
              <div className="text-center">
                <div className="w-12 h-12 bg-gradient-to-br from-violet-500 to-violet-600 rounded-2xl flex items-center justify-center mx-auto mb-3 shadow-lg shadow-violet-500/25">
                  <Zap className="h-6 w-6 text-white" />
                </div>
                <h3 className="font-semibold text-slate-900 dark:text-slate-100 mb-2">Zero Configuration</h3>
                <p className="text-sm text-slate-700 dark:text-slate-300">Automatic discovery and connection management</p>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="space-y-4">
              <h4 className="font-semibold">Key Features</h4>
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-sm">
                  <div className="w-4 h-4 bg-gradient-to-br from-green-500 to-green-600 rounded-full flex items-center justify-center">
                    <span className="w-1.5 h-1.5 bg-white rounded-full"></span>
                  </div>
                  <span>One-click installation and setup</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <div className="w-4 h-4 bg-gradient-to-br from-green-500 to-green-600 rounded-full flex items-center justify-center">
                    <span className="w-1.5 h-1.5 bg-white rounded-full"></span>
                  </div>
                  <span>Automatic server discovery</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <div className="w-4 h-4 bg-gradient-to-br from-green-500 to-green-600 rounded-full flex items-center justify-center">
                    <span className="w-1.5 h-1.5 bg-white rounded-full"></span>
                  </div>
                  <span>Secure token-based authentication</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <div className="w-4 h-4 bg-gradient-to-br from-green-500 to-green-600 rounded-full flex items-center justify-center">
                    <span className="w-1.5 h-1.5 bg-white rounded-full"></span>
                  </div>
                  <span>Real-time connection monitoring</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <div className="w-4 h-4 bg-gradient-to-br from-green-500 to-green-600 rounded-full flex items-center justify-center">
                    <span className="w-1.5 h-1.5 bg-white rounded-full"></span>
                  </div>
                  <span>Background operation (system tray)</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <div className="w-4 h-4 bg-gradient-to-br from-green-500 to-green-600 rounded-full flex items-center justify-center">
                    <span className="w-1.5 h-1.5 bg-white rounded-full"></span>
                  </div>
                  <span>Cross-platform compatibility</span>
                </div>
              </div>
            </div>
            
            <div className="space-y-4">
              <h4 className="font-semibold">Supported Platforms</h4>
              <div className="space-y-3">
                <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded-lg">
                  <div className="flex items-center gap-2">
                    <Laptop className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                    <span className="text-sm font-medium">Windows 10/11</span>
                  </div>
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300">Supported</Badge>
                </div>
                <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded-lg">
                  <div className="flex items-center gap-2">
                    <Laptop className="h-4 w-4 text-gray-600 dark:text-gray-400" />
                    <span className="text-sm font-medium">macOS 11+</span>
                  </div>
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300">Supported</Badge>
                </div>
                <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded-lg">
                  <div className="flex items-center gap-2">
                    <Terminal className="h-4 w-4 text-orange-600 dark:text-orange-400" />
                    <span className="text-sm font-medium">Linux (Ubuntu/RHEL)</span>
                  </div>
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300">Supported</Badge>
                </div>
              </div>
            </div>
          </div>

          <Alert className="bg-blue-50 dark:bg-blue-950/30 border-blue-200 dark:border-blue-800">
            <Monitor className="h-4 w-4" />
            <AlertDescription className="text-sm">
              <strong>Getting Started:</strong> MCP Desk automatically detects and connects to MCP servers on your network. 
              Simply install the application and add your access token to get started.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>

      {/* Architecture Overview */}
      <Card className="border-slate-200 dark:border-slate-800/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-lg hover:shadow-xl transition-all duration-300">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-slate-900 dark:text-slate-100">
            <div className="w-8 h-8 bg-gradient-to-br from-slate-500 to-slate-600 rounded-lg flex items-center justify-center shadow-md">
              <Network className="h-4 w-4 text-white" />
            </div>
            How MCP Desk Works
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Architecture Diagram Placeholder */}
          <div className="bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-800 dark:to-slate-900 rounded-lg p-8 border border-slate-200 dark:border-slate-700">
            <div className="text-center space-y-4">
              <p className="text-lg font-semibold text-slate-700 dark:text-slate-300">MCP Desk Architecture</p>
              <div className="bg-white dark:bg-slate-800 rounded-lg p-6 border-2 border-dashed border-slate-300 dark:border-slate-600">
                <p className="text-muted-foreground mb-2">[Architecture Diagram Placeholder]</p>
                <p className="text-sm text-muted-foreground">
                  Add diagram showing: AI Assistant ↔ MCP Desk ↔ MCP Server ↔ External Tools/APIs
                </p>
              </div>
              <Button variant="outline" size="sm">
                <Terminal className="h-4 w-4 mr-2" />
                View Technical Documentation
              </Button>
            </div>
          </div>

          {/* Complete Setup Flow */}
          <div className="space-y-4">
            <h4 className="font-semibold text-center">Complete Setup Workflow</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="text-center p-4 bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded-lg">
                <div className="w-10 h-10 bg-red-600 rounded-full flex items-center justify-center mx-auto mb-3 text-white font-bold">1</div>
                <Lock className="h-5 w-5 mx-auto mb-2 text-red-600 dark:text-red-400" />
                <h4 className="font-semibold mb-2 text-red-800 dark:text-red-300">Set Master Password</h4>
                <p className="text-xs text-red-700 dark:text-red-300">Create a master password to protect all your connections and enable biometric authentication</p>
              </div>
              <div className="text-center p-4 bg-orange-50 dark:bg-orange-950/30 border border-orange-200 dark:border-orange-800 rounded-lg">
                <div className="w-10 h-10 bg-orange-600 rounded-full flex items-center justify-center mx-auto mb-3 text-white font-bold">2</div>
                <Clock className="h-5 w-5 mx-auto mb-2 text-orange-600 dark:text-orange-400" />
                <h4 className="font-semibold mb-2 text-orange-800 dark:text-orange-300">Set Auto-Lock Timer</h4>
                <p className="text-xs text-orange-700 dark:text-orange-300">Configure how long MCP Desk stays unlocked before requiring password re-entry</p>
              </div>
              <div className="text-center p-4 bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 rounded-lg">
                <div className="w-10 h-10 bg-blue-600 rounded-full flex items-center justify-center mx-auto mb-3 text-white font-bold">3</div>
                <Server className="h-5 w-5 mx-auto mb-2 text-blue-600 dark:text-blue-400" />
                <h4 className="font-semibold mb-2 text-blue-800 dark:text-blue-300">Configure MCP Server</h4>
                <p className="text-xs text-blue-700 dark:text-blue-300">Add Kimbap MCP Console servers or other MCP-compatible servers using access tokens</p>
              </div>
              <div className="text-center p-4 bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded-lg">
                <div className="w-10 h-10 bg-green-600 rounded-full flex items-center justify-center mx-auto mb-3 text-white font-bold">4</div>
                <RefreshCw className="h-5 w-5 mx-auto mb-2 text-green-600 dark:text-green-400" />
                <h4 className="font-semibold mb-2 text-green-800 dark:text-green-300">Restart MCP Client</h4>
                <p className="text-xs text-green-700 dark:text-green-300">Restart your AI client (Claude Desktop, Cursor) to apply the new MCP configuration</p>
              </div>
            </div>
            
            {/* Additional Security Features */}
            <div className="mt-6 p-4 bg-gradient-to-r from-purple-50 to-indigo-50 dark:from-purple-950/30 dark:to-indigo-950/30 border border-purple-200 dark:border-purple-800 rounded-lg">
              <h5 className="font-semibold text-purple-800 dark:text-purple-300 mb-3 flex items-center gap-2">
                <Shield className="h-4 w-4" />
                Advanced Security Features
              </h5>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="flex items-start gap-3">
                  <Fingerprint className="h-5 w-5 text-purple-600 dark:text-purple-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium text-purple-800 dark:text-purple-300">Biometric Authentication</p>
                    <p className="text-xs text-purple-600 dark:text-purple-400">Support for fingerprint, Face ID, and Windows Hello</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <Eye className="h-5 w-5 text-purple-600 dark:text-purple-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium text-purple-800 dark:text-purple-300">Granular Controls</p>
                    <p className="text-xs text-purple-600 dark:text-purple-400">Per-client, server, tool, and data access switches</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <Lock className="h-5 w-5 text-purple-600 dark:text-purple-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium text-purple-800 dark:text-purple-300">Session Security</p>
                    <p className="text-xs text-purple-600 dark:text-purple-400">Automatic locking and secure credential storage</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Installation Guide */}
      <Card className="border-slate-200 dark:border-slate-800/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-lg hover:shadow-xl transition-all duration-300">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-slate-900 dark:text-slate-100">
            <div className="w-8 h-8 bg-gradient-to-br from-green-500 to-emerald-600 rounded-lg flex items-center justify-center shadow-md">
              <Download className="h-4 w-4 text-white" />
            </div>
            Quick Installation Guide
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <Tabs defaultValue="windows" className="w-full">
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="windows">Windows</TabsTrigger>
              <TabsTrigger value="macos">macOS</TabsTrigger>
              <TabsTrigger value="linux">Linux</TabsTrigger>
            </TabsList>
            
            <TabsContent value="windows" className="space-y-4">
              <div className="space-y-3">
                <h4 className="font-semibold">System Requirements</h4>
                <div className="p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                  <ul className="text-sm space-y-1">
                    <li>• Windows 10 version 1903 or later</li>
                    <li>• Windows 11 (all versions)</li>
                    <li>• 4GB RAM minimum, 8GB recommended</li>
                    <li>• 500MB available disk space</li>
                    <li>• Internet connection for initial setup</li>
                  </ul>
                </div>
                
                <h4 className="font-semibold">Installation Steps</h4>
                <div className="space-y-2">
                  <div className="flex items-start gap-3 p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                    <div className="w-6 h-6 bg-green-600 rounded-full flex items-center justify-center text-white text-xs font-bold">1</div>
                    <div>
                      <p className="text-sm font-medium">Download MCP Desk</p>
                      <p className="text-xs text-muted-foreground">Download the .exe installer from the official website</p>
                      <Button size="sm" className="mt-2 bg-green-600 hover:bg-green-700">
                        <Download className="h-3 w-3 mr-1" />
                        Download for Windows
                      </Button>
                    </div>
                  </div>
                  
                  <div className="flex items-start gap-3 p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                    <div className="w-6 h-6 bg-green-600 rounded-full flex items-center justify-center text-white text-xs font-bold">2</div>
                    <div>
                      <p className="text-sm font-medium">Run the Installer</p>
                      <p className="text-xs text-muted-foreground">Double-click the downloaded .exe file and follow the installation wizard</p>
                    </div>
                  </div>
                  
                  <div className="flex items-start gap-3 p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                    <div className="w-6 h-6 bg-green-600 rounded-full flex items-center justify-center text-white text-xs font-bold">3</div>
                    <div>
                      <p className="text-sm font-medium">Launch Application</p>
                      <p className="text-xs text-muted-foreground">MCP Desk will start automatically and appear in your system tray</p>
                    </div>
                  </div>
                </div>
              </div>
            </TabsContent>
            
            <TabsContent value="macos" className="space-y-4">
              <div className="space-y-3">
                <h4 className="font-semibold">System Requirements</h4>
                <div className="p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                  <ul className="text-sm space-y-1">
                    <li>• macOS 11 (Big Sur) or later</li>
                    <li>• Intel or Apple Silicon processor</li>
                    <li>• 4GB RAM minimum, 8GB recommended</li>
                    <li>• 500MB available disk space</li>
                    <li>• Internet connection for initial setup</li>
                  </ul>
                </div>
                
                <h4 className="font-semibold">Installation Steps</h4>
                <div className="space-y-2">
                  <div className="flex items-start gap-3 p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                    <div className="w-6 h-6 bg-green-600 rounded-full flex items-center justify-center text-white text-xs font-bold">1</div>
                    <div>
                      <p className="text-sm font-medium">Download MCP Desk</p>
                      <p className="text-xs text-muted-foreground">Download the .dmg file for your processor type (Intel/Apple Silicon)</p>
                      <Button size="sm" className="mt-2 bg-green-600 hover:bg-green-700">
                        <Download className="h-3 w-3 mr-1" />
                        Download for macOS
                      </Button>
                    </div>
                  </div>
                  
                  <div className="flex items-start gap-3 p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                    <div className="w-6 h-6 bg-green-600 rounded-full flex items-center justify-center text-white text-xs font-bold">2</div>
                    <div>
                      <p className="text-sm font-medium">Install Application</p>
                      <p className="text-xs text-muted-foreground">Open the .dmg file and drag MCP Desk to your Applications folder</p>
                    </div>
                  </div>
                  
                  <div className="flex items-start gap-3 p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                    <div className="w-6 h-6 bg-green-600 rounded-full flex items-center justify-center text-white text-xs font-bold">3</div>
                    <div>
                      <p className="text-sm font-medium">Grant Permissions</p>
                      <p className="text-xs text-muted-foreground">Allow MCP Desk through Security & Privacy settings when prompted</p>
                    </div>
                  </div>
                </div>
              </div>
            </TabsContent>
            
            <TabsContent value="linux" className="space-y-4">
              <div className="space-y-3">
                <h4 className="font-semibold">System Requirements</h4>
                <div className="p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                  <ul className="text-sm space-y-1">
                    <li>• Ubuntu 20.04 LTS or later</li>
                    <li>• RHEL 8 or later / CentOS 8 or later</li>
                    <li>• 4GB RAM minimum, 8GB recommended</li>
                    <li>• 500MB available disk space</li>
                    <li>• X11 or Wayland display server</li>
                  </ul>
                </div>
                
                <h4 className="font-semibold">Installation Options</h4>
                <div className="space-y-3">
                  <div className="p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                    <h5 className="text-sm font-medium mb-2">AppImage (Recommended)</h5>
                    <div className="space-y-2 text-sm">
                      <p>1. Download the .AppImage file</p>
                      <p>2. Make it executable: <code className="bg-gray-100 dark:bg-gray-900/50 px-1 rounded">chmod +x MCPDesk.AppImage</code></p>
                      <p>3. Run: <code className="bg-gray-100 dark:bg-gray-900/50 px-1 rounded">./MCPDesk.AppImage</code></p>
                    </div>
                    <Button size="sm" className="mt-2 bg-green-600 hover:bg-green-700">
                      <Download className="h-3 w-3 mr-1" />
                      Download AppImage
                    </Button>
                  </div>
                  
                  <div className="p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                    <h5 className="text-sm font-medium mb-2">Package Manager</h5>
                    <div className="space-y-1 text-sm font-mono bg-gray-100 dark:bg-gray-900/50 p-2 rounded">
                      <p># Ubuntu/Debian</p>
                      <p>wget -qO- https://repo.kimbapio.com/key.gpg | sudo apt-key add -</p>
                      <p>sudo apt update && sudo apt install mcp-desk</p>
                    </div>
                  </div>
                </div>
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

      {/* Connection Status */}
      <Card className="border-slate-200 dark:border-slate-800/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-lg hover:shadow-xl transition-all duration-300">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-slate-900 dark:text-slate-100">
            <div className="w-8 h-8 bg-gradient-to-br from-indigo-500 to-indigo-600 rounded-lg flex items-center justify-center shadow-md">
              <Activity className="h-4 w-4 text-white" />
            </div>
            Connection Management
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="bg-gradient-to-br from-indigo-50 to-blue-100 dark:from-indigo-950/30 dark:to-blue-900/30 rounded-lg p-6 border border-indigo-200 dark:border-indigo-800">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="text-center">
                <div className="w-3 h-3 bg-green-500 rounded-full mx-auto mb-2 animate-pulse"></div>
                <p className="text-sm font-medium text-indigo-800 dark:text-indigo-300">Connected</p>
                <p className="text-xs text-indigo-700 dark:text-indigo-300">3 servers active</p>
              </div>
              <div className="text-center">
                <div className="w-3 h-3 bg-blue-500 rounded-full mx-auto mb-2"></div>
                <p className="text-sm font-medium text-indigo-800 dark:text-indigo-300">Authenticated</p>
                <p className="text-xs text-indigo-700 dark:text-indigo-300">Token valid</p>
              </div>
              <div className="text-center">
                <div className="w-3 h-3 bg-purple-500 rounded-full mx-auto mb-2"></div>
                <p className="text-sm font-medium text-indigo-800 dark:text-indigo-300">Monitoring</p>
                <p className="text-xs text-indigo-700 dark:text-indigo-300">Real-time status</p>
              </div>
            </div>
          </div>

          <div className="space-y-3">
            <h4 className="font-semibold">Active Connections</h4>
            <div className="space-y-2">
              <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-indigo-200 dark:border-indigo-800 rounded-lg">
                <div className="flex items-center gap-3">
                  <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                  <div>
                    <p className="text-sm font-medium">Production Server</p>
                    <p className="text-xs text-muted-foreground">https://mcp.company.com</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Connected</Badge>
                  <span className="text-xs text-muted-foreground">142ms</span>
                </div>
              </div>
              
              <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-indigo-200 dark:border-indigo-800 rounded-lg">
                <div className="flex items-center gap-3">
                  <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                  <div>
                    <p className="text-sm font-medium">Staging Server</p>
                    <p className="text-xs text-muted-foreground">https://staging-mcp.company.com</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Connected</Badge>
                  <span className="text-xs text-muted-foreground">289ms</span>
                </div>
              </div>
              
              <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-yellow-200 dark:border-yellow-800 rounded-lg">
                <div className="flex items-center gap-3">
                  <div className="w-2 h-2 bg-yellow-500 rounded-full"></div>
                  <div>
                    <p className="text-sm font-medium">Configured Server</p>
                    <p className="text-xs text-muted-foreground">https://dev-mcp.company.com</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Badge className="bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300 text-xs">Reconnecting</Badge>
                  <span className="text-xs text-muted-foreground">timeout</span>
                </div>
              </div>
            </div>
          </div>

          <Alert className="bg-indigo-50 dark:bg-indigo-950/30 border-indigo-200 dark:border-indigo-800">
            <Network className="h-4 w-4" />
            <AlertDescription className="text-sm">
              <strong>Auto-Discovery:</strong> MCP Desk automatically discovers and connects to MCP servers on your local network. 
              Manual server addresses can be added in settings.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>

    </div>
  )
}
