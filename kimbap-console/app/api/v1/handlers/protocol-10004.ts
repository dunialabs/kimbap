import { ApiError, ErrorCode } from '@/lib/error-codes';
import { KimbapCloudApiService, ToolTemplate } from '@/lib/KimbapCloudApiService';

interface Request10004 {
  common: {
    cmdId: number;
    userid: string;
  };
  params: {};
}


interface Response10004Data {
  toolTmplList: ToolTemplate[];
}

export async function handleProtocol10004(_body: Request10004): Promise<Response10004Data> {
  try {
    const kimbapCloudApi = new KimbapCloudApiService();
    const templates = await kimbapCloudApi.fetchToolTemplatesWithFallback();
    return {
      toolTmplList: templates
    };
  } catch (error) {
    console.error('Failed to fetch tool templates:', error);
    
    // If it's already an ApiError, re-throw it
    if (error instanceof ApiError) {
      throw error;
    }
    
    // Return empty list as last resort
    return {
      toolTmplList: []
    };
  }
}