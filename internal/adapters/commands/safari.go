package commands

func SafariCommands() map[string]Command {
	return map[string]Command{
		"safari-get-url": {
			Name: "safari-get-url", TargetApp: "Safari",
			Script: stdinReader + `
var app = Application("Safari");
app.includeStandardAdditions = false;

var windows = app.windows();
if (windows.length === 0) throw new Error("[NOT_FOUND] no safari windows open");

var tab = windows[0].currentTab();
JSON.stringify({
	url: tab.url(),
	title: tab.name()
});`,
		},
		"safari-open-url": {
			Name: "safari-open-url", TargetApp: "Safari",
			Script: stdinReader + `
var app = Application("Safari");
app.includeStandardAdditions = true;
if (!input.url) throw new Error("url is required");

var urlStr = String(input.url);
if (!/^https?:\/\//i.test(urlStr)) {
	throw new Error("url must use http or https scheme");
}
app.openLocation(urlStr);
JSON.stringify({url: urlStr, opened: true});`,
		},
		"safari-list-tabs": {
			Name: "safari-list-tabs", TargetApp: "Safari",
			Script: stdinReader + `
var app = Application("Safari");
app.includeStandardAdditions = false;

var result = [];
app.windows().forEach(function(win, wi) {
	win.tabs().forEach(function(tab, ti) {
		result.push({
			url: tab.url(),
			title: tab.name(),
			windowIndex: wi,
			tabIndex: ti
		});
	});
});

JSON.stringify(result);`,
		},
		"safari-close-tab": {
			Name: "safari-close-tab", TargetApp: "Safari",
			Script: stdinReader + `
var app = Application("Safari");
app.includeStandardAdditions = false;

if (typeof input.window_index !== "number") throw new Error("window_index is required");
if (typeof input.tab_index !== "number") throw new Error("tab_index is required");

var windows = app.windows();
if (input.window_index < 0 || input.window_index >= windows.length) {
	throw new Error("[NOT_FOUND] window not found");
}

var tabs = windows[input.window_index].tabs();
if (input.tab_index < 0 || input.tab_index >= tabs.length) {
	throw new Error("[NOT_FOUND] tab not found");
}

tabs[input.tab_index].close();
JSON.stringify({closed: true});`,
		},
		"safari-get-source": {
			Name: "safari-get-source", TargetApp: "Safari",
			Script: stdinReader + `
var app = Application("Safari");
app.includeStandardAdditions = false;

var windows = app.windows();
if (windows.length === 0) throw new Error("[NOT_FOUND] no safari windows open");

var tab = windows[0].currentTab();
var text = app.doJavaScript("document.documentElement.outerHTML", {in: tab});

JSON.stringify({
	url: tab.url(),
	html: text
});`,
		},
	}
}
