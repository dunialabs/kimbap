/**
 * Example of how to implement multi-language support for error codes
 * This file demonstrates how to extend the error system for internationalization
 */

import { ErrorCode } from './error-codes';

// Example: Chinese error messages
export const ErrorMessagesCN: Record<ErrorCode, string> = {
  // Common errors
  [ErrorCode.MISSING_CMD_ID]: '请求中缺少 cmdId',
  [ErrorCode.PROTOCOL_NOT_IMPLEMENTED]: '协议 {cmdId} 未实现',
  [ErrorCode.INVALID_REQUEST]: '无效的请求格式',
  [ErrorCode.INTERNAL_SERVER_ERROR]: '内部服务器错误',
  
  // Authentication errors
  [ErrorCode.MISSING_MASTER_PASSWORD]: '需要主密码',
  [ErrorCode.INVALID_MASTER_PASSWORD]: '主密码无效',
  [ErrorCode.UNAUTHORIZED]: '未授权',
  [ErrorCode.FORBIDDEN]: '禁止访问',
  [ErrorCode.TOKEN_EXPIRED]: '令牌已过期',
  [ErrorCode.TOKEN_INVALID]: '无效的令牌',
  
  // Validation errors
  [ErrorCode.MISSING_REQUIRED_FIELD]: '缺少必填字段 {field}',
  [ErrorCode.INVALID_FIELD_FORMAT]: '字段 {field} 格式无效',
  [ErrorCode.FIELD_TOO_LONG]: '字段 {field} 超过最大长度',
  [ErrorCode.FIELD_TOO_SHORT]: '字段 {field} 低于最小长度',
  
  // Database errors
  [ErrorCode.DATABASE_CONNECTION_ERROR]: '数据库连接错误',
  [ErrorCode.RECORD_NOT_FOUND]: '记录未找到',
  [ErrorCode.DUPLICATE_RECORD]: '记录已存在',
  [ErrorCode.TRANSACTION_FAILED]: '事务失败',
  
  // Business logic errors
  [ErrorCode.PROXY_ALREADY_INITIALIZED]: '代理服务器已初始化',
  [ErrorCode.SERVER_NOT_FOUND]: '服务器未找到',
  [ErrorCode.USER_NOT_FOUND]: '用户未找到',
  [ErrorCode.PERMISSION_DENIED]: '权限被拒绝',
};

// Example: Japanese error messages
export const ErrorMessagesJP: Record<ErrorCode, string> = {
  // Common errors
  [ErrorCode.MISSING_CMD_ID]: 'リクエストにcmdIdがありません',
  [ErrorCode.PROTOCOL_NOT_IMPLEMENTED]: 'プロトコル {cmdId} は実装されていません',
  // ... add more translations
};

// Language type
export type Language = 'en' | 'cn' | 'jp';

// Error message collections
export const ErrorMessagesByLanguage: Record<Language, Record<ErrorCode, string>> = {
  'en': ErrorMessages, // Import from error-codes.ts
  'cn': ErrorMessagesCN,
  'jp': ErrorMessagesJP,
};

/**
 * Get error message in specific language
 */
export function getLocalizedErrorMessage(
  code: ErrorCode, 
  language: Language = 'en',
  params?: Record<string, any>
): string {
  const messages = ErrorMessagesByLanguage[language];
  let message = messages[code] || ErrorMessagesByLanguage['en'][code] || 'Unknown error';
  
  if (params) {
    // Replace {param} with actual values
    Object.entries(params).forEach(([key, value]) => {
      message = message.replace(`{${key}}`, String(value));
    });
  }
  
  return message;
}

// Example usage in API:
// 
// import { getLocalizedErrorMessage } from './error-codes-i18n';
// 
// // Get language from request header or user preference
// const language = request.headers.get('Accept-Language')?.split('-')[0] || 'en';
// 
// // Return localized error
// const message = getLocalizedErrorMessage(ErrorCode.MISSING_MASTER_PASSWORD, language);