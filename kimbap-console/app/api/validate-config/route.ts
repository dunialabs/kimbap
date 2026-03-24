import { NextRequest, NextResponse } from 'next/server';

import { generateRestApiValidationReport } from '@/lib/rest-api-utils';

export const dynamic = 'force-dynamic';
export const revalidate = 0;

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    if (body?.config === undefined || body?.config === null) {
      return NextResponse.json(
        { success: false, error: 'Request body must include a "config" field.' },
        { status: 400 }
      );
    }

    const configInput = typeof body.config === 'string' ? body.config : JSON.stringify(body.config, null, 2);
    const report = await generateRestApiValidationReport(configInput);
    return NextResponse.json({ success: true, report });
  } catch (error: any) {
    console.error('Failed to validate REST API configuration:', error);
    return NextResponse.json(
      { success: false, error: error?.message || 'Failed to validate configuration.' },
      { status: 500 }
    );
  }
}
