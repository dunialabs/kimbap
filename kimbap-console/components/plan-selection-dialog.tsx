"use client"

import { Check, Crown, Star, Building, type LucideIcon } from "lucide-react"
import { useState } from "react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, ScrollableDialogContent, DialogTitle } from "@/components/ui/dialog"
import { PLAN_DISPLAY, getPricingUrl, resolvePlanId, type PlanId } from "@/lib/plan-config"

interface PlanConfig {
  name: string
  icon: LucideIcon
  description: string
  features: string[]
  popular: boolean
}

const plans: Record<PlanId, PlanConfig> = {
  free: {
    name: PLAN_DISPLAY.free.name,
    icon: Star,
    description: "For getting started with MCP",
    features: [
      "Up to 30 Tools",
      "Up to 30 Access Tokens",
      "Unlimited API requests",
      "Community support",
    ],
    popular: false,
  },
  pro: {
    name: PLAN_DISPLAY.pro.name,
    icon: Crown,
    description: "For teams",
    features: [
      "Everything in Community, plus:",
      "Up to 100 Tools & 100 Access Tokens",
      "Advanced client management",
      "24/7 priority support",
      "OAuth & API key auth",
      "Advanced analytics dashboard",
    ],
    popular: true,
  },
  enterprise: {
    name: PLAN_DISPLAY.enterprise.name,
    icon: Building,
    description: "For large organizations",
    features: [
      "Everything in Business, plus:",
      "Unlimited Tools & Access Tokens",
      "On-premise deployment",
      "Custom SLA",
      "White-label options",
      "24/7 phone support",
    ],
    popular: false,
  },
}

const planOrder: PlanId[] = ["free", "pro", "enterprise"]

interface PlanSelectionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentPlan?: PlanId
}

export function PlanSelectionDialog({ open, onOpenChange, currentPlan = "free" }: PlanSelectionDialogProps) {
  const [showUpgradeDialog, setShowUpgradeDialog] = useState(false)
  const [selectedPlan, setSelectedPlan] = useState<PlanId | null>(null)

  const handlePlanSelect = (planKey: PlanId) => {
    setSelectedPlan(planKey)
    setShowUpgradeDialog(true)
  }

  const handleGetLicense = () => {
    window.open(getPricingUrl(), "_blank", "noopener,noreferrer")
    setShowUpgradeDialog(false)
    onOpenChange(false)
  }

  const handleActivateLicense = () => {
    setShowUpgradeDialog(false)
    onOpenChange(false)
    window.location.href = "/dashboard/billing"
  }

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <ScrollableDialogContent className="max-w-4xl">
          <DialogHeader>
            <DialogTitle className="text-2xl">Plans &amp; Licensing</DialogTitle>
            <DialogDescription>
              Compare plans and activate a license.
            </DialogDescription>
          </DialogHeader>

          <div>
          <div className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              {planOrder.map((key) => {
                const plan = plans[key]
                return (
                <Card key={key} className={`relative ${plan.popular ? "ring-2 ring-blue-500 shadow-lg" : ""}`}>
                  {plan.popular && (
                    <div className="absolute -top-3 left-1/2 transform -translate-x-1/2">
                      <Badge className="bg-blue-500 text-white">Popular</Badge>
                    </div>
                  )}

                  <CardHeader className="text-center pb-4">
                    <div className="flex items-center justify-center mb-4">
                      <plan.icon className="h-8 w-8 text-blue-600 dark:text-blue-400" />
                    </div>
                    <CardTitle className="text-xl">{plan.name}</CardTitle>
                    <CardDescription className="text-sm">{plan.description}</CardDescription>
                  </CardHeader>

                  <CardContent className="space-y-4">
                    <div className="space-y-2">
                      {plan.features.map((feature, index) => (
                        <div key={index} className="flex items-start gap-2">
                          <Check className="h-4 w-4 text-green-500 dark:text-green-400 mt-0.5 flex-shrink-0" />
                          <span className="text-xs">{feature}</span>
                        </div>
                      ))}
                    </div>

                    <div className="pt-4">
                      {resolvePlanId(currentPlan) === key ? (
                        <Button variant="outline" className="w-full bg-transparent" disabled>
                          Current Plan
                        </Button>
                      ) : key === "enterprise" ? (
                        <Button
                          className="w-full"
                          onClick={() => window.open(getPricingUrl(), "_blank", "noopener,noreferrer")}
                        >
                          Contact Sales
                        </Button>
                      ) : (
                        <Button className="w-full" onClick={() => handlePlanSelect(key)}>
                          {resolvePlanId(currentPlan) === "free" ? "Upgrade" : "Switch"} to {plan.name}
                        </Button>
                      )}
                    </div>
                  </CardContent>
                </Card>
                )
              })}
            </div>
          </div>
          </div>
        </ScrollableDialogContent>
      </Dialog>

      <Dialog open={showUpgradeDialog} onOpenChange={setShowUpgradeDialog}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Upgrade to {selectedPlan ? plans[selectedPlan].name : ""}</DialogTitle>
            <DialogDescription>
              Buy a license on kimbap.io, then activate it here.
            </DialogDescription>
          </DialogHeader>

          <DialogFooter>
            <Button variant="outline" onClick={handleActivateLicense}>Activate Key</Button>
            <Button onClick={handleGetLicense}>Get License</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
