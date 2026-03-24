import { Server, Monitor, Shield, ArrowRight, Globe, Code, Database, Lock, Zap } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

export function OverviewSection() {
  return (
    <div className="space-y-8 lg:space-y-12">
      {/* Header */}
      <div className="text-center relative">
        <div className="absolute inset-0 bg-gradient-to-br from-indigo-50 via-white to-purple-50/50 dark:from-indigo-950/20 dark:via-slate-900 dark:to-purple-950/20 rounded-3xl -z-10" />
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,rgba(99,102,241,0.1),transparent_70%)] dark:bg-[radial-gradient(circle_at_50%_50%,rgba(99,102,241,0.05),transparent_70%)] rounded-3xl -z-10" />
        
        <div className="py-8 lg:py-12">
          <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-2xl mb-6 shadow-2xl shadow-indigo-500/25">
            <Globe className="h-8 w-8 text-white" />
          </div>
          <h1 className="text-3xl lg:text-4xl font-bold mb-4 text-slate-900 dark:text-slate-100">
            Platform <span className="bg-gradient-to-r from-indigo-600 via-purple-600 to-indigo-800 bg-clip-text text-transparent">Overview</span>
          </h1>
          <p className="text-lg lg:text-xl text-slate-600 dark:text-slate-300 max-w-2xl mx-auto leading-relaxed">
            Understanding the Kimbap.io architecture and how all components work together to deliver secure AI tool orchestration.
          </p>
        </div>
      </div>

      {/* Architecture Overview */}
      <Card className="border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-xl hover:shadow-2xl transition-all duration-300 dark:border-slate-700/60">
        <CardHeader>
          <CardTitle className="text-2xl lg:text-3xl text-center text-slate-900 dark:text-slate-100">Architecture Overview</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-base lg:text-lg text-slate-600 dark:text-slate-300 max-w-5xl mx-auto leading-relaxed text-center">
            Kimbap.io consists of three integrated components that work together to provide secure AI tool orchestration. 
            The Console manages configuration, Core executes MCP servers, and Desk connects AI assistants to your infrastructure.
          </p>
        </CardContent>
      </Card>

      {/* Platform Components */}
      <div>
        <h2 className="text-2xl lg:text-3xl font-semibold mb-8 text-slate-900 dark:text-slate-100">Platform Components</h2>
        <div className="grid lg:grid-cols-3 gap-6 lg:gap-8">
          {/* Core Server */}
          <Card className="group relative overflow-hidden border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm hover:shadow-2xl hover:shadow-green-500/10 transition-all duration-500 hover:scale-[1.02] dark:border-slate-700/60">
            <div className="absolute inset-0 bg-gradient-to-br from-green-50/50 to-emerald-50/20 dark:from-green-950/20 dark:to-emerald-950/10" />
            <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-green-500/10 to-emerald-500/10 rounded-full -mr-16 -mt-16 transition-all duration-500 group-hover:scale-150" />
            
            <CardHeader className="relative pb-4">
              <CardTitle className="text-xl font-bold text-slate-900 dark:text-slate-100 flex items-center gap-2">
                <Server className="h-5 w-5 text-green-600 dark:text-green-400" />
                Kimbap MCP Core
              </CardTitle>
            </CardHeader>
            <CardContent className="relative space-y-4">
              <p className="text-sm text-slate-600 dark:text-slate-300">
                The runtime engine that executes MCP servers and handles tool integrations on your infrastructure.
              </p>
              
              <div>
                <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs border-green-200 dark:border-green-800">Core Functions</Badge>
                <ul className="text-xs space-y-1 text-slate-600 dark:text-slate-400 mt-2">
                  <li>• MCP server hosting and execution</li>
                  <li>• Tool integration processing</li>
                  <li>• Request routing and load balancing</li>
                  <li>• Security boundary enforcement</li>
                </ul>
              </div>
              
              <div className="space-y-1">
                <p className="text-xs font-medium text-slate-700 dark:text-slate-300">Deployment:</p>
                <p className="text-xs text-slate-500 dark:text-slate-400">Self-hosted on your infrastructure</p>
              </div>
            </CardContent>
          </Card>

          {/* Console */}
          <Card className="group relative overflow-hidden border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm hover:shadow-2xl hover:shadow-blue-500/10 transition-all duration-500 hover:scale-[1.02] dark:border-slate-700/60">
            <div className="absolute inset-0 bg-gradient-to-br from-blue-50/50 to-cyan-50/20 dark:from-blue-950/20 dark:to-cyan-950/10" />
            <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-blue-500/10 to-cyan-500/10 rounded-full -mr-16 -mt-16 transition-all duration-500 group-hover:scale-150" />
            
            <CardHeader className="relative pb-4">
              <CardTitle className="text-xl font-bold text-slate-900 dark:text-slate-100 flex items-center gap-2">
                <Monitor className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                Kimbap MCP Console
              </CardTitle>
            </CardHeader>
            <CardContent className="relative space-y-4">
              <p className="text-sm text-slate-600 dark:text-slate-300">
                Web-based administration interface for configuring servers, managing tools, and controlling access.
              </p>
              
              <div>
                <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs border-blue-200 dark:border-blue-800">Admin Functions</Badge>
                <ul className="text-xs space-y-1 text-slate-600 dark:text-slate-400 mt-2">
                  <li>• Server configuration and monitoring</li>
                  <li>• Tool management and testing</li>
                  <li>• User access control and tokens</li>
                  <li>• Usage analytics and logging</li>
                </ul>
              </div>
              
              <div className="space-y-1">
                <p className="text-xs font-medium text-slate-700 dark:text-slate-300">Access:</p>
                <p className="text-xs text-slate-500 dark:text-slate-400">Web interface for administrators</p>
              </div>
            </CardContent>
          </Card>

          {/* Desk */}
          <Card className="group relative overflow-hidden border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm hover:shadow-2xl hover:shadow-purple-500/10 transition-all duration-500 hover:scale-[1.02] dark:border-slate-700/60">
            <div className="absolute inset-0 bg-gradient-to-br from-purple-50/50 to-violet-50/20 dark:from-purple-950/20 dark:to-violet-950/10" />
            <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-purple-500/10 to-violet-500/10 rounded-full -mr-16 -mt-16 transition-all duration-500 group-hover:scale-150" />
            
            <CardHeader className="relative pb-4">
              <CardTitle className="text-xl font-bold text-slate-900 dark:text-slate-100 flex items-center gap-2">
                <Shield className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                Kimbap Desk
              </CardTitle>
            </CardHeader>
            <CardContent className="relative space-y-4">
              <p className="text-sm text-slate-600 dark:text-slate-300">
                Desktop client that connects AI assistants to your MCP servers with secure authentication.
              </p>
              
              <div>
                <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 text-xs border-purple-200 dark:border-purple-800">Client Functions</Badge>
                <ul className="text-xs space-y-1 text-slate-600 dark:text-slate-400 mt-2">
                  <li>• AI assistant connection management</li>
                  <li>• Secure token-based authentication</li>
                  <li>• Real-time tool availability sync</li>
                  <li>• Cross-platform compatibility</li>
                </ul>
              </div>
              
              <div className="space-y-1">
                <p className="text-xs font-medium text-slate-700 dark:text-slate-300">Installation:</p>
                <p className="text-xs text-slate-500 dark:text-slate-400">Desktop app for end users</p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Architecture Flow */}
      <Card className="border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-xl hover:shadow-2xl transition-all duration-300 dark:border-slate-700/60">
        <CardHeader>
          <CardTitle className="text-2xl lg:text-3xl text-slate-900 dark:text-slate-100">How It All Works Together</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
            <div className="text-center space-y-3">
              <div className="w-12 h-12 bg-gradient-to-br from-blue-100 to-blue-200 dark:from-blue-800 dark:to-blue-700 rounded-full flex items-center justify-center mx-auto">
                <span className="text-blue-700 dark:text-blue-200 font-bold">1</span>
              </div>
              <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Admin Setup</h4>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                Administrator uses Console to configure servers, add tools, and create access tokens
              </p>
            </div>

            <div className="text-center space-y-3">
              <div className="w-12 h-12 bg-gradient-to-br from-green-100 to-green-200 dark:from-green-800 dark:to-green-700 rounded-full flex items-center justify-center mx-auto">
                <span className="text-green-700 dark:text-green-200 font-bold">2</span>
              </div>
              <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Core Deployment</h4>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                MCP Core runs on infrastructure, hosts configured tools, and enforces security policies
              </p>
            </div>

            <div className="text-center space-y-3">
              <div className="w-12 h-12 bg-gradient-to-br from-purple-100 to-purple-200 dark:from-purple-800 dark:to-purple-700 rounded-full flex items-center justify-center mx-auto">
                <span className="text-purple-700 dark:text-purple-200 font-bold">3</span>
              </div>
              <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">User Connection</h4>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                End users install Desk, input access token, and connect their AI assistants
              </p>
            </div>

            <div className="text-center space-y-3">
              <div className="w-12 h-12 bg-gradient-to-br from-orange-100 to-orange-200 dark:from-orange-800 dark:to-orange-700 rounded-full flex items-center justify-center mx-auto">
                <span className="text-orange-700 dark:text-orange-200 font-bold">4</span>
              </div>
              <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Secure Access</h4>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                AI assistants can now securely access organizational tools through encrypted channels
              </p>
            </div>
          </div>

          <Alert className="bg-white/90 dark:bg-slate-800/90 border-slate-200/60 dark:border-slate-700/60">
            <Shield className="h-5 w-5" />
            <AlertDescription className="text-sm text-slate-700 dark:text-slate-300">
              <strong>Enterprise Ready:</strong> Designed for organizations that need complete control over AI tool access and data security.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    </div>
  )
}