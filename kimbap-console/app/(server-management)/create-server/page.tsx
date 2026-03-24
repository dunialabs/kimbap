"use client"

import { Loader2, Server, ArrowLeft, CheckCircle, Wrench, ArrowRight } from "lucide-react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { useState } from "react"

import { MCPCard } from "@/components/layouts/mcp-card"
import { MCPLoading } from "@/components/layouts/mcp-loading"
import { MCPPageLayout } from "@/components/layouts/mcp-page-layout"
import { Button } from "@/components/ui/button"

export default function CreateServerPage() {
  const [isCreating, setIsCreating] = useState(false)
  const [isCreated, setIsCreated] = useState(false)
  const router = useRouter()

  const serverName = "user's server" // Default server name

  const handleCreateServer = async () => {
    setIsCreating(true)

    // Simulate server creation
    await new Promise((resolve) => setTimeout(resolve, 2000))

    // Store the newly created server info with owner role
    localStorage.setItem(
      "selectedServer",
      JSON.stringify({
        name: serverName,
        role: "Owner",
        status: "Running",
      }),
    )

    setIsCreating(false)
    setIsCreated(true)
  }

  const handleGoToTools = () => {
    router.push("/dashboard/tool-configure")
  }

  const handleGoToDashboard = () => {
    router.push("/dashboard")
  }

  if (isCreated) {
    return (
      <MCPPageLayout containerSize="md" centerContent>
        <MCPCard variant="elevated" size="lg" className="text-center">
          <div className="space-y-6">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-green-50 dark:bg-green-950/20 border-2 border-green-100 dark:border-green-800">
              <CheckCircle className="h-8 w-8 text-green-600 dark:text-green-400" />
            </div>
            <div className="space-y-2">
              <h1 className="text-2xl font-bold text-green-900 dark:text-green-100">Server Created Successfully!</h1>
              <p className="text-base text-muted-foreground">
                Your MCP server "{serverName}" is now running and ready to use.
              </p>
            </div>
            <div className="p-4 bg-green-50 dark:bg-green-950/20 rounded-lg border border-green-200 dark:border-green-800">
              <div className="flex items-center gap-3 mb-3">
                <Server className="h-5 w-5 text-green-600 dark:text-green-400" />
                <div>
                  <h3 className="font-semibold text-green-900 dark:text-green-100">{serverName}</h3>
                  <div className="flex items-center gap-2 mt-1">
                    <span className="relative flex h-3 w-3">
                      <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                      <span className="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
                    </span>
                    <span className="text-sm font-medium text-green-700 dark:text-green-300">Running</span>
                  </div>
                </div>
              </div>
              <div className="grid grid-cols-3 gap-4 text-center">
                <div>
                  <div className="text-2xl font-bold text-green-900 dark:text-green-100">0</div>
                  <div className="text-xs text-green-700 dark:text-green-300">Tools Configured</div>
                </div>
                <div>
                  <div className="text-2xl font-bold text-green-900 dark:text-green-100">0</div>
                  <div className="text-xs text-green-700 dark:text-green-300">Access Tokens</div>
                </div>
                <div>
                  <div className="text-2xl font-bold text-green-900 dark:text-green-100">0</div>
                  <div className="text-xs text-green-700 dark:text-green-300">API Requests</div>
                </div>
              </div>
            </div>

            <div className="space-y-4">
              <h3 className="text-lg font-semibold text-center">Next Steps</h3>

              <div className="p-4 border border-blue-200 dark:border-blue-800 bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-950/20 dark:to-indigo-950/20 rounded-lg">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <Wrench className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                      <div>
                        <h4 className="font-semibold text-blue-900 dark:text-blue-100">Add Tools</h4>
                        <p className="text-sm text-blue-700 dark:text-blue-300">Configure tools like Web Search, GitHub, etc.</p>
                      </div>
                    </div>
                    <Button onClick={handleGoToTools} className="bg-blue-600 hover:bg-blue-700">
                      Configure Tools
                      <ArrowRight className="ml-2 h-4 w-4" />
                    </Button>
                  </div>
              </div>

              <div className="text-center">
                <Button variant="outline" onClick={handleGoToDashboard}>
                  Go to Dashboard
                </Button>
              </div>
            </div>
          </div>
        </MCPCard>
      </MCPPageLayout>
    )
  }

  if (isCreating) {
    return (
      <MCPLoading 
        fullPage
        title="Creating Your Server..."
        description={`Setting up "${serverName}" and starting the MCP server.`}
      />
    )
  }

  return (
    <MCPPageLayout containerSize="sm" centerContent>
      <div className="w-full relative">
        <div className="absolute -top-16 left-0">
          <Link
            href="/"
            onClick={() => {
              localStorage.removeItem('userid')
            }}
          >
            <Button variant="ghost" size="sm" className="text-muted-foreground hover:text-foreground">
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to Home
            </Button>
          </Link>
        </div>

        <MCPCard 
          variant="elevated"
          title="Create a New Server"
          description="Your server will be created with default settings. You can configure it afterward."
        >
          <div className="space-y-4">
            <div className="p-3 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-md">
              <p className="text-sm text-blue-800 dark:text-blue-200">
                <strong>What happens next:</strong> Your server starts immediately. Configure tools and tokens from the
                dashboard.
              </p>
            </div>

            <Button onClick={handleCreateServer} className="w-full">
              <Server className="mr-2 h-4 w-4" />
              Create Server
            </Button>
          </div>
        </MCPCard>
      </div>
    </MCPPageLayout>
  )
}
