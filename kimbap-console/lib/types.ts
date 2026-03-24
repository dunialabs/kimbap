export interface Host {
  hostId: string
  hostUrl: string
  lastHeartbeat: string
  tools: Tool[]
  status?: 'online' | 'offline'
}

export interface Tool {
  name: string
  description: string
  inputSchema: {
    type: string
    properties: Record<string, any>
    required?: string[]
  }
}

export interface ToolResult {
  content: Array<{
    type: string
    text: string
  }>
  isError?: boolean
}

export interface ApiResponse<T = any> {
  success: boolean
  data?: T
  error?: string
  hosts?: Host[]
}

// Window interface for Electron API
declare global {
  interface Window {
    electronAPI?: {
      getGatewayURL: () => string
    }
    GATEWAY_URL?: string
  }
}