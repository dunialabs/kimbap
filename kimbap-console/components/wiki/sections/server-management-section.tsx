import { Server, Settings, Shield, Key, Crown, Database, Monitor, Activity, RefreshCw, Power, AlertCircle, CheckCircle, AlertTriangle, Eye, Download, Upload, Copy, Bell, Zap, Clock, Users, Network, Lock, Terminal, Code, Play, Pause, Square, Trash2, Edit, Plus, ArrowRight } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function ServerManagementSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-red-500 to-orange-600 rounded-2xl mb-4">
          <Server className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">Server Management Guide</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Complete administrative control over your MCP server infrastructure. Monitor performance, 
          manage resources, configure advanced settings, and maintain optimal system health.
        </p>
      </div>

      {/* Server Overview Dashboard */}
      <Card className="border-2 border-slate-200 dark:border-slate-700">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Monitor className="h-5 w-5" />
            Server Overview Dashboard
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-800 dark:to-slate-900 rounded-lg p-6 border border-slate-200 dark:border-slate-700">
            <div className="text-center space-y-4">
              <p className="text-lg font-semibold text-slate-700 dark:text-slate-300">Server Monitoring Interface</p>
              <div className="bg-white dark:bg-slate-800 rounded-lg p-6 border-2 border-dashed border-slate-300 dark:border-slate-600">
                <p className="text-muted-foreground mb-2">[Server Dashboard Screenshot]</p>
                <p className="text-sm text-muted-foreground">
                  Add comprehensive server dashboard showing: System metrics, Active connections, 
                  Resource usage, Service health, Recent activities, Performance graphs
                </p>
              </div>
              <Button variant="outline" size="sm">
                <Monitor className="h-4 w-4 mr-2" />
                View Live Dashboard Tour
              </Button>
            </div>
          </div>

          {/* Server Status Grid */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
              <CardContent className="pt-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <div className="w-3 h-3 bg-green-500 rounded-full animate-pulse"></div>
                    <span className="font-medium text-green-800 dark:text-green-300">Production Server</span>
                  </div>
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300">Online</Badge>
                </div>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span>Uptime:</span>
                    <span className="font-mono text-green-600 dark:text-green-400">99.97%</span>
                  </div>
                  <div className="flex justify-between">
                    <span>CPU Usage:</span>
                    <span className="font-mono text-green-600 dark:text-green-400">23%</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Active Sessions:</span>
                    <span className="font-mono text-green-600 dark:text-green-400">147</span>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
              <CardContent className="pt-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <Database className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                    <span className="font-medium text-blue-800 dark:text-blue-300">Database</span>
                  </div>
                  <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300">Healthy</Badge>
                </div>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span>Connections:</span>
                    <span className="font-mono text-blue-600 dark:text-blue-400">45/100</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Query Time:</span>
                    <span className="font-mono text-blue-600 dark:text-blue-400">12ms</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Storage:</span>
                    <span className="font-mono text-blue-600 dark:text-blue-400">2.4GB</span>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="border-orange-200 bg-orange-50 dark:bg-orange-950/30 dark:border-orange-800">
              <CardContent className="pt-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <Activity className="h-4 w-4 text-orange-600 dark:text-orange-400" />
                    <span className="font-medium text-orange-800 dark:text-orange-300">API Services</span>
                  </div>
                  <Badge className="bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300">Warning</Badge>
                </div>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span>Requests/min:</span>
                    <span className="font-mono text-orange-600 dark:text-orange-400">2,847</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Error Rate:</span>
                    <span className="font-mono text-red-600 dark:text-red-400">2.1%</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Avg Response:</span>
                    <span className="font-mono text-orange-600 dark:text-orange-400">245ms</span>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>

      {/* Management Tabs */}
      <Tabs defaultValue="services" className="w-full">
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="services">Services</TabsTrigger>
          <TabsTrigger value="resources">Resources</TabsTrigger>
          <TabsTrigger value="scaling">Scaling</TabsTrigger>
          <TabsTrigger value="maintenance">Maintenance</TabsTrigger>
          <TabsTrigger value="monitoring">Monitoring</TabsTrigger>
        </TabsList>

        {/* Services Management */}
        <TabsContent value="services" className="space-y-6">
          <Card className="border-2 border-blue-200 dark:border-blue-800">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-blue-800 dark:text-blue-300">
                <Settings className="h-5 w-5" />
                Service Management
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Core Services Status</h4>
                  
                  <div className="space-y-3">
                    <div className="flex items-center justify-between p-4 border border-slate-200 dark:border-slate-700 rounded-lg hover:bg-slate-50 dark:bg-slate-950/50 dark:hover:bg-slate-800/50">
                      <div className="flex items-center gap-3">
                        <div className="w-3 h-3 bg-green-500 rounded-full"></div>
                        <div>
                          <p className="font-medium text-sm">MCP Protocol Server</p>
                          <p className="text-xs text-muted-foreground">Port 8080 • v2.1.3</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Running</Badge>
                        <Button size="sm" variant="ghost" className="h-6 w-6 p-0">
                          <Pause className="h-3 w-3" />
                        </Button>
                        <Button size="sm" variant="ghost" className="h-6 w-6 p-0">
                          <RefreshCw className="h-3 w-3" />
                        </Button>
                      </div>
                    </div>

                    <div className="flex items-center justify-between p-4 border border-slate-200 dark:border-slate-700 rounded-lg hover:bg-slate-50 dark:bg-slate-950/50 dark:hover:bg-slate-800/50">
                      <div className="flex items-center gap-3">
                        <div className="w-3 h-3 bg-green-500 rounded-full"></div>
                        <div>
                          <p className="font-medium text-sm">Authentication Service</p>
                          <p className="text-xs text-muted-foreground">Port 3001 • v1.8.2</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Running</Badge>
                        <Button size="sm" variant="ghost" className="h-6 w-6 p-0">
                          <Pause className="h-3 w-3" />
                        </Button>
                        <Button size="sm" variant="ghost" className="h-6 w-6 p-0">
                          <RefreshCw className="h-3 w-3" />
                        </Button>
                      </div>
                    </div>

                    <div className="flex items-center justify-between p-4 border border-slate-200 dark:border-slate-700 rounded-lg hover:bg-slate-50 dark:bg-slate-950/50 dark:hover:bg-slate-800/50">
                      <div className="flex items-center gap-3">
                        <div className="w-3 h-3 bg-yellow-500 rounded-full"></div>
                        <div>
                          <p className="font-medium text-sm">Tool Integration Service</p>
                          <p className="text-xs text-muted-foreground">Port 3002 • v2.0.1</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge className="bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300 text-xs">Degraded</Badge>
                        <Button size="sm" variant="ghost" className="h-6 w-6 p-0">
                          <Play className="h-3 w-3" />
                        </Button>
                        <Button size="sm" variant="ghost" className="h-6 w-6 p-0">
                          <RefreshCw className="h-3 w-3" />
                        </Button>
                      </div>
                    </div>

                    <div className="flex items-center justify-between p-4 border border-red-200 dark:border-red-800 rounded-lg bg-red-50 dark:bg-red-950/30">
                      <div className="flex items-center gap-3">
                        <div className="w-3 h-3 bg-red-500 rounded-full"></div>
                        <div>
                          <p className="font-medium text-sm">Analytics Service</p>
                          <p className="text-xs text-muted-foreground">Port 3003 • v1.5.0</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Stopped</Badge>
                        <Button size="sm" variant="ghost" className="h-6 w-6 p-0">
                          <Play className="h-3 w-3 text-green-600 dark:text-green-400" />
                        </Button>
                        <Button size="sm" variant="ghost" className="h-6 w-6 p-0">
                          <Settings className="h-3 w-3" />
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Service Configuration</h4>
                  
                  <Card className="border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
                    <CardContent className="pt-4">
                      <div className="space-y-4">
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-medium">Auto-restart on failure</span>
                          <div className="flex items-center gap-2">
                            <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Enabled</Badge>
                            <Button size="sm" variant="outline" className="h-6 text-xs">
                              Configure
                            </Button>
                          </div>
                        </div>
                        
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-medium">Health check interval</span>
                          <div className="flex items-center gap-2">
                            <span className="text-xs font-mono">30s</span>
                            <Button size="sm" variant="outline" className="h-6 text-xs">
                              Edit
                            </Button>
                          </div>
                        </div>
                        
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-medium">Log retention</span>
                          <div className="flex items-center gap-2">
                            <span className="text-xs font-mono">30 days</span>
                            <Button size="sm" variant="outline" className="h-6 text-xs">
                              Edit
                            </Button>
                          </div>
                        </div>

                        <div className="pt-3 border-t">
                          <div className="flex gap-2">
                            <Button size="sm" className="flex-1 bg-blue-600 hover:bg-blue-700">
                              <RefreshCw className="h-3 w-3 mr-1" />
                              Restart All
                            </Button>
                            <Button size="sm" variant="outline" className="flex-1">
                              <Download className="h-3 w-3 mr-1" />
                              Export Logs
                            </Button>
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>

                  <Alert className="bg-amber-50 dark:bg-amber-950/30 border-amber-200 dark:border-amber-800">
                    <AlertTriangle className="h-4 w-4" />
                    <AlertDescription className="text-sm">
                      <strong>Service Alert:</strong> Analytics Service has been stopped due to memory issues. 
                      Check resource usage before restarting.
                    </AlertDescription>
                  </Alert>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Resource Management */}
        <TabsContent value="resources" className="space-y-6">
          <Card className="border-2 border-green-200 dark:border-green-800">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-green-800 dark:text-green-300">
                <Activity className="h-5 w-5" />
                Resource Management
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Resource Usage Overview */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <div className="p-4 bg-gradient-to-br from-blue-50 to-blue-100 dark:from-blue-950/30 dark:to-blue-900/30 rounded-lg border border-blue-200 dark:border-blue-800">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium text-blue-800 dark:text-blue-300">CPU Usage</span>
                    <Activity className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                  </div>
                  <div className="text-2xl font-bold text-blue-900 dark:text-blue-200">23%</div>
                  <Progress value={23} className="h-2 mt-2" />
                  <div className="text-xs text-blue-700 dark:text-blue-300 mt-1">4 cores available</div>
                </div>

                <div className="p-4 bg-gradient-to-br from-purple-50 to-purple-100 dark:from-purple-950/30 dark:to-purple-900/30 rounded-lg border border-purple-200 dark:border-purple-800">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium text-purple-800 dark:text-purple-300">Memory</span>
                    <Database className="h-4 w-4 text-purple-600 dark:text-purple-400" />
                  </div>
                  <div className="text-2xl font-bold text-purple-900 dark:text-purple-200">67%</div>
                  <Progress value={67} className="h-2 mt-2" />
                  <div className="text-xs text-purple-700 dark:text-purple-300 mt-1">5.4GB / 8GB</div>
                </div>

                <div className="p-4 bg-gradient-to-br from-green-50 to-green-100 dark:from-green-950/30 dark:to-green-900/30 rounded-lg border border-green-200 dark:border-green-800">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium text-green-800 dark:text-green-300">Storage</span>
                    <Server className="h-4 w-4 text-green-600 dark:text-green-400" />
                  </div>
                  <div className="text-2xl font-bold text-green-900 dark:text-green-200">42%</div>
                  <Progress value={42} className="h-2 mt-2" />
                  <div className="text-xs text-green-700 dark:text-green-300 mt-1">84GB / 200GB</div>
                </div>

                <div className="p-4 bg-gradient-to-br from-orange-50 to-orange-100 dark:from-orange-950/30 dark:to-orange-900/30 rounded-lg border border-orange-200 dark:border-orange-800">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium text-orange-800 dark:text-orange-300">Network</span>
                    <Network className="h-4 w-4 text-orange-600 dark:text-orange-400" />
                  </div>
                  <div className="text-2xl font-bold text-orange-900 dark:text-orange-200">156</div>
                  <div className="text-xs text-orange-700 dark:text-orange-300">Mbps throughput</div>
                  <div className="text-xs text-orange-600 dark:text-orange-400 mt-1">2.4GB today</div>
                </div>
              </div>

              {/* Resource Allocation */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Process Resource Usage</h4>
                  <div className="space-y-3">
                    <div className="p-3 border border-slate-200 dark:border-slate-700 rounded-lg">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-sm font-medium">MCP Protocol Server</span>
                        <span className="text-xs text-muted-foreground">PID 1234</span>
                      </div>
                      <div className="grid grid-cols-3 gap-4 text-xs">
                        <div>
                          <span className="text-muted-foreground">CPU:</span>
                          <span className="font-mono ml-1">8.2%</span>
                        </div>
                        <div>
                          <span className="text-muted-foreground">Memory:</span>
                          <span className="font-mono ml-1">1.2GB</span>
                        </div>
                        <div>
                          <span className="text-muted-foreground">Threads:</span>
                          <span className="font-mono ml-1">24</span>
                        </div>
                      </div>
                    </div>

                    <div className="p-3 border border-slate-200 dark:border-slate-700 rounded-lg">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-sm font-medium">Authentication Service</span>
                        <span className="text-xs text-muted-foreground">PID 1235</span>
                      </div>
                      <div className="grid grid-cols-3 gap-4 text-xs">
                        <div>
                          <span className="text-muted-foreground">CPU:</span>
                          <span className="font-mono ml-1">3.1%</span>
                        </div>
                        <div>
                          <span className="text-muted-foreground">Memory:</span>
                          <span className="font-mono ml-1">512MB</span>
                        </div>
                        <div>
                          <span className="text-muted-foreground">Threads:</span>
                          <span className="font-mono ml-1">12</span>
                        </div>
                      </div>
                    </div>

                    <div className="p-3 border border-red-200 dark:border-red-800 rounded-lg bg-red-50 dark:bg-red-950/30">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-sm font-medium">Tool Integration Service</span>
                        <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">High Memory</Badge>
                      </div>
                      <div className="grid grid-cols-3 gap-4 text-xs">
                        <div>
                          <span className="text-muted-foreground">CPU:</span>
                          <span className="font-mono ml-1 text-red-600 dark:text-red-400">15.7%</span>
                        </div>
                        <div>
                          <span className="text-muted-foreground">Memory:</span>
                          <span className="font-mono ml-1 text-red-600 dark:text-red-400">3.2GB</span>
                        </div>
                        <div>
                          <span className="text-muted-foreground">Threads:</span>
                          <span className="font-mono ml-1">45</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Resource Limits & Quotas</h4>
                  <Card className="border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
                    <CardContent className="pt-4">
                      <div className="space-y-4">
                        <div>
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-sm font-medium">CPU Limit per Process</span>
                            <span className="text-xs font-mono">25%</span>
                          </div>
                          <Progress value={25} className="h-2" />
                        </div>
                        
                        <div>
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-sm font-medium">Memory Limit per Process</span>
                            <span className="text-xs font-mono">2GB</span>
                          </div>
                          <Progress value={62} className="h-2" />
                        </div>
                        
                        <div>
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-sm font-medium">Disk I/O Bandwidth</span>
                            <span className="text-xs font-mono">100MB/s</span>
                          </div>
                          <Progress value={43} className="h-2" />
                        </div>

                        <div className="pt-3 border-t">
                          <div className="flex gap-2">
                            <Button size="sm" variant="outline" className="flex-1">
                              <Edit className="h-3 w-3 mr-1" />
                              Configure Limits
                            </Button>
                            <Button size="sm" variant="outline" className="flex-1">
                              <AlertTriangle className="h-3 w-3 mr-1" />
                              Set Alerts
                            </Button>
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>

                  <Alert className="bg-blue-50 dark:bg-blue-950/30 border-blue-200 dark:border-blue-800">
                    <Activity className="h-4 w-4" />
                    <AlertDescription className="text-xs">
                      <strong>Optimization Tip:</strong> Tool Integration Service is using excessive memory. 
                      Consider restarting or reviewing tool configurations.
                    </AlertDescription>
                  </Alert>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Scaling Configuration */}
        <TabsContent value="scaling" className="space-y-6">
          <Card className="border-2 border-purple-200 dark:border-purple-800">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-purple-800 dark:text-purple-300">
                <Zap className="h-5 w-5" />
                Auto-Scaling Configuration
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Scaling Policies</h4>
                  
                  <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
                    <CardHeader>
                      <CardTitle className="text-base">CPU-Based Scaling</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-sm">Scale up threshold</span>
                        <span className="text-sm font-mono">80% CPU</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-sm">Scale down threshold</span>
                        <span className="text-sm font-mono">30% CPU</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-sm">Min instances</span>
                        <span className="text-sm font-mono">2</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-sm">Max instances</span>
                        <span className="text-sm font-mono">10</span>
                      </div>
                      <div className="pt-2 border-t">
                        <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Enabled</Badge>
                        <Button size="sm" variant="outline" className="ml-2 h-6 text-xs">
                          <Edit className="h-3 w-3 mr-1" />
                          Configure
                        </Button>
                      </div>
                    </CardContent>
                  </Card>

                  <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
                    <CardHeader>
                      <CardTitle className="text-base">Memory-Based Scaling</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-sm">Scale up threshold</span>
                        <span className="text-sm font-mono">85% Memory</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-sm">Scale down threshold</span>
                        <span className="text-sm font-mono">40% Memory</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-sm">Cooldown period</span>
                        <span className="text-sm font-mono">5 minutes</span>
                      </div>
                      <div className="pt-2 border-t">
                        <Badge className="bg-gray-100 dark:bg-gray-900/50 text-gray-800 dark:text-gray-300 text-xs">Disabled</Badge>
                        <Button size="sm" variant="outline" className="ml-2 h-6 text-xs">
                          <Play className="h-3 w-3 mr-1" />
                          Enable
                        </Button>
                      </div>
                    </CardContent>
                  </Card>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Scaling History</h4>
                  
                  <div className="space-y-3">
                    <div className="flex items-start gap-3 p-3 border border-slate-200 dark:border-slate-700 rounded-lg">
                      <div className="w-2 h-2 bg-green-500 rounded-full mt-2"></div>
                      <div className="flex-1">
                        <div className="flex items-center justify-between mb-1">
                          <p className="text-sm font-medium">Scaled up to 4 instances</p>
                          <span className="text-xs text-muted-foreground">2 hours ago</span>
                        </div>
                        <p className="text-xs text-muted-foreground">Triggered by CPU usage: 85%</p>
                        <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs mt-1">Successful</Badge>
                      </div>
                    </div>

                    <div className="flex items-start gap-3 p-3 border border-slate-200 dark:border-slate-700 rounded-lg">
                      <div className="w-2 h-2 bg-blue-500 rounded-full mt-2"></div>
                      <div className="flex-1">
                        <div className="flex items-center justify-between mb-1">
                          <p className="text-sm font-medium">Scaled down to 2 instances</p>
                          <span className="text-xs text-muted-foreground">6 hours ago</span>
                        </div>
                        <p className="text-xs text-muted-foreground">Triggered by CPU usage: 25%</p>
                        <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs mt-1">Successful</Badge>
                      </div>
                    </div>

                    <div className="flex items-start gap-3 p-3 border border-orange-200 dark:border-orange-800 rounded-lg bg-orange-50 dark:bg-orange-950/30">
                      <div className="w-2 h-2 bg-orange-500 rounded-full mt-2"></div>
                      <div className="flex-1">
                        <div className="flex items-center justify-between mb-1">
                          <p className="text-sm font-medium">Scale up blocked</p>
                          <span className="text-xs text-muted-foreground">1 day ago</span>
                        </div>
                        <p className="text-xs text-muted-foreground">Max instance limit reached (10)</p>
                        <Badge className="bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 text-xs mt-1">Blocked</Badge>
                      </div>
                    </div>
                  </div>

                  <Card className="border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
                    <CardContent className="pt-4">
                      <div className="space-y-3">
                        <h5 className="text-sm font-medium">Current Configuration</h5>
                        <div className="space-y-2 text-sm">
                          <div className="flex justify-between">
                            <span>Current instances:</span>
                            <span className="font-mono">4</span>
                          </div>
                          <div className="flex justify-between">
                            <span>Target instances:</span>
                            <span className="font-mono">4</span>
                          </div>
                          <div className="flex justify-between">
                            <span>Next evaluation:</span>
                            <span className="font-mono">3m 45s</span>
                          </div>
                        </div>
                        <div className="pt-2 border-t">
                          <Button size="sm" variant="outline" className="w-full">
                            <RefreshCw className="h-3 w-3 mr-1" />
                            Force Evaluation
                          </Button>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Maintenance */}
        <TabsContent value="maintenance" className="space-y-6">
          <Card className="border-2 border-orange-200 dark:border-orange-800">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-orange-800 dark:text-orange-300">
                <Settings className="h-5 w-5" />
                Server Maintenance
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Scheduled Maintenance</h4>
                  
                  <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
                    <CardContent className="pt-4">
                      <div className="space-y-4">
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-medium">Automatic backups</span>
                          <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Active</Badge>
                        </div>
                        <div className="text-xs text-muted-foreground space-y-1">
                          <p>• Daily at 2:00 AM UTC</p>
                          <p>• Retention: 30 days</p>
                          <p>• Next backup: Tonight 2:00 AM</p>
                        </div>
                        <Button size="sm" variant="outline" className="w-full">
                          <Settings className="h-3 w-3 mr-1" />
                          Configure Backups
                        </Button>
                      </div>
                    </CardContent>
                  </Card>

                  <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
                    <CardContent className="pt-4">
                      <div className="space-y-4">
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-medium">Security updates</span>
                          <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Auto</Badge>
                        </div>
                        <div className="text-xs text-muted-foreground space-y-1">
                          <p>• Weekly security patches</p>
                          <p>• Last update: 3 days ago</p>
                          <p>• Next check: Tomorrow</p>
                        </div>
                        <Button size="sm" variant="outline" className="w-full">
                          <RefreshCw className="h-3 w-3 mr-1" />
                          Check Now
                        </Button>
                      </div>
                    </CardContent>
                  </Card>

                  <Card className="border-purple-200 bg-purple-50 dark:bg-purple-950/30 dark:border-purple-800">
                    <CardContent className="pt-4">
                      <div className="space-y-4">
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-medium">Log cleanup</span>
                          <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 text-xs">Scheduled</Badge>
                        </div>
                        <div className="text-xs text-muted-foreground space-y-1">
                          <p>• Monthly cleanup task</p>
                          <p>• Keep logs for 30 days</p>
                          <p>• Next cleanup: In 12 days</p>
                        </div>
                        <Button size="sm" variant="outline" className="w-full">
                          <Trash2 className="h-3 w-3 mr-1" />
                          Clean Now
                        </Button>
                      </div>
                    </CardContent>
                  </Card>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Manual Operations</h4>
                  
                  <div className="grid grid-cols-2 gap-3">
                    <Button variant="outline" className="h-16 flex-col gap-1">
                      <Download className="h-4 w-4" />
                      <span className="text-xs">Backup Now</span>
                    </Button>
                    <Button variant="outline" className="h-16 flex-col gap-1">
                      <RefreshCw className="h-4 w-4" />
                      <span className="text-xs">Restart Server</span>
                    </Button>
                    <Button variant="outline" className="h-16 flex-col gap-1">
                      <Upload className="h-4 w-4" />
                      <span className="text-xs">Update System</span>
                    </Button>
                    <Button variant="outline" className="h-16 flex-col gap-1">
                      <Terminal className="h-4 w-4" />
                      <span className="text-xs">System Shell</span>
                    </Button>
                  </div>

                  <Alert className="bg-amber-50 dark:bg-amber-950/30 border-amber-200 dark:border-amber-800">
                    <AlertTriangle className="h-4 w-4" />
                    <AlertDescription className="text-sm">
                      <strong>Maintenance Window:</strong> Scheduled maintenance is planned for this Sunday 2:00-4:00 AM UTC. 
                      All services will be briefly unavailable during updates.
                    </AlertDescription>
                  </Alert>

                  <Card className="border-slate-200 dark:border-slate-700">
                    <CardHeader>
                      <CardTitle className="text-base">Recent Maintenance</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <div className="flex items-center justify-between text-sm">
                        <span>Database optimization</span>
                        <span className="text-muted-foreground">3 days ago</span>
                      </div>
                      <div className="flex items-center justify-between text-sm">
                        <span>Security patch v2.1.3</span>
                        <span className="text-muted-foreground">1 week ago</span>
                      </div>
                      <div className="flex items-center justify-between text-sm">
                        <span>SSL certificate renewal</span>
                        <span className="text-muted-foreground">2 weeks ago</span>
                      </div>
                      <Button size="sm" variant="outline" className="w-full">
                        <Clock className="h-3 w-3 mr-1" />
                        View Full History
                      </Button>
                    </CardContent>
                  </Card>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Monitoring */}
        <TabsContent value="monitoring" className="space-y-6">
          <Card className="border-2 border-indigo-200 dark:border-indigo-800">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-indigo-800 dark:text-indigo-300">
                <Monitor className="h-5 w-5" />
                Advanced Monitoring
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Performance Metrics</h4>
                  
                  <div className="bg-slate-100 dark:bg-slate-800 rounded-lg p-6 text-center">
                    <p className="text-muted-foreground mb-2">[Performance Charts Placeholder]</p>
                    <p className="text-sm text-muted-foreground">
                      Add real-time performance monitoring charts: CPU usage over time, 
                      Memory consumption patterns, Network throughput, Response times
                    </p>
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div className="p-3 border border-slate-200 dark:border-slate-700 rounded-lg">
                      <div className="text-sm font-medium">Avg Response Time</div>
                      <div className="text-lg font-bold text-green-600 dark:text-green-400">142ms</div>
                      <div className="text-xs text-green-600 dark:text-green-400">↓ 12ms from last hour</div>
                    </div>
                    <div className="p-3 border border-slate-200 dark:border-slate-700 rounded-lg">
                      <div className="text-sm font-medium">Error Rate</div>
                      <div className="text-lg font-bold text-orange-600 dark:text-orange-400">0.8%</div>
                      <div className="text-xs text-orange-600 dark:text-orange-400">↑ 0.2% from last hour</div>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Alert Configuration</h4>
                  
                  <div className="space-y-3">
                    <div className="flex items-center justify-between p-3 border border-slate-200 dark:border-slate-700 rounded-lg">
                      <div>
                        <p className="font-medium text-sm">High CPU Usage</p>
                        <p className="text-xs text-muted-foreground">Trigger when CPU {'>'} 90% for 5 minutes</p>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Active</Badge>
                        <Button size="sm" variant="ghost" className="h-6 w-6 p-0">
                          <Edit className="h-3 w-3" />
                        </Button>
                      </div>
                    </div>

                    <div className="flex items-center justify-between p-3 border border-slate-200 dark:border-slate-700 rounded-lg">
                      <div>
                        <p className="font-medium text-sm">Memory Exhaustion</p>
                        <p className="text-xs text-muted-foreground">Trigger when memory {'>'} 95% for 2 minutes</p>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Active</Badge>
                        <Button size="sm" variant="ghost" className="h-6 w-6 p-0">
                          <Edit className="h-3 w-3" />
                        </Button>
                      </div>
                    </div>

                    <div className="flex items-center justify-between p-3 border border-slate-200 dark:border-slate-700 rounded-lg">
                      <div>
                        <p className="font-medium text-sm">Service Downtime</p>
                        <p className="text-xs text-muted-foreground">Trigger when service is unreachable</p>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge className="bg-gray-100 dark:bg-gray-900/50 text-gray-800 dark:text-gray-300 text-xs">Disabled</Badge>
                        <Button size="sm" variant="ghost" className="h-6 w-6 p-0">
                          <Edit className="h-3 w-3" />
                        </Button>
                      </div>
                    </div>

                    <div className="flex items-center justify-between p-3 border border-slate-200 dark:border-slate-700 rounded-lg">
                      <div>
                        <p className="font-medium text-sm">High Error Rate</p>
                        <p className="text-xs text-muted-foreground">Trigger when error rate &gt; 5% for 10 minutes</p>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Active</Badge>
                        <Button size="sm" variant="ghost" className="h-6 w-6 p-0">
                          <Edit className="h-3 w-3" />
                        </Button>
                      </div>
                    </div>
                  </div>

                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" className="flex-1">
                      <Plus className="h-3 w-3 mr-1" />
                      Add Alert
                    </Button>
                    <Button size="sm" variant="outline" className="flex-1">
                      <Bell className="h-3 w-3 mr-1" />
                      Test Alerts
                    </Button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Best Practices */}
      <Card className="bg-gradient-to-r from-red-50 dark:from-red-950 to-orange-50 dark:to-orange-950 border-red-200 dark:border-red-800">
        <CardHeader>
          <CardTitle className="text-red-800 dark:text-red-300">Server Management Best Practices</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-3">
              <h4 className="font-semibold text-red-800 dark:text-red-300">Performance Optimization</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <Activity className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Monitor Resource Usage</p>
                    <p className="text-xs text-muted-foreground">Track CPU, memory, and disk usage trends</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Zap className="h-4 w-4 text-yellow-600 dark:text-yellow-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Optimize Auto-Scaling</p>
                    <p className="text-xs text-muted-foreground">Configure thresholds based on actual usage patterns</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <RefreshCw className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Regular Restarts</p>
                    <p className="text-xs text-muted-foreground">Schedule periodic service restarts to prevent memory leaks</p>
                  </div>
                </li>
              </ul>
            </div>
            
            <div className="space-y-3">
              <h4 className="font-semibold text-red-800 dark:text-red-300">Operational Security</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <Shield className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Regular Backups</p>
                    <p className="text-xs text-muted-foreground">Verify backup integrity and test restore procedures</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Lock className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Access Control</p>
                    <p className="text-xs text-muted-foreground">Limit server management access to essential personnel</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Bell className="h-4 w-4 text-orange-600 dark:text-orange-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Proactive Alerts</p>
                    <p className="text-xs text-muted-foreground">Set up comprehensive monitoring and alerting</p>
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