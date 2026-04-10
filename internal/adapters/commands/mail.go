package commands

func MailCommands() map[string]Command {
	return map[string]Command{
		"list-mailboxes": {
			Name: "list-mailboxes", TargetApp: "Mail",
			Script: stdinReader + `
var app = Application("com.apple.mail");
app.includeStandardAdditions = false;
function flattenMailboxes(boxes, accountName, result) {
	boxes.forEach(function(box) {
		var unreadCount = 0;
		try { unreadCount = box.unreadCount(); } catch (e) {}
		result.push({name: box.name(), accountName: accountName, unreadCount: unreadCount});
		try { flattenMailboxes(box.mailboxes(), accountName, result); } catch (e) {}
	});
}
var accounts = app.accounts();
var result = [];
accounts.forEach(function(account) {
	flattenMailboxes(account.mailboxes(), account.name(), result);
});
JSON.stringify(result);`,
		},
		"list-messages": {
			Name: "list-messages", TargetApp: "Mail",
			Script: stdinReader + `
var app = Application("com.apple.mail");
app.includeStandardAdditions = false;
function collectMailboxes(boxes, all) {
	boxes.forEach(function(box) {
		all.push(box);
		try { collectMailboxes(box.mailboxes(), all); } catch (e) {}
	});
}
var mailboxName = input.mailbox || "INBOX";
var parsedLimit = parseInt(input.limit, 10);
var limit = isNaN(parsedLimit) || parsedLimit <= 0 ? 20 : parsedLimit;
var mailbox;
if (mailboxName === "INBOX") {
	mailbox = app.inbox();
} else {
	var allBoxes = [];
	app.accounts().forEach(function(account) { collectMailboxes(account.mailboxes(), allBoxes); });
	var matches = allBoxes.filter(function(box) { return box.name() === mailboxName; });
	mailbox = matches.length > 0 ? matches[0] : null;
}
var messages = mailbox ? mailbox.messages() : [];
var result = messages.slice(0, limit).map(function(m) {
	var sent = m.dateSent();
	var read = false;
	try {
		read = m.readStatus();
	} catch (e) {
		read = false;
	}
	return {
		subject: m.subject(),
		sender: m.sender(),
		dateSent: sent ? sent.toISOString() : null,
		read: read,
		mailbox: mailbox ? mailbox.name() : mailboxName
	};
});
JSON.stringify(result);`,
		},
		"get-message": {
			Name: "get-message", TargetApp: "Mail",
			Script: stdinReader + `
var app = Application("com.apple.mail");
app.includeStandardAdditions = false;
if (!input.subject) throw new Error("subject is required");
function collectMailboxes(boxes, all) {
	boxes.forEach(function(box) { all.push(box); try { collectMailboxes(box.mailboxes(), all); } catch (e) {} });
}
var found = null;
var foundMailbox = null;
var accounts = app.accounts();
outer: for (var i = 0; i < accounts.length; i++) {
	var allBoxes = []; collectMailboxes(accounts[i].mailboxes(), allBoxes);
	for (var j = 0; j < allBoxes.length; j++) {
		var boxName = allBoxes[j].name();
		var msgs = allBoxes[j].messages();
		for (var k = 0; k < msgs.length; k++) {
			if (msgs[k].subject() === input.subject) {
				found = msgs[k];
				foundMailbox = boxName;
				break outer;
			}
		}
	}
}
if (!found) throw new Error("[NOT_FOUND] message not found");
var sent = found.dateSent();
var read = false;
try {
	read = found.readStatus();
} catch (e) {
	read = false;
}
var content = "";
try {
	content = found.content();
} catch (e) {
	content = "";
}
var result = {
	subject: found.subject(),
	sender: found.sender(),
	dateSent: sent ? sent.toISOString() : null,
	read: read,
	mailbox: foundMailbox,
	content: content
};
JSON.stringify(result);`,
		},
		"send-message": {
			Name: "send-message", TargetApp: "Mail",
			Script: stdinReader + `
var app = Application("com.apple.mail");
app.includeStandardAdditions = false;
if (!input.to) throw new Error("to is required");
if (!input.subject) throw new Error("subject is required");
if (!input.body) throw new Error("body is required");
var toList = Array.isArray(input.to) ? input.to : [input.to];
var ccList = input.cc ? (Array.isArray(input.cc) ? input.cc : [input.cc]) : [];
var bccList = input.bcc ? (Array.isArray(input.bcc) ? input.bcc : [input.bcc]) : [];
var msg = app.OutgoingMessage({
	subject: input.subject,
	content: input.body
});
msg.visible = false;
toList.forEach(function(addr) {
	msg.toRecipients.push(app.Recipient({address: addr}));
});
ccList.forEach(function(addr) {
	msg.ccRecipients.push(app.Recipient({address: addr}));
});
bccList.forEach(function(addr) {
	msg.bccRecipients.push(app.Recipient({address: addr}));
});
app.outgoingMessages.push(msg);
msg.send();
JSON.stringify({subject: input.subject, to: toList, sent: true});`,
		},
		"search-messages": {
			Name: "search-messages", TargetApp: "Mail",
			Script: stdinReader + `
var app = Application("com.apple.mail");
app.includeStandardAdditions = false;
function collectMailboxes(boxes, all) {
	boxes.forEach(function(box) { all.push(box); try { collectMailboxes(box.mailboxes(), all); } catch (e) {} });
}
var query = (input.query || "").toLowerCase();
var result = [];
var accounts = app.accounts();
for (var i = 0; i < accounts.length; i++) {
	var allBoxes = []; collectMailboxes(accounts[i].mailboxes(), allBoxes);
	for (var j = 0; j < allBoxes.length; j++) {
		var boxName = allBoxes[j].name();
		var msgs = allBoxes[j].messages();
		for (var k = 0; k < msgs.length; k++) {
			var m = msgs[k];
			var subject = (m.subject() || "").toLowerCase();
			var sender = (m.sender() || "").toLowerCase();
			if (subject.indexOf(query) >= 0 || sender.indexOf(query) >= 0) {
				var sent = m.dateSent();
				var read = false;
				try {
					read = m.readStatus();
				} catch (e) {
					read = false;
				}
				result.push({
					subject: m.subject(),
					sender: m.sender(),
					dateSent: sent ? sent.toISOString() : null,
					read: read,
					mailbox: boxName
				});
			}
		}
	}
}
JSON.stringify(result);`,
		},
	}
}
