package commands

func FinderCommands() map[string]Command {
	return map[string]Command{
		"finder-list-items": {
			Name: "finder-list-items", TargetApp: "Finder",
			Script: stdinReader + `
var app = Application("Finder");
app.includeStandardAdditions = false;
if (!input.path) throw new Error("path is required");

function safeISODate(item, prop) {
	try {
		var v = item[prop]();
		return v ? v.toISOString() : null;
	} catch (e) {
		return null;
	}
}

function safeSize(item) {
	try { return item.size(); } catch (e) { return 0; }
}

function decodeItemPath(item) {
	try {
		var url = item.url();
		if (!url) return null;
		var path = url.replace(/^file:\/\/(localhost)?/, "");
		return decodeURIComponent(path);
	} catch (e) {
		return null;
	}
}

function isFolder(item) {
	try { return item.class() === "folder"; } catch (e) { return false; }
}

var folder;
try {
	folder = app.folders[Path(input.path)];
	folder.name();
} catch (e) {
	throw new Error("[NOT_FOUND] folder not found");
}

var result = folder.items().map(function(item) {
	return {
		name: item.name(),
		path: decodeItemPath(item),
		kind: item.kind(),
		size: safeSize(item),
		modifiedDate: safeISODate(item, "modificationDate"),
		isFolder: isFolder(item)
	};
});

JSON.stringify(result);`,
		},
		"finder-get-info": {
			Name: "finder-get-info", TargetApp: "Finder",
			Script: stdinReader + `
var app = Application("Finder");
app.includeStandardAdditions = false;
if (!input.path) throw new Error("path is required");

function safeISODate(item, prop) {
	try {
		var v = item[prop]();
		return v ? v.toISOString() : null;
	} catch (e) {
		return null;
	}
}

function safeSize(item) {
	try { return item.size(); } catch (e) { return 0; }
}

function decodeItemPath(item) {
	try {
		var url = item.url();
		if (!url) return null;
		var path = url.replace(/^file:\/\/(localhost)?/, "");
		return decodeURIComponent(path);
	} catch (e) {
		return null;
	}
}

function isFolder(item) {
	try { return item.class() === "folder"; } catch (e) { return false; }
}

var item;
try {
	item = app.items[Path(input.path)];
	item.name();
} catch (e) {
	throw new Error("[NOT_FOUND] item not found");
}

var result = {
	name: item.name(),
	path: decodeItemPath(item),
	kind: item.kind(),
	size: safeSize(item),
	modifiedDate: safeISODate(item, "modificationDate"),
	creationDate: safeISODate(item, "creationDate"),
	isFolder: isFolder(item)
};

JSON.stringify(result);`,
		},
		"finder-create-folder": {
			Name: "finder-create-folder", TargetApp: "Finder",
			Script: stdinReader + `
var app = Application("Finder");
app.includeStandardAdditions = false;
if (!input.path) throw new Error("path is required");
if (!input.name) throw new Error("name is required");

var container;
try {
	container = app.folders[Path(input.path)];
	container.name();
} catch (e) {
	throw new Error("[NOT_FOUND] folder not found");
}

var created = app.make({
	new: "folder",
	at: container,
	withProperties: {name: input.name}
});

var createdPath = null;
try {
	createdPath = decodeURIComponent(created.url().replace(/^file:\/\/(localhost)?/, ""));
} catch (e) {
	createdPath = input.path.replace(/\/$/, "") + "/" + input.name;
}

JSON.stringify({
	name: created.name(),
	path: createdPath
});`,
		},
		"finder-move-item": {
			Name: "finder-move-item", TargetApp: "Finder",
			Script: stdinReader + `
var app = Application("Finder");
app.includeStandardAdditions = false;
if (!input.source_path) throw new Error("source_path is required");
if (!input.destination_path) throw new Error("destination_path is required");

var sourceItem;
try {
	sourceItem = app.items[Path(input.source_path)];
	sourceItem.name();
} catch (e) {
	throw new Error("[NOT_FOUND] source item not found");
}

var destinationFolder;
try {
	destinationFolder = app.folders[Path(input.destination_path)];
	destinationFolder.name();
} catch (e) {
	throw new Error("[NOT_FOUND] destination folder not found");
}

var itemName = sourceItem.name();
var moved = app.move(sourceItem, {to: destinationFolder});
var movedPath = null;
try {
	movedPath = decodeURIComponent(moved.url().replace(/^file:\/\/(localhost)?/, ""));
} catch (e) {
	movedPath = input.destination_path.replace(/\/$/, "") + "/" + itemName;
}

JSON.stringify({
	name: itemName,
	path: movedPath
});`,
		},
		"finder-copy-item": {
			Name: "finder-copy-item", TargetApp: "Finder",
			Script: stdinReader + `
var app = Application("Finder");
app.includeStandardAdditions = false;
if (!input.source_path) throw new Error("source_path is required");
if (!input.destination_path) throw new Error("destination_path is required");

var sourceItem;
try {
	sourceItem = app.items[Path(input.source_path)];
	sourceItem.name();
} catch (e) {
	throw new Error("[NOT_FOUND] source item not found");
}

var destinationFolder;
try {
	destinationFolder = app.folders[Path(input.destination_path)];
	destinationFolder.name();
} catch (e) {
	throw new Error("[NOT_FOUND] destination folder not found");
}

var itemName = sourceItem.name();
var copied = app.duplicate(sourceItem, {to: destinationFolder});
var copiedPath = null;
try {
	copiedPath = decodeURIComponent(copied.url().replace(/^file:\/\/(localhost)?/, ""));
} catch (e) {
	copiedPath = input.destination_path.replace(/\/$/, "") + "/" + itemName;
}

JSON.stringify({
	name: itemName,
	path: copiedPath
});`,
		},
		"finder-delete-item": {
			Name: "finder-delete-item", TargetApp: "Finder",
			Script: stdinReader + `
var app = Application("Finder");
app.includeStandardAdditions = false;
if (!input.path) throw new Error("path is required");

var item;
try {
	item = app.items[Path(input.path)];
	item.name();
} catch (e) {
	throw new Error("[NOT_FOUND] item not found");
}

var name = item.name();
app.delete(item);
JSON.stringify({name: name, deleted: true});`,
		},
		"finder-open-item": {
			Name: "finder-open-item", TargetApp: "Finder",
			Script: stdinReader + `
var app = Application("Finder");
app.includeStandardAdditions = false;
if (!input.path) throw new Error("path is required");

var item;
try {
	item = app.items[Path(input.path)];
	item.name();
} catch (e) {
	throw new Error("[NOT_FOUND] item not found");
}

app.open(item);
JSON.stringify({path: input.path, opened: true});`,
		},
	}
}
