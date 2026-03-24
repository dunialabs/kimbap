import { Shield, Crown, UserCheck, User, Key, Lock, Eye, Settings, CheckCircle, AlertCircle, Users, Building, Server, Monitor, Globe, Edit, Plus, Trash2, ChevronRight, Copy, Zap } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function UserRolesSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-emerald-500 to-teal-600 rounded-2xl mb-4">
          <UserCheck className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">User Roles & Permissions</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Comprehensive role-based access control system for managing user permissions across tenants. 
          Define roles, assign permissions, and control access to tools and administrative functions.
        </p>
      </div>

      {/* RBAC Overview */}
      <Card className="border-2 border-emerald-200 dark:border-emerald-800 bg-gradient-to-br from-emerald-50 to-teal-50 dark:from-emerald-950/30 dark:to-teal-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-emerald-800 dark:text-emerald-300">
            <Shield className="h-6 w-6" />
            Role-Based Access Control (RBAC)
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-emerald-700 dark:text-emerald-300">
            Kimbap.io implements a flexible RBAC system that allows you to define granular permissions for different user roles within each tenant. Control access to tools, administrative functions, and data based on organizational hierarchy and responsibilities.
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-emerald-200 dark:border-emerald-800">
              <div className="w-10 h-10 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Crown className="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
              </div>
              <p className="font-medium text-sm mb-1">Hierarchical Roles</p>
              <p className="text-xs text-muted-foreground">Owner → Admin → Member structure</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-blue-200 dark:border-blue-800">
              <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Lock className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              </div>
              <p className="font-medium text-sm mb-1">Granular Permissions</p>
              <p className="text-xs text-muted-foreground">Fine-grained access controls</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-purple-200 dark:border-purple-800">
              <div className="w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Building className="h-5 w-5 text-purple-600 dark:text-purple-400" />
              </div>
              <p className="font-medium text-sm mb-1">Tenant Isolation</p>
              <p className="text-xs text-muted-foreground">Roles scoped to specific tenants</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-orange-200 dark:border-orange-800">
              <div className="w-10 h-10 bg-orange-100 dark:bg-orange-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <UserCheck className="h-5 w-5 text-orange-600 dark:text-orange-400" />
              </div>
              <p className="font-medium text-sm mb-1">Flexible Assignment</p>
              <p className="text-xs text-muted-foreground">Multiple roles per user</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Roles & Permissions Tabs */}
      <Tabs defaultValue="roles" className="w-full">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="roles" className="flex items-center gap-2 text-xs">
            <Crown className="h-4 w-4" />
            Role Types
          </TabsTrigger>
          <TabsTrigger value="permissions" className="flex items-center gap-2 text-xs">
            <Lock className="h-4 w-4" />
            Permissions
          </TabsTrigger>
          <TabsTrigger value="assignment" className="flex items-center gap-2 text-xs">
            <UserCheck className="h-4 w-4" />
            Assignment
          </TabsTrigger>
          <TabsTrigger value="management" className="flex items-center gap-2 text-xs">
            <Settings className="h-4 w-4" />
            Management
          </TabsTrigger>
        </TabsList>

        {/* Role Types */}
        <TabsContent value="roles" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Crown className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                Standard Role Hierarchy
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="border-purple-200 bg-purple-50 dark:bg-purple-950/30 dark:border-purple-800">
                <Crown className="h-4 w-4" />
                <AlertDescription>
                  <strong>Role Hierarchy:</strong> Kimbap.io uses a three-tier role system with inheritable permissions. 
                  Higher-level roles inherit all permissions from lower-level roles.
                </AlertDescription>
              </Alert>

              <div className="space-y-6">
                {/* Owner Role */}
                <Card className="border-2 border-red-200 dark:border-red-800 bg-gradient-to-br from-red-50 to-pink-50 dark:from-red-950/30 dark:to-pink-950/30">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-lg flex items-center gap-2 text-red-800 dark:text-red-300">
                      <Crown className="h-5 w-5" />
                      Owner
                      <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Highest Authority</Badge>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <p className="text-sm text-red-700 dark:text-red-300">
                      Ultimate authority within a tenant. Can perform all administrative tasks and has full control over tenant resources and users.
                    </p>
                    
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <h5 className="text-sm font-medium text-red-800 dark:text-red-300">Core Capabilities</h5>
                        <ul className="space-y-1">
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-red-600 dark:text-red-400" />
                            <span>Full tenant administration</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-red-600 dark:text-red-400" />
                            <span>User and role management</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-red-600 dark:text-red-400" />
                            <span>Security policy configuration</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-red-600 dark:text-red-400" />
                            <span>Resource limit management</span>
                          </li>
                        </ul>
                      </div>
                      
                      <div className="space-y-2">
                        <h5 className="text-sm font-medium text-red-800 dark:text-red-300">Exclusive Powers</h5>
                        <ul className="space-y-1">
                          <li className="flex items-center gap-2 text-sm">
                            <Key className="h-3 w-3 text-red-600 dark:text-red-400" />
                            <span>Owner token management</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <Key className="h-3 w-3 text-red-600 dark:text-red-400" />
                            <span>Tenant deletion</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <Key className="h-3 w-3 text-red-600 dark:text-red-400" />
                            <span>Billing and subscription</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <Key className="h-3 w-3 text-red-600 dark:text-red-400" />
                            <span>Transfer ownership</span>
                          </li>
                        </ul>
                      </div>
                    </div>
                    
                    <Alert className="bg-red-100 dark:bg-red-900/30 border-red-300 dark:border-red-700">
                      <AlertCircle className="h-4 w-4" />
                      <AlertDescription className="text-xs">
                        <strong>Owner Limitations:</strong> Only one Owner per tenant. Owner cannot be demoted by other users. Owner role cannot be deleted.
                      </AlertDescription>
                    </Alert>
                  </CardContent>
                </Card>

                {/* Admin Role */}
                <Card className="border-2 border-blue-200 dark:border-blue-800 bg-gradient-to-br from-blue-50 to-indigo-50 dark:from-blue-950/30 dark:to-indigo-950/30">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-lg flex items-center gap-2 text-blue-800 dark:text-blue-300">
                      <Shield className="h-5 w-5" />
                      Administrator
                      <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Management Authority</Badge>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <p className="text-sm text-blue-700 dark:text-blue-300">
                      Responsible for day-to-day tenant management and user administration. Can manage most tenant resources but requires owner approval for critical changes.
                    </p>
                    
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <h5 className="text-sm font-medium text-blue-800 dark:text-blue-300">Administrative Rights</h5>
                        <ul className="space-y-1">
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>Tool configuration and management</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>User invitation and management</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>Member role assignment</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>Usage monitoring and analytics</span>
                          </li>
                        </ul>
                      </div>
                      
                      <div className="space-y-2">
                        <h5 className="text-sm font-medium text-blue-800 dark:text-blue-300">Operational Access</h5>
                        <ul className="space-y-1">
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>All tool access and configuration</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>Generate user access tokens</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>View all tenant analytics</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>Manage security settings</span>
                          </li>
                        </ul>
                      </div>
                    </div>
                    
                    <Alert className="bg-blue-100 dark:bg-blue-900/30 border-blue-300 dark:border-blue-700">
                      <AlertCircle className="h-4 w-4" />
                      <AlertDescription className="text-xs">
                        <strong>Admin Restrictions:</strong> Cannot modify owner settings, delete tenant, or change billing information. Cannot promote users to admin without owner approval.
                      </AlertDescription>
                    </Alert>
                  </CardContent>
                </Card>

                {/* Member Role */}
                <Card className="border-2 border-green-200 dark:border-green-800 bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950/30 dark:to-emerald-950/30">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-lg flex items-center gap-2 text-green-800 dark:text-green-300">
                      <User className="h-5 w-5" />
                      Member
                      <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Standard User</Badge>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <p className="text-sm text-green-700 dark:text-green-300">
                      Standard user role for accessing and using tenant tools. Can use all configured tools within their permission scope but cannot modify tenant settings.
                    </p>
                    
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <h5 className="text-sm font-medium text-green-800 dark:text-green-300">Tool Access</h5>
                        <ul className="space-y-1">
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>Use all authorized tools</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>View personal usage statistics</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>Manage personal settings</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>Generate personal access tokens</span>
                          </li>
                        </ul>
                      </div>
                      
                      <div className="space-y-2">
                        <h5 className="text-sm font-medium text-green-800 dark:text-green-300">Collaboration Features</h5>
                        <ul className="space-y-1">
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>Share tool outputs with team</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>Access shared resources</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>View active token list</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>Request new tool access</span>
                          </li>
                        </ul>
                      </div>
                    </div>
                    
                    <Alert className="bg-green-100 dark:bg-green-900/30 border-green-300 dark:border-green-700">
                      <AlertCircle className="h-4 w-4" />
                      <AlertDescription className="text-xs">
                        <strong>Member Scope:</strong> Cannot modify tenant settings, invite users, or access administrative functions. Tool access is controlled by administrators.
                      </AlertDescription>
                    </Alert>
                  </CardContent>
                </Card>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Permissions */}
        <TabsContent value="permissions" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Lock className="h-5 w-5 text-orange-600 dark:text-orange-400" />
                Permission Categories & Controls
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="border-orange-200 bg-orange-50 dark:bg-orange-950/30 dark:border-orange-800">
                <Lock className="h-4 w-4" />
                <AlertDescription>
                  <strong>Granular Permissions:</strong> Permissions are organized into categories for easy management. 
                  Each category contains specific actions that can be granted or denied to roles.
                </AlertDescription>
              </Alert>

              <div className="space-y-6">
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                  {/* Tenant Management Permissions */}
                  <Card className="border-purple-200 bg-purple-50 dark:bg-purple-950/30 dark:border-purple-800">
                    <CardHeader className="pb-3">
                      <CardTitle className="text-base flex items-center gap-2 text-purple-800 dark:text-purple-300">
                        <Building className="h-4 w-4" />
                        Tenant Management
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <div className="space-y-2">
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">Create Sub-Tenants</span>
                          <div className="flex gap-1">
                            <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Owner</Badge>
                          </div>
                        </div>
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">Modify Tenant Settings</span>
                          <div className="flex gap-1">
                            <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Owner</Badge>
                            <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Admin</Badge>
                          </div>
                        </div>
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">View Tenant Info</span>
                          <div className="flex gap-1">
                            <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">All</Badge>
                          </div>
                        </div>
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">Delete Tenant</span>
                          <div className="flex gap-1">
                            <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Owner</Badge>
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>

                  {/* User Management Permissions */}
                  <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
                    <CardHeader className="pb-3">
                      <CardTitle className="text-base flex items-center gap-2 text-blue-800 dark:text-blue-300">
                        <Users className="h-4 w-4" />
                        User Management
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <div className="space-y-2">
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">Invite Users</span>
                          <div className="flex gap-1">
                            <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Owner</Badge>
                            <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Admin</Badge>
                          </div>
                        </div>
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">Assign Roles</span>
                          <div className="flex gap-1">
                            <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Owner</Badge>
                            <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Admin*</Badge>
                          </div>
                        </div>
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">Remove Users</span>
                          <div className="flex gap-1">
                            <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Owner</Badge>
                            <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Admin</Badge>
                          </div>
                        </div>
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">View User List</span>
                          <div className="flex gap-1">
                            <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">All</Badge>
                          </div>
                        </div>
                      </div>
                      <p className="text-xs text-blue-600 dark:text-blue-400">* Admin can only assign Member roles</p>
                    </CardContent>
                  </Card>

                  {/* Tool Management Permissions */}
                  <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
                    <CardHeader className="pb-3">
                      <CardTitle className="text-base flex items-center gap-2 text-green-800 dark:text-green-300">
                        <Settings className="h-4 w-4" />
                        Tool Management
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <div className="space-y-2">
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">Install/Uninstall Tools</span>
                          <div className="flex gap-1">
                            <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Owner</Badge>
                            <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Admin</Badge>
                          </div>
                        </div>
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">Configure Tool Settings</span>
                          <div className="flex gap-1">
                            <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Owner</Badge>
                            <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Admin</Badge>
                          </div>
                        </div>
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">Use Tools</span>
                          <div className="flex gap-1">
                            <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">All</Badge>
                          </div>
                        </div>
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">Manage Tool Permissions</span>
                          <div className="flex gap-1">
                            <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Owner</Badge>
                            <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Admin</Badge>
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>

                  {/* Security & Monitoring Permissions */}
                  <Card className="border-orange-200 bg-orange-50 dark:bg-orange-950/30 dark:border-orange-800">
                    <CardHeader className="pb-3">
                      <CardTitle className="text-base flex items-center gap-2 text-orange-800 dark:text-orange-300">
                        <Shield className="h-4 w-4" />
                        Security & Monitoring
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <div className="space-y-2">
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">Modify Security Policies</span>
                          <div className="flex gap-1">
                            <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Owner</Badge>
                          </div>
                        </div>
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">View Audit Logs</span>
                          <div className="flex gap-1">
                            <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Owner</Badge>
                            <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Admin</Badge>
                          </div>
                        </div>
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">View Usage Analytics</span>
                          <div className="flex gap-1">
                            <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Owner</Badge>
                            <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Admin</Badge>
                          </div>
                        </div>
                        <div className="flex items-center justify-between p-2 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                          <span className="text-sm">Generate API Tokens</span>
                          <div className="flex gap-1">
                            <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">All</Badge>
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Custom Permission Templates</h4>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <Card className="border-indigo-200 bg-indigo-50 dark:bg-indigo-950/30 dark:border-indigo-800">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-2">
                          <Eye className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
                          <span className="font-medium text-sm">Read-Only Admin</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          View all tenant data and analytics without modification rights
                        </p>
                        <Button size="sm" variant="outline" className="w-full text-xs">
                          Create Template
                        </Button>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-cyan-200 bg-cyan-50 dark:bg-cyan-950/30 dark:border-cyan-800">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-2">
                          <Settings className="h-4 w-4 text-cyan-600 dark:text-cyan-400" />
                          <span className="font-medium text-sm">Tool Manager</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          Manage tools and configurations without user management
                        </p>
                        <Button size="sm" variant="outline" className="w-full text-xs">
                          Create Template
                        </Button>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-pink-200 dark:border-pink-800 bg-pink-50 dark:bg-pink-950/50">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-2">
                          <Users className="h-4 w-4 text-pink-600 dark:text-pink-400" />
                          <span className="font-medium text-sm">Team Lead</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          Manage access tokens and their tool permissions
                        </p>
                        <Button size="sm" variant="outline" className="w-full text-xs">
                          Create Template
                        </Button>
                      </CardContent>
                    </Card>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Role Assignment */}
        <TabsContent value="assignment" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <UserCheck className="h-5 w-5 text-green-600 dark:text-green-400" />
                Role Assignment & Management
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
                <UserCheck className="h-4 w-4" />
                <AlertDescription>
                  <strong>Role Assignment:</strong> Users can have different roles across multiple tenants. 
                  Role assignments are tenant-specific and can be modified by authorized administrators.
                </AlertDescription>
              </Alert>

              <div className="space-y-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Current Role Assignments</h4>
                  <div className="space-y-3">
                    <div className="flex items-center justify-between p-4 border rounded-lg">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center">
                          <User className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                        </div>
                        <div>
                          <p className="font-medium text-sm">john.smith@company.com</p>
                          <p className="text-xs text-muted-foreground">Engineering Team</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge className="bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300 text-xs">Owner</Badge>
                        <Button size="sm" variant="outline" className="h-7 text-xs">
                          <Edit className="h-3 w-3 mr-1" />
                          Edit
                        </Button>
                      </div>
                    </div>
                    
                    <div className="flex items-center justify-between p-4 border rounded-lg">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center">
                          <User className="h-4 w-4 text-green-600 dark:text-green-400" />
                        </div>
                        <div>
                          <p className="font-medium text-sm">sarah.johnson@company.com</p>
                          <p className="text-xs text-muted-foreground">Engineering Team, Marketing Department</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Admin</Badge>
                        <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Member</Badge>
                        <Button size="sm" variant="outline" className="h-7 text-xs">
                          <Edit className="h-3 w-3 mr-1" />
                          Edit
                        </Button>
                      </div>
                    </div>
                    
                    <div className="flex items-center justify-between p-4 border rounded-lg">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 bg-purple-100 dark:bg-purple-900/30 rounded-full flex items-center justify-center">
                          <User className="h-4 w-4 text-purple-600 dark:text-purple-400" />
                        </div>
                        <div>
                          <p className="font-medium text-sm">mike.chen@company.com</p>
                          <p className="text-xs text-muted-foreground">Engineering Team</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Member</Badge>
                        <Button size="sm" variant="outline" className="h-7 text-xs">
                          <Edit className="h-3 w-3 mr-1" />
                          Edit
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Assign New Role</h4>
                  <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-4 border dark:border-slate-700">
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                      <div className="space-y-3">
                        <div>
                          <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">User Email</label>
                          <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <input type="email" placeholder="user@company.com" className="w-full bg-transparent border-none outline-none text-sm" />
                          </div>
                        </div>
                        <div>
                          <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Tenant</label>
                          <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <select className="w-full bg-transparent border-none outline-none text-sm">
                              <option>Engineering Team</option>
                              <option>Marketing Department</option>
                              <option>Customer Support</option>
                            </select>
                          </div>
                        </div>
                        <div>
                          <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Role</label>
                          <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <select className="w-full bg-transparent border-none outline-none text-sm">
                              <option>Member</option>
                              <option>Administrator</option>
                              <option>Custom: Read-Only Admin</option>
                              <option>Custom: Tool Manager</option>
                            </select>
                          </div>
                        </div>
                      </div>
                      
                      <div className="space-y-3">
                        <div>
                          <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Assignment Reason</label>
                          <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <textarea rows={3} placeholder="Brief reason for this role assignment" className="w-full bg-transparent border-none outline-none text-sm resize-none"></textarea>
                          </div>
                        </div>
                        <div>
                          <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Notification Settings</label>
                          <div className="space-y-2">
                            <div className="flex items-center gap-2">
                              <input type="checkbox" className="w-4 h-4" defaultChecked />
                              <span className="text-sm">Send welcome email to user</span>
                            </div>
                            <div className="flex items-center gap-2">
                              <input type="checkbox" className="w-4 h-4" defaultChecked />
                              <span className="text-sm">Notify tenant administrators</span>
                            </div>
                            <div className="flex items-center gap-2">
                              <input type="checkbox" className="w-4 h-4" />
                              <span className="text-sm">Require user confirmation</span>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                    
                    <div className="flex gap-2 mt-4">
                      <Button size="sm" className="flex-1">
                        <UserCheck className="h-4 w-4 mr-2" />
                        Assign Role
                      </Button>
                      <Button size="sm" variant="outline" className="flex-1">
                        <Plus className="h-4 w-4 mr-2" />
                        Invite New User
                      </Button>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Bulk Role Operations</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <Button variant="outline" className="justify-start">
                      <Users className="h-4 w-4 mr-2" />
                      Import Users from CSV
                    </Button>
                    <Button variant="outline" className="justify-start">
                      <Copy className="h-4 w-4 mr-2" />
                      Export Role Assignments
                    </Button>
                    <Button variant="outline" className="justify-start">
                      <UserCheck className="h-4 w-4 mr-2" />
                      Bulk Role Update
                    </Button>
                    <Button variant="outline" className="justify-start">
                      <Trash2 className="h-4 w-4 mr-2" />
                      Remove Inactive Users
                    </Button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Role Management */}
        <TabsContent value="management" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Settings className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
                Advanced Role Management
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="border-indigo-200 bg-indigo-50 dark:bg-indigo-950/30 dark:border-indigo-800">
                <Settings className="h-4 w-4" />
                <AlertDescription>
                  <strong>Role Management:</strong> Advanced features for creating custom roles, managing role inheritance, 
                  and implementing organization-specific access patterns.
                </AlertDescription>
              </Alert>

              <div className="space-y-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Custom Role Creation</h4>
                  <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-4 border dark:border-slate-700">
                    <div className="space-y-4">
                      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                        <div>
                          <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Role Name</label>
                          <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <input type="text" placeholder="e.g., DevOps Lead" className="w-full bg-transparent border-none outline-none text-sm" />
                          </div>
                        </div>
                        <div>
                          <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Base Role</label>
                          <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <select className="w-full bg-transparent border-none outline-none text-sm">
                              <option>Member (inherit basic permissions)</option>
                              <option>Administrator (inherit admin permissions)</option>
                              <option>Custom Template</option>
                            </select>
                          </div>
                        </div>
                      </div>
                      
                      <div>
                        <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2 block">Additional Permissions</label>
                        <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                          <div className="flex items-center gap-2">
                            <input type="checkbox" className="w-4 h-4" />
                            <span className="text-sm">Server Management</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <input type="checkbox" className="w-4 h-4" />
                            <span className="text-sm">Advanced Analytics</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <input type="checkbox" className="w-4 h-4" />
                            <span className="text-sm">Billing Access</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <input type="checkbox" className="w-4 h-4" />
                            <span className="text-sm">API Management</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <input type="checkbox" className="w-4 h-4" />
                            <span className="text-sm">Audit Logs</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <input type="checkbox" className="w-4 h-4" />
                            <span className="text-sm">Tool Development</span>
                          </div>
                        </div>
                      </div>
                      
                      <Button size="sm">
                        <Plus className="h-4 w-4 mr-2" />
                        Create Custom Role
                      </Button>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Role Analytics & Insights</h4>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <Card className="text-center">
                      <CardContent className="p-4">
                        <div className="text-2xl font-bold text-purple-600 dark:text-purple-400 mb-1">47</div>
                        <p className="text-xs text-muted-foreground">Total Users</p>
                        <div className="text-xs text-muted-foreground mt-1">
                          <span className="text-red-600 dark:text-red-400">1 Owner</span> • 
                          <span className="text-blue-600 dark:text-blue-400"> 6 Admins</span> • 
                          <span className="text-green-600 dark:text-green-400"> 40 Members</span>
                        </div>
                      </CardContent>
                    </Card>
                    
                    <Card className="text-center">
                      <CardContent className="p-4">
                        <div className="text-2xl font-bold text-indigo-600 dark:text-indigo-400 mb-1">8</div>
                        <p className="text-xs text-muted-foreground">Custom Roles</p>
                        <div className="text-xs text-muted-foreground mt-1">
                          Active across 4 tenants
                        </div>
                      </CardContent>
                    </Card>
                    
                    <Card className="text-center">
                      <CardContent className="p-4">
                        <div className="text-2xl font-bold text-orange-600 dark:text-orange-400 mb-1">12</div>
                        <p className="text-xs text-muted-foreground">Pending Invitations</p>
                        <div className="text-xs text-muted-foreground mt-1">
                          Awaiting user acceptance
                        </div>
                      </CardContent>
                    </Card>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Role Compliance & Security</h4>
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                    <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-3">
                          <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400" />
                          <span className="font-medium text-sm">Role Audit</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          Review role assignments and identify potential security risks
                        </p>
                        <Button size="sm" variant="outline" className="w-full">
                          <Eye className="h-3 w-3 mr-1" />
                          Run Audit
                        </Button>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-orange-200 bg-orange-50 dark:bg-orange-950/30 dark:border-orange-800">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-3">
                          <AlertCircle className="h-5 w-5 text-orange-600 dark:text-orange-400" />
                          <span className="font-medium text-sm">Access Review</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          Periodic review of user access rights and role appropriateness
                        </p>
                        <Button size="sm" variant="outline" className="w-full">
                          <Settings className="h-3 w-3 mr-1" />
                          Schedule Review
                        </Button>
                      </CardContent>
                    </Card>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Best Practices */}
      <Card className="bg-gradient-to-r from-emerald-50 dark:from-emerald-950 to-teal-50 dark:to-teal-950 border-emerald-200 dark:border-emerald-800">
        <CardHeader>
          <CardTitle className="text-emerald-800 dark:text-emerald-300">Role Management Best Practices</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-3">
              <h4 className="font-semibold">Security Guidelines</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-emerald-600 dark:text-emerald-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Principle of Least Privilege</p>
                    <p className="text-xs text-muted-foreground">Grant minimum permissions necessary for job function</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-emerald-600 dark:text-emerald-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Regular Access Reviews</p>
                    <p className="text-xs text-muted-foreground">Quarterly review of role assignments and permissions</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-emerald-600 dark:text-emerald-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Role Segregation</p>
                    <p className="text-xs text-muted-foreground">Separate administrative and operational roles</p>
                  </div>
                </li>
              </ul>
            </div>
            
            <div className="space-y-3">
              <h4 className="font-semibold">Operational Efficiency</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-emerald-600 dark:text-emerald-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Role Templates</p>
                    <p className="text-xs text-muted-foreground">Create standard roles for common job functions</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-emerald-600 dark:text-emerald-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Automated Provisioning</p>
                    <p className="text-xs text-muted-foreground">Set up workflows for role assignment</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-emerald-600 dark:text-emerald-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Documentation</p>
                    <p className="text-xs text-muted-foreground">Maintain clear role definitions and responsibilities</p>
                  </div>
                </li>
              </ul>
            </div>
          </div>
          
          <div className="flex flex-col sm:flex-row gap-3 mt-6">
            <Button className="flex-1">
              <UserCheck className="h-4 w-4 mr-2" />
              Manage User Roles
            </Button>
            <Button variant="outline" className="flex-1">
              <Building className="h-4 w-4 mr-2" />
              Tenant Management
            </Button>
            <Button variant="outline" className="flex-1">
              <User className="h-4 w-4 mr-2" />
              Agent Assignment
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}