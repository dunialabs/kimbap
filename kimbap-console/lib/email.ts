/*
 * @Author: xudada 1820064201@qq.com
 * @Date: 2025-08-12 11:31:46
 * @LastEditors: xudada 1820064201@qq.com
 * @LastEditTime: 2025-08-12 12:06:44
 * @FilePath: /kimbap-console/lib/email.ts
 * @Description: 这是默认设置,请设置`customMade`, 打开koroFileHeader查看配置 进行设置: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%AE
 */
import nodemailer from 'nodemailer';

// Configure email transporter
// In production, use real SMTP settings
function createTransporter() {
  const host = process.env.SMTP_HOST;
  const user = process.env.SMTP_USER;
  const pass = process.env.SMTP_PASS;
  if (!host || !user || !pass) {
    return null;
  }
  return nodemailer.createTransport({
    host,
    port: parseInt(process.env.SMTP_PORT || '587'),
    secure: false,
    auth: { user, pass },
  });
}

const transporter = createTransporter();

export const sendVerificationCode = async (email: string, code: string) => {
  // In development, just log the code
  if (process.env.NODE_ENV === 'development') {
    console.log(`[DEV] Verification code for ${email}: ${code}`);
    return true;
  }

  if (!transporter) {
    console.error('[EMAIL] SMTP not configured (SMTP_HOST, SMTP_USER, SMTP_PASS required)');
    return false;
  }

  try {
    await transporter!.sendMail({
      from: process.env.SMTP_FROM || '"KIMBAP Console" <noreply@kimbap.io>',
      to: email,
      subject: 'Your KIMBAP Console verification code',
      html: `
        <div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
          <h2 style="color: #333;">KIMBAP Console Login</h2>
          <p>Your verification code is:</p>
          <h1 style="color: #000; letter-spacing: 5px; text-align: center; background: #f4f4f4; padding: 20px; border-radius: 8px;">
            ${code}
          </h1>
          <p>This code will expire in 5 minutes.</p>
          <p style="color: #666; font-size: 12px;">If you didn't request this code, please ignore this email.</p>
        </div>
      `,
    });
    return true;
  } catch (error) {
    console.error('Failed to send email:', error);
    return false;
  }
};
