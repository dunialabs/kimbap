'use client'

export const runtime = 'edge'

import {
  ArrowLeft,
  Lock,
  Zap,
  Shield,
  Check,
  Settings,
  Users,
  Eye,
  EyeOff,
  Search,
  CheckSquare,
  Square,
  Minus,
  Database,
  Bot,
  LucideIcon
} from 'lucide-react'
import Link from 'next/link'
import { useParams, useRouter, useSearchParams } from 'next/navigation'
import React, { useState, useEffect, useCallback } from 'react'

import { api } from '@/lib/api-client'
import { Tool, Credential } from '@/types/api'
import { MasterPasswordDialog } from '@/components/master-password-dialog'
import { MasterPasswordManager } from '@/lib/crypto'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { MCPServerConfig } from '@/types/mcp-server'

// Helper function to map runningState number to status string
const getStatusFromRunningState = (runningState: number): string => {
  switch (runningState) {
    case 0:
      return 'connected' // 在线
    case 1:
      return 'connect_failed' // 离线
    case 2:
      return 'connecting' // 连接中
    case 3:
      return 'error' // 异常
    default:
      return 'connect_failed'
  }
}

// Icon mapper for dynamic icon rendering
const IconMapper: Record<string, LucideIcon> = {
  Search,
  Database,
  Bot
}

const DynamicIcon = ({
  iconName,
  className
}: {
  iconName: string
  className?: string
}) => {
  const IconComponent = IconMapper[iconName] || Database
  return <IconComponent className={className} />
}

interface CredentialsTabProps {
  server: MCPServerConfig
  onUpdate: (server: MCPServerConfig) => void
  toolTemplate?: Tool
  dynamicCredentials: Record<string, string>
  onCredentialUpdate: (key: string, value: string) => void
}

function CredentialsTab({
  server,
  onUpdate,
  toolTemplate,
  dynamicCredentials,
  onCredentialUpdate
}: CredentialsTabProps) {
  const [showSecrets, setShowSecrets] = useState<Record<string, boolean>>({})
  const [allowUserInput, setAllowUserInput] = useState(false)

  // Dynamic credentials are now managed by parent component

  const toggleSecret = (field: string) => {
    setShowSecrets((prev) => ({ ...prev, [field]: !prev[field] }))
  }

  const handleCredentialChange = (key: string, value: any) => {
    const updatedServer = {
      ...server,
      credentials: {
        ...server.credentials,
        [key]: typeof value === 'string' ? value : { encrypted: true, value }
      }
    }
    onUpdate(updatedServer)
  }

  // Dynamic credential field renderer
  const renderDynamicCredentialField = (
    credential: Credential,
    index: number
  ) => {
    const { name, description, dataType, key, options } = credential
    // Use name to generate unique field identifier, key is the default value
    const fieldKey = `credential_${index}_${name
      .toLowerCase()
      .replace(/\s+/g, '_')}`
    const fieldValue = dynamicCredentials[fieldKey] || key || ''

    const handleChange = (value: string) => {
      onCredentialUpdate(fieldKey, value)
      handleCredentialChange(fieldKey, value)
    }

    switch (dataType) {
      case 1: // Input field
        // Determine if field should be treated as secret based on name
        const isSecret =
          name.toLowerCase().includes('token') ||
          name.toLowerCase().includes('key') ||
          name.toLowerCase().includes('secret') ||
          name.toLowerCase().includes('password') ||
          name.toLowerCase().includes('api')

        if (isSecret) {
          return (
            <div key={`cred-${index}-${fieldKey}`} className="space-y-2">
              <Label htmlFor={fieldKey}>{name}</Label>
              <div className="relative">
                <Input
                  id={fieldKey}
                  type={showSecrets[fieldKey] ? 'text' : 'password'}
                  placeholder={description || `Enter ${name}`}
                  value={
                    showSecrets[fieldKey]
                      ? fieldValue
                      : fieldValue
                      ? '*'.repeat(Math.min(fieldValue.length, 24))
                      : ''
                  }
                  onChange={(e) => handleChange(e.target.value)}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-2 top-1/2 -translate-y-1/2"
                  onClick={() => toggleSecret(fieldKey)}
                >
                  {showSecrets[fieldKey] ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </Button>
              </div>
              {description && (
                <p className="text-xs text-gray-500">{description}</p>
              )}
            </div>
          )
        } else {
          return (
            <div key={`cred-${index}-${fieldKey}`} className="space-y-2">
              <Label htmlFor={fieldKey}>{name}</Label>
              <Input
                id={fieldKey}
                placeholder={description || `Enter ${name}`}
                value={fieldValue}
                onChange={(e) => handleChange(e.target.value)}
              />
              {description && (
                <p className="text-xs text-gray-500">{description}</p>
              )}
            </div>
          )
        }

      case 2: // Radio buttons
        return (
          <div key={`cred-${index}-${fieldKey}`} className="space-y-2">
            <Label>{name}</Label>
            <div className="space-y-2">
              {options.map((option) => (
                <label key={option.key} className="flex items-center">
                  <input
                    type="radio"
                    name={fieldKey}
                    value={option.value}
                    checked={fieldValue === option.value}
                    onChange={(e) => handleChange(e.target.value)}
                    className="text-blue-600"
                  />
                  <span className="text-sm ml-2">{option.key}</span>
                </label>
              ))}
            </div>
            {description && (
              <p className="text-xs text-gray-500">{description}</p>
            )}
          </div>
        )

      case 3: // Checkbox
        return (
          <div key={`cred-${index}-${fieldKey}`} className="space-y-2">
            <label className="flex items-center space-x-2">
              <input
                type="checkbox"
                checked={fieldValue === 'true'}
                onChange={(e) =>
                  handleChange(e.target.checked ? 'true' : 'false')
                }
                className="text-blue-600"
              />
              <span className="text-sm font-medium">{name}</span>
            </label>
            {description && (
              <p className="text-xs text-gray-500">{description}</p>
            )}
          </div>
        )

      case 4: // Select dropdown
        return (
          <div key={`cred-${index}-${fieldKey}`} className="space-y-2">
            <Label htmlFor={fieldKey}>{name}</Label>
            <select
              id={fieldKey}
              value={fieldValue}
              onChange={(e) => handleChange(e.target.value)}
              className="w-full p-2 border border-gray-300 rounded-md text-sm"
            >
              <option value="">Select {name}</option>
              {options.map((option) => (
                <option key={option.key} value={option.value}>
                  {option.key}
                </option>
              ))}
            </select>
            {description && (
              <p className="text-xs text-gray-500">{description}</p>
            )}
          </div>
        )

      default:
        return (
          <div key={`cred-${index}-${fieldKey}`} className="space-y-2">
            <Label htmlFor={fieldKey}>{name}</Label>
            <Input
              id={fieldKey}
              placeholder={description || `Enter ${name}`}
              value={fieldValue}
              onChange={(e) => handleChange(e.target.value)}
            />
            {description && (
              <p className="text-xs text-gray-500">{description}</p>
            )}
          </div>
        )
    }
  }

  const renderCredentialFields = () => {
    // If tool template has credentials, use dynamic rendering
    if (
      toolTemplate &&
      toolTemplate.credentials &&
      toolTemplate.credentials.length > 0
    ) {
      return (
        <div className="space-y-4">
          {toolTemplate.credentials.map((credential, index) =>
            renderDynamicCredentialField(credential, index)
          )}
        </div>
      )
    }

    // If no template, show empty state
    return (
      <div className="text-center py-8 text-gray-500">
        <p>Please select a tool template to configure credentials</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Console Authentication */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Settings className="h-4 w-4" />
            <h3 className="text-sm font-semibold text-gray-900">
              Console Authentication
            </h3>
          </div>
          <Button size="sm" variant="outline" className="text-xs">
            <Check className="h-3 w-3 mr-1" />
            Test
          </Button>
        </div>
        <p className="text-xs text-gray-500">
          Configure shared credentials for all users
        </p>
        <div className="border rounded-lg p-3">{renderCredentialFields()}</div>
      </div>

      {/* User Input Option */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Users className="h-4 w-4" />
            <div>
              <h4 className="text-sm font-medium text-gray-900">
                Allow user input via Kimbap MCP Desk
              </h4>
              <p className="text-xs text-gray-500">
                Users can input their own credentials when using this tool
              </p>
            </div>
          </div>
          <Switch
            checked={allowUserInput}
            onCheckedChange={setAllowUserInput}
          />
        </div>

        {allowUserInput && (
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
            <div className="flex items-start gap-2">
              <Users className="h-4 w-4 text-blue-600 mt-0.5 flex-shrink-0" />
              <div>
                <h4 className="text-sm font-medium text-blue-900 mb-1">
                  User Authentication Enabled
                </h4>
                <div className="text-xs text-blue-800 space-y-0.5">
                  <p>• Users can input credentials on first use</p>
                  <p>• Credentials stored securely in local Kimbap MCP Desk</p>
                  <p>• Enhanced security through personalized access</p>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

interface FunctionsTabProps {
  server: MCPServerConfig
  onUpdate: (server: MCPServerConfig) => void
  toolTemplate?: Tool
  dynamicCredentials?: Record<string, string>
  onCredentialUpdate?: (key: string, value: string) => void
}

function FunctionsTab({ server, onUpdate, toolTemplate }: FunctionsTabProps) {
  const [serverCapabilities, setServerCapabilities] = useState<{
    functions?: any[]
    resources?: any[]
  }>({})
  const [isLoadingCapabilities, setIsLoadingCapabilities] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedFunctions, setSelectedFunctions] = useState<Set<string>>(
    new Set(server.enabledFunctions)
  )

  // Load server capabilities when component mounts or server.id changes
  useEffect(() => {
    const loadServerCapabilities = async () => {
      if (!server?.id) return

      try {
        setIsLoadingCapabilities(true)
        const { api } = await import('@/lib/api-client')

        // Call protocol 10010 to get server capabilities
        const response = await api.tools.getServerCapabilities({
          toolId: server.id
        })

        if (response.data?.data) {
          setServerCapabilities(response.data.data)
        } else {
        }
      } catch (error) {
        console.error('Failed to load server capabilities:', error)
      } finally {
        setIsLoadingCapabilities(false)
      }
    }

    loadServerCapabilities()
  }, [server?.id])

  // Update selectedFunctions when server capabilities are loaded
  useEffect(() => {
    if (serverCapabilities?.functions) {
      const enabledFromCapabilities = new Set(
        serverCapabilities.functions
          .filter((f: any) => f.enabled)
          .map((f: any) => f.funcName)
      )
      setSelectedFunctions(enabledFromCapabilities)

      // Create allFunctions array with all functions from capabilities and their status
      const allFunctionsWithStatus = serverCapabilities.functions.map(
        (f: any) => ({
          funcName: f.funcName,
          enabled: f.enabled
        })
      )

      // Also update the server state
      const updatedServer = {
        ...server,
        enabledFunctions: Array.from(enabledFromCapabilities),
        allFunctions: allFunctionsWithStatus
      }
      onUpdate(updatedServer)
    }
  }, [serverCapabilities?.functions]) // 更具体的依赖

  // Use server capabilities if available, otherwise fallback to hardcoded
  const serverFunctions = serverCapabilities?.functions || []

  // Convert server functions to expected format
  const dynamicFunctions =
    serverFunctions.length > 0
      ? serverFunctions.map((func) => {
          const mappedFunc = {
            id: func.funcName || func.name,
            name: func.funcName || func.name,
            description:
              func.description || `${func.funcName || func.name} function`,
            category: 'Server Functions',
            enabled: func.enabled
          }
          return mappedFunc
        })
      : []

  const functions = dynamicFunctions.length > 0 ? dynamicFunctions : []

  // Sync functions state to parent when selected functions change (but avoid infinite loops)
  useEffect(() => {
    // Only sync when we have meaningful changes and avoid infinite loops
    const allAvailableFunctions = [...serverFunctions]

    if (allAvailableFunctions.length > 0 && selectedFunctions.size > 0) {
      const currentEnabledFunctions = Array.from(selectedFunctions)

      // Check if current selection is different from server state to avoid unnecessary updates
      const serverEnabledFunctions = server.enabledFunctions || []
      const isDifferent =
        currentEnabledFunctions.length !== serverEnabledFunctions.length ||
        !currentEnabledFunctions.every((func) =>
          serverEnabledFunctions.includes(func)
        )

      if (isDifferent) {
        // Create functions array with all functions and their enabled status
        const allFunctionsWithStatus = allAvailableFunctions.map((func) => ({
          funcName: func.name || func.funcName || func.id,
          enabled: selectedFunctions.has(func.id || func.funcName || func.name)
        }))

        const updatedServer = {
          ...server,
          enabledFunctions: currentEnabledFunctions,
          allFunctions: allFunctionsWithStatus
        }

        onUpdate(updatedServer)
      }
    }
  }, [selectedFunctions]) // Only depend on selectedFunctions to avoid infinite loops

  // Group functions by category
  const functionsByCategory = functions.reduce((acc, func) => {
    if (!acc[func.category]) {
      acc[func.category] = []
    }
    acc[func.category].push(func)
    return acc
  }, {} as Record<string, typeof functions>)

  // Filter functions based on search
  const filteredFunctionsByCategory = Object.entries(
    functionsByCategory
  ).reduce((acc, [category, funcs]) => {
    const filtered = funcs.filter(
      (func) =>
        func.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        func.description.toLowerCase().includes(searchQuery.toLowerCase())
    )
    if (filtered.length > 0) {
      acc[category] = filtered
    }
    return acc
  }, {} as Record<string, typeof functions>)

  const handleFunctionToggle = useCallback(
    (functionId: string) => {
      const newSelected = new Set(selectedFunctions)
      if (newSelected.has(functionId)) {
        newSelected.delete(functionId)
      } else {
        newSelected.add(functionId)
      }
      setSelectedFunctions(newSelected)
      // Note: onUpdate will be called by the useEffect that watches selectedFunctions
    },
    [selectedFunctions]
  )

  const handleCategoryToggle = useCallback(
    (category: string) => {
      const categoryFunctions = functionsByCategory[category]
      const categoryFunctionIds = categoryFunctions.map((f) => f.id)
      const allSelected = categoryFunctionIds.every((id) =>
        selectedFunctions.has(id)
      )

      const newSelected = new Set(selectedFunctions)
      if (allSelected) {
        categoryFunctionIds.forEach((id) => newSelected.delete(id))
      } else {
        categoryFunctionIds.forEach((id) => newSelected.add(id))
      }

      setSelectedFunctions(newSelected)
      // Note: onUpdate will be called by the useEffect that watches selectedFunctions
    },
    [selectedFunctions, functionsByCategory]
  )

  const getCategorySelectionState = (category: string) => {
    const categoryFunctions = functionsByCategory[category]
    const categoryFunctionIds = categoryFunctions.map((f) => f.id)
    const selectedCount = categoryFunctionIds.filter((id) =>
      selectedFunctions.has(id)
    ).length

    if (selectedCount === 0) return 'none'
    if (selectedCount === categoryFunctionIds.length) return 'all'
    return 'some'
  }

  if (isLoadingCapabilities) {
    return (
      <Card>
        <CardContent className="pt-6 text-center">
          <div className="flex items-center justify-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            <span className="ml-3 text-muted-foreground">
              Loading server capabilities...
            </span>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (functions.length === 0) {
    return (
      <Card>
        <CardContent className="pt-6 text-center">
          <p className="text-muted-foreground">No functions available</p>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      {/* Show server capabilities info if loaded */}
      {serverCapabilities?.functions &&
        serverCapabilities.functions.length > 0 && (
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
            <div className="flex items-start gap-2">
              <Zap className="h-4 w-4 text-blue-600 mt-0.5 flex-shrink-0" />
              <div className="text-sm text-blue-800">
                <p className="font-medium mb-1">
                  Server Capabilities Loaded from Tool
                </p>
                <p className="text-xs">
                  Found {serverCapabilities.functions.length} functions and{' '}
                  {serverCapabilities.resources?.length || 0} resources
                  available
                </p>
              </div>
            </div>
          </div>
        )}

      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <Input
            placeholder="Search functions..."
            className="pl-10"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
        <div className="text-sm text-muted-foreground">
          {selectedFunctions.size} / {functions.length} functions enabled
        </div>
      </div>

      <div className="space-y-4">
        {Object.entries(filteredFunctionsByCategory).map(
          ([category, categoryFunctions]) => {
            const selectionState = getCategorySelectionState(category)

            return (
              <Card key={category}>
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="text-base">{category}</CardTitle>
                      <CardDescription>
                        {categoryFunctions.length} functions
                      </CardDescription>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleCategoryToggle(category)}
                    >
                      {selectionState === 'all' ? (
                        <CheckSquare className="w-4 h-4 text-blue-600" />
                      ) : selectionState === 'some' ? (
                        <Minus className="w-4 h-4 text-blue-600" />
                      ) : (
                        <Square className="w-4 h-4 text-gray-400" />
                      )}
                    </Button>
                  </div>
                </CardHeader>
                <CardContent className="space-y-3">
                  {categoryFunctions.map((func) => (
                    <div
                      key={func.id}
                      className="flex items-start space-x-3 p-3 rounded-lg border hover:bg-gray-50 transition-colors"
                    >
                      <Switch
                        checked={selectedFunctions.has(func.id)}
                        onCheckedChange={() => handleFunctionToggle(func.id)}
                      />
                      <div className="flex-1">
                        <div className="font-medium text-sm">{func.name}</div>
                        <div className="text-xs text-muted-foreground">
                          {func.description}
                        </div>
                        <div className="text-xs text-gray-500 mt-1 font-mono">
                          {func.id}
                        </div>
                      </div>
                    </div>
                  ))}
                </CardContent>
              </Card>
            )
          }
        )}
      </div>

      {Object.keys(filteredFunctionsByCategory).length === 0 && (
        <Card>
          <CardContent className="pt-6 text-center">
            <p className="text-muted-foreground">
              No functions match "{searchQuery}"
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

interface PermissionsTabProps {
  server: MCPServerConfig
  onUpdate: (server: MCPServerConfig) => void
  toolTemplate?: Tool
  dynamicCredentials?: Record<string, string>
  onCredentialUpdate?: (key: string, value: string) => void
}

function PermissionsTab({
  server,
  onUpdate,
  toolTemplate
}: PermissionsTabProps) {
  const [serverCapabilities, setServerCapabilities] = useState<{
    functions?: any[]
    resources?: any[]
  }>({})
  const [isLoadingCapabilities, setIsLoadingCapabilities] = useState(false)
  const [newResource, setNewResource] = useState('')

  // Load server capabilities when component mounts
  useEffect(() => {
    const loadServerCapabilities = async () => {
      if (!server?.id) return

      try {
        setIsLoadingCapabilities(true)
        const { api } = await import('@/lib/api-client')

        // Call protocol 10010 to get server capabilities
        const response = await api.tools.getServerCapabilities({
          toolId: server.id
        })

        if (response.data?.data) {
          setServerCapabilities(response.data.data)
        } else {
        }
      } catch (error) {
        console.error('Failed to load server resources:', error)
      } finally {
        setIsLoadingCapabilities(false)
      }
    }

    loadServerCapabilities()
  }, [server?.id])

  // Use server resources if available, otherwise fallback to template
  const serverResources =
    serverCapabilities?.resources?.map((res) => {
      if (typeof res === 'string') {
        return res
      }
      // Handle Protocol 10010 format: {uri: string, enabled: boolean}
      return res.uri || res.name
    }) || []

  // Track enabled resources separately for Protocol 10010 format
  const serverResourcesWithStatus = serverCapabilities?.resources || []
  const enabledServerResources = serverResourcesWithStatus
    .filter((res) => typeof res === 'object' && res.enabled === true)
    .map((res) => res.uri)
  const templateResources =
    toolTemplate?.toolResources?.map((res) => res.uri) || []
  const initialResources =
    enabledServerResources.length > 0
      ? enabledServerResources
      : serverResources.length > 0
      ? serverResources
      : templateResources.length > 0
      ? templateResources
      : server.dataPermissions?.allowedResources || []

  const [resources, setResources] = useState<string[]>(initialResources)
  const [availableResources, setAvailableResources] =
    useState<string[]>(serverResources)

  // Update resources when server capabilities are loaded
  useEffect(() => {
    if (serverCapabilities?.resources) {
      // Set available resources (all resources from server)
      setAvailableResources(serverResources)

      // Set enabled resources (only those with enabled=true)
      if (enabledServerResources.length > 0) {
        setResources(enabledServerResources)

        // Auto-update server with enabled resources
        const updatedServer = {
          ...server,
          dataPermissions: {
            ...server.dataPermissions,
            type: server.dataPermissions?.type || 'default',
            allowedResources: enabledServerResources
          }
        }
        onUpdate(updatedServer)
      } else if (serverResources.length > 0) {
        // If no enabled resources specified, treat all as available but not enabled
        setAvailableResources(serverResources)
      }
    }
  }, [serverCapabilities?.resources])

  const handleAddResource = () => {
    if (newResource.trim() && !resources.includes(newResource.trim())) {
      const updatedResources = [...resources, newResource.trim()]
      setResources(updatedResources)
      setNewResource('')

      const updatedServer = {
        ...server,
        dataPermissions: {
          ...server.dataPermissions,
          type: server.dataPermissions?.type || 'default',
          allowedResources: updatedResources
        }
      }
      onUpdate(updatedServer)
    }
  }

  const handleRemoveResource = (resource: string) => {
    const updatedResources = resources.filter((r) => r !== resource)
    setResources(updatedResources)

    const updatedServer = {
      ...server,
      dataPermissions: {
        ...server.dataPermissions,
        type: server.dataPermissions?.type || 'default',
        allowedResources: updatedResources
      }
    }
    onUpdate(updatedServer)
  }

  const getPermissionDescription = () => {
    if (toolTemplate) {
      return `Configure resource access permissions for ${toolTemplate.name}. Define which resources this tool can access.`
    }
    return 'Configure data access permissions for this tool'
  }

  const getResourcePlaceholder = () => {
    if (
      toolTemplate &&
      toolTemplate.toolResources &&
      toolTemplate.toolResources.length > 0
    ) {
      const sampleResource = toolTemplate.toolResources[0].uri
      return `e.g., ${sampleResource}`
    }
    return 'Resource identifier'
  }

  if (isLoadingCapabilities) {
    return (
      <Card>
        <CardContent className="pt-6 text-center">
          <div className="flex items-center justify-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            <span className="ml-3 text-muted-foreground">
              Loading server resources...
            </span>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      {/* Show server resources info if loaded */}
      {serverCapabilities?.resources &&
        serverCapabilities.resources.length > 0 && (
          <div className="bg-green-50 border border-green-200 rounded-lg p-3">
            <div className="flex items-start gap-2">
              <Shield className="h-4 w-4 text-green-600 mt-0.5 flex-shrink-0" />
              <div className="text-sm text-green-800">
                <p className="font-medium mb-1">
                  Server Resources Loaded from Tool
                </p>
                <p className="text-xs">
                  Found {serverCapabilities.resources.length} resource
                  permissions configured
                </p>
              </div>
            </div>
          </div>
        )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Resource Permissions
          </CardTitle>
          <CardDescription>{getPermissionDescription()}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Available Resources from Server */}
          {availableResources.length > 0 && (
            <div>
              <h4 className="text-sm font-medium mb-2">
                Available Resources from Server
              </h4>
              <div className="space-y-2">
                {availableResources.map((resource) => {
                  const isEnabled = resources.includes(resource)
                  return (
                    <div
                      key={resource}
                      className="flex items-center justify-between p-2 border rounded"
                    >
                      <span className="text-sm font-mono">{resource}</span>
                      <div className="flex items-center gap-2">
                        <Switch
                          checked={isEnabled}
                          onCheckedChange={(checked) => {
                            if (checked) {
                              // Enable resource
                              if (!resources.includes(resource)) {
                                const updatedResources = [
                                  ...resources,
                                  resource
                                ]
                                setResources(updatedResources)
                                const updatedServer = {
                                  ...server,
                                  dataPermissions: {
                                    ...server.dataPermissions,
                                    type:
                                      server.dataPermissions?.type || 'default',
                                    allowedResources: updatedResources
                                  }
                                }
                                onUpdate(updatedServer)
                              }
                            } else {
                              // Disable resource
                              handleRemoveResource(resource)
                            }
                          }}
                          size="sm"
                        />
                        <span className="text-xs text-muted-foreground">
                          {isEnabled ? 'Enabled' : 'Disabled'}
                        </span>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}

          {/* Add Custom Resource */}
          <div>
            <h4 className="text-sm font-medium mb-2">Add Custom Resource</h4>
            <div className="flex gap-2">
              <Input
                placeholder={getResourcePlaceholder()}
                value={newResource}
                onChange={(e) => setNewResource(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleAddResource()}
              />
              <Button onClick={handleAddResource}>Add Resource</Button>
            </div>
          </div>

          <div className="space-y-2">
            <Label>Allowed Resources ({resources.length})</Label>
            {resources.length === 0 ? (
              <div className="text-sm text-muted-foreground py-4 text-center border-2 border-dashed rounded-lg">
                No resources configured. Add resources above to grant access
                permissions.
              </div>
            ) : (
              <div className="space-y-2">
                {resources.map((resource, index) => (
                  <div
                    key={index}
                    className="flex items-center justify-between p-2 bg-gray-50 rounded-lg"
                  >
                    <span className="text-sm font-mono">{resource}</span>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleRemoveResource(resource)}
                      className="text-red-600 hover:text-red-700"
                    >
                      Remove
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

// Helper function to map toolType number to ServerType string
const getServerTypeFromToolType = (toolType: number): string => {
  const typeMap: Record<number, string> = {
    1: 'github',
    2: 'notion',
    3: 'postgresql',
    4: 'slack',
    5: 'openai',
    6: 'stripe',
    7: 'linear',
    8: 'aws',
    9: 'figma',
    10: 'googledrive',
    11: 'sequential-thinking',
    12: 'wcgw',
    13: 'mongodb',
    14: 'mysql',
    15: 'redis',
    16: 'elasticsearch',
    17: 'salesforce',
    18: 'brave-search'
  }
  return typeMap[toolType] || 'github'
}

export default function ToolConfigPage() {
  const params = useParams()
  const router = useRouter()
  const searchParams = useSearchParams()
  const mode = searchParams.get('mode') || 'setup'
  const serverType = searchParams.get('type')
  const templateId = searchParams.get('templateId')
  const stepParam = searchParams.get('step') // 从URL获取step参数
  const toolId = params.id as string // 获取toolId

  // Master password dialog state
  const [showMasterPasswordDialog, setShowMasterPasswordDialog] =
    useState(false)
  const [masterPasswordAction, setMasterPasswordAction] = useState<
    'create' | 'update' | null
  >(null)
  const [isProcessingWithPassword, setIsProcessingWithPassword] =
    useState(false)
  const [masterPasswordForSession, setMasterPasswordForSession] = useState<
    string | null
  >(null)

  // Create a server configuration object from template or serverType
  const createServerFromTemplate = (
    template: Tool | null,
    serverType?: string
  ): MCPServerConfig => {
    if (!template && !serverType) {
      return {
        id: String(params.id),
        type: 'unknown' as any,
        name: 'Unknown Tool',
        credentials: {},
        enabledFunctions: [],
        dataPermissions: {
          type: 'default',
          allowedResources: []
        },
        status: 'connect_failed',
        enabled: false
      }
    }

    // If no template but serverType is provided, create basic config
    if (!template && serverType) {
      return {
        id: String(params.id),
        type: serverType as any,
        name: `${
          serverType.charAt(0).toUpperCase() + serverType.slice(1)
        } Tool`,
        credentials: {},
        enabledFunctions: [],
        dataPermissions: {
          type: 'default',
          allowedResources: []
        },
        status: 'connect_failed',
        enabled: false
      }
    }

    return {
      id: String(params.id),
      type: String(template.toolType) as any,
      name: template.name,
      credentials: {},
      enabledFunctions: template.toolFuncs?.map((f) => f.funcName) || [],
      dataPermissions: {
        type: 'default',
        allowedResources: template.toolResources?.map((r) => r.uri) || []
      },
      status: 'connect_failed',
      enabled: template.enabled || false
    }
  }

  // State hooks must be called before all conditional returns
  const [currentServer, setCurrentServer] = useState<MCPServerConfig | null>(
    null
  )
  const [toolTemplate, setToolTemplate] = useState<Tool | null>(null)
  const [loading, setLoading] = useState(false)
  const [creating, setCreating] = useState(false)
  const [dynamicCredentials, setDynamicCredentials] = useState<
    Record<string, string>
  >({})
  const [toolCreated, setToolCreated] = useState(false)
  const [createdToolId, setCreatedToolId] = useState<string | null>(null)

  // 确定初始步骤
  const getInitialStep = () => {
    // 如果URL有step参数，优先使用
    if (stepParam) {
      const step = parseInt(stepParam, 10)
      if (!isNaN(step)) {
        return step
      }
    }

    // 否则根据mode确定
    switch (mode) {
      case 'credentials':
        return 0
      case 'functions':
        return 1
      case 'permissions':
        return 2
      case 'setup':
      default:
        return 0
    }
  }

  const [currentStep, setCurrentStep] = useState(getInitialStep())

  // 当step变化时更新URL
  const updateStepInUrl = (newStep: number) => {
    const current = new URLSearchParams(searchParams.toString())
    current.set('step', newStep.toString())
    router.push(`/dashboard/tool-configure/${toolId}?${current.toString()}`)
    setCurrentStep(newStep)
  }

  // Load tool template data
  useEffect(() => {
    const loadToolTemplate = async () => {
      if (templateId) {
        try {
          setLoading(true)
          const response = await api.tools.getTemplates()
          if (response.data?.data?.toolTmplList) {
            const template = response.data.data.toolTmplList.find(
              (t: Tool) => t.toolTmplId === templateId
            )
            if (template) {
              setToolTemplate(template)
              // Create server configuration based on template
              setCurrentServer(
                createServerFromTemplate(template, serverType || undefined)
              )
              // Initialize dynamic credentials
              const initialCreds: Record<string, string> = {}
              if (template.credentials) {
                template.credentials.forEach((cred, index) => {
                  const fieldKey = `credential_${index}_${cred.name
                    .toLowerCase()
                    .replace(/\s+/g, '_')}`
                  initialCreds[fieldKey] = cred.key || cred.value || ''
                })
              }
              setDynamicCredentials(initialCreds)
            } else if (serverType) {
              // No template but serverType provided, create basic config
              setCurrentServer(createServerFromTemplate(null, serverType))
            }
          }
        } catch (error) {
          console.error('Failed to load tool template:', error)
        } finally {
          setLoading(false)
        }
      }
    }

    loadToolTemplate()
  }, [templateId])

  // Load existing tool data if in edit mode
  useEffect(() => {
    const loadExistingTool = async () => {
      // 如果toolId不是'new'且没有serverType，说明是编辑模式
      if (toolId !== 'new' && !serverType && !toolCreated) {
        try {
          setLoading(true)
          const { api } = await import('@/lib/api-client')

          // Get server info first to get proxyId
          const serverInfoResponse = await api.servers.getInfo()
          const proxyId = serverInfoResponse.data?.data?.proxyId

          if (!proxyId) {
            return
          }

          // Get tool list and find the specific tool
          const response = await api.tools.getToolList({
            proxyId: proxyId,
            handleType: 1 // 1-all
          })

          if (response.data?.data?.toolList) {
            const tool = response.data.data.toolList.find(
              (t: any) => t.toolId === toolId
            )

            if (tool) {
              // Set toolCreated to true since we're editing
              setToolCreated(true)
              setCreatedToolId(toolId)

              // Create server configuration from existing tool
              const serverConfig: MCPServerConfig = {
                id: tool.toolId,
                type: getServerTypeFromToolType(tool.toolType),
                name: tool.name,
                credentials: tool.credentials || {},
                enabledFunctions:
                  tool.toolFuncs
                    ?.filter((f: any) => f.enabled)
                    .map((f: any) => f.funcName) || [],
                dataPermissions: {
                  type: 'default',
                  allowedResources:
                    tool.toolResources
                      ?.filter((r: any) => r.enabled)
                      .map((r: any) => r.uri) || []
                },
                status: getStatusFromRunningState(tool.runningState),
                lastValidated: tool.lastUsed
                  ? new Date(tool.lastUsed * 1000).toISOString()
                  : undefined,
                enabled: tool.enabled
              }

              setCurrentServer(serverConfig)

              // Initialize dynamic credentials
              const initialCreds: Record<string, string> = {}
              if (tool.credentials && Array.isArray(tool.credentials)) {
                tool.credentials.forEach((cred: any, index: number) => {
                  const fieldKey = `credential_${index}_${(
                    cred.name || cred.key
                  )
                    .toLowerCase()
                    .replace(/\s+/g, '_')}`
                  initialCreds[fieldKey] = cred.value || ''
                })
              }
              setDynamicCredentials(initialCreds)

              // If we have a template ID, load it
              if (tool.toolTmplId) {
                const templateResponse = await api.tools.getTemplates()
                if (templateResponse.data?.data?.toolTmplList) {
                  const template = templateResponse.data.data.toolTmplList.find(
                    (t: Tool) => t.toolTmplId === tool.toolTmplId
                  )
                  if (template) {
                    setToolTemplate(template)
                  }
                }
              }
            }
          }
        } catch (error) {
          console.error('Failed to load existing tool:', error)
        } finally {
          setLoading(false)
        }
      }
    }

    loadExistingTool()
  }, [toolId, serverType])

  // 早期返回必须在所有钩子之后
  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Loading tool template...</p>
        </div>
      </div>
    )
  }

  if (!currentServer) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-gray-900 mb-4">
            Tool Configuration Not Available
          </h1>
          <p className="text-gray-600 mb-4">
            Unable to load tool configuration. Please try again.
          </p>
          <Link
            href="/dashboard/tool-configure"
            className="text-blue-600 hover:text-blue-700"
          >
            Back to Tool List
          </Link>
        </div>
      </div>
    )
  }

  // Get tool metadata (icon, color, etc.)
  const getToolMetadata = () => {
    const typeStyles: Record<number, { icon: string; bgColor: string }> = {
      1: { icon: 'Search', bgColor: 'bg-blue-500' }, // Brave Search
      2: { icon: 'Database', bgColor: 'bg-gray-800' }, // GitHub
      3: { icon: 'Bot', bgColor: 'bg-purple-500' } // Notion
    }
    if (toolTemplate) {
      return (
        typeStyles[toolTemplate.toolType] || {
          icon: 'Bot',
          bgColor: 'bg-gray-500'
        }
      )
    }
    // Default for no template
    return { icon: 'Bot', bgColor: 'bg-gray-500' }
  }

  const metadata = getToolMetadata()

  // 定义所有可能的步骤
  const allSteps = [
    {
      id: 'credentials',
      title: 'Credentials',
      description: 'Configure authentication',
      icon: Lock,
      color: 'text-orange-500',
      bgColor: 'bg-orange-500',
      component: CredentialsTab
    },
    {
      id: 'functions',
      title: 'Functions',
      description: 'Enable tool features',
      icon: Zap,
      color: 'text-yellow-500',
      bgColor: 'bg-yellow-500',
      component: FunctionsTab
    },
    {
      id: 'permissions',
      title: 'Resource Permissions',
      description: 'Set resource access',
      icon: Shield,
      color: 'text-red-500',
      bgColor: 'bg-red-500',
      component: PermissionsTab
    }
  ]

  // 基于模式过滤步骤
  const getStepsForMode = () => {
    switch (mode) {
      case 'credentials':
        return [allSteps[0]]
      case 'functions':
        return [allSteps[1]]
      case 'permissions':
        return [allSteps[2]]
      case 'setup':
      default:
        return allSteps
    }
  }

  const steps = getStepsForMode()
  // 添加边界检查，防止数组越界
  const safeCurrentStep = Math.min(Math.max(currentStep, 0), steps.length - 1)
  const CurrentStepComponent = steps[safeCurrentStep]?.component
  const progress = ((safeCurrentStep + 1) / steps.length) * 100

  // 如果currentStep越界，重置为安全值
  if (currentStep !== safeCurrentStep) {
    setCurrentStep(safeCurrentStep)
  }

  const handleServerUpdate = (updatedServer: MCPServerConfig) => {
    console.log('[handleServerUpdate] Updating server state:', {
      enabledFunctions: updatedServer.enabledFunctions,
      allowedResources: updatedServer.dataPermissions?.allowedResources
    })
    setCurrentServer(updatedServer)
  }

  // Handle tool creation (first step only)
  const handleCreateTool = async () => {
    if (!currentServer) {
      console.error('Missing required data for tool creation')
      return
    }

    try {
      setCreating(true)

      // Show master password dialog
      setMasterPasswordAction('create')
      setShowMasterPasswordDialog(true)
      return
    } catch (error) {
      console.error('Failed to create tool:', error)
      // TODO: Show error message to user
    } finally {
      setCreating(false)
    }
  }

  const performCreateTool = async (masterPwd: string) => {
    if (!currentServer) {
      console.error('Missing required data for tool creation')
      return
    }

    try {
      setCreating(true)

      // 保存master password供后续步骤使用
      setMasterPasswordForSession(masterPwd)

      console.log(
        '[performCreateTool] Dynamic credentials:',
        dynamicCredentials
      )
      console.log('[performCreateTool] Tool template:', toolTemplate)

      // Prepare auth configuration from dynamic credentials
      const authConf = Object.entries(dynamicCredentials)
        .filter(([_, value]) => value.trim() !== '')
        .map(([key, value]) => {
          // Find the corresponding credential definition if template exists
          if (toolTemplate) {
            const credIndex = parseInt(key.split('_')[1])
            const credential = toolTemplate.credentials?.[credIndex]
            const authConfItem = {
              key: credential?.key || key,
              value: value,
              dataType: credential?.dataType || 1,
              name: credential?.name || key,
              description: credential?.description || ''
            }
            console.log(
              `[performCreateTool] Auth config item ${credIndex}:`,
              authConfItem
            )
            return authConfItem
          }
          // If no template, use basic configuration
          return {
            key: key,
            value: value,
            dataType: 1, // Default to input type
            name: key,
            description: ''
          }
        })

      console.log('[performCreateTool] Final authConf:', authConf)

      // Get server info first to get proxyId
      const { api } = await import('@/lib/api-client')
      const serverInfoResponse = await api.servers.getInfo()
      const proxyId = serverInfoResponse.data?.data?.proxyId || '1'

      // Prepare initial functions and resources from template

      const initialResources =
        toolTemplate?.toolResources?.map((res) => ({
          uri: res.uri,
          enabled: res.enabled || true // Default to enabled if not specified
        })) || []

      console.log(
        '[performCreateTool] Initial resources from template:',
        initialResources
      )

      // Call protocol 10005 to create the tool (step 1)
      const response = await api.tools.operateTool({
        handleType: 1, // 1 = add tool
        proxyId,
        // toolId: currentServer.id,
        toolTmplId: templateId || undefined,
        toolType: toolTemplate ? toolTemplate.toolType : 1, // Default to type 1 if no template
        authConf,
        masterPwd
      })

      if (response.data?.data?.toolId) {
        console.log('Tool created successfully:', response.data.data.toolId)
        setToolCreated(true)
        setCreatedToolId(response.data.data.toolId)

        // Update currentServer with the real toolId and initial configurations for next steps
        setCurrentServer((prev) =>
          prev
            ? {
                ...prev,
                id: response.data.data.toolId,
                enabledFunctions: [],
                dataPermissions: {
                  ...prev.dataPermissions,
                  allowedResources: initialResources.map((r) => r.uri)
                }
              }
            : prev
        )

        // Update URL with the real toolId
        const newUrl = `/dashboard/tool-configure/${response.data.data.toolId}?mode=setup&step=1`
        if (toolTemplate) {
          router.push(`${newUrl}&templateId=${toolTemplate.toolTmplId}`)
        } else {
          router.push(newUrl)
        }

        // Move to next step instead of navigating away
        setCurrentStep(1)
      } else {
        throw new Error('No tool ID returned from server')
      }
    } catch (error) {
      console.error('Failed to create tool:', error)
      // TODO: Show error message to user
    } finally {
      setCreating(false)
    }
  }

  // Handle tool update (steps 2 and 3)
  const handleUpdateTool = async (
    stepData: 'functions' | 'permissions' | 'complete'
  ) => {
    if (!currentServer || !createdToolId) {
      console.error('Missing required data for tool update')
      return
    }

    try {
      setCreating(true)

      // For steps 2 and 3, check if we have master password from session
      // If not, don't require it - just proceed with empty password
      if (masterPasswordForSession) {
        // Use the password from session
        await performUpdateTool(stepData, masterPasswordForSession)
      } else if (stepData === 'complete' || safeCurrentStep > 0) {
        // For steps 2 and 3, don't require master password
        // The tool was already created with credentials in step 1
        await performUpdateTool(stepData, '')
      } else {
        // Only for step 1, show master password dialog if needed
        setMasterPasswordAction('update')
        setShowMasterPasswordDialog(true)
      }
    } catch (error) {
      console.error('Failed to update tool:', error)
      setCreating(false)
      // TODO: Show error message to user
    }
  }

  const performUpdateTool = async (
    stepData: 'functions' | 'permissions' | 'complete',
    masterPwd: string
  ) => {
    if (!currentServer || !createdToolId) {
      console.error('Missing required data for tool update')
      return
    }

    try {
      setCreating(true)

      // Get server info first to get proxyId
      const { api } = await import('@/lib/api-client')
      const serverInfoResponse = await api.servers.getInfo()
      const proxyId = serverInfoResponse.data?.data?.proxyId || '1'

      let functions: any[] = []
      let resources: any[] = []

      if (stepData === 'functions' || stepData === 'complete') {
        // Prepare functions configuration
        console.log(
          '[performUpdateTool] Current server enabled functions:',
          currentServer.enabledFunctions
        )
        console.log(
          '[performUpdateTool] Current server all functions:',
          currentServer.allFunctions
        )

        // Use allFunctions if available (with actual enabled/disabled status)
        // Otherwise fall back to old behavior for backward compatibility
        if (currentServer.allFunctions) {
          functions = currentServer.allFunctions
        } else {
          // Fallback: only enabled functions
          functions = currentServer.enabledFunctions.map((funcName) => ({
            funcName,
            enabled: true
          }))
        }

        console.log(
          '[performUpdateTool] Prepared functions for API:',
          functions
        )
      }

      if (stepData === 'permissions' || stepData === 'complete') {
        // Prepare resources configuration
        resources =
          currentServer.dataPermissions?.allowedResources?.map((uri) => ({
            uri,
            enabled: true
          })) || []
        console.log(
          '[performUpdateTool] Prepared resources for API:',
          resources
        )
      }
      console.log(
        currentServer,
        'currentserver!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!'
      )
      // Call protocol 10005 to update the tool
      const requestParams: any = {
        handleType: 2, // 2 = edit tool
        proxyId,
        toolId: createdToolId,
        toolTmplId: templateId || undefined,
        toolType: toolTemplate ? toolTemplate.toolType : 1, // Default to type 1 if no template
        authConf: [], // Already set during creation
        functions,
        resources
      }

      // Only include masterPwd if it's provided and not empty
      if (masterPwd && masterPwd.trim() !== '') {
        requestParams.masterPwd = masterPwd
      }

      const response = await api.tools.operateTool(requestParams)

      if (stepData === 'complete') {
        console.log('Tool configuration completed')
        // Don't reset creating state here since we're navigating away
        // Navigate back to tool list
        router.push('/dashboard/tool-configure')
        return // Exit without resetting creating state
      } else {
        // Move to next step
        setCurrentStep((prev) => prev + 1)
        updateStepInUrl(safeCurrentStep + 1)
      }
      setCreating(false)
    } catch (error) {
      console.error('Failed to update tool:', error)
      setCreating(false)
      // TODO: Show error message to user
    }
  }

  // Handle credential updates from child components
  const handleCredentialUpdate = (key: string, value: string) => {
    console.log(`[handleCredentialUpdate] Updating ${key} = ${value}`)
    setDynamicCredentials((prev) => {
      const updated = { ...prev, [key]: value }
      console.log('[handleCredentialUpdate] Updated credentials:', updated)
      return updated
    })
  }

  // Handle master password confirmation
  const handleMasterPasswordConfirm = async (password: string) => {
    try {
      setIsProcessingWithPassword(true)

      // Execute the pending action
      if (masterPasswordAction === 'create') {
        await performCreateTool(password)
      } else if (masterPasswordAction === 'update') {
        const stepMap = {
          functions: 'functions' as const,
          permissions: 'permissions' as const,
          complete: 'complete' as const
        }
        const stepData =
          stepMap[
            safeCurrentStep === 1
              ? 'functions'
              : safeCurrentStep === 2
              ? 'permissions'
              : 'complete'
          ]
        await performUpdateTool(stepData, password)
      }

      // Reset dialog state
      setShowMasterPasswordDialog(false)
      setMasterPasswordAction(null)
    } catch (error) {
      console.error('Error with master password:', error)
      // TODO: Show error message to user
    } finally {
      setIsProcessingWithPassword(false)
    }
  }

  // 获取模式特定的标题
  const getModeTitle = () => {
    switch (mode) {
      case 'credentials':
        return 'Credentials'
      case 'functions':
        return 'Functions'
      case 'permissions':
        return 'Resource Permissions'
      case 'setup':
      default:
        return 'Setup'
    }
  }

  return (
    <div className="space-y-4">
      {/* 紧凑的头部导航 */}
      <div className="bg-white rounded-lg border p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Link
              href="/dashboard/tool-configure"
              className="inline-flex items-center text-gray-500 hover:text-gray-700 transition-colors"
            >
              <ArrowLeft className="w-4 h-4 mr-1" />
              Back
            </Link>
            <div className="h-4 w-px bg-gray-300" />
            <div
              className={`w-10 h-10 rounded-lg flex items-center justify-center ${metadata.bgColor}`}
            >
              <DynamicIcon
                iconName={metadata.icon}
                className="w-5 h-5 text-white"
              />
            </div>
            <div>
              <h1 className="text-lg font-bold text-gray-900">
                {toolTemplate?.name || currentServer.name}
              </h1>
              <p className="text-xs text-gray-500">
                {toolTemplate?.name || currentServer.name} • {getModeTitle()}
              </p>
            </div>
          </div>
          <div className="text-xs text-gray-500">
            Progress: {Math.round(progress)}%
          </div>
        </div>
      </div>

      {/* 步骤导航 - 仅在多步骤设置中显示 */}
      {mode === 'setup' && (
        <div className="bg-white rounded-lg border p-3">
          <div
            className={`grid gap-2 mb-3 ${
              steps.length === 1
                ? 'grid-cols-1'
                : steps.length === 2
                ? 'grid-cols-2'
                : 'grid-cols-3'
            }`}
          >
            {steps.map((step, index) => {
              const Icon = step.icon
              const isActive = index === safeCurrentStep
              const isCompleted = index < safeCurrentStep
              const isClickable = isCompleted || isActive

              return (
                <button
                  key={step.id}
                  onClick={() => isClickable && updateStepInUrl(index)}
                  disabled={!isClickable}
                  className={`
                    p-3 rounded-md border transition-all duration-200 text-left
                    ${
                      isActive
                        ? 'border-blue-500 bg-blue-50'
                        : isCompleted
                        ? 'border-green-500 bg-green-50 cursor-pointer hover:bg-green-100'
                        : isClickable
                        ? 'border-gray-200 hover:border-gray-300 cursor-pointer'
                        : 'border-gray-200 opacity-50 cursor-not-allowed'
                    }
                  `}
                >
                  <div className="flex items-center space-x-2">
                    <div
                      className={`
                      w-6 h-6 rounded-md flex items-center justify-center
                      ${
                        isCompleted
                          ? 'bg-green-500 text-white'
                          : isActive
                          ? `${step.bgColor} text-white`
                          : 'bg-gray-200 text-gray-400'
                      }
                    `}
                    >
                      {isCompleted ? (
                        <Check className="w-3 h-3" />
                      ) : (
                        <Icon className="w-3 h-3" />
                      )}
                    </div>
                    <div>
                      <h4
                        className={`text-sm font-medium ${
                          isActive
                            ? 'text-blue-900'
                            : isCompleted
                            ? 'text-green-900'
                            : 'text-gray-700'
                        }`}
                      >
                        {step.title}
                      </h4>
                      <p className="text-xs text-gray-500">
                        {step.description}
                      </p>
                    </div>
                  </div>
                </button>
              )
            })}
          </div>

          {/* 进度条 */}
          <div className="w-full bg-gray-200 rounded-full h-1.5">
            <div
              className="bg-blue-500 h-1.5 rounded-full transition-all duration-500"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>
      )}

      {/* 主要内容 */}
      <div className="bg-white rounded-lg border">
        {/* 内容头部 - 仅在多步骤设置中显示 */}
        {mode === 'setup' && (
          <div className="px-4 py-3 border-b border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-lg font-bold text-gray-900">
                  {steps[safeCurrentStep]?.title || 'Loading...'}
                </h2>
                <p className="text-xs text-gray-600 mt-0.5">
                  {steps[safeCurrentStep]?.id === 'credentials' &&
                    `Configure authentication for ${
                      toolTemplate?.name || currentServer.name
                    }`}
                  {steps[safeCurrentStep]?.id === 'functions' &&
                    `Select functions to enable for ${
                      toolTemplate?.name || currentServer.name
                    }`}
                  {steps[safeCurrentStep]?.id === 'permissions' &&
                    `Set data access permissions for ${
                      toolTemplate?.name || currentServer.name
                    }`}
                </p>
              </div>
              <div className="flex items-center gap-2">
                {safeCurrentStep > 0 &&
                  !(safeCurrentStep === 1 && toolCreated) && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => updateStepInUrl(safeCurrentStep - 1)}
                      disabled={creating}
                    >
                      <ArrowLeft className="w-3 h-3 mr-1" />
                      Previous
                    </Button>
                  )}
                {safeCurrentStep === steps.length - 1 ? (
                  <Button
                    size="sm"
                    onClick={() => handleUpdateTool('complete')}
                    disabled={creating}
                    className="bg-green-600 hover:bg-green-700"
                  >
                    {creating ? (
                      <>
                        <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-white mr-1"></div>
                        Completing...
                      </>
                    ) : (
                      <>
                        <Check className="w-3 h-3 mr-1" />
                        Complete Setup
                      </>
                    )}
                  </Button>
                ) : (
                  <Button
                    size="sm"
                    onClick={() => {
                      if (safeCurrentStep === 0 && !toolCreated) {
                        // First step: create tool with credentials
                        handleCreateTool()
                      } else if (safeCurrentStep === 1 && toolCreated) {
                        // Second step: update functions (no master password needed if we have it from session)
                        handleUpdateTool('functions')
                      } else if (safeCurrentStep === 2 && toolCreated) {
                        // Third step: update permissions (no master password needed)
                        handleUpdateTool('permissions')
                      } else {
                        // Other steps: just move forward
                        updateStepInUrl(safeCurrentStep + 1)
                      }
                    }}
                    disabled={creating}
                  >
                    {creating && safeCurrentStep === 0 ? (
                      <>
                        <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-white mr-1"></div>
                        Creating...
                      </>
                    ) : creating ? (
                      <>
                        <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-white mr-1"></div>
                        Updating...
                      </>
                    ) : (
                      <>
                        Next
                        <ArrowLeft className="w-3 h-3 ml-1 rotate-180" />
                      </>
                    )}
                  </Button>
                )}
              </div>
            </div>
          </div>
        )}

        {/* 单模式保存按钮 */}
        {mode !== 'setup' && (
          <div className="px-4 py-3 border-b border-gray-200 flex justify-end">
            <Button
              size="sm"
              onClick={() => {
                if (!toolCreated) {
                  handleCreateTool()
                } else {
                  handleUpdateTool('complete')
                }
              }}
              disabled={creating}
              className="bg-green-600 hover:bg-green-700"
            >
              {creating ? (
                <>
                  <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-white mr-1"></div>
                  {!toolCreated ? 'Creating...' : 'Saving...'}
                </>
              ) : (
                <>
                  <Check className="w-3 h-3 mr-1" />
                  {!toolCreated ? 'Create Tool' : 'Save Changes'}
                </>
              )}
            </Button>
          </div>
        )}

        {/* 组件内容 */}
        <div className="p-4">
          {CurrentStepComponent ? (
            <CurrentStepComponent
              server={currentServer}
              onUpdate={handleServerUpdate}
              toolTemplate={toolTemplate || undefined}
              dynamicCredentials={dynamicCredentials}
              onCredentialUpdate={handleCredentialUpdate}
            />
          ) : (
            <div className="text-center py-8">
              <div className="text-gray-500">Loading step component...</div>
            </div>
          )}
        </div>
      </div>

      {/* Master Password Dialog */}
      <MasterPasswordDialog
        open={showMasterPasswordDialog}
        onOpenChange={(open) => {
          setShowMasterPasswordDialog(open)
          if (!open) {
            setMasterPasswordAction(null)
          }
        }}
        onConfirm={handleMasterPasswordConfirm}
        title={
          masterPasswordAction === 'create'
            ? 'Create Tool - Master Password Required'
            : 'Update Tool - Master Password Required'
        }
        description={
          masterPasswordAction === 'create'
            ? 'Please enter your master password to create the tool with encrypted credentials.'
            : 'Please enter your master password to update the tool configuration.'
        }
        isLoading={isProcessingWithPassword}
      />
    </div>
  )
}
