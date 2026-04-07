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
var parsedLimit = parseInt(input.limit, 10);
var limit = isNaN(parsedLimit) || parsedLimit <= 0 ? 20 : parsedLimit;
var notes;
if (input.folder) {
	var folders = app.folders.whose({name: input.folder})();
	if (folders.length === 0) throw new Error("[NOT_FOUND] folder not found: " + input.folder);
	if (folders.length > 1) throw new Error("[AMBIGUOUS] multiple folders with name " + JSON.stringify(input.folder));
	var folderNotes = folders[0].notes;
	var folderTotal = folderNotes.length;
	var folderEnd = Math.min(limit, folderTotal);
	notes = [];
	for (var i = 0; i < folderEnd; i++) {
		notes.push(folderNotes[i]);
	}
} else {
	var total = app.notes.length;
	var end = Math.min(limit, total);
	notes = [];
	for (var i = 0; i < end; i++) {
		var n = app.notes[i];
		notes.push(n);
	}
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
var parsedLimit = parseInt(input.limit, 10);
var limit = isNaN(parsedLimit) || parsedLimit <= 0 ? 20 : parsedLimit;
var query = (input.query || "").toLowerCase();
var total = app.notes.length;
var result = [];
for (var i = 0; i < total && result.length < limit; i++) {
	var n = app.notes[i];
	var noteName = n.name();
	var body = n.plaintext();
	if (noteName.toLowerCase().indexOf(query) >= 0 ||
	    body.toLowerCase().indexOf(query) >= 0) {
		result.push({
			name: noteName,
			folder: n.container().name(),
			snippet: body.substring(0, 200)
		});
	}
}
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
  if (folders.length > 1) throw new Error("[AMBIGUOUS] multiple folders with name " + JSON.stringify(input.folder));
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
