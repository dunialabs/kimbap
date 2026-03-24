/**
 * Tool Template File Storage
 * 持久化存储工具模板到本地文件
 */

import fs from 'fs';
import path from 'path';
import { ToolTemplate } from './KimbapCloudApiService';

// Use /app/data for Docker volume persistence, fallback to local path for development
const STORAGE_DIR = process.env.NODE_ENV === 'production'
  ? '/app/data'
  : path.join(process.cwd(), 'data');
const STORAGE_FILE = path.join(STORAGE_DIR, 'tool-templates.json');

/**
 * 从本地文件读取工具模板
 * @returns 工具模板数组,如果文件不存在或读取失败则返回null
 */
export function readTemplatesFromFile(): ToolTemplate[] | null {
  try {
    if (!fs.existsSync(STORAGE_FILE)) {
      return null;
    }

    const content = fs.readFileSync(STORAGE_FILE, 'utf-8');
    const templates = JSON.parse(content);

    if (!Array.isArray(templates)) {
      console.error('Invalid tool templates file format');
      return null;
    }

    return templates;
  } catch (error) {
    console.error('Failed to read tool templates from file:', error);
    return null;
  }
}

/**
 * 保存工具模板到本地文件
 * @param templates 工具模板数组
 * @returns 是否保存成功
 */
export function saveTemplatesToFile(templates: ToolTemplate[]): boolean {
  try {
    // 确保目录存在
    const dir = path.dirname(STORAGE_FILE);
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }

    // 保存到文件
    const content = JSON.stringify(templates, null, 2);
    fs.writeFileSync(STORAGE_FILE, content, 'utf-8');

    return true;
  } catch (error) {
    console.error('Failed to save tool templates to file:', error);
    return false;
  }
}

/**
 * 比较并更新本地文件中的工具模板
 * 只有当云端数据与本地文件不同时才更新
 * @param cloudTemplates 从云端获取的模板数据
 * @returns 是否有更新
 */
export function compareAndUpdateTemplates(cloudTemplates: ToolTemplate[]): boolean {
  try {
    const localTemplates = readTemplatesFromFile();

    // 如果本地没有文件,直接保存
    if (!localTemplates) {
      console.log('No local templates found, saving cloud templates to file');
      return saveTemplatesToFile(cloudTemplates);
    }

    // 比较云端和本地数据是否相同
    const cloudStr = JSON.stringify(cloudTemplates);
    const localStr = JSON.stringify(localTemplates);

    if (cloudStr === localStr) {
      // 数据相同,无需更新
      return false;
    }

    // 数据不同,更新本地文件
    console.log('Cloud templates differ from local, updating file');
    return saveTemplatesToFile(cloudTemplates);
  } catch (error) {
    console.error('Failed to compare and update templates:', error);
    return false;
  }
}
