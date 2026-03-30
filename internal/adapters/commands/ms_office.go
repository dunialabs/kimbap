package commands

func MSOfficeCommands() map[string]Command {
	return map[string]Command{
		"word-create-document": {
			Name: "word-create-document", TargetApp: "Microsoft Word",
			Script: stdinReader + `
var app = Application("Microsoft Word");
app.includeStandardAdditions = false;

var doc = app.make({new: "document"});
JSON.stringify({created: true, name: doc.name()});`,
		},
		"word-open-document": {
			Name: "word-open-document", TargetApp: "Microsoft Word",
			Script: stdinReader + `
var app = Application("Microsoft Word");
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
		"word-get-text": {
			Name: "word-get-text", TargetApp: "Microsoft Word",
			Script: stdinReader + `
var app = Application("Microsoft Word");
app.includeStandardAdditions = false;

var doc;
try {
	doc = app.activeDocument();
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active document not found");
}

JSON.stringify({
	name: doc.name(),
	text: doc.textObject().content()
});`,
		},
		"word-set-text": {
			Name: "word-set-text", TargetApp: "Microsoft Word",
			Script: stdinReader + `
var app = Application("Microsoft Word");
app.includeStandardAdditions = false;
if (typeof input.text !== "string") throw new Error("text is required");

var doc;
try {
	doc = app.activeDocument();
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active document not found");
}

doc.textObject().content = input.text;
JSON.stringify({name: doc.name(), updated: true});`,
		},
		"word-find-replace": {
			Name: "word-find-replace", TargetApp: "Microsoft Word",
			Script: stdinReader + `
var app = Application("Microsoft Word");
app.includeStandardAdditions = false;
if (typeof input.find !== "string") throw new Error("find is required");
if (input.find.length === 0) throw new Error("find must not be empty");
if (typeof input.replace !== "string") throw new Error("replace is required");

var doc;
try {
	doc = app.activeDocument();
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active document not found");
}

var source = doc.textObject().content() || "";
var flags = input.match_case ? "g" : "gi";
var escaped = input.find.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
var re = new RegExp(escaped, flags);
var matches = source.match(re);
var count = matches ? matches.length : 0;
doc.textObject().content = source.replace(re, input.replace);

JSON.stringify({name: doc.name(), replacements: count});`,
		},
		"word-save-as-pdf": {
			Name: "word-save-as-pdf", TargetApp: "Microsoft Word",
			Script: stdinReader + `
var app = Application("Microsoft Word");
app.includeStandardAdditions = false;
if (!input.output_path) throw new Error("output_path is required");

var doc;
try {
	doc = app.activeDocument();
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active document not found");
}

var exported = false;
try {
	doc.exportAsFixedFormat({outputFileName: Path(input.output_path), exportFormat: "wdExportFormatPDF"});
	exported = true;
} catch (e1) {
	try {
		doc.saveAs({fileName: Path(input.output_path), fileFormat: 17});
		exported = true;
	} catch (e2) {
		throw new Error("[NOT_SUPPORTED] failed to export Word document to PDF");
	}
}
JSON.stringify({name: doc.name(), exported: true, outputPath: input.output_path});`,
		},
		"word-close-document": {
			Name: "word-close-document", TargetApp: "Microsoft Word",
			Script: stdinReader + `
var app = Application("Microsoft Word");
app.includeStandardAdditions = false;

var doc;
try {
	doc = app.activeDocument();
	doc.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active document not found");
}

var save = !!input.save;
var docName = doc.name();
doc.close({saving: save ? "yes" : "no"});
JSON.stringify({name: docName, closed: true, saved: save});`,
		},
		"excel-create-workbook": {
			Name: "excel-create-workbook", TargetApp: "Microsoft Excel",
			Script: stdinReader + `
var app = Application("Microsoft Excel");
app.includeStandardAdditions = false;

var workbook = app.make({new: "workbook"});
JSON.stringify({created: true, name: workbook.name()});`,
		},
		"excel-open-workbook": {
			Name: "excel-open-workbook", TargetApp: "Microsoft Excel",
			Script: stdinReader + `
var app = Application("Microsoft Excel");
app.includeStandardAdditions = false;
if (!input.path) throw new Error("path is required");

var workbook;
try {
	workbook = app.open(Path(input.path));
} catch (e) {
	throw new Error("[NOT_FOUND] workbook not found");
}

JSON.stringify({opened: true, path: input.path, name: workbook.name()});`,
		},
		"excel-read-cell": {
			Name: "excel-read-cell", TargetApp: "Microsoft Excel",
			Script: stdinReader + `
var app = Application("Microsoft Excel");
app.includeStandardAdditions = false;
if (!input.cell) throw new Error("cell is required");

var workbook;
try {
	workbook = app.activeWorkbook();
	workbook.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active workbook not found");
}

var sheet = input.sheet ? workbook.worksheets.whose({name: input.sheet})()[0] : workbook.activeSheet();
if (!sheet) throw new Error("[NOT_FOUND] worksheet not found");

var value = sheet.range(input.cell).value();
JSON.stringify({workbook: workbook.name(), sheet: sheet.name(), cell: input.cell, value: value});`,
		},
		"excel-write-cell": {
			Name: "excel-write-cell", TargetApp: "Microsoft Excel",
			Script: stdinReader + `
var app = Application("Microsoft Excel");
app.includeStandardAdditions = false;
if (!input.cell) throw new Error("cell is required");
if (typeof input.value === "undefined") throw new Error("value is required");

var workbook;
try {
	workbook = app.activeWorkbook();
	workbook.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active workbook not found");
}

var sheet = input.sheet ? workbook.worksheets.whose({name: input.sheet})()[0] : workbook.activeSheet();
if (!sheet) throw new Error("[NOT_FOUND] worksheet not found");

sheet.range(input.cell).value = input.value;
JSON.stringify({workbook: workbook.name(), sheet: sheet.name(), cell: input.cell, written: true});`,
		},
		"excel-read-range": {
			Name: "excel-read-range", TargetApp: "Microsoft Excel",
			Script: stdinReader + `
var app = Application("Microsoft Excel");
app.includeStandardAdditions = false;
if (!input.range) throw new Error("range is required");

var workbook;
try {
	workbook = app.activeWorkbook();
	workbook.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active workbook not found");
}

var sheet = input.sheet ? workbook.worksheets.whose({name: input.sheet})()[0] : workbook.activeSheet();
if (!sheet) throw new Error("[NOT_FOUND] worksheet not found");

var values = sheet.range(input.range).value();
JSON.stringify({workbook: workbook.name(), sheet: sheet.name(), range: input.range, values: values});`,
		},
		"excel-save-as-pdf": {
			Name: "excel-save-as-pdf", TargetApp: "Microsoft Excel",
			Script: stdinReader + `
var app = Application("Microsoft Excel");
app.includeStandardAdditions = false;
if (!input.output_path) throw new Error("output_path is required");

var workbook;
try {
	workbook = app.activeWorkbook();
	workbook.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active workbook not found");
}

try {
	workbook.exportAsFixedFormat({type: "xlTypePDF", fileName: Path(input.output_path)});
} catch (e1) {
	try {
		workbook.exportAsFixedFormat({type: 0, fileName: Path(input.output_path)});
	} catch (e2) {
		try {
			workbook.saveAs({filename: Path(input.output_path), fileFormat: 57});
		} catch (e3) {
			throw new Error("[NOT_SUPPORTED] failed to export Excel workbook to PDF");
		}
	}
}
JSON.stringify({name: workbook.name(), exported: true, outputPath: input.output_path});`,
		},
		"excel-close-workbook": {
			Name: "excel-close-workbook", TargetApp: "Microsoft Excel",
			Script: stdinReader + `
var app = Application("Microsoft Excel");
app.includeStandardAdditions = false;

var workbook;
try {
	workbook = app.activeWorkbook();
	workbook.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active workbook not found");
}

var save = !!input.save;
var name = workbook.name();
workbook.close({saving: save ? "yes" : "no"});
JSON.stringify({name: name, closed: true, saved: save});`,
		},
		"ppt-create-presentation": {
			Name: "ppt-create-presentation", TargetApp: "Microsoft PowerPoint",
			Script: stdinReader + `
var app = Application("Microsoft PowerPoint");
app.includeStandardAdditions = false;

var pres = app.make({new: "presentation"});
JSON.stringify({created: true, name: pres.name()});`,
		},
		"ppt-open-presentation": {
			Name: "ppt-open-presentation", TargetApp: "Microsoft PowerPoint",
			Script: stdinReader + `
var app = Application("Microsoft PowerPoint");
app.includeStandardAdditions = false;
if (!input.path) throw new Error("path is required");

var pres;
try {
	pres = app.open(Path(input.path));
} catch (e) {
	throw new Error("[NOT_FOUND] presentation not found");
}

JSON.stringify({opened: true, path: input.path, name: pres.name()});`,
		},
		"ppt-add-slide": {
			Name: "ppt-add-slide", TargetApp: "Microsoft PowerPoint",
			Script: stdinReader + `
var app = Application("Microsoft PowerPoint");
app.includeStandardAdditions = false;

var pres;
try {
	pres = app.activePresentation();
	pres.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active presentation not found");
}

var slide = pres.make({new: "slide", at: pres.slides.end});
var slideIndex = pres.slides().length;

JSON.stringify({presentation: pres.name(), added: true, slideIndex: slideIndex, slideId: slide.slideID()});`,
		},
		"ppt-set-slide-text": {
			Name: "ppt-set-slide-text", TargetApp: "Microsoft PowerPoint",
			Script: stdinReader + `
var app = Application("Microsoft PowerPoint");
app.includeStandardAdditions = false;
if (typeof input.slide_index !== "number") throw new Error("slide_index is required");
if (typeof input.text !== "string") throw new Error("text is required");

var pres;
try {
	pres = app.activePresentation();
	pres.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active presentation not found");
}

var slides = pres.slides();
var slideIdx = input.slide_index - 1;
if (slideIdx < 0 || slideIdx >= slides.length) throw new Error("[NOT_FOUND] slide not found");

var slide = slides[slideIdx];
var shapeIndex = parseInt(input.shape_index, 10);
if (isNaN(shapeIndex) || shapeIndex < 1) shapeIndex = 1;
var shapes = slide.shapes();
if (shapeIndex - 1 >= shapes.length) throw new Error("[NOT_FOUND] shape not found");

var shape = shapes[shapeIndex - 1];
shape.textFrame().textRange().text = input.text;

JSON.stringify({presentation: pres.name(), slideIndex: input.slide_index, shapeIndex: shapeIndex, updated: true});`,
		},
		"ppt-save-as-pdf": {
			Name: "ppt-save-as-pdf", TargetApp: "Microsoft PowerPoint",
			Script: stdinReader + `
var app = Application("Microsoft PowerPoint");
app.includeStandardAdditions = false;
if (!input.output_path) throw new Error("output_path is required");

var pres;
try {
	pres = app.activePresentation();
	pres.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active presentation not found");
}

try {
	pres.saveAs(Path(input.output_path), {fileFormat: "PDF"});
} catch (e1) {
	try {
		pres.saveAs(Path(input.output_path), {fileFormat: 32});
	} catch (e2) {
		throw new Error("[NOT_SUPPORTED] failed to export PowerPoint presentation to PDF");
	}
}
JSON.stringify({name: pres.name(), exported: true, outputPath: input.output_path});`,
		},
		"ppt-save-as-png": {
			Name: "ppt-save-as-png", TargetApp: "Microsoft PowerPoint",
			Script: stdinReader + `
var app = Application("Microsoft PowerPoint");
app.includeStandardAdditions = false;
if (!input.output_dir) throw new Error("output_dir is required");

var pres;
try {
	pres = app.activePresentation();
	pres.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active presentation not found");
}

try {
	pres.export({path: Path(input.output_dir), filterName: "PNG"});
} catch (e1) {
	try {
		pres.saveAs(Path(input.output_dir), {fileFormat: "PNG"});
	} catch (e2) {
		throw new Error("[NOT_SUPPORTED] failed to export PowerPoint presentation to PNG");
	}
}
JSON.stringify({name: pres.name(), exported: true, outputDir: input.output_dir});`,
		},
		"ppt-close-presentation": {
			Name: "ppt-close-presentation", TargetApp: "Microsoft PowerPoint",
			Script: stdinReader + `
var app = Application("Microsoft PowerPoint");
app.includeStandardAdditions = false;

var pres;
try {
	pres = app.activePresentation();
	pres.name();
} catch (e) {
	throw new Error("[NOT_FOUND] active presentation not found");
}

var save = !!input.save;
var name = pres.name();
pres.close({saving: save ? "yes" : "no"});
JSON.stringify({name: name, closed: true, saved: save});`,
		},
	}
}
