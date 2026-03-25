'use client'

import { useState, useEffect, useCallback } from 'react'
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
  { value: 'ALLOW', label: 'Allow', color: 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20' },
  {
    value: 'REQUIRE_APPROVAL',
    label: 'Needs approval',
    color: 'bg-amber-500/10 text-amber-600 border-amber-500/20',
  },
  { value: 'DENY', label: 'Block', color: 'bg-red-500/10 text-red-600 border-red-500/20' },
] as const

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
        extractEntries[e.name] = { path: e.path, type: e.type }
      })
      rule.extract = extractEntries
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
  if (summary.length > 80) summary = `${summary.slice(0, 77)}...`
  return summary
}

function generatePolicyTitle(rules: PolicyRule[]): string {
  if (!rules || rules.length === 0) return 'Empty policy'
  const first = rules[0]
  const decision = DECISIONS.find((d) => d.value === first.effect.decision)?.label || first.effect.decision
  const tool = first.match.tool === '*' || !first.match.tool ? 'All tools' : first.match.tool
  const serverPart = first.match.serverId ? ` on ${first.match.serverId}` : ''
  return `${tool}${serverPart} — ${decision}`
}

function RuleCard({
  rule,
  index,
  canMoveUp,
  canMoveDown,
  onChange,
  onMove,
  onRemove,
}: {
  rule: PolicyRule
  index: number
  canMoveUp: boolean
  canMoveDown: boolean
  onChange: (updated: PolicyRule) => void
  onMove: (direction: 'up' | 'down') => void
  onRemove: () => void
}) {
  const [expanded, setExpanded] = useState(false)
  const [extractOpen, setExtractOpen] = useState(rule.extract.length > 0)
  const decisionMeta = DECISIONS.find((d) => d.value === rule.effect.decision)

  return (
    <Card className="border border-border/60 shadow-sm">
      <div className="flex items-center gap-2 rounded-t-lg bg-muted/30 px-4 py-3">
        <button
          type="button"
          onClick={() => setExpanded((prev) => !prev)}
          className="flex min-w-0 flex-1 items-center gap-2 text-left"
        >
          {expanded ? <ChevronUp className="h-4 w-4 shrink-0" /> : <ChevronDown className="h-4 w-4 shrink-0" />}
          <span className="shrink-0 text-sm font-medium">Rule {index + 1}</span>
          <Badge variant="outline" className={`shrink-0 text-xs ${decisionMeta?.color || ''}`}>
            {decisionMeta?.label || rule.effect.decision}
          </Badge>
          <span className="truncate text-xs text-muted-foreground">{generateRuleSummary(rule)}</span>
        </button>

        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => onMove('up')}
            disabled={!canMoveUp}
            aria-label="Move rule up"
          >
            <ArrowUp className="h-3.5 w-3.5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => onMove('down')}
            disabled={!canMoveDown}
            aria-label="Move rule down"
          >
            <ArrowDown className="h-3.5 w-3.5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7 text-destructive hover:bg-destructive/10 hover:text-destructive"
            onClick={onRemove}
            aria-label="Remove rule"
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
                className="h-9"
              />
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
                className="h-7 text-xs"
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

            {rule.when.length === 0 && rule.extract.length === 0 && (
              <p className="text-xs italic text-muted-foreground">No conditions — applies to all matching tool calls.</p>
            )}

            {rule.when.map((cond, ci) => (
              <div key={cond.id} className="flex items-end gap-2">
                <div className="flex-1 space-y-1">
                  <Label className="text-xs">Field</Label>
                  <Input
                    placeholder="$varName or literal"
                    value={cond.left}
                    onChange={(e) => {
                      const next = [...rule.when]
                      next[ci] = { ...cond, left: e.target.value }
                      onChange({ ...rule, when: next })
                    }}
                    className="h-9 font-mono text-sm"
                  />
                </div>
                <div className="w-28 space-y-1">
                  <Label className="text-xs">Check</Label>
                  <Select
                    value={cond.op}
                    onValueChange={(v) => {
                      const next = [...rule.when]
                      next[ci] = { ...cond, op: v as ConditionEntry['op'] }
                      onChange({ ...rule, when: next })
                    }}
                  >
                    <SelectTrigger className="h-9 font-mono text-sm">
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
                  <Label className="text-xs">Value</Label>
                  <Input
                    placeholder="Expected value"
                    value={cond.right}
                    onChange={(e) => {
                      const next = [...rule.when]
                      next[ci] = { ...cond, right: e.target.value }
                      onChange({ ...rule, when: next })
                    }}
                    className="h-9 font-mono text-sm"
                  />
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-9 w-9 shrink-0 text-destructive hover:bg-destructive/10 hover:text-destructive"
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
                  className="flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
                >
                  {extractOpen ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                  Data extraction (advanced)
                  {rule.extract.length > 0 && (
                    <Badge variant="secondary" className="ml-1 h-4 px-1.5 text-[10px]">
                      {rule.extract.length}
                    </Badge>
                  )}
                </button>
              </CollapsibleTrigger>
              <CollapsibleContent className="space-y-2 pt-2">
                <p className="text-[11px] text-muted-foreground">Extract fields from the tool call payload to use in conditions.</p>
                {rule.extract.map((ext, ei) => (
                  <div key={ext.id} className="flex items-end gap-2">
                    <div className="flex-1 space-y-1">
                      <Label className="text-xs">Label</Label>
                      <div className="relative">
                        <span className="absolute left-2.5 top-1/2 -translate-y-1/2 font-mono text-xs text-muted-foreground">$</span>
                        <Input
                          placeholder="varName"
                          value={ext.name}
                          onChange={(e) => {
                            const next = [...rule.extract]
                            next[ei] = { ...ext, name: e.target.value }
                            onChange({ ...rule, extract: next })
                          }}
                          className="h-9 pl-6 font-mono text-sm"
                        />
                      </div>
                    </div>
                    <div className="flex-1 space-y-1">
                      <Label className="text-xs">Path</Label>
                      <Input
                        placeholder="e.g., url, config.host"
                        value={ext.path}
                        onChange={(e) => {
                          const next = [...rule.extract]
                          next[ei] = { ...ext, path: e.target.value }
                          onChange({ ...rule, extract: next })
                        }}
                        className="h-9 font-mono text-sm"
                      />
                    </div>
                    <div className="w-32 space-y-1">
                      <Label className="text-xs">Type</Label>
                      <Select
                        value={ext.type}
                        onValueChange={(v) => {
                          const next = [...rule.extract]
                          next[ei] = { ...ext, type: v as ExtractEntry['type'] }
                          onChange({ ...rule, extract: next })
                        }}
                      >
                        <SelectTrigger className="h-9 text-sm">
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
                      className="h-9 w-9 shrink-0 text-destructive hover:bg-destructive/10 hover:text-destructive"
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
                  className="h-7 text-xs"
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
            <Label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Action</Label>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label className="text-xs">Decision</Label>
                <Select
                  value={rule.effect.decision}
                  onValueChange={(v) =>
                    onChange({ ...rule, effect: { ...rule.effect, decision: v as PolicyRule['effect']['decision'] } })
                  }
                >
                  <SelectTrigger className="h-9 text-sm">
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
                <Label className="text-xs">Reason (optional)</Label>
                <Input
                  placeholder="e.g., Requires manager approval"
                  value={rule.effect.reason}
                  onChange={(e) => onChange({ ...rule, effect: { ...rule.effect, reason: e.target.value } })}
                  className="h-9 text-sm"
                />
              </div>
            </div>
          </div>
        </CardContent>
      )}
    </Card>
  )
}

export default function PoliciesPage() {
  const [policies, setPolicies] = useState<PolicySet[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<PolicySet | null>(null)
  const [saving, setSaving] = useState(false)
  const [isDirty, setIsDirty] = useState(false)

  const [editingId, setEditingId] = useState<string | null>(null)
  const [formRules, setFormRules] = useState<PolicyRule[]>([])
  const [discardDialogOpen, setDiscardDialogOpen] = useState(false)

  const fetchPolicies = useCallback(async () => {
    try {
      setLoading(true)
      const res = await api.policies.list()
      const data = res.data?.data || res.data
      setPolicies(data?.policySets || [])
    } catch {
      toast.error('Could not load policies')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchPolicies()
  }, [fetchPolicies])

  const openCreate = () => {
    setEditingId(null)
    setFormRules([emptyRule()])
    setIsDirty(false)
    setDialogOpen(true)
  }

  const openEdit = (p: PolicySet) => {
    setEditingId(p.id)
    const rules = deserializeRules(p.dsl?.rules || [])
    rules.sort((a, b) => a.priority - b.priority)
    setFormRules(rules)
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
      if (editingId) {
        await api.policies.update({
          id: editingId,
          dsl: { schemaVersion: 1 as const, rules: serializeRules(orderedRules) },
        })
        toast.success('Policy updated')
      } else {
        await api.policies.create({
          dsl: { schemaVersion: 1 as const, rules: serializeRules(orderedRules) },
        })
        toast.success('Policy created')
      }
      setIsDirty(false)
      setDialogOpen(false)
      fetchPolicies()
    } catch {
      toast.error('Could not save policy')
    } finally {
      setSaving(false)
    }
  }

  const handleToggle = async (p: PolicySet) => {
    const nextStatus = p.status === 'active' ? 'archived' : 'active'
    try {
      await api.policies.update({ id: p.id, status: nextStatus })
      setPolicies((prev) => prev.map((x) => (x.id === p.id ? { ...x, status: nextStatus } : x)))
      toast.success(`Policy ${nextStatus === 'active' ? 'enabled' : 'disabled'}`)
    } catch {
      toast.error('Could not update policy status')
    }
  }

  const confirmDelete = (p: PolicySet) => {
    setDeleteTarget(p)
    setDeleteDialogOpen(true)
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await api.policies.delete(deleteTarget.id)
      toast.success('Policy deleted')
      setDeleteDialogOpen(false)
      setDeleteTarget(null)
      fetchPolicies()
    } catch {
      toast.error('Could not delete policy')
    }
  }

  const handleRuleChange = (index: number, updated: PolicyRule) => {
    setIsDirty(true)
    setFormRules((prev) => prev.map((r, i) => (i === index ? updated : r)))
  }

  const handleRuleRemove = (index: number) => {
    setIsDirty(true)
    setFormRules((prev) => prev.filter((_, i) => i !== index))
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
    if (!open && isDirty) {
      setDiscardDialogOpen(true)
      return
    }
    if (!open) setDialogOpen(false)
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div className="space-y-2">
          <h1 className="flex items-center gap-2 text-[30px] font-bold tracking-tight">
            <Shield className="h-6 w-6" />
            Tool Access Policies
          </h1>
          <p className="text-base text-muted-foreground">
            Policies decide which tool calls run automatically, which are blocked, and which need approval.
          </p>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="outline" className="bg-emerald-500/10 text-emerald-600 border-emerald-500/20">
              Allow
            </Badge>
            <Badge variant="outline" className="bg-amber-500/10 text-amber-600 border-amber-500/20">
              Needs approval
            </Badge>
            <Badge variant="outline" className="bg-red-500/10 text-red-600 border-red-500/20">
              Block
            </Badge>
          </div>
        </div>
        <Button onClick={openCreate}>
          <Plus className="mr-2 h-4 w-4" />
          Create Policy
        </Button>
      </div>

      <Card>
        {policies.length > 0 && (
          <CardHeader>
            <CardDescription>
              {policies.length} {policies.length === 1 ? 'policy' : 'policies'}
            </CardDescription>
          </CardHeader>
        )}
        <CardContent>
          {loading ? (
            <div className="flex min-h-[200px] items-center justify-center">
              <div className="text-center">
                <Loader2 className="mx-auto mb-3 h-8 w-8 animate-spin text-muted-foreground" />
                <p className="text-sm text-muted-foreground">Loading policies...</p>
              </div>
            </div>
          ) : policies.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <Shield className="mb-3 h-10 w-10 text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">No access policies yet</p>
              <Button variant="outline" className="mt-4" onClick={openCreate}>
                <Plus className="mr-2 h-4 w-4" />
                Create your first policy
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Policy</TableHead>
                  <TableHead>What happens</TableHead>
                  <TableHead>Updated</TableHead>
                  <TableHead className="text-center">Status</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {policies.map((p) => {
                  const rules = deserializeRules(p.dsl?.rules || [])
                  rules.sort((a, b) => a.priority - b.priority)
                  const title = generatePolicyTitle(rules)

                  return (
                    <TableRow key={p.id}>
                      <TableCell>
                        <button
                          type="button"
                          className="text-left space-y-1 w-full rounded cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 group"
                          onClick={() => openEdit(p)}
                          aria-label={`Edit policy: ${title}`}
                        >
                          <p className="text-sm font-medium group-hover:underline group-focus-visible:underline">{title}</p>
                          <p className="text-xs text-muted-foreground">
                            {rules.length} rules · v{p.version}
                          </p>
                        </button>
                      </TableCell>
                      <TableCell>
                        {rules.length === 0 ? (
                          <p className="text-sm text-muted-foreground">No rules defined</p>
                        ) : (
                          <div className="space-y-1.5">
                            {rules.slice(0, 2).map((rule) => {
                              const decisionMeta = DECISIONS.find((d) => d.value === rule.effect.decision)
                              return (
                                <div key={rule.id} className="flex items-center gap-2 text-sm">
                                  <Badge
                                    variant="outline"
                                    className={`px-1.5 py-0 text-[10px] leading-5 ${decisionMeta?.color || ''}`}
                                  >
                                    {decisionMeta?.label || rule.effect.decision}
                                  </Badge>
                                  <span className="font-mono text-xs">{rule.match.tool || '*'}</span>
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
                        <span className="text-sm text-muted-foreground">
                          {new Date(p.updatedAt).toLocaleDateString()}
                        </span>
                      </TableCell>
                      <TableCell className="text-center">
                        <Switch
                          checked={p.status === 'active'}
                          onCheckedChange={() => handleToggle(p)}
                          aria-label={p.status === 'active' ? 'Deactivate policy' : 'Activate policy'}
                        />
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8"
                            onClick={() => openEdit(p)}
                            aria-label="Edit policy"
                          >
                            <Pencil className="h-3.5 w-3.5" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 text-destructive hover:bg-destructive/10 hover:text-destructive"
                            onClick={() => confirmDelete(p)}
                            aria-label="Delete policy"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={tryCloseDialog}>
        <ScrollableDialogContent className="max-w-2xl p-0">
          <div className="border-b px-6 pb-4 pt-6">
            <DialogHeader>
              <DialogTitle>{editingId ? 'Edit Access Policy' : 'Create Access Policy'}</DialogTitle>
              <DialogDescription>
                {editingId
                  ? 'Update the rules for this access policy.'
                  : 'Set up rules to control how tool calls are handled.'}
              </DialogDescription>
            </DialogHeader>
          </div>

          <div className="px-6 py-5">
            <div className="space-y-5">
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div>
                    <Label className="text-sm font-semibold">Rules{formRules.length > 0 ? ` (${formRules.length})` : ''}</Label>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      {formRules.length === 0
                        ? 'Add rules to define how tool calls are handled.'
                        : 'Rules are evaluated in order. The first matching rule applies.'}
                    </p>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      setFormRules((prev) => [...prev, emptyRule()])
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
                    <p className="text-sm text-muted-foreground">No rules yet</p>
                    <Button
                      variant="link"
                      size="sm"
                      className="mt-1"
                      onClick={() => {
                        setFormRules((prev) => [...prev, emptyRule()])
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
                        canMoveUp={i > 0}
                        canMoveDown={i < formRules.length - 1}
                        onChange={(updated) => handleRuleChange(i, updated)}
                        onMove={(direction) => handleRuleMove(i, direction)}
                        onRemove={() => handleRuleRemove(i)}
                      />
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>

          <div className="border-t px-6 py-4">
            <div className="flex items-center justify-end gap-2">
              <Button variant="outline" onClick={() => tryCloseDialog(false)}>
                Cancel
              </Button>
              <Button onClick={handleSave} disabled={saving || formRules.length === 0}>
                {saving ? 'Saving...' : editingId ? 'Save Changes' : 'Create Policy'}
              </Button>
            </div>
          </div>
        </ScrollableDialogContent>
      </Dialog>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent className="max-w-md">
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Policy</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this policy? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={(e) => {
                e.preventDefault()
                handleDelete()
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={discardDialogOpen} onOpenChange={setDiscardDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Discard changes?</AlertDialogTitle>
            <AlertDialogDescription>
              Your unsaved changes will be lost.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
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
