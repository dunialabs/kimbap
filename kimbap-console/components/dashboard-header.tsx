import { Menu, Package2 } from "lucide-react"
import Link from "next/link"

import { Button } from "@/components/ui/button"
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet"

export function DashboardHeader() {
  return (
    <header className="flex h-14 items-center gap-4 border-b bg-background px-4 lg:h-[60px] lg:px-6">
      <Sheet>
        <SheetTrigger asChild>
          <Button variant="outline" size="icon" className="shrink-0 md:hidden bg-transparent">
            <Menu className="h-5 w-5" />
            <span className="sr-only">Toggle navigation menu</span>
          </Button>
        </SheetTrigger>
        <SheetContent side="left" className="flex flex-col">
          <nav className="grid gap-2 text-lg font-medium">
            <Link href="#" className="flex items-center gap-2 text-lg font-semibold">
              <Package2 className="h-6 w-6" />
              <span className="sr-only">kimbap.sh</span>
            </Link>
            <Link
              href="/dashboard/policies"
              className="mx-[-0.65rem] flex items-center gap-4 rounded-xl px-3 py-2 text-muted-foreground hover:text-foreground"
            >
              Policies
            </Link>
            <Link
              href="/dashboard/approvals"
              className="mx-[-0.65rem] flex items-center gap-4 rounded-xl bg-muted px-3 py-2 text-foreground hover:text-foreground"
            >
              Approvals
            </Link>
            <Link
              href="/dashboard/usage"
              className="mx-[-0.65rem] flex items-center gap-4 rounded-xl px-3 py-2 text-muted-foreground hover:text-foreground"
            >
              Usage Overview
            </Link>
            <Link
              href="/dashboard/usage/tool-usage"
              className="mx-[-0.65rem] flex items-center gap-4 rounded-xl px-3 py-2 text-muted-foreground hover:text-foreground ml-4"
            >
              Tool Usage
            </Link>
            <Link
              href="/dashboard/usage/token-usage"
              className="mx-[-0.65rem] flex items-center gap-4 rounded-xl px-3 py-2 text-muted-foreground hover:text-foreground ml-4"
            >
              Token Usage
            </Link>
            <Link
              href="/dashboard/logs"
              className="mx-[-0.65rem] flex items-center gap-4 rounded-xl px-3 py-2 text-muted-foreground hover:text-foreground"
            >
              Logs & Monitoring
            </Link>
          </nav>
        </SheetContent>
      </Sheet>

      <div className="w-full flex-1">
        <h1 className="text-lg font-semibold md:text-2xl">Dashboard</h1>
      </div>
    </header>
  )
}
