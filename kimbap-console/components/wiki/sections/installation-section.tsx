import { Download, CheckCircle, Monitor, Server, Code, Copy, Shield, Settings, Zap } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function InstallationSection() {
  return (
    <div className="space-y-8 lg:space-y-12">
      {/* Header */}
      <div className="text-center relative">
        <div className="absolute inset-0 bg-gradient-to-br from-blue-50 via-white to-cyan-50/50 dark:from-blue-950/20 dark:via-slate-900 dark:to-cyan-950/20 rounded-3xl -z-10" />
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,rgba(59,130,246,0.1),transparent_70%)] dark:bg-[radial-gradient(circle_at_50%_50%,rgba(59,130,246,0.05),transparent_70%)] rounded-3xl -z-10" />
        
        <div className="py-8 lg:py-12">
          <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-blue-500 to-cyan-600 rounded-2xl mb-6 shadow-2xl shadow-blue-500/25">
            <Download className="h-8 w-8 text-white" />
          </div>
          <h1 className="text-3xl lg:text-4xl font-bold mb-4 text-slate-900 dark:text-slate-100">
            Install <span className="bg-gradient-to-r from-blue-600 via-cyan-600 to-blue-800 bg-clip-text text-transparent">MCP Console</span>
          </h1>
          <p className="text-lg lg:text-xl text-slate-600 dark:text-slate-300 max-w-2xl mx-auto leading-relaxed">
            Quick and easy installation on any platform. Minimal hardware requirements and universal compatibility.
          </p>
        </div>
      </div>

      {/* Low Requirements Highlight */}
      <Card className="border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-xl hover:shadow-2xl transition-all duration-300 dark:border-slate-700/60">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-slate-900 dark:text-slate-100">
            <CheckCircle className="h-6 w-6 text-green-600 dark:text-green-400" />
            Minimal System Requirements
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="text-center p-4 bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950/20 dark:to-emerald-950/10 rounded-lg border border-slate-200/60 dark:border-slate-700/60">
              <div className="text-2xl font-bold text-green-600 dark:text-green-400 mb-1">2GB</div>
              <p className="text-sm font-medium text-slate-900 dark:text-slate-100">RAM Minimum</p>
              <p className="text-xs text-slate-500 dark:text-slate-400">Runs efficiently on modest hardware</p>
            </div>
            <div className="text-center p-4 bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950/20 dark:to-emerald-950/10 rounded-lg border border-slate-200/60 dark:border-slate-700/60">
              <div className="text-2xl font-bold text-green-600 dark:text-green-400 mb-1">1GB</div>
              <p className="text-sm font-medium text-slate-900 dark:text-slate-100">Storage Space</p>
              <p className="text-xs text-slate-500 dark:text-slate-400">Lightweight installation</p>
            </div>
            <div className="text-center p-4 bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950/20 dark:to-emerald-950/10 rounded-lg border border-slate-200/60 dark:border-slate-700/60">
              <div className="text-2xl font-bold text-green-600 dark:text-green-400 mb-1">Any</div>
              <p className="text-sm font-medium text-slate-900 dark:text-slate-100">CPU Architecture</p>
              <p className="text-xs text-slate-500 dark:text-slate-400">x86, ARM, Apple Silicon</p>
            </div>
            <div className="text-center p-4 bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950/20 dark:to-emerald-950/10 rounded-lg border border-slate-200/60 dark:border-slate-700/60">
              <div className="text-2xl font-bold text-green-600 dark:text-green-400 mb-1">All</div>
              <p className="text-sm font-medium text-slate-900 dark:text-slate-100">Platforms</p>
              <p className="text-xs text-slate-500 dark:text-slate-400">Windows, macOS, Linux</p>
            </div>
          </div>
          <div className="text-center mt-6">
            <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 border-green-200 dark:border-green-800">
              Works on virtually any computer or server
            </Badge>
          </div>
        </CardContent>
      </Card>

      {/* Universal Platform Support */}
      <Card className="border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-xl hover:shadow-2xl transition-all duration-300 dark:border-slate-700/60">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-slate-900 dark:text-slate-100">
            <Monitor className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            Universal Platform Support
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue="windows" className="w-full">
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="windows">Windows</TabsTrigger>
              <TabsTrigger value="macos">macOS</TabsTrigger>
              <TabsTrigger value="linux">Linux</TabsTrigger>
            </TabsList>
            
            <TabsContent value="windows" className="space-y-4">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-3">
                  <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Windows Installation</h4>
                  <div className="space-y-2">
                    <div className="flex items-center gap-2 p-2 bg-blue-50 dark:bg-blue-950/20 rounded border border-blue-200 dark:border-blue-800">
                      <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                      <span className="text-sm text-slate-700 dark:text-slate-300">Windows 10/11 (64-bit)</span>
                    </div>
                    <div className="flex items-center gap-2 p-2 bg-blue-50 dark:bg-blue-950/20 rounded border border-blue-200 dark:border-blue-800">
                      <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                      <span className="text-sm text-slate-700 dark:text-slate-300">PowerShell 5.1+</span>
                    </div>
                  </div>
                  <Button className="w-full bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 shadow-lg hover:shadow-xl transition-all duration-200">
                    <Download className="h-4 w-4 mr-2" />
                    Download Windows Installer
                  </Button>
                </div>
                
                <div className="space-y-3">
                  <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Quick Install</h4>
                  <div className="bg-slate-900 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-slate-300 text-xs">PowerShell</span>
                      <Button size="sm" variant="ghost" className="h-6 text-xs text-slate-300 hover:bg-slate-800">
                        <Copy className="h-3 w-3 mr-1" />
                        Copy
                      </Button>
                    </div>
                    <div className="text-slate-100 font-mono text-xs space-y-1">
                      <div className="text-yellow-400"># One-line install</div>
                      <div className="text-green-400">iwr https://get.kimbap.io/install.ps1 | iex</div>
                      <div className="text-green-400">kimbap-console start</div>
                    </div>
                  </div>
                </div>
              </div>
            </TabsContent>
            
            <TabsContent value="macos" className="space-y-4">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-3">
                  <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">macOS Installation</h4>
                  <div className="space-y-2">
                    <div className="flex items-center gap-2 p-2 bg-blue-50 dark:bg-blue-950/20 rounded border border-blue-200 dark:border-blue-800">
                      <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                      <span className="text-sm text-slate-700 dark:text-slate-300">macOS 10.15+ (Catalina)</span>
                    </div>
                    <div className="flex items-center gap-2 p-2 bg-blue-50 dark:bg-blue-950/20 rounded border border-blue-200 dark:border-blue-800">
                      <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                      <span className="text-sm text-slate-700 dark:text-slate-300">Intel & Apple Silicon</span>
                    </div>
                  </div>
                  <Button className="w-full bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 shadow-lg hover:shadow-xl transition-all duration-200">
                    <Download className="h-4 w-4 mr-2" />
                    Download macOS App
                  </Button>
                </div>
                
                <div className="space-y-3">
                  <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Terminal Install</h4>
                  <div className="bg-slate-900 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-slate-300 text-xs">Terminal</span>
                      <Button size="sm" variant="ghost" className="h-6 text-xs text-slate-300 hover:bg-slate-800">
                        <Copy className="h-3 w-3 mr-1" />
                        Copy
                      </Button>
                    </div>
                    <div className="text-slate-100 font-mono text-xs space-y-1">
                      <div className="text-yellow-400"># Homebrew install</div>
                      <div className="text-green-400">brew install kimbap-io/tap/kimbap-console</div>
                      <div className="text-green-400">kimbap-console start</div>
                    </div>
                  </div>
                </div>
              </div>
            </TabsContent>
            
            <TabsContent value="linux" className="space-y-4">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-3">
                  <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Linux Installation</h4>
                  <div className="space-y-2">
                    <div className="flex items-center gap-2 p-2 bg-blue-50 dark:bg-blue-950/20 rounded border border-blue-200 dark:border-blue-800">
                      <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                      <span className="text-sm text-slate-700 dark:text-slate-300">Ubuntu 18.04+, CentOS 7+, Debian 9+</span>
                    </div>
                    <div className="flex items-center gap-2 p-2 bg-blue-50 dark:bg-blue-950/20 rounded border border-blue-200 dark:border-blue-800">
                      <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                      <span className="text-sm text-slate-700 dark:text-slate-300">x86_64, ARM64, ARMv7</span>
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <Button className="bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 shadow-lg hover:shadow-xl transition-all duration-200">
                      <Download className="h-4 w-4 mr-2" />
                      x86_64
                    </Button>
                    <Button className="bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 shadow-lg hover:shadow-xl transition-all duration-200">
                      <Download className="h-4 w-4 mr-2" />
                      ARM64
                    </Button>
                  </div>
                </div>
                
                <div className="space-y-3">
                  <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Package Manager</h4>
                  <div className="bg-slate-900 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-slate-300 text-xs">Shell</span>
                      <Button size="sm" variant="ghost" className="h-6 text-xs text-slate-300 hover:bg-slate-800">
                        <Copy className="h-3 w-3 mr-1" />
                        Copy
                      </Button>
                    </div>
                    <div className="text-slate-100 font-mono text-xs space-y-1">
                      <div className="text-yellow-400"># Install script</div>
                      <div className="text-green-400">curl -fsSL https://get.kimbap.io/install.sh | sh</div>
                      <div className="text-green-400">kimbap-console start</div>
                    </div>
                  </div>
                </div>
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

      {/* Docker Option */}
      <Card className="border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-xl hover:shadow-2xl transition-all duration-300 dark:border-slate-700/60">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-slate-900 dark:text-slate-100">
            <Server className="h-5 w-5 text-purple-600 dark:text-purple-400" />
            Docker Deployment (Recommended)
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-slate-600 dark:text-slate-300">
            Docker provides the easiest and most consistent deployment experience across all platforms.
          </p>
          
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="space-y-3">
              <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Quick Start with Docker</h4>
              <div className="bg-slate-900 rounded-lg p-4">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-slate-300 text-xs">Docker Compose</span>
                  <Button size="sm" variant="ghost" className="h-6 text-xs text-slate-300 hover:bg-slate-800">
                    <Copy className="h-3 w-3 mr-1" />
                    Copy
                  </Button>
                </div>
                <div className="text-slate-100 font-mono text-xs space-y-1">
                  <div className="text-yellow-400"># Download and start</div>
                  <div className="text-green-400">curl -O https://get.kimbap.io/docker-compose.yml</div>
                  <div className="text-green-400">docker-compose up -d</div>
                  <div></div>
                  <div className="text-yellow-400"># Open browser</div>
                  <div className="text-green-400">open http://localhost:8080</div>
                </div>
              </div>
            </div>
            
            <div className="space-y-3">
              <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Docker Benefits</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-3">
                  <div className="h-2 w-2 bg-purple-600 rounded-full flex-shrink-0 mt-2"></div>
                  <div>
                    <p className="text-sm font-medium text-slate-900 dark:text-slate-100">Consistent Environment</p>
                    <p className="text-xs text-slate-500 dark:text-slate-400">Same setup across all platforms</p>
                  </div>
                </li>
                <li className="flex items-start gap-3">
                  <div className="h-2 w-2 bg-purple-600 rounded-full flex-shrink-0 mt-2"></div>
                  <div>
                    <p className="text-sm font-medium text-slate-900 dark:text-slate-100">Easy Updates</p>
                    <p className="text-xs text-slate-500 dark:text-slate-400">Pull new images for updates</p>
                  </div>
                </li>
                <li className="flex items-start gap-3">
                  <div className="h-2 w-2 bg-purple-600 rounded-full flex-shrink-0 mt-2"></div>
                  <div>
                    <p className="text-sm font-medium text-slate-900 dark:text-slate-100">Isolated Dependencies</p>
                    <p className="text-xs text-slate-500 dark:text-slate-400">No conflicts with system packages</p>
                  </div>
                </li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Post Installation */}
      <Card className="border-slate-200/60 bg-white/80 dark:bg-slate-800/80 backdrop-blur-sm shadow-xl hover:shadow-2xl transition-all duration-300 dark:border-slate-700/60">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-slate-900 dark:text-slate-100">
            <Settings className="h-5 w-5 text-orange-600 dark:text-orange-400" />
            After Installation
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="text-center p-4 bg-gradient-to-br from-orange-50 to-amber-50 dark:from-orange-950/20 dark:to-amber-950/10 rounded-lg border border-slate-200/60 dark:border-slate-700/60">
              <div className="w-10 h-10 bg-gradient-to-br from-orange-100 to-amber-100 dark:from-orange-800 dark:to-amber-800 rounded-full flex items-center justify-center mx-auto mb-2">
                <span className="font-bold text-orange-600 dark:text-orange-300">1</span>
              </div>
              <p className="font-medium text-sm text-slate-900 dark:text-slate-100">Access Console</p>
              <p className="text-xs text-slate-500 dark:text-slate-400">Open browser to localhost:8080</p>
            </div>
            <div className="text-center p-4 bg-gradient-to-br from-orange-50 to-amber-50 dark:from-orange-950/20 dark:to-amber-950/10 rounded-lg border border-slate-200/60 dark:border-slate-700/60">
              <div className="w-10 h-10 bg-gradient-to-br from-orange-100 to-amber-100 dark:from-orange-800 dark:to-amber-800 rounded-full flex items-center justify-center mx-auto mb-2">
                <span className="font-bold text-orange-600 dark:text-orange-300">2</span>
              </div>
              <p className="font-medium text-sm text-slate-900 dark:text-slate-100">Initialize Server</p>
              <p className="text-xs text-slate-500 dark:text-slate-400">Run the setup wizard</p>
            </div>
            <div className="text-center p-4 bg-gradient-to-br from-orange-50 to-amber-50 dark:from-orange-950/20 dark:to-amber-950/10 rounded-lg border border-slate-200/60 dark:border-slate-700/60">
              <div className="w-10 h-10 bg-gradient-to-br from-orange-100 to-amber-100 dark:from-orange-800 dark:to-amber-800 rounded-full flex items-center justify-center mx-auto mb-2">
                <span className="font-bold text-orange-600 dark:text-orange-300">3</span>
              </div>
              <p className="font-medium text-sm text-slate-900 dark:text-slate-100">Start Using</p>
              <p className="text-xs text-slate-500 dark:text-slate-400">Add tools and invite team</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}