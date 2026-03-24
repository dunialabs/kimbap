import { CheckCircle, AlertTriangle, Download, Server, Monitor, Zap, Key, Globe, Users, PlayCircle, Code, Database, Lock, Copy, GitBranch, Terminal, Crown } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function QuickSetupSection() {
  return (
    <div className="space-y-8 lg:space-y-12">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-green-500 to-emerald-600 rounded-2xl mb-4 shadow-lg shadow-green-500/25">
          <Zap className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-2xl lg:text-3xl font-bold mb-2 text-slate-900 dark:text-slate-100">Quick Setup Guide</h1>
        <p className="text-base lg:text-lg text-slate-600 dark:text-slate-400 max-w-2xl mx-auto">
          Get your AI tool orchestration running in 3 simple steps, ~15 minutes total.
        </p>
      </div>


      {/* Step 1: Install Console */}
      <Card className="border-slate-200 dark:border-slate-800/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-lg">
        <CardHeader>
          <CardTitle className="flex items-center gap-3 text-xl text-slate-900 dark:text-slate-100">
            <div className="w-8 h-8 bg-gradient-to-br from-green-500 to-emerald-600 rounded-lg flex items-center justify-center">
              <span className="font-bold text-white text-sm">1</span>
            </div>
            Install Console
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-slate-600 dark:text-slate-300">
            Deploy MCP Console on your server using Docker.
          </p>
          <Alert className="bg-blue-50 dark:bg-blue-950/30 border-blue-200 dark:border-blue-800">
            <CheckCircle className="h-4 w-4" />
            <AlertDescription>
              Console will be accessible at <strong>https://your-domain.com</strong> after installation
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>

      {/* Step 2: Add Tools */}
      <Card className="border-slate-200 dark:border-slate-800/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-lg">
        <CardHeader>
          <CardTitle className="flex items-center gap-3 text-xl text-slate-900 dark:text-slate-100">
            <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-cyan-600 rounded-lg flex items-center justify-center">
              <span className="font-bold text-white text-sm">2</span>
            </div>
            Add Tools
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-slate-600 dark:text-slate-300">
            Configure GitHub, Notion, and other tools through the web console.
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div className="p-4 bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 rounded-lg">
              <p className="font-medium mb-1">Setup Server</p>
              <p className="text-sm text-slate-600 dark:text-slate-400">Initialize and save Owner Token</p>
            </div>
            <div className="p-4 bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded-lg">
              <p className="font-medium mb-1">Configure Tools</p>
              <p className="text-sm text-slate-600 dark:text-slate-400">Add API keys for GitHub, Notion, etc.</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Step 3: Connect AI */}
      <Card className="border-slate-200 dark:border-slate-800/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-lg">
        <CardHeader>
          <CardTitle className="flex items-center gap-3 text-xl text-slate-900 dark:text-slate-100">
            <div className="w-8 h-8 bg-gradient-to-br from-purple-500 to-violet-600 rounded-lg flex items-center justify-center">
              <span className="font-bold text-white text-sm">3</span>
            </div>
            Connect AI
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-slate-600 dark:text-slate-300">
            Install MCP Desk and configure Claude Desktop or Cursor to connect.
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div className="p-4 bg-purple-50 dark:bg-purple-950/30 border border-purple-200 dark:border-purple-800 rounded-lg">
              <p className="font-medium mb-1">Install MCP Desk</p>
              <p className="text-sm text-slate-600 dark:text-slate-400">Download and install desktop client</p>
            </div>
            <div className="p-4 bg-indigo-50 dark:bg-indigo-950/30 border border-indigo-200 dark:border-indigo-800 rounded-lg">
              <p className="font-medium mb-1">Configure AI Client</p>
              <p className="text-sm text-slate-600 dark:text-slate-400">Connect Claude Desktop or Cursor</p>
            </div>
          </div>
          <Alert className="bg-green-50 dark:bg-green-950/30 border-green-200 dark:border-green-800">
            <PlayCircle className="h-4 w-4" />
            <AlertDescription>
              <strong>Test:</strong> Ask Claude "list my GitHub repositories" to verify everything works!
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
      
      {/* Next Steps */}
       <Card className="border-slate-200/60 dark:border-slate-800/60 bg-gradient-to-r from-green-50 to-emerald-50 dark:from-green-950/30 dark:to-emerald-950/30 border-green-200 dark:border-green-800">
        <CardHeader>
          <CardTitle className="text-center text-slate-900 dark:text-slate-100">🎉 You're All Set!</CardTitle>
        </CardHeader>
        <CardContent className="text-center space-y-4">
          <p className="text-slate-600 dark:text-slate-400">
            Your AI tool orchestration is now running. Try asking Claude to interact with your tools!
          </p>
          <div className="flex flex-wrap justify-center gap-3">
            <Badge variant="outline">✅ Console Running</Badge>
            <Badge variant="outline">✅ Tools Connected</Badge>
            <Badge variant="outline">✅ AI Ready</Badge>
          </div>
        </CardContent>
      </Card>

    </div>
  )
}