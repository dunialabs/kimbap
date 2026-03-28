'use client'

import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { toast } from 'sonner'
import {
  Plus,
  Pencil,
  Trash2,
  Shield,
  ChevronDown,
  ChevronUp,
  ArrowUp,
  ArrowDown,
  X,
  Loader2,
  AlertTriangle,
} from 'lucide-react'

import { api } from '@/lib/api-client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  ScrollableDialogContent,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Separator } from '@/components/ui/separator'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { useUserRole } from '@/hooks/use-user-role'
import { cn, formatDateTime, formatDisplayNumber } from '@/lib/utils'

function ToolPatternInput({
  value,
  onChange,
  placeholder,
  className,
}: {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  className?: string
}) {
  return (
    <Input
      placeholder={placeholder}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className={className}
    />
  )
}

interface ExtractEntry {
  id: string
  name: string
  path: string
  type: 'string' | 'number' | 'boolean' | 'url.host' | 'bytes.length'
}

interface ConditionEntry {
  id: string
  left: string
  op: 'eq' | 'neq' | 'gt' | 'gte' | 'lt' | 'lte' | 'in' | 'not_in' | 'matches'
  right: string
}

interface PolicyRule {
  id: string
  priority: number
  match: { tool: string; serverId: string }
  extract: ExtractEntry[]
  when: ConditionEntry[]
  effect: { decision: 'ALLOW' | 'REQUIRE_APPROVAL' | 'DENY'; reason: string }
}

interface PolicySet {
  id: string
  serverId: string | null
  version: number
  status: string
  dsl: { rules: PolicyRule[] }
  createdAt: string
  updatedAt: string
}

interface SerializedRule {
  id: string
  priority: number
  match: { tool?: string; serverId?: string }
  extract?: Record<string, { path: string; type: string }>
  when?: Array<{ left: string; op: string; right: unknown }>
  effect: { decision: string; reason?: string }
}

const OPERATORS = [
  { value: 'eq', label: '=' },
  { value: 'neq', label: '≠' },
  { value: 'gt', label: '>' },
  { value: 'gte', label: '≥' },
  { value: 'lt', label: '<' },
  { value: 'lte', label: '≤' },
  { value: 'in', label: 'in' },
  { value: 'not_in', label: 'not in' },
  { value: 'matches', label: 'matches' },
] as const



const EXTRACT_TYPES = [
  { value: 'string', label: 'String' },
  { value: 'number', label: 'Number' },
  { value: 'boolean', label: 'Boolean' },
  { value: 'url.host', label: 'URL Host' },
  { value: 'bytes.length', label: 'Byte Length' },
] as const

const DECISIONS = [
  { value: 'ALLOW', label: 'Allow', color: 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20 dark:bg-emerald-500/20 dark:text-emerald-400 dark:border-emerald-500/40' },
  {
    value: 'REQUIRE_APPROVAL',
    label: 'Needs approval',
    color: 'bg-amber-500/10 text-amber-600 border-amber-500/20 dark:bg-amber-500/20 dark:text-amber-400 dark:border-amber-500/40',
  },
  { value: 'DENY', label: 'Block', color: 'bg-red-500/10 text-red-600 border-red-500/20 dark:bg-red-500/20 dark:text-red-400 dark:border-red-500/40' },
] as const

function getRequestErrorMessage(
  error: unknown,
  messages: { auth: string; network: string; fallback: string }
): string {
  const requestError = error as {
    response?: { status?: number; data?: { common?: { message?: string } } }
    userMessage?: string
    message?: string
    code?: string
  }
  const status = requestError.response?.status
  const rawMessage =
    requestError.userMessage ||
    requestError.response?.data?.common?.message ||
    requestError.message ||
    ''
  if (status === 401 || status === 403) return rawMessage || messages.auth
  if (!requestError.response || requestError.code === 'ECONNABORTED') return messages.network
  return rawMessage || messages.fallback
}

function generateId(): string {
  return crypto.randomUUID()
}

function emptyRule(): PolicyRule {
  return {
    id: generateId(),
    priority: 1000,
    match: { tool: '*', serverId: '' },
    extract: [],
    when: [],
    effect: { decision: 'ALLOW', reason: '' },
  }
}

function isConditionOp(value: unknown): value is ConditionEntry['op'] {
  return OPERATORS.some((operator) => operator.value === value)
}

function isExtractType(value: unknown): value is ExtractEntry['type'] {
  return EXTRACT_TYPES.some((extractType) => extractType.value === value)
}

function isDecision(value: unknown): value is PolicyRule['effect']['decision'] {
  return DECISIONS.some((decision) => decision.value === value)
}

function serializeRules(rules: PolicyRule[]): SerializedRule[] {
  return rules.map((r) => {
    const rule: SerializedRule = {
      id: r.id,
      priority: r.priority,
      match: {},
      effect: { decision: r.effect.decision },
    }
    if (r.match.tool) rule.match.tool = r.match.tool
    if (r.match.serverId) rule.match.serverId = r.match.serverId
    if (r.extract.length > 0) {
      const extractEntries: Record<string, { path: string; type: string }> = {}
      r.extract.forEach((e) => {
        if (e.name && e.name.trim()) {
          extractEntries[e.name.trim()] = { path: e.path, type: e.type }
        }
      })
      if (Object.keys(extractEntries).length > 0) {
        rule.extract = extractEntries
      }
    }
    if (r.when.length > 0) {
      rule.when = r.when.map((w) => {
        let rightVal: unknown = w.right
        if (w.op === 'in' || w.op === 'not_in') {
          try {
            rightVal = JSON.parse(w.right)
          } catch {
            rightVal = w.right.split(',').map((s: string) => s.trim())
          }
        } else if (w.right === 'true') {
          rightVal = true
        } else if (w.right === 'false') {
          rightVal = false
        } else if (!isNaN(Number(w.right)) && w.right !== '') {
          rightVal = Number(w.right)
        }
        return { left: w.left, op: w.op, right: rightVal }
      })
    }
    if (r.effect.reason) rule.effect.reason = r.effect.reason
    return rule
  })
}

function deserializeRules(raw: unknown): PolicyRule[] {
  if (!Array.isArray(raw)) return []
  return raw.map((item) => {
    const ruleData = item && typeof item === 'object' ? (item as Record<string, unknown>) : {}
    const extract: ExtractEntry[] = []
    const extractData = ruleData.extract
    if (extractData && typeof extractData === 'object' && !Array.isArray(extractData)) {
      Object.entries(extractData as Record<string, unknown>).forEach(([name, val]) => {
        const extractEntry = val && typeof val === 'object' ? (val as Record<string, unknown>) : {}
        extract.push({
          id: generateId(),
          name,
          path: typeof extractEntry.path === 'string' ? extractEntry.path : '',
          type: isExtractType(extractEntry.type) ? extractEntry.type : 'string',
        })
      })
    }

    const whenData = Array.isArray(ruleData.when) ? ruleData.when : []
    const when: ConditionEntry[] = whenData.map((whenItem) => {
      const condition = whenItem && typeof whenItem === 'object' ? (whenItem as Record<string, unknown>) : {}
      const rightValue = condition.right

      return {
          id: generateId(),
          left: typeof condition.left === 'string' ? condition.left : String(condition.left ?? ''),
          op: isConditionOp(condition.op) ? condition.op : 'eq',
          right: Array.isArray(rightValue)
            ? rightValue.length === 1
              ? String(rightValue[0] ?? '')
              : JSON.stringify(rightValue)
            : String(rightValue ?? ''),
        }
    })

    const matchData =
      ruleData.match && typeof ruleData.match === 'object' ? (ruleData.match as Record<string, unknown>) : {}
    const effectData =
      ruleData.effect && typeof ruleData.effect === 'object' ? (ruleData.effect as Record<string, unknown>) : {}

    return {
      id: typeof ruleData.id === 'string' ? ruleData.id : generateId(),
      priority: typeof ruleData.priority === 'number' ? ruleData.priority : 1000,
      match: {
        tool: typeof matchData.tool === 'string' ? matchData.tool : '',
        serverId: typeof matchData.serverId === 'string' ? matchData.serverId : '',
      },
      extract,
      when,
      effect: {
        decision: isDecision(effectData.decision) ? effectData.decision : 'ALLOW',
        reason: typeof effectData.reason === 'string' ? effectData.reason : '',
      },
    }
  })
}

function generateRuleSummary(rule: PolicyRule): string {
  const decision = DECISIONS.find((d) => d.value === rule.effect.decision)?.label || rule.effect.decision
  const tool = rule.match.tool === '*' || !rule.match.tool ? 'All tools' : rule.match.tool
  const serverPart = rule.match.serverId ? ` on ${rule.match.serverId}` : ''

  let condPart = ''
  if (rule.when.length > 0) {
    const cond = rule.when[0]
    const opLabel = OPERATORS.find((o) => o.value === cond.op)?.label || cond.op
    condPart = ` (when ${cond.left} ${opLabel} ${cond.right}`
    if (rule.when.length > 1) condPart += `, +${rule.when.length - 1} more`
    condPart += ')'
  }

  let summary = `${tool}${serverPart} → ${decision}${condPart}`
  if (summary.length > 110) summary = `${summary.slice(0, 107)}…`
  return summary
}

function generatePolicyTitle(rules: PolicyRule[]): string {
  if (!rules || rules.length === 0) return 'Empty policy'
  const first = rules[0]
  const decision = DECISIONS.find((d) => d.value === first.effect.decision)?.label || first.effect.decision
  const tool = first.match.tool === '*' || !first.match.tool ? 'All tools' : first.match.tool
  const serverPart = first.match.serverId ? ` on ${first.match.serverId}` : ''
  const condPart = (first.when?.length ?? 0) > 0 ? ' (conditional)' : ''
  return `${tool}${serverPart} — ${decision}${condPart}`
}

function getCatchAllRuleWarning(rule: PolicyRule): string | null {
  const matchesAllTools = !rule.match.tool || rule.match.tool === '*'

  if (!matchesAllTools || rule.when.length > 0) {
    return null
  }

  switch (rule.effect.decision) {
    case 'ALLOW':
      return 'This rule will allow all tool calls unconditionally. If it stays above more specific rules, those later rules will never be checked. Add a specific tool pattern or a condition to narrow the scope.'
    case 'REQUIRE_APPROVAL':
      return 'This rule will send every tool call to the approval queue. If it stays above more specific rules, those later rules will never be checked. Add a specific tool pattern or a condition to narrow the scope.'
    case 'DENY':
      return 'This rule will block all tool calls unconditionally. If it stays above more specific rules, those later rules will never be checked. Add a specific tool pattern or a condition to narrow the scope.'
    default:
      return null
  }
}

function RuleCard({
  rule,
  index,
  expanded,
  canMoveUp,
  canMoveDown,
  onToggle,
  onChange,
  onMove,
  onRemove,
}: {
  rule: PolicyRule
  index: number
  expanded: boolean
  canMoveUp: boolean
  canMoveDown: boolean
  onToggle: () => void
  onChange: (updated: PolicyRule) => void
  onMove: (direction: 'up' | 'down') => void
  onRemove: () => void
}) {
  const [extractOpen, setExtractOpen] = useState(rule.extract.length > 0)
  const decisionMeta = DECISIONS.find((d) => d.value === rule.effect.decision)
  const catchAllRuleWarning = getCatchAllRuleWarning(rule)

  return (
    <Card className="border border-border/60 shadow-sm">
      <div className="flex items-center gap-2 rounded-t-lg bg-muted/30 px-4 py-3">
        <button
          type="button"
          onClick={onToggle}
          aria-expanded={expanded}
          className="flex min-h-11 min-w-0 flex-1 items-center gap-2 rounded text-left transition-colors duration-200 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        >
          {expanded ? <ChevronUp className="h-4 w-4 shrink-0" /> : <ChevronDown className="h-4 w-4 shrink-0" />}
          <span className="shrink-0 text-sm font-medium">Rule {index + 1}</span>
          <Badge variant="secondary" className="shrink-0 text-[11px]">
            {index === 0 ? 'First match' : `Priority ${index + 1}`}
          </Badge>
          <Badge variant="outline" className={`shrink-0 text-xs ${decisionMeta?.color || ''}`}>
            {decisionMeta?.label || rule.effect.decision}
          </Badge>
          <span className="truncate text-xs text-muted-foreground" title={generateRuleSummary(rule)}>{generateRuleSummary(rule)}</span>
        </button>

        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-11 w-11"
            onClick={() => onMove('up')}
            disabled={!canMoveUp}
            aria-label={`Move rule ${index + 1} up`}
          >
            <ArrowUp className="h-3.5 w-3.5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-11 w-11"
            onClick={() => onMove('down')}
            disabled={!canMoveDown}
            aria-label={`Move rule ${index + 1} down`}
          >
            <ArrowDown className="h-3.5 w-3.5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-11 w-11 text-destructive hover:bg-destructive/10 hover:text-destructive"
            onClick={onRemove}
            aria-label={`Remove rule ${index + 1}`}
          >
            <X className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      {expanded && (
        <CardContent className="space-y-5 pt-4">
          <div className="space-y-3">
            <Label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Matching</Label>
            <div className="space-y-1.5">
              <Label className="text-xs">Tool name or pattern</Label>
              <ToolPatternInput
                placeholder="e.g., delete_*, *"
                value={rule.match.tool}
                onChange={(v) => onChange({ ...rule, match: { ...rule.match, tool: v } })}
                className="h-11"
              />
              <p className="text-xs text-muted-foreground">
                Use <code className="font-mono">*</code> to match all tools, or <code className="font-mono">prefix_*</code> to match tools starting with a prefix.
              </p>
            </div>
          </div>

          <Separator />

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Conditions</Label>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-11 text-xs"
                onClick={() =>
                  onChange({
                    ...rule,
                    when: [...rule.when, { id: generateId(), left: '', op: 'eq', right: '' }],
                  })
                }
              >
                <Plus className="mr-1 h-3 w-3" />
                Add Condition
              </Button>
            </div>

            <div className="rounded-md border border-border/60 bg-muted/20 px-3 py-2 text-xs leading-5 text-muted-foreground">
              Add a condition when this rule should apply only to specific values. You can compare a literal directly, or
              first extract a field below and reference it here as <code className="font-mono">$name</code>. Example:
              extract <code className="font-mono">$domain</code> from <code className="font-mono">url</code> as{' '}
              <code className="font-mono">URL Host</code>, then compare <code className="font-mono">$domain</code>{' '}
              to <code className="font-mono">api.stripe.com</code>.
            </div>

            {rule.when.length === 0 && rule.extract.length === 0 && (
              <p className="text-xs italic text-muted-foreground">No conditions — applies to all matching tool calls.</p>
            )}

            {rule.when.length === 0 && rule.extract.length > 0 && (
              <p className="text-xs italic text-muted-foreground">
                Extracted fields do nothing until a condition references them as <code className="font-mono">$name</code>.
              </p>
            )}

            {rule.when.map((cond, ci) => (
              <div key={cond.id} className="flex flex-col items-stretch gap-2 sm:flex-row sm:items-end">
                <div className="flex-1 space-y-1">
                   <Label className="text-xs">Field, variable, or literal</Label>
                   <Input
                     placeholder="e.g., $domain or body.size"
                    value={cond.left}
                    onChange={(e) => {
                      const next = [...rule.when]
                      next[ci] = { ...cond, left: e.target.value }
                      onChange({ ...rule, when: next })
                    }}
                    className="h-11 font-mono text-sm"
                  />
                </div>
                <div className="w-full space-y-1 sm:w-28">
                  <Label className="text-xs">Operator</Label>
                  <Select
                    value={cond.op}
                    onValueChange={(v) => {
                      const next = [...rule.when]
                      next[ci] = { ...cond, op: v as ConditionEntry['op'] }
                      onChange({ ...rule, when: next })
                    }}
                  >
                    <SelectTrigger className="h-11 font-mono text-sm">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {OPERATORS.map((op) => (
                        <SelectItem key={op.value} value={op.value}>
                          {op.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex-1 space-y-1">
                   <Label className="text-xs">Compare against</Label>
                   <Input
                     placeholder={
                       cond.op === 'in' || cond.op === 'not_in'
                         ? 'e.g., admin, owner'
                         : cond.op === 'matches'
                         ? 'e.g., ^api\..*'
                         : 'e.g., api.stripe.com or 1000'
                     }
                     value={cond.right}
                     onChange={(e) => {
                       const next = [...rule.when]
                       next[ci] = { ...cond, right: e.target.value }
                       onChange({ ...rule, when: next })
                     }}
                     className="h-11 font-mono text-sm"
                   />
                 </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-11 w-11 shrink-0 text-destructive hover:bg-destructive/10 hover:text-destructive"
                  onClick={() => {
                    const next = rule.when.filter((_, i) => i !== ci)
                    onChange({ ...rule, when: next })
                  }}
                  aria-label="Remove condition"
                >
                  <X className="h-3.5 w-3.5" />
                </Button>
              </div>
            ))}

            <Collapsible open={extractOpen} onOpenChange={setExtractOpen}>
              <CollapsibleTrigger asChild>
                 <button
                   type="button"
                   aria-expanded={extractOpen}
                   title="Extract specific fields from tool call arguments to use as variables in conditions"
                   className="flex min-h-11 items-center gap-1 rounded px-2 py-2 text-xs text-muted-foreground transition-colors duration-200 hover:bg-muted/50 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                 >
                   {extractOpen ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                   Extract fields for conditions
                  {rule.extract.length > 0 && (
                    <Badge variant="secondary" className="ml-1 h-4 px-1.5 text-xs">
                      {rule.extract.length}
                    </Badge>
                  )}
                </button>
              </CollapsibleTrigger>
              <CollapsibleContent className="space-y-2 pt-2">
                <p className="text-[11px] leading-5 text-muted-foreground">
                  Extract a value from the tool call, name it, then reference it in a condition as <code className="font-mono">$name</code>.
                  Example: <code className="font-mono">$domain</code> from <code className="font-mono">url</code> as <code className="font-mono">URL Host</code>.
                </p>
                {rule.extract.map((ext, ei) => (
                  <div key={ext.id} className="flex flex-col items-stretch gap-2 sm:flex-row sm:items-end">
                    <div className="flex-1 space-y-1">
                      <Label className="text-xs">Variable name</Label>
                      <div className="relative">
                        <span className="absolute left-2.5 top-1/2 -translate-y-1/2 font-mono text-xs text-muted-foreground">$</span>
                        <Input
                          placeholder="e.g., domain"
                          value={ext.name}
                          onChange={(e) => {
                            const next = [...rule.extract]
                            next[ei] = { ...ext, name: e.target.value }
                            onChange({ ...rule, extract: next })
                          }}
                          className="h-11 pl-6 font-mono text-sm"
                        />
                      </div>
                    </div>
                    <div className="flex-1 space-y-1">
                       <Label className="text-xs">Field path (dot notation)</Label>
                       <Input
                         placeholder="e.g., url, body.size, config.host"
                        value={ext.path}
                        onChange={(e) => {
                          const next = [...rule.extract]
                          next[ei] = { ...ext, path: e.target.value }
                          onChange({ ...rule, extract: next })
                        }}
                        className="h-11 font-mono text-sm"
                      />
                    </div>
                    <div className="w-full space-y-1 sm:w-32">
                      <Label className="text-xs">Type</Label>
                      <Select
                        value={ext.type}
                        onValueChange={(v) => {
                          const next = [...rule.extract]
                          next[ei] = { ...ext, type: v as ExtractEntry['type'] }
                          onChange({ ...rule, extract: next })
                        }}
                      >
                        <SelectTrigger className="h-11 text-sm">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {EXTRACT_TYPES.map((t) => (
                            <SelectItem key={t.value} value={t.value}>
                              {t.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="h-11 w-11 shrink-0 text-destructive hover:bg-destructive/10 hover:text-destructive"
                      onClick={() => {
                        const next = rule.extract.filter((_, i) => i !== ei)
                        onChange({ ...rule, extract: next })
                      }}
                      aria-label="Remove extract entry"
                    >
                      <X className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                ))}
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-11 text-xs"
                  onClick={() =>
                    onChange({
                      ...rule,
                      extract: [...rule.extract, { id: generateId(), name: '', path: '', type: 'string' }],
                    })
                  }
                >
                  <Plus className="mr-1 h-3 w-3" />
                  Add extract
                </Button>
              </CollapsibleContent>
            </Collapsible>
          </div>

          <Separator />

          <div className="space-y-3">
            <Label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Effect</Label>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label className="text-xs">Decision</Label>
                <Select
                  value={rule.effect.decision}
                  onValueChange={(v) =>
                    onChange({ ...rule, effect: { ...rule.effect, decision: v as PolicyRule['effect']['decision'] } })
                  }
                >
                  <SelectTrigger className="h-11 text-sm">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {DECISIONS.map((d) => (
                      <SelectItem key={d.value} value={d.value}>
                        {d.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs">Decision reason (optional)</Label>
                <Input
                  placeholder="e.g., Requires manager approval"
                  value={rule.effect.reason}
                  onChange={(e) => onChange({ ...rule, effect: { ...rule.effect, reason: e.target.value } })}
                  className="h-11 text-sm"
                />
              </div>
            </div>
            {catchAllRuleWarning && (
              <p className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
                ⚠ {catchAllRuleWarning}
              </p>
            )}
          </div>
        </CardContent>
      )}
    </Card>
  )
}

export default function PoliciesPage() {
  const { isOwner, isAdmin } = useUserRole()
  const [policies, setPolicies] = useState<PolicySet[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<PolicySet | null>(null)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [togglingPolicyId, setTogglingPolicyId] = useState<string | null>(null)
  const [isDirty, setIsDirty] = useState(false)
  const [expandedRuleId, setExpandedRuleId] = useState<string | null>(null)

  const [editingId, setEditingId] = useState<string | null>(null)
  const [formRules, setFormRules] = useState<PolicyRule[]>([])
  const [discardDialogOpen, setDiscardDialogOpen] = useState(false)
  const policyDialogBodyRef = useRef<HTMLDivElement>(null)
  const lastDialogTriggerRef = useRef<HTMLElement | null>(null)
  const canManagePolicies = isOwner || isAdmin
  const canTogglePolicy = isOwner || isAdmin

  const rememberDialogTrigger = useCallback((element?: HTMLElement | null) => {
    lastDialogTriggerRef.current =
      element ?? (document.activeElement instanceof HTMLElement ? document.activeElement : null)
  }, [])
  const orderedPolicies = useMemo(() => {
    return [...policies].sort((a, b) => {
      const statusDelta = Number(a.status !== 'active') - Number(b.status !== 'active')
      if (statusDelta !== 0) {
        return statusDelta
      }
      return (Date.parse(b.updatedAt) || 0) - (Date.parse(a.updatedAt) || 0)
    })
  }, [policies])
  const activePolicyCount = orderedPolicies.filter((policy) => policy.status === 'active').length
  const archivedPolicyCount = orderedPolicies.length - activePolicyCount

  const fetchPolicies = useCallback(async () => {
    try {
      setLoading(true)
      const res = await api.policies.list()
      const data = res.data?.data || res.data
      setPolicies(data?.policySets || [])
      setLoadError(null)
    } catch (error: unknown) {
      setLoadError(
        getRequestErrorMessage(error, {
        auth: 'Session expired or access revoked. Sign in again.',
          network: 'Could not load policies. Check your connection and retry.',
          fallback: 'Could not load policies right now. Retry to refresh the access policy list.'
        })
      )
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    document.title = 'Policies | Kimbap Console'
  }, [])

  useEffect(() => {
    fetchPolicies()
  }, [fetchPolicies])

  useEffect(() => {
    if (!dialogOpen) {
      return
    }
  
    const frame = window.requestAnimationFrame(() => {
      policyDialogBodyRef.current?.querySelector<HTMLInputElement>('input')?.focus()
    })
  
    return () => window.cancelAnimationFrame(frame)
  }, [dialogOpen])

  const openCreate = (trigger?: HTMLElement | null) => {
    const initialRule = emptyRule()
    rememberDialogTrigger(trigger)
    setEditingId(null)
    setFormRules([initialRule])
    setExpandedRuleId(initialRule.id)
    setIsDirty(false)
    setDialogOpen(true)
  }

  const openEdit = (p: PolicySet, trigger?: HTMLElement | null) => {
    rememberDialogTrigger(trigger)
    setEditingId(p.id)
    const rules = deserializeRules(p.dsl?.rules || [])
    rules.sort((a, b) => a.priority - b.priority)
    setFormRules(rules)
    setExpandedRuleId(rules[0]?.id ?? null)
    setIsDirty(false)
    setDialogOpen(true)
  }

  const handleSave = async () => {
    if (formRules.length === 0) {
      toast.error('Add at least one rule before saving.')
      return
    }
    setSaving(true)
    try {
      const orderedRules = formRules.map((r, i) => ({ ...r, priority: (i + 1) * 100 }))
      const policyTitle = generatePolicyTitle(deserializeRules(serializeRules(orderedRules)))
      if (editingId) {
        await api.policies.update({
          id: editingId,
          dsl: { schemaVersion: 1 as const, rules: serializeRules(orderedRules) },
        })
        toast.success(`Updated policy: ${policyTitle}.`)
      } else {
        await api.policies.create({
          dsl: { schemaVersion: 1 as const, rules: serializeRules(orderedRules) },
        })
        toast.success(`Created policy: ${policyTitle}.`)
      }
      setIsDirty(false)
      setDialogOpen(false)
      fetchPolicies()
    } catch (error: unknown) {
      toast.error(
        getRequestErrorMessage(error, {
        auth: 'Session expired or access revoked. Sign in again.',
          network: 'Could not save this policy. Check your connection and retry.',
          fallback: 'Could not save this policy right now.'
        })
      )
    } finally {
      setSaving(false)
    }
  }

  const handleToggle = async (p: PolicySet) => {
    const nextStatus = p.status === 'active' ? 'archived' : 'active'
    setTogglingPolicyId(p.id)
    try {
      await api.policies.update({ id: p.id, status: nextStatus })
      setPolicies((prev) => prev.map((x) => (x.id === p.id ? { ...x, status: nextStatus } : x)))
      const toggledTitle = generatePolicyTitle(deserializeRules(p.dsl?.rules || []))
      toast.success(`${nextStatus === 'active' ? 'Enabled' : 'Disabled'} policy: ${toggledTitle}.`)
    } catch (error: unknown) {
      toast.error(
        getRequestErrorMessage(error, {
        auth: 'Session expired or access revoked. Sign in again.',
          network: 'Could not change the policy status. Check your connection and retry.',
          fallback: 'Could not update this policy status right now.'
        })
      )
    } finally {
      setTogglingPolicyId(null)
    }
  }

  const confirmDelete = (p: PolicySet) => {
    setDeleteTarget(p)
    setDeleteDialogOpen(true)
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      const deletedTitle = generatePolicyTitle(deserializeRules(deleteTarget.dsl?.rules || []))
      await api.policies.delete(deleteTarget.id)
      toast.success(`Deleted policy: ${deletedTitle}.`)
      setDeleteDialogOpen(false)
      setDeleteTarget(null)
      fetchPolicies()
    } catch (error: unknown) {
      toast.error(
        getRequestErrorMessage(error, {
          auth: 'Session expired or access revoked. Sign in again.',
          network: 'Could not delete this policy. Check your connection and retry.',
          fallback: 'Could not delete this policy right now.'
        })
      )
    } finally {
      setDeleting(false)
    }
  }

  const handleRuleChange = (index: number, updated: PolicyRule) => {
    setIsDirty(true)
    setFormRules((prev) => prev.map((r, i) => (i === index ? updated : r)))
  }

  const handleRuleRemove = (index: number) => {
    setIsDirty(true)
    setFormRules((prev) => {
      const removedRuleId = prev[index]?.id
      const nextRules = prev.filter((_, i) => i !== index)
      setExpandedRuleId((current) => {
        if (!nextRules.length) return null
        if (current && current !== removedRuleId) return current
        return nextRules[Math.min(index, nextRules.length - 1)]?.id ?? null
      })
      return nextRules
    })
  }

  const handleRuleMove = (index: number, direction: 'up' | 'down') => {
    setIsDirty(true)
    setFormRules((prev) => {
      const next = [...prev]
      const targetIndex = direction === 'up' ? index - 1 : index + 1
      if (targetIndex < 0 || targetIndex >= next.length) return prev
      ;[next[index], next[targetIndex]] = [next[targetIndex], next[index]]
      return next
    })
  }

  const tryCloseDialog = (open: boolean) => {
    if (!open && saving) {
      return
    }
    if (!open && isDirty) {
      setDiscardDialogOpen(true)
      return
    }
    if (!open) {
      setDialogOpen(false)
      setExpandedRuleId(null)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <div className="space-y-2">
          <h1 className="flex items-center gap-2 text-[30px] font-bold tracking-tight">
            <Shield className="h-6 w-6" />
            Policies
          </h1>
          <p className="text-sm leading-6 text-muted-foreground">
            Policies decide whether matching tool calls run automatically, require approval, or stay blocked. Rules are checked top to bottom, and the first match wins.
          </p>
          <div className="flex flex-wrap items-center gap-2">
            {DECISIONS.map((d) => (
              <Badge key={d.value} variant="outline" className={d.color}>
                {d.label}
              </Badge>
            ))}
            {orderedPolicies.length > 0 ? (
              <>
                <Badge variant="outline" className="text-xs">{formatDisplayNumber(activePolicyCount)} active</Badge>
                <Badge variant="outline" className="text-xs">{formatDisplayNumber(archivedPolicyCount)} archived</Badge>
              </>
            ) : null}
          </div>
        </div>
        {canManagePolicies ? (
          <Button onClick={(event) => openCreate(event.currentTarget)} className="min-h-11 w-full sm:w-auto">
            <Plus className="mr-2 h-4 w-4" />
            Create Policy
          </Button>
        ) : null}
      </div>

      <Card>
        {orderedPolicies.length > 0 && (
          <CardHeader>
            <CardDescription>
              {formatDisplayNumber(orderedPolicies.length)} {orderedPolicies.length === 1 ? 'policy' : 'policies'} · {formatDisplayNumber(activePolicyCount)} active · {formatDisplayNumber(archivedPolicyCount)} archived
            </CardDescription>
          </CardHeader>
        )}
        <CardContent>
          {loadError ? (
            <div role="alert" className="mb-4 flex flex-col items-start gap-2 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300 sm:flex-row sm:items-center">
              <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span>{loadError}</span>
              <Button variant="outline" size="sm" className="min-h-11 w-full sm:ml-auto sm:w-auto" onClick={fetchPolicies}>Retry</Button>
            </div>
          ) : null}
          {loading ? (
            <div className="flex min-h-[200px] items-center justify-center">
              <div className="text-center">
                <Loader2 className="mx-auto mb-3 h-8 w-8 animate-spin text-muted-foreground" />
                <p className="text-sm text-muted-foreground">Loading access policies…</p>
              </div>
            </div>
          ) : orderedPolicies.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <Shield className="mb-3 h-10 w-10 text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">
                  {loadError ? 'Policies could not be loaded right now.' : canManagePolicies ? 'No policies yet. Create one to define allow, block, and approval rules.' : 'No access policies configured. Contact an administrator to set up tool access policies.'}
              </p>
              {loadError ? (
                <Button variant="outline" className="mt-4 min-h-11" onClick={fetchPolicies}>
                  Retry
                </Button>
              ) : canManagePolicies ? (
                <Button className="mt-4 min-h-11" onClick={(event) => openCreate(event.currentTarget)}>
                  <Plus className="mr-2 h-4 w-4" />
                  Create your first policy
                </Button>
              ) : null}
            </div>
          ) : (
            <>
              <div className="space-y-3 md:hidden">
                {orderedPolicies.map((p) => {
                  const rules = deserializeRules(p.dsl?.rules || [])
                  rules.sort((a, b) => a.priority - b.priority)
                  const title = generatePolicyTitle(rules)
                  const isActive = p.status === 'active'

                  return (
                    <Card key={p.id} className={cn('border border-border/60 shadow-sm', !isActive && 'opacity-70')}>
                      <CardContent className="space-y-4 p-4">
                        <div className="flex items-start justify-between gap-3">
                          <div className="min-w-0 flex-1 space-y-1">
                            {canManagePolicies ? (
                              <button
                                type="button"
                                className="group w-full rounded text-left transition-colors duration-200 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                                onClick={(event) => openEdit(p, event.currentTarget)}
                                aria-label={`Edit policy: ${title}`}
                              >
                                <p className="text-sm font-medium group-hover:underline group-focus-visible:underline">{title}</p>
                              </button>
                            ) : (
                              <p className="text-sm font-medium">{title}</p>
                            )}
                            <p className="text-xs text-muted-foreground">
                              {formatDisplayNumber(rules.length)} rules · v{p.version}
                            </p>
                          </div>
                          <Badge
                            variant="outline"
                            className={isActive
                              ? 'shrink-0 border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950/20 dark:text-emerald-300'
                              : 'shrink-0 border-slate-200 bg-slate-50 text-slate-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300'}
                          >
                            {isActive ? 'Active' : 'Archived'}
                          </Badge>
                        </div>

                        <div className="flex items-center justify-between gap-3 rounded-lg border border-border/60 bg-muted/20 px-3 py-2">
                          <div className="min-w-0">
                            <p className="text-xs text-muted-foreground">Updated</p>
                            <p className="text-sm text-muted-foreground" title={`Created: ${formatDateTime(p.createdAt, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}`}>
                              {formatDateTime(p.updatedAt, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                            </p>
                          </div>
                          <div className="flex items-center gap-2">
                            {togglingPolicyId === p.id ? (
                              <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" aria-hidden="true" />
                            ) : null}
                            <Switch
                              className="h-11 w-[72px] [&>span]:h-8 [&>span]:w-8 data-[state=checked]:[&>span]:translate-x-7"
                              checked={isActive}
                              onCheckedChange={() => handleToggle(p)}
                              disabled={!canTogglePolicy || togglingPolicyId === p.id}
                              aria-label={isActive ? 'Deactivate policy' : 'Activate policy'}
                              title={!canTogglePolicy ? 'Requires admin or owner role to enable or disable policies' : undefined}
                            />
                          </div>
                        </div>

                        <div className="space-y-2">
                          <div className="space-y-1">
                            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Top rules</p>
                            <p className="text-xs text-muted-foreground">Shown in evaluation order. Rule #1 is checked first.</p>
                          </div>
                          {rules.length === 0 ? (
                            <p className="text-sm text-muted-foreground">No rules defined yet. Open this policy to add your first rule.</p>
                          ) : (
                            <div className="space-y-1.5">
                              {rules.slice(0, 2).map((rule, index) => {
                                const decisionMeta = DECISIONS.find((d) => d.value === rule.effect.decision)

                                return (
                                  <div key={rule.id} className="flex flex-wrap items-center gap-2 text-sm">
                                    <Badge variant="secondary" className="px-1.5 py-0 text-[10px] leading-5">
                                      Rule {index + 1}
                                    </Badge>
                                    <Badge
                                      variant="outline"
                                      className={`px-1.5 py-0 text-xs leading-5 ${decisionMeta?.color || ''}`}
                                    >
                                      {decisionMeta?.label || rule.effect.decision}
                                    </Badge>
                                    <span className="font-mono text-xs">{rule.match.tool === '*' || !rule.match.tool ? 'All tools' : rule.match.tool}</span>
                                    {(rule.when?.length ?? 0) > 0 ? (
                                      <span className="text-xs text-muted-foreground">· {rule.when.length} condition{rule.when.length === 1 ? '' : 's'}</span>
                                    ) : null}
                                  </div>
                                )
                              })}
                              {rules.length > 2 ? (
                                <p className="text-xs text-muted-foreground">+ {rules.length - 2} more rules</p>
                              ) : null}
                            </div>
                          )}
                        </div>

                        {canManagePolicies ? (
                          <div className="flex flex-col gap-2 sm:flex-row">
                            <Button
                              variant="outline"
                              className="min-h-11 flex-1"
                              onClick={(event) => openEdit(p, event.currentTarget)}
                              disabled={togglingPolicyId === p.id}
                            >
                              <Pencil className="mr-2 h-4 w-4" />
                              Edit policy
                            </Button>
                            <Button
                              variant="outline"
                              className="min-h-11 flex-1 text-destructive hover:bg-destructive/10 hover:text-destructive"
                              onClick={() => confirmDelete(p)}
                              disabled={togglingPolicyId === p.id}
                            >
                              <Trash2 className="mr-2 h-4 w-4" />
                              Delete policy
                            </Button>
                          </div>
                        ) : null}
                      </CardContent>
                    </Card>
                  )
                })}
              </div>

              <div className="hidden md:block">
                <div className="overflow-x-auto">
                  <Table className="min-w-[720px]">
                    <TableHeader>
                    <TableRow>
                      <TableHead scope="col">Policy</TableHead>
                      <TableHead scope="col">Top rules (checked first)</TableHead>
                      <TableHead scope="col">Updated</TableHead>
                      <TableHead scope="col" className="text-center">Status</TableHead>
                      <TableHead scope="col" className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {orderedPolicies.map((p) => {
                      const rules = deserializeRules(p.dsl?.rules || [])
                      rules.sort((a, b) => a.priority - b.priority)
                      const title = generatePolicyTitle(rules)

                      return (
                        <TableRow key={p.id} className={p.status !== 'active' ? 'opacity-50' : ''}>
                          <TableCell>
                            {canManagePolicies ? (
                              <button
                                type="button"
                                className="group w-full cursor-pointer rounded py-2 text-left space-y-1 transition-colors duration-200 hover:bg-muted/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                                onClick={(event) => openEdit(p, event.currentTarget)}
                                aria-label={`Edit policy: ${title}`}
                              >
                                <p className="text-sm font-medium group-hover:underline group-focus-visible:underline">{title}</p>
                                <p className="text-xs text-muted-foreground">
                                  {formatDisplayNumber(rules.length)} rules · v{p.version}
                                </p>
                              </button>
                            ) : (
                              <div className="space-y-1">
                                <p className="text-sm font-medium">{title}</p>
                                <p className="text-xs text-muted-foreground">
                                  {formatDisplayNumber(rules.length)} rules · v{p.version}
                                </p>
                              </div>
                            )}
                          </TableCell>
                          <TableCell>
                            {rules.length === 0 ? (
                              <p className="text-sm text-muted-foreground">No rules defined yet. Open this policy to add your first rule.</p>
                            ) : (
                              <div className="space-y-1.5">
                                {rules.slice(0, 2).map((rule) => {
                                  const decisionMeta = DECISIONS.find((d) => d.value === rule.effect.decision)
                                   return (
                                     <div key={rule.id} className="flex items-center gap-2 text-sm flex-wrap">
                                        <Badge variant="secondary" className="px-1.5 py-0 text-[10px] leading-5">
                                          Rule {rules.findIndex((candidate) => candidate.id === rule.id) + 1}
                                        </Badge>
                                       <Badge
                                         variant="outline"
                                         className={`px-1.5 py-0 text-xs leading-5 ${decisionMeta?.color || ''}`}
                                       >
                                         {decisionMeta?.label || rule.effect.decision}
                                       </Badge>
                                       <span className="font-mono text-xs">{rule.match.tool === '*' || !rule.match.tool ? 'All tools' : rule.match.tool}</span>
                                       {(rule.when?.length ?? 0) > 0 && (
                                         <span className="text-xs text-muted-foreground">· {rule.when.length} condition{rule.when.length === 1 ? '' : 's'}</span>
                                       )}
                                     </div>
                                   )
                                })}
                                {rules.length > 2 && (
                                  <p className="text-xs text-muted-foreground">+ {rules.length - 2} more rules</p>
                                )}
                              </div>
                            )}
                          </TableCell>
                           <TableCell>
                             <span className="text-sm text-muted-foreground" title={`Created: ${formatDateTime(p.createdAt, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}`}>
                               {formatDateTime(p.updatedAt, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                             </span>
                           </TableCell>
                          <TableCell className="text-center">
                            <div className="flex items-center justify-center gap-2">
                              <Badge
                                variant="outline"
                                className={p.status === 'active'
                                  ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950/20 dark:text-emerald-300'
                                  : 'border-slate-200 bg-slate-50 text-slate-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300'}
                              >
                                {p.status === 'active' ? 'Active' : 'Archived'}
                              </Badge>
                              {togglingPolicyId === p.id ? (
                                <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" aria-hidden="true" />
                              ) : null}
                              <Switch
                                className="h-11 w-[72px] [&>span]:h-8 [&>span]:w-8 data-[state=checked]:[&>span]:translate-x-7"
                                checked={p.status === 'active'}
                                onCheckedChange={() => handleToggle(p)}
                                disabled={!canTogglePolicy || togglingPolicyId === p.id}
                                aria-label={p.status === 'active' ? 'Deactivate policy' : 'Activate policy'}
                                title={!canTogglePolicy ? 'Requires admin or owner role to enable or disable policies' : undefined}
                              />
                            </div>
                          </TableCell>
                          <TableCell className="text-right">
                            {canManagePolicies ? (
                              <div className="flex items-center justify-end gap-1">
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-11 w-11"
                                  onClick={(event) => openEdit(p, event.currentTarget)}
                                  aria-label={`Edit policy: ${title}`}
                                   title="Edit policy"
                                   disabled={togglingPolicyId === p.id}
                                 >
                                   <Pencil className="h-3.5 w-3.5" />
                                 </Button>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-11 w-11 text-destructive hover:bg-destructive/10 hover:text-destructive"
                                    onClick={() => confirmDelete(p)}
                                    aria-label={`Delete policy: ${title}`}
                                    title="Delete policy"
                                    disabled={togglingPolicyId === p.id}
                                 >
                                  <Trash2 className="h-3.5 w-3.5" />
                                </Button>
                              </div>
                            ) : null}
                          </TableCell>
                        </TableRow>
                      )
                    })}
                    </TableBody>
                  </Table>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={tryCloseDialog}>
        <ScrollableDialogContent
          className="max-w-3xl p-0"
          onCloseAutoFocus={(event) => {
            event.preventDefault()
            lastDialogTriggerRef.current?.focus()
          }}
        >
          <div className="border-b px-6 pb-4 pt-6">
            <DialogHeader>
              <DialogTitle>{editingId ? 'Edit access policy' : 'Create access policy'}</DialogTitle>
              <DialogDescription>
                {editingId
                  ? 'Update the rules for this access policy.'
                  : 'Set up rules to control how tool calls are handled.'}
              </DialogDescription>
            </DialogHeader>
          </div>

          <div ref={policyDialogBodyRef} className="px-6 py-5 pb-28 sm:pb-24">
            <div className="space-y-5">
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div>
                    <Label className="text-sm font-semibold">Rules{formRules.length > 0 ? ` (${formatDisplayNumber(formRules.length)})` : ''}</Label>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      {formRules.length === 0
                        ? 'Add rules to define how tool calls are handled.'
                        : 'Rules are evaluated in order. The first matching rule applies.'}
                    </p>
                    {formRules.length > 1 ? (
                      <p className="mt-1 text-xs text-muted-foreground">Only the first rule starts expanded to keep long policy sets scannable.</p>
                    ) : null}
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    className="min-h-11"
                    onClick={() => {
                      const nextRule = emptyRule()
                      setFormRules((prev) => [...prev, nextRule])
                      setExpandedRuleId(nextRule.id)
                      setIsDirty(true)
                    }}
                  >
                    <Plus className="mr-1 h-3.5 w-3.5" />
                    Add Rule
                  </Button>
                </div>

                {formRules.length === 0 && (
                  <div className="flex flex-col items-center rounded-lg border border-dashed py-8 text-center">
                    <Shield className="mb-2 h-8 w-8 text-muted-foreground/30" />
                    <p className="text-sm text-muted-foreground">No rules added yet</p>
                    <Button
                      variant="link"
                      size="sm"
                      className="mt-1 min-h-11"
                      onClick={() => {
                        const nextRule = emptyRule()
                        setFormRules((prev) => [...prev, nextRule])
                        setExpandedRuleId(nextRule.id)
                        setIsDirty(true)
                      }}
                    >
                      + Add your first rule
                    </Button>
                  </div>
                )}

                {formRules.length > 0 && (
                  <div className="space-y-3">
                    {formRules.map((rule, i) => (
                      <RuleCard
                        key={rule.id}
                        rule={rule}
                        index={i}
                        expanded={expandedRuleId === rule.id}
                        canMoveUp={i > 0}
                        canMoveDown={i < formRules.length - 1}
                        onToggle={() => setExpandedRuleId((prev) => (prev === rule.id ? null : rule.id))}
                        onChange={(updated) => handleRuleChange(i, updated)}
                        onMove={(direction) => handleRuleMove(i, direction)}
                        onRemove={() => handleRuleRemove(i)}
                      />
                    ))}
                    <Button
                      variant="outline"
                      size="sm"
                      className="min-h-11 w-full"
                      onClick={() => {
                        const nextRule = emptyRule()
                        setFormRules((prev) => [...prev, nextRule])
                        setExpandedRuleId(nextRule.id)
                        setIsDirty(true)
                      }}
                    >
                      <Plus className="mr-1 h-3.5 w-3.5" />
                      Add Rule
                    </Button>
                  </div>
                )}
              </div>
            </div>
          </div>

          <div className="sticky bottom-0 border-t bg-background/95 px-6 py-4 pb-[max(1rem,env(safe-area-inset-bottom))] backdrop-blur supports-[backdrop-filter]:bg-background/80">
            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <Button variant="outline" className="min-h-11 w-full sm:w-auto" onClick={() => tryCloseDialog(false)} disabled={saving}>
                Cancel
              </Button>
              <Button className="min-h-11 w-full sm:w-auto" onClick={handleSave} disabled={saving || formRules.length === 0}>
                {saving ? (
                  <><Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />{editingId ? 'Save Changes' : 'Create Policy'}</>
                ) : editingId ? 'Save Changes' : 'Create Policy'}
              </Button>
            </div>
          </div>
        </ScrollableDialogContent>
      </Dialog>

      <AlertDialog open={deleteDialogOpen} onOpenChange={(open) => { if (!deleting) setDeleteDialogOpen(open) }}>
        <AlertDialogContent className="max-w-md">
          <AlertDialogHeader>
             <AlertDialogTitle>Delete policy</AlertDialogTitle>
              <AlertDialogDescription asChild>
                <div>
                  {deleteTarget && (
                    <span>
                      <span className="font-medium text-foreground">
                        {generatePolicyTitle(deserializeRules(deleteTarget.dsl?.rules || []))}
                      </span>
                      {deleteTarget.status === 'active' && (
                        <span className="ml-1.5 inline-flex items-center rounded-full border border-emerald-200 bg-emerald-50 px-1.5 py-0.5 text-[11px] font-medium text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950/20 dark:text-emerald-300">Active</span>
                      )}
                    </span>
                  )}
                  {deleteTarget ? ' will be permanently deleted. This action cannot be undone.' : 'Are you sure you want to delete this policy? This action cannot be undone.'}
                  {deleteTarget?.status === 'active' && (
                    <p className="mt-2 text-amber-700 dark:text-amber-400 text-xs">This policy is currently active and enforcing rules. Deleting it will immediately stop enforcement.</p>
                  )}
                </div>
              </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="min-h-11" disabled={deleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="min-h-11 bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={deleting}
              onClick={(e) => {
                e.preventDefault()
                handleDelete()
              }}
            >
              {deleting ? (
                <span className="inline-flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                  Delete
                </span>
              ) : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={discardDialogOpen} onOpenChange={setDiscardDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Discard changes?</AlertDialogTitle>
            <AlertDialogDescription>
              Your unsaved changes will be lost. Continue editing or discard them.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="min-h-11" disabled={saving}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="min-h-11 bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={saving}
              onClick={() => {
                setDiscardDialogOpen(false)
                setDialogOpen(false)
                setIsDirty(false)
              }}
            >
              Discard
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
