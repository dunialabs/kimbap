"use client"

import type React from "react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

interface MCPCardProps {
  title?: string
  description?: string
  children: React.ReactNode
  className?: string
  variant?: "default" | "elevated" | "outlined" | "ghost"
  size?: "sm" | "md" | "lg"
  hover?: boolean
}

export function MCPCard({ 
  title, 
  description, 
  children, 
  className,
  variant = "default",
  size = "md",
  hover = false
}: MCPCardProps) {
  const variants = {
    default: "bg-card/60 backdrop-blur-sm border-border/60",
    elevated: "bg-card shadow-lg border-border/40 shadow-black/5",
    outlined: "bg-transparent border-2 border-border",
    ghost: "bg-transparent border-none shadow-none"
  }

  const sizes = {
    sm: "p-4",
    md: "p-6", 
    lg: "p-8"
  }

  return (
    <Card className={cn(
      variants[variant],
      hover && "transition-all duration-200 hover:shadow-lg hover:shadow-black/5 hover:border-border/80",
      className
    )}>
      {(title || description) && (
        <CardHeader className={sizes[size]}>
          {title && <CardTitle className="text-xl font-semibold">{title}</CardTitle>}
          {description && <CardDescription className="text-muted-foreground">{description}</CardDescription>}
        </CardHeader>
      )}
      <CardContent className={cn(
        !title && !description && sizes[size],
        title || description ? "pt-0" : ""
      )}>
        {children}
      </CardContent>
    </Card>
  )
}