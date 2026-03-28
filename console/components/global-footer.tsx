export function GlobalFooter() {
  return (
    <footer className="w-full border-t border-border bg-background/80 px-4 py-4 backdrop-blur-sm sm:px-6">
      <div className="mx-auto flex max-w-[1080px] flex-col items-center gap-1 text-center text-xs leading-5 text-muted-foreground sm:flex-row sm:justify-center">
        <span>© 2026</span>
        <a
          href="https://kimbap.sh"
          target="_blank"
          rel="noopener noreferrer"
          aria-label="Dunia Labs website (opens in new tab)"
          className="rounded-sm text-foreground/80 hover:text-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2"
        >
          Dunia Labs, Inc.
        </a>
        <span className="hidden sm:inline">·</span>
        <span>Operations console for the Kimbap platform.</span>
      </div>
    </footer>
  )
}
