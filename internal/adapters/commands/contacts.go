package commands

func ContactsCommands() map[string]Command {
	return map[string]Command{
		"contacts-list": {
			Name: "contacts-list", TargetApp: "Contacts",
			Script: stdinReader + `
var app = Application("com.apple.AddressBook");
app.includeStandardAdditions = false;

function mapLabeledValues(values) {
	try {
		return values.map(function(v) {
			var label = "";
			var value = "";
			try { label = v.label(); } catch (e) {}
			try { value = v.value(); } catch (e) {}
			return {label: label, value: value};
		});
	} catch (e) {
		return [];
	}
}

function mapPerson(person, includeExtra) {
	var name = "";
	var firstName = "";
	var lastName = "";
	try { name = person.name() || ""; } catch (e) {}
	try { firstName = person.firstName() || ""; } catch (e) {}
	try { lastName = person.lastName() || ""; } catch (e) {}
	var rawEmails = [];
	var rawPhones = [];
	try { rawEmails = person.emails(); } catch (e) {}
	try { rawPhones = person.phones(); } catch (e) {}
	var result = {
		name: name,
		firstName: firstName,
		lastName: lastName,
		emails: mapLabeledValues(rawEmails),
		phones: mapLabeledValues(rawPhones)
	};

	if (includeExtra) {
		var org = "";
		var job = "";
		var notes = "";
		try { org = person.organization(); } catch (e) {}
		try { job = person.jobTitle(); } catch (e) {}
		try { notes = person.note(); } catch (e) {}
		result.organization = org;
		result.jobTitle = job;
		result.notes = notes;
	}

	return result;
}

var parsedLimit = parseInt(input.limit, 10);
var limit = isNaN(parsedLimit) || parsedLimit <= 0 ? 50 : parsedLimit;
var total = app.people.length;
var end = Math.min(limit, total);
var result = [];
for (var i = 0; i < end; i++) {
	result.push(mapPerson(app.people[i], false));
}

JSON.stringify(result);`,
		},
		"contacts-search": {
			Name: "contacts-search", TargetApp: "Contacts",
			Script: stdinReader + `
var app = Application("com.apple.AddressBook");
app.includeStandardAdditions = false;
if (!input.query) throw new Error("query is required");

function mapLabeledValues(values) {
	try {
		return values.map(function(v) {
			var label = "";
			var value = "";
			try { label = v.label(); } catch (e) {}
			try { value = v.value(); } catch (e) {}
			return {label: label, value: value};
		});
	} catch (e) {
		return [];
	}
}

function mapPerson(person) {
	var name = "";
	var firstName = "";
	var lastName = "";
	try { name = person.name() || ""; } catch (e) {}
	try { firstName = person.firstName() || ""; } catch (e) {}
	try { lastName = person.lastName() || ""; } catch (e) {}
	var rawEmails = [];
	var rawPhones = [];
	try { rawEmails = person.emails(); } catch (e) {}
	try { rawPhones = person.phones(); } catch (e) {}
	return {
		name: name,
		firstName: firstName,
		lastName: lastName,
		emails: mapLabeledValues(rawEmails),
		phones: mapLabeledValues(rawPhones)
	};
}

var q = String(input.query).toLowerCase();
var byName;
try {
	byName = app.people.whose({name: {_contains: input.query}})();
} catch (e) {
	byName = [];
}

var seenIds = {};
var result = [];

byName.forEach(function(person) {
	if (result.length >= 50) return;
	var key = "";
	try { key = person.id(); } catch (e) {}
	if (!key) {
		try { key = person.name(); } catch (e) {}
	}
	if (!key) return;
	if (seenIds[key]) return;
	seenIds[key] = true;
	result.push(mapPerson(person));
});

if (result.length < 10) {
	var total = app.people.length;
	var checked = 0;
	for (var i = 0; i < total && checked < 100 && result.length < 50; i++) {
		var person = app.people[i];
		checked++;
		var key = "";
		try { key = person.id(); } catch (e) {}
		if (!key) {
			try { key = person.name(); } catch (e) {}
		}
		if (!key || seenIds[key]) continue;

		var matched = false;
		try {
			var emails = person.emails();
			for (var j = 0; j < emails.length; j++) {
				var emailVal = "";
				try { emailVal = (emails[j].value() || "").toLowerCase(); } catch (e) {}
				if (emailVal.indexOf(q) >= 0) {
					matched = true;
					break;
				}
			}
		} catch (e) {}

		if (matched) {
			seenIds[key] = true;
			result.push(mapPerson(person));
		}
	}
}

JSON.stringify(result);`,
		},
		"contacts-get": {
			Name: "contacts-get", TargetApp: "Contacts",
			Script: stdinReader + `
var app = Application("com.apple.AddressBook");
app.includeStandardAdditions = false;
if (!input.name) throw new Error("name is required");

function mapLabeledValues(values) {
	try {
		return values.map(function(v) {
			var label = "";
			var value = "";
			try { label = v.label(); } catch (e) {}
			try { value = v.value(); } catch (e) {}
			return {label: label, value: value};
		});
	} catch (e) {
		return [];
	}
}

function mapPerson(person) {
	var name = "";
	var firstName = "";
	var lastName = "";
	var org = "";
	var job = "";
	var notes = "";
	try { name = person.name() || ""; } catch (e) {}
	try { firstName = person.firstName() || ""; } catch (e) {}
	try { lastName = person.lastName() || ""; } catch (e) {}
	try { org = person.organization(); } catch (e) {}
	try { job = person.jobTitle(); } catch (e) {}
	try { notes = person.note(); } catch (e) {}
	var rawEmails = [];
	var rawPhones = [];
	try { rawEmails = person.emails(); } catch (e) {}
	try { rawPhones = person.phones(); } catch (e) {}

	return {
		name: name,
		firstName: firstName,
		lastName: lastName,
		emails: mapLabeledValues(rawEmails),
		phones: mapLabeledValues(rawPhones),
		organization: org,
		jobTitle: job,
		notes: notes
	};
}

var targetName = String(input.name).toLowerCase();
var total = app.people.length;
var match = null;

for (var i = 0; i < total; i++) {
	var p = app.people[i];
	var name = "";
	try { name = (p.name() || "").toLowerCase(); } catch (e) {}
	if (name === targetName) {
		match = p;
		break;
	}
}

if (!match) {
	for (var j = 0; j < total; j++) {
		var p2 = app.people[j];
		var name2 = "";
		try { name2 = (p2.name() || "").toLowerCase(); } catch (e) {}
		if (name2.indexOf(targetName) >= 0) {
			match = p2;
			break;
		}
	}
}

if (!match) throw new Error("[NOT_FOUND] contact not found");
JSON.stringify(mapPerson(match));`,
		},
		"contacts-create": {
			Name: "contacts-create", TargetApp: "Contacts",
			Script: stdinReader + `
var app = Application("com.apple.AddressBook");
app.includeStandardAdditions = false;
if (!input.first_name) throw new Error("first_name is required");

var person = app.Person({
	firstName: input.first_name,
	lastName: input.last_name || ""
});

if (input.organization) {
	person.organization = input.organization;
}

app.people.push(person);

if (input.email) {
	var emailEntry = app.Email({label: "home", value: input.email});
	person.emails.push(emailEntry);
}
if (input.phone) {
	var phoneEntry = app.Phone({label: "mobile", value: input.phone});
	person.phones.push(phoneEntry);
}

try {
	app.save();
} catch (e) {
	throw new Error("Failed to save contact");
}

var name = "";
var firstName = "";
var lastName = "";
try { name = person.name(); } catch(e) {}
try { firstName = person.firstName(); } catch(e) {}
try { lastName = person.lastName(); } catch(e) {}

JSON.stringify({
	name: name,
	firstName: firstName,
	lastName: lastName
});`,
		},
	}
}
