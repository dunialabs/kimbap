package commands

// Command represents a built-in AppleScript/JXA command.
type Command struct {
	Name      string
	TargetApp string
	Script    string // Complete JXA script
}

// stdinReader is the JXA preamble that reads JSON from stdin.
const stdinReader = `ObjC.import('stdlib');
ObjC.import('Foundation');
var stdin = $.NSFileHandle.fileHandleWithStandardInput;
var data = stdin.readDataToEndOfFile;
var str = $.NSString.alloc.initWithDataEncoding(data, $.NSUTF8StringEncoding).js;
var input = str.length > 0 ? JSON.parse(str) : {};`

// NotesCommands returns the command registry for Apple Notes.
func NotesCommands() map[string]Command {
	return map[string]Command{
		"list-folders": {
			Name: "list-folders", TargetApp: "Notes",
			Script: stdinReader + `
var app = Application("Notes");
app.includeStandardAdditions = false;
var folders = app.folders();
var result = folders.map(function(f) { return {name: f.name()}; });
JSON.stringify(result);`,
		},
		"list-notes": {
			Name: "list-notes", TargetApp: "Notes",
			Script: stdinReader + `
var app = Application("Notes");
app.includeStandardAdditions = false;
var notes;
if (input.folder) {
	var folders = app.folders.whose({name: input.folder})();
	notes = folders.length > 0 ? folders[0].notes() : [];
} else {
	notes = app.notes();
}
var result = notes.map(function(n) {
	return {
		name: n.name(),
		folder: n.container().name(),
		snippet: n.plaintext().substring(0, 200),
		modifiedDate: n.modificationDate().toISOString()
	};
});
JSON.stringify(result);`,
		},
		"get-note": {
			Name: "get-note", TargetApp: "Notes",
			Script: stdinReader + `
var app = Application("Notes");
app.includeStandardAdditions = false;
var matches = app.notes.whose({name: input.name})();
if (matches.length === 0) throw new Error("[NOT_FOUND] note not found");
if (matches.length > 1) throw new Error("[AMBIGUOUS] multiple notes with name " + JSON.stringify(input.name) + "; specify folder");
var n = matches[0];
var result = {
	name: n.name(),
	folder: n.container().name(),
	body: n.plaintext(),
	creationDate: n.creationDate().toISOString(),
	modifiedDate: n.modificationDate().toISOString()
};
JSON.stringify(result);`,
		},
		"search-notes": {
			Name: "search-notes", TargetApp: "Notes",
			Script: stdinReader + `
var app = Application("Notes");
app.includeStandardAdditions = false;
var all = app.notes();
var query = (input.query || "").toLowerCase();
var result = all.filter(function(n) {
	return n.name().toLowerCase().indexOf(query) >= 0 ||
	       n.plaintext().toLowerCase().indexOf(query) >= 0;
}).map(function(n) {
	return {
		name: n.name(),
		folder: n.container().name(),
		snippet: n.plaintext().substring(0, 200)
	};
});
JSON.stringify(result);`,
		},
		"create-note": {
			Name: "create-note", TargetApp: "Notes",
			Script: stdinReader + `
var app = Application("Notes");
app.includeStandardAdditions = false;
var targetFolder;
if (input.folder) {
  var folders = app.folders.whose({name: input.folder})();
  if (folders.length === 0) throw new Error("[NOT_FOUND] folder not found: " + input.folder);
  targetFolder = folders[0];
} else {
  targetFolder = app.defaultAccount().defaultFolder();
}
var note = app.Note({name: input.title, body: input.body});
targetFolder.notes.push(note);
JSON.stringify({name: input.title, folder: targetFolder.name()});`,
		},
	}
}
