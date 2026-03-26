package commands

func MessagesCommands() map[string]Command {
	return map[string]Command{
		"messages-send": {
			Name: "messages-send", TargetApp: "Messages",
			Script: stdinReader + `
var app = Application("Messages");
app.includeStandardAdditions = false;

if (!input.to) throw new Error("to is required");
if (!input.body) throw new Error("body is required");

var target = null;
var services = app.services();
for (var i = 0; i < services.length; i++) {
	var service = services[i];
	var buddies = service.buddies.whose({handle: input.to})();
	if (buddies.length > 0) {
		target = buddies[0];
		break;
	}
}

if (!target) {
	throw new Error("[NOT_FOUND] contact not found: " + input.to);
}

app.send(input.body, {to: target});
JSON.stringify({to: input.to, sent: true});`,
		},
		"messages-list-chats": {
			Name: "messages-list-chats", TargetApp: "Messages",
			Script: stdinReader + `
var app = Application("Messages");
app.includeStandardAdditions = false;

var parsedLimit = parseInt(input.limit, 10);
var limit = isNaN(parsedLimit) || parsedLimit <= 0 ? 10 : parsedLimit;

var chats = app.chats();
var result = chats.slice(0, limit).map(function(chat) {
	var participants = [];
	var lastMessage = null;

	try {
		participants = chat.participants().map(function(p) {
			var name = "";
			var handle = "";
			try { name = p.name(); } catch (e) {}
			try { handle = p.handle(); } catch (e) {}
			return {name: name, handle: handle};
		});
	} catch (e) {
		participants = [];
	}

	try {
		var messages = chat.messages();
		if (messages.length > 0) {
			var m = messages[messages.length - 1];
			try { lastMessage = m.text(); } catch (e1) {
				try { lastMessage = m.body(); } catch (e2) {
					lastMessage = null;
				}
			}
		}
	} catch (e) {
		lastMessage = null;
	}

	var displayName = "";
	try { displayName = chat.displayName(); } catch (e) {}

	return {
		id: chat.id(),
		participants: participants,
		displayName: displayName,
		lastMessage: lastMessage
	};
});

JSON.stringify(result);`,
		},
	}
}
