import { Crown, Users, Shield, CheckCircle, AlertCircle, Settings, Plus, Edit, Trash2, Eye, Key, Lock, Globe, Server, Monitor, User, Building, ChevronRight, Copy, Zap } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function TenantManagementSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-purple-500 to-indigo-600 rounded-2xl mb-4">
          <Crown className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">Tenant Management</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Organize and manage multiple tenants within your Kimbap.io platform. Create separate environments for different teams, projects, or organizations with complete isolation and customized access controls.
        </p>
      </div>

      {/* Multi-Tenancy Overview */}
      <Card className="border-2 border-purple-200 dark:border-purple-800 bg-gradient-to-br from-purple-50 to-indigo-50 dark:from-purple-950/30 dark:to-indigo-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-purple-800 dark:text-purple-300">
            <Building className="h-6 w-6" />
            Multi-Tenancy Architecture
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-purple-700 dark:text-purple-300">
            Kimbap.io's multi-tenancy system provides complete isolation between different organizational units while maintaining centralized administration. Each tenant operates independently with its own users, tools, and data.
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-purple-200 dark:border-purple-800">
              <div className="w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Building className="h-5 w-5 text-purple-600 dark:text-purple-400" />
              </div>
              <p className="font-medium text-sm mb-1">Complete Isolation</p>
              <p className="text-xs text-muted-foreground">Separate data and access per tenant</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-blue-200 dark:border-blue-800">
              <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Shield className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              </div>
              <p className="font-medium text-sm mb-1">Security Boundaries</p>
              <p className="text-xs text-muted-foreground">Tenant-level security policies</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-green-200 dark:border-green-800">
              <div className="w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Settings className="h-5 w-5 text-green-600 dark:text-green-400" />
              </div>
              <p className="font-medium text-sm mb-1">Custom Configuration</p>
              <p className="text-xs text-muted-foreground">Tenant-specific settings and tools</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-orange-200 dark:border-orange-800">
              <div className="w-10 h-10 bg-orange-100 dark:bg-orange-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Monitor className="h-5 w-5 text-orange-600 dark:text-orange-400" />
              </div>
              <p className="font-medium text-sm mb-1">Centralized Management</p>
              <p className="text-xs text-muted-foreground">Unified administration interface</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Tenant Management Tabs */}
      <Tabs defaultValue="overview" className="w-full">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="overview" className="flex items-center gap-2 text-xs">
            <Eye className="h-4 w-4" />
            Overview
          </TabsTrigger>
          <TabsTrigger value="creation" className="flex items-center gap-2 text-xs">
            <Plus className="h-4 w-4" />
            Create Tenant
          </TabsTrigger>
          <TabsTrigger value="management" className="flex items-center gap-2 text-xs">
            <Settings className="h-4 w-4" />
            Management
          </TabsTrigger>
          <TabsTrigger value="hierarchy" className="flex items-center gap-2 text-xs">
            <Building className="h-4 w-4" />
            Hierarchy
          </TabsTrigger>
        </TabsList>

        {/* Tenant Overview */}
        <TabsContent value="overview" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Building className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                Current Tenants
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-4">
                <h4 className="font-semibold">Active Tenant Organizations</h4>
                <div className="space-y-3">
                  {/* Engineering Tenant */}
                  <div className="flex items-center justify-between p-4 border rounded-lg bg-blue-50 dark:bg-blue-950/30 border-blue-200 dark:border-blue-800">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center">
                        <Building className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <p className="font-medium text-sm">Engineering Team</p>
                          <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Primary</Badge>
                        </div>
                        <p className="text-xs text-muted-foreground">engineering.company.com</p>
                        <p className="text-xs text-blue-600 dark:text-blue-400">24 users • 15 tools • 1,247 requests today</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button size="sm" variant="outline" className="h-7 text-xs">
                        <Eye className="h-3 w-3 mr-1" />
                        View
                      </Button>
                      <Button size="sm" variant="outline" className="h-7 text-xs">
                        <Settings className="h-3 w-3 mr-1" />
                        Manage
                      </Button>
                    </div>
                  </div>

                  {/* Marketing Tenant */}
                  <div className="flex items-center justify-between p-4 border rounded-lg bg-green-50 dark:bg-green-950/30 border-green-200 dark:border-green-800">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-lg flex items-center justify-center">
                        <Building className="h-5 w-5 text-green-600 dark:text-green-400" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <p className="font-medium text-sm">Marketing Department</p>
                          <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Active</Badge>
                        </div>
                        <p className="text-xs text-muted-foreground">marketing.company.com</p>
                        <p className="text-xs text-green-600 dark:text-green-400">12 users • 8 tools • 342 requests today</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button size="sm" variant="outline" className="h-7 text-xs">
                        <Eye className="h-3 w-3 mr-1" />
                        View
                      </Button>
                      <Button size="sm" variant="outline" className="h-7 text-xs">
                        <Settings className="h-3 w-3 mr-1" />
                        Manage
                      </Button>
                    </div>
                  </div>

                  {/* Customer Support Tenant */}
                  <div className="flex items-center justify-between p-4 border rounded-lg bg-orange-50 dark:bg-orange-950/30 border-orange-200 dark:border-orange-800">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-orange-100 dark:bg-orange-900/30 rounded-lg flex items-center justify-center">
                        <Building className="h-5 w-5 text-orange-600 dark:text-orange-400" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <p className="font-medium text-sm">Customer Support</p>
                          <Badge className="bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 text-xs">Active</Badge>
                        </div>
                        <p className="text-xs text-muted-foreground">support.company.com</p>
                        <p className="text-xs text-orange-600 dark:text-orange-400">8 users • 6 tools • 156 requests today</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button size="sm" variant="outline" className="h-7 text-xs">
                        <Eye className="h-3 w-3 mr-1" />
                        View
                      </Button>
                      <Button size="sm" variant="outline" className="h-7 text-xs">
                        <Settings className="h-3 w-3 mr-1" />
                        Manage
                      </Button>
                    </div>
                  </div>

                  {/* Partner Tenant */}
                  <div className="flex items-center justify-between p-4 border rounded-lg bg-gray-50 dark:bg-gray-950/50 border-gray-200 dark:border-gray-800">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-gray-100 dark:bg-gray-900/50 rounded-lg flex items-center justify-center">
                        <Building className="h-5 w-5 text-gray-600 dark:text-gray-400" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <p className="font-medium text-sm">External Partners</p>
                          <Badge variant="outline" className="text-xs">Limited Access</Badge>
                        </div>
                        <p className="text-xs text-muted-foreground">partners.company.com</p>
                        <p className="text-xs text-gray-600 dark:text-gray-400">3 users • 2 tools • 12 requests today</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button size="sm" variant="outline" className="h-7 text-xs">
                        <Eye className="h-3 w-3 mr-1" />
                        View
                      </Button>
                      <Button size="sm" variant="outline" className="h-7 text-xs">
                        <Settings className="h-3 w-3 mr-1" />
                        Manage
                      </Button>
                    </div>
                  </div>
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Card className="text-center">
                  <CardContent className="p-4">
                    <div className="text-2xl font-bold text-purple-600 dark:text-purple-400 mb-1">4</div>
                    <p className="text-xs text-muted-foreground">Active Tenants</p>
                  </CardContent>
                </Card>
                
                <Card className="text-center">
                  <CardContent className="p-4">
                    <div className="text-2xl font-bold text-blue-600 dark:text-blue-400 mb-1">47</div>
                    <p className="text-xs text-muted-foreground">Total Users</p>
                  </CardContent>
                </Card>
                
                <Card className="text-center">
                  <CardContent className="p-4">
                    <div className="text-2xl font-bold text-green-600 dark:text-green-400 mb-1">1,757</div>
                    <p className="text-xs text-muted-foreground">Requests Today</p>
                  </CardContent>
                </Card>
              </div>

              <div className="flex gap-3">
                <Button className="flex-1">
                  <Plus className="h-4 w-4 mr-2" />
                  Create New Tenant
                </Button>
                <Button variant="outline" className="flex-1">
                  <Settings className="h-4 w-4 mr-2" />
                  Tenant Settings
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Tenant Creation */}
        <TabsContent value="creation" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Plus className="h-5 w-5 text-green-600 dark:text-green-400" />
                Create New Tenant
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
                <Plus className="h-4 w-4" />
                <AlertDescription>
                  <strong>New Tenant Setup:</strong> Create a new tenant organization with isolated resources, 
                  custom configuration, and dedicated user management.
                </AlertDescription>
              </Alert>

              <div className="space-y-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Basic Information</h4>
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                    <div className="space-y-3">
                      <div>
                        <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Tenant Name</label>
                        <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                          <input type="text" placeholder="e.g., Sales Team" className="w-full bg-transparent border-none outline-none text-sm" />
                        </div>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Tenant ID</label>
                        <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                          <input type="text" placeholder="e.g., sales-team" className="w-full bg-transparent border-none outline-none text-sm font-mono" />
                        </div>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Description</label>
                        <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                          <textarea placeholder="Brief description of this tenant" rows={3} className="w-full bg-transparent border-none outline-none text-sm resize-none"></textarea>
                        </div>
                      </div>
                    </div>
                    
                    <div className="space-y-3">
                      <div>
                        <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Domain</label>
                        <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                          <input type="text" placeholder="sales.company.com" className="w-full bg-transparent border-none outline-none text-sm font-mono" />
                        </div>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Tenant Type</label>
                        <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                          <select className="w-full bg-transparent border-none outline-none text-sm">
                            <option>Internal Department</option>
                            <option>External Partner</option>
                            <option>Development Environment</option>
                            <option>Customer Organization</option>
                          </select>
                        </div>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Parent Tenant</label>
                        <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                          <select className="w-full bg-transparent border-none outline-none text-sm">
                            <option>None (Root Level)</option>
                            <option>Engineering Team</option>
                            <option>Marketing Department</option>
                          </select>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Access & Security Configuration</h4>
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div className="space-y-3">
                      <h5 className="text-sm font-medium">Security Settings</h5>
                      <div className="space-y-2">
                        <div className="flex items-center gap-2">
                          <input type="checkbox" className="w-4 h-4" defaultChecked />
                          <span className="text-sm">Require MFA for all users</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <input type="checkbox" className="w-4 h-4" defaultChecked />
                          <span className="text-sm">Enable IP whitelisting</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <input type="checkbox" className="w-4 h-4" />
                          <span className="text-sm">Restrict external tool access</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <input type="checkbox" className="w-4 h-4" defaultChecked />
                          <span className="text-sm">Enable audit logging</span>
                        </div>
                      </div>
                    </div>

                    <div className="space-y-3">
                      <h5 className="text-sm font-medium">Resource Limits</h5>
                      <div className="space-y-2">
                        <div className="flex items-center justify-between">
                          <span className="text-sm">Max Users</span>
                          <div className="w-20 p-1 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <input type="number" defaultValue={50} className="w-full bg-transparent border-none outline-none text-sm text-center" />
                          </div>
                        </div>
                        <div className="flex items-center justify-between">
                          <span className="text-sm">Max Tools</span>
                          <div className="w-20 p-1 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <input type="number" defaultValue={20} className="w-full bg-transparent border-none outline-none text-sm text-center" />
                          </div>
                        </div>
                        <div className="flex items-center justify-between">
                          <span className="text-sm">Daily API Limit</span>
                          <div className="w-20 p-1 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <input type="number" defaultValue={10000} className="w-full bg-transparent border-none outline-none text-sm text-center" />
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Initial Configuration</h4>
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                    <div className="space-y-3">
                      <h5 className="text-sm font-medium">Admin User</h5>
                      <div className="space-y-2">
                        <div>
                          <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Admin Email</label>
                          <div className="mt-1 p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <input type="email" placeholder="admin@company.com" className="w-full bg-transparent border-none outline-none text-sm" />
                          </div>
                        </div>
                        <div>
                          <label className="text-xs font-medium text-slate-600 dark:text-slate-400">Admin Name</label>
                          <div className="mt-1 p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <input type="text" placeholder="John Smith" className="w-full bg-transparent border-none outline-none text-sm" />
                          </div>
                        </div>
                      </div>
                    </div>
                    
                    <div className="space-y-3">
                      <h5 className="text-sm font-medium">Default Tools</h5>
                      <div className="space-y-2">
                        <div className="flex items-center gap-2">
                          <input type="checkbox" className="w-4 h-4" defaultChecked />
                          <span className="text-sm">File System Access</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <input type="checkbox" className="w-4 h-4" defaultChecked />
                          <span className="text-sm">Web Search</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <input type="checkbox" className="w-4 h-4" />
                          <span className="text-sm">Database Access</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <input type="checkbox" className="w-4 h-4" />
                          <span className="text-sm">External APIs</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="flex gap-3">
                  <Button className="flex-1">
                    <CheckCircle className="h-4 w-4 mr-2" />
                    Create Tenant
                  </Button>
                  <Button variant="outline" className="flex-1">
                    <Copy className="h-4 w-4 mr-2" />
                    Save as Template
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Tenant Management */}
        <TabsContent value="management" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Settings className="h-5 w-5 text-orange-600 dark:text-orange-400" />
                Tenant Management Operations
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="border-orange-200 bg-orange-50 dark:bg-orange-950/30 dark:border-orange-800">
                <Settings className="h-4 w-4" />
                <AlertDescription>
                  <strong>Management Operations:</strong> Perform administrative tasks on existing tenants including 
                  configuration updates, user management, and resource monitoring.
                </AlertDescription>
              </Alert>

              <div className="space-y-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Common Management Tasks</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-3">
                          <Users className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                          <span className="font-medium text-sm">User Management</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          Add, remove, and manage user accounts within tenants
                        </p>
                        <Button size="sm" variant="outline" className="w-full">
                          <Users className="h-3 w-3 mr-1" />
                          Manage Users
                        </Button>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-3">
                          <Settings className="h-5 w-5 text-green-600 dark:text-green-400" />
                          <span className="font-medium text-sm">Tool Configuration</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          Configure and manage tools available to tenant users
                        </p>
                        <Button size="sm" variant="outline" className="w-full">
                          <Settings className="h-3 w-3 mr-1" />
                          Configure Tools
                        </Button>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-purple-200 bg-purple-50 dark:bg-purple-950/30 dark:border-purple-800">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-3">
                          <Shield className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                          <span className="font-medium text-sm">Security Policies</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          Update security settings and access policies
                        </p>
                        <Button size="sm" variant="outline" className="w-full">
                          <Shield className="h-3 w-3 mr-1" />
                          Security Settings
                        </Button>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-orange-200 bg-orange-50 dark:bg-orange-950/30 dark:border-orange-800">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-3">
                          <Monitor className="h-5 w-5 text-orange-600 dark:text-orange-400" />
                          <span className="font-medium text-sm">Usage Analytics</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          Monitor tenant resource usage and performance
                        </p>
                        <Button size="sm" variant="outline" className="w-full">
                          <Monitor className="h-3 w-3 mr-1" />
                          View Analytics
                        </Button>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-cyan-200 bg-cyan-50 dark:bg-cyan-950/30 dark:border-cyan-800">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-3">
                          <Key className="h-5 w-5 text-cyan-600 dark:text-cyan-400" />
                          <span className="font-medium text-sm">Access Tokens</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          Generate and manage tenant access tokens
                        </p>
                        <Button size="sm" variant="outline" className="w-full">
                          <Key className="h-3 w-3 mr-1" />
                          Manage Tokens
                        </Button>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-red-200 bg-red-50 dark:bg-red-950/30 dark:border-red-800">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-3">
                          <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
                          <span className="font-medium text-sm">Tenant Suspension</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          Temporarily suspend or deactivate tenants
                        </p>
                        <Button size="sm" variant="outline" className="w-full">
                          <AlertCircle className="h-3 w-3 mr-1" />
                          Manage Status
                        </Button>
                      </CardContent>
                    </Card>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Bulk Operations</h4>
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div className="space-y-3">
                      <h5 className="text-sm font-medium">Multi-Tenant Operations</h5>
                      <div className="space-y-2">
                        <Button variant="outline" className="w-full justify-start">
                          <CheckCircle className="h-4 w-4 mr-2" />
                          Update Security Policies Across Tenants
                        </Button>
                        <Button variant="outline" className="w-full justify-start">
                          <Settings className="h-4 w-4 mr-2" />
                          Deploy Tool Updates to All Tenants
                        </Button>
                        <Button variant="outline" className="w-full justify-start">
                          <Monitor className="h-4 w-4 mr-2" />
                          Generate Cross-Tenant Usage Report
                        </Button>
                        <Button variant="outline" className="w-full justify-start">
                          <Key className="h-4 w-4 mr-2" />
                          Rotate All Tenant API Keys
                        </Button>
                      </div>
                    </div>
                    
                    <div className="space-y-3">
                      <h5 className="text-sm font-medium">Migration & Backup</h5>
                      <div className="space-y-2">
                        <Button variant="outline" className="w-full justify-start">
                          <Copy className="h-4 w-4 mr-2" />
                          Export Tenant Configuration
                        </Button>
                        <Button variant="outline" className="w-full justify-start">
                          <Server className="h-4 w-4 mr-2" />
                          Backup Tenant Data
                        </Button>
                        <Button variant="outline" className="w-full justify-start">
                          <Globe className="h-4 w-4 mr-2" />
                          Migrate Tenant to New Server
                        </Button>
                        <Button variant="outline" className="w-full justify-start">
                          <Trash2 className="h-4 w-4 mr-2" />
                          Archive Inactive Tenants
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Tenant Hierarchy */}
        <TabsContent value="hierarchy" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Building className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
                Tenant Hierarchy & Relationships
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="border-indigo-200 bg-indigo-50 dark:bg-indigo-950/30 dark:border-indigo-800">
                <Building className="h-4 w-4" />
                <AlertDescription>
                  <strong>Hierarchical Structure:</strong> Organize tenants in a hierarchical structure to reflect your 
                  organizational structure and enable inheritance of policies and configurations.
                </AlertDescription>
              </Alert>

              <div className="space-y-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Current Tenant Hierarchy</h4>
                  <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-6 border dark:border-slate-700">
                    {/* Root Level */}
                    <div className="space-y-4">
                      <div className="flex items-center gap-2 p-3 bg-white dark:bg-slate-800 border-2 border-blue-300 dark:border-blue-700 rounded-lg">
                        <Crown className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                        <span className="font-medium">Company Root</span>
                        <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Root</Badge>
                      </div>
                      
                      {/* Level 1 - Departments */}
                      <div className="ml-6 space-y-3">
                        <div className="flex items-center gap-2 p-3 bg-white dark:bg-slate-800 border border-purple-200 dark:border-purple-800 rounded-lg">
                          <Building className="h-4 w-4 text-purple-600 dark:text-purple-400" />
                          <span className="font-medium text-sm">Engineering Division</span>
                          <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 text-xs">Department</Badge>
                        </div>
                        
                        {/* Level 2 - Teams */}
                        <div className="ml-6 space-y-2">
                          <div className="flex items-center gap-2 p-2 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded">
                            <Users className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                            <span className="text-sm">Frontend Team</span>
                            <Badge variant="outline" className="text-xs">Team</Badge>
                          </div>
                          <div className="flex items-center gap-2 p-2 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded">
                            <Users className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                            <span className="text-sm">Backend Team</span>
                            <Badge variant="outline" className="text-xs">Team</Badge>
                          </div>
                          <div className="flex items-center gap-2 p-2 bg-white dark:bg-slate-800 border border-blue-200 dark:border-blue-800 rounded">
                            <Users className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                            <span className="text-sm">DevOps Team</span>
                            <Badge variant="outline" className="text-xs">Team</Badge>
                          </div>
                        </div>
                        
                        <div className="flex items-center gap-2 p-3 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded-lg">
                          <Building className="h-4 w-4 text-green-600 dark:text-green-400" />
                          <span className="font-medium text-sm">Sales & Marketing</span>
                          <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Department</Badge>
                        </div>
                        
                        <div className="ml-6 space-y-2">
                          <div className="flex items-center gap-2 p-2 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded">
                            <Users className="h-4 w-4 text-green-600 dark:text-green-400" />
                            <span className="text-sm">Sales Team</span>
                            <Badge variant="outline" className="text-xs">Team</Badge>
                          </div>
                          <div className="flex items-center gap-2 p-2 bg-white dark:bg-slate-800 border border-green-200 dark:border-green-800 rounded">
                            <Users className="h-4 w-4 text-green-600 dark:text-green-400" />
                            <span className="text-sm">Marketing Team</span>
                            <Badge variant="outline" className="text-xs">Team</Badge>
                          </div>
                        </div>
                        
                        <div className="flex items-center gap-2 p-3 bg-white dark:bg-slate-800 border border-orange-200 dark:border-orange-800 rounded-lg">
                          <Building className="h-4 w-4 text-orange-600 dark:text-orange-400" />
                          <span className="font-medium text-sm">External Partners</span>
                          <Badge className="bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 text-xs">External</Badge>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Hierarchy Benefits</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="space-y-3">
                      <h5 className="text-sm font-medium">Policy Inheritance</h5>
                      <ul className="space-y-2">
                        <li className="flex items-start gap-2">
                          <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5" />
                          <div>
                            <p className="text-sm font-medium">Security Policies</p>
                            <p className="text-xs text-muted-foreground">Child tenants inherit parent security settings</p>
                          </div>
                        </li>
                        <li className="flex items-start gap-2">
                          <CheckCircle className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5" />
                          <div>
                            <p className="text-sm font-medium">Tool Access</p>
                            <p className="text-xs text-muted-foreground">Share tool configurations down the hierarchy</p>
                          </div>
                        </li>
                        <li className="flex items-start gap-2">
                          <CheckCircle className="h-4 w-4 text-purple-600 dark:text-purple-400 mt-0.5" />
                          <div>
                            <p className="text-sm font-medium">Resource Limits</p>
                            <p className="text-xs text-muted-foreground">Parent limits apply to all child tenants</p>
                          </div>
                        </li>
                      </ul>
                    </div>
                    
                    <div className="space-y-3">
                      <h5 className="text-sm font-medium">Management Efficiency</h5>
                      <ul className="space-y-2">
                        <li className="flex items-start gap-2">
                          <CheckCircle className="h-4 w-4 text-orange-600 dark:text-orange-400 mt-0.5" />
                          <div>
                            <p className="text-sm font-medium">Centralized Control</p>
                            <p className="text-xs text-muted-foreground">Manage multiple tenants from parent level</p>
                          </div>
                        </li>
                        <li className="flex items-start gap-2">
                          <CheckCircle className="h-4 w-4 text-cyan-600 dark:text-cyan-400 mt-0.5" />
                          <div>
                            <p className="text-sm font-medium">Delegated Administration</p>
                            <p className="text-xs text-muted-foreground">Department admins manage their sub-tenants</p>
                          </div>
                        </li>
                        <li className="flex items-start gap-2">
                          <CheckCircle className="h-4 w-4 text-indigo-600 dark:text-indigo-400 mt-0.5" />
                          <div>
                            <p className="text-sm font-medium">Reporting Aggregation</p>
                            <p className="text-xs text-muted-foreground">Roll up usage and metrics across hierarchy</p>
                          </div>
                        </li>
                      </ul>
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

    </div>
  )
}