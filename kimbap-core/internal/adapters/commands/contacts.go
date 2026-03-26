package commands

func ContactsCommands() map[string]Command {
	return map[string]Command{
		"contacts-list": {
			Name: "contacts-list", TargetApp: "Contacts",
			Script: stdinReader + `
var app = Application("Contacts");
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
	var result = {
		name: person.name(),
		firstName: person.firstName(),
		lastName: person.lastName(),
		emails: mapLabeledValues(person.emails()),
		phones: mapLabeledValues(person.phones())
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
var people = app.people();
var result = people.slice(0, limit).map(function(person) {
	return mapPerson(person, false);
});

JSON.stringify(result);`,
		},
		"contacts-search": {
			Name: "contacts-search", TargetApp: "Contacts",
			Script: stdinReader + `
var app = Application("Contacts");
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
	return {
		name: person.name(),
		firstName: person.firstName(),
		lastName: person.lastName(),
		emails: mapLabeledValues(person.emails()),
		phones: mapLabeledValues(person.phones())
	};
}

var q = String(input.query).toLowerCase();
var people = app.people();

var filtered = people.filter(function(person) {
	var name = "";
	try { name = (person.name() || "").toLowerCase(); } catch (e) {}
	if (name.indexOf(q) >= 0) return true;

	try {
		var emails = person.emails();
		for (var i = 0; i < emails.length; i++) {
			var emailVal = "";
			try { emailVal = (emails[i].value() || "").toLowerCase(); } catch (e) {}
			if (emailVal.indexOf(q) >= 0) return true;
		}
	} catch (e) {}

	return false;
});

JSON.stringify(filtered.slice(0, 50).map(mapPerson));`,
		},
		"contacts-get": {
			Name: "contacts-get", TargetApp: "Contacts",
			Script: stdinReader + `
var app = Application("Contacts");
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
	var org = "";
	var job = "";
	var notes = "";
	try { org = person.organization(); } catch (e) {}
	try { job = person.jobTitle(); } catch (e) {}
	try { notes = person.note(); } catch (e) {}

	return {
		name: person.name(),
		firstName: person.firstName(),
		lastName: person.lastName(),
		emails: mapLabeledValues(person.emails()),
		phones: mapLabeledValues(person.phones()),
		organization: org,
		jobTitle: job,
		notes: notes
	};
}

var targetName = String(input.name).toLowerCase();
var people = app.people();
var match = null;

for (var i = 0; i < people.length; i++) {
	var p = people[i];
	var name = "";
	try { name = (p.name() || "").toLowerCase(); } catch (e) {}
	if (name === targetName) {
		match = p;
		break;
	}
}

if (!match) {
	for (var j = 0; j < people.length; j++) {
		var p2 = people[j];
		var name2 = "";
		try { name2 = (p2.name() || "").toLowerCase(); } catch (e) {}
		if (name2.indexOf(targetName) >= 0) {
			match = p2;
			break;
		}
	}
}

if (!match) throw new Error("[NOT_FOUND] contact not found: " + input.name);
JSON.stringify(mapPerson(match));`,
		},
		"contacts-create": {
			Name: "contacts-create", TargetApp: "Contacts",
			Script: stdinReader + `
var app = Application("Contacts");
app.includeStandardAdditions = false;
if (!input.first_name) throw new Error("first_name is required");

var person = app.Person({
	firstName: input.first_name,
	lastName: input.last_name || ""
});

if (input.email) {
	person.emails.push(app.Email({label: "home", value: input.email}));
}

if (input.phone) {
	person.phones.push(app.Phone({label: "mobile", value: input.phone}));
}

if (input.organization) {
	person.organization = input.organization;
}

app.people.push(person);
app.save();

JSON.stringify({
	name: person.name(),
	firstName: person.firstName(),
	lastName: person.lastName()
});`,
		},
	}
}
