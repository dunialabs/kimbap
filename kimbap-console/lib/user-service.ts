/**
 * User Service
 * 从本地user表获取accessToken的工具函数
 */

import { prisma } from '@/lib/prisma';

export class UserService {
  /**
   * 根据userid从本地user表获取accessToken
   * @param userid - 用户ID
   * @returns accessToken或null（如果用户不存在）
   */
  static async getAccessTokenByUserId(userid: string): Promise<string | null> {
    try {
      const user = await prisma.user.findUnique({
        where: {
          userid: userid
        },
        select: {
          accessToken: true
        }
      });

      if (!user) {
        console.warn(`User not found in local database: ${userid}`);
        return null;
      }

      return user.accessToken;
    } catch (error) {
      console.error('Failed to get access token from local database:', error);
      return null;
    }
  }

  /**
   * 检查userid是否存在于本地user表
   * @param userid - 用户ID
   * @returns 是否存在
   */
  static async userExists(userid: string): Promise<boolean> {
    try {
      const user = await prisma.user.findUnique({
        where: {
          userid: userid
        },
        select: {
          userid: true
        }
      });

      return !!user;
    } catch (error) {
      console.error('Failed to check user existence:', error);
      return false;
    }
  }
}