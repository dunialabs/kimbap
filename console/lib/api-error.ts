export function getApiErrorMessage(error: any, fallback = 'Request failed, please try again later'): string {
  const data = error?.response?.data

  // Parse /api/v1 standard response: { common: { code, message, cmdId, errorCode }, data }
  if (data?.common?.message) {
    return data.common.message
  }

  // Some custom endpoints return { message } or a plain string directly
  if (typeof data?.message === 'string' && data.message.trim()) {
    return data.message
  }

  if (typeof data === 'string' && data.trim()) {
    return data
  }

  // Axios / native JS error
  if (typeof error?.message === 'string' && error.message.trim()) {
    return error.message
  }

  return fallback
}

export function getApiErrorCode(error: any): string | undefined {
  const data = error?.response?.data
  const code = data?.common?.errorCode || data?.errorCode || data?.code
  return typeof code === 'string' && code.trim() ? code : undefined
}


