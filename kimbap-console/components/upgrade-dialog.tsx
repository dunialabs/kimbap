"use client"

import { AlertTriangle, Crown, Building, Key, Zap } from "lucide-react"
import { Button, buttonVariants } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { cn } from "@/lib/utils"
import { getPricingUrl } from "@/lib/plan-config"

interface UpgradeDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  type: "server" | "tool" | "token" | "api"
  currentPlan?: "free" | "pro" | "enterprise"
}

const planRecommendations = {
  server: {
    free: {
      title: "Server Limit Reached",
      description: "You've reached the maximum of 1 server on the Community plan.",
      recommendation: "pro",
    },
    pro: {
      title: "Server Limit Reached",
      description: "You've reached the maximum of 10 servers on the Business plan.",
      recommendation: "enterprise",
    },
  },
  tool: {
    free: {
      title: "Tool Limit Reached",
      description: "You've reached the maximum of 30 tools per server on the Community plan.",
      recommendation: "pro",
    },
    pro: {
      title: "Tool Limit Reached",
      description: "You've reached the maximum of 100 tools per server on the Business plan.",
      recommendation: "enterprise",
    },
  },
  token: {
    free: {
      title: "Access Token Limit Reached",
      description: "You've reached the maximum of 30 access tokens on the Community plan.",
      recommendation: "pro",
    },
    pro: {
      title: "Access Token Limit Reached",
      description:
        "You've reached the maximum of 100 access tokens on the Business plan. You can purchase additional tokens or upgrade to Enterprise.",
      recommendation: "enterprise",
    },
  },
  api: {
    free: {
      title: "API Request Limit Reached",
      description: "You've reached the maximum of 3,000 API requests per month on the Community plan.",
      recommendation: "pro",
    },
    pro: {
      title: "API Request Limit Reached",
      description:
        "You've reached the maximum of 1,000,000 API requests per month on the Business plan. You can purchase additional requests or upgrade to Enterprise.",
      recommendation: "enterprise",
    },
  },
}

const planDetails = {
  pro: {
    name: "Business",
    icon: Crown,
    features: [
      "10 Servers",
      "100 Tools per server",
      "100 Access tokens",
      "1M API requests/month",
    ],
  },
  enterprise: {
    name: "Enterprise",
    icon: Building,
    features: ["Unlimited servers", "Unlimited tools", "Unlimited tokens", "Unlimited API requests"],
  },
}

export function UpgradeDialog({ open, onOpenChange, type, currentPlan = "free" }: UpgradeDialogProps) {
  const recommendation = planRecommendations[type][currentPlan as keyof (typeof planRecommendations)[typeof type]]
  const recommendedPlan = planDetails[recommendation.recommendation as keyof typeof planDetails]

  if (!recommendation || !recommendedPlan) return null

  const isProAddOnAvailable = currentPlan === "pro" && (type === "token" || type === "api")

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <div className="flex items-center gap-2 mb-2">
            <AlertTriangle className="h-5 w-5 text-amber-500 dark:text-amber-400" />
            <DialogTitle>{recommendation.title}</DialogTitle>
          </div>
          <DialogDescription>{recommendation.description}</DialogDescription>
        </DialogHeader>

        <div className="py-4 space-y-4">
          {/* Main Recommendation */}
          <div className="bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-950/20 dark:to-purple-950/20 p-4 rounded-lg border border-blue-200 dark:border-blue-800">
            <div className="flex items-center gap-3 mb-3">
              <recommendedPlan.icon className="h-6 w-6 text-blue-600 dark:text-blue-400" />
              <div>
                <h3 className="font-semibold text-blue-900 dark:text-blue-100">Upgrade to {recommendedPlan.name}</h3>
                <p className="text-sm text-blue-700 dark:text-blue-200">Pricing on kimbap.io</p>
              </div>
            </div>
            <ul className="space-y-1">
              {recommendedPlan.features.map((feature) => (
                <li key={feature} className="text-sm text-blue-800 dark:text-blue-200 flex items-center gap-2">
                  <span className="w-1 h-1 bg-blue-600 rounded-full"></span>
                  {feature}
                </li>
              ))}
            </ul>
          </div>

          {/* Pro Add-on Option */}
          {isProAddOnAvailable && (
            <div className="bg-gradient-to-r from-purple-50 to-pink-50 dark:from-purple-950/20 dark:to-pink-950/20 p-4 rounded-lg border border-purple-200 dark:border-purple-800">
              <div className="flex items-center gap-3 mb-3">
                {type === "token" ? (
                  <Key className="h-6 w-6 text-purple-600 dark:text-purple-400" />
                ) : (
                  <Zap className="h-6 w-6 text-purple-600 dark:text-purple-400" />
                )}
                <div>
                  <h3 className="font-semibold text-purple-900 dark:text-purple-100">
                    Buy {type === "token" ? "Additional Tokens" : "Additional API Requests"}
                  </h3>
                  <p className="text-sm text-purple-700 dark:text-purple-200">
                    Add {type === "token" ? "tokens" : "API requests"} to your Business plan
                  </p>
                </div>
              </div>
              <p className="text-sm text-purple-800 dark:text-purple-200">Keep Business and add capacity.</p>
            </div>
          )}
        </div>

        <DialogFooter className="flex gap-2">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Later
          </Button>
          <a
            href={getPricingUrl()}
            target="_blank"
            rel="noopener noreferrer"
            className={cn(
              buttonVariants(),
              'bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700'
            )}
          >
            <recommendedPlan.icon className="mr-2 h-4 w-4" />
            Get License
          </a>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
