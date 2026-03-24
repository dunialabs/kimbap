import { Monitor, Terminal, Download, Shield, CheckCircle, Lock, Clock, Key, RefreshCw } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

export function DeskDownloadSection() {
  return (
    <div className="space-y-8 lg:space-y-12">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-purple-500 to-violet-600 rounded-2xl mb-4 shadow-lg shadow-purple-500/25">
          <Monitor className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-2xl lg:text-3xl font-bold mb-2 text-slate-900 dark:text-slate-100">Download Kimbap Desk</h1>
        <p className="text-base lg:text-lg text-slate-600 dark:text-slate-400 max-w-2xl mx-auto">
          Install the MCP Desk desktop client to connect AI assistants to your organization's MCP servers. 
          Available for Windows, macOS, and Linux with native performance and secure authentication.
        </p>
      </div>

      {/* Desk Overview */}
       <Card className="border-slate-200 dark:border-slate-800/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-lg hover:shadow-xl transition-all duration-300 border-2 border-purple-200 dark:border-purple-800 bg-gradient-to-br from-purple-50 to-violet-50 dark:from-purple-950/30 dark:to-violet-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-slate-900 dark:text-slate-100">
            <div className="w-8 h-8 bg-gradient-to-br from-purple-500 to-violet-600 rounded-lg flex items-center justify-center shadow-md">
              <Monitor className="h-4 w-4 text-white" />
            </div>
            What is MCP Desk?
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-purple-700 dark:text-purple-300">
            MCP Desk is the essential desktop client that connects AI assistants like Claude Desktop and Cursor to your organization's MCP servers. It acts as a secure bridge, providing authentication, connection management, and enhanced performance for AI tool integrations.
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-purple-200 dark:border-purple-800">
              <div className="w-10 h-10 bg-gradient-to-br from-purple-500 to-purple-600 rounded-lg flex items-center justify-center mx-auto mb-2 shadow-lg shadow-purple-500/25">
                <Shield className="h-5 w-5 text-white" />
              </div>
              <p className="font-medium text-sm mb-1">Secure Bridge</p>
              <p className="text-xs text-slate-600 dark:text-slate-400">Token-based authentication</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-blue-200 dark:border-blue-800">
              <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-blue-600 rounded-lg flex items-center justify-center mx-auto mb-2 shadow-lg shadow-blue-500/25">
                <Monitor className="h-5 w-5 text-white" />
              </div>
              <p className="font-medium text-sm mb-1">Native Performance</p>
              <p className="text-xs text-slate-600 dark:text-slate-400">Optimized for desktop use</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-green-200 dark:border-green-800">
              <div className="w-10 h-10 bg-gradient-to-br from-green-500 to-green-600 rounded-lg flex items-center justify-center mx-auto mb-2 shadow-lg shadow-green-500/25">
                <CheckCircle className="h-5 w-5 text-white" />
              </div>
              <p className="font-medium text-sm mb-1">Easy Setup</p>
              <p className="text-xs text-slate-600 dark:text-slate-400">One-click server connection</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-orange-200 dark:border-orange-800">
              <div className="w-10 h-10 bg-gradient-to-br from-orange-500 to-orange-600 rounded-lg flex items-center justify-center mx-auto mb-2 shadow-lg shadow-orange-500/25">
                <Terminal className="h-5 w-5 text-white" />
              </div>
              <p className="font-medium text-sm mb-1">Cross-Platform</p>
              <p className="text-xs text-slate-600 dark:text-slate-400">Windows, macOS, Linux</p>
            </div>
          </div>
          <div className="flex justify-center">
            <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300">Required for End Users</Badge>
          </div>
        </CardContent>
      </Card>

      <Alert className="border-purple-200 bg-purple-50 dark:bg-purple-950/30 dark:border-purple-800">
        <Monitor className="h-4 w-4" />
        <AlertDescription>
          <strong>MCP Desk</strong> is essential for users to access MCP tools. It acts as a secure bridge between 
          AI assistants and your organization's MCP servers, providing authentication and connection management.
        </AlertDescription>
      </Alert>

      {/* Platform Downloads */}
      <div>
        <h2 className="text-xl font-semibold mb-4">Download MCP Desk</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
           <Card className="border-slate-200 dark:border-slate-800/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-lg hover:shadow-xl transition-all duration-300 border-blue-200 dark:border-blue-800 bg-gradient-to-b from-blue-50 dark:from-blue-950 to-white dark:to-slate-900">
            <CardHeader className="text-center">
              <div className="w-16 h-16 bg-gradient-to-br from-blue-500 to-blue-600 rounded-2xl flex items-center justify-center mx-auto mb-2 shadow-lg shadow-blue-500/25">
                <Monitor className="h-8 w-8 text-white" />
              </div>
              <CardTitle className="text-lg text-slate-900 dark:text-slate-100">Windows</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-center">
              <p className="text-sm text-muted-foreground">Windows 10 or later</p>
              <Button className="w-full bg-blue-600 hover:bg-blue-700">
                <Download className="h-4 w-4 mr-2" />
                Download .exe
              </Button>
              <div className="text-xs text-muted-foreground space-y-1">
                <p>Version 2.1.0</p>
                <p>64-bit Intel/AMD</p>
                <p>~85MB download</p>
              </div>
            </CardContent>
          </Card>

           <Card className="border-slate-200 dark:border-slate-800/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-lg hover:shadow-xl transition-all duration-300 border-gray-200 dark:border-gray-800 bg-gradient-to-b from-gray-50 dark:from-gray-950 to-white dark:to-slate-900">
            <CardHeader className="text-center">
              <div className="w-16 h-16 bg-gradient-to-br from-slate-500 to-slate-600 rounded-2xl flex items-center justify-center mx-auto mb-2 shadow-lg shadow-slate-500/25">
                <Monitor className="h-8 w-8 text-white" />
              </div>
              <CardTitle className="text-lg text-slate-900 dark:text-slate-100">macOS</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-center">
              <p className="text-sm text-muted-foreground">macOS 11.0 or later</p>
              <Button className="w-full bg-gray-800 hover:bg-gray-900">
                <Download className="h-4 w-4 mr-2" />
                Download .dmg
              </Button>
              <div className="text-xs text-muted-foreground space-y-1">
                <p>Version 2.1.0</p>
                <p>Universal Binary (Intel/Apple Silicon)</p>
                <p>~92MB download</p>
              </div>
            </CardContent>
          </Card>

           <Card className="border-slate-200 dark:border-slate-800/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-lg hover:shadow-xl transition-all duration-300 border-orange-200 dark:border-orange-800 bg-gradient-to-b from-orange-50 dark:from-orange-950 to-white dark:to-slate-900">
            <CardHeader className="text-center">
              <div className="w-16 h-16 bg-gradient-to-br from-orange-500 to-orange-600 rounded-2xl flex items-center justify-center mx-auto mb-2 shadow-lg shadow-orange-500/25">
                <Terminal className="h-8 w-8 text-white" />
              </div>
              <CardTitle className="text-lg text-slate-900 dark:text-slate-100">Linux</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-center">
              <p className="text-sm text-muted-foreground">Ubuntu 20.04+ / equivalent</p>
              <Button className="w-full bg-orange-600 hover:bg-orange-700">
                <Download className="h-4 w-4 mr-2" />
                Download .AppImage
              </Button>
              <div className="text-xs text-muted-foreground space-y-1">
                <p>Version 2.1.0</p>
                <p>x64 architecture</p>
                <p>~88MB download</p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Installation Steps */}
      <Card className="border-slate-200 dark:border-slate-800/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-lg hover:shadow-xl transition-all duration-300">
        <CardHeader>
          <CardTitle className="text-slate-900 dark:text-slate-100">Installation Process</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="space-y-4">
              <h4 className="font-semibold">Step-by-Step Installation</h4>
              <ol className="space-y-3">
                <li className="flex items-start gap-3">
                  <div className="w-6 h-6 bg-purple-100 dark:bg-purple-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                    1
                  </div>
                  <div>
                    <p className="font-medium text-sm">Download Installer</p>
                    <p className="text-xs text-muted-foreground">Choose the appropriate version for your operating system</p>
                  </div>
                </li>
                <li className="flex items-start gap-3">
                  <div className="w-6 h-6 bg-purple-100 dark:bg-purple-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                    2
                  </div>
                  <div>
                    <p className="font-medium text-sm">Run Installation</p>
                    <p className="text-xs text-muted-foreground">Follow the setup wizard (requires admin rights)</p>
                  </div>
                </li>
                <li className="flex items-start gap-3">
                  <div className="w-6 h-6 bg-purple-100 dark:bg-purple-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                    3
                  </div>
                  <div>
                    <p className="font-medium text-sm">Launch Application</p>
                    <p className="text-xs text-muted-foreground">MCP Desk will start automatically after installation</p>
                  </div>
                </li>
                <li className="flex items-start gap-3">
                  <div className="w-6 h-6 bg-purple-100 dark:bg-purple-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                    4
                  </div>
                  <div>
                    <p className="font-medium text-sm">Add Server Connection</p>
                    <p className="text-xs text-muted-foreground">Enter your server token to connect</p>
                  </div>
                </li>
              </ol>
            </div>

            <div className="space-y-4">
              <h4 className="font-semibold">Initial Security Setup</h4>
              <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-4 border border-slate-200 dark:border-slate-700 space-y-4">
                <Alert className="bg-red-50 dark:bg-red-950/30 border-red-200 dark:border-red-800">
                  <Lock className="h-4 w-4" />
                  <AlertDescription className="text-xs">
                    <strong>First Launch:</strong> You'll be prompted to create a Master Password to secure all your server connections and enable biometric authentication features.
                  </AlertDescription>
                </Alert>
                
                <div className="grid grid-cols-1 gap-3">
                  <div className="flex items-start gap-3 p-3 bg-white dark:bg-slate-800 border border-red-200 dark:border-red-800 rounded-lg">
                    <Lock className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5" />
                    <div>
                      <p className="text-xs font-medium text-red-800 dark:text-red-300">Set Master Password</p>
                      <p className="text-xs text-red-600 dark:text-red-400">Protects all connections and credentials</p>
                    </div>
                  </div>
                  
                  <div className="flex items-start gap-3 p-3 bg-white dark:bg-slate-800 border border-orange-200 dark:border-orange-800 rounded-lg">
                    <Clock className="h-4 w-4 text-orange-600 dark:text-orange-400 mt-0.5" />
                    <div>
                      <p className="text-xs font-medium text-orange-800 dark:text-orange-300">Configure Auto-Lock</p>
                      <p className="text-xs text-orange-600 dark:text-orange-400">Set timeout before re-authentication required</p>
                    </div>
                  </div>
                  
                  <div className="flex items-start gap-3 p-3 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded-lg">
                    <Key className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                    <div>
                      <p className="text-xs font-medium text-blue-800 dark:text-blue-300">Add Server Connections</p>
                      <p className="text-xs text-blue-600 dark:text-blue-400">Use access tokens from your MCP Console</p>
                    </div>
                  </div>
                  
                  <div className="flex items-start gap-3 p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                    <RefreshCw className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                    <div>
                      <p className="text-xs font-medium text-green-800 dark:text-green-300">Restart AI Clients</p>
                      <p className="text-xs text-green-600 dark:text-green-400">Restart Claude Desktop/Cursor to apply changes</p>
                    </div>
                  </div>
                </div>
              </div>
              
              <Alert className="bg-amber-50 dark:bg-amber-950/30 border-amber-200 dark:border-amber-800">
                <Shield className="h-4 w-4" />
                <AlertDescription className="text-xs">
                  <strong>Security Note:</strong> The master password cannot be recovered if lost. MCP Desk uses local encryption and doesn't store passwords on remote servers.
                </AlertDescription>
              </Alert>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* System Requirements */}
      <Card className="border-slate-200 dark:border-slate-800/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-lg hover:shadow-xl transition-all duration-300">
        <CardHeader>
          <CardTitle className="text-slate-900 dark:text-slate-100">System Requirements</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="p-4 border rounded-lg">
              <h4 className="font-semibold mb-3 text-blue-800 dark:text-blue-300">Windows</h4>
              <ul className="text-sm space-y-1 text-muted-foreground">
                <li>• Windows 10 (1903) or later</li>
                <li>• 4GB RAM minimum</li>
                <li>• 200MB disk space</li>
                <li>• .NET Framework 4.8+</li>
                <li>• Internet connection</li>
              </ul>
            </div>
            <div className="p-4 border rounded-lg">
              <h4 className="font-semibold mb-3 text-gray-800 dark:text-gray-200">macOS</h4>
              <ul className="text-sm space-y-1 text-muted-foreground">
                <li>• macOS 11.0 Big Sur or later</li>
                <li>• 4GB RAM minimum</li>
                <li>• 200MB disk space</li>
                <li>• Apple Silicon or Intel processor</li>
                <li>• Internet connection</li>
              </ul>
            </div>
            <div className="p-4 border rounded-lg">
              <h4 className="font-semibold mb-3 text-orange-800 dark:text-orange-300">Linux</h4>
              <ul className="text-sm space-y-1 text-muted-foreground">
                <li>• Ubuntu 20.04+ or equivalent</li>
                <li>• 4GB RAM minimum</li>
                <li>• 200MB disk space</li>
                <li>• x64 architecture</li>
                <li>• AppImage support (FUSE)</li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Connection Setup */}
       <Card className="border-slate-200 dark:border-slate-800/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-lg hover:shadow-xl transition-all duration-300 border-2 border-green-200 dark:border-green-800 bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950/30 dark:to-emerald-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-slate-900 dark:text-slate-100">
            <div className="w-8 h-8 bg-gradient-to-br from-green-500 to-emerald-600 rounded-lg flex items-center justify-center shadow-md">
              <Shield className="h-4 w-4 text-white" />
            </div>
            Server Connection Setup
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-green-700 dark:text-green-300">
            After installing MCP Desk, you'll need to connect it to your organization's MCP servers. Your administrator will provide you with the necessary connection details.
          </p>
          
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <div className="space-y-3">
              <h4 className="font-semibold">What You'll Need</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <div className="w-4 h-4 bg-gradient-to-br from-green-500 to-green-600 rounded-full flex items-center justify-center mt-0.5">
                    <span className="w-1.5 h-1.5 bg-white rounded-full"></span>
                  </div>
                  <div>
                    <p className="text-sm font-medium">Server URL</p>
                    <p className="text-xs text-slate-600 dark:text-slate-400">Your organization's MCP server address</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <div className="w-4 h-4 bg-gradient-to-br from-green-500 to-green-600 rounded-full flex items-center justify-center mt-0.5">
                    <span className="w-1.5 h-1.5 bg-white rounded-full"></span>
                  </div>
                  <div>
                    <p className="text-sm font-medium">Access Token</p>
                    <p className="text-xs text-slate-600 dark:text-slate-400">Personal access token from administrator</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <div className="w-4 h-4 bg-gradient-to-br from-green-500 to-green-600 rounded-full flex items-center justify-center mt-0.5">
                    <span className="w-1.5 h-1.5 bg-white rounded-full"></span>
                  </div>
                  <div>
                    <p className="text-sm font-medium">Network Access</p>
                    <p className="text-xs text-slate-600 dark:text-slate-400">Ensure you can reach the server (VPN if needed)</p>
                  </div>
                </li>
              </ul>
            </div>
            
            <div className="space-y-3">
              <h4 className="font-semibold">Connection Process</h4>
              <ol className="space-y-2">
                <li className="flex items-start gap-2">
                  <span className="w-5 h-5 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">1</span>
                  <p className="text-sm">Launch MCP Desk</p>
                </li>
                <li className="flex items-start gap-2">
                  <span className="w-5 h-5 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">2</span>
                  <p className="text-sm">Click "Add Server Connection"</p>
                </li>
                <li className="flex items-start gap-2">
                  <span className="w-5 h-5 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">3</span>
                  <p className="text-sm">Enter server URL and token</p>
                </li>
                <li className="flex items-start gap-2">
                  <span className="w-5 h-5 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">4</span>
                  <p className="text-sm">Test connection and save</p>
                </li>
              </ol>
            </div>
          </div>
        </CardContent>
      </Card>

    </div>
  )
}