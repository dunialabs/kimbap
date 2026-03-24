"use client"

import type React from "react"

import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

interface MCPPageHeaderProps {
  title: string
  description?: string
  badge?: {
    text: string
    icon?: React.ReactNode
  }
  actions?: React.ReactNode
  centered?: boolean
  className?: string
}

export function MCPPageHeader({ 
  title, 
  description, 
  badge, 
  actions,
  centered = false,
  className 
}: MCPPageHeaderProps) {
  return (
    <div className={cn(
      "mb-8",
      centered && "text-center",
      className
    )}>
      {badge && (
        <div className={cn(
          "mb-4",
          centered && "flex justify-center"
        )}>
          <Badge variant="outline" className="inline-flex items-center gap-2 px-4 py-2 bg-background/60 backdrop-blur-sm border-border/60 text-muted-foreground">
            {badge.icon}
            <span className="text-sm font-medium">{badge.text}</span>
          </Badge>
        </div>
      )}
      
      <div className={cn(
        "space-y-2",
        !centered && actions && "flex items-start justify-between gap-4"
      )}>
        <div className="space-y-2">
          <h1 className="text-3xl font-bold tracking-tight text-foreground">{title}</h1>
          {description && (
            <p className={cn(
              "text-muted-foreground text-lg leading-relaxed",
              centered && "max-w-2xl mx-auto"
            )}>
              {description}
            </p>
          )}
        </div>
        
        {actions && (
          <div className={cn(
            centered && "mt-6 flex justify-center gap-3"
          )}>
            {actions}
          </div>
        )}
      </div>
    </div>
  )
}