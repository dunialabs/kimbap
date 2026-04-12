package commands

import "maps"

func AppleScriptRegistries() []map[string]Command {
	return []map[string]Command{
		NotesCommands(),
		CalendarCommands(),
		RemindersCommands(),
		MailCommands(),
		FinderCommands(),
		SafariCommands(),
		MessagesCommands(),
		ContactsCommands(),
		MSOfficeCommands(),
		IWorkCommands(),
		SpotifyCommands(),
		ShortcutsCommands(),
	}
}

func AllCommands() map[string]Command {
	all := make(map[string]Command)
	for _, registry := range AppleScriptRegistries() {
		maps.Copy(all, registry)
	}
	return all
}

func CommandNameSet() map[string]struct{} {
	allowed := make(map[string]struct{})
	for name := range AllCommands() {
		allowed[name] = struct{}{}
	}
	return allowed
}
