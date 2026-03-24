"use client"

import { useState } from "react"
import Image from "next/image"
import { ChevronRight, Book, Rocket, Users, Shield, Code, HelpCircle, Server, Monitor, Settings, Key, Download, CheckCircle, AlertCircle, Zap, Crown, UserCheck, User, Network, CreditCard, BarChart, Terminal, Lock, RefreshCw, Upload, GitBranch, Bell, MessageSquare, X, Menu, Globe } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { cn } from "@/lib/utils"
import { GettingStartedSection } from "./sections/getting-started-section"
import { OverviewSection } from "./sections/overview-section"
import { WhyChooseSection } from "./sections/why-choose-section"
import { QuickSetupSection } from "./sections/quick-setup-section"
import { QuickStartSection } from "./sections/quick-start-section"
import { DeploymentGuideSection } from "./sections/deployment-guide-section"
import { ConsoleOverviewSection } from "./sections/console-overview-section"
import { InstallationSection } from "./sections/installation-section"
import { FirstServerSection } from "./sections/first-server-section"
import { DashboardSection } from "./sections/dashboard-section"
import { ToolConfigurationSection } from "./sections/tool-configuration-section"
import { MemberManagementSection } from "./sections/member-management-section"
import { AnalyticsSection } from "./sections/analytics-section"
import { ServerManagementSection } from "./sections/server-management-section"
import { SecuritySettingsSection } from "./sections/security-settings-section"
import { NetworkAccessSection } from "./sections/network-access-section"
import { BillingSection } from "./sections/billing-section"
import { DeskOverviewSection } from "./sections/desk-overview-section"
import { DeskDownloadSection } from "./sections/desk-download-section"
import { DeskSetupSection } from "./sections/desk-setup-section"
import { ClientConfigSection } from "./sections/client-config-section"
import { ConnectionManagementSection } from "./sections/connection-management-section"
import { TenantManagementSection } from "./sections/tenant-management-section"
import { UserRolesSection } from "./sections/user-roles-section"
import { AgentAssignmentSection } from "./sections/agent-assignment-section"
import { ApiReferenceSection } from "./sections/api-reference-section"
import { TroubleshootingSection } from "./sections/troubleshooting-section"

export function WikiSimple() {
  const [activeSection, setActiveSection] = useState("getting-started")
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(["getting-started"]))
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)

  const toggleSection = (sectionId: string) => {
    const newExpanded = new Set(expandedSections)
    if (newExpanded.has(sectionId)) {
      newExpanded.delete(sectionId)
    } else {
      newExpanded.add(sectionId)
    }
    setExpandedSections(newExpanded)
  }

  const sections = [
    {
      id: "getting-started",
      title: "Getting Started",
      icon: Rocket,
      subsections: [
        { id: "welcome", title: "Welcome to Kimbap.io", icon: Book },
        { id: "platform-overview", title: "Platform Overview", icon: Globe },
        { id: "why-choose-kimbap", title: "Why Choose Kimbap.io", icon: Zap }
      ]
    },
    {
      id: "deployment",
      title: "Installation",
      icon: Download,
      subsections: [
        { id: "quick-setup", title: "Quick Setup Guide", icon: Rocket },
        { id: "installation", title: "Install MCP Console", icon: Download },
        { id: "desk-download", title: "Download MCP Desk", icon: Monitor },
        { id: "deployment-guide", title: "Advanced Deployment", icon: Server }
      ]
    },
    {
      id: "administration",
      title: "Kimbap MCP Console",
      icon: Monitor,
      subsections: [
        { id: "console-overview", title: "Console Overview", icon: Monitor },
        { id: "setup", title: "Initial Setup", icon: Settings },
        { id: "dashboard", title: "Dashboard", icon: BarChart },
        { id: "tools", title: "Tools", icon: Settings },
        { id: "members", title: "Access Tokens", icon: Users },
        { id: "analytics", title: "Usage Analytics", icon: BarChart },
        { id: "security", title: "Security Settings", icon: Lock },
        { id: "network", title: "Network Access", icon: Network },
        { id: "billing", title: "Billing & License", icon: CreditCard }
      ]
    },
    {
      id: "user-guide",
      title: "Kimbap Desk",
      icon: Shield,
      subsections: [
        { id: "desk-overview", title: "MCP Desk Overview", icon: Shield },
        { id: "desk-setup", title: "Connect to Servers", icon: Settings },
        { id: "client-config", title: "AI Client Setup", icon: Code },
        { id: "connections", title: "Manage Connections", icon: Network }
      ]
    },
    {
      id: "developer",
      title: "Developer Guide",
      icon: Code,
      subsections: [
        { id: "api-reference", title: "API Reference", icon: Code },
        { id: "custom-tools", title: "Building Custom Tools", icon: Settings },
        { id: "sdk-docs", title: "SDK Documentation", icon: Book },
        { id: "webhooks", title: "Webhooks & Events", icon: Bell }
      ]
    },
    {
      id: "multitenancy",
      title: "Multi-Tenancy & Teams",
      icon: Users,
      subsections: [
        { id: "tenant-management", title: "Tenants", icon: Crown },
        { id: "user-roles", title: "Roles & Permissions", icon: UserCheck },
        { id: "agent-assignment", title: "Agent Assignment", icon: User },
        { id: "team-collaboration", title: "Team Collaboration", icon: Users }
      ]
    },
    {
      id: "troubleshooting",
      title: "Troubleshooting",
      icon: HelpCircle,
      subsections: [
        { id: "diagnostics", title: "Diagnostics", icon: Terminal },
        { id: "common-issues", title: "Common Issues", icon: AlertCircle },
        { id: "performance", title: "Performance Tuning", icon: Zap },
        { id: "faq", title: "FAQ", icon: HelpCircle },
        { id: "support", title: "Get Support", icon: MessageSquare }
      ]
    },
    {
      id: "reference",
      title: "Reference",
      icon: Book,
      subsections: [
        { id: "configuration", title: "Configuration Reference", icon: Settings },
        { id: "environment-vars", title: "Environment Variables", icon: Terminal },
        { id: "cli-reference", title: "CLI Commands", icon: Terminal },
        { id: "glossary", title: "Glossary", icon: Book }
      ]
    }
  ]

  const renderContent = () => {
    switch (activeSection) {
      case "getting-started":
      case "welcome":
        return <GettingStartedSection onNavigate={(sectionId) => {
          setActiveSection(sectionId);
          // Also expand the parent section if needed
          if (sectionId === 'console-overview') {
            const newExpanded = new Set(expandedSections);
            newExpanded.add('administration');
            setExpandedSections(newExpanded);
          } else if (sectionId === 'desk-overview') {
            const newExpanded = new Set(expandedSections);
            newExpanded.add('user-guide');
            setExpandedSections(newExpanded);
          }
          setIsMobileMenuOpen(false);
        }} />
        
      case "platform-overview":
        return <OverviewSection />

      case "why-choose-kimbap":
        return <WhyChooseSection />

      case "quick-setup":
        return <QuickSetupSection />
        
      case "deployment":
      case "deployment-guide":
        return <DeploymentGuideSection />

      case "console-overview":
        return <ConsoleOverviewSection />

      case "installation":
        return <InstallationSection />

      case "desk-download":
        return <DeskDownloadSection />

      case "setup":
        return <FirstServerSection />

      case "dashboard":
        return <DashboardSection />

      case "tools":
        return <ToolConfigurationSection />

      case "members":
        return <MemberManagementSection />

      case "analytics":
        return <AnalyticsSection />

      case "security":
        return <SecuritySettingsSection />

      case "network":
        return <NetworkAccessSection />

      case "billing":
        return <BillingSection />

      case "desk-overview":
        return <DeskOverviewSection />


      case "desk-setup":
        return <DeskSetupSection />

      case "client-config":
        return <ClientConfigSection />

      case "connections":
        return <ConnectionManagementSection />

      case "tenant-management":
        return <TenantManagementSection />

      case "user-roles":
        return <UserRolesSection />

      case "agent-assignment":
        return <AgentAssignmentSection />

      case "api-reference":
      case "custom-tools":
      case "sdk-docs":
      case "webhooks":
        return <ApiReferenceSection />
        
      case "troubleshooting":
      case "diagnostics":
      case "common-issues":
      case "performance":
        return <TroubleshootingSection />
        
      case "team-collaboration":
        return <TenantManagementSection />
        
      case "common":
        return (
          <div className="space-y-6">
            <h2 className="text-2xl font-bold">Common Issues & Solutions</h2>
            
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Cannot Access Console</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                 <div className="p-3 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800 rounded">
                  <p className="text-sm font-medium text-red-900 dark:text-red-100 mb-2">Symptoms:</p>
                  <ul className="text-sm text-red-800 dark:text-red-200 space-y-1">
                    <li>• Page doesn't load or shows error</li>
                    <li>• Master password not accepted</li>
                    <li>• Session expires immediately</li>
                  </ul>
                </div>
                
                 <div className="p-3 bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-800 rounded">
                  <p className="text-sm font-medium text-green-900 dark:text-green-100 mb-2">Solutions:</p>
                  <ol className="text-sm text-green-800 dark:text-green-200 space-y-1">
                    <li>1. Clear browser cache and cookies</li>
                    <li>2. Verify master password with administrator</li>
                    <li>3. Check if IP is whitelisted (if enabled)</li>
                    <li>4. Try incognito/private browsing mode</li>
                    <li>5. Ensure JavaScript is enabled</li>
                  </ol>
                </div>
              </CardContent>
            </Card>
            
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Tool Connection Failures</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                 <div className="p-3 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800 rounded">
                  <p className="text-sm font-medium text-red-900 dark:text-red-100 mb-2">Error Messages:</p>
                  <ul className="text-sm text-red-800 dark:text-red-200 space-y-1">
                    <li>• "Authentication failed"</li>
                    <li>• "Connection timeout"</li>
                    <li>• "Invalid credentials"</li>
                  </ul>
                </div>
                
                 <div className="p-3 bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-800 rounded">
                  <p className="text-sm font-medium text-green-900 dark:text-green-100 mb-2">Troubleshooting Steps:</p>
                  <ol className="text-sm text-green-800 dark:text-green-200 space-y-1">
                    <li>1. Verify API keys/tokens are correct</li>
                    <li>2. Check tool service status</li>
                    <li>3. Test network connectivity</li>
                    <li>4. Review firewall settings</li>
                    <li>5. Check rate limits</li>
                  </ol>
                </div>
                
                <Alert>
                  <Terminal className="h-4 w-4" />
                  <AlertDescription className="text-xs">
                    Use the "Test Connection" button in tool settings to diagnose issues
                  </AlertDescription>
                </Alert>
              </CardContent>
            </Card>
          </div>
        )

      case "faq":
      case "support":
        return <TroubleshootingSection />
        
      case "configuration":
      case "environment-vars":
      case "cli-reference":
      case "glossary":
        return (
          <div className="space-y-6">
            <h2 className="text-2xl font-bold">Reference Documentation</h2>
            
            <Card>
              <CardHeader>
                <CardTitle className="text-base">General Questions</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-3">
                  <div>
                    <p className="font-medium text-sm mb-1">What is the difference between MCP Console and MCP Desk?</p>
                    <p className="text-sm text-muted-foreground">
                      MCP Console is the web-based management interface for administrators, while MCP Desk is the desktop client that end users install to connect AI assistants to MCP servers.
                    </p>
                  </div>
                  
                  <div>
                    <p className="font-medium text-sm mb-1">Can I use Kimbap.io without Claude or Cursor?</p>
                    <p className="text-sm text-muted-foreground">
                      Yes, Kimbap.io supports any MCP-compatible client. You can also integrate with custom applications using our API.
                    </p>
                  </div>
                  
                  <div>
                    <p className="font-medium text-sm mb-1">How many servers can I manage?</p>
                    <p className="text-sm text-muted-foreground">
                      The number of servers depends on your subscription plan. Business and Enterprise plans support additional servers.
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
            
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Security Questions</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-3">
                  <div>
                    <p className="font-medium text-sm mb-1">How are credentials stored?</p>
                    <p className="text-sm text-muted-foreground">
                      All credentials are encrypted using AES-256 encryption before storage. Keys are never logged or transmitted in plain text.
                    </p>
                  </div>
                  
                  <div>
                    <p className="font-medium text-sm mb-1">What happens if I lose my Owner Token?</p>
                    <p className="text-sm text-muted-foreground">
                      Owner tokens cannot be recovered if lost. You'll need to reinitialize the server, which will invalidate all existing access tokens.
                    </p>
                  </div>
                  
                  <div>
                    <p className="font-medium text-sm mb-1">Is my data encrypted in transit?</p>
                    <p className="text-sm text-muted-foreground">
                      Yes, all communication uses TLS 1.3 encryption for data in transit.
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        )

      default:
        return (
          <div className="space-y-6">
            <Card>
              <CardContent className="pt-6">
                <div className="text-center py-12">
                  <Book className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                  <h3 className="text-lg font-medium mb-2">Documentation Section</h3>
                  <p className="text-sm text-muted-foreground">
                    Select a topic from the sidebar to view documentation
                  </p>
                   <div className="mt-6 bg-slate-100 dark:bg-slate-800 rounded-lg p-6">
                    <p className="text-sm text-muted-foreground">
                      [Content Placeholder]
                    </p>
                    <p className="text-xs text-muted-foreground mt-2">
                      Additional documentation content will be added here
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        )
    }
  }

  return (
    <div className="flex h-full bg-gradient-to-br from-slate-50 via-white to-blue-50/30 dark:from-slate-950 dark:via-slate-900 dark:to-blue-950/30">
      {/* Mobile Menu Button */}
      <Button
        variant="ghost"
        size="icon"
        className="fixed top-4 left-4 z-50 md:hidden bg-white/80 backdrop-blur-sm border border-slate-200/60 hover:bg-white shadow-lg dark:bg-slate-800/80 dark:border-slate-700/60 dark:hover:bg-slate-800"
        onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
        aria-label={isMobileMenuOpen ? 'Close documentation navigation' : 'Open documentation navigation'}
        aria-expanded={isMobileMenuOpen}
        aria-controls="documentation-sidebar"
      >
        {isMobileMenuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
      </Button>

      {/* Sidebar Navigation */}
      <aside id="documentation-sidebar" className={cn(
        "w-64 lg:w-72 border-r border-slate-200/60 bg-white/90 backdrop-blur-sm overflow-y-auto flex-shrink-0 dark:border-slate-700/60 dark:bg-slate-800/90",
        "fixed md:relative inset-y-0 left-0 z-40 shadow-xl md:shadow-none",
        "transform transition-transform duration-200 ease-in-out",
        isMobileMenuOpen ? "translate-x-0" : "-translate-x-full md:translate-x-0"
      )}>
        <div className="p-5">
          <div className="flex items-center gap-3 mb-5">
            <Image
              src="/logo-icon.png"
              alt="Kimbap.io"
              width={24}
              height={24}
              className="flex-shrink-0"
            />
            <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">Documentation</h2>
          </div>
          <nav className="space-y-0.5">
            {sections.map((section) => (
              <div key={section.id}>
                <button
                  type="button"
                  onClick={() => {
                    if (section.subsections.length > 0) {
                      toggleSection(section.id)
                    } else {
                      setActiveSection(section.id)
                      setIsMobileMenuOpen(false)
                    }
                  }}
                  className={cn(
                    "w-full flex items-center gap-2.5 px-3 py-2 text-sm rounded-lg transition-all duration-150",
                    activeSection === section.id
                      ? "bg-blue-600 text-white"
                      : "text-slate-700 hover:bg-slate-100 hover:text-slate-900 dark:text-slate-300 dark:hover:bg-slate-700 dark:hover:text-slate-100"
                  )}
                >
                  <section.icon className="h-4 w-4" />
                  <span className="flex-1 text-left">{section.title}</span>
                  {section.subsections.length > 0 && (
                    <ChevronRight
                      className={cn(
                        "h-4 w-4 transition-transform",
                        expandedSections.has(section.id) && "rotate-90"
                      )}
                    />
                  )}
                </button>
                {section.subsections.length > 0 && expandedSections.has(section.id) && (
                  <div className="ml-3 mt-1 space-y-0.5">
                    {section.subsections.map((subsection) => (
                      <button
                        type="button"
                        key={subsection.id}
                        onClick={() => {
                          setActiveSection(subsection.id)
                          setIsMobileMenuOpen(false)
                        }}
                        className={cn(
                          "w-full flex items-center gap-2 px-3 py-1.5 text-xs rounded-md transition-all duration-150",
                          activeSection === subsection.id
                            ? "bg-blue-500 text-white"
                            : "text-slate-600 hover:bg-slate-100 hover:text-slate-800 dark:text-slate-400 dark:hover:bg-slate-700 dark:hover:text-slate-200"
                        )}
                      >
                        <subsection.icon className="h-3 w-3 flex-shrink-0" />
                        <span className="text-left">{subsection.title}</span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </nav>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-y-auto">
        <div className="container max-w-6xl mx-auto p-4 md:p-6 lg:p-8">
          <div className="max-w-none">
            <div className="relative">
              {renderContent()}
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}
