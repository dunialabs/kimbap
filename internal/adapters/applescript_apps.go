package adapters

import "strings"

func AppleScriptAppReference(targetApp string) string {
	switch strings.TrimSpace(targetApp) {
	case "Notes":
		return "com.apple.Notes"
	case "Calendar":
		return "com.apple.iCal"
	case "Reminders":
		return "com.apple.reminders"
	case "Mail":
		return "com.apple.mail"
	case "Contacts":
		return "com.apple.AddressBook"
	case "Safari":
		return "com.apple.Safari"
	case "Finder":
		return "com.apple.finder"
	case "Messages":
		return "com.apple.MobileSMS"
	case "Shortcuts":
		return "com.apple.shortcuts"
	default:
		return strings.TrimSpace(targetApp)
	}
}
