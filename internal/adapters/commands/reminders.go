package commands

func RemindersCommands() map[string]Command {
	return map[string]Command{
		"list-lists": {
			Name: "list-lists", TargetApp: "Reminders",
			Script: stdinReader + `
var app = Application("com.apple.reminders");
app.includeStandardAdditions = false;
var lists = app.lists();
var result = lists.map(function(l) { return {name: l.name()}; });
JSON.stringify(result);`,
		},
		"list-reminders": {
			Name: "list-reminders", TargetApp: "Reminders",
			Script: stdinReader + `
var app = Application("com.apple.reminders");
app.includeStandardAdditions = false;
throw new Error("[NOT_SUPPORTED] list-reminders is temporarily disabled because Reminders enumeration can hang in JXA; use get-reminder with an exact name instead");`,
		},
		"get-reminder": {
			Name: "get-reminder", TargetApp: "Reminders",
			Script: stdinReader + `
var app = Application("com.apple.reminders");
app.includeStandardAdditions = false;
var matches = app.reminders.whose({name: input.name})();
if (matches.length === 0) throw new Error("[NOT_FOUND] reminder not found");
var r = matches[0];
var due = r.dueDate();
var result = {
	name: r.name(),
	completed: r.completed(),
	dueDate: due ? due.toISOString() : null,
	priority: r.priority(),
	notes: r.body(),
	list: r.container().name()
};
JSON.stringify(result);`,
		},
		"create-reminder": {
			Name: "create-reminder", TargetApp: "Reminders",
			Script: stdinReader + `
var app = Application("com.apple.reminders");
app.includeStandardAdditions = false;
var targetList = null;
if (input.list) {
	var matches = app.lists.whose({name: input.list})();
	if (matches.length === 0) throw new Error("[NOT_FOUND] list not found: " + input.list);
	targetList = matches[0];
} else {
	var allLists = app.lists();
	if (allLists.length === 0) throw new Error("no reminders list available");
	targetList = allLists[0];
}
var props = {name: input.name};
if (input.due_date) props.dueDate = new Date(input.due_date);
var parsedPriority = parseInt(input.priority, 10);
if (!isNaN(parsedPriority)) props.priority = parsedPriority;
if (input.notes) props.body = input.notes;
var reminder = app.Reminder(props);
targetList.reminders.push(reminder);
JSON.stringify({name: reminder.name(), list: targetList.name()});`,
		},
		"complete-reminder": {
			Name: "complete-reminder", TargetApp: "Reminders",
			Script: stdinReader + `
var app = Application("com.apple.reminders");
app.includeStandardAdditions = false;
var matches = app.reminders.whose({name: input.name})();
if (matches.length === 0) throw new Error("[NOT_FOUND] reminder not found");
var r = matches[0];
r.completed = true;
JSON.stringify({name: r.name(), completed: true});`,
		},
	}
}
