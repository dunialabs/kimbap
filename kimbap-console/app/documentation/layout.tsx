import type { Metadata } from "next"
import { GlobalFooter } from '@/components/global-footer'

export const metadata: Metadata = {
  title: "Documentation - Kimbap.io MCP Platform",
  description: "Comprehensive documentation for Kimbap.io MCP (Model Context Protocol) platform. Learn how to set up, configure, and manage your AI assistant integrations with tools like GitHub, Notion, PostgreSQL, and more.",
  keywords: "Kimbap.io, MCP, Model Context Protocol, AI, documentation, Claude, Cursor, API integration, tools, GitHub, Notion, PostgreSQL, server management",
  authors: [{ name: "Kimbap.io Team" }],
  robots: "index, follow",
  openGraph: {
    title: "Documentation - Kimbap.io MCP Platform",
    description: "Complete guide to Kimbap.io MCP platform - from quick start to advanced administration",
    type: "website",
    locale: "en_US",
  },
  twitter: {
    card: "summary_large_image",
    title: "Documentation - Kimbap.io MCP Platform",
    description: "Complete guide to Kimbap.io MCP platform - from quick start to advanced administration",
  },
}

export default function DocumentationLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="documentation-layout min-h-screen flex flex-col">
      <div className="flex-1">
        {children}
      </div>
      <GlobalFooter />
    </div>
  )
}