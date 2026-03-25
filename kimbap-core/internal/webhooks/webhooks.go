package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"sync"
	"time"
)

type EventType string

const (
	EventTokenCreated   EventType = "token.created"
	EventTokenDeleted   EventType = "token.deleted"
	EventTokenUpdated   EventType = "token.updated"
	EventPolicyCreated  EventType = "policy.created"
	EventPolicyUpdated  EventType = "policy.updated"
	EventPolicyDeleted  EventType = "policy.deleted"
	EventSkillInstalled EventType = "skill.installed"
	EventSkillRemoved   EventType = "skill.removed"

	// Approval events are fired by the REST API when approval lifecycle transitions occur.
	// These are intended for management webhook subscribers that want to react to approval
	// state changes (e.g., audit systems, external ticketing).
	// Note: these are separate from approvals.Notifier which handles synchronous notification
	// to human approvers (Slack, Telegram, email).
	EventApprovalRequested EventType = "approval.requested"
	EventApprovalApproved  EventType = "approval.approved"
	EventApprovalDenied    EventType = "approval.denied"
	EventApprovalExpired   EventType = "approval.expired"
)

type Event struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	TenantID  string    `json:"tenant_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

type Subscription struct {
	ID       string      `json:"id"`
	URL      string      `json:"url"`
	Secret   string      `json:"secret,omitempty"`
	Events   []EventType `json:"events"`
	TenantID string      `json:"tenant_id,omitempty"`
	Active   bool        `json:"active"`
}

type Dispatcher struct {
	mu            sync.RWMutex
	subscriptions []Subscription
	client        *http.Client
	events        []Event
	maxEvents     int
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		maxEvents: 1000,
	}
}

func (d *Dispatcher) Subscribe(sub Subscription) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := range d.subscriptions {
		if d.subscriptions[i].ID == sub.ID && d.subscriptions[i].TenantID == sub.TenantID {
			d.subscriptions[i] = sub
			d.subscriptions[i].Active = true
			return
		}
	}
	sub.Active = true
	d.subscriptions = append(d.subscriptions, sub)
}

func (d *Dispatcher) Unsubscribe(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := range d.subscriptions {
		if d.subscriptions[i].ID == id {
			d.subscriptions[i].Active = false
		}
	}
}

func (d *Dispatcher) UnsubscribeByTenant(id, tenantID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := range d.subscriptions {
		if d.subscriptions[i].ID == id && d.subscriptions[i].TenantID == tenantID {
			d.subscriptions[i].Active = false
		}
	}
}

func (d *Dispatcher) ListSubscriptions() []Subscription {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]Subscription, 0, len(d.subscriptions))
	for _, sub := range d.subscriptions {
		if sub.Active {
			safe := sub
			safe.Secret = ""
			out = append(out, safe)
		}
	}
	return out
}

func (d *Dispatcher) ListSubscriptionsByTenant(tenantID string) []Subscription {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]Subscription, 0)
	for _, sub := range d.subscriptions {
		if sub.Active && sub.TenantID == tenantID {
			safe := sub
			safe.Secret = ""
			out = append(out, safe)
		}
	}
	return out
}

func (d *Dispatcher) RecentEventsByTenant(tenantID string, limit int) []Event {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]Event, 0)
	for i := len(d.events) - 1; i >= 0 && len(out) < limit; i-- {
		if d.events[i].TenantID == tenantID {
			out = append(out, d.events[i])
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func (d *Dispatcher) Emit(eventType EventType, data any) {
	d.EmitForTenant("", eventType, data)
}

func (d *Dispatcher) EmitForTenant(tenantID string, eventType EventType, data any) {
	event := Event{
		ID:        fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		Type:      eventType,
		TenantID:  tenantID,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	d.mu.Lock()
	d.events = append(d.events, event)
	if len(d.events) > d.maxEvents {
		d.events = d.events[len(d.events)-d.maxEvents:]
	}
	subs := make([]Subscription, len(d.subscriptions))
	copy(subs, d.subscriptions)
	d.mu.Unlock()

	for _, sub := range subs {
		if !sub.Active || !matchesEvent(sub.Events, eventType) {
			continue
		}
		if sub.TenantID != "" && sub.TenantID != event.TenantID {
			continue
		}
		go d.deliver(sub, event)
	}
}

func (d *Dispatcher) RecentEvents(limit int) []Event {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if limit <= 0 || limit > len(d.events) {
		limit = len(d.events)
	}
	start := len(d.events) - limit
	out := make([]Event, limit)
	copy(out, d.events[start:])
	return out
}

func (d *Dispatcher) deliver(sub Subscription, event Event) {
	body, err := json.Marshal(event)
	if err != nil {
		log.Printf("webhook deliver: marshal event %s: %v", event.ID, err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, sub.URL, bytes.NewReader(body))
	if err != nil {
		log.Printf("webhook deliver: build request id=%s url=%s: %v", sub.ID, sub.URL, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Kimbap-Event", string(event.Type))
	req.Header.Set("X-Kimbap-Event-ID", event.ID)

	if sub.Secret != "" {
		mac := hmac.New(sha256.New, []byte(sub.Secret))
		_, _ = mac.Write(body)
		req.Header.Set("X-Kimbap-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := d.client.Do(req)
	if err != nil {
		log.Printf("webhook deliver: post id=%s url=%s: %v", sub.ID, sub.URL, err)
		return
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("webhook deliver: id=%s url=%s status=%d", sub.ID, sub.URL, resp.StatusCode)
	}
}

func matchesEvent(events []EventType, target EventType) bool {
	if len(events) == 0 {
		return true
	}
	return slices.Contains(events, target)
}
