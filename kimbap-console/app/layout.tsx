/*
 * @Author: xudada 1820064201@qq.com
 * @Date: 2025-08-21 18:58:06
 * @LastEditors: xudada 1820064201@qq.com
 * @LastEditTime: 2025-08-22 11:03:59
 * @FilePath: /kimbap-console/app/layout.tsx
 * @Description: Root layout for Kimbap Console
 */
import type { Metadata } from 'next'
import { IBM_Plex_Mono } from 'next/font/google'
import './globals.css'
import { ConditionalLayout } from '@/components/conditional-layout'
import { Toaster } from '@/components/ui/sonner'
import { ThemeProvider } from '@/components/theme-provider'

const ibmPlexMono = IBM_Plex_Mono({
  subsets: ['latin'],
  weight: ['400', '500', '600', '700'],
  variable: '--font-ibm-plex-mono',
  display: 'swap'
})

export const metadata: Metadata = {
  title: 'Kimbap Console',
  description: 'Operations console for the Kimbap platform.'
}

export default function RootLayout({
  children
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" className={ibmPlexMono.variable} suppressHydrationWarning>
      <body>
        <ThemeProvider>
          <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-blue-50/30 dark:from-slate-950 dark:via-slate-900 dark:to-blue-950/30">
            {/* Background Pattern */}
            {/* <div
              className="absolute inset-0 opacity-50"
              style={{
                backgroundImage: `url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23e2e8f0' fill-opacity='0.1'%3E%3Ccircle cx='30' cy='30' r='1'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E")`,
                backgroundSize: '60px 60px'
              }}
            />
            <div
              className="absolute inset-0 opacity-50 dark:block hidden"
              style={{
                backgroundImage: `url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23475569' fill-opacity='0.05'%3E%3Ccircle cx='30' cy='30' r='1'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E")`,
                backgroundSize: '60px 60px'
              }}
            /> */}

            <ConditionalLayout>{children}</ConditionalLayout>
            <Toaster />
          </div>
        </ThemeProvider>
      </body>
    </html>
  )
}
