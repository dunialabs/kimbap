"use client"

import { RefreshCw, Server } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

interface ServerUpdatesDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  servers: any[]
}

export function ServerUpdatesDialog({ open, onOpenChange, servers }: ServerUpdatesDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <RefreshCw className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            Servers with Updates
          </DialogTitle>
          <DialogDescription>
            These servers have updates ready to sync.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3 max-h-60 overflow-y-auto">
          {servers.map((server) => (
            <Card key={server.id} className="border border-slate-200 dark:border-slate-700">
              <CardContent className="p-3">
                <div className="flex items-center gap-3">
                  <Server className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="text-sm font-medium text-slate-900 dark:text-slate-100">{server.name}</span>
                      <Badge className="text-xs font-medium bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 border-red-200 dark:border-red-800">
                        {server.updates?.length || 0} update{(server.updates?.length || 0) !== 1 ? "s" : ""}
                      </Badge>
                    </div>
                    <p className="text-xs text-slate-500 dark:text-slate-400">{server.address}</p>
                    <p className="text-xs text-slate-600 dark:text-slate-400 mt-1">Tools changed and are ready to sync.</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>

        <div className="bg-blue-50 dark:bg-blue-950/20 p-3 rounded-lg border border-blue-200 dark:border-blue-800">
          <p className="text-xs text-blue-800 dark:text-blue-200">
            <strong>Note:</strong> Sync applies these updates to connected MCP clients.
          </p>
        </div>

        <DialogFooter>
          <Button onClick={() => onOpenChange(false)}>Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
