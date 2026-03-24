import { Network, Globe, Shield, Lock, CheckCircle } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

export function NetworkAccessSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-cyan-500 to-blue-600 rounded-2xl mb-4">
          <Network className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">Network Access</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Control network-level access and configure security for your MCP server.
        </p>
      </div>

      {/* Network Status */}
      <Card className="border-2 border-cyan-200 dark:border-cyan-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-cyan-800 dark:text-cyan-300">
            <Network className="h-5 w-5" />
            Network Status
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
              <CardContent className="pt-4">
                <div className="flex items-center justify-between mb-2">
                  <Network className="h-5 w-5 text-green-600 dark:text-green-400" />
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Healthy</Badge>
                </div>
                <div className="text-lg font-bold text-green-800 dark:text-green-300">99.9%</div>
                <p className="text-xs text-green-700 dark:text-green-300">Network Uptime</p>
              </CardContent>
            </Card>

            <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
              <CardContent className="pt-4">
                <div className="flex items-center justify-between mb-2">
                  <Globe className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                  <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Normal</Badge>
                </div>
                <div className="text-lg font-bold text-blue-800 dark:text-blue-300">142ms</div>
                <p className="text-xs text-blue-700 dark:text-blue-300">Avg Response</p>
              </CardContent>
            </Card>

            <Card className="border-purple-200 bg-purple-50 dark:bg-purple-950/30 dark:border-purple-800">
              <CardContent className="pt-4">
                <div className="flex items-center justify-between mb-2">
                  <Globe className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                  <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 text-xs">Active</Badge>
                </div>
                <div className="text-lg font-bold text-purple-800 dark:text-purple-300">23</div>
                <p className="text-xs text-purple-700 dark:text-purple-300">Countries</p>
              </CardContent>
            </Card>

            <Card className="border-orange-200 bg-orange-50 dark:bg-orange-950/30 dark:border-orange-800">
              <CardContent className="pt-4">
                <div className="flex items-center justify-between mb-2">
                  <Shield className="h-5 w-5 text-orange-600 dark:text-orange-400" />
                  <Badge className="bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 text-xs">Protected</Badge>
                </div>
                <div className="text-lg font-bold text-orange-800 dark:text-orange-300">1,247</div>
                <p className="text-xs text-orange-700 dark:text-orange-300">Blocked IPs</p>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>

      {/* IP Access Control */}
      <Card className="border-2 border-green-200 dark:border-green-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-green-800 dark:text-green-300">
            <Shield className="h-5 w-5" />
            IP Access Control
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="space-y-4">
              <h4 className="font-semibold">Allowed IP Ranges</h4>
              <div className="space-y-2">
                <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                  <div>
                    <p className="text-sm font-medium">Corporate Office</p>
                    <p className="text-xs text-muted-foreground font-mono">203.0.113.0/24</p>
                  </div>
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Active</Badge>
                </div>
                <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                  <div>
                    <p className="text-sm font-medium">VPN Gateway</p>
                    <p className="text-xs text-muted-foreground font-mono">198.51.100.5/32</p>
                  </div>
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Active</Badge>
                </div>
              </div>
              <Button className="w-full bg-green-600 hover:bg-green-700">
                Manage IP Allowlist
              </Button>
            </div>
            
            <div className="space-y-4">
              <h4 className="font-semibold">Access Statistics</h4>
              <div className="p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                <div className="space-y-3">
                  <div className="flex justify-between items-center">
                    <span className="text-sm">Allowed requests (24h):</span>
                    <span className="font-mono text-green-600 dark:text-green-400 text-sm">34,682</span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm">Blocked requests (24h):</span>
                    <span className="font-mono text-red-600 dark:text-red-400 text-sm">1,247</span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm">Success rate:</span>
                    <span className="font-mono text-green-600 dark:text-green-400 text-sm">96.5%</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Domain Security */}
      <Card className="border-2 border-blue-200 dark:border-blue-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-blue-800 dark:text-blue-300">
            <Globe className="h-5 w-5" />
            Domain Security
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="space-y-4">
              <h4 className="font-semibold">Trusted Domains</h4>
              <div className="space-y-2">
                <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded-lg">
                  <div>
                    <p className="text-sm font-medium">company.com</p>
                    <p className="text-xs text-muted-foreground">Primary corporate domain</p>
                  </div>
                  <div className="flex gap-1">
                    <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">HTTPS</Badge>
                    <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Verified</Badge>
                  </div>
                </div>
                <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded-lg">
                  <div>
                    <p className="text-sm font-medium">app.company.com</p>
                    <p className="text-xs text-muted-foreground">Application subdomain</p>
                  </div>
                  <div className="flex gap-1">
                    <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">HTTPS</Badge>
                    <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Verified</Badge>
                  </div>
                </div>
              </div>
              <Button className="w-full bg-blue-600 hover:bg-blue-700">
                Manage Domains
              </Button>
            </div>
            
            <div className="space-y-4">
              <h4 className="font-semibold">SSL Certificate Status</h4>
              <div className="p-3 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded-lg">
                <div className="flex items-center justify-between mb-2">
                  <h5 className="text-sm font-medium">Wildcard Certificate</h5>
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Valid</Badge>
                </div>
                <div className="space-y-1 text-xs">
                  <div className="flex justify-between">
                    <span>Domain:</span>
                    <span className="font-mono">*.company.com</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Issuer:</span>
                    <span>Let's Encrypt</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Expires:</span>
                    <span>2024-04-15</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Auto-renewal:</span>
                    <CheckCircle className="h-3 w-3 text-green-600 dark:text-green-400" />
                  </div>
                </div>
              </div>
              <Button className="w-full" variant="outline">
                Manage Certificates
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Network Monitoring */}
      <Card className="border-2 border-purple-200 dark:border-purple-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-purple-800 dark:text-purple-300">
            <Network className="h-5 w-5" />
            Network Monitoring
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="p-4 bg-white dark:bg-slate-800 border border-purple-200 dark:border-purple-800 rounded-lg">
              <div className="flex items-center justify-between mb-3">
                <h4 className="font-semibold">Network Health</h4>
                <div className="w-3 h-3 bg-green-500 rounded-full animate-pulse"></div>
              </div>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span>Uptime:</span>
                  <span className="font-mono text-green-600 dark:text-green-400">99.97%</span>
                </div>
                <div className="flex justify-between">
                  <span>Latency:</span>
                  <span className="font-mono text-green-600 dark:text-green-400">142ms</span>
                </div>
                <div className="flex justify-between">
                  <span>Packet Loss:</span>
                  <span className="font-mono text-green-600 dark:text-green-400">0.03%</span>
                </div>
              </div>
            </div>
            
            <div className="p-4 bg-white dark:bg-slate-800 border border-purple-200 dark:border-purple-800 rounded-lg">
              <div className="flex items-center justify-between mb-3">
                <h4 className="font-semibold">Traffic Volume</h4>
                <Network className="h-4 w-4 text-blue-600 dark:text-blue-400" />
              </div>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span>Requests today:</span>
                  <span className="font-mono text-blue-600 dark:text-blue-400">847K</span>
                </div>
                <div className="flex justify-between">
                  <span>Peak rate:</span>
                  <span className="font-mono text-orange-600 dark:text-orange-400">567/sec</span>
                </div>
                <div className="flex justify-between">
                  <span>Error rate:</span>
                  <span className="font-mono text-green-600 dark:text-green-400">0.3%</span>
                </div>
              </div>
            </div>
            
            <div className="p-4 bg-white dark:bg-slate-800 border border-purple-200 dark:border-purple-800 rounded-lg">
              <div className="flex items-center justify-between mb-3">
                <h4 className="font-semibold">Security Events</h4>
                <Shield className="h-4 w-4 text-red-600 dark:text-red-400" />
              </div>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span>Blocked IPs:</span>
                  <span className="font-mono text-red-600 dark:text-red-400">1,247</span>
                </div>
                <div className="flex justify-between">
                  <span>Failed logins:</span>
                  <span className="font-mono text-orange-600 dark:text-orange-400">23</span>
                </div>
                <div className="flex justify-between">
                  <span>Alerts:</span>
                  <span className="font-mono text-green-600 dark:text-green-400">0</span>
                </div>
              </div>
            </div>
          </div>

          <div className="flex gap-3">
            <Button className="bg-purple-600 hover:bg-purple-700">
              View Detailed Metrics
            </Button>
            <Button variant="outline">
              Configure Alerts
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Best Practices */}
      <Card className="bg-gradient-to-r from-cyan-50 dark:from-cyan-950 to-blue-50 dark:to-blue-950 border-cyan-200 dark:border-cyan-800">
        <CardHeader>
          <CardTitle className="text-cyan-800 dark:text-cyan-300">Network Security Best Practices</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-3">
              <h4 className="font-semibold text-cyan-800 dark:text-cyan-300">Essential Network Controls</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <Shield className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">IP Allowlisting</p>
                    <p className="text-xs text-muted-foreground">Restrict access to known secure networks</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Lock className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Strong SSL/TLS</p>
                    <p className="text-xs text-muted-foreground">Enforce TLS 1.3 with perfect forward secrecy</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Globe className="h-4 w-4 text-purple-600 dark:text-purple-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Domain Validation</p>
                    <p className="text-xs text-muted-foreground">Configure CORS properly for web applications</p>
                  </div>
                </li>
              </ul>
            </div>
            
            <div className="space-y-3">
              <h4 className="font-semibold text-cyan-800 dark:text-cyan-300">Monitoring & Alerting</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <Network className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Real-time Monitoring</p>
                    <p className="text-xs text-muted-foreground">Monitor traffic patterns and performance</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Shield className="h-4 w-4 text-orange-600 dark:text-orange-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Security Alerts</p>
                    <p className="text-xs text-muted-foreground">Get notified of suspicious activity</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Regular Audits</p>
                    <p className="text-xs text-muted-foreground">Review network settings quarterly</p>
                  </div>
                </li>
              </ul>
            </div>
          </div>

          <Alert className="bg-white dark:bg-slate-800 border-cyan-200 dark:border-cyan-800 mt-6">
            <Network className="h-4 w-4" />
            <AlertDescription className="text-sm">
              <strong>Network Security Tip:</strong> Regular network security audits and penetration testing help identify vulnerabilities before attackers do.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    </div>
  )
}