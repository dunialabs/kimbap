export function GlobalFooter() {
  const year = new Date().getFullYear()

  return (
    <footer className="w-full border-t border-border bg-background/80 px-4 py-4 backdrop-blur-sm sm:px-6">
      <div className="mx-auto flex max-w-[1080px] flex-col gap-3 text-center text-xs leading-5 text-muted-foreground sm:flex-row sm:items-center sm:justify-between sm:text-left">
        <div className="space-y-1">
          <div className="flex flex-wrap items-center justify-center gap-x-2 gap-y-1 sm:justify-start">
            <span>© {year}</span>
            <span aria-hidden="true">·</span>
            <a
              href="https://kimbap.sh"
              target="_blank"
              rel="noopener noreferrer"
              aria-label="Kimbap website (opens in a new tab)"
              className="rounded-sm text-foreground/80 transition-colors duration-200 hover:text-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
            >
              Kimbap Console
            </a>
          </div>
          <p>Manage approvals, policies, logs, and usage for your Kimbap server.</p>
        </div>

        <nav aria-label="Helpful links" className="flex flex-wrap items-center justify-center gap-x-4 gap-y-1">
          <a
            href="https://kimbap.sh/quick-start"
            target="_blank"
            rel="noopener noreferrer"
            aria-label="Kimbap quick start guide (opens in a new tab)"
            className="rounded-sm text-foreground/80 transition-colors duration-200 hover:text-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            Quick start
          </a>
          <a
            href="https://docs.kimbap.sh"
            target="_blank"
            rel="noopener noreferrer"
            aria-label="Kimbap documentation (opens in a new tab)"
            className="rounded-sm text-foreground/80 transition-colors duration-200 hover:text-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
          >
            Docs
          </a>
        </nav>
      </div>
    </footer>
  )
}
