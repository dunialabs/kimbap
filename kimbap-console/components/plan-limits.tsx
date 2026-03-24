"use client"
import { AlertTriangle, Crown, Star } from "lucide-react"


import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { cn } from "@/lib/utils"
import { getPricingUrl } from "@/lib/plan-config"


interface PlanLimitsProps {
  currentPlan?: "free" | "pro" | "enterprise"
  serverCount?: number
  toolCount?: number
  tokenCount?: number
  apiUsage?: number
  maxServers?: number
  maxTools?: number
  maxTokens?: number
  maxApiRequests?: number
}

const planDetails = {
  free: {
    name: "Community",
    icon: Star,
    maxServers: 1,
    maxTools: 30,
    maxTokens: 30,
    maxApiRequests: 3000,
    color: "bg-gray-100 dark:bg-gray-800 text-gray-800 dark:text-gray-200 border-gray-200 dark:border-gray-700",
  },
  pro: {
    name: "Business",
    icon: Crown,
    maxServers: 10,
    maxTools: 100,
    maxTokens: 100,
    maxApiRequests: 1000000,
    color: "bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-300 border-purple-200 dark:border-purple-800",
  },
  enterprise: {
    name: "Enterprise",
    icon: Crown,
    maxServers: -1, // unlimited
    maxTools: -1, // unlimited
    maxTokens: -1, // unlimited
    maxApiRequests: -1, // unlimited
    color: "bg-amber-100 dark:bg-amber-900/30 text-amber-800 dark:text-amber-300 border-amber-200 dark:border-amber-800",
  },
}

function formatNumber(num: number): string {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + "M"
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + "K"
  }
  return num.toString()
}

export function PlanLimits({
  currentPlan = "free",
  serverCount = 1,
  toolCount = 3,
  tokenCount = 2,
  apiUsage = 1250,
  maxServers,
  maxTools,
  maxTokens,
  maxApiRequests,
}: PlanLimitsProps) {
  const plan = planDetails[currentPlan]
  const actualMaxServers = maxServers || plan.maxServers
  const actualMaxTools = maxTools || plan.maxTools
  const actualMaxTokens = maxTokens || plan.maxTokens
  const actualMaxApiRequests = maxApiRequests || plan.maxApiRequests

  const serverUsagePercent = actualMaxServers === -1 ? 0 : (serverCount / actualMaxServers) * 100
  const toolUsagePercent = actualMaxTools === -1 ? 0 : (toolCount / actualMaxTools) * 100
  const tokenUsagePercent = actualMaxTokens === -1 ? 0 : (tokenCount / actualMaxTokens) * 100
  const apiUsagePercent = actualMaxApiRequests === -1 ? 0 : (apiUsage / actualMaxApiRequests) * 100

  const isServerLimitReached = actualMaxServers !== -1 && serverCount >= actualMaxServers
  const isToolLimitReached = actualMaxTools !== -1 && toolCount >= actualMaxTools
  const isTokenLimitReached = actualMaxTokens !== -1 && tokenCount >= actualMaxTokens
  const isApiLimitReached = actualMaxApiRequests !== -1 && apiUsage >= actualMaxApiRequests

  const anyLimitReached = isServerLimitReached || isToolLimitReached || isTokenLimitReached || isApiLimitReached
  const anyLimitApproaching =
    serverUsagePercent >= 80 || toolUsagePercent >= 80 || tokenUsagePercent >= 80 || apiUsagePercent >= 80

  return (
    <Card className="mb-6">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-lg">
              Plan Usage
            </CardTitle>
            <CardDescription>Current plan limits and usage</CardDescription>
          </div>
          <Badge className={plan.color}>{plan.name} Plan</Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Server Usage */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">Servers</span>
            <span className="text-sm text-muted-foreground">
              {serverCount} / {actualMaxServers === -1 ? "∞" : actualMaxServers}
            </span>
          </div>
          {actualMaxServers !== -1 && (
            <Progress
              value={serverUsagePercent}
              className={`h-2 ${serverUsagePercent >= 80 ? "bg-red-100 dark:bg-red-900/30" : "bg-gray-100 dark:bg-gray-800"}`}
            />
          )}
          {isServerLimitReached && (
            <div className="flex items-center gap-2 text-sm text-amber-600 dark:text-amber-400">
              <AlertTriangle className="h-4 w-4" />
              <span>Server limit reached</span>
            </div>
          )}
        </div>

        {/* Tool Usage */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">Tools (Current Server)</span>
            <span className="text-sm text-muted-foreground">
              {toolCount} / {actualMaxTools === -1 ? "∞" : actualMaxTools}
            </span>
          </div>
          {actualMaxTools !== -1 && (
            <Progress
              value={toolUsagePercent}
              className={`h-2 ${toolUsagePercent >= 80 ? "bg-red-100 dark:bg-red-900/30" : "bg-gray-100 dark:bg-gray-800"}`}
            />
          )}
          {isToolLimitReached && (
            <div className="flex items-center gap-2 text-sm text-amber-600 dark:text-amber-400">
              <AlertTriangle className="h-4 w-4" />
              <span>Tool limit reached</span>
            </div>
          )}
        </div>

        {/* Token Usage */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">Access Tokens (Current Server)</span>
            <span className="text-sm text-muted-foreground">
              {tokenCount} / {actualMaxTokens === -1 ? "∞" : actualMaxTokens}
            </span>
          </div>
          {actualMaxTokens !== -1 && (
            <Progress
              value={tokenUsagePercent}
              className={`h-2 ${tokenUsagePercent >= 80 ? "bg-red-100 dark:bg-red-900/30" : "bg-gray-100 dark:bg-gray-800"}`}
            />
          )}
          {isTokenLimitReached && (
            <div className="flex items-center gap-2 text-sm text-amber-600 dark:text-amber-400">
              <AlertTriangle className="h-4 w-4" />
              <span>Access token limit reached</span>
            </div>
          )}
        </div>

        {/* API Usage */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">API Requests (This Month)</span>
            <span className="text-sm text-muted-foreground">
              {formatNumber(apiUsage)} / {actualMaxApiRequests === -1 ? "∞" : formatNumber(actualMaxApiRequests)}
            </span>
          </div>
          {actualMaxApiRequests !== -1 && (
            <Progress
              value={apiUsagePercent}
              className={`h-2 ${apiUsagePercent >= 80 ? "bg-red-100 dark:bg-red-900/30" : "bg-gray-100 dark:bg-gray-800"}`}
            />
          )}
          {isApiLimitReached && (
            <div className="flex items-center gap-2 text-sm text-amber-600 dark:text-amber-400">
              <AlertTriangle className="h-4 w-4" />
              <span>API request limit reached</span>
            </div>
          )}
        </div>

        {/* Upgrade Prompt */}
        {(anyLimitReached || anyLimitApproaching) && currentPlan !== "enterprise" && (
          <div className="mt-4 p-3 bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-950/20 dark:to-purple-950/20 border border-blue-200 dark:border-blue-800 rounded-md">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-blue-900 dark:text-blue-100">
                  {anyLimitReached ? "Limit Reached" : "Approaching Limit"}
                </p>
                <p className="text-xs text-blue-700 dark:text-blue-200">
                  {currentPlan === "pro"
                    ? "Purchase add-ons or upgrade to Enterprise"
                    : "Upgrade to get more resources"}
                </p>
              </div>
              <a
                href={getPricingUrl()}
                target="_blank"
                rel="noopener noreferrer"
                className={cn(
                  buttonVariants({ size: 'sm' }),
                  'bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700'
                )}
              >
                  <Crown className="mr-1 h-3 w-3" />
                  Get License
              </a>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
