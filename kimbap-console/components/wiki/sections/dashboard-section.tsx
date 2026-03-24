import { BarChart3, Users, Settings, Server, Shield, Activity, Plus, RefreshCw, CheckCircle, Zap, UserPlus } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"

export function DashboardSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-2xl mb-4">
          <BarChart3 className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">Dashboard Management</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Your central hub for managing servers, tools, and access tokens. Get an overview of system status and quick access to key functions.
        </p>
      </div>

      {/* Dashboard Overview */}
      <Card className="border-2 border-blue-200 dark:border-blue-800 bg-gradient-to-br from-blue-50 to-cyan-50 dark:from-blue-950/30 dark:to-cyan-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            Dashboard Features
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border">
              <Server className="h-6 w-6 mx-auto mb-2 text-blue-600 dark:text-blue-400" />
              <p className="font-medium text-sm">Server Status</p>
              <p className="text-xs text-muted-foreground">Monitor health</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border">
              <Settings className="h-6 w-6 mx-auto mb-2 text-green-600 dark:text-green-400" />
              <p className="font-medium text-sm">Tool Management</p>
              <p className="text-xs text-muted-foreground">Add & configure</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border">
              <Users className="h-6 w-6 mx-auto mb-2 text-purple-600 dark:text-purple-400" />
              <p className="font-medium text-sm">Access Tokens</p>
              <p className="text-xs text-muted-foreground">Create & manage</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border">
              <Activity className="h-6 w-6 mx-auto mb-2 text-orange-600 dark:text-orange-400" />
              <p className="font-medium text-sm">Usage Stats</p>
              <p className="text-xs text-muted-foreground">View analytics</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Zap className="h-5 w-5 text-green-600 dark:text-green-400" />
              Quick Actions
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <Button className="w-full justify-start">
              <Plus className="h-4 w-4 mr-2" />
              Add New Tool
            </Button>
            <Button variant="outline" className="w-full justify-start">
              <UserPlus className="h-4 w-4 mr-2" />
              Generate Access Token
            </Button>
            <Button variant="outline" className="w-full justify-start">
              <RefreshCw className="h-4 w-4 mr-2" />
              Refresh Status
            </Button>
          </CardContent>
        </Card>
        
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              System Status
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center justify-between p-2 bg-green-50 dark:bg-green-950/30 rounded">
              <div className="flex items-center gap-2">
                <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
                <span className="text-sm">Server Online</span>
              </div>
              <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Healthy</Badge>
            </div>
            <div className="flex items-center justify-between p-2 bg-blue-50 dark:bg-blue-950/30 rounded">
              <div className="flex items-center gap-2">
                <Settings className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                <span className="text-sm">3 Tools Active</span>
              </div>
              <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Running</Badge>
            </div>
            <div className="flex items-center justify-between p-2 bg-purple-50 dark:bg-purple-950/30 rounded">
              <div className="flex items-center gap-2">
                <Users className="h-4 w-4 text-purple-600 dark:text-purple-400" />
                <span className="text-sm">5 Active Tokens</span>
              </div>
              <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 text-xs">Active</Badge>
            </div>
          </CardContent>
        </Card>
      </div>
      
    </div>
  )
}