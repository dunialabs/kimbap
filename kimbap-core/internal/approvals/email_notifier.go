package approvals

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

const defaultSMTPPort = 587

type EmailNotifier struct {
	host     string
	port     int
	from     string
	to       []string
	username string
	password string
}

func NewEmailNotifier(host string, port int, from string, to []string, username, password string) *EmailNotifier {
	if port <= 0 {
		port = defaultSMTPPort
	}
	return &EmailNotifier{
		host:     strings.TrimSpace(host),
		port:     port,
		from:     strings.TrimSpace(from),
		to:       to,
		username: username,
		password: password,
	}
}

func (e *EmailNotifier) Notify(ctx context.Context, req *ApprovalRequest) error {
	if e.host == "" {
		return fmt.Errorf("email notifier: smtp host is required")
	}
	if e.from == "" {
		return fmt.Errorf("email notifier: from address is required")
	}
	if len(e.to) == 0 {
		return fmt.Errorf("email notifier: at least one recipient required")
	}

	subject := fmt.Sprintf("[Kimbap] Approval Required: %s.%s", req.Service, req.Action)
	body := fmt.Sprintf("Action: %s.%s\nAgent: %s\nRisk: %s\nApproval ID: %s\n\nRun: kimbap approve %s",
		req.Service, req.Action, req.AgentName, req.Risk, req.ID, req.ID)
	msg := []byte("From: " + e.from + "\r\n" +
		"To: " + strings.Join(e.to, ", ") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" + body)

	addr := fmt.Sprintf("%s:%d", e.host, e.port)
	var auth smtp.Auth
	if strings.TrimSpace(e.username) != "" {
		auth = smtp.PlainAuth("", e.username, e.password, e.host)
	}

	type sendResult struct{ err error }
	ch := make(chan sendResult, 1)
	go func() {
		ch <- sendResult{smtp.SendMail(addr, auth, e.from, e.to, msg)}
	}()
	select {
	case r := <-ch:
		return r.err
	case <-ctx.Done():
		return ctx.Err()
	}
}
