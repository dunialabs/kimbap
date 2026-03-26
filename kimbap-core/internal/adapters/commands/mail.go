package commands

func MailCommands() map[string]Command {
	return map[string]Command{
		"list-mailboxes": {
			Name: "list-mailboxes", TargetApp: "Mail",
			Script: stdinReader + `
var app = Application("Mail");
app.includeStandardAdditions = false;
var accounts = app.accounts();
var result = [];
accounts.forEach(function(account) {
	var accountName = account.name();
	var boxes = account.mailboxes();
	boxes.forEach(function(box) {
		var unreadCount = 0;
		try {
			unreadCount = box.unreadCount();
		} catch (e) {
			unreadCount = 0;
		}
		result.push({
			name: box.name(),
			accountName: accountName,
			unreadCount: unreadCount
		});
	});
});
JSON.stringify(result);`,
		},
		"list-messages": {
			Name: "list-messages", TargetApp: "Mail",
			Script: stdinReader + `
var app = Application("Mail");
app.includeStandardAdditions = false;
var mailboxName = input.mailbox || "INBOX";
var parsedLimit = parseInt(input.limit, 10);
var limit = isNaN(parsedLimit) || parsedLimit <= 0 ? 20 : parsedLimit;
var mailbox;
if (mailboxName === "INBOX") {
	mailbox = app.inbox();
} else {
	var allBoxes = [];
	app.accounts().forEach(function(account) {
		allBoxes = allBoxes.concat(account.mailboxes());
	});
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
var app = Application("Mail");
app.includeStandardAdditions = false;
if (!input.subject) throw new Error("subject is required");
var matches = app.messages.whose({subject: input.subject})();
if (matches.length === 0) throw new Error("message not found");
var m = matches[0];
var sent = m.dateSent();
var read = false;
try {
	read = m.readStatus();
} catch (e) {
	read = false;
}
var result = {
	subject: m.subject(),
	sender: m.sender(),
	dateSent: sent ? sent.toISOString() : null,
	read: read,
	mailbox: m.mailbox().name(),
	content: m.content()
};
JSON.stringify(result);`,
		},
		"send-message": {
			Name: "send-message", TargetApp: "Mail",
			Script: stdinReader + `
var app = Application("Mail");
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
var app = Application("Mail");
app.includeStandardAdditions = false;
var query = (input.query || "").toLowerCase();
var all = app.messages();
var result = all.filter(function(m) {
	var subject = (m.subject() || "").toLowerCase();
	var sender = (m.sender() || "").toLowerCase();
	return subject.indexOf(query) >= 0 || sender.indexOf(query) >= 0;
}).map(function(m) {
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
		mailbox: m.mailbox().name()
	};
});
JSON.stringify(result);`,
		},
	}
}
