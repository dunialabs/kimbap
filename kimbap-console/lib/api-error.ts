export function getApiErrorMessage(error: any, fallback = 'Request failed, please try again later'): string {
  const data = error?.response?.data

  // 统一解析 /api/v1 返回格式：{ common: { code, message, cmdId, errorCode }, data }
  if (data?.common?.message) {
    return data.common.message
  }

  // 某些自定义接口可能直接返回 { message } 或字符串
  if (typeof data?.message === 'string' && data.message.trim()) {
    return data.message
  }

  if (typeof data === 'string' && data.trim()) {
    return data
  }

  // Axios / JS 原生错误
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


