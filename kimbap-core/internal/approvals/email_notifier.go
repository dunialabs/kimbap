package approvals

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

const (
	defaultSMTPPort    = 587
	defaultSMTPTimeout = 30 * time.Second
)

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
	if e.port == 465 {
		return fmt.Errorf("email notifier: implicit TLS SMTP on port 465 is not supported; use a server/port that supports STARTTLS (commonly 587)")
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultSMTPTimeout)
		defer cancel()
	}

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("email notifier: dial: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, e.host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("email notifier: smtp client: %w", err)
	}
	defer func() {
		_ = client.Quit()
		_ = conn.Close()
	}()

	tlsActive := false
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: e.host}); err != nil {
			return fmt.Errorf("email notifier: STARTTLS: %w", err)
		}
		tlsActive = true
	}

	if strings.TrimSpace(e.username) != "" {
		if !tlsActive {
			return fmt.Errorf("email notifier: STARTTLS not available; refusing to send credentials over plaintext")
		}
		auth := smtp.PlainAuth("", e.username, e.password, e.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("email notifier: auth: %w", err)
		}
	}
	if err := client.Mail(e.from); err != nil {
		return fmt.Errorf("email notifier: MAIL FROM: %w", err)
	}
	for _, rcpt := range e.to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("email notifier: RCPT TO %s: %w", rcpt, err)
		}
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("email notifier: DATA: %w", err)
	}
	if _, err := wc.Write(msg); err != nil {
		_ = wc.Close()
		return fmt.Errorf("email notifier: write message: %w", err)
	}
	return wc.Close()
}
