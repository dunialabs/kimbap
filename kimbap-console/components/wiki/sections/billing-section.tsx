import { DollarSign, Receipt, CheckCircle, Settings, Bell, RefreshCw, Download, Calendar, Crown, TrendingUp, TrendingDown, PieChart, BarChart3, Info } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { getPricingUrl } from "@/lib/plan-config"

export function BillingSection() {
  return (
    <div className="space-y-8">
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-green-500 to-emerald-600 rounded-2xl mb-4">
          <DollarSign className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">License & Usage Management</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Comprehensive overview and usage analytics for your Kimbap.io MCP platform.
          Monitor consumption, manage licenses, and optimize your resource allocation.
        </p>
      </div>

      <Card className="border-2 border-green-200 dark:border-green-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-green-800 dark:text-green-300">
            <Receipt className="h-5 w-5" />
            License Overview
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="bg-gradient-to-br from-green-50 to-emerald-100 dark:from-green-950/30 dark:to-emerald-900/30 rounded-lg p-6 border border-green-200 dark:border-green-800">
            <div className="space-y-4">
              <div>
                <h3 className="text-lg font-bold text-green-800 dark:text-green-300 mb-2">Community Plan</h3>
                <p className="text-sm text-green-700 dark:text-green-300">Kimbap MCP Console includes a free plan with generous limits and no time restrictions.</p>
              </div>
              <div className="space-y-2 text-sm">
                <div className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5 flex-shrink-0" />
                  <span className="text-green-700 dark:text-green-300">Up to 30 tools</span>
                </div>
                <div className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5 flex-shrink-0" />
                  <span className="text-green-700 dark:text-green-300">Up to 30 access tokens</span>
                </div>
                <div className="flex items-start gap-2">
                  <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5 flex-shrink-0" />
                  <span className="text-green-700 dark:text-green-300">No time limit</span>
                </div>
              </div>
              <div className="pt-4 border-t border-green-200 dark:border-green-700">
                <p className="text-sm text-green-700 dark:text-green-300 mb-3">For higher limits and additional features, visit <strong>kimbap.io/pricing</strong> to get a license key.</p>
                <p className="text-sm text-green-700 dark:text-green-300">Activate your license on the License page in the dashboard.</p>
              </div>
            </div>
          </div>

          <Alert className="bg-green-50 dark:bg-green-950/30 border-green-200 dark:border-green-800">
            <Info className="h-4 w-4" />
            <AlertDescription className="text-sm">
              <strong>License Management:</strong> Your Community plan includes up to 30 tools and 30 access tokens. To increase these limits, activate a license key from kimbap.io/pricing on the License page.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>

      <Card className="border-2 border-blue-200 dark:border-blue-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-blue-800 dark:text-blue-300">
            <Crown className="h-5 w-5" />
            Plan Information
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-4">
            <div className="space-y-3">
              <h4 className="font-semibold">Community Plan Features</h4>
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-sm">
                  <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                  <span>Up to 30 tools</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                  <span>Up to 30 access tokens</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                  <span>No time limit</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                  <span>Community support</span>
                </div>
              </div>
            </div>
            <div className="pt-4 border-t border-blue-200 dark:border-blue-700">
              <h4 className="font-semibold mb-3">Upgrade Your Plan</h4>
              <p className="text-sm text-muted-foreground mb-4">Need more tools or tokens? Visit kimbap.io/pricing to explore premium plans with higher limits and additional features.</p>
              <Button
                className="w-full bg-blue-600 hover:bg-blue-700"
                onClick={() => window.open(getPricingUrl(), '_blank', 'noopener,noreferrer')}
              >
                <TrendingUp className="h-4 w-4 mr-2" />
                Get License
              </Button>
            </div>
          </div>

          <Alert className="bg-blue-50 dark:bg-blue-950/30 border-blue-200 dark:border-blue-800">
            <Info className="h-4 w-4" />
            <AlertDescription className="text-sm">
              <strong>License Activation:</strong> To activate a premium license, visit the License page in your dashboard and enter your license key from kimbap.io/pricing.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>

      <Card className="border-2 border-indigo-200 dark:border-indigo-800">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-indigo-800 dark:text-indigo-300">
            <BarChart3 className="h-5 w-5" />
            Usage Analytics & Trends
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <Alert className="bg-indigo-50 dark:bg-indigo-950/30 border-indigo-200 dark:border-indigo-800">
            <Info className="h-4 w-4" />
            <AlertDescription className="text-sm">
              <strong>Usage Tracking:</strong> The console tracks your tool and token usage. Monitor your current usage on the License page to ensure you stay within your plan limits.
            </AlertDescription>
          </Alert>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <Card className="border-slate-200 dark:border-slate-700">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base">API Usage Trends</CardTitle>
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" className="text-xs">
                      <Calendar className="h-3 w-3 mr-1" />
                      Last 30 days
                    </Button>
                    <Button size="sm" variant="ghost" className="text-xs">
                      <Download className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="bg-slate-100 dark:bg-slate-800 rounded-lg p-8 text-center">
                  <p className="text-muted-foreground mb-2">[API Usage Chart]</p>
                  <p className="text-sm text-muted-foreground">
                    Line chart showing daily API call volume and trends over time
                  </p>
                </div>
              </CardContent>
            </Card>
            
            <Card className="border-slate-200 dark:border-slate-700">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base">Resource Distribution</CardTitle>
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" className="text-xs">
                      <PieChart className="h-3 w-3 mr-1" />
                      By Service
                    </Button>
                    <Button size="sm" variant="ghost" className="text-xs">
                      <Download className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="bg-slate-100 dark:bg-slate-800 rounded-lg p-8 text-center">
                  <p className="text-muted-foreground mb-2">[Resource Distribution Chart]</p>
                  <p className="text-sm text-muted-foreground">
                    Chart showing resource distribution across services and features
                  </p>
                </div>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>

      <Card className="bg-gradient-to-r from-green-50 dark:from-green-950 to-emerald-50 dark:to-emerald-950 border-green-200 dark:border-green-800">
        <CardHeader>
          <CardTitle className="text-green-800 dark:text-green-300">License & Usage Best Practices</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-3">
              <h4 className="font-semibold text-green-800 dark:text-green-300">Usage Optimization</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <TrendingDown className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Monitor Usage Regularly</p>
                    <p className="text-xs text-muted-foreground">Review usage analytics monthly to identify optimization opportunities</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Bell className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Monitor Usage Limits</p>
                    <p className="text-xs text-muted-foreground">Check the License page regularly to track your usage against plan limits</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Calendar className="h-4 w-4 text-purple-600 dark:text-purple-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Consider Upgrading</p>
                    <p className="text-xs text-muted-foreground">Visit kimbap.io/pricing for plan comparison and upgrade options</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <RefreshCw className="h-4 w-4 text-orange-600 dark:text-orange-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Review Tool Usage</p>
                    <p className="text-xs text-muted-foreground">Regularly audit and remove unused integrations</p>
                  </div>
                </li>
              </ul>
            </div>
            
            <div className="space-y-3">
              <h4 className="font-semibold text-green-800 dark:text-green-300">License Management</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <Receipt className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Activate License Keys</p>
                    <p className="text-xs text-muted-foreground">Import license keys from kimbap.io on the License page</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Download className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">View License History</p>
                    <p className="text-xs text-muted-foreground">Track license activations and renewals on the License page</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <Settings className="h-4 w-4 text-orange-600 dark:text-orange-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Configure Usage Limits</p>
                    <p className="text-xs text-muted-foreground">Set hard limits to prevent unexpected overages</p>
                  </div>
                </li>
              </ul>
            </div>
          </div>

          <Alert className="bg-white dark:bg-slate-800 border-green-200 dark:border-green-800">
            <DollarSign className="h-4 w-4" />
            <AlertDescription className="text-sm">
              <strong>Usage Tip:</strong> Review your license usage on the License page regularly and visit kimbap.io/pricing to explore upgrade options when approaching your plan limits.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    </div>
  )
}