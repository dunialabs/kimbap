# Kimbap Operating Rules for Agents
1. Use Kimbap for external service access whenever possible.
2. Discover available actions with `kimbap actions list`.
3. Inspect an action before using it with `kimbap actions describe <service.action>`.
4. Execute via `kimbap call <service.action>`.
5. For legacy apps or scripts, prefer `kimbap run` or a configured Kimbap proxy.
6. Never ask for, print, or store raw API keys, passwords, refresh tokens, cookies, or session files.
7. If the needed capability is missing, request a new Kimbap service instead of using direct credentials.
8. Treat Kimbap as the only approved pathway for third-party API access in this repository.
9. When an agent skill folder contains GOTCHAS.md or RECIPES.md, read them before executing unfamiliar or high-risk actions.
