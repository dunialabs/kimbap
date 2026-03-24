'use client'

import { Check, Wrench, Asterisk } from 'lucide-react'
import * as React from 'react'
import { useRef, useState, useCallback, useMemo, useEffect } from 'react'

import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { cn } from '@/lib/utils'

export interface ToolSuggestion {
  name: string
  serverName?: string
  description?: string
}

interface ToolPatternInputProps {
  value: string
  onChange: (value: string) => void
  suggestions?: ToolSuggestion[]
  placeholder?: string
  disabled?: boolean
  className?: string
}

export function ToolPatternInput({
  value,
  onChange,
  suggestions = [],
  placeholder = 'e.g., delete_*, *',
  disabled = false,
  className,
}: ToolPatternInputProps) {
  const [open, setOpen] = useState(false)
  const [inputValue, setInputValue] = useState(value)
  const inputRef = useRef<HTMLInputElement>(null)
  const cancelCloseRef = useRef(false)

  useEffect(() => {
    setInputValue(value)
  }, [value])

  const filteredSuggestions = useMemo(() => {
    const query = inputValue.trim().toLowerCase()
    if (!query || query === '*') return suggestions
    return suggestions.filter(
      (s) =>
        s.name.toLowerCase().includes(query) ||
        s.serverName?.toLowerCase().includes(query) ||
        s.description?.toLowerCase().includes(query)
    )
  }, [suggestions, inputValue])

  const isPattern = useMemo(() => {
    const trimmed = inputValue.trim()
    return trimmed.includes('*') || trimmed.includes('?')
  }, [inputValue])

  const isExactMatch = useMemo(() => {
    return suggestions.some((s) => s.name === inputValue.trim())
  }, [suggestions, inputValue])

  const showCustomOption = useMemo(() => {
    const trimmed = inputValue.trim()
    if (!trimmed) return false
    if (isExactMatch) return false
    return true
  }, [inputValue, isExactMatch])

  const handleSelect = useCallback(
    (toolName: string) => {
      onChange(toolName)
      setInputValue(toolName)
      setOpen(false)
    },
    [onChange]
  )

  const handleInputChange = useCallback(
    (newValue: string) => {
      setInputValue(newValue)
    },
    []
  )

  const handleEscapeKeyDown = useCallback(() => {
    cancelCloseRef.current = true
    setInputValue(value)
  }, [value])

  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 0)
    }
  }, [open])

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) {
        if (cancelCloseRef.current) {
          cancelCloseRef.current = false
        } else {
          const trimmed = inputValue.trim()
          if (trimmed !== value) {
            onChange(trimmed)
          }
        }
      }
      setOpen(nextOpen)
    },
    [inputValue, value, onChange]
  )

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <button
          type="button"
          role="combobox"
          aria-expanded={open}
          disabled={disabled}
          className={cn(
            'flex h-9 w-full items-center rounded-md border border-input bg-background px-3 text-sm font-mono text-left ring-offset-background',
            'focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:outline-none',
            !value && 'text-muted-foreground',
            disabled && 'cursor-not-allowed opacity-50',
            className
          )}
        >
          <span className="truncate">{value || placeholder}</span>
        </button>
      </PopoverTrigger>
      <PopoverContent
        className="w-[--radix-popover-trigger-width] p-0"
        align="start"
        onOpenAutoFocus={(e) => e.preventDefault()}
        onEscapeKeyDown={handleEscapeKeyDown}
      >
        <Command shouldFilter={false}>
          <CommandInput
            ref={inputRef}
            placeholder="Search tools or type a pattern..."
            value={inputValue}
            onValueChange={handleInputChange}
          />
          <CommandList>
            {showCustomOption && (
              <>
                <CommandGroup heading={isPattern ? 'Pattern' : 'Custom'}>
                  <CommandItem
                    onSelect={() => handleSelect(inputValue.trim())}
                    className="gap-2"
                  >
                    <Asterisk className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                    <span>
                      Use{' '}
                      <span className="font-medium font-mono">
                        &quot;{inputValue.trim()}&quot;
                      </span>
                    </span>
                  </CommandItem>
                </CommandGroup>
                {filteredSuggestions.length > 0 && <CommandSeparator />}
              </>
            )}

            {filteredSuggestions.length > 0 && (
              <CommandGroup heading="Available tools">
                {filteredSuggestions.map((tool) => (
                  <CommandItem
                    key={`${tool.serverName || ''}-${tool.name}`}
                    onSelect={() => handleSelect(tool.name)}
                    className="gap-2"
                  >
                    {value === tool.name ? (
                      <Check className="h-3.5 w-3.5 shrink-0 text-primary" />
                    ) : (
                      <Wrench className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                    )}
                    <div className="min-w-0 flex-1">
                      <span className="font-mono text-sm">{tool.name}</span>
                      {(tool.serverName || tool.description) && (
                        <p className="text-xs text-muted-foreground truncate">
                          {tool.serverName}
                          {tool.serverName && tool.description ? ' · ' : ''}
                          {tool.description}
                        </p>
                      )}
                    </div>
                  </CommandItem>
                ))}
              </CommandGroup>
            )}

            {filteredSuggestions.length === 0 && !showCustomOption && (
              <CommandEmpty>
                {suggestions.length === 0
                  ? 'No tools available'
                  : 'No matching tools'}
              </CommandEmpty>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
