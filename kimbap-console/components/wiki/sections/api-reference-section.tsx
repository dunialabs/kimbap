import { Code, Key, Shield, Terminal, Globe, Database, Zap, Copy, ArrowRight, CheckCircle, AlertCircle, Book, Lock, Server, Network } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function ApiReferenceSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-purple-500 to-indigo-600 rounded-2xl mb-4">
          <Code className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">API Reference</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Complete API documentation for integrating with Kimbap.io MCP platform. Build custom tools, manage servers programmatically, and extend platform capabilities.
        </p>
      </div>

      {/* API Overview */}
      <Card className="border-2 border-purple-200 dark:border-purple-800 bg-gradient-to-br from-purple-50 to-indigo-50 dark:from-purple-950/30 dark:to-indigo-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5 text-purple-600 dark:text-purple-400" />
            API Overview
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="text-center p-4 bg-white dark:bg-slate-800 rounded-lg border">
              <div className="w-12 h-12 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Server className="h-6 w-6 text-purple-600 dark:text-purple-400" />
              </div>
              <h3 className="font-semibold text-sm mb-1">RESTful API</h3>
              <p className="text-xs text-muted-foreground">Standard HTTP methods with JSON payloads</p>
            </div>
            <div className="text-center p-4 bg-white dark:bg-slate-800 rounded-lg border">
              <div className="w-12 h-12 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Key className="h-6 w-6 text-blue-600 dark:text-blue-400" />
              </div>
              <h3 className="font-semibold text-sm mb-1">Token Authentication</h3>
              <p className="text-xs text-muted-foreground">Bearer token-based secure authentication</p>
            </div>
            <div className="text-center p-4 bg-white dark:bg-slate-800 rounded-lg border">
              <div className="w-12 h-12 bg-green-100 dark:bg-green-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Zap className="h-6 w-6 text-green-600 dark:text-green-400" />
              </div>
              <h3 className="font-semibold text-sm mb-1">Rate Limiting</h3>
              <p className="text-xs text-muted-foreground">1000 requests/minute per token</p>
            </div>
          </div>
          
          <div className="bg-slate-900 rounded-lg p-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-slate-300 text-xs">Base URL</span>
              <Button size="sm" variant="ghost" className="h-6 text-xs text-slate-300 hover:text-white">
                <Copy className="h-3 w-3 mr-1" />
                Copy
              </Button>
            </div>
            <div className="text-slate-100 font-mono text-sm">
              https://your-domain.com/api/v1
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Authentication */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Key className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            Authentication
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            All API requests require authentication using a Bearer token in the Authorization header.
          </p>
          
          <Tabs defaultValue="obtaining" className="w-full">
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="obtaining">Obtaining Tokens</TabsTrigger>
              <TabsTrigger value="usage">Using Tokens</TabsTrigger>
              <TabsTrigger value="management">Token Management</TabsTrigger>
            </TabsList>
            
            <TabsContent value="obtaining" className="space-y-4">
              <div className="bg-slate-900 rounded-lg p-4">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-slate-300 text-xs">Generate API Token</span>
                  <Button size="sm" variant="ghost" className="h-6 text-xs text-slate-300 hover:text-white">
                    <Copy className="h-3 w-3 mr-1" />
                    Copy
                  </Button>
                </div>
                <div className="text-slate-100 font-mono text-xs space-y-1">
                  <div className="text-yellow-400"># Login to get session token</div>
                  <div>POST /api/v1/auth/login</div>
                  <div>Content-Type: application/json</div>
                  <div></div>
                  <div>{`{`}</div>
                  <div>  "email": "admin@company.com",</div>
                  <div>  "password": "your-password"</div>
                  <div>{`}`}</div>
                  <div></div>
                  <div className="text-yellow-400"># Response</div>
                  <div>{`{`}</div>
                  <div>  "token": "pk_live_abc123...",</div>
                  <div>  "expires_at": "2024-12-31T23:59:59Z"</div>
                  <div>{`}`}</div>
                </div>
              </div>
            </TabsContent>
            
            <TabsContent value="usage" className="space-y-4">
              <div className="bg-slate-900 rounded-lg p-4">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-slate-300 text-xs">Using Bearer Token</span>
                  <Button size="sm" variant="ghost" className="h-6 text-xs text-slate-300 hover:text-white">
                    <Copy className="h-3 w-3 mr-1" />
                    Copy
                  </Button>
                </div>
                <div className="text-slate-100 font-mono text-xs space-y-1">
                  <div className="text-yellow-400"># Include in Authorization header</div>
                  <div>GET /api/v1/servers</div>
                  <div>Authorization: Bearer pk_live_abc123...</div>
                  <div></div>
                  <div className="text-yellow-400"># cURL example</div>
                  <div>curl -H "Authorization: Bearer pk_live_abc123..." \</div>
                  <div>  https://your-domain.com/api/v1/servers</div>
                </div>
              </div>
            </TabsContent>
            
            <TabsContent value="management" className="space-y-4">
              <div className="space-y-3">
                <div className="p-3 border rounded-lg">
                  <h4 className="font-semibold text-sm mb-2">Token Types</h4>
                  <ul className="text-xs space-y-1">
                    <li>• <strong>Owner Token:</strong> Full administrative access</li>
                    <li>• <strong>Admin Token:</strong> Management capabilities</li>
                    <li>• <strong>Access Token:</strong> Standard user access</li>
                    <li>• <strong>Service Token:</strong> For automated systems</li>
                  </ul>
                </div>
                <div className="p-3 border rounded-lg">
                  <h4 className="font-semibold text-sm mb-2">Token Lifecycle</h4>
                  <ul className="text-xs space-y-1">
                    <li>• Tokens expire after 30 days by default</li>
                    <li>• Refresh tokens available for long-lived access</li>
                    <li>• Revoke tokens immediately via API or Console</li>
                    <li>• Audit log tracks all token usage</li>
                  </ul>
                </div>
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

      {/* Core Endpoints */}
      <div>
        <h2 className="text-xl font-semibold mb-4">Core API Endpoints</h2>
        
        {/* Servers API */}
        <Card className="mb-4">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <Server className="h-5 w-5 text-green-600 dark:text-green-400" />
              Servers
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-3">
              {/* List Servers */}
              <div className="border rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300">GET</Badge>
                    <code className="text-sm font-mono">/api/v1/servers</code>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground mb-2">List all MCP servers</p>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2 text-xs font-mono">
                  Response: [{` id, name, status, created_at, tools[] `}]
                </div>
              </div>
              
              {/* Create Server */}
              <div className="border rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300">POST</Badge>
                    <code className="text-sm font-mono">/api/v1/servers</code>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground mb-2">Create a new MCP server</p>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2 text-xs font-mono">
                  Body: {`{ name, description, config: {} }`}
                </div>
              </div>
              
              {/* Get Server */}
              <div className="border rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300">GET</Badge>
                    <code className="text-sm font-mono">/api/v1/servers/{`{id}`}</code>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground mb-2">Get server details</p>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2 text-xs font-mono">
                  Response: {`{ id, name, status, config, tools[], metrics }`}
                </div>
              </div>
              
              {/* Update Server */}
              <div className="border rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Badge className="bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300">PUT</Badge>
                    <code className="text-sm font-mono">/api/v1/servers/{`{id}`}</code>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground mb-2">Update server configuration</p>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2 text-xs font-mono">
                  Body: {`{ name?, description?, config?: {} }`}
                </div>
              </div>
              
              {/* Delete Server */}
              <div className="border rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300">DELETE</Badge>
                    <code className="text-sm font-mono">/api/v1/servers/{`{id}`}</code>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground mb-2">Delete a server</p>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2 text-xs font-mono">
                  Response: 204 No Content
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Tools API */}
        <Card className="mb-4">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <Zap className="h-5 w-5 text-purple-600 dark:text-purple-400" />
              Tools
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-3">
              {/* List Tools */}
              <div className="border rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300">GET</Badge>
                    <code className="text-sm font-mono">/api/v1/tools</code>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground mb-2">List available tools</p>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2 text-xs font-mono">
                  Response: [{` id, name, type, status, capabilities[] `}]
                </div>
              </div>
              
              {/* Configure Tool */}
              <div className="border rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300">POST</Badge>
                    <code className="text-sm font-mono">/api/v1/tools/{`{id}`}/configure</code>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground mb-2">Configure tool settings</p>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2 text-xs font-mono">
                  Body: {`{ credentials: {}, settings: {} }`}
                </div>
              </div>
              
              {/* Test Tool */}
              <div className="border rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300">POST</Badge>
                    <code className="text-sm font-mono">/api/v1/tools/{`{id}`}/test</code>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground mb-2">Test tool connectivity</p>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2 text-xs font-mono">
                  Response: {`{ success: boolean, message, latency_ms }`}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Users API */}
        <Card className="mb-4">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <Shield className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              Users & Access
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-3">
              {/* List Users */}
              <div className="border rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300">GET</Badge>
                    <code className="text-sm font-mono">/api/v1/users</code>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground mb-2">List all users</p>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2 text-xs font-mono">
                  Response: [{` id, email, role, status, created_at `}]
                </div>
              </div>
              
              {/* Create User */}
              <div className="border rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300">POST</Badge>
                    <code className="text-sm font-mono">/api/v1/users</code>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground mb-2">Create new user</p>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2 text-xs font-mono">
                  Body: {`{ email, role, tenant_id?, permissions[] }`}
                </div>
              </div>
              
              {/* Generate Token */}
              <div className="border rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300">POST</Badge>
                    <code className="text-sm font-mono">/api/v1/users/{`{id}`}/tokens</code>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground mb-2">Generate access token for user</p>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2 text-xs font-mono">
                  Response: {`{ token, expires_at, permissions[] }`}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Webhooks */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Network className="h-5 w-5 text-orange-600 dark:text-orange-400" />
            Webhooks & Events
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Subscribe to real-time events using webhooks for automated workflows and monitoring.
          </p>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="p-3 border rounded-lg">
              <h4 className="font-semibold text-sm mb-2">Available Events</h4>
              <ul className="text-xs space-y-1">
                <li>• <code>server.created</code> - New server initialized</li>
                <li>• <code>server.status_changed</code> - Server online/offline</li>
                <li>• <code>tool.configured</code> - Tool settings updated</li>
                <li>• <code>tool.error</code> - Tool connection failed</li>
                <li>• <code>user.created</code> - New user added</li>
                <li>• <code>token.created</code> - Access token generated</li>
                <li>• <code>usage.threshold</code> - Usage limit reached</li>
              </ul>
            </div>
            
            <div className="p-3 border rounded-lg">
              <h4 className="font-semibold text-sm mb-2">Webhook Configuration</h4>
              <div className="bg-slate-900 rounded p-2">
                <div className="text-slate-100 font-mono text-xs space-y-1">
                  <div>POST /api/v1/webhooks</div>
                  <div>{`{`}</div>
                  <div>  "url": "https://your-app.com/webhook",</div>
                  <div>  "events": ["server.*", "tool.error"],</div>
                  <div>  "secret": "webhook_secret_key"</div>
                  <div>{`}`}</div>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Error Handling */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
            Error Handling
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-3">
            <div className="p-3 bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded-lg">
              <div className="flex items-start gap-2">
                <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300">400</Badge>
                <div>
                  <p className="font-medium text-sm">Bad Request</p>
                  <p className="text-xs text-muted-foreground">Invalid parameters or malformed request</p>
                </div>
              </div>
            </div>
            
            <div className="p-3 bg-orange-50 dark:bg-orange-950/30 border border-orange-200 dark:border-orange-800 rounded-lg">
              <div className="flex items-start gap-2">
                <Badge className="bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300">401</Badge>
                <div>
                  <p className="font-medium text-sm">Unauthorized</p>
                  <p className="text-xs text-muted-foreground">Missing or invalid authentication token</p>
                </div>
              </div>
            </div>
            
            <div className="p-3 bg-yellow-50 dark:bg-yellow-950/30 border border-yellow-200 dark:border-yellow-800 rounded-lg">
              <div className="flex items-start gap-2">
                <Badge className="bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300">403</Badge>
                <div>
                  <p className="font-medium text-sm">Forbidden</p>
                  <p className="text-xs text-muted-foreground">Insufficient permissions for requested resource</p>
                </div>
              </div>
            </div>
            
            <div className="p-3 bg-gray-50 dark:bg-gray-950/50 border border-gray-200 dark:border-gray-800 rounded-lg">
              <div className="flex items-start gap-2">
                <Badge variant="outline">404</Badge>
                <div>
                  <p className="font-medium text-sm">Not Found</p>
                  <p className="text-xs text-muted-foreground">Requested resource does not exist</p>
                </div>
              </div>
            </div>
            
            <div className="p-3 bg-purple-50 dark:bg-purple-950/30 border border-purple-200 dark:border-purple-800 rounded-lg">
              <div className="flex items-start gap-2">
                <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300">429</Badge>
                <div>
                  <p className="font-medium text-sm">Rate Limited</p>
                  <p className="text-xs text-muted-foreground">Too many requests - check X-RateLimit headers</p>
                </div>
              </div>
            </div>
          </div>
          
          <div className="bg-slate-100 dark:bg-slate-800 rounded-lg p-3">
            <p className="text-xs font-mono mb-2">Error Response Format:</p>
            <div className="bg-white dark:bg-slate-800 rounded p-2 font-mono text-xs">
              {`{`}<br/>
              {`  "error": {`}<br/>
              {`    "code": "invalid_parameter",`}<br/>
              {`    "message": "The 'name' field is required",`}<br/>
              {`    "details": { "field": "name" }`}<br/>
              {`  }`}<br/>
              {`}`}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* SDKs and Libraries */}
      <Card className="border-2 border-indigo-200 dark:border-indigo-800 bg-gradient-to-r from-indigo-50 to-purple-50 dark:from-indigo-950/30 dark:to-purple-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Book className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
            SDKs & Client Libraries
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="p-3 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded-lg">
              <h4 className="font-semibold text-sm mb-2">JavaScript/TypeScript</h4>
              <div className="bg-slate-100 dark:bg-slate-800 rounded p-2 font-mono text-xs mb-2">
                npm install @kimbapio/sdk
              </div>
              <Button size="sm" variant="outline" className="w-full text-xs">
                <ArrowRight className="h-3 w-3 mr-1" />
                View Documentation
              </Button>
            </div>
            
            <div className="p-3 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded-lg">
              <h4 className="font-semibold text-sm mb-2">Python</h4>
              <div className="bg-slate-100 dark:bg-slate-800 rounded p-2 font-mono text-xs mb-2">
                pip install kimbapio-sdk
              </div>
              <Button size="sm" variant="outline" className="w-full text-xs">
                <ArrowRight className="h-3 w-3 mr-1" />
                View Documentation
              </Button>
            </div>
            
            <div className="p-3 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded-lg">
              <h4 className="font-semibold text-sm mb-2">Go</h4>
              <div className="bg-slate-100 dark:bg-slate-800 rounded p-2 font-mono text-xs mb-2">
                go get github.com/kimbapio/go-sdk
              </div>
              <Button size="sm" variant="outline" className="w-full text-xs">
                <ArrowRight className="h-3 w-3 mr-1" />
                View Documentation
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

    </div>
  )
}
