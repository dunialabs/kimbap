package commands

func CalendarCommands() map[string]Command {
	return map[string]Command{
		"list-calendars": {
			Name: "list-calendars", TargetApp: "Calendar",
			Script: stdinReader + `
var app = Application("Calendar");
app.includeStandardAdditions = false;
var calendars = app.calendars();
var result = calendars.map(function(c) {
	return {
		name: c.name(),
		writable: c.writable()
	};
});
JSON.stringify(result);`,
		},
		"list-events": {
			Name: "list-events", TargetApp: "Calendar",
			Script: stdinReader + `
var app = Application("Calendar");
app.includeStandardAdditions = false;

var start = input.start_date ? new Date(input.start_date) : new Date();
var end = input.end_date ? new Date(input.end_date) : new Date(start.getTime() + 7 * 24 * 60 * 60 * 1000);
var parsedLimit = parseInt(input.limit, 10);
var limit = (isNaN(parsedLimit) || parsedLimit <= 0) ? 100 : parsedLimit;

var calendars;
if (input.calendar) {
	calendars = app.calendars.whose({name: input.calendar})();
	if (calendars.length === 0) throw new Error("[NOT_FOUND] calendar not found: " + input.calendar);
} else {
	calendars = app.calendars();
	if (calendars.length > 1) throw new Error("[NOT_SUPPORTED] list-events across multiple calendars is too slow; specify --calendar");
}

var result = [];
outer: for (var i = 0; i < calendars.length; i++) {
	if (result.length >= limit) break;
	var c = calendars[i];
	var calName = c.name();
	var events = c.events.whose({
		startDate: {_lessThan: end},
		endDate: {_greaterThan: start}
	})();
	for (var j = 0; j < events.length; j++) {
		if (result.length >= limit) break outer;
		var e = events[j];
		var eventStart = e.startDate();
		var eventEnd = e.endDate();
		result.push({
			title: e.summary(),
			startDate: eventStart ? eventStart.toISOString() : null,
			endDate: eventEnd ? eventEnd.toISOString() : null,
			location: e.location(),
			notes: e.description(),
			calendar: calName
		});
	}
}

JSON.stringify(result);`,
		},
		"get-event": {
			Name: "get-event", TargetApp: "Calendar",
			Script: stdinReader + `
var app = Application("Calendar");
app.includeStandardAdditions = false;

var calendars = app.calendars();
var matches = [];
var checked = 0;
var limit = 500;

for (var i = 0; i < calendars.length; i++) {
	if (checked >= limit) break;
	var c = calendars[i];
	var calName = c.name();
	var events = c.events();
	for (var j = 0; j < events.length && checked < limit; j++) {
		checked++;
		var e = events[j];
		if (e.summary() === input.title) {
			matches.push({
				title: e.summary(),
				startDate: e.startDate() ? e.startDate().toISOString() : null,
				endDate: e.endDate() ? e.endDate().toISOString() : null,
				location: e.location(),
				notes: e.description(),
				calendar: calName
			});
		}
	}
}

if (matches.length === 0) throw new Error("[NOT_FOUND] event not found");
if (matches.length > 1) throw new Error("[AMBIGUOUS] multiple events with title " + JSON.stringify(input.title));
var result = matches[0];
JSON.stringify(result);`,
		},
		"create-event": {
			Name: "create-event", TargetApp: "Calendar",
			Script: stdinReader + `
var app = Application("Calendar");
app.includeStandardAdditions = false;

var calendars;
if (input.calendar) {
	calendars = app.calendars.whose({name: input.calendar})();
	if (calendars.length === 0) throw new Error("[NOT_FOUND] calendar not found");
} else {
	calendars = app.calendars().filter(function(c) { return c.writable(); });
	if (calendars.length === 0) throw new Error("no writable calendar available");
}

var targetCalendar = calendars[0];
var startDate = new Date(input.start_date);
var endDate = new Date(input.end_date);
var event = app.Event({
	summary: input.title,
	startDate: startDate,
	endDate: endDate,
	location: input.location || "",
	description: input.notes || ""
});
targetCalendar.events.push(event);

var result = {
	title: input.title,
	calendar: targetCalendar.name()
};

JSON.stringify(result);`,
		},
		"search-events": {
			Name: "search-events", TargetApp: "Calendar",
			Script: stdinReader + `
var app = Application("Calendar");
app.includeStandardAdditions = false;

var query = (input.query || "").toLowerCase();
var calendars = app.calendars();
var result = [];
var checked = 0;
var limit = 500;

for (var i = 0; i < calendars.length && checked < limit; i++) {
	var c = calendars[i];
	var calName = c.name();
	var events = c.events();
	for (var j = 0; j < events.length && checked < limit; j++) {
		checked++;
		var e = events[j];
		var title = (e.summary() || "");
		var notes = (e.description() || "");
		if (title.toLowerCase().indexOf(query) >= 0 || notes.toLowerCase().indexOf(query) >= 0) {
			result.push({
				title: title,
				startDate: e.startDate() ? e.startDate().toISOString() : null,
				endDate: e.endDate() ? e.endDate().toISOString() : null,
				location: e.location(),
				notes: notes,
				calendar: calName
			});
		}
	}
}

JSON.stringify(result);`,
		},
	}
}
