package commands

func ShortcutsCommands() map[string]Command {
	return map[string]Command{
		"shortcuts-list": {
			Name: "shortcuts-list", TargetApp: "Shortcuts",
			Script: stdinReader + `
var app = Application("com.apple.shortcuts");
app.includeStandardAdditions = false;

var parsedLimit = parseInt(input.limit, 10);
var limit = isNaN(parsedLimit) || parsedLimit <= 0 ? 50 : parsedLimit;
var total = app.shortcuts.length;
var end = Math.min(limit, total);
var result = [];
for (var i = 0; i < end; i++) {
	var sc = app.shortcuts[i];
	var folderName = null;
	try {
		var folder = sc.folder();
		folderName = folder ? folder.name() : null;
	} catch (e) {
		folderName = null;
	}
	result.push({name: sc.name(), folder: folderName});
}
JSON.stringify(result);`,
		},
		"shortcuts-run": {
			Name: "shortcuts-run", TargetApp: "Shortcuts",
			Script: stdinReader + `
var app = Application("com.apple.shortcuts");
app.includeStandardAdditions = false;

if (!input.name) throw new Error("name is required");

var name = String(input.name).trim();
if (!name) throw new Error("name is required");

var matches = app.shortcuts.whose({name: name})();
if (matches.length === 0) throw new Error("[NOT_FOUND] shortcut not found");
if (matches.length > 1) throw new Error("[AMBIGUOUS] multiple shortcuts found with name " + JSON.stringify(name) + "; use a unique name");

var shortcut = matches[0];
if (input.input !== undefined && input.input !== null) {
	shortcut.run({withInput: String(input.input)});
} else {
	shortcut.run();
}

var result = {
	name: name,
	ran: true,
	with_input: input.input !== undefined && input.input !== null
};
JSON.stringify(result);`,
		},
		"shortcuts-run-with-input": {
			Name: "shortcuts-run-with-input", TargetApp: "Shortcuts",
			Script: stdinReader + `
var app = Application("com.apple.shortcuts");
app.includeStandardAdditions = false;

if (!input.name) throw new Error("name is required");
if (input.input === undefined || input.input === null) throw new Error("input is required");

var name = String(input.name).trim();
if (!name) throw new Error("name is required");

var matches = app.shortcuts.whose({name: name})();
if (matches.length === 0) throw new Error("[NOT_FOUND] shortcut not found");
if (matches.length > 1) throw new Error("[AMBIGUOUS] multiple shortcuts found with name " + JSON.stringify(name) + "; use a unique name");

var inputStr = String(input.input);
matches[0].run({withInput: inputStr});

var result = {
	name: name,
	ran: true,
	with_input: true
};
JSON.stringify(result);`,
		},
	}
}
