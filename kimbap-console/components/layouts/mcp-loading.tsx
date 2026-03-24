"use client"

import { Loader2 } from "lucide-react"

import { cn } from "@/lib/utils"

import { MCPCard } from "./mcp-card"

interface MCPLoadingProps {
  title?: string
  description?: string
  fullPage?: boolean
  className?: string
}

export function MCPLoading({ 
  title = "Loading...", 
  description, 
  fullPage = false,
  className 
}: MCPLoadingProps) {
  if (fullPage) {
    return (
      <div className="min-h-screen w-full bg-gradient-to-br from-background to-muted/20 flex items-center justify-center p-4">
        <MCPCard variant="elevated" size="lg" className={cn("w-full max-w-md text-center", className)}>
          <div className="space-y-4">
            <Loader2 className="h-12 w-12 mx-auto animate-spin text-primary" />
            <div className="space-y-2">
              <h3 className="text-xl font-semibold">{title}</h3>
              {description && <p className="text-muted-foreground">{description}</p>}
            </div>
          </div>
        </MCPCard>
      </div>
    )
  }

  return (
    <div className={cn("flex items-center justify-center p-8", className)}>
      <div className="text-center space-y-4">
        <Loader2 className="h-8 w-8 mx-auto animate-spin text-primary" />
        <div className="space-y-1">
          <h3 className="font-medium">{title}</h3>
          {description && <p className="text-sm text-muted-foreground">{description}</p>}
        </div>
      </div>
    </div>
  )
}