'use client'

import { X, Plus } from 'lucide-react'
import * as React from 'react'
import { useRef, useState, useCallback, useMemo, useEffect } from 'react'

import { Badge } from '@/components/ui/badge'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger
} from '@/components/ui/popover'
import { cn } from '@/lib/utils'

interface TagInputProps {
  value: string[]
  onChange: (tags: string[]) => void
  suggestions?: string[]
  placeholder?: string
  maxTags?: number
  maxTagLength?: number
  disabled?: boolean
  className?: string
}

const TAG_PATTERN = /^[a-zA-Z0-9][a-zA-Z0-9._-]*$/

function normalizeTag(raw: string): string {
  return raw.trim().toLowerCase()
}

export function TagInput({
  value,
  onChange,
  suggestions = [],
  placeholder = 'Add tags...',
  maxTags = 50,
  maxTagLength = 32,
  disabled = false,
  className
}: TagInputProps) {
  const [open, setOpen] = useState(false)
  const [inputValue, setInputValue] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const listboxId = React.useId()

  const selectedSet = useMemo(() => new Set(value), [value])

  const availableSuggestions = useMemo(() => {
    const query = normalizeTag(inputValue)
    return suggestions
      .filter((s) => !selectedSet.has(s))
      .filter((s) => !query || s.includes(query))
  }, [suggestions, selectedSet, inputValue])

  const canAddAsNew = useMemo(() => {
    const normalized = normalizeTag(inputValue)
    if (!normalized) return false
    if (normalized.length > maxTagLength) return false
    if (!TAG_PATTERN.test(normalized)) return false
    if (selectedSet.has(normalized)) return false
    if (value.length >= maxTags) return false
    return !suggestions.includes(normalized)
  }, [inputValue, selectedSet, value.length, maxTags, maxTagLength, suggestions])

  const addTag = useCallback(
    (tag: string) => {
      const normalized = normalizeTag(tag)
      if (!normalized) return
      if (normalized.length > maxTagLength) return
      if (!TAG_PATTERN.test(normalized)) return
      if (selectedSet.has(normalized)) return
      if (value.length >= maxTags) return

      onChange([...value, normalized])
      setInputValue('')
    },
    [value, onChange, selectedSet, maxTags, maxTagLength]
  )

  const removeTag = useCallback(
    (tag: string) => {
      onChange(value.filter((t) => t !== tag))
    },
    [value, onChange]
  )

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Backspace' && !inputValue && value.length > 0) {
        removeTag(value[value.length - 1])
        e.preventDefault()
      }
      if ((e.key === 'Enter' || e.key === ',') && inputValue.trim()) {
        e.preventDefault()
        addTag(inputValue)
      }
    },
    [inputValue, value, addTag, removeTag]
  )

  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 0)
    }
  }, [open])

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          role="combobox"
          aria-haspopup="listbox"
          aria-controls={listboxId}
          aria-expanded={open}
          disabled={disabled}
          className={cn(
            'flex min-h-10 w-full flex-wrap items-center gap-1.5 rounded-md border border-input bg-background px-3 py-2 text-sm text-left ring-offset-background',
            'focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:outline-none',
            disabled && 'cursor-not-allowed opacity-50',
            className
          )}
        >
          {value.map((tag) => (
            <Badge
              key={tag}
              variant="secondary"
              className="gap-1 pr-1 text-xs"
            >
              {tag}
              {!disabled && (
                <button
                  type="button"
                  aria-label={`Remove tag ${tag}`}
                  className="ml-0.5 rounded-full p-0.5 hover:bg-muted-foreground/20 cursor-pointer"
                  onKeyDown={(e) => e.stopPropagation()}
                  onClick={(e) => {
                    e.stopPropagation()
                    removeTag(tag)
                  }}
                >
                  <X className="h-3 w-3" aria-hidden="true" />
                </button>
              )}
            </Badge>
          ))}
          {!disabled && value.length < maxTags && (
            <span className="text-muted-foreground text-xs">
              {value.length === 0 ? placeholder : '+'}
            </span>
          )}
        </button>
      </PopoverTrigger>
      <PopoverContent
        className="w-[--radix-popover-trigger-width] p-0"
        align="start"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <Command shouldFilter={false}>
          <CommandInput
            ref={inputRef}
            placeholder="Search or create tag..."
            value={inputValue}
            onValueChange={setInputValue}
            onKeyDown={handleKeyDown}
          />
          <CommandList id={listboxId}>
            {availableSuggestions.length === 0 && !canAddAsNew && (
              <CommandEmpty>
                {inputValue
                  ? 'No matching tags found'
                  : 'No more tags available'}
              </CommandEmpty>
            )}

            {canAddAsNew && (
              <CommandGroup heading="Create new">
                <CommandItem
                  onSelect={() => addTag(inputValue)}
                  className="gap-2"
                >
                  <Plus className="h-3.5 w-3.5 text-muted-foreground" />
                  <span>
                    Create &quot;
                    <span className="font-medium font-mono">
                      {normalizeTag(inputValue)}
                    </span>
                    &quot;
                  </span>
                </CommandItem>
              </CommandGroup>
            )}

            {canAddAsNew && availableSuggestions.length > 0 && (
              <CommandSeparator />
            )}

            {availableSuggestions.length > 0 && (
              <CommandGroup heading="Existing tags">
                {availableSuggestions.map((tag) => (
                  <CommandItem
                    key={tag}
                    onSelect={() => addTag(tag)}
                    className="gap-2"
                  >
                    <div className="h-4 w-4" />
                    <span className="font-mono text-sm">{tag}</span>
                  </CommandItem>
                ))}
              </CommandGroup>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
