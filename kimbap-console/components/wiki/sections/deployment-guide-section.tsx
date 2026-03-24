import { Server, Cloud, Shield, Settings, CheckCircle, Copy, Zap } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

export function DeploymentGuideSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-purple-500 to-indigo-600 rounded-2xl mb-4">
          <Server className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">Advanced Deployment Options</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Optional advanced configurations for specialized deployment scenarios and enterprise requirements.
        </p>
      </div>

      {/* Note about simplicity */}
      <Alert className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
        <CheckCircle className="h-4 w-4" />
        <AlertDescription>
          <strong>Most users don't need these advanced options.</strong> The standard installation works great for teams of all sizes. 
          These configurations are for specific enterprise requirements only.
        </AlertDescription>
      </Alert>

      {/* Load Balancing */}
      <Card className="border-2 border-blue-200 dark:border-blue-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Cloud className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            Load Balancing & High Availability
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            For organizations requiring 99.99% uptime and handling 1000+ concurrent users.
          </p>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-3">
              <h4 className="font-semibold">Multi-Instance Setup</h4>
              <div className="bg-slate-900 rounded-lg p-4">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-slate-300 text-xs">Docker Swarm</span>
                  <Button size="sm" variant="ghost" className="h-6 text-xs text-slate-300">
                    <Copy className="h-3 w-3 mr-1" />
                    Copy
                  </Button>
                </div>
                <div className="text-slate-100 font-mono text-xs space-y-1">
                  <div className="text-yellow-400"># Scale to 3 replicas</div>
                  <div className="text-green-400">docker service create --replicas 3 kimbap-console</div>
                  <div className="text-green-400">docker service ls</div>
                </div>
              </div>
            </div>
            
            <div className="space-y-3">
              <h4 className="font-semibold">When You Need This</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Large Teams (500+ users)</p>
                    <p className="text-xs text-muted-foreground">Distribute load across instances</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Critical Uptime Requirements</p>
                    <p className="text-xs text-muted-foreground">Zero-downtime deployments</p>
                  </div>
                </li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Enterprise Security */}
      <Card className="border-2 border-red-200 dark:border-red-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5 text-red-600 dark:text-red-400" />
            Enterprise Security Hardening
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Additional security configurations for regulated industries (finance, healthcare, government).
          </p>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="p-3 bg-red-50 dark:bg-red-950/30 rounded-lg border border-red-200 dark:border-red-800">
              <h4 className="font-semibold text-sm mb-2">Network Isolation</h4>
              <ul className="text-xs space-y-1 text-muted-foreground">
                <li>• Private network deployment</li>
                <li>• VPN-only access</li>
                <li>• Custom firewall rules</li>
                <li>• Network segmentation</li>
              </ul>
            </div>
            <div className="p-3 bg-red-50 dark:bg-red-950/30 rounded-lg border border-red-200 dark:border-red-800">
              <h4 className="font-semibold text-sm mb-2">Data Encryption</h4>
              <ul className="text-xs space-y-1 text-muted-foreground">
                <li>• Custom encryption keys</li>
                <li>• Hardware security modules</li>
                <li>• Key rotation policies</li>
                <li>• Encrypted at-rest storage</li>
              </ul>
            </div>
            <div className="p-3 bg-red-50 dark:bg-red-950/30 rounded-lg border border-red-200 dark:border-red-800">
              <h4 className="font-semibold text-sm mb-2">Compliance</h4>
              <ul className="text-xs space-y-1 text-muted-foreground">
                <li>• GDPR configuration</li>
                <li>• HIPAA compliance</li>
                <li>• SOX controls</li>
                <li>• Audit logging</li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Custom Integrations */}
      <Card className="border-2 border-purple-200 dark:border-purple-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Settings className="h-5 w-5 text-purple-600 dark:text-purple-400" />
            Custom Integration Scenarios
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Specialized configurations for unique enterprise environments and custom toolchains.
          </p>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-3">
              <h4 className="font-semibold">Custom Tool Development</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-purple-600 dark:text-purple-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Internal API Connections</p>
                    <p className="text-xs text-muted-foreground">Connect to proprietary systems</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-purple-600 dark:text-purple-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Legacy System Integration</p>
                    <p className="text-xs text-muted-foreground">Bridge older enterprise tools</p>
                  </div>
                </li>
              </ul>
            </div>
            
            <div className="space-y-3">
              <h4 className="font-semibold">Advanced Authentication</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-purple-600 dark:text-purple-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">SAML/LDAP Integration</p>
                    <p className="text-xs text-muted-foreground">Enterprise identity providers</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-purple-600 dark:text-purple-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Multi-Factor Authentication</p>
                    <p className="text-xs text-muted-foreground">Hardware tokens, biometrics</p>
                  </div>
                </li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Enterprise Support */}
      <Card className="bg-gradient-to-r from-indigo-50 dark:from-indigo-950 to-purple-50 dark:to-purple-950 border-indigo-200 dark:border-indigo-800">
        <CardHeader>
          <CardTitle className="text-indigo-800 dark:text-indigo-300">Need Help with Advanced Deployment?</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground mb-4">
            These advanced scenarios typically require customization and planning. Our enterprise support team can help design the right solution for your specific requirements.
          </p>
          
          <div className="flex flex-col sm:flex-row gap-3">
            <Button className="flex-1">
              <Zap className="h-4 w-4 mr-2" />
              Contact Enterprise Support
            </Button>
            <Button variant="outline" className="flex-1">
              Schedule Architecture Review
            </Button>
          </div>
          
          <Alert className="bg-white dark:bg-slate-800 border-indigo-200 dark:border-indigo-800 mt-4">
            <Settings className="h-4 w-4" />
            <AlertDescription className="text-xs">
              <strong>Remember:</strong> Most organizations start with the standard installation and scale up as needed. 
              You can always add these features later.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    </div>
  )
}