import { User, Users, Server, Bot, Shield, CheckCircle, AlertCircle, Settings, Plus, Edit, Trash2, Eye, Key, Lock, Globe, Monitor, Crown, ChevronRight, Copy, Zap, UserCheck, Building } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function AgentAssignmentSection() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-cyan-500 to-blue-600 rounded-2xl mb-4">
          <User className="h-8 w-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold mb-2">Agent Assignment</h1>
        <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
          Manage the assignment of AI agents to users across different tenants. Control which users can access specific agents, configure agent pools, and optimize resource allocation for your organization.
        </p>
      </div>

      {/* Agent Assignment Overview */}
      <Card className="border-2 border-cyan-200 dark:border-cyan-800 bg-gradient-to-br from-cyan-50 to-blue-50 dark:from-cyan-950/30 dark:to-blue-950/30">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-cyan-800 dark:text-cyan-300">
            <Bot className="h-6 w-6" />
            Agent Assignment Model
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-cyan-700 dark:text-cyan-300">
            Kimbap.io supports flexible agent assignment strategies to optimize AI resource utilization. Users can be assigned dedicated agents, share agent pools, or have dynamic access based on availability and workload.
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-cyan-200 dark:border-cyan-800">
              <div className="w-10 h-10 bg-cyan-100 dark:bg-cyan-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <User className="h-5 w-5 text-cyan-600 dark:text-cyan-400" />
              </div>
              <p className="font-medium text-sm mb-1">Personal Agents</p>
              <p className="text-xs text-muted-foreground">Dedicated AI assistants per user</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-blue-200 dark:border-blue-800">
              <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Users className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              </div>
              <p className="font-medium text-sm mb-1">Shared Pools</p>
              <p className="text-xs text-muted-foreground">Team-based agent sharing</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-green-200 dark:border-green-800">
              <div className="w-10 h-10 bg-green-100 dark:bg-green-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Zap className="h-5 w-5 text-green-600 dark:text-green-400" />
              </div>
              <p className="font-medium text-sm mb-1">Dynamic Allocation</p>
              <p className="text-xs text-muted-foreground">AI-optimized assignment</p>
            </div>
            <div className="text-center p-3 bg-white dark:bg-slate-800 rounded-lg border border-purple-200 dark:border-purple-800">
              <div className="w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center mx-auto mb-2">
                <Shield className="h-5 w-5 text-purple-600 dark:text-purple-400" />
              </div>
              <p className="font-medium text-sm mb-1">Access Control</p>
              <p className="text-xs text-muted-foreground">Permission-based access</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Agent Assignment Tabs */}
      <Tabs defaultValue="overview" className="w-full">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="overview" className="flex items-center gap-2 text-xs">
            <Eye className="h-4 w-4" />
            Current Assignments
          </TabsTrigger>
          <TabsTrigger value="strategies" className="flex items-center gap-2 text-xs">
            <Bot className="h-4 w-4" />
            Assignment Strategies
          </TabsTrigger>
          <TabsTrigger value="pools" className="flex items-center gap-2 text-xs">
            <Users className="h-4 w-4" />
            Agent Pools
          </TabsTrigger>
          <TabsTrigger value="management" className="flex items-center gap-2 text-xs">
            <Settings className="h-4 w-4" />
            Management
          </TabsTrigger>
        </TabsList>

        {/* Current Assignments Overview */}
        <TabsContent value="overview" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <User className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                Current Agent Assignments
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-4">
                <h4 className="font-semibold">User-Agent Assignments by Tenant</h4>
                
                {/* Engineering Team */}
                <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Building className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                        Engineering Team
                      </div>
                      <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">24 Users • 8 Agents</Badge>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div className="space-y-2">
                      <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center">
                            <User className="h-4 w-4 text-green-600 dark:text-green-400" />
                          </div>
                          <div>
                            <p className="font-medium text-sm">john.smith@company.com</p>
                            <p className="text-xs text-muted-foreground">Frontend Lead</p>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Agent-001 (Dedicated)</Badge>
                          <Button size="sm" variant="outline" className="h-6 text-xs">
                            <Edit className="h-3 w-3 mr-1" />
                            Edit
                          </Button>
                        </div>
                      </div>
                      
                      <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center">
                            <User className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                          </div>
                          <div>
                            <p className="font-medium text-sm">sarah.johnson@company.com</p>
                            <p className="text-xs text-muted-foreground">Backend Developer</p>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Pool-Backend (Shared)</Badge>
                          <Button size="sm" variant="outline" className="h-6 text-xs">
                            <Edit className="h-3 w-3 mr-1" />
                            Edit
                          </Button>
                        </div>
                      </div>
                      
                      <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 bg-purple-100 dark:bg-purple-900/30 rounded-full flex items-center justify-center">
                            <User className="h-4 w-4 text-purple-600 dark:text-purple-400" />
                          </div>
                          <div>
                            <p className="font-medium text-sm">mike.chen@company.com</p>
                            <p className="text-xs text-muted-foreground">DevOps Engineer</p>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 text-xs">Pool-DevOps (Shared)</Badge>
                          <Button size="sm" variant="outline" className="h-6 text-xs">
                            <Edit className="h-3 w-3 mr-1" />
                            Edit
                          </Button>
                        </div>
                      </div>
                    </div>
                    
                    <Button variant="outline" size="sm" className="w-full">
                      <Eye className="h-4 w-4 mr-2" />
                      View All Engineering Assignments (24 users)
                    </Button>
                  </CardContent>
                </Card>

                {/* Marketing Team */}
                <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Building className="h-4 w-4 text-green-600 dark:text-green-400" />
                        Marketing Department
                      </div>
                      <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">12 Users • 3 Agents</Badge>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div className="space-y-2">
                      <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center">
                            <User className="h-4 w-4 text-green-600 dark:text-green-400" />
                          </div>
                          <div>
                            <p className="font-medium text-sm">lisa.wong@company.com</p>
                            <p className="text-xs text-muted-foreground">Marketing Director</p>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <Badge className="bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 text-xs">Agent-Marketing-Lead (Dedicated)</Badge>
                          <Button size="sm" variant="outline" className="h-6 text-xs">
                            <Edit className="h-3 w-3 mr-1" />
                            Edit
                          </Button>
                        </div>
                      </div>
                      
                      <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 rounded border dark:border-slate-700">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center">
                            <Users className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                          </div>
                          <div>
                            <p className="font-medium text-sm">Marketing Team (11 users)</p>
                            <p className="text-xs text-muted-foreground">Content creators, analysts, coordinators</p>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Pool-Marketing (Shared)</Badge>
                          <Button size="sm" variant="outline" className="h-6 text-xs">
                            <Eye className="h-3 w-3 mr-1" />
                            View
                          </Button>
                        </div>
                      </div>
                    </div>
                    
                    <Button variant="outline" size="sm" className="w-full">
                      <Eye className="h-4 w-4 mr-2" />
                      View All Marketing Assignments (12 users)
                    </Button>
                  </CardContent>
                </Card>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <Card className="text-center">
                  <CardContent className="p-4">
                    <div className="text-2xl font-bold text-blue-600 dark:text-blue-400 mb-1">47</div>
                    <p className="text-xs text-muted-foreground">Total Users</p>
                  </CardContent>
                </Card>
                
                <Card className="text-center">
                  <CardContent className="p-4">
                    <div className="text-2xl font-bold text-green-600 dark:text-green-400 mb-1">15</div>
                    <p className="text-xs text-muted-foreground">Active Agents</p>
                  </CardContent>
                </Card>
                
                <Card className="text-center">
                  <CardContent className="p-4">
                    <div className="text-2xl font-bold text-purple-600 dark:text-purple-400 mb-1">6</div>
                    <p className="text-xs text-muted-foreground">Agent Pools</p>
                  </CardContent>
                </Card>
                
                <Card className="text-center">
                  <CardContent className="p-4">
                    <div className="text-2xl font-bold text-orange-600 dark:text-orange-400 mb-1">89%</div>
                    <p className="text-xs text-muted-foreground">Utilization Rate</p>
                  </CardContent>
                </Card>
              </div>

              <div className="flex gap-3">
                <Button className="flex-1">
                  <Plus className="h-4 w-4 mr-2" />
                  New Assignment
                </Button>
                <Button variant="outline" className="flex-1">
                  <Bot className="h-4 w-4 mr-2" />
                  Manage Agents
                </Button>
                <Button variant="outline" className="flex-1">
                  <Users className="h-4 w-4 mr-2" />
                  Agent Pools
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Assignment Strategies */}
        <TabsContent value="strategies" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Bot className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                Agent Assignment Strategies
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="border-purple-200 bg-purple-50 dark:bg-purple-950/30 dark:border-purple-800">
                <Bot className="h-4 w-4" />
                <AlertDescription>
                  <strong>Assignment Strategies:</strong> Choose the right strategy based on your organization's needs, 
                  user roles, and resource constraints. Different strategies optimize for performance, cost, or security.
                </AlertDescription>
              </Alert>

              <div className="space-y-6">
                {/* Dedicated Assignment */}
                <Card className="border-2 border-green-200 dark:border-green-800 bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950/30 dark:to-emerald-950/30">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-lg flex items-center gap-2 text-green-800 dark:text-green-300">
                      <User className="h-5 w-5" />
                      Dedicated Agent Assignment
                      <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">Premium</Badge>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <p className="text-sm text-green-700 dark:text-green-300">
                      Each user gets their own dedicated AI agent with persistent context and personalized learning. 
                      Ideal for executive users or specialized roles requiring consistent AI interaction patterns.
                    </p>
                    
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <h5 className="text-sm font-medium text-green-800 dark:text-green-300">Advantages</h5>
                        <ul className="space-y-1">
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>Persistent conversation history</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>Personalized learning and adaptation</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>No wait times or queuing</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>Enhanced privacy and data isolation</span>
                          </li>
                        </ul>
                      </div>
                      
                      <div className="space-y-2">
                        <h5 className="text-sm font-medium text-green-800 dark:text-green-300">Best For</h5>
                        <ul className="space-y-1">
                          <li className="flex items-center gap-2 text-sm">
                            <Crown className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>C-level executives and directors</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <Crown className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>Technical leads and architects</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <Crown className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>High-security sensitive roles</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <Crown className="h-3 w-3 text-green-600 dark:text-green-400" />
                            <span>Heavy AI users (&gt;100 requests/day)</span>
                          </li>
                        </ul>
                      </div>
                    </div>
                    
                    <Alert className="bg-green-100 dark:bg-green-900/30 border-green-300 dark:border-green-700">
                      <AlertCircle className="h-4 w-4" />
                      <AlertDescription className="text-xs">
                        <strong>Resource Impact:</strong> Dedicated agents consume more resources but provide optimal user experience. 
                        Recommended for 10-20% of your user base.
                      </AlertDescription>
                    </Alert>
                  </CardContent>
                </Card>

                {/* Pool-Based Assignment */}
                <Card className="border-2 border-blue-200 dark:border-blue-800 bg-gradient-to-br from-blue-50 to-cyan-50 dark:from-blue-950/30 dark:to-cyan-950/30">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-lg flex items-center gap-2 text-blue-800 dark:text-blue-300">
                      <Users className="h-5 w-5" />
                      Pool-Based Assignment
                      <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">Efficient</Badge>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <p className="text-sm text-blue-700 dark:text-blue-300">
                      Users share access to a pool of agents optimized for their team or department. 
                      Agents are assigned dynamically based on availability and specialization.
                    </p>
                    
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <h5 className="text-sm font-medium text-blue-800 dark:text-blue-300">Advantages</h5>
                        <ul className="space-y-1">
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>Optimal resource utilization</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>Team-specific tool configurations</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>Load balancing and redundancy</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>Cost-effective scaling</span>
                          </li>
                        </ul>
                      </div>
                      
                      <div className="space-y-2">
                        <h5 className="text-sm font-medium text-blue-800 dark:text-blue-300">Pool Types</h5>
                        <ul className="space-y-1">
                          <li className="flex items-center gap-2 text-sm">
                            <Building className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>Department pools (Engineering, Marketing)</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <Building className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>Functional pools (Development, Analysis)</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <Building className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>Project-based pools</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <Building className="h-3 w-3 text-blue-600 dark:text-blue-400" />
                            <span>General-purpose pools</span>
                          </li>
                        </ul>
                      </div>
                    </div>
                    
                    <Alert className="bg-blue-100 dark:bg-blue-900/30 border-blue-300 dark:border-blue-700">
                      <AlertCircle className="h-4 w-4" />
                      <AlertDescription className="text-xs">
                        <strong>Optimization:</strong> Pool size should be configured based on peak usage patterns. 
                        Typical ratio is 1 agent per 3-5 active users.
                      </AlertDescription>
                    </Alert>
                  </CardContent>
                </Card>

                {/* Dynamic Assignment */}
                <Card className="border-2 border-purple-200 dark:border-purple-800 bg-gradient-to-br from-purple-50 to-indigo-50 dark:from-purple-950/30 dark:to-indigo-950/30">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-lg flex items-center gap-2 text-purple-800 dark:text-purple-300">
                      <Zap className="h-5 w-5" />
                      Dynamic Smart Assignment
                      <Badge className="bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 text-xs">AI-Optimized</Badge>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <p className="text-sm text-purple-700 dark:text-purple-300">
                      AI-powered assignment system that dynamically allocates agents based on user needs, 
                      current workload, agent specialization, and historical performance patterns.
                    </p>
                    
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <h5 className="text-sm font-medium text-purple-800 dark:text-purple-300">Intelligence Features</h5>
                        <ul className="space-y-1">
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-purple-600 dark:text-purple-400" />
                            <span>Workload prediction and balancing</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-purple-600 dark:text-purple-400" />
                            <span>Skill-based agent matching</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-purple-600 dark:text-purple-400" />
                            <span>Performance-driven optimization</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <CheckCircle className="h-3 w-3 text-purple-600 dark:text-purple-400" />
                            <span>Context-aware assignments</span>
                          </li>
                        </ul>
                      </div>
                      
                      <div className="space-y-2">
                        <h5 className="text-sm font-medium text-purple-800 dark:text-purple-300">Optimization Factors</h5>
                        <ul className="space-y-1">
                          <li className="flex items-center gap-2 text-sm">
                            <Zap className="h-3 w-3 text-purple-600 dark:text-purple-400" />
                            <span>Current agent availability</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <Zap className="h-3 w-3 text-purple-600 dark:text-purple-400" />
                            <span>User request type and complexity</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <Zap className="h-3 w-3 text-purple-600 dark:text-purple-400" />
                            <span>Historical interaction success</span>
                          </li>
                          <li className="flex items-center gap-2 text-sm">
                            <Zap className="h-3 w-3 text-purple-600 dark:text-purple-400" />
                            <span>Agent specialization match</span>
                          </li>
                        </ul>
                      </div>
                    </div>
                    
                    <Alert className="bg-purple-100 dark:bg-purple-900/30 border-purple-300 dark:border-purple-700">
                      <AlertCircle className="h-4 w-4" />
                      <AlertDescription className="text-xs">
                        <strong>Learning System:</strong> Dynamic assignment improves over time as the system learns user patterns 
                        and agent performance characteristics. Recommended for large, diverse teams.
                      </AlertDescription>
                    </Alert>
                  </CardContent>
                </Card>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Agent Pools */}
        <TabsContent value="pools" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Users className="h-5 w-5 text-orange-600 dark:text-orange-400" />
                Agent Pool Management
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="border-orange-200 bg-orange-50 dark:bg-orange-950/30 dark:border-orange-800">
                <Users className="h-4 w-4" />
                <AlertDescription>
                  <strong>Agent Pools:</strong> Organize agents into logical groups for efficient sharing among users. 
                  Pools can be configured by department, function, or project needs.
                </AlertDescription>
              </Alert>

              <div className="space-y-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Active Agent Pools</h4>
                  <div className="space-y-3">
                    {/* Engineering Pool */}
                    <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
                      <CardHeader className="pb-3">
                        <CardTitle className="text-base flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <Server className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                            Engineering Development Pool
                          </div>
                          <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">5 Agents • 18 Users</Badge>
                        </CardTitle>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <div className="space-y-2">
                            <h5 className="text-sm font-medium">Pool Configuration</h5>
                            <div className="space-y-1 text-sm">
                              <div className="flex justify-between">
                                <span>Pool Size:</span>
                                <span className="font-medium">5 agents</span>
                              </div>
                              <div className="flex justify-between">
                                <span>Max Concurrent:</span>
                                <span className="font-medium">15 sessions</span>
                              </div>
                              <div className="flex justify-between">
                                <span>Scaling:</span>
                                <span className="font-medium">Auto (2-8 agents)</span>
                              </div>
                              <div className="flex justify-between">
                                <span>Specialization:</span>
                                <span className="font-medium">Software Development</span>
                              </div>
                            </div>
                          </div>
                          
                          <div className="space-y-2">
                            <h5 className="text-sm font-medium">Usage Statistics</h5>
                            <div className="space-y-1 text-sm">
                              <div className="flex justify-between">
                                <span>Current Load:</span>
                                <span className="font-medium text-green-600 dark:text-green-400">68% (10/15)</span>
                              </div>
                              <div className="flex justify-between">
                                <span>Peak Today:</span>
                                <span className="font-medium">87% at 2:30 PM</span>
                              </div>
                              <div className="flex justify-between">
                                <span>Avg Response:</span>
                                <span className="font-medium">1.2s</span>
                              </div>
                              <div className="flex justify-between">
                                <span>Requests Today:</span>
                                <span className="font-medium">1,247</span>
                              </div>
                            </div>
                          </div>
                        </div>
                        
                        <div className="flex gap-2">
                          <Button size="sm" variant="outline" className="flex-1">
                            <Eye className="h-3 w-3 mr-1" />
                            View Details
                          </Button>
                          <Button size="sm" variant="outline" className="flex-1">
                            <Settings className="h-3 w-3 mr-1" />
                            Configure
                          </Button>
                          <Button size="sm" variant="outline" className="flex-1">
                            <Users className="h-3 w-3 mr-1" />
                            Manage Users
                          </Button>
                        </div>
                      </CardContent>
                    </Card>

                    {/* Marketing Pool */}
                    <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
                      <CardHeader className="pb-3">
                        <CardTitle className="text-base flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <Globe className="h-4 w-4 text-green-600 dark:text-green-400" />
                            Marketing & Content Pool
                          </div>
                          <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">2 Agents • 11 Users</Badge>
                        </CardTitle>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <div className="space-y-2">
                            <h5 className="text-sm font-medium">Pool Configuration</h5>
                            <div className="space-y-1 text-sm">
                              <div className="flex justify-between">
                                <span>Pool Size:</span>
                                <span className="font-medium">2 agents</span>
                              </div>
                              <div className="flex justify-between">
                                <span>Max Concurrent:</span>
                                <span className="font-medium">6 sessions</span>
                              </div>
                              <div className="flex justify-between">
                                <span>Scaling:</span>
                                <span className="font-medium">Manual</span>
                              </div>
                              <div className="flex justify-between">
                                <span>Specialization:</span>
                                <span className="font-medium">Content & Marketing</span>
                              </div>
                            </div>
                          </div>
                          
                          <div className="space-y-2">
                            <h5 className="text-sm font-medium">Usage Statistics</h5>
                            <div className="space-y-1 text-sm">
                              <div className="flex justify-between">
                                <span>Current Load:</span>
                                <span className="font-medium text-orange-600 dark:text-orange-400">83% (5/6)</span>
                              </div>
                              <div className="flex justify-between">
                                <span>Peak Today:</span>
                                <span className="font-medium text-red-600 dark:text-red-400">100% at 11:15 AM</span>
                              </div>
                              <div className="flex justify-between">
                                <span>Avg Response:</span>
                                <span className="font-medium">2.1s</span>
                              </div>
                              <div className="flex justify-between">
                                <span>Requests Today:</span>
                                <span className="font-medium">342</span>
                              </div>
                            </div>
                          </div>
                        </div>
                        
                        <Alert className="bg-orange-100 dark:bg-orange-900/30 border-orange-300 dark:border-orange-700">
                          <AlertCircle className="h-4 w-4" />
                          <AlertDescription className="text-xs">
                            <strong>Capacity Warning:</strong> This pool frequently reaches capacity. Consider adding an additional agent or implementing auto-scaling.
                          </AlertDescription>
                        </Alert>
                        
                        <div className="flex gap-2">
                          <Button size="sm" variant="outline" className="flex-1">
                            <Plus className="h-3 w-3 mr-1" />
                            Add Agent
                          </Button>
                          <Button size="sm" variant="outline" className="flex-1">
                            <Settings className="h-3 w-3 mr-1" />
                            Auto-Scale
                          </Button>
                          <Button size="sm" variant="outline" className="flex-1">
                            <Eye className="h-3 w-3 mr-1" />
                            Monitor
                          </Button>
                        </div>
                      </CardContent>
                    </Card>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Create New Agent Pool</h4>
                  <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-4 border dark:border-slate-700">
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                      <div className="space-y-3">
                        <div>
                          <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Pool Name</label>
                          <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <input type="text" placeholder="e.g., Data Science Pool" className="w-full bg-transparent border-none outline-none text-sm" />
                          </div>
                        </div>
                        <div>
                          <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Department/Tenant</label>
                          <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <select className="w-full bg-transparent border-none outline-none text-sm">
                              <option>Engineering Team</option>
                              <option>Marketing Department</option>
                              <option>Customer Support</option>
                              <option>Cross-Department</option>
                            </select>
                          </div>
                        </div>
                        <div>
                          <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Specialization</label>
                          <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <select className="w-full bg-transparent border-none outline-none text-sm">
                              <option>General Purpose</option>
                              <option>Software Development</option>
                              <option>Data Analysis</option>
                              <option>Content Creation</option>
                              <option>Customer Support</option>
                              <option>Research & Documentation</option>
                            </select>
                          </div>
                        </div>
                      </div>
                      
                      <div className="space-y-3">
                        <div>
                          <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Initial Pool Size</label>
                          <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <input type="number" defaultValue={2} min={1} max={10} className="w-full bg-transparent border-none outline-none text-sm" />
                          </div>
                        </div>
                        <div>
                          <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Max Concurrent Sessions</label>
                          <div className="p-2 bg-white dark:bg-slate-800 border dark:border-slate-700 rounded">
                            <input type="number" defaultValue={6} min={1} max={50} className="w-full bg-transparent border-none outline-none text-sm" />
                          </div>
                        </div>
                        <div>
                          <label className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 block">Scaling Policy</label>
                          <div className="space-y-2">
                            <div className="flex items-center gap-2">
                              <input type="radio" name="scaling" className="w-3 h-3" defaultChecked />
                              <span className="text-sm">Manual scaling</span>
                            </div>
                            <div className="flex items-center gap-2">
                              <input type="radio" name="scaling" className="w-3 h-3" />
                              <span className="text-sm">Auto-scale (2-8 agents)</span>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                    
                    <div className="flex gap-2 mt-4">
                      <Button size="sm" className="flex-1">
                        <Plus className="h-4 w-4 mr-2" />
                        Create Pool
                      </Button>
                      <Button size="sm" variant="outline" className="flex-1">
                        <Copy className="h-4 w-4 mr-2" />
                        Clone Existing
                      </Button>
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Assignment Management */}
        <TabsContent value="management" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Settings className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
                Assignment Management & Optimization
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert className="border-indigo-200 bg-indigo-50 dark:bg-indigo-950/30 dark:border-indigo-800">
                <Settings className="h-4 w-4" />
                <AlertDescription>
                  <strong>Management Tools:</strong> Advanced features for optimizing agent assignments, 
                  monitoring performance, and automating assignment decisions based on organizational needs.
                </AlertDescription>
              </Alert>

              <div className="space-y-6">
                <div className="space-y-4">
                  <h4 className="font-semibold">Assignment Analytics</h4>
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div className="space-y-4">
                      <h5 className="text-sm font-medium">Performance Metrics</h5>
                      <div className="space-y-3">
                        <Card className="border-green-200 bg-green-50 dark:bg-green-950/30 dark:border-green-800">
                          <CardContent className="p-3">
                            <div className="flex items-center justify-between mb-2">
                              <span className="text-sm font-medium">Overall Satisfaction</span>
                              <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 text-xs">4.7/5.0</Badge>
                            </div>
                            <div className="flex items-center gap-2">
                              <div className="flex-1 bg-green-200 rounded-full h-2">
                                <div className="bg-green-600 h-2 rounded-full" style={{width: '94%'}}></div>
                              </div>
                              <span className="text-xs text-green-700 dark:text-green-300">Excellent</span>
                            </div>
                          </CardContent>
                        </Card>
                        
                        <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950/30 dark:border-blue-800">
                          <CardContent className="p-3">
                            <div className="flex items-center justify-between mb-2">
                              <span className="text-sm font-medium">Resource Utilization</span>
                              <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 text-xs">89%</Badge>
                            </div>
                            <div className="flex items-center gap-2">
                              <div className="flex-1 bg-blue-200 rounded-full h-2">
                                <div className="bg-blue-600 h-2 rounded-full" style={{width: '89%'}}></div>
                              </div>
                              <span className="text-xs text-blue-700 dark:text-blue-300">Optimal</span>
                            </div>
                          </CardContent>
                        </Card>
                        
                        <Card className="border-orange-200 bg-orange-50 dark:bg-orange-950/30 dark:border-orange-800">
                          <CardContent className="p-3">
                            <div className="flex items-center justify-between mb-2">
                              <span className="text-sm font-medium">Response Time</span>
                              <Badge className="bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 text-xs">1.4s avg</Badge>
                            </div>
                            <div className="flex items-center gap-2">
                              <div className="flex-1 bg-orange-200 rounded-full h-2">
                                <div className="bg-orange-600 h-2 rounded-full" style={{width: '76%'}}></div>
                              </div>
                              <span className="text-xs text-orange-700 dark:text-orange-300">Good</span>
                            </div>
                          </CardContent>
                        </Card>
                      </div>
                    </div>
                    
                    <div className="space-y-4">
                      <h5 className="text-sm font-medium">Assignment Distribution</h5>
                      <div className="bg-slate-50 dark:bg-slate-800/50 rounded-lg p-4 border dark:border-slate-700">
                        <div className="space-y-3">
                          <div className="flex items-center justify-between">
                            <span className="text-sm">Dedicated Assignments</span>
                            <div className="flex items-center gap-2">
                              <div className="w-16 bg-green-200 rounded-full h-2">
                                <div className="bg-green-600 h-2 rounded-full" style={{width: '25%'}}></div>
                              </div>
                              <span className="text-xs font-medium">12 users</span>
                            </div>
                          </div>
                          <div className="flex items-center justify-between">
                            <span className="text-sm">Pool Assignments</span>
                            <div className="flex items-center gap-2">
                              <div className="w-16 bg-blue-200 rounded-full h-2">
                                <div className="bg-blue-600 h-2 rounded-full" style={{width: '65%'}}></div>
                              </div>
                              <span className="text-xs font-medium">31 users</span>
                            </div>
                          </div>
                          <div className="flex items-center justify-between">
                            <span className="text-sm">Dynamic Assignments</span>
                            <div className="flex items-center gap-2">
                              <div className="w-16 bg-purple-200 rounded-full h-2">
                                <div className="bg-purple-600 h-2 rounded-full" style={{width: '10%'}}></div>
                              </div>
                              <span className="text-xs font-medium">4 users</span>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Optimization Tools</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    <Card className="border-indigo-200 bg-indigo-50 dark:bg-indigo-950/30 dark:border-indigo-800">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-3">
                          <Zap className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
                          <span className="font-medium text-sm">Auto-Optimization</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          AI-powered assignment optimization based on usage patterns
                        </p>
                        <Button size="sm" variant="outline" className="w-full">
                          <Zap className="h-3 w-3 mr-1" />
                          Run Optimization
                        </Button>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-cyan-200 bg-cyan-50 dark:bg-cyan-950/30 dark:border-cyan-800">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-3">
                          <Monitor className="h-5 w-5 text-cyan-600 dark:text-cyan-400" />
                          <span className="font-medium text-sm">Load Balancer</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          Redistribute users across pools for optimal performance
                        </p>
                        <Button size="sm" variant="outline" className="w-full">
                          <Monitor className="h-3 w-3 mr-1" />
                          Balance Load
                        </Button>
                      </CardContent>
                    </Card>
                    
                    <Card className="border-pink-200 dark:border-pink-800 bg-pink-50 dark:bg-pink-950/50">
                      <CardContent className="p-4">
                        <div className="flex items-center gap-2 mb-3">
                          <Eye className="h-5 w-5 text-pink-600 dark:text-pink-400" />
                          <span className="font-medium text-sm">Assignment Audit</span>
                        </div>
                        <p className="text-xs text-muted-foreground mb-3">
                          Review and validate current assignment configurations
                        </p>
                        <Button size="sm" variant="outline" className="w-full">
                          <Eye className="h-3 w-3 mr-1" />
                          Run Audit
                        </Button>
                      </CardContent>
                    </Card>
                  </div>
                </div>

                <div className="space-y-4">
                  <h4 className="font-semibold">Bulk Operations</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <Button variant="outline" className="justify-start">
                      <Users className="h-4 w-4 mr-2" />
                      Migrate Users Between Pools
                    </Button>
                    <Button variant="outline" className="justify-start">
                      <Copy className="h-4 w-4 mr-2" />
                      Clone Assignment Template
                    </Button>
                    <Button variant="outline" className="justify-start">
                      <Settings className="h-4 w-4 mr-2" />
                      Update Pool Configurations
                    </Button>
                    <Button variant="outline" className="justify-start">
                      <Bot className="h-4 w-4 mr-2" />
                      Provision New Agents
                    </Button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Best Practices */}
      <Card className="bg-gradient-to-r from-cyan-50 dark:from-cyan-950 to-blue-50 dark:to-blue-950 border-cyan-200 dark:border-cyan-800">
        <CardHeader>
          <CardTitle className="text-cyan-800 dark:text-cyan-300">Agent Assignment Best Practices</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-3">
              <h4 className="font-semibold">Strategic Planning</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-cyan-600 dark:text-cyan-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Assess User Needs</p>
                    <p className="text-xs text-muted-foreground">Match assignment strategy to user roles and usage patterns</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-cyan-600 dark:text-cyan-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Monitor Resource Usage</p>
                    <p className="text-xs text-muted-foreground">Track utilization and adjust pool sizes accordingly</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-cyan-600 dark:text-cyan-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Plan for Peak Usage</p>
                    <p className="text-xs text-muted-foreground">Configure scaling policies for high-demand periods</p>
                  </div>
                </li>
              </ul>
            </div>
            
            <div className="space-y-3">
              <h4 className="font-semibold">Operational Excellence</h4>
              <ul className="space-y-2">
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-cyan-600 dark:text-cyan-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Regular Optimization</p>
                    <p className="text-xs text-muted-foreground">Run monthly assignment optimization reviews</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-cyan-600 dark:text-cyan-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">User Feedback Integration</p>
                    <p className="text-xs text-muted-foreground">Collect and act on user satisfaction data</p>
                  </div>
                </li>
                <li className="flex items-start gap-2">
                  <ChevronRight className="h-4 w-4 text-cyan-600 dark:text-cyan-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Cost Optimization</p>
                    <p className="text-xs text-muted-foreground">Balance performance with resource costs</p>
                  </div>
                </li>
              </ul>
            </div>
          </div>
          
          <div className="flex flex-col sm:flex-row gap-3 mt-6">
            <Button className="flex-1">
              <User className="h-4 w-4 mr-2" />
              Optimize Assignments
            </Button>
            <Button variant="outline" className="flex-1">
              <Building className="h-4 w-4 mr-2" />
              Tenant Management
            </Button>
            <Button variant="outline" className="flex-1">
              <UserCheck className="h-4 w-4 mr-2" />
              User Roles & Permissions
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}