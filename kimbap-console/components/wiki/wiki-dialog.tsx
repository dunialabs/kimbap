"use client"

import { useState } from "react"
import { Book, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { WikiSimple } from "./wiki-simple"

export function WikiDialog() {
  const [open, setOpen] = useState(false)

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={() => setOpen(true)}
        className="gap-2"
      >
        <Book className="h-4 w-4" />
        Docs
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-[90vw] w-full h-[90vh] p-0">
          <DialogHeader className="sr-only">
            <DialogTitle>Documentation</DialogTitle>
          </DialogHeader>
          <div className="relative h-full">
            <Button
              variant="ghost"
              size="icon"
              className="absolute right-4 top-4 z-50"
              onClick={() => setOpen(false)}
              aria-label="Close documentation dialog"
            >
              <X className="h-4 w-4" />
            </Button>
            <WikiSimple />
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
