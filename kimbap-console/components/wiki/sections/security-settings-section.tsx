import { Shield, Lock, Key, Users, CheckCircle } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

export function SecuritySettingsSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-red-500 to-pink-600 rounded-2xl mb-4">
          <Shield className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">Security Settings</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Configure authentication, encryption, and access controls to protect your MCP server.
        </p>
      </div>

      {/* Security Status */}
      <Card className="border-2 border-red-200 dark:border-red-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-red-800 dark:text-red-300">
            <Shield className="h-5 w-5" />
            Security Status
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="bg-gradient-to-br from-green-50 to-emerald-100 dark:from-green-950/30 dark:to-emerald-900/30 rounded-lg p-6 border border-green-200 dark:border-green-800">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-2xl font-bold text-green-800 dark:text-green-300">Security Score: 94/100</h3>
                <p className="text-sm text-green-700 dark:text-green-300">Excellent security posture</p>
              </div>
              <CheckCircle className="h-12 w-12 text-green-600 dark:text-green-400" />
            </div>
            <div className="grid grid-cols-2 gap-4 text-xs text-green-700 dark:text-green-300">
              <span>Critical issues: 0</span>
              <span>Recommendations: 2</span>
            </div>
          </div>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
              <CardContent className="pt-4">
                <div className="flex items-center justify-between mb-2">
                  <Lock className="h-5 w-5 text-green-600 dark:text-green-400" />
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Active</Badge>
                </div>
                <div className="text-lg font-bold text-green-800 dark:text-green-300">AES-256</div>
                <p className="text-xs text-green-700 dark:text-green-300">Encryption</p>
              </CardContent>
            </Card>

            <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
              <CardContent className="pt-4">
                <div className="flex items-center justify-between mb-2">
                  <Key className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                  <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Enabled</Badge>
                </div>
                <div className="text-lg font-bold text-blue-800 dark:text-blue-300">MFA</div>
                <p className="text-xs text-blue-700 dark:text-blue-300">Multi-Factor Auth</p>
              </CardContent>
            </Card>

            <Card className="border-purple-200 bg-purple-50 dark:bg-purple-950/30 dark:border-purple-800">
              <CardContent className="pt-4">
                <div className="flex items-center justify-between mb-2">
                  <Shield className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                  <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 text-xs">SOC 2</Badge>
                </div>
                <div className="text-lg font-bold text-purple-800 dark:text-purple-300">Type II</div>
                <p className="text-xs text-purple-700 dark:text-purple-300">Compliance</p>
              </CardContent>
            </Card>

            <Card className="border-orange-200 bg-orange-50 dark:bg-orange-950/30 dark:border-orange-800">
              <CardContent className="pt-4">
                <div className="flex items-center justify-between mb-2">
                  <Users className="h-5 w-5 text-orange-600 dark:text-orange-400" />
                  <Badge className="bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 text-xs">24/7</Badge>
                </div>
                <div className="text-lg font-bold text-orange-800 dark:text-orange-300">Monitor</div>
                <p className="text-xs text-orange-700 dark:text-orange-300">Security Logs</p>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>

      {/* Authentication */}
      <Card className="border-2 border-blue-200 dark:border-blue-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-blue-800 dark:text-blue-300">
            <Key className="h-5 w-5" />
            Authentication & Access
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
              <CardHeader>
                <CardTitle className="text-base">Multi-Factor Authentication</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <p className="text-sm text-blue-800 dark:text-blue-300">
                  Secure your account with additional authentication factors.
                </p>
                <div className="space-y-2">
                  <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded">
                    <span className="text-sm">Authenticator Apps</span>
                    <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Enabled</Badge>
                  </div>
                  <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded">
                    <span className="text-sm">SMS Verification</span>
                    <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Enabled</Badge>
                  </div>
                  <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded">
                    <span className="text-sm">Hardware Keys</span>
                    <Button size="sm" variant="outline" className="text-xs">Configure</Button>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
              <CardHeader>
                <CardTitle className="text-base">Single Sign-On</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <p className="text-sm text-green-800 dark:text-green-300">
                  Integrate with your organization's identity provider.
                </p>
                <div className="p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium">SAML 2.0</span>
                    <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Active</Badge>
                  </div>
                  <div className="space-y-1 text-xs">
                    <div className="flex justify-between">
                      <span>Provider:</span>
                      <span>Okta</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Users:</span>
                      <span>127 active</span>
                    </div>
                  </div>
                </div>
                <Button className="w-full bg-green-600 hover:bg-green-700">
                  Manage SSO
                </Button>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>

      {/* Encryption */}
      <Card className="border-2 border-purple-200 dark:border-purple-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-purple-800 dark:text-purple-300">
            <Lock className="h-5 w-5" />
            Data Protection
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
            <Card className="border-purple-200 bg-purple-50 dark:bg-purple-950/30 dark:border-purple-800">
              <CardHeader>
                <CardTitle className="text-base">Encryption at Rest</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span>Algorithm:</span>
                    <span className="font-mono">AES-256-GCM</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Key rotation:</span>
                    <span>90 days</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Status:</span>
                    <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                  </div>
                </div>
              </CardContent>
            </Card>
            
            <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
              <CardHeader>
                <CardTitle className="text-base">Encryption in Transit</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span>TLS Version:</span>
                    <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">TLS 1.3</Badge>
                  </div>
                  <div className="flex justify-between">
                    <span>Perfect Forward Secrecy:</span>
                    <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                  </div>
                  <div className="flex justify-between">
                    <span>HSTS:</span>
                    <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                  </div>
                </div>
              </CardContent>
            </Card>
            
            <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
              <CardHeader>
                <CardTitle className="text-base">Key Management</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span>HSM Storage:</span>
                    <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                  </div>
                  <div className="flex justify-between">
                    <span>FIPS 140-2:</span>
                    <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Level 3</Badge>
                  </div>
                  <div className="flex justify-between">
                    <span>Auto-rotation:</span>
                    <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>

      {/* Access Control */}
      <Card className="border-2 border-indigo-200 dark:border-indigo-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-indigo-800 dark:text-indigo-300">
            <Users className="h-5 w-5" />
            Access Control
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="space-y-4">
              <h4 className="font-semibold">IP Allowlist</h4>
              <div className="space-y-2">
                <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-indigo-200 dark:border-indigo-800 rounded-lg">
                  <div>
                    <p className="text-sm font-medium">Corporate Office</p>
                    <p className="text-xs text-muted-foreground font-mono">203.0.113.0/24</p>
                  </div>
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Active</Badge>
                </div>
                <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-indigo-200 dark:border-indigo-800 rounded-lg">
                  <div>
                    <p className="text-sm font-medium">VPN Gateway</p>
                    <p className="text-xs text-muted-foreground font-mono">198.51.100.5/32</p>
                  </div>
                  <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Active</Badge>
                </div>
              </div>
              <Button className="w-full bg-indigo-600 hover:bg-indigo-700">
                Manage IP Access
              </Button>
            </div>
            
            <div className="space-y-4">
              <h4 className="font-semibold">API Security</h4>
              <div className="grid grid-cols-3 gap-3 text-center">
                <div className="p-3 bg-white dark:bg-slate-800 border border-indigo-200 dark:border-indigo-800 rounded-lg">
                  <div className="text-xl font-bold text-indigo-800 dark:text-indigo-300">1000</div>
                  <p className="text-xs text-indigo-700 dark:text-indigo-300">Requests/min</p>
                </div>
                <div className="p-3 bg-white dark:bg-slate-800 border border-indigo-200 dark:border-indigo-800 rounded-lg">
                  <div className="text-xl font-bold text-indigo-800 dark:text-indigo-300">50K</div>
                  <p className="text-xs text-indigo-700 dark:text-indigo-300">Daily limit</p>
                </div>
                <div className="p-3 bg-white dark:bg-slate-800 border border-indigo-200 dark:border-indigo-800 rounded-lg">
                  <div className="text-xl font-bold text-indigo-800 dark:text-indigo-300">128</div>
                  <p className="text-xs text-indigo-700 dark:text-indigo-300">Active keys</p>
                </div>
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span>Rate limiting</span>
                  <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                </div>
                <div className="flex items-center justify-between text-sm">
                  <span>Request signing</span>
                  <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                </div>
                <div className="flex items-center justify-between text-sm">
                  <span>API key rotation</span>
                  <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Security Best Practices */}
      <Card className="bg-gradient-to-r from-red-50 dark:from-red-950 to-pink-50 dark:to-pink-950 border-red-200 dark:border-red-800">
        <CardHeader>
          <CardTitle className="text-red-800 dark:text-red-300">Security Best Practices</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-3">
              <h4 className="font-semibold text-red-800 dark:text-red-300">Essential Security</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <Shield className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Enable Multi-Factor Authentication</p>
                    <p className="text-xs text-muted-foreground">Require MFA for all admin accounts</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Key className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Regular Key Rotation</p>
                    <p className="text-xs text-muted-foreground">Rotate access tokens periodically</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Users className="h-4 w-4 text-purple-600 dark:text-purple-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Monitor Access Patterns</p>
                    <p className="text-xs text-muted-foreground">Set up alerts for unusual activity</p>
                  </div>
                </li>
              </ul>
            </div>
            
            <div className="space-y-3">
              <h4 className="font-semibold text-red-800 dark:text-red-300">Advanced Security</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <Lock className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">End-to-End Encryption</p>
                    <p className="text-xs text-muted-foreground">Secure data at rest and in transit</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Regular Security Reviews</p>
                    <p className="text-xs text-muted-foreground">Audit settings quarterly</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Shield className="h-4 w-4 text-orange-600 dark:text-orange-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Least Privilege Access</p>
                    <p className="text-xs text-muted-foreground">Grant minimum necessary permissions</p>
                  </div>
                </li>
              </ul>
            </div>
          </div>

          <Alert className="bg-white dark:bg-slate-800 border-red-200 dark:border-red-800 mt-6">
            <Shield className="h-4 w-4" />
            <AlertDescription className="text-sm">
              <strong>Security Notice:</strong> Regular security training and awareness programs are essential for maintaining a strong security posture.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    </div>
  )
}