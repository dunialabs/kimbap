import { Users, UserPlus, Shield, Key, Crown, User, UserCheck, Plus, Monitor, Code, Settings, Clock, Eye, EyeOff, Trash2 } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

export function MemberManagementSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-blue-500 to-purple-600 rounded-2xl mb-4">
          <Key className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">Access Token Management</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Create and manage access tokens for users and agents to connect to your MCP server.
        </p>
      </div>

      {/* Usage Scenarios */}
      <Card className="border-2 border-blue-200 dark:border-blue-800 bg-gradient-to-br from-blue-50 to-indigo-50 dark:from-blue-950/30 dark:to-indigo-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-blue-800 dark:text-blue-300">
            <Code className="h-5 w-5" />
            Access Token Usage Scenarios
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-4">
              <div className="flex items-start gap-3">
                <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center">
                  <Monitor className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                </div>
                <div>
                  <h4 className="font-semibold text-blue-800 dark:text-blue-300 mb-2">MCP Desk Application</h4>
                  <p className="text-sm text-blue-700 dark:text-blue-300">
                    End users can use access tokens in the MCP Desk desktop application to connect their AI assistants 
                    (like Claude or Cursor) to your MCP server and access configured tools.
                  </p>
                </div>
              </div>
            </div>
            
            <div className="space-y-4">
              <div className="flex items-start gap-3">
                <div className="w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center">
                  <Code className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                </div>
                <div>
                  <h4 className="font-semibold text-purple-800 dark:text-purple-300 mb-2">Agent API Calls</h4>
                  <p className="text-sm text-purple-700 dark:text-purple-300">
                    Automated agents and services can use access tokens in API calls to programmatically access 
                    MCP tools and integrate with your server infrastructure.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* User Roles */}
      <Card className="border-2 border-slate-200 dark:border-slate-700">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            User Roles
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <Card className="border-yellow-200 dark:border-yellow-800 bg-gradient-to-br from-yellow-50 to-yellow-100 dark:from-yellow-950/30 dark:to-yellow-900/30">
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-yellow-800 dark:text-yellow-300">
                  <Crown className="h-5 w-5" />
                  Owner
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-yellow-700 dark:text-yellow-300 mb-3">
                  Full administrative access to all server functions
                </p>
                <ul className="text-xs space-y-1 text-yellow-800 dark:text-yellow-300">
                  <li>✓ User management</li>
                  <li>✓ Tool configuration</li>
                  <li>✓ Server settings</li>
                  <li>✓ Billing access</li>
                </ul>
              </CardContent>
            </Card>

            <Card className="border-blue-200 dark:border-blue-800 bg-gradient-to-br from-blue-50 to-blue-100 dark:from-blue-950/30 dark:to-blue-900/30">
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-blue-800 dark:text-blue-300">
                  <UserCheck className="h-5 w-5" />
                  Admin
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-blue-700 dark:text-blue-300 mb-3">
                  Administrative access for day-to-day management
                </p>
                <ul className="text-xs space-y-1 text-blue-800 dark:text-blue-300">
                  <li>✓ User management</li>
                  <li>✓ Tool configuration</li>
                  <li>✓ Usage analytics</li>
                  <li>✗ Server deletion</li>
                </ul>
              </CardContent>
            </Card>

            <Card className="border-green-200 dark:border-green-800 bg-gradient-to-br from-green-50 to-green-100 dark:from-green-950/30 dark:to-green-900/30">
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-green-800 dark:text-green-300">
                  <User className="h-5 w-5" />
                  Member
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-green-700 dark:text-green-300 mb-3">
                  Access to use MCP tools through AI clients
                </p>
                <ul className="text-xs space-y-1 text-green-800 dark:text-green-300">
                  <li>✓ Tool usage</li>
                  <li>✓ Personal analytics</li>
                  <li>✗ User management</li>
                  <li>✗ Configuration</li>
                </ul>
                <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs mt-2">Default Role</Badge>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>

      {/* Create Access Tokens */}
      <Card className="border-2 border-green-200 dark:border-green-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-green-800 dark:text-green-300">
            <Key className="h-5 w-5" />
            Create Access Tokens
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="text-center">
              <div className="w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto mb-2">
                <span className="font-bold text-green-600 dark:text-green-400">1</span>
              </div>
              <p className="font-medium text-sm">Generate Token</p>
              <p className="text-xs text-muted-foreground">Click "Generate Access Token"</p>
            </div>
            <div className="text-center">
              <div className="w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto mb-2">
                <span className="font-bold text-green-600 dark:text-green-400">2</span>
              </div>
              <p className="font-medium text-sm">Copy & Secure</p>
              <p className="text-xs text-muted-foreground">Save token securely - it won't be shown again</p>
            </div>
            <div className="text-center">
              <div className="w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto mb-2">
                <span className="font-bold text-green-600 dark:text-green-400">3</span>
              </div>
              <p className="font-medium text-sm">Distribute</p>
              <p className="text-xs text-muted-foreground">Share with users/agents offline</p>
            </div>
          </div>
          
          <div className="text-center pt-4">
            <Button className="bg-green-600 hover:bg-green-700">
              <Plus className="h-4 w-4 mr-2" />
              Generate Access Token
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Access Tokens */}
      <Card className="border-2 border-purple-200 dark:border-purple-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-purple-800 dark:text-purple-300">
            <Key className="h-5 w-5" />
            Access Tokens
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <Alert className="border-amber-200 bg-amber-50 dark:bg-amber-950/30 dark:border-amber-800">
            <Shield className="h-4 w-4" />
            <AlertDescription>
              <strong>Token Security:</strong> Access tokens provide direct server access. Keep them secure and never share publicly.
            </AlertDescription>
          </Alert>

          {/* Tool-specific Configuration */}
          <div className="space-y-4">
            <h4 className="font-semibold text-purple-800 dark:text-purple-300 flex items-center gap-2">
              <Settings className="h-4 w-4" />
              Tool-specific Configuration
            </h4>
            <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
              <CardContent className="pt-4">
                <p className="text-sm text-blue-800 dark:text-blue-300 mb-3">
                  Access tokens can be configured to grant access to specific tools only, providing fine-grained control over what resources each token can access.
                </p>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  <div className="bg-white dark:bg-slate-800 p-3 rounded border dark:border-slate-700">
                    <div className="flex items-center gap-2 mb-2">
                      <Code className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                      <span className="text-xs font-medium">GitHub Integration</span>
                      <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Allowed</Badge>
                    </div>
                    <p className="text-xs text-muted-foreground">Repository read/write access</p>
                  </div>
                  <div className="bg-white dark:bg-slate-800 p-3 rounded border dark:border-slate-700">
                    <div className="flex items-center gap-2 mb-2">
                      <Code className="h-3 w-3 text-gray-600 dark:text-gray-400" />
                      <span className="text-xs font-medium">Database Access</span>
                      <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Restricted</Badge>
                    </div>
                    <p className="text-xs text-muted-foreground">No database permissions</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Security Features */}
          <div className="space-y-4">
            <h4 className="font-semibold text-purple-800 dark:text-purple-300 flex items-center gap-2">
              <Shield className="h-4 w-4" />
              Security Features
            </h4>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Card className="border-orange-200 bg-orange-50 dark:bg-orange-950/30 dark:border-orange-800">
                <CardContent className="pt-4">
                  <div className="text-center">
                    <Clock className="h-6 w-6 mx-auto mb-2 text-orange-600 dark:text-orange-400" />
                    <p className="font-medium text-sm text-orange-800 dark:text-orange-300">Token Expiry</p>
                    <p className="text-xs text-orange-700 dark:text-orange-300 mt-1">
                      Tokens can be set to expire after a specified time period for enhanced security
                    </p>
                  </div>
                </CardContent>
              </Card>

              <Card className="border-red-200 bg-red-50 dark:bg-red-950/30 dark:border-red-800">
                <CardContent className="pt-4">
                  <div className="text-center">
                    <Trash2 className="h-6 w-6 mx-auto mb-2 text-red-600 dark:text-red-400" />
                    <p className="font-medium text-sm text-red-800 dark:text-red-300">Instant Revocation</p>
                    <p className="text-xs text-red-700 dark:text-red-300 mt-1">
                      Tokens can be immediately revoked and invalidated when no longer needed
                    </p>
                  </div>
                </CardContent>
              </Card>

              <Card className="border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-950/50">
                <CardContent className="pt-4">
                  <div className="text-center">
                    <EyeOff className="h-6 w-6 mx-auto mb-2 text-gray-600 dark:text-gray-400" />
                    <p className="font-medium text-sm text-gray-800 dark:text-gray-200">One-time View</p>
                    <p className="text-xs text-gray-700 dark:text-gray-300 mt-1">
                      Generated tokens are shown only once and cannot be retrieved again
                    </p>
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>

          {/* Generate Token Section */}
          <div className="text-center pt-4 border-t">
            <Button className="bg-purple-600 hover:bg-purple-700">
              <Key className="h-4 w-4 mr-2" />
              Generate New Access Token
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Best Practices */}
      <Card className="bg-gradient-to-r from-indigo-50 dark:from-indigo-950 to-purple-50 dark:to-purple-950 border-indigo-200 dark:border-indigo-800">
        <CardHeader>
          <CardTitle className="text-indigo-800 dark:text-indigo-300">Access Token Management Best Practices</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-3">
              <h4 className="font-semibold text-indigo-800 dark:text-indigo-300">Security Guidelines</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <Shield className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Use Appropriate Roles</p>
                    <p className="text-xs text-muted-foreground">Grant minimum necessary permissions</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Key className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Secure Token Storage</p>
                    <p className="text-xs text-muted-foreground">Keep access tokens private and secure</p>
                  </div>
                </li>
              </ul>
            </div>
            
            <div className="space-y-3">
              <h4 className="font-semibold text-indigo-800 dark:text-indigo-300">Token Management</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <Key className="h-4 w-4 text-orange-600 dark:text-orange-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Regular Token Audits</p>
                    <p className="text-xs text-muted-foreground">Review and rotate access tokens periodically</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Shield className="h-4 w-4 text-purple-600 dark:text-purple-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Secure Distribution</p>
                    <p className="text-xs text-muted-foreground">Share tokens through secure channels only</p>
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