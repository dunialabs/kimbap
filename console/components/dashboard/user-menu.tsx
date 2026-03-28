"use client"

import { CircleUser } from "lucide-react"

import { CardTitle } from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { clearAuthState } from "@/lib/api-client"

export interface UserData {
  email: string
}

interface UserMenuProps {
  currentUser: UserData
  onLogout: () => void
}

export function UserMenu({
  currentUser,
  onLogout,
}: UserMenuProps) {
  return (
      <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`Open account menu for ${currentUser.email}`}
          className="flex w-full items-center justify-between rounded-md px-2 py-1.5 transition-colors duration-200 hover:bg-accent/60 hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        >
          <div className="flex items-center gap-2 flex-1">
            <CircleUser className="h-5 w-5 text-muted-foreground" />
            <div className="flex-1 min-w-0">
              <CardTitle className="truncate text-sm" title={currentUser.email}>
                {currentUser.email}
              </CardTitle>
            </div>
          </div>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuLabel>
          <span className="block max-w-[220px] truncate" title={currentUser.email}>{currentUser.email}</span>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={() => {
          clearAuthState()
          onLogout()
        }}>
          Sign out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
