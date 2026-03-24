export function GlobalFooter() {
  return (
    <footer className="w-full py-4 px-6 border-t border-slate-200 dark:border-slate-800 bg-white/50 dark:bg-slate-900/50 backdrop-blur-sm">
      <div className="max-w-[1080px] mx-auto text-center text-xs text-slate-500 dark:text-slate-400">
        <span>© 2026 </span>
        <a
          href="https://kimbap.sh"
          target="_blank"
          rel="noopener noreferrer"
          className="hover:underline text-slate-600 dark:text-slate-300"
        >
          Dunia Labs, Inc.
        </a>
        <span> The control plane for MCP servers.</span>
      </div>
    </footer>
  )
}
