import { CheckCircle, Shield, Users, Star, Crown, Globe, Lock, Key, Database, Monitor } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"

export function WhyChooseSection() {
  return (
    <div className="space-y-8 lg:space-y-12">
      {/* Header */}
      <div className="text-center relative">
        <div className="absolute inset-0 bg-gradient-to-br from-blue-50 via-white to-purple-50/50 dark:from-blue-950/20 dark:via-slate-900 dark:to-purple-950/20 rounded-3xl -z-10" />
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,rgba(59,130,246,0.1),transparent_70%)] dark:bg-[radial-gradient(circle_at_50%_50%,rgba(59,130,246,0.05),transparent_70%)] rounded-3xl -z-10" />
        
        <div className="py-8 lg:py-12">
          <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-2xl mb-6 shadow-2xl shadow-blue-500/25">
            <Star className="h-8 w-8 text-white" />
          </div>
          <h1 className="text-3xl lg:text-4xl font-bold mb-4 text-slate-900 dark:text-slate-100">Why Choose Kimbap.io?</h1>
          <p className="text-lg lg:text-xl text-slate-600 dark:text-slate-300 max-w-3xl mx-auto leading-relaxed">
            Enterprise-grade AI tool orchestration designed for security, reliability, and scale.
          </p>
        </div>
      </div>

      {/* Core Value Propositions */}
      <div className="grid lg:grid-cols-2 gap-8">
        {/* Security First */}
        <Card className="group relative overflow-hidden border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm hover:shadow-2xl hover:shadow-green-500/10 transition-all duration-500 hover:scale-[1.02] dark:border-slate-700/60">
          <div className="absolute inset-0 bg-gradient-to-br from-green-50/50 to-emerald-50/20 dark:from-green-950/20 dark:to-emerald-950/10" />
          <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-green-500/10 to-emerald-500/10 rounded-full -mr-16 -mt-16 transition-all duration-500 group-hover:scale-150" />
          
          <CardHeader className="relative pb-4">
            <div className="flex items-center gap-3 mb-4">
              <div className="h-10 w-10 bg-gradient-to-br from-green-600 to-emerald-600 rounded-xl flex items-center justify-center shadow-lg">
                <Shield className="h-5 w-5 text-white" />
              </div>
              <CardTitle className="text-xl font-bold text-slate-900 dark:text-slate-100">Uncompromising Security</CardTitle>
            </div>
          </CardHeader>
          <CardContent className="relative space-y-4">
            <p className="text-slate-600 dark:text-slate-300 text-sm">
              Zero-knowledge proof encryption ensures your credentials remain secure even if our servers are compromised.
            </p>
            <div className="space-y-3">
              <div className="flex items-start gap-3 text-sm">
                <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5" />
                <div>
                  <span className="font-medium text-slate-900 dark:text-slate-100">Local Infrastructure</span>
                  <p className="text-slate-600 dark:text-slate-400 text-xs">MCP Core runs entirely on your servers</p>
                </div>
              </div>
              <div className="flex items-start gap-3 text-sm">
                <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5" />
                <div>
                  <span className="font-medium text-slate-900 dark:text-slate-100">Dual Encryption</span>
                  <p className="text-slate-600 dark:text-slate-400 text-xs">Master Password + Access Token protection</p>
                </div>
              </div>
              <div className="flex items-start gap-3 text-sm">
                <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5" />
                <div>
                  <span className="font-medium text-slate-900 dark:text-slate-100">Zero-Knowledge Architecture</span>
                  <p className="text-slate-600 dark:text-slate-400 text-xs">We can't access your plaintext credentials</p>
                </div>
              </div>
            </div>
            <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs border-green-200 dark:border-green-800">
              Military-grade encryption standards
            </Badge>
          </CardContent>
        </Card>

        {/* Reliable & Vetted Tools */}
        <Card className="group relative overflow-hidden border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm hover:shadow-2xl hover:shadow-blue-500/10 transition-all duration-500 hover:scale-[1.02] dark:border-slate-700/60">
          <div className="absolute inset-0 bg-gradient-to-br from-blue-50/50 to-cyan-50/20 dark:from-blue-950/20 dark:to-cyan-950/10" />
          <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-blue-500/10 to-cyan-500/10 rounded-full -mr-16 -mt-16 transition-all duration-500 group-hover:scale-150" />
          
          <CardHeader className="relative pb-4">
            <div className="flex items-center gap-3 mb-4">
              <div className="h-10 w-10 bg-gradient-to-br from-blue-600 to-cyan-600 rounded-xl flex items-center justify-center shadow-lg">
                <CheckCircle className="h-5 w-5 text-white" />
              </div>
              <CardTitle className="text-xl font-bold text-slate-900 dark:text-slate-100">Vetted & Reliable Tools</CardTitle>
            </div>
          </CardHeader>
          <CardContent className="relative space-y-4">
            <p className="text-slate-600 dark:text-slate-300 text-sm">
              Curated library of production-ready tools, rigorously tested for security and reliability.
            </p>
            <div className="space-y-3">
              <div className="flex items-start gap-3 text-sm">
                <CheckCircle className="h-4 w-4 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
                <div>
                  <span className="font-medium text-slate-900 dark:text-slate-100">Quality Assurance</span>
                  <p className="text-slate-600 dark:text-slate-400 text-xs">All tools tested by our security team</p>
                </div>
              </div>
              <div className="flex items-start gap-3 text-sm">
                <CheckCircle className="h-4 w-4 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
                <div>
                  <span className="font-medium text-slate-900 dark:text-slate-100">Production Ready</span>
                  <p className="text-slate-600 dark:text-slate-400 text-xs">Notion, GitHub, databases, and more</p>
                </div>
              </div>
              <div className="flex items-start gap-3 text-sm">
                <CheckCircle className="h-4 w-4 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
                <div>
                  <span className="font-medium text-slate-900 dark:text-slate-100">Continuous Updates</span>
                  <p className="text-slate-600 dark:text-slate-400 text-xs">Regular security patches and improvements</p>
                </div>
              </div>
            </div>
            <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs border-blue-200 dark:border-blue-800">
              100% uptime SLA guarantee
            </Badge>
          </CardContent>
        </Card>

        {/* Multi-Tenant Management */}
        <Card className="group relative overflow-hidden border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm hover:shadow-2xl hover:shadow-purple-500/10 transition-all duration-500 hover:scale-[1.02] dark:border-slate-700/60">
          <div className="absolute inset-0 bg-gradient-to-br from-purple-50/50 to-violet-50/20 dark:from-purple-950/20 dark:to-violet-950/10" />
          <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-purple-500/10 to-violet-500/10 rounded-full -mr-16 -mt-16 transition-all duration-500 group-hover:scale-150" />
          
          <CardHeader className="relative pb-4">
            <div className="flex items-center gap-3 mb-4">
              <div className="h-10 w-10 bg-gradient-to-br from-purple-600 to-violet-600 rounded-xl flex items-center justify-center shadow-lg">
                <Users className="h-5 w-5 text-white" />
              </div>
              <CardTitle className="text-xl font-bold text-slate-900 dark:text-slate-100">Enterprise Team Management</CardTitle>
            </div>
          </CardHeader>
          <CardContent className="relative space-y-4">
            <p className="text-slate-600 dark:text-slate-300 text-sm">
              Sophisticated access control and permission system designed for enterprise teams and multi-tenancy.
            </p>
            <div className="space-y-3">
              <div className="flex items-start gap-3 text-sm">
                <CheckCircle className="h-4 w-4 text-purple-600 dark:text-purple-400 flex-shrink-0 mt-0.5" />
                <div>
                  <span className="font-medium text-slate-900 dark:text-slate-100">Access Token Distribution</span>
                  <p className="text-slate-600 dark:text-slate-400 text-xs">Secure token-based team member onboarding</p>
                </div>
              </div>
              <div className="flex items-start gap-3 text-sm">
                <CheckCircle className="h-4 w-4 text-purple-600 dark:text-purple-400 flex-shrink-0 mt-0.5" />
                <div>
                  <span className="font-medium text-slate-900 dark:text-slate-100">Fine-Grained Permissions</span>
                  <p className="text-slate-600 dark:text-slate-400 text-xs">Control access by tool, function, and resource</p>
                </div>
              </div>
              <div className="flex items-start gap-3 text-sm">
                <CheckCircle className="h-4 w-4 text-purple-600 dark:text-purple-400 flex-shrink-0 mt-0.5" />
                <div>
                  <span className="font-medium text-slate-900 dark:text-slate-100">Multi-Tenant Architecture</span>
                  <p className="text-slate-600 dark:text-slate-400 text-xs">Isolated environments for different teams</p>
                </div>
              </div>
            </div>
            <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 text-xs border-purple-200 dark:border-purple-800">
              Enterprise-grade permission system
            </Badge>
          </CardContent>
        </Card>

        {/* Cross-Platform Support */}
        <Card className="group relative overflow-hidden border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm hover:shadow-2xl hover:shadow-orange-500/10 transition-all duration-500 hover:scale-[1.02] dark:border-slate-700/60">
          <div className="absolute inset-0 bg-gradient-to-br from-orange-50/50 to-amber-50/20 dark:from-orange-950/20 dark:to-amber-950/10" />
          <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-orange-500/10 to-amber-500/10 rounded-full -mr-16 -mt-16 transition-all duration-500 group-hover:scale-150" />
          
          <CardHeader className="relative pb-4">
            <div className="flex items-center gap-3 mb-4">
              <div className="h-10 w-10 bg-gradient-to-br from-orange-600 to-amber-600 rounded-xl flex items-center justify-center shadow-lg">
                <Globe className="h-5 w-5 text-white" />
              </div>
              <CardTitle className="text-xl font-bold text-slate-900 dark:text-slate-100">Universal Compatibility</CardTitle>
            </div>
          </CardHeader>
          <CardContent className="relative space-y-4">
            <p className="text-slate-600 dark:text-slate-300 text-sm">
              Native support for all major platforms and AI assistants with seamless deployment across environments.
            </p>
            <div className="space-y-3">
              <div className="flex items-start gap-3 text-sm">
                <CheckCircle className="h-4 w-4 text-orange-600 dark:text-orange-400 flex-shrink-0 mt-0.5" />
                <div>
                  <span className="font-medium text-slate-900 dark:text-slate-100">Cross-Platform Desktop</span>
                  <p className="text-slate-600 dark:text-slate-400 text-xs">Windows, macOS, Linux native applications</p>
                </div>
              </div>
              <div className="flex items-start gap-3 text-sm">
                <CheckCircle className="h-4 w-4 text-orange-600 dark:text-orange-400 flex-shrink-0 mt-0.5" />
                <div>
                  <span className="font-medium text-slate-900 dark:text-slate-100">AI Assistant Support</span>
                  <p className="text-slate-600 dark:text-slate-400 text-xs">Claude Desktop, Cursor, and MCP-compatible clients</p>
                </div>
              </div>
              <div className="flex items-start gap-3 text-sm">
                <CheckCircle className="h-4 w-4 text-orange-600 dark:text-orange-400 flex-shrink-0 mt-0.5" />
                <div>
                  <span className="font-medium text-slate-900 dark:text-slate-100">Flexible Deployment</span>
                  <p className="text-slate-600 dark:text-slate-400 text-xs">Docker, native install, or cloud infrastructure</p>
                </div>
              </div>
            </div>
            <Badge className="bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 text-xs border-orange-200 dark:border-orange-800">
              Works everywhere, deploys anywhere
            </Badge>
          </CardContent>
        </Card>
      </div>

      {/* Key Differentiators */}
      <Card className="border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-xl hover:shadow-2xl transition-all duration-300 dark:border-slate-700/60">
        <CardHeader>
          <CardTitle className="text-2xl lg:text-3xl text-center text-slate-900 dark:text-slate-100">Key Differentiators</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid md:grid-cols-3 gap-6 text-center">
            <div className="space-y-4">
              <div className="h-12 w-12 bg-gradient-to-br from-green-500 to-emerald-600 rounded-xl flex items-center justify-center mx-auto">
                <Lock className="h-6 w-6 text-white" />
              </div>
              <div>
                <h4 className="font-semibold text-slate-900 dark:text-slate-100 mb-2">Zero-Knowledge Security</h4>
                <p className="text-sm text-slate-600 dark:text-slate-300">
                  Your data stays encrypted even from us. Military-grade zero-knowledge proof architecture.
                </p>
              </div>
            </div>
            <div className="space-y-4">
              <div className="h-12 w-12 bg-gradient-to-br from-blue-500 to-cyan-600 rounded-xl flex items-center justify-center mx-auto">
                <Database className="h-6 w-6 text-white" />
              </div>
              <div>
                <h4 className="font-semibold text-slate-900 dark:text-slate-100 mb-2">Self-Hosted Control</h4>
                <p className="text-sm text-slate-600 dark:text-slate-300">
                  Deploy on your infrastructure. Complete control over data, security, and compliance.
                </p>
              </div>
            </div>
            <div className="space-y-4">
              <div className="h-12 w-12 bg-gradient-to-br from-purple-500 to-violet-600 rounded-xl flex items-center justify-center mx-auto">
                <Crown className="h-6 w-6 text-white" />
              </div>
              <div>
                <h4 className="font-semibold text-slate-900 dark:text-slate-100 mb-2">Enterprise Ready</h4>
                <p className="text-sm text-slate-600 dark:text-slate-300">
                  Built for teams. Advanced permissions, audit logs, and multi-tenant architecture.
                </p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}