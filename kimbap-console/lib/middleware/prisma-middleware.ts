import { NextRequest, NextResponse } from 'next/server'
import { dbService } from '@/lib/db-service'

export function withDb(
  handler: (req: NextRequest) => Promise<NextResponse>
) {
  return async (req: NextRequest) => {
    const startTime = Date.now()
    
    try {
      const response = await handler(req)
      const responseTime = Date.now() - startTime
      
      // Log API request asynchronously
      dbService.apiRequests.create({
        method: req.method,
        url: req.url,
        statusCode: response.status,
        responseTime
      }).catch(error => {
        console.error('Failed to log API request:', error)
      })
      
      return response
    } catch (error) {
      console.error('API error:', error)
      
      // Try to log the error
      dbService.logs.create(
        'error',
        `API error: ${req.method} ${req.url}`,
        { error: error instanceof Error ? error.message : String(error) }
      ).catch(logError => {
        console.error('Failed to log error:', logError)
      })
      
      return NextResponse.json(
        { success: false, error: 'Internal server error' },
        { status: 500 }
      )
    }
  }
}