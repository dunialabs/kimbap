# KIMBAP Console 加密系统文档

## 1. 加密体系概述

KIMBAP Console 采用了多层级的加密体系，实现了安全的密钥派生（PBKDF2）和数据加密（AES-GCM）：

```
主密码 (Master Password)
    ↓ PBKDF2 + 随机盐值
主密钥哈希 + 盐值 (存储用于验证)
    ↓ 
主密码 → 主密钥 (Master Key) (PBKDF2 + 随机盐值)
    ↓ AES-GCM 加密
Token (加密后的访问令牌)
    ↓ 
Token → Token密钥 (PBKDF2 + 随机盐值)
    ↓ AES-GCM 加密
API Keys (加密后的API密钥)
```

## 2. 加密算法和原理

### 2.1 使用的加密算法

- **密钥派生**: PBKDF2-SHA256（100,000次迭代）
- **对称加密**: AES-256-GCM
- **哈希算法**: SHA-256
- **随机数生成**: Web Crypto API 的安全随机数生成器

### 2.2 加密原理说明

#### PBKDF2（Password-Based Key Derivation Function 2）
- 通过对密码进行多次迭代哈希，生成固定长度的密钥
- 使用盐值防止彩虹表攻击
- 高迭代次数（100,000次）增加暴力破解的计算成本

#### AES-GCM（Advanced Encryption Standard - Galois/Counter Mode）
- 提供机密性和完整性保护
- 使用96位初始化向量（IV）
- 生成16字节的认证标签（Authentication Tag）
- 防止密文被篡改

## 3. 安全性分析

### 3.1 安全特性

1. **密码安全存储**
   - 主密码永不以明文形式存储
   - 仅存储PBKDF2派生的哈希值和随机盐值
   - 每个用户使用唯一的随机盐值，防止彩虹表攻击

2. **多层加密保护**
   - Token使用主密码加密
   - API Keys使用Token加密
   - 每层使用独立的密钥
   - 数据库中仅存储加密后的token和apikey

3. **防重放攻击**
   - 每次加密使用随机IV
   - 认证标签确保数据完整性

### 3.2 潜在风险和缓解措施

1. **浏览器存储风险**
   - localStorage 可能被XSS攻击访问
   - 建议：配合HTTPOnly Cookie或更安全的存储方案
   - 缓解：使用随机盐值，即使数据泄露也难以批量破解

2. **本地攻击风险**
   - 恶意软件可能访问localStorage数据
   - 缓解：高强度PBKDF2迭代次数，增加破解成本

## 4. 核心加密函数说明

### 4.1 密钥生成函数

```typescript
static generateToken(length: number = 64): string
```
- **功能**: 生成安全的随机令牌
- **参数**: `length` - 生成的字节数（默认64字节）
- **返回**: 十六进制字符串（长度为 length × 2）
- **用途**: 生成访问令牌、会话密钥等

```typescript
static generateSalt(): Uint8Array
```
- **功能**: 生成随机盐值
- **返回**: 128位随机盐值
- **用途**: 用于密钥派生，防止彩虹表攻击

### 4.2 密钥派生函数

```typescript
static async deriveKey(
  password: string,
  salt: Uint8Array,
  iterations: number = 100000,
  extractable: boolean = false
): Promise<CryptoKey>
```
- **功能**: 使用PBKDF2从密码派生AES密钥
- **参数**:
  - `password`: 原始密码
  - `salt`: 盐值
  - `iterations`: 迭代次数
  - `extractable`: 密钥是否可导出
- **返回**: CryptoKey对象（AES-256-GCM密钥）

### 4.3 加密函数

```typescript
static async encryptData(
  originalData: string,
  key: string
): Promise<EncryptedData>
```
- **功能**: 使用密钥加密数据
- **流程**:
  1. 生成随机盐值
  2. 使用密钥和盐值派生AES密钥
  3. 生成随机IV
  4. 使用AES-GCM加密
  5. 返回加密数据、IV、盐值和认证标签

### 4.4 解密函数

```typescript
static async decryptDataFromString(
  encryptedDataString: string,
  key: string
): Promise<string>
```
- **功能**: 解密JSON字符串格式的加密数据
- **流程**:
  1. 解析JSON获取加密数据结构
  2. 使用密钥和存储的盐值派生AES密钥
  3. 验证认证标签
  4. 解密数据

### 4.5 密码哈希和验证函数

```typescript
static async hashPasswordWithSalt(
  password: string, 
  salt: Uint8Array
): Promise<string>
```
- **功能**: 使用自定义盐值对密码进行哈希
- **参数**: 
  - `password`: 原始密码
  - `salt`: 随机盐值
- **返回**: Base64编码的密码哈希

```typescript
static async verifyPasswordWithSalt(
  password: string,
  storedHash: string,
  salt: Uint8Array
): Promise<boolean>
```
- **功能**: 使用盐值验证密码
- **流程**:
  1. 使用提供的盐值和迭代次数重新计算密码哈希
  2. 比较计算结果与存储的哈希值

## 5. 加密数据结构

```typescript
interface EncryptedData {
  data: string;   // Base64编码的加密数据
  iv: string;     // Base64编码的初始化向量
  salt: string;   // Base64编码的盐值
  tag: string;    // Base64编码的认证标签
}

interface MasterPasswordData {
  hash: string;   // 密码哈希（使用随机盐值）
  salt: string;   // Base64编码的随机盐值
}
```

## 6. 实际应用场景

### 6.1 主密码设置和验证

```typescript
// 设置主密码 - 现在使用随机盐值
await MasterPasswordManager.setMasterPassword("user_password");
// 存储格式：{ hash: "...", salt: "base64_encoded_random_salt" }

// 验证主密码 - 使用存储的盐值
const isValid = await MasterPasswordManager.verifyMasterPassword("user_password");
```

### 6.2 Token加密流程

```typescript
// 1. 生成Token
const token = CryptoUtils.generateToken();

// 2. 使用主密码加密Token
const encryptedToken = await CryptoUtils.encryptData(token, masterPassword);

// 3. 存储加密后的Token
database.save(JSON.stringify(encryptedToken));
```

### 6.3 API Key加密流程

```typescript
// 1. 解密Token获取明文
const token = await CryptoUtils.decryptDataFromString(encryptedToken, masterPassword);

// 2. 使用Token加密API Key
const encryptedApiKey = await CryptoUtils.encryptData(apiKey, token);
```

## 7. 安全建议

1. **定期更新密码**: 建议用户定期更新主密码
2. **密码强度**: 强制要求至少10个字符的密码长度
3. **安全传输**: 所有敏感数据传输应使用HTTPS
4. **密钥轮换**: 考虑实现Token定期轮换机制
5. **审计日志**: 记录所有密钥操作的审计日志
6. **多因素认证**: 考虑添加2FA增强安全性

## 8. 安全改进总结

### 8.1 最新安全增强（v2.0）

1. **随机盐值保护**：主密码验证现在使用随机盐值，消除了彩虹表攻击风险
2. **全链路随机化**：从主密码到API Key的所有加密层都使用随机盐值
3. **增强存储安全**：密码数据包含独立的盐值，提升本地存储安全性

### 8.2 完整安全特性

KIMBAP Console的加密系统采用了业界标准的加密算法和最佳实践，提供了多层次的安全保护：

- **PBKDF2密钥派生**：100,000次迭代 + 随机盐值
- **AES-GCM加密**：256位密钥 + 认证保护
- **零知识架构**：服务器端不存储明文密码或未加密数据
- **防彩虹表**：每个密码使用唯一随机盐值
- **完整性验证**：所有加密数据带有认证标签

通过这些改进，系统在保持高性能和易用性的同时，显著提升了安全防护能力。
