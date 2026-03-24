import { AlertTriangle, CheckCircle, XCircle, HelpCircle, Terminal, Shield, Network, Database, Server, Monitor, Key, Globe, RefreshCw, Search, Settings, MessageSquare, Book, Copy, ChevronRight } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function TroubleshootingSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-orange-500 to-red-600 rounded-2xl mb-4">
          <AlertTriangle className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">Troubleshooting & Support</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Comprehensive troubleshooting guide for common issues, debugging techniques, and support resources for Kimbap.io platform.
        </p>
      </div>

      {/* Quick Diagnostics */}
      <Card className="border-2 border-orange-200 dark:border-orange-800 bg-gradient-to-br from-orange-50 to-amber-50 dark:from-orange-950/30 dark:to-amber-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Search className="h-5 w-5 text-orange-600 dark:text-orange-400" />
            Quick Diagnostics
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Run these diagnostic commands to quickly identify common issues:
          </p>
          
          <div className="bg-slate-900 rounded-lg p-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-slate-300 text-xs">Diagnostic Commands</span>
              <Button size="sm" variant="ghost" className="h-6 text-xs text-slate-300 hover:text-white">
                <Copy className="h-3 w-3 mr-1" />
                Copy All
              </Button>
            </div>
            <div className="text-slate-100 font-mono text-xs space-y-2">
              <div className="text-yellow-400"># Check system status</div>
              <div className="text-green-400">kimbapio status --all</div>
              <div></div>
              <div className="text-yellow-400"># Test database connection</div>
              <div className="text-green-400">kimbapio test database</div>
              <div></div>
              <div className="text-yellow-400"># Verify network connectivity</div>
              <div className="text-green-400">kimbapio test network</div>
              <div></div>
              <div className="text-yellow-400"># Check service health</div>
              <div className="text-green-400">curl -f http://localhost:3000/health</div>
              <div></div>
              <div className="text-yellow-400"># View recent logs</div>
              <div className="text-green-400">kimbapio logs --tail 100 --follow</div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Common Issues by Category */}
      <div>
        <h2 className="text-xl font-semibold mb-4">Common Issues & Solutions</h2>
        
        <Tabs defaultValue="connection" className="w-full">
          <TabsList className="grid w-full grid-cols-5">
            <TabsTrigger value="connection" className="text-xs">Connection</TabsTrigger>
            <TabsTrigger value="authentication" className="text-xs">Auth</TabsTrigger>
            <TabsTrigger value="performance" className="text-xs">Performance</TabsTrigger>
            <TabsTrigger value="tools" className="text-xs">Tools</TabsTrigger>
            <TabsTrigger value="deployment" className="text-xs">Deployment</TabsTrigger>
          </TabsList>
          
          {/* Connection Issues */}
          <TabsContent value="connection" className="space-y-4">
            <Card className="border-red-200 dark:border-red-800">
              <CardHeader>
                <CardTitle className="text-lg flex items-center gap-2">
                  <Network className="h-5 w-5 text-red-600 dark:text-red-400" />
                  Connection Issues
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Issue 1: Cannot Connect to Server */}
                <div className="border rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <XCircle className="h-5 w-5 text-red-600 dark:text-red-400 mt-0.5" />
                    <div className="flex-1">
                      <h4 className="font-semibold text-sm mb-2">Cannot Connect to MCP Server</h4>
                      
                      <div className="bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded p-3 mb-3">
                        <p className="text-xs font-medium text-red-900 dark:text-red-200 mb-2">Error Messages:</p>
                        <ul className="text-xs text-red-800 dark:text-red-300 space-y-1">
                          <li>• "Connection refused"</li>
                          <li>• "Unable to reach server"</li>
                          <li>• "ERR_CONNECTION_TIMED_OUT"</li>
                        </ul>
                      </div>
                      
                      <div className="bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded p-3">
                        <p className="text-xs font-medium text-green-900 dark:text-green-200 mb-2">Solutions:</p>
                        <ol className="text-xs text-green-800 dark:text-green-300 space-y-2">
                          <li>
                            <strong>1. Check Server Status:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1 font-mono">
                              systemctl status kimbapio-core
                            </div>
                          </li>
                          <li>
                            <strong>2. Verify Network Connectivity:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1 font-mono">
                              ping your-server.com
                            </div>
                          </li>
                          <li>
                            <strong>3. Check Firewall Rules:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1 font-mono">
                              sudo ufw status
                            </div>
                          </li>
                          <li>
                            <strong>4. Verify SSL Certificate:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1 font-mono">
                              openssl s_client -connect your-server.com:443
                            </div>
                          </li>
                        </ol>
                      </div>
                    </div>
                  </div>
                </div>
                
                {/* Issue 2: Connection Drops */}
                <div className="border rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <AlertTriangle className="h-5 w-5 text-orange-600 dark:text-orange-400 mt-0.5" />
                    <div className="flex-1">
                      <h4 className="font-semibold text-sm mb-2">Frequent Connection Drops</h4>
                      
                      <div className="bg-orange-50 dark:bg-orange-950/30 border border-orange-200 dark:border-orange-800 rounded p-3 mb-3">
                        <p className="text-xs font-medium text-orange-900 dark:text-orange-200 mb-2">Symptoms:</p>
                        <ul className="text-xs text-orange-800 dark:text-orange-300 space-y-1">
                          <li>• Connection lost every few minutes</li>
                          <li>• "Reconnecting..." messages</li>
                          <li>• Intermittent timeouts</li>
                        </ul>
                      </div>
                      
                      <div className="bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 rounded p-3">
                        <p className="text-xs font-medium text-blue-900 dark:text-blue-200 mb-2">Solutions:</p>
                        <ol className="text-xs text-blue-800 dark:text-blue-300 space-y-2">
                          <li>
                            <strong>1. Increase Keep-Alive Interval:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1">
                              Edit config: <code>keep_alive_interval: 30</code>
                            </div>
                          </li>
                          <li>
                            <strong>2. Check Network Stability:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1 font-mono">
                              mtr -r your-server.com
                            </div>
                          </li>
                          <li>
                            <strong>3. Adjust Timeout Settings:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1">
                              Set <code>connection_timeout: 60</code>
                            </div>
                          </li>
                        </ol>
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
          
          {/* Authentication Issues */}
          <TabsContent value="authentication" className="space-y-4">
            <Card className="border-yellow-200 dark:border-yellow-800">
              <CardHeader>
                <CardTitle className="text-lg flex items-center gap-2">
                  <Key className="h-5 w-5 text-yellow-600 dark:text-yellow-400" />
                  Authentication Issues
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Issue: Invalid Token */}
                <div className="border rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <XCircle className="h-5 w-5 text-red-600 dark:text-red-400 mt-0.5" />
                    <div className="flex-1">
                      <h4 className="font-semibold text-sm mb-2">Invalid or Expired Token</h4>
                      
                      <div className="bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded p-3 mb-3">
                        <p className="text-xs font-medium text-red-900 dark:text-red-200 mb-2">Error Messages:</p>
                        <ul className="text-xs text-red-800 dark:text-red-300 space-y-1">
                          <li>• "Invalid authentication token"</li>
                          <li>• "Token has expired"</li>
                          <li>• "Unauthorized access"</li>
                        </ul>
                      </div>
                      
                      <div className="bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded p-3">
                        <p className="text-xs font-medium text-green-900 dark:text-green-200 mb-2">Solutions:</p>
                        <ol className="text-xs text-green-800 dark:text-green-300 space-y-2">
                          <li><strong>1.</strong> Request new token from administrator</li>
                          <li><strong>2.</strong> Check token format (should start with pk_)</li>
                          <li><strong>3.</strong> Verify token hasn't been revoked</li>
                          <li><strong>4.</strong> Ensure system time is synchronized</li>
                        </ol>
                      </div>
                    </div>
                  </div>
                </div>
                
                {/* Issue: Permission Denied */}
                <div className="border rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <Shield className="h-5 w-5 text-orange-600 dark:text-orange-400 mt-0.5" />
                    <div className="flex-1">
                      <h4 className="font-semibold text-sm mb-2">Permission Denied</h4>
                      
                      <div className="bg-orange-50 dark:bg-orange-950/30 border border-orange-200 dark:border-orange-800 rounded p-3 mb-3">
                        <p className="text-xs font-medium text-orange-900 dark:text-orange-200 mb-2">Common Scenarios:</p>
                        <ul className="text-xs text-orange-800 dark:text-orange-300 space-y-1">
                          <li>• Cannot access certain tools</li>
                          <li>• Unable to modify settings</li>
                          <li>• "Insufficient privileges" error</li>
                        </ul>
                      </div>
                      
                      <div className="bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 rounded p-3">
                        <p className="text-xs font-medium text-blue-900 dark:text-blue-200 mb-2">Resolution Steps:</p>
                        <ol className="text-xs text-blue-800 dark:text-blue-300 space-y-2">
                          <li><strong>1.</strong> Verify user role and permissions</li>
                          <li><strong>2.</strong> Check tenant assignment</li>
                          <li><strong>3.</strong> Review tool access policies</li>
                          <li><strong>4.</strong> Contact administrator for role upgrade</li>
                        </ol>
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
          
          {/* Performance Issues */}
          <TabsContent value="performance" className="space-y-4">
            <Card className="border-purple-200 dark:border-purple-800">
              <CardHeader>
                <CardTitle className="text-lg flex items-center gap-2">
                  <RefreshCw className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                  Performance Issues
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Slow Response Times */}
                <div className="border rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <AlertTriangle className="h-5 w-5 text-orange-600 dark:text-orange-400 mt-0.5" />
                    <div className="flex-1">
                      <h4 className="font-semibold text-sm mb-2">Slow Response Times</h4>
                      
                      <div className="bg-orange-50 dark:bg-orange-950/30 border border-orange-200 dark:border-orange-800 rounded p-3 mb-3">
                        <p className="text-xs font-medium text-orange-900 dark:text-orange-200 mb-2">Symptoms:</p>
                        <ul className="text-xs text-orange-800 dark:text-orange-300 space-y-1">
                          <li>• API calls take &gt;5 seconds</li>
                          <li>• Console loads slowly</li>
                          <li>• Tool operations timeout</li>
                        </ul>
                      </div>
                      
                      <div className="bg-purple-50 dark:bg-purple-950/30 border border-purple-200 dark:border-purple-800 rounded p-3">
                        <p className="text-xs font-medium text-purple-900 dark:text-purple-200 mb-2">Optimization Steps:</p>
                        <ol className="text-xs text-purple-800 dark:text-purple-300 space-y-2">
                          <li>
                            <strong>1. Check Resource Usage:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1 font-mono">
                              docker stats
                            </div>
                          </li>
                          <li>
                            <strong>2. Analyze Database Performance:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1 font-mono">
                              kimbapio db analyze
                            </div>
                          </li>
                          <li>
                            <strong>3. Clear Redis Cache:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1 font-mono">
                              redis-cli FLUSHDB
                            </div>
                          </li>
                          <li>
                            <strong>4. Scale Resources:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1">
                              Increase CPU/RAM allocation
                            </div>
                          </li>
                        </ol>
                      </div>
                    </div>
                  </div>
                </div>
                
                {/* High Memory Usage */}
                <div className="border rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <XCircle className="h-5 w-5 text-red-600 dark:text-red-400 mt-0.5" />
                    <div className="flex-1">
                      <h4 className="font-semibold text-sm mb-2">High Memory Usage</h4>
                      
                      <div className="bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded p-3 mb-3">
                        <p className="text-xs font-medium text-red-900 dark:text-red-200 mb-2">Indicators:</p>
                        <ul className="text-xs text-red-800 dark:text-red-300 space-y-1">
                          <li>• Memory usage &gt;90%</li>
                          <li>• Out of memory errors</li>
                          <li>• System becomes unresponsive</li>
                        </ul>
                      </div>
                      
                      <div className="bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded p-3">
                        <p className="text-xs font-medium text-green-900 dark:text-green-200 mb-2">Solutions:</p>
                        <ol className="text-xs text-green-800 dark:text-green-300 space-y-2">
                          <li><strong>1.</strong> Identify memory leaks with profiling</li>
                          <li><strong>2.</strong> Adjust container memory limits</li>
                          <li><strong>3.</strong> Enable swap memory (temporary fix)</li>
                          <li><strong>4.</strong> Implement pagination for large datasets</li>
                          <li><strong>5.</strong> Schedule regular service restarts</li>
                        </ol>
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
          
          {/* Tool Issues */}
          <TabsContent value="tools" className="space-y-4">
            <Card className="border-green-200 dark:border-green-800">
              <CardHeader>
                <CardTitle className="text-lg flex items-center gap-2">
                  <Settings className="h-5 w-5 text-green-600 dark:text-green-400" />
                  Tool Integration Issues
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Tool Not Working */}
                <div className="border rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <XCircle className="h-5 w-5 text-red-600 dark:text-red-400 mt-0.5" />
                    <div className="flex-1">
                      <h4 className="font-semibold text-sm mb-2">Tool Not Functioning</h4>
                      
                      <div className="bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded p-3 mb-3">
                        <p className="text-xs font-medium text-red-900 dark:text-red-200 mb-2">Common Issues:</p>
                        <ul className="text-xs text-red-800 dark:text-red-300 space-y-1">
                          <li>• Tool shows as "Disconnected"</li>
                          <li>• API calls fail</li>
                          <li>• "Invalid credentials" error</li>
                        </ul>
                      </div>
                      
                      <div className="bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded p-3">
                        <p className="text-xs font-medium text-green-900 dark:text-green-200 mb-2">Troubleshooting Steps:</p>
                        <ol className="text-xs text-green-800 dark:text-green-300 space-y-2">
                          <li>
                            <strong>1. Test Tool Connection:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1 font-mono">
                              kimbapio tool test [tool-name]
                            </div>
                          </li>
                          <li><strong>2.</strong> Verify API credentials are valid</li>
                          <li><strong>3.</strong> Check tool service status</li>
                          <li><strong>4.</strong> Review tool logs for errors</li>
                          <li><strong>5.</strong> Re-configure tool with fresh credentials</li>
                        </ol>
                      </div>
                    </div>
                  </div>
                </div>
                
                {/* Tool Rate Limiting */}
                <div className="border rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <AlertTriangle className="h-5 w-5 text-orange-600 dark:text-orange-400 mt-0.5" />
                    <div className="flex-1">
                      <h4 className="font-semibold text-sm mb-2">Rate Limiting Errors</h4>
                      
                      <div className="bg-orange-50 dark:bg-orange-950/30 border border-orange-200 dark:border-orange-800 rounded p-3 mb-3">
                        <p className="text-xs font-medium text-orange-900 dark:text-orange-200 mb-2">Error Messages:</p>
                        <ul className="text-xs text-orange-800 dark:text-orange-300 space-y-1">
                          <li>• "Rate limit exceeded"</li>
                          <li>• "Too many requests"</li>
                          <li>• HTTP 429 errors</li>
                        </ul>
                      </div>
                      
                      <div className="bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 rounded p-3">
                        <p className="text-xs font-medium text-blue-900 dark:text-blue-200 mb-2">Mitigation Strategies:</p>
                        <ol className="text-xs text-blue-800 dark:text-blue-300 space-y-2">
                          <li><strong>1.</strong> Implement request queuing</li>
                          <li><strong>2.</strong> Add exponential backoff</li>
                          <li><strong>3.</strong> Distribute requests across time</li>
                          <li><strong>4.</strong> Upgrade API plan if available</li>
                          <li><strong>5.</strong> Cache frequently accessed data</li>
                        </ol>
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
          
          {/* Deployment Issues */}
          <TabsContent value="deployment" className="space-y-4">
            <Card className="border-blue-200 dark:border-blue-800">
              <CardHeader>
                <CardTitle className="text-lg flex items-center gap-2">
                  <Server className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                  Deployment Issues
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Docker Issues */}
                <div className="border rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <AlertTriangle className="h-5 w-5 text-orange-600 dark:text-orange-400 mt-0.5" />
                    <div className="flex-1">
                      <h4 className="font-semibold text-sm mb-2">Docker Container Issues</h4>
                      
                      <div className="bg-orange-50 dark:bg-orange-950/30 border border-orange-200 dark:border-orange-800 rounded p-3 mb-3">
                        <p className="text-xs font-medium text-orange-900 dark:text-orange-200 mb-2">Common Problems:</p>
                        <ul className="text-xs text-orange-800 dark:text-orange-300 space-y-1">
                          <li>• Container keeps restarting</li>
                          <li>• Cannot pull images</li>
                          <li>• Port conflicts</li>
                        </ul>
                      </div>
                      
                      <div className="bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 rounded p-3">
                        <p className="text-xs font-medium text-blue-900 dark:text-blue-200 mb-2">Solutions:</p>
                        <ol className="text-xs text-blue-800 dark:text-blue-300 space-y-2">
                          <li>
                            <strong>1. Check Container Logs:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1 font-mono">
                              docker logs -f [container-name]
                            </div>
                          </li>
                          <li>
                            <strong>2. Verify Port Availability:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1 font-mono">
                              netstat -tulpn | grep :3000
                            </div>
                          </li>
                          <li>
                            <strong>3. Clean Docker Resources:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1 font-mono">
                              docker system prune -a
                            </div>
                          </li>
                        </ol>
                      </div>
                    </div>
                  </div>
                </div>
                
                {/* Database Issues */}
                <div className="border rounded-lg p-4">
                  <div className="flex items-start gap-3">
                    <Database className="h-5 w-5 text-red-600 dark:text-red-400 mt-0.5" />
                    <div className="flex-1">
                      <h4 className="font-semibold text-sm mb-2">Database Connection Failed</h4>
                      
                      <div className="bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded p-3 mb-3">
                        <p className="text-xs font-medium text-red-900 dark:text-red-200 mb-2">Error Messages:</p>
                        <ul className="text-xs text-red-800 dark:text-red-300 space-y-1">
                          <li>• "ECONNREFUSED"</li>
                          <li>• "Database does not exist"</li>
                          <li>• "Authentication failed"</li>
                        </ul>
                      </div>
                      
                      <div className="bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded p-3">
                        <p className="text-xs font-medium text-green-900 dark:text-green-200 mb-2">Resolution:</p>
                        <ol className="text-xs text-green-800 dark:text-green-300 space-y-2">
                          <li>
                            <strong>1. Test Database Connection:</strong>
                            <div className="bg-slate-100 dark:bg-slate-800 rounded p-1 mt-1 font-mono">
                              psql -h localhost -U postgres -d kimbapio
                            </div>
                          </li>
                          <li><strong>2.</strong> Verify DATABASE_URL environment variable</li>
                          <li><strong>3.</strong> Check PostgreSQL service status</li>
                          <li><strong>4.</strong> Run database migrations</li>
                        </ol>
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>

      {/* Debugging Tools */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Terminal className="h-5 w-5 text-purple-600 dark:text-purple-400" />
            Debugging Tools & Commands
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="p-3 border rounded-lg">
              <h4 className="font-semibold text-sm mb-3">System Diagnostics</h4>
              <div className="space-y-2">
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2">
                  <p className="text-xs font-medium mb-1">Check All Services:</p>
                  <code className="text-xs font-mono">kimbapio status --verbose</code>
                </div>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2">
                  <p className="text-xs font-medium mb-1">View Error Logs:</p>
                  <code className="text-xs font-mono">kimbapio logs --level error</code>
                </div>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2">
                  <p className="text-xs font-medium mb-1">Health Check:</p>
                  <code className="text-xs font-mono">curl http://localhost:3000/health</code>
                </div>
              </div>
            </div>
            
            <div className="p-3 border rounded-lg">
              <h4 className="font-semibold text-sm mb-3">Performance Analysis</h4>
              <div className="space-y-2">
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2">
                  <p className="text-xs font-medium mb-1">Resource Usage:</p>
                  <code className="text-xs font-mono">htop</code>
                </div>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2">
                  <p className="text-xs font-medium mb-1">Network Analysis:</p>
                  <code className="text-xs font-mono">netstat -tuln</code>
                </div>
                <div className="bg-slate-100 dark:bg-slate-800 rounded p-2">
                  <p className="text-xs font-medium mb-1">Database Queries:</p>
                  <code className="text-xs font-mono">kimbapio db slow-queries</code>
                </div>
              </div>
            </div>
          </div>
          
          <Alert className="bg-purple-50 dark:bg-purple-950/30 border-purple-200 dark:border-purple-800">
            <Terminal className="h-4 w-4" />
            <AlertDescription className="text-xs">
              <strong>Debug Mode:</strong> Enable detailed logging with <code>KIMBAP_DEBUG=true</code> environment variable for verbose output.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>

      {/* Log Analysis */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Search className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
            Log Analysis Guide
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="bg-slate-900 rounded-lg p-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-slate-300 text-xs">Common Log Locations</span>
            </div>
            <div className="text-slate-100 font-mono text-xs space-y-1">
              <div className="text-yellow-400"># Application Logs</div>
              <div>/var/log/kimbapio/app.log</div>
              <div></div>
              <div className="text-yellow-400"># Error Logs</div>
              <div>/var/log/kimbapio/error.log</div>
              <div></div>
              <div className="text-yellow-400"># Access Logs</div>
              <div>/var/log/kimbapio/access.log</div>
              <div></div>
              <div className="text-yellow-400"># Docker Logs</div>
              <div>docker logs [container-name]</div>
            </div>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <div className="p-3 bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded-lg">
              <h4 className="font-medium text-sm mb-2 text-red-800 dark:text-red-300">Error Patterns</h4>
              <ul className="text-xs space-y-1">
                <li>• FATAL: Critical failures</li>
                <li>• ERROR: Operation failures</li>
                <li>• ECONNREFUSED: Connection issues</li>
                <li>• TIMEOUT: Performance problems</li>
              </ul>
            </div>
            
            <div className="p-3 bg-yellow-50 dark:bg-yellow-950/30 border border-yellow-200 dark:border-yellow-800 rounded-lg">
              <h4 className="font-medium text-sm mb-2 text-yellow-800 dark:text-yellow-300">Warning Patterns</h4>
              <ul className="text-xs space-y-1">
                <li>• WARN: Non-critical issues</li>
                <li>• DEPRECATED: Outdated features</li>
                <li>• SLOW: Performance warnings</li>
                <li>• RETRY: Temporary failures</li>
              </ul>
            </div>
            
            <div className="p-3 bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded-lg">
              <h4 className="font-medium text-sm mb-2 text-green-800 dark:text-green-300">Info Patterns</h4>
              <ul className="text-xs space-y-1">
                <li>• INFO: Normal operations</li>
                <li>• SUCCESS: Completed tasks</li>
                <li>• STARTUP: Service initialization</li>
                <li>• CONFIG: Configuration loaded</li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* FAQ Section */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <HelpCircle className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            Frequently Asked Questions
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-3">
            <div className="border rounded-lg p-3">
              <div className="flex items-start gap-2">
                <ChevronRight className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                <div>
                  <p className="font-medium text-sm mb-1">How do I reset my master password?</p>
                  <p className="text-xs text-muted-foreground">
                    Master passwords cannot be reset for security reasons. You'll need to reinitialize the server with a new password. 
                    Make sure to backup your data first.
                  </p>
                </div>
              </div>
            </div>
            
            <div className="border rounded-lg p-3">
              <div className="flex items-start gap-2">
                <ChevronRight className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                <div>
                  <p className="font-medium text-sm mb-1">Why is my token not working?</p>
                  <p className="text-xs text-muted-foreground">
                    Tokens may expire, be revoked, or have insufficient permissions. Verify the token is still valid, 
                    check your role permissions, and ensure you're connecting to the correct server.
                  </p>
                </div>
              </div>
            </div>
            
            <div className="border rounded-lg p-3">
              <div className="flex items-start gap-2">
                <ChevronRight className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                <div>
                  <p className="font-medium text-sm mb-1">How can I improve performance?</p>
                  <p className="text-xs text-muted-foreground">
                    Start by checking resource usage, optimizing database queries, enabling caching, and scaling horizontally 
                    if needed. Use the performance monitoring tools to identify bottlenecks.
                  </p>
                </div>
              </div>
            </div>
            
            <div className="border rounded-lg p-3">
              <div className="flex items-start gap-2">
                <ChevronRight className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                <div>
                  <p className="font-medium text-sm mb-1">Can I migrate from another MCP platform?</p>
                  <p className="text-xs text-muted-foreground">
                    Yes, Kimbap.io supports standard MCP protocols. Export your tool configurations from your current platform 
                    and import them using our migration tools. Contact support for assistance.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Support Resources */}
      <Card className="border-2 border-green-200 dark:border-green-800 bg-gradient-to-r from-green-50 to-emerald-50 dark:from-green-950/30 dark:to-emerald-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <MessageSquare className="h-5 w-5 text-green-600 dark:text-green-400" />
            Get Support
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="text-center p-4 bg-white dark:bg-slate-800 rounded-lg border">
              <div className="w-12 h-12 bg-blue-100 dark:bg-blue-900/30 rounded-xl flex items-center justify-center mx-auto mb-3">
                <Book className="h-6 w-6 text-blue-600 dark:text-blue-400" />
              </div>
              <h4 className="font-semibold text-sm mb-2">Documentation</h4>
              <p className="text-xs text-muted-foreground mb-3">
                Comprehensive guides and API references
              </p>
              <Button size="sm" variant="outline" className="w-full text-xs">
                <Book className="h-3 w-3 mr-1" />
                View Docs
              </Button>
            </div>
            
            <div className="text-center p-4 bg-white dark:bg-slate-800 rounded-lg border">
              <div className="w-12 h-12 bg-purple-100 dark:bg-purple-900/30 rounded-xl flex items-center justify-center mx-auto mb-3">
                <MessageSquare className="h-6 w-6 text-purple-600 dark:text-purple-400" />
              </div>
              <h4 className="font-semibold text-sm mb-2">Community Forum</h4>
              <p className="text-xs text-muted-foreground mb-3">
                Get help from the Kimbap.io community
              </p>
              <Button size="sm" variant="outline" className="w-full text-xs">
                <MessageSquare className="h-3 w-3 mr-1" />
                Visit Forum
              </Button>
            </div>
            
            <div className="text-center p-4 bg-white dark:bg-slate-800 rounded-lg border">
              <div className="w-12 h-12 bg-green-100 dark:bg-green-900/30 rounded-xl flex items-center justify-center mx-auto mb-3">
                <Globe className="h-6 w-6 text-green-600 dark:text-green-400" />
              </div>
              <h4 className="font-semibold text-sm mb-2">Enterprise Support</h4>
              <p className="text-xs text-muted-foreground mb-3">
                Priority support for enterprise customers
              </p>
              <Button size="sm" className="w-full text-xs bg-green-600 hover:bg-green-700">
                <MessageSquare className="h-3 w-3 mr-1" />
                Contact Support
              </Button>
            </div>
          </div>
          
          <Alert className="mt-4 bg-blue-50 dark:bg-blue-950/30 border-blue-200 dark:border-blue-800">
            <HelpCircle className="h-4 w-4" />
            <AlertDescription className="text-xs">
              <strong>Before contacting support:</strong> Please gather system diagnostics, error logs, and steps to reproduce the issue. 
              This helps us resolve your problem faster.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    </div>
  )
}