import { Settings, Plus } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"

export function ToolConfigurationSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-emerald-500 to-cyan-600 rounded-2xl mb-4">
          <Settings className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">Tool Management</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Add and configure tools to extend your MCP server capabilities. Connect popular services like GitHub, databases, and APIs.
        </p>
      </div>

      {/* Simple Overview */}
      <Alert className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
        <Settings className="h-4 w-4" />
        <AlertDescription>
          <strong>Tools</strong> are integrations that connect your MCP server to external services like GitHub, databases, and APIs.
        </AlertDescription>
      </Alert>

      {/* Popular Tools */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card className="border-gray-200 dark:border-gray-800 hover:shadow-md transition-shadow">
          <CardContent className="p-4">
            <div className="flex items-center gap-3 mb-2">
              <div className="w-8 h-8 bg-gray-900 rounded flex items-center justify-center">
                <span className="text-white text-xs font-bold">GH</span>
              </div>
              <div>
                <h3 className="font-semibold">GitHub</h3>
                <p className="text-xs text-muted-foreground">Repository management</p>
              </div>
            </div>
            <Button size="sm" className="w-full">Add GitHub</Button>
          </CardContent>
        </Card>

        <Card className="border-blue-200 dark:border-blue-800 hover:shadow-md transition-shadow">
          <CardContent className="p-4">
            <div className="flex items-center gap-3 mb-2">
              <div className="w-8 h-8 bg-blue-600 rounded flex items-center justify-center">
                <span className="text-white text-xs font-bold">DB</span>
              </div>
              <div>
                <h3 className="font-semibold">PostgreSQL</h3>
                <p className="text-xs text-muted-foreground">Database queries</p>
              </div>
            </div>
            <Button size="sm" variant="outline" className="w-full">Add Database</Button>
          </CardContent>
        </Card>

        <Card className="border-black hover:shadow-md transition-shadow">
          <CardContent className="p-4">
            <div className="flex items-center gap-3 mb-2">
              <div className="w-8 h-8 bg-black rounded flex items-center justify-center">
                <span className="text-white text-xs font-bold">N</span>
              </div>
              <div>
                <h3 className="font-semibold">Notion</h3>
                <p className="text-xs text-muted-foreground">Knowledge base</p>
              </div>
            </div>
            <Button size="sm" variant="outline" className="w-full">Add Notion</Button>
          </CardContent>
        </Card>
      </div>

      {/* Simple How-To */}
      <Card className="border-2 border-green-200 dark:border-green-800 bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950/30 dark:to-emerald-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Plus className="h-5 w-5 text-green-600 dark:text-green-400" />
            How to Add Tools
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="text-center">
              <div className="w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto mb-2">
                <span className="font-bold text-green-600 dark:text-green-400">1</span>
              </div>
              <p className="font-medium text-sm">Click "Add Tool"</p>
              <p className="text-xs text-muted-foreground">Choose from available integrations</p>
            </div>
            <div className="text-center">
              <div className="w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto mb-2">
                <span className="font-bold text-green-600 dark:text-green-400">2</span>
              </div>
              <p className="font-medium text-sm">Enter Credentials</p>
              <p className="text-xs text-muted-foreground">Provide API keys or tokens</p>
            </div>
            <div className="text-center">
              <div className="w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto mb-2">
                <span className="font-bold text-green-600 dark:text-green-400">3</span>
              </div>
              <p className="font-medium text-sm">Test & Save</p>
              <p className="text-xs text-muted-foreground">Verify connection works</p>
            </div>
          </div>
          <div className="text-center pt-4">
            <Button className="bg-green-600 hover:bg-green-700">
              <Plus className="h-4 w-4 mr-2" />
              Add Your First Tool
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Best Practices */}
      <Card className="bg-gradient-to-r from-emerald-50 dark:from-emerald-950 to-cyan-50 dark:to-cyan-950 border-emerald-200 dark:border-emerald-800">
        <CardHeader>
          <CardTitle className="text-emerald-800 dark:text-emerald-300">Tool Configuration Best Practices</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-3">
              <h4 className="font-semibold text-emerald-800 dark:text-emerald-300">Security Guidelines</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <Settings className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Use Service Accounts</p>
                    <p className="text-xs text-muted-foreground">Create dedicated accounts for MCP integrations</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Settings className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Test Connections</p>
                    <p className="text-xs text-muted-foreground">Verify tools work before deployment</p>
                  </div>
                </li>
              </ul>
            </div>
            
            <div className="space-y-3">
              <h4 className="font-semibold text-emerald-800 dark:text-emerald-300">Performance Tips</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <Settings className="h-4 w-4 text-purple-600 dark:text-purple-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Monitor Usage</p>
                    <p className="text-xs text-muted-foreground">Check tool performance regularly</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Settings className="h-4 w-4 text-orange-600 dark:text-orange-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Manage API Limits</p>
                    <p className="text-xs text-muted-foreground">Stay within service rate limits</p>
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