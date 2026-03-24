export async function extractResponseBody(response: Response): Promise<any> {
  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    try {
      return await response.json();
    } catch {
      // fallback to text
    }
  }
  try {
    return await response.text();
  } catch {
    return undefined;
  }
}

export function collectResponseHeaders(response: Response, limit = 12): Record<string, string> {
  const headers: Record<string, string> = {};
  let count = 0;
  response.headers.forEach((value, key) => {
    if (count >= limit) {
      return;
    }
    headers[key] = value;
    count += 1;
  });
  return headers;
}

export function formatBodyPreview(body: any, maxLength = 2000): string | undefined {
  if (body === undefined || body === null) {
    return undefined;
  }

  let serialized: string;
  if (typeof body === 'string') {
    serialized = body;
  } else {
    try {
      serialized = JSON.stringify(body, null, 2);
    } catch {
      serialized = String(body);
    }
  }

  if (serialized.length > maxLength) {
    return `${serialized.slice(0, maxLength)}…`;
  }
  return serialized;
}
