"use client"

import type React from "react"

import { cn } from "@/lib/utils"

interface MCPPageLayoutProps {
  children: React.ReactNode
  className?: string
  containerSize?: "sm" | "md" | "lg" | "xl" | "full"
  centerContent?: boolean
  showBackground?: boolean
}

export function MCPPageLayout({ 
  children, 
  className,
  containerSize = "lg",
  centerContent = false,
  showBackground = true
}: MCPPageLayoutProps) {
  const containerSizes = {
    sm: "max-w-md",
    md: "max-w-2xl", 
    lg: "max-w-4xl",
    xl: "max-w-6xl",
    full: "max-w-full"
  }

  return (
    <div className={cn(
      "min-h-screen w-full",
      showBackground && "bg-gradient-to-br from-background to-muted/20",
      centerContent && "flex items-center justify-center",
      !centerContent && "p-4 lg:p-6"
    )}>
      <div className={cn(
        "w-full mx-auto",
        containerSizes[containerSize],
        centerContent && "p-4",
        className
      )}>
        {children}
      </div>
    </div>
  )
}