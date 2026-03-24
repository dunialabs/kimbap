import { Rocket, Monitor, Server, Users, Code, ArrowRight, Book } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import Image from "next/image"

interface GettingStartedSectionProps {
  onNavigate?: (sectionId: string) => void;
}

export function GettingStartedSection({ onNavigate }: GettingStartedSectionProps = {}) {
  return (
    <div className="space-y-8 lg:space-y-12">
      {/* Header */}
      <div className="text-center relative">
        {/* Background decoration */}
        <div className="absolute inset-0 bg-gradient-to-br from-blue-50 via-white to-purple-50/50 dark:from-blue-950/20 dark:via-slate-900 dark:to-purple-950/20 rounded-3xl -z-10" />
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,rgba(59,130,246,0.1),transparent_70%)] dark:bg-[radial-gradient(circle_at_50%_50%,rgba(59,130,246,0.05),transparent_70%)] rounded-3xl -z-10" />
        
        <div className="py-12">
          <h1 className="text-4xl lg:text-5xl font-bold mb-6 text-slate-900 dark:text-slate-100 flex flex-col items-center gap-4">
            <span>Welcome to</span>
            <Image
              src="/logo-with-text.png"
              alt="Kimbap Console"
              width={320}
              height={80}
              className="h-12 lg:h-16 w-auto"
            />
          </h1>
          <p className="text-lg lg:text-xl text-slate-600 dark:text-slate-300 max-w-4xl mx-auto leading-relaxed mb-8">
            Bridge the gap between AI assistants and your organization's tools through the Model Context Protocol (MCP). 
            Deploy on your infrastructure, maintain complete control over data, and empower teams with secure AI-powered workflows.
          </p>
          <div className="flex flex-wrap justify-center gap-3">
            <Badge className="bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-950/50 dark:text-blue-300 dark:border-blue-800 px-3 py-1">Self-Hosted</Badge>
            <Badge className="bg-green-50 text-green-700 border-green-200 dark:bg-green-950/50 dark:text-green-300 dark:border-green-800 px-3 py-1">Enterprise Security</Badge>
            <Badge className="bg-purple-50 text-purple-700 border-purple-200 dark:bg-purple-950/50 dark:text-purple-300 dark:border-purple-800 px-3 py-1">MCP Protocol</Badge>
          </div>
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 lg:gap-8">
          {/* Kimbap MCP Console */}
          <Card className="group relative overflow-hidden border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm hover:shadow-2xl hover:shadow-blue-500/10 transition-all duration-500 hover:scale-[1.02] dark:border-slate-700/60 h-full">
            <div className="absolute inset-0 bg-gradient-to-br from-blue-50/50 to-purple-50/20 dark:from-blue-950/20 dark:to-purple-950/10" />
            <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-blue-500/10 to-purple-500/10 rounded-full -mr-16 -mt-16 transition-all duration-500 group-hover:scale-150" />
            
            <CardHeader className="relative pb-4">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center space-x-4">
                  <div className="h-12 w-12 rounded-xl bg-gradient-to-br from-blue-600 to-blue-700 flex items-center justify-center shadow-lg">
                    <Monitor className="h-6 w-6 text-white" />
                  </div>
                  <div>
                    <CardTitle className="text-xl font-bold text-slate-900 dark:text-slate-100">Kimbap MCP Console</CardTitle>
                    <p className="text-slate-600 dark:text-slate-400 text-sm">Configuration Tool • For Administrators</p>
                  </div>
                </div>
                <Badge className="border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-800 dark:bg-blue-950/50 dark:text-blue-300">
                  Web App
                </Badge>
              </div>
            </CardHeader>

            <CardContent className="relative space-y-6">
              <p className="text-slate-600 dark:text-slate-300 text-sm">
                Comprehensive web dashboard for configuring MCP servers, managing tool integrations, and controlling user access with enterprise-grade security.
              </p>
              
              <div className="space-y-3">
                <div className="flex items-center space-x-3 text-sm">
                  <div className="h-2 w-2 bg-blue-600 rounded-full flex-shrink-0" />
                  <span className="text-slate-700 dark:text-slate-300">Server deployment & lifecycle management</span>
                </div>
                <div className="flex items-center space-x-3 text-sm">
                  <div className="h-2 w-2 bg-blue-600 rounded-full flex-shrink-0" />
                  <span className="text-slate-700 dark:text-slate-300">Tool integration & health monitoring</span>
                </div>
                <div className="flex items-center space-x-3 text-sm">
                  <div className="h-2 w-2 bg-blue-600 rounded-full flex-shrink-0" />
                  <span className="text-slate-700 dark:text-slate-300">Access token & permissions control</span>
                </div>
                <div className="flex items-center space-x-3 text-sm">
                  <div className="h-2 w-2 bg-blue-600 rounded-full flex-shrink-0" />
                  <span className="text-slate-700 dark:text-slate-300">Enterprise security & audit logging</span>
                </div>
              </div>

              <div className="pt-4 space-y-3">
                <Button 
                  className="w-full bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 shadow-lg hover:shadow-xl transition-all duration-200"
                  onClick={() => onNavigate?.('console-overview')}
                >
                  <ArrowRight className="h-4 w-4 mr-2" />
                  Console Documentation
                </Button>
                <p className="text-xs text-center text-slate-500 dark:text-slate-400">
                  For administrators & DevOps teams
                </p>
              </div>
            </CardContent>
          </Card>

          {/* Kimbap Desk */}
          <Card className="group relative overflow-hidden border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm hover:shadow-2xl hover:shadow-green-500/10 transition-all duration-500 hover:scale-[1.02] dark:border-slate-700/60 h-full">
            <div className="absolute inset-0 bg-gradient-to-br from-green-50/50 to-emerald-50/20 dark:from-green-950/20 dark:to-emerald-950/10" />
            <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-green-500/10 to-emerald-500/10 rounded-full -mr-16 -mt-16 transition-all duration-500 group-hover:scale-150" />
            
            <CardHeader className="relative pb-4">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center space-x-4">
                  <div className="h-12 w-12 rounded-xl bg-gradient-to-br from-green-600 to-emerald-600 flex items-center justify-center shadow-lg">
                    <Users className="h-6 w-6 text-white" />
                  </div>
                  <div>
                    <CardTitle className="text-xl font-bold text-slate-900 dark:text-slate-100">Kimbap Desk</CardTitle>
                    <p className="text-slate-600 dark:text-slate-400 text-sm">Desktop Client • For End Users</p>
                  </div>
                </div>
                <Badge className="border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950/50 dark:text-green-300">
                  Desktop App
                </Badge>
              </div>
            </CardHeader>

            <CardContent className="relative space-y-6">
              <p className="text-slate-600 dark:text-slate-300 text-sm">
                Lightweight desktop application that seamlessly bridges AI assistants (Claude, Cursor) with your organization's MCP servers using secure token authentication.
              </p>
              
              <div className="space-y-3">
                <div className="flex items-center space-x-3 text-sm">
                  <div className="h-2 w-2 bg-green-600 rounded-full flex-shrink-0" />
                  <span className="text-slate-700 dark:text-slate-300">Cross-platform desktop installation</span>
                </div>
                <div className="flex items-center space-x-3 text-sm">
                  <div className="h-2 w-2 bg-green-600 rounded-full flex-shrink-0" />
                  <span className="text-slate-700 dark:text-slate-300">Secure server connection management</span>
                </div>
                <div className="flex items-center space-x-3 text-sm">
                  <div className="h-2 w-2 bg-green-600 rounded-full flex-shrink-0" />
                  <span className="text-slate-700 dark:text-slate-300">AI assistant configuration (Claude, Cursor)</span>
                </div>
                <div className="flex items-center space-x-3 text-sm">
                  <div className="h-2 w-2 bg-green-600 rounded-full flex-shrink-0" />
                  <span className="text-slate-700 dark:text-slate-300">Real-time tool availability synchronization</span>
                </div>
              </div>

              <div className="pt-4 space-y-3">
                <Button 
                  className="w-full bg-gradient-to-r from-green-600 to-emerald-600 hover:from-green-700 hover:to-emerald-700 shadow-lg hover:shadow-xl transition-all duration-200"
                  onClick={() => onNavigate?.('desk-overview')}
                >
                  <ArrowRight className="h-4 w-4 mr-2" />
                  Desk Documentation
                </Button>
                <p className="text-xs text-center text-slate-500 dark:text-slate-400">
                  For end users & knowledge workers
                </p>
              </div>
            </CardContent>
          </Card>

          {/* Kimbap MCP Core */}
          <Card className="group relative overflow-hidden border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm hover:shadow-2xl hover:shadow-purple-500/10 transition-all duration-500 hover:scale-[1.02] dark:border-slate-700/60 h-full">
            <div className="absolute inset-0 bg-gradient-to-br from-purple-50/50 to-violet-50/20 dark:from-purple-950/20 dark:to-violet-950/10" />
            <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-purple-500/10 to-violet-500/10 rounded-full -mr-16 -mt-16 transition-all duration-500 group-hover:scale-150" />
            
            <CardHeader className="relative pb-4">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center space-x-4">
                  <div className="h-12 w-12 rounded-xl bg-gradient-to-br from-purple-600 to-violet-600 flex items-center justify-center shadow-lg">
                    <Server className="h-6 w-6 text-white" />
                  </div>
                  <div>
                    <CardTitle className="text-xl font-bold text-slate-900 dark:text-slate-100">Kimbap MCP Core</CardTitle>
                    <p className="text-slate-600 dark:text-slate-400 text-sm">Server Engine • Self-Managed</p>
                  </div>
                </div>
                <Badge className="border-purple-200 bg-purple-50 text-purple-700 dark:border-purple-800 dark:bg-purple-950/50 dark:text-purple-300">
                  Local Server
                </Badge>
              </div>
            </CardHeader>

            <CardContent className="relative space-y-6">
              <p className="text-slate-600 dark:text-slate-300 text-sm">
                High-performance runtime engine that executes MCP servers, processes tool requests, and enforces security policies on your infrastructure.
              </p>
              
              <div className="space-y-3">
                <div className="flex items-center space-x-3 text-sm">
                  <div className="h-2 w-2 bg-purple-600 rounded-full flex-shrink-0" />
                  <span className="text-slate-700 dark:text-slate-300">MCP protocol server execution</span>
                </div>
                <div className="flex items-center space-x-3 text-sm">
                  <div className="h-2 w-2 bg-purple-600 rounded-full flex-shrink-0" />
                  <span className="text-slate-700 dark:text-slate-300">High-performance tool request processing</span>
                </div>
                <div className="flex items-center space-x-3 text-sm">
                  <div className="h-2 w-2 bg-purple-600 rounded-full flex-shrink-0" />
                  <span className="text-slate-700 dark:text-slate-300">Zero-trust authentication framework</span>
                </div>
                <div className="flex items-center space-x-3 text-sm">
                  <div className="h-2 w-2 bg-purple-600 rounded-full flex-shrink-0" />
                  <span className="text-slate-700 dark:text-slate-300">End-to-end encryption & compliance</span>
                </div>
              </div>

              <div className="pt-4 space-y-3">
                <Button disabled className="w-full bg-slate-100 text-slate-500 cursor-default dark:bg-slate-800 dark:text-slate-400">
                  <Server className="mr-2 h-4 w-4" />
                  Auto-deployed via Console
                </Button>
                <p className="text-xs text-center text-slate-500 dark:text-slate-400">
                  The foundation server platform
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}