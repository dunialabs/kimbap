package approvals

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
)

func startMockSMTPServer(t *testing.T) (addr string, msgCh <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ch := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		defer ln.Close()

		var message strings.Builder
		r := bufio.NewReader(conn)
		fmt.Fprintf(conn, "220 mock SMTP\r\n")
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				break
			}
			line = strings.TrimRight(line, "\r\n")
			if inData {
				if line == "." {
					inData = false
					fmt.Fprintf(conn, "250 OK\r\n")
					ch <- message.String()
				} else {
					message.WriteString(line + "\n")
				}
				continue
			}
			upper := strings.ToUpper(line)
			switch {
			case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
				fmt.Fprintf(conn, "250 OK\r\n")
			case strings.HasPrefix(upper, "MAIL FROM"):
				fmt.Fprintf(conn, "250 OK\r\n")
			case strings.HasPrefix(upper, "RCPT TO"):
				fmt.Fprintf(conn, "250 OK\r\n")
			case upper == "DATA":
				inData = true
				fmt.Fprintf(conn, "354 Start\r\n")
			case upper == "QUIT":
				fmt.Fprintf(conn, "221 Bye\r\n")
				return
			default:
				fmt.Fprintf(conn, "500 Unrecognized\r\n")
			}
		}
	}()
	return ln.Addr().String(), ch
}

func TestEmailNotifierSendsCorrectMessage(t *testing.T) {
	addr, msgCh := startMockSMTPServer(t)
	parts := strings.SplitN(addr, ":", 2)
	host := parts[0]
	var port int
	fmt.Sscanf(parts[1], "%d", &port)

	notifier := NewEmailNotifier(host, port, "kimbap@example.com", []string{"ops@example.com"}, "", "")
	req := &ApprovalRequest{ID: "appr_em1", Service: "stripe", Action: "refund", AgentName: "billing-bot", Risk: "high"}

	if err := notifier.Notify(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := <-msgCh
	if !strings.Contains(msg, "[Kimbap] Approval Required: stripe.refund") {
		t.Errorf("message missing expected subject, got:\n%s", msg)
	}
	if !strings.Contains(msg, "appr_em1") {
		t.Errorf("message missing approval ID")
	}
	if !strings.Contains(msg, "kimbap approve") {
		t.Errorf("message missing approve instruction")
	}
}

func TestEmailNotifierErrorOnConnectionFailure(t *testing.T) {
	notifier := NewEmailNotifier("127.0.0.1", 19999, "from@x.com", []string{"to@x.com"}, "", "")
	err := notifier.Notify(context.Background(), &ApprovalRequest{ID: "x", Service: "s", Action: "a"})
	if err == nil {
		t.Fatal("expected error on connection refused")
	}
}

func TestEmailNotifierErrorOnEmptyHost(t *testing.T) {
	notifier := NewEmailNotifier("", 587, "from@x.com", []string{"to@x.com"}, "", "")
	if err := notifier.Notify(context.Background(), &ApprovalRequest{ID: "x"}); err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestEmailNotifierErrorOnEmptyTo(t *testing.T) {
	notifier := NewEmailNotifier("smtp.example.com", 587, "from@x.com", []string{}, "", "")
	if err := notifier.Notify(context.Background(), &ApprovalRequest{ID: "x"}); err == nil {
		t.Fatal("expected error for empty recipients")
	}
}

func TestEmailNotifierDefaultPort(t *testing.T) {
	notifier := NewEmailNotifier("smtp.example.com", 0, "from@x.com", []string{"to@x.com"}, "", "")
	if notifier.port != defaultSMTPPort {
		t.Errorf("expected default port %d, got %d", defaultSMTPPort, notifier.port)
	}
	notifier2 := NewEmailNotifier("smtp.example.com", -1, "from@x.com", []string{"to@x.com"}, "", "")
	if notifier2.port != defaultSMTPPort {
		t.Errorf("expected default port %d for negative port, got %d", defaultSMTPPort, notifier2.port)
	}
}

func TestEmailNotifierContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	notifier := NewEmailNotifier("198.51.100.1", 587, "from@x.com", []string{"to@x.com"}, "", "")
	err := notifier.Notify(ctx, &ApprovalRequest{ID: "x", Service: "s", Action: "a"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !strings.Contains(err.Error(), "context canceled") && !strings.Contains(err.Error(), "dial") {
		t.Errorf("expected context-canceled or dial error, got: %v", err)
	}
}
