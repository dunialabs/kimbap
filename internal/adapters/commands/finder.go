package commands

const finderPathHelpers = `
function normalizedPath(path) {
	var trimmed = String(path || "").replace(/\/+$/, "");
	return trimmed || "/";
}

function pathLocator(path) {
	var normalized = normalizedPath(path);
	if (normalized === "/") {
		return {node: app.startupDisk, parts: []};
	}
	var parts = normalized.replace(/^\/+/, "").split("/");
	if (parts[0] === "Volumes") {
		if (parts.length < 2) throw new Error("[NOT_FOUND] volume not found");
		return {node: app.disks.byName(parts[1]), parts: parts.slice(2)};
	}
	return {node: app.startupDisk, parts: parts};
}

function findFolderByPath(path) {
	var locator = pathLocator(path);
	var node = locator.node;
	for (var i = 0; i < locator.parts.length; i++) {
		var sub = node.folders.whose({name: {_equals: locator.parts[i]}})();
		if (!sub.length) throw new Error("[NOT_FOUND] folder not found");
		node = sub[0];
	}
	return node;
}

function findItemByPath(path) {
	var locator = pathLocator(path);
	var node = locator.node;
	for (var i = 0; i < locator.parts.length - 1; i++) {
		var sub = node.folders.whose({name: {_equals: locator.parts[i]}})();
		if (!sub.length) throw new Error("[NOT_FOUND] item not found");
		node = sub[0];
	}
	if (!locator.parts.length) return node;
	var name = locator.parts[locator.parts.length - 1];
	var sub = node.folders.whose({name: {_equals: name}})();
	if (sub.length) return sub[0];
	sub = node.files.whose({name: {_equals: name}})();
	if (sub.length) return sub[0];
	throw new Error("[NOT_FOUND] item not found");
}
`

func FinderCommands() map[string]Command {
	return map[string]Command{
		"finder-list-items": {
			Name: "finder-list-items", TargetApp: "Finder",
			Script: stdinReader + `
var app = Application("com.apple.finder");
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
` + finderPathHelpers + `

var folder;
try {
	folder = findFolderByPath(input.path);
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
var app = Application("com.apple.finder");
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
` + finderPathHelpers + `

var item;
try {
	item = findItemByPath(input.path);
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
var app = Application("com.apple.finder");
app.includeStandardAdditions = false;
if (!input.path) throw new Error("path is required");
if (!input.name) throw new Error("name is required");
` + finderPathHelpers + `

var container;
try {
	container = findFolderByPath(input.path);
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
var app = Application("com.apple.finder");
app.includeStandardAdditions = false;
if (!input.source_path) throw new Error("source_path is required");
if (!input.destination_path) throw new Error("destination_path is required");
` + finderPathHelpers + `

var sourceItem;
try {
	sourceItem = findItemByPath(input.source_path);
} catch (e) {
	throw new Error("[NOT_FOUND] source item not found");
}

var destinationFolder;
try {
	destinationFolder = findFolderByPath(input.destination_path);
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
var app = Application("com.apple.finder");
app.includeStandardAdditions = false;
if (!input.source_path) throw new Error("source_path is required");
if (!input.destination_path) throw new Error("destination_path is required");
` + finderPathHelpers + `

var sourceItem;
try {
	sourceItem = findItemByPath(input.source_path);
} catch (e) {
	throw new Error("[NOT_FOUND] source item not found");
}

var destinationFolder;
try {
	destinationFolder = findFolderByPath(input.destination_path);
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
var app = Application("com.apple.finder");
app.includeStandardAdditions = false;
if (!input.path) throw new Error("path is required");
` + finderPathHelpers + `

var item;
try {
	item = findItemByPath(input.path);
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
var app = Application("com.apple.finder");
app.includeStandardAdditions = false;
if (!input.path) throw new Error("path is required");
` + finderPathHelpers + `

var item;
try {
	item = findItemByPath(input.path);
} catch (e) {
	throw new Error("[NOT_FOUND] item not found");
}

app.open(item);
JSON.stringify({path: input.path, opened: true});`,
		},
	}
}
