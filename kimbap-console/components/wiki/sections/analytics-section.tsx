import { BarChart3, TrendingUp, Users, Activity } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

export function AnalyticsSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-purple-500 to-indigo-600 rounded-2xl mb-4">
          <BarChart3 className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">Usage Analytics</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Monitor your team's AI tool usage and server performance.
        </p>
      </div>

      {/* Analytics Overview */}
      <Card className="border-2 border-slate-200 dark:border-slate-700">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            Analytics Dashboard
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Key Metrics */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <Card className="border-blue-200 dark:border-blue-800 bg-gradient-to-br from-blue-50 to-blue-100 dark:from-blue-950/30 dark:to-blue-900/30">
              <CardContent className="pt-4">
                <div className="text-center space-y-2">
                  <Activity className="h-6 w-6 text-blue-600 dark:text-blue-400 mx-auto" />
                  <div className="text-2xl font-bold text-blue-900 dark:text-blue-200">2.4M</div>
                  <div className="text-xs text-blue-700 dark:text-blue-300">Total API Calls</div>
                  <div className="flex items-center justify-center gap-1">
                    <TrendingUp className="h-3 w-3 text-green-600 dark:text-green-400" />
                    <span className="text-xs text-green-600 dark:text-green-400">+12.5%</span>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="border-green-200 dark:border-green-800 bg-gradient-to-br from-green-50 to-green-100 dark:from-green-950/30 dark:to-green-900/30">
              <CardContent className="pt-4">
                <div className="text-center space-y-2">
                  <Users className="h-6 w-6 text-green-600 dark:text-green-400 mx-auto" />
                  <div className="text-2xl font-bold text-green-900 dark:text-green-200">147</div>
                  <div className="text-xs text-green-700 dark:text-green-300">Active Users</div>
                  <div className="flex items-center justify-center gap-1">
                    <TrendingUp className="h-3 w-3 text-green-600 dark:text-green-400" />
                    <span className="text-xs text-green-600 dark:text-green-400">+8.2%</span>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="border-purple-200 dark:border-purple-800 bg-gradient-to-br from-purple-50 to-purple-100 dark:from-purple-950/30 dark:to-purple-900/30">
              <CardContent className="pt-4">
                <div className="text-center space-y-2">
                  <BarChart3 className="h-6 w-6 text-purple-600 dark:text-purple-400 mx-auto" />
                  <div className="text-2xl font-bold text-purple-900 dark:text-purple-200">89%</div>
                  <div className="text-xs text-purple-700 dark:text-purple-300">Tool Utilization</div>
                  <div className="flex items-center justify-center gap-1">
                    <TrendingUp className="h-3 w-3 text-purple-600 dark:text-purple-400" />
                    <span className="text-xs text-purple-600 dark:text-purple-400">+2.1%</span>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="border-orange-200 dark:border-orange-800 bg-gradient-to-br from-orange-50 to-orange-100 dark:from-orange-950/30 dark:to-orange-900/30">
              <CardContent className="pt-4">
                <div className="text-center space-y-2">
                  <Activity className="h-6 w-6 text-orange-600 dark:text-orange-400 mx-auto" />
                  <div className="text-2xl font-bold text-orange-900 dark:text-orange-200">156ms</div>
                  <div className="text-xs text-orange-700 dark:text-orange-300">Avg Response</div>
                  <div className="flex items-center justify-center gap-1">
                    <TrendingUp className="h-3 w-3 text-green-600 dark:text-green-400" />
                    <span className="text-xs text-green-600 dark:text-green-400">-5.8%</span>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>

      {/* Recent Activity */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Activity className="h-4 w-4" />
              Recent Activity
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded-lg">
              <div>
                <p className="font-medium text-sm">API Calls Today</p>
                <p className="text-xs text-muted-foreground">34,521 successful requests</p>
              </div>
              <div className="text-right">
                <div className="text-lg font-bold text-green-600 dark:text-green-400">+18%</div>
                <div className="text-xs text-muted-foreground">vs yesterday</div>
              </div>
            </div>

            <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded-lg">
              <div>
                <p className="font-medium text-sm">Active Users</p>
                <p className="text-xs text-muted-foreground">73 users in last 24h</p>
              </div>
              <div className="text-right">
                <div className="text-lg font-bold text-purple-600 dark:text-purple-400">+5%</div>
                <div className="text-xs text-muted-foreground">vs yesterday</div>
              </div>
            </div>

            <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded-lg">
              <div>
                <p className="font-medium text-sm">Error Rate</p>
                <p className="text-xs text-muted-foreground">1.2% of total requests</p>
              </div>
              <div className="text-right">
                <div className="text-lg font-bold text-green-600 dark:text-green-400">-0.3%</div>
                <div className="text-xs text-muted-foreground">improvement</div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <BarChart3 className="h-4 w-4" />
              Most Used Tools
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 bg-gray-900 rounded flex items-center justify-center">
                  <span className="text-white text-xs font-bold">GH</span>
                </div>
                <div>
                  <p className="font-medium text-sm">GitHub Integration</p>
                  <p className="text-xs text-muted-foreground">12,453 requests today</p>
                </div>
              </div>
              <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">85% usage</Badge>
            </div>

            <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 bg-black rounded flex items-center justify-center">
                  <span className="text-white text-xs font-bold">N</span>
                </div>
                <div>
                  <p className="font-medium text-sm">Notion Workspace</p>
                  <p className="text-xs text-muted-foreground">8,721 requests today</p>
                </div>
              </div>
              <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">67% usage</Badge>
            </div>

            <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 bg-blue-600 rounded flex items-center justify-center">
                  <span className="text-white text-xs font-bold">DB</span>
                </div>
                <div>
                  <p className="font-medium text-sm">PostgreSQL</p>
                  <p className="text-xs text-muted-foreground">5,234 queries today</p>
                </div>
              </div>
              <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 text-xs">42% usage</Badge>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Analytics Features */}
      <Card className="border-2 border-purple-200 dark:border-purple-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-purple-800 dark:text-purple-300">
            <BarChart3 className="h-5 w-5" />
            Analytics Features
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="text-center p-4 bg-purple-50 dark:bg-purple-950/30 rounded-lg">
              <Activity className="h-8 w-8 mx-auto mb-2 text-purple-600 dark:text-purple-400" />
              <p className="font-medium text-sm">Real-time Monitoring</p>
              <p className="text-xs text-muted-foreground">Live usage statistics and performance metrics</p>
            </div>
            <div className="text-center p-4 bg-blue-50 dark:bg-blue-950/30 rounded-lg">
              <Users className="h-8 w-8 mx-auto mb-2 text-blue-600 dark:text-blue-400" />
              <p className="font-medium text-sm">User Analytics</p>
              <p className="text-xs text-muted-foreground">Individual and team usage patterns</p>
            </div>
            <div className="text-center p-4 bg-green-50 dark:bg-green-950/30 rounded-lg">
              <TrendingUp className="h-8 w-8 mx-auto mb-2 text-green-600 dark:text-green-400" />
              <p className="font-medium text-sm">Usage Trends</p>
              <p className="text-xs text-muted-foreground">Historical data and growth insights</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Best Practices */}
      <Card className="bg-gradient-to-r from-indigo-50 dark:from-indigo-950 to-purple-50 dark:to-purple-950 border-indigo-200 dark:border-indigo-800">
        <CardHeader>
          <CardTitle className="text-indigo-800 dark:text-indigo-300">Analytics Best Practices</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-3">
              <h4 className="font-semibold text-indigo-800 dark:text-indigo-300">Monitoring Guidelines</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <BarChart3 className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Regular Review Schedule</p>
                    <p className="text-xs text-muted-foreground">Check key metrics weekly</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <TrendingUp className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Track Trends</p>
                    <p className="text-xs text-muted-foreground">Focus on patterns over individual data points</p>
                  </div>
                </li>
              </ul>
            </div>
            
            <div className="space-y-3">
              <h4 className="font-semibold text-indigo-800 dark:text-indigo-300">Optimization Tips</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <Users className="h-4 w-4 text-purple-600 dark:text-purple-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">User Adoption Focus</p>
                    <p className="text-xs text-muted-foreground">Identify unused tools and provide training</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Activity className="h-4 w-4 text-orange-600 dark:text-orange-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Performance Optimization</p>
                    <p className="text-xs text-muted-foreground">Monitor response times and optimize slow tools</p>
                  </div>
                </li>
              </ul>
            </div>
          </div>

          <Alert className="bg-white dark:bg-slate-800 border-indigo-200 dark:border-indigo-800 mt-6">
            <BarChart3 className="h-4 w-4" />
            <AlertDescription className="text-sm">
              <strong>Analytics Tip:</strong> Use analytics data to understand your team's AI tool usage patterns and optimize your MCP server configuration.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    </div>
  )
}