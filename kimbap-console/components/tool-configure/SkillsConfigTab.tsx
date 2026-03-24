'use client'

import React, { useState, useCallback, useEffect } from 'react'
import {
  Upload,
  Search,
  Folder,
  MoreHorizontal,
  Trash2,
  Loader2,
  Info,
  ArrowUpDown
} from 'lucide-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle
} from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/components/ui/alert-dialog'
import { cn } from '@/lib/utils'
import { SKILLS_MAX_FILE_SIZE } from '@/lib/mcp-constants'
import { LazyStartConfiguration } from './LazyStartConfiguration'
import { PublicAccessConfiguration } from './PublicAccessConfiguration'
import { AnonymousAccessConfiguration } from './AnonymousAccessConfiguration'

interface SkillInfo {
  name: string
  description: string
  version: string
  updatedAt?: string
}

interface SkillsConfigTabProps {
  serverId: string
  serverName?: string
  lazyStartEnabled: boolean
  onLazyStartEnabledChange: (value: boolean) => void
  publicAccess: boolean
  onPublicAccessChange: (value: boolean) => void
  anonymousAccess: boolean
  onAnonymousAccessChange: (value: boolean) => void
  anonymousRateLimit: number
  onAnonymousRateLimitChange: (value: number) => void
  onDelete?: () => void
  allowUserInput: boolean
}

export const SkillsConfigTab = React.memo<SkillsConfigTabProps>(
  ({
    serverId,
    serverName,
    lazyStartEnabled,
    onLazyStartEnabledChange,
    publicAccess,
    onPublicAccessChange,
    anonymousAccess,
    onAnonymousAccessChange,
    anonymousRateLimit,
    onAnonymousRateLimitChange,
    onDelete,
    allowUserInput,
  }) => {
    const [skills, setSkills] = useState<SkillInfo[]>([])
    const [loading, setLoading] = useState(false)
    const [uploading, setUploading] = useState(false)
    const [searchQuery, setSearchQuery] = useState('')
    const [sortBy, setSortBy] = useState<'time-desc' | 'time-asc' | 'name-asc' | 'name-desc'>('time-desc')
    const [dragActive, setDragActive] = useState(false)
    const [skillToDelete, setSkillToDelete] = useState<string | null>(null)
    const [deleting, setDeleting] = useState(false)
    const [showDeleteAllDialog, setShowDeleteAllDialog] = useState(false)
    const [deletingAll, setDeletingAll] = useState(false)

    // Load skills list
    const loadSkills = useCallback(async () => {
      if (!serverId) return

      try {
        setLoading(true)
        const { api } = await import('@/lib/api-client')
        const response = await api.tools.listSkills({ serverId })

        if (response.data?.data?.skills) {
          setSkills(response.data.data.skills)
        }
      } catch (error) {
        // Error already shown via toast below
        toast.error('Could not load skills')
      } finally {
        setLoading(false)
      }
    }, [serverId])

    useEffect(() => {
      loadSkills()
    }, [loadSkills])

    // Handle file upload
    const handleFileUpload = useCallback(
      async (file: File) => {
        if (!file.name.toLowerCase().endsWith('.zip')) {
          toast.error('Please upload a ZIP file')
          return
        }

        if (file.size > SKILLS_MAX_FILE_SIZE) {
          toast.error(`File size exceeds the maximum limit of ${SKILLS_MAX_FILE_SIZE / 1024 / 1024}MB`)
          return
        }

        try {
          setUploading(true)
          const zipBuffer = await file.arrayBuffer()
          const { api } = await import('@/lib/api-client')

          // Convert to base64 (much more efficient than JSON number array)
          const uint8Array = new Uint8Array(zipBuffer)
          let binary = ''
          for (let i = 0; i < uint8Array.length; i++) {
            binary += String.fromCharCode(uint8Array[i])
          }
          const base64Data = btoa(binary)

          const response = await api.tools.uploadSkills({
            serverId,
            data: base64Data
          })

          if (response.data?.common?.code === 0) {
            toast.success('Skills uploaded')
            await loadSkills()
          } else {
            toast.error(
              response.data?.common?.message || 'Could not upload skills'
            )
          }
        } catch (error: any) {
          // Error already shown via toast below
          toast.error(error.message || 'Could not upload skills')
        } finally {
          setUploading(false)
        }
      },
      [serverId, loadSkills]
    )

    // Handle drag and drop
    const handleDragEnter = useCallback((e: React.DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      setDragActive(true)
    }, [])

    const handleDragLeave = useCallback((e: React.DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
      setDragActive(false)
    }, [])

    const handleDragOver = useCallback((e: React.DragEvent) => {
      e.preventDefault()
      e.stopPropagation()
    }, [])

    const handleDrop = useCallback(
      (e: React.DragEvent) => {
        e.preventDefault()
        e.stopPropagation()
        setDragActive(false)

        const file = e.dataTransfer.files?.[0]
        if (file) {
          handleFileUpload(file)
        }
      },
      [handleFileUpload]
    )

    // Handle file input change
    const handleFileInputChange = useCallback(
      (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0]
        if (file) {
          handleFileUpload(file)
        }
        // Reset input
        e.target.value = ''
      },
      [handleFileUpload]
    )

    // Handle delete skill
    const handleDeleteSkill = useCallback(async () => {
      if (!skillToDelete) return

      try {
        setDeleting(true)
        const { api } = await import('@/lib/api-client')

        const response = await api.tools.deleteSkill({
          serverId,
          skillName: skillToDelete
        })

        if (response.data?.common?.code === 0) {
          toast.success(`Skill "${skillToDelete}" deleted`)
          await loadSkills()
        } else {
          toast.error(
            response.data?.common?.message || 'Could not delete skill'
          )
        }
      } catch (error: any) {
        // Error already shown via toast below
        toast.error(error.message || 'Could not delete skill')
      } finally {
        setDeleting(false)
        setSkillToDelete(null)
      }
    }, [serverId, skillToDelete, loadSkills])

    // Handle delete all skills
    const handleDeleteAllSkills = useCallback(async () => {
      try {
        setDeletingAll(true)
        const { api } = await import('@/lib/api-client')

        const response = await api.tools.deleteServerSkills({
          serverId
        })

        if (response.data?.common?.code === 0) {
          toast.success('All skills deleted')
          await loadSkills()
        } else {
          toast.error(
            response.data?.common?.message || 'Could not delete all skills'
          )
        }
      } catch (error: any) {
        // Error already shown via toast below
        toast.error(error.message || 'Could not delete all skills')
      } finally {
        setDeletingAll(false)
        setShowDeleteAllDialog(false)
      }
    }, [serverId, loadSkills])

    // Filter and sort skills
    const filteredSkills = skills
      .filter(
        (skill) =>
          skill.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          skill.description?.toLowerCase().includes(searchQuery.toLowerCase())
      )
      .sort((a, b) => {
        switch (sortBy) {
          case 'time-desc': {
            // Sort by time descending (newest first)
            const timeA = a.updatedAt ? new Date(a.updatedAt).getTime() : 0
            const timeB = b.updatedAt ? new Date(b.updatedAt).getTime() : 0
            return timeB - timeA
          }
          case 'time-asc': {
            // Sort by time ascending (oldest first)
            const timeA = a.updatedAt ? new Date(a.updatedAt).getTime() : 0
            const timeB = b.updatedAt ? new Date(b.updatedAt).getTime() : 0
            return timeA - timeB
          }
          case 'name-asc':
            // Sort by name ascending (A-Z)
            return a.name.localeCompare(b.name)
          case 'name-desc':
            // Sort by name descending (Z-A)
            return b.name.localeCompare(a.name)
          default:
            return 0
        }
      })

    return (
      <div className="space-y-6">
        {/* Server Name Display */}
        {serverName && (
          <div className="text-sm text-muted-foreground">
            Configuring: <span className="font-medium">{serverName}</span>
          </div>
        )}

        {/* Skills Directory */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <CardTitle className="text-lg">Skills Directory</CardTitle>
                <span className="px-2 py-0.5 text-xs font-medium bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded-full">
                  {skills.length} skill{skills.length !== 1 ? 's' : ''}
                </span>
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Search skills..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="pl-9 w-[180px]"
                  />
                </div>
              </div>
              <div className="flex items-center gap-2">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button type="button" className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors">
                      <ArrowUpDown className="h-3.5 w-3.5" />
                      <span>
                        {sortBy === 'time-desc' && 'Newest'}
                        {sortBy === 'time-asc' && 'Oldest'}
                        {sortBy === 'name-asc' && 'A-Z'}
                        {sortBy === 'name-desc' && 'Z-A'}
                      </span>
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onSelect={() => setSortBy('time-desc')}>
                      Newest
                      {sortBy === 'time-desc' && (
                        <span className="ml-auto text-blue-500 dark:text-blue-400">✓</span>
                      )}
                    </DropdownMenuItem>
                    <DropdownMenuItem onSelect={() => setSortBy('time-asc')}>
                      Oldest
                      {sortBy === 'time-asc' && (
                        <span className="ml-auto text-blue-500 dark:text-blue-400">✓</span>
                      )}
                    </DropdownMenuItem>
                    <DropdownMenuItem onSelect={() => setSortBy('name-asc')}>
                      Name A-Z
                      {sortBy === 'name-asc' && (
                        <span className="ml-auto text-blue-500 dark:text-blue-400">✓</span>
                      )}
                    </DropdownMenuItem>
                    <DropdownMenuItem onSelect={() => setSortBy('name-desc')}>
                      Name Z-A
                      {sortBy === 'name-desc' && (
                        <span className="ml-auto text-blue-500 dark:text-blue-400">✓</span>
                      )}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
                {skills.length > 0 && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-9 text-muted-foreground hover:text-red-600 dark:text-red-400 dark:hover:text-red-400"
                    onClick={() => setShowDeleteAllDialog(true)}
                  >
                    <Trash2 className="h-4 w-4" />
                    <span className="hidden sm:inline ml-1">Delete All</span>
                  </Button>
                )}
              </div>
            </div>
          </CardHeader>
          <div className="border-t border-gray-200 dark:border-gray-700" />
          <CardContent className="p-0">
            {loading ? (
              <div className="flex items-center justify-center py-8 px-4">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                <span className="ml-2 text-sm text-muted-foreground">
                  Loading skills...
                </span>
              </div>
            ) : filteredSkills.length === 0 ? (
              <div className="text-center py-8 px-4">
                <Folder className="h-12 w-12 text-gray-300 dark:text-gray-600 mx-auto mb-3" />
                <p className="text-sm text-muted-foreground">
                  {searchQuery
                    ? 'No skills match your search'
                    : 'No skills uploaded yet'}
                </p>
                {!searchQuery && (
                  <p className="text-xs text-muted-foreground mt-1">
                    Upload a ZIP file to add skills
                  </p>
                )}
              </div>
            ) : (
              <div className="divide-y divide-gray-200 dark:divide-gray-700">
                {filteredSkills.map((skill) => (
                  <div
                    key={skill.name}
                    className="flex items-center justify-between py-3 hover:bg-gray-50 dark:hover:bg-gray-800/50 px-4 transition-colors"
                  >
                    <div className="flex items-center gap-3 flex-1 min-w-0">
                      <Folder className="h-5 w-5 text-blue-500 dark:text-blue-400 flex-shrink-0" />
                      <div className="flex-1 min-w-0">
                        <p className="font-medium text-gray-900 dark:text-white">
                          {skill.name}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {skill.description || 'No description'}
                        </p>
                        <p className="text-xs text-muted-foreground/70 mt-1">
                          {skill.version && `v${skill.version}`}
                          {skill.version && skill.updatedAt && ' • '}
                          {skill.updatedAt && (
                            <>
                              {'Updated at '}
                              {new Date(skill.updatedAt).toLocaleTimeString('en-US', {
                                hour: '2-digit',
                                minute: '2-digit',
                                hour12: false
                              })}
                              {' '}
                              {new Date(skill.updatedAt).toLocaleDateString('en-US', {
                                month: 'short',
                                day: 'numeric',
                                year: 'numeric'
                              })}
                            </>
                          )}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="h-8 w-8" aria-label={`Open actions for ${skill.name}`}>
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            className="text-red-600 dark:text-red-400"
                            onClick={() => setSkillToDelete(skill.name)}
                          >
                            <Trash2 className="h-4 w-4 mr-2" />
                            Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Upload Area */}
        <div className="space-y-2">
          <Label>Add Skills</Label>
          <button
            type="button"
            className={cn(
              'border border-dashed rounded-lg py-12 px-4 text-center cursor-pointer transition-colors',
              'w-full',
              dragActive
                ? 'border-blue-500 bg-blue-50 dark:bg-blue-950'
                : 'border-gray-300 dark:border-gray-700 hover:border-gray-400 dark:hover:border-gray-600'
            )}
            onDragEnter={handleDragEnter}
            onDragLeave={handleDragLeave}
            onDragOver={handleDragOver}
            onDrop={handleDrop}
            onClick={() => document.getElementById('skills-file-input')?.click()}
          >
            <input
              id="skills-file-input"
              type="file"
              accept=".zip"
              className="hidden"
              onChange={handleFileInputChange}
              disabled={uploading}
            />

            {uploading ? (
              <div className="flex items-center justify-center gap-2">
                <Loader2 className="h-5 w-5 text-blue-500 dark:text-blue-400 animate-spin" />
                <p className="text-sm text-muted-foreground">Uploading...</p>
              </div>
            ) : (
              <div className="flex items-center justify-center gap-3">
                <Upload className="h-5 w-5 text-muted-foreground" />
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  Drag & drop your skills ZIP file or click to select
                </p>
              </div>
            )}
          </button>
        </div>

        {/* ZIP Structure Info */}
        <div className="bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
          <div className="flex items-start gap-2">
            <Info className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
            <div className="text-sm text-blue-800 dark:text-blue-200">
              <p className="font-medium mb-1">ZIP Structure:</p>
              <ul className="text-xs space-y-1 list-disc list-inside">
                <li>Each subdirectory containing a SKILL.md file will be treated as an individual skill</li>
                <li>You can compress your entire skills folder or select multiple skill directories</li>
                <li>Skills with the same name will be replaced</li>
                <li>Maximum ZIP file size: {SKILLS_MAX_FILE_SIZE / 1024 / 1024}MB</li>
              </ul>
            </div>
          </div>
        </div>

        {/* Configuration Options */}
        <PublicAccessConfiguration
          checked={publicAccess}
          onCheckedChange={onPublicAccessChange}
        />

        {!allowUserInput && (
          <AnonymousAccessConfiguration
            checked={anonymousAccess}
            onCheckedChange={onAnonymousAccessChange}
            rateLimit={anonymousRateLimit}
            onRateLimitChange={onAnonymousRateLimitChange}
          />
        )}

        <LazyStartConfiguration
          checked={lazyStartEnabled}
          onCheckedChange={onLazyStartEnabledChange}
        />

        {/* Delete Tool Button */}
        {onDelete && (
          <div className="pt-4 border-t border-gray-200 dark:border-gray-700">
            <Button
              variant="destructive"
              onClick={onDelete}
              className="w-full sm:w-auto"
            >
              <Trash2 className="h-4 w-4 mr-2" />
              Delete Skills Server
            </Button>
          </div>
        )}

        {/* Delete Skill Confirmation Dialog */}
        <AlertDialog
          open={!!skillToDelete}
          onOpenChange={(open) => !open && setSkillToDelete(null)}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete Skill</AlertDialogTitle>
              <AlertDialogDescription>
                Are you sure you want to delete the skill "{skillToDelete}"?
                This action cannot be undone.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
              <AlertDialogAction
                onClick={handleDeleteSkill}
                disabled={deleting}
                className="bg-red-600 hover:bg-red-700"
              >
                {deleting ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Deleting...
                  </>
                ) : (
                  'Delete Skill'
                )}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        {/* Delete All Skills Confirmation Dialog */}
        <AlertDialog
          open={showDeleteAllDialog}
          onOpenChange={(open) => !open && setShowDeleteAllDialog(false)}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete All Skills</AlertDialogTitle>
              <AlertDialogDescription>
                Are you sure you want to delete all {skills.length} skill{skills.length !== 1 ? 's' : ''}?
                This action cannot be undone.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={deletingAll}>Cancel</AlertDialogCancel>
              <AlertDialogAction
                onClick={handleDeleteAllSkills}
                disabled={deletingAll}
                className="bg-red-600 hover:bg-red-700"
              >
                {deletingAll ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Deleting...
                  </>
                ) : (
                  'Delete All Skills'
                )}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    )
  }
)

SkillsConfigTab.displayName = 'SkillsConfigTab'
