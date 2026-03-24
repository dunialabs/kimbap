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
          className="flex items-center justify-between w-full cursor-pointer rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        >
          <div className="flex items-center gap-2 flex-1">
            <CircleUser className="h-5 w-5 text-muted-foreground" />
            <div className="flex-1">
              <CardTitle className="text-sm">
                {currentUser.email}
              </CardTitle>
            </div>
          </div>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuLabel>
          <span>{currentUser.email}</span>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={() => {
          localStorage.removeItem('userid')
          localStorage.removeItem('token')
          localStorage.removeItem('auth_token')
          localStorage.removeItem('accessToken')
          localStorage.removeItem('manualAccessToken')
          localStorage.removeItem('selectedServer')
          onLogout()
        }}>
          Log Out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
