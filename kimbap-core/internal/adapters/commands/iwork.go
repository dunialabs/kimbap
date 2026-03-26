package commands

func IWorkCommands() map[string]Command {
	return map[string]Command{
		"keynote-create-presentation": {
			Name: "keynote-create-presentation", TargetApp: "Keynote",
			Script: stdinReader + `
var app = Application("Keynote");
app.includeStandardAdditions = false;

var doc = app.Document();
app.documents.push(doc);
JSON.stringify({created: true, name: doc.name()});`,
		},
		"keynote-open-presentation": {
			Name: "keynote-open-presentation", TargetApp: "Keynote",
			Script: stdinReader + `
var app = Application("Keynote");
app.includeStandardAdditions = false;
if (!input.path) throw new Error("path is required");

var doc;
try {
	doc = app.open(Path(input.path));
} catch (e) {
	throw new Error("[NOT_FOUND] presentation not found");
}

JSON.stringify({opened: true, path: input.path, name: doc.name()});`,
		},
		"keynote-add-slide": {
			Name: "keynote-add-slide", TargetApp: "Keynote",
			Script: stdinReader + `
var app = Application("Keynote");
app.includeStandardAdditions = false;

var doc;
try {
	doc = app.documents[0];
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active presentation not found");
}

var masterSlides = doc.masterSlides();
var layoutProvided = typeof input.layout === "string" && input.layout.length > 0;
var layoutName = layoutProvided ? input.layout : "Blank";
var selectedMaster = null;
for (var i = 0; i < masterSlides.length; i++) {
	if (masterSlides[i].name() === layoutName) {
		selectedMaster = masterSlides[i];
		break;
	}
}
if (!selectedMaster && layoutProvided) {
	throw new Error("[NOT_FOUND] layout not found");
}
if (!selectedMaster && masterSlides.length > 0) selectedMaster = masterSlides[0];

var slide = app.Slide({baseSlide: selectedMaster});
doc.slides.push(slide);

JSON.stringify({presentation: doc.name(), added: true, slideNumber: doc.slides().length});`,
		},
		"keynote-set-slide-text": {
			Name: "keynote-set-slide-text", TargetApp: "Keynote",
			Script: stdinReader + `
var app = Application("Keynote");
app.includeStandardAdditions = false;
if (typeof input.slide_number !== "number") throw new Error("slide_number is required");
if (typeof input.text !== "string") throw new Error("text is required");

var doc;
try {
	doc = app.documents[0];
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active presentation not found");
}

var slides = doc.slides();
var slideIdx = input.slide_number - 1;
if (slideIdx < 0 || slideIdx >= slides.length) throw new Error("[NOT_FOUND] slide not found");

var slide = slides[slideIdx];
var itemIndex = parseInt(input.item_index, 10);
if (isNaN(itemIndex) || itemIndex < 1) itemIndex = 1;
var textItems = slide.defaultTitleItem() ? [slide.defaultTitleItem()].concat(slide.defaultBodyItem() ? [slide.defaultBodyItem()] : []) : slide.textItems();
if (itemIndex - 1 >= textItems.length) throw new Error("[NOT_FOUND] text item not found");

textItems[itemIndex - 1].objectText = input.text;
JSON.stringify({presentation: doc.name(), slideNumber: input.slide_number, itemIndex: itemIndex, updated: true});`,
		},
		"keynote-export-pdf": {
			Name: "keynote-export-pdf", TargetApp: "Keynote",
			Script: stdinReader + `
var app = Application("Keynote");
app.includeStandardAdditions = false;
if (!input.output_path) throw new Error("output_path is required");

var doc;
try {
	doc = app.documents[0];
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active presentation not found");
}

app.export(doc, {to: Path(input.output_path), as: "PDF"});
JSON.stringify({name: doc.name(), exported: true, outputPath: input.output_path});`,
		},
		"keynote-start-slideshow": {
			Name: "keynote-start-slideshow", TargetApp: "Keynote",
			Script: stdinReader + `
var app = Application("Keynote");
app.includeStandardAdditions = false;

var doc;
try {
	doc = app.documents[0];
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active presentation not found");
}

app.play(doc);
JSON.stringify({name: doc.name(), slideshowStarted: true});`,
		},
		"keynote-close-presentation": {
			Name: "keynote-close-presentation", TargetApp: "Keynote",
			Script: stdinReader + `
var app = Application("Keynote");
app.includeStandardAdditions = false;

var doc;
try {
	doc = app.documents[0];
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active presentation not found");
}

var save = !!input.save;
var name = doc.name();
doc.close({saving: save ? "yes" : "no"});
JSON.stringify({name: name, closed: true, saved: save});`,
		},
		"numbers-create-spreadsheet": {
			Name: "numbers-create-spreadsheet", TargetApp: "Numbers",
			Script: stdinReader + `
var app = Application("Numbers");
app.includeStandardAdditions = false;

var doc = app.Document();
app.documents.push(doc);
JSON.stringify({created: true, name: doc.name()});`,
		},
		"numbers-open-spreadsheet": {
			Name: "numbers-open-spreadsheet", TargetApp: "Numbers",
			Script: stdinReader + `
var app = Application("Numbers");
app.includeStandardAdditions = false;
if (!input.path) throw new Error("path is required");

var doc;
try {
	doc = app.open(Path(input.path));
} catch (e) {
	throw new Error("[NOT_FOUND] spreadsheet not found");
}

JSON.stringify({opened: true, path: input.path, name: doc.name()});`,
		},
		"numbers-read-cell": {
			Name: "numbers-read-cell", TargetApp: "Numbers",
			Script: stdinReader + `
var app = Application("Numbers");
app.includeStandardAdditions = false;
if (!input.cell) throw new Error("cell is required");

var doc;
try {
	doc = app.documents[0];
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active spreadsheet not found");
}

var sheetName = input.sheet_name || null;
var sheet = sheetName ? doc.sheets.whose({name: sheetName})()[0] : doc.sheets()[0];
if (!sheet) throw new Error("[NOT_FOUND] sheet not found");

var table = sheet.tables()[0];
if (!table) throw new Error("[NOT_FOUND] table not found");

var value;
try {
	value = table.ranges[input.cell].cells[0].value();
} catch (e) {
	throw new Error("[NOT_FOUND] cell not found");
}
JSON.stringify({spreadsheet: doc.name(), sheet: sheet.name(), cell: input.cell, value: value});`,
		},
		"numbers-write-cell": {
			Name: "numbers-write-cell", TargetApp: "Numbers",
			Script: stdinReader + `
var app = Application("Numbers");
app.includeStandardAdditions = false;
if (!input.cell) throw new Error("cell is required");
if (typeof input.value === "undefined") throw new Error("value is required");

var doc;
try {
	doc = app.documents[0];
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active spreadsheet not found");
}

var sheetName = input.sheet_name || null;
var sheet = sheetName ? doc.sheets.whose({name: sheetName})()[0] : doc.sheets()[0];
if (!sheet) throw new Error("[NOT_FOUND] sheet not found");

var table = sheet.tables()[0];
if (!table) throw new Error("[NOT_FOUND] table not found");

try {
	table.ranges[input.cell].cells[0].value = input.value;
} catch (e) {
	throw new Error("[NOT_FOUND] cell not found");
}

JSON.stringify({spreadsheet: doc.name(), sheet: sheet.name(), cell: input.cell, written: true});`,
		},
		"numbers-export-pdf": {
			Name: "numbers-export-pdf", TargetApp: "Numbers",
			Script: stdinReader + `
var app = Application("Numbers");
app.includeStandardAdditions = false;
if (!input.output_path) throw new Error("output_path is required");

var doc;
try {
	doc = app.documents[0];
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active spreadsheet not found");
}

app.export(doc, {to: Path(input.output_path), as: "PDF"});
JSON.stringify({name: doc.name(), exported: true, outputPath: input.output_path});`,
		},
		"numbers-close-spreadsheet": {
			Name: "numbers-close-spreadsheet", TargetApp: "Numbers",
			Script: stdinReader + `
var app = Application("Numbers");
app.includeStandardAdditions = false;

var doc;
try {
	doc = app.documents[0];
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active spreadsheet not found");
}

var save = !!input.save;
var name = doc.name();
doc.close({saving: save ? "yes" : "no"});
JSON.stringify({name: name, closed: true, saved: save});`,
		},
		"pages-create-document": {
			Name: "pages-create-document", TargetApp: "Pages",
			Script: stdinReader + `
var app = Application("Pages");
app.includeStandardAdditions = false;

var doc = app.Document();
app.documents.push(doc);
JSON.stringify({created: true, name: doc.name()});`,
		},
		"pages-open-document": {
			Name: "pages-open-document", TargetApp: "Pages",
			Script: stdinReader + `
var app = Application("Pages");
app.includeStandardAdditions = false;
if (!input.path) throw new Error("path is required");

var doc;
try {
	doc = app.open(Path(input.path));
} catch (e) {
	throw new Error("[NOT_FOUND] document not found");
}

JSON.stringify({opened: true, path: input.path, name: doc.name()});`,
		},
		"pages-get-text": {
			Name: "pages-get-text", TargetApp: "Pages",
			Script: stdinReader + `
var app = Application("Pages");
app.includeStandardAdditions = false;

var doc;
try {
	doc = app.documents[0];
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active document not found");
}

JSON.stringify({name: doc.name(), text: doc.bodyText()});`,
		},
		"pages-set-text": {
			Name: "pages-set-text", TargetApp: "Pages",
			Script: stdinReader + `
var app = Application("Pages");
app.includeStandardAdditions = false;
if (typeof input.text !== "string") throw new Error("text is required");

var doc;
try {
	doc = app.documents[0];
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active document not found");
}

doc.bodyText = input.text;
JSON.stringify({name: doc.name(), updated: true});`,
		},
		"pages-export-pdf": {
			Name: "pages-export-pdf", TargetApp: "Pages",
			Script: stdinReader + `
var app = Application("Pages");
app.includeStandardAdditions = false;
if (!input.output_path) throw new Error("output_path is required");

var doc;
try {
	doc = app.documents[0];
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active document not found");
}

app.export(doc, {to: Path(input.output_path), as: "PDF"});
JSON.stringify({name: doc.name(), exported: true, outputPath: input.output_path});`,
		},
		"pages-export-word": {
			Name: "pages-export-word", TargetApp: "Pages",
			Script: stdinReader + `
var app = Application("Pages");
app.includeStandardAdditions = false;
if (!input.output_path) throw new Error("output_path is required");

var doc;
try {
	doc = app.documents[0];
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active document not found");
}

app.export(doc, {to: Path(input.output_path), as: "Microsoft Word"});
JSON.stringify({name: doc.name(), exported: true, outputPath: input.output_path});`,
		},
		"pages-close-document": {
			Name: "pages-close-document", TargetApp: "Pages",
			Script: stdinReader + `
var app = Application("Pages");
app.includeStandardAdditions = false;

var doc;
try {
	doc = app.documents[0];
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active document not found");
}

var save = !!input.save;
var name = doc.name();
doc.close({saving: save ? "yes" : "no"});
JSON.stringify({name: name, closed: true, saved: save});`,
		},
	}
}
