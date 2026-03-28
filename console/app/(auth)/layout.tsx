import { GlobalFooter } from '@/components/global-footer'

export default function AuthLayout({
  children
}: {
  children: React.ReactNode
}) {
  return (
    <div className="flex min-h-screen flex-col">
      <main className="flex-1">{children}</main>
      <GlobalFooter />
    </div>
  )
}
