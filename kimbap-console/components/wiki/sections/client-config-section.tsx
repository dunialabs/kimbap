import { Monitor, Settings, CheckCircle, AlertCircle, Code, Download, Copy, Play, Zap, ChevronRight, Terminal, Globe, Shield, Eye, User, Server, RefreshCw, FileCode, Wrench } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function ClientConfigSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-2xl mb-4">
          <Code className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">AI Client Setup</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Configure your AI assistants to work with MCP Desk and access your organization's tools. 
          Detailed setup guides for Claude Desktop, Cursor IDE, and other MCP-compatible clients.
        </p>
      </div>

      {/* Official Supported Clients */}
      <Card className="border-2 border-green-200 dark:border-green-800 bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950/30 dark:to-emerald-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-green-800 dark:text-green-300">
            <CheckCircle className="h-6 w-6" />
            Official Supported Clients
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Alert className="bg-green-50 dark:bg-green-950/30 border-green-200 dark:border-green-800">
            <Zap className="h-4 w-4" />
            <AlertDescription>
              <strong>Automatic Configuration:</strong> These clients are officially supported by MCP Desk and require no manual JSON configuration. Simply restart the client after setting up MCP Desk.
            </AlertDescription>
          </Alert>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="text-center p-4 bg-white dark:bg-slate-800 rounded-lg border border-green-200 dark:border-green-800">
              <div className="w-12 h-12 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center mx-auto mb-3">
                <Code className="h-6 w-6 text-blue-600 dark:text-blue-400" />
              </div>
              <p className="font-medium text-sm mb-1">Claude Desktop</p>
              <p className="text-xs text-muted-foreground mb-2">Anthropic's official desktop app</p>
              <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Auto-Config</Badge>
            </div>
            <div className="text-center p-4 bg-white dark:bg-slate-800 rounded-lg border border-green-200 dark:border-green-800">
              <div className="w-12 h-12 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center mx-auto mb-3">
                <Terminal className="h-6 w-6 text-purple-600 dark:text-purple-400" />
              </div>
              <p className="font-medium text-sm mb-1">Cursor</p>
              <p className="text-xs text-muted-foreground mb-2">AI-powered code editor</p>
              <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Auto-Config</Badge>
            </div>
            <div className="text-center p-4 bg-white dark:bg-slate-800 rounded-lg border border-green-200 dark:border-green-800">
              <div className="w-12 h-12 bg-indigo-100 dark:bg-indigo-900/30 rounded-lg flex items-center justify-center mx-auto mb-3">
                <Globe className="h-6 w-6 text-indigo-600 dark:text-indigo-400" />
              </div>
              <p className="font-medium text-sm mb-1">Windsurf</p>
              <p className="text-xs text-muted-foreground mb-2">Codeium's AI IDE</p>
              <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Auto-Config</Badge>
            </div>
          </div>
          
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border border-green-200 dark:border-green-800">
            <h4 className="font-semibold text-green-800 dark:text-green-300 mb-3 flex items-center gap-2">
              <RefreshCw className="h-4 w-4" />
              How Auto-Configuration Works
            </h4>
            <ol className="space-y-2 text-sm text-green-700 dark:text-green-300">
              <li className="flex items-start gap-2">
                <span className="font-semibold text-green-600 dark:text-green-400">1.</span>
                <span>MCP Desk automatically detects installed supported clients</span>
              </li>
              <li className="flex items-start gap-2">
                <span className="font-semibold text-green-600 dark:text-green-400">2.</span>
                <span>Generates and writes MCP configuration files automatically</span>
              </li>
              <li className="flex items-start gap-2">
                <span className="font-semibold text-green-600 dark:text-green-400">3.</span>
                <span>Simply restart your AI client - no manual configuration needed</span>
              </li>
              <li className="flex items-start gap-2">
                <span className="font-semibold text-green-600 dark:text-green-400">4.</span>
                <span>Tools become immediately available in your AI assistant</span>
              </li>
            </ol>
          </div>
        </CardContent>
      </Card>

      {/* Manual Configuration */}
      <Card className="border-2 border-orange-200 dark:border-orange-800 bg-gradient-to-br from-orange-50 to-yellow-50 dark:from-orange-950/30 dark:to-yellow-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-orange-800 dark:text-orange-300">
            <Wrench className="h-6 w-6" />
            Manual Client Configuration
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <Alert className="bg-orange-50 dark:bg-orange-950/30 border-orange-200 dark:border-orange-800">
            <Settings className="h-4 w-4" />
            <AlertDescription>
              <strong>For Non-Official Clients:</strong> If you're using an MCP-compatible client that's not officially supported, you'll need to manually add the MCP configuration JSON to your client's settings.
            </AlertDescription>
          </Alert>
          
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="space-y-4">
              <h4 className="font-semibold text-orange-800 dark:text-orange-300">Manual Setup Steps</h4>
              <ol className="space-y-3">
                <li className="flex items-start gap-3">
                  <div className="w-6 h-6 bg-orange-100 dark:bg-orange-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                    1
                  </div>
                  <div>
                    <p className="font-medium text-sm">Get Configuration JSON</p>
                    <p className="text-xs text-muted-foreground">Copy the MCP configuration from MCP Desk</p>
                  </div>
                </li>
                <li className="flex items-start gap-3">
                  <div className="w-6 h-6 bg-orange-100 dark:bg-orange-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                    2
                  </div>
                  <div>
                    <p className="font-medium text-sm">Locate Client Config</p>
                    <p className="text-xs text-muted-foreground">Find your AI client's MCP configuration file</p>
                  </div>
                </li>
                <li className="flex items-start gap-3">
                  <div className="w-6 h-6 bg-orange-100 dark:bg-orange-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                    3
                  </div>
                  <div>
                    <p className="font-medium text-sm">Paste Configuration</p>
                    <p className="text-xs text-muted-foreground">Add the JSON to your client's config file</p>
                  </div>
                </li>
                <li className="flex items-start gap-3">
                  <div className="w-6 h-6 bg-orange-100 dark:bg-orange-900/30 rounded-full flex items-center justify-center text-xs font-semibold mt-0.5">
                    4
                  </div>
                  <div>
                    <p className="font-medium text-sm">Restart Client</p>
                    <p className="text-xs text-muted-foreground">Restart your AI client to apply changes</p>
                  </div>
                </li>
              </ol>
            </div>
            
            <div className="space-y-4">
              <h4 className="font-semibold text-orange-800 dark:text-orange-300">Sample Configuration</h4>
              <div className="bg-slate-900 text-slate-100 p-4 rounded-lg text-xs font-mono">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-orange-400">MCP Configuration JSON</span>
                  <Button size="sm" variant="outline" className="h-6 text-xs">
                    <Copy className="h-3 w-3 mr-1" />
                    Copy
                  </Button>
                </div>
                <pre className="whitespace-pre-wrap text-xs">
{`{
  "mcpServers": {
    "kimbap-mcp-desk": {
      "command": "npx",
      "args": ["-y", "@kimbap-io/mcp-client"],
      "env": {
        "DESK_HOST": "localhost:3001",
        "AUTH_TOKEN": "your-access-token-here"
      }
    }
  }
}`}
                </pre>
              </div>
              
              <Alert className="bg-amber-50 dark:bg-amber-950/30 border-amber-200 dark:border-amber-800">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription className="text-xs">
                  <strong>Note:</strong> The exact configuration format may vary depending on your AI client. Check your client's MCP documentation for specific requirements.
                </AlertDescription>
              </Alert>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}