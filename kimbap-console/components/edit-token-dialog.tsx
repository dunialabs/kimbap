'use client'

import {
  Edit,
  ChevronDown,
  Globe,
  Github,
  Mail,
  Database,
  Loader2,
  AlertTriangle
} from 'lucide-react'
import { useState } from 'react'

import { TagInput } from '@/components/tag-input'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Collapsible, CollapsibleContent } from '@/components/ui/collapsible'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  ScrollableDialogContent,
  DialogTitle,
  DialogTrigger
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

interface Tool {
  toolId: string
  name: string
  icon: any
  enabled: boolean
  toolFuncs: Array<{
    funcName: string
    enabled: boolean
  }>
}

interface Token {
  id: string
  name: string
  purpose?: string
  tools: Tool[]
  namespace?: string
  tags?: string[]
}

interface EditTokenDialogProps {
  token: Token
  onSave: (updatedToken: Token) => Promise<void>
  allTags?: string[]
}

const availableTools = [
  {
    toolId: 'web-search',
    name: 'Web Search',
    icon: Globe,
    enabled: true,
    toolFuncs: [
      { funcName: 'Google Search', enabled: true },
      { funcName: 'Bing Search', enabled: false },
      { funcName: 'DuckDuckGo', enabled: true }
    ]
  },
  {
    toolId: 'github',
    name: 'GitHub',
    icon: Github,
    enabled: true,
    toolFuncs: [
      { funcName: 'Repository Access', enabled: true },
      { funcName: 'Issue Management', enabled: true },
      { funcName: 'Pull Requests', enabled: false },
      { funcName: 'Webhooks', enabled: false }
    ]
  },
  {
    toolId: 'email',
    name: 'Email',
    icon: Mail,
    enabled: false,
    toolFuncs: [
      { funcName: 'Send Email', enabled: false },
      { funcName: 'Read Email', enabled: false },
      { funcName: 'Manage Folders', enabled: false }
    ]
  },
  {
    toolId: 'database',
    name: 'Database',
    icon: Database,
    enabled: true,
    toolFuncs: [
      { funcName: 'Read Operations', enabled: true },
      { funcName: 'Write Operations', enabled: false },
      { funcName: 'Schema Management', enabled: true },
      { funcName: 'Backup & Restore', enabled: false }
    ]
  }
]

export function EditTokenDialog({ token, onSave, allTags }: EditTokenDialogProps) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState(token.name)
  const [purpose, setPurpose] = useState(token.purpose || '')
  const [namespace, setNamespace] = useState(token.namespace || 'default')
  const [tags, setTags] = useState<string[]>(token.tags || [])
  const [tools, setTools] = useState<Tool[]>(token.tools || availableTools)
  const [expandedTools, setExpandedTools] = useState<Set<string>>(new Set())
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState('')
  const toggleToolExpansion = (toolId: string) => {
    const newExpanded = new Set(expandedTools)
    if (newExpanded.has(toolId)) {
      newExpanded.delete(toolId)
    } else {
      newExpanded.add(toolId)
    }
    setExpandedTools(newExpanded)
  }

  const handleToolToggle = (toolId: string, enabled: boolean) => {
    setTools((prev) =>
      prev.map((tool) => (tool.toolId === toolId ? { ...tool, enabled } : tool))
    )
  }

  const handleSubFunctionToggle = (
    toolId: string,
    funcName: string,
    enabled: boolean
  ) => {
    setTools((prev) =>
      prev.map((tool) =>
        tool.toolId === toolId
          ? {
              ...tool,
              toolFuncs: (tool.toolFuncs || []).map((func) =>
                func.funcName === funcName
                  ? { ...func, enabled }
                  : func
              )
            }
          : tool
      )
    )
  }

  const handleSave = async () => {
    setIsSaving(true)
    setError('')

    try {
      const updatedToken = {
        ...token,
        name,
        purpose,
        tools,
        namespace,
        tags
      }
      await onSave(updatedToken)
      setOpen(false)
    } catch (err: any) {
      setError(err.message || 'Could not save changes')
    } finally {
      setIsSaving(false)
    }
  }

  const handleCancel = () => {
    setName(token.name)
    setPurpose(token.purpose || '')
    setNamespace(token.namespace || 'default')
    setTags(token.tags || [])
    setTools(token.tools || availableTools)
    setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm" className="text-xs bg-transparent">
          <Edit className="h-3 w-3" />
        </Button>
      </DialogTrigger>
      <ScrollableDialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Edit Access Token</DialogTitle>
          <DialogDescription>
            Update token permissions and tool access.
          </DialogDescription>
        </DialogHeader>

        <form
          onSubmit={(e) => {
            e.preventDefault()
            handleSave()
          }}
        >
          <div>
          <div className="space-y-6">
          {/* Basic Information */}
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="edit-token-name">
                Label <span className="text-red-500">*</span>
              </Label>
              <Input
                id="edit-token-name"
                placeholder="Production API"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                autoFocus
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-token-notes">Notes</Label>
              <Textarea
                id="edit-token-notes"
                placeholder="Used for CI/CD pipeline"
                value={purpose}
                onChange={(e) => setPurpose(e.target.value)}
                rows={2}
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="edit-token-namespace">Namespace</Label>
                <Input
                  id="edit-token-namespace"
                  placeholder="default"
                  value={namespace}
                  onChange={(e) => setNamespace(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label>Tags</Label>
                <TagInput
                  value={tags}
                  onChange={setTags}
                  suggestions={allTags || []}
                  placeholder="Add tags..."
                />
              </div>
            </div>
          </div>

          {/* Tool Permissions */}
          <div className="space-y-4">
            <div>
              <h3 className="text-lg font-medium">Tool Permissions</h3>
              <p className="text-sm text-muted-foreground">
                Select which tools this token can access and configure specific
                functions.
              </p>
            </div>

            <div className="space-y-3">
              {tools.map((tool) => (
                <Card key={tool.toolId} className="bg-muted/20">
                  <CardHeader className="pb-3">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        {/* <tool.icon className="h-5 w-5" /> */}
                        <CardTitle className="text-base">{tool.name}</CardTitle>
                      </div>
                      <div className="flex items-center gap-2">
                        <Switch
                          checked={tool.enabled}
                          onCheckedChange={(checked) =>
                            handleToolToggle(tool.toolId, checked)
                          }
                        />
                        {tool.toolFuncs?.length > 0 && (
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            className="h-6 w-6 p-0"
                            onClick={() => toggleToolExpansion(tool.toolId)}
                          >
                            <ChevronDown
                              className={cn(
                                'h-4 w-4 transition-transform',
                                expandedTools.has(tool.toolId) && 'rotate-180'
                              )}
                            />
                          </Button>
                        )}
                      </div>
                    </div>
                  </CardHeader>

                  <Collapsible open={expandedTools.has(tool.toolId)}>
                    <CollapsibleContent>
                      <CardContent className="pt-0">
                        <div className="space-y-2">
                          <p className="text-sm font-medium text-muted-foreground">
                            Functions:
                          </p>
                          {(tool.toolFuncs || []).map((subFunc) => (
                            <div
                              key={`${tool.toolId}-${subFunc.funcName}`}
                              className="flex items-center justify-between py-1"
                            >
                              <span className="text-sm">{subFunc.funcName}</span>
                              <Switch
                                checked={subFunc.enabled}
                                onCheckedChange={(checked) =>
                                  handleSubFunctionToggle(
                                    tool.toolId,
                                    subFunc.funcName,
                                    checked
                                  )
                                }
                                disabled={!tool.enabled}
                              />
                            </div>
                          ))}
                        </div>
                      </CardContent>
                    </CollapsibleContent>
                  </Collapsible>
                </Card>
              ))}
            </div>
          </div>

          {/* Error Display */}
          {error && (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          </div>

          </div>
          <DialogFooter className="border-t pt-4">
            <Button type="button" variant="outline" onClick={handleCancel} disabled={isSaving}>
              Cancel
            </Button>
            <Button type="submit" disabled={!name.trim() || isSaving}>
              {isSaving ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Saving...
                </>
              ) : (
                'Save Changes'
              )}
            </Button>
          </DialogFooter>
        </form>
      </ScrollableDialogContent>
    </Dialog>
  )
}
