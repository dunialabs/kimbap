"use client"

import { Loader2, Server, ArrowLeft } from "lucide-react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import type React from "react"
import { useState } from "react"

import { MCPCard } from "@/components/layouts/mcp-card"
import { MCPPageLayout } from "@/components/layouts/mcp-page-layout"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"



export default function ConnectServerPage() {
  const [isConnecting, setIsConnecting] = useState(false)
  const router = useRouter()

  const handleConnect = (e: React.FormEvent) => {
    e.preventDefault()
    setIsConnecting(true)
    setTimeout(() => {
      setIsConnecting(false)

      // Store the connected server info with admin role (assuming successful connection grants admin access)
      localStorage.setItem(
        "selectedServer",
        JSON.stringify({
          name: "Connected Server", // Could be extracted from form input
          role: "Admin", // Set as admin for manually connected servers
          status: "Running",
        }),
      )

      router.push("/dashboard/tool-configure")
    }, 1500)
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
          title="Connect to a Server"
          description="Enter the server address and access token."
        >
          <form onSubmit={handleConnect} className="space-y-6">
            <div className="grid gap-4">
              <div className="grid gap-2">
                <Label htmlFor="server-address">Server Address</Label>
                <Input id="server-address" placeholder="https://mcp.example.com" required />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="token">Access Token</Label>
                <Input id="token" type="password" placeholder="Enter your access token" required />
              </div>
            </div>
            <div className="flex flex-col gap-4">
              <Button type="submit" className="w-full" disabled={isConnecting}>
                {isConnecting ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Connecting...
                  </>
                ) : (
                  <>
                    <Server className="mr-2 h-4 w-4" />
                    Connect
                  </>
                )}
              </Button>
              <Button variant="link" asChild className="text-muted-foreground">
                <Link href="/">Cancel</Link>
              </Button>
            </div>
          </form>
        </MCPCard>
      </div>
    </MCPPageLayout>
  )
}
