package main

import (
	"testing"

	"github.com/dunialabs/kimbap-core/internal/approvals"
	"github.com/dunialabs/kimbap-core/internal/config"
)

func TestBuildNotifierFromConfigEmpty(t *testing.T) {
	cfg := config.DefaultConfig()
	n := buildNotifierFromConfig(cfg)
	if _, ok := n.(*approvals.LogNotifier); !ok {
		t.Errorf("expected LogNotifier for empty config, got %T", n)
	}
}

func TestBuildNotifierFromConfigSlack(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Notifications.Slack.WebhookURL = "https://hooks.slack.com/test"
	n := buildNotifierFromConfig(cfg)
	if _, ok := n.(*approvals.SlackNotifier); !ok {
		t.Errorf("expected SlackNotifier for slack config, got %T", n)
	}
}

func TestBuildNotifierFromConfigMultiple(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Notifications.Slack.WebhookURL = "https://hooks.slack.com/test"
	cfg.Notifications.Telegram.BotToken = "sometoken"
	cfg.Notifications.Telegram.ChatID = "123456"
	n := buildNotifierFromConfig(cfg)
	if _, ok := n.(*approvals.MultiNotifier); !ok {
		t.Errorf("expected MultiNotifier for multiple adapters, got %T", n)
	}
}

func TestBuildNotifierFromConfigNilCfg(t *testing.T) {
	n := buildNotifierFromConfig(nil)
	if _, ok := n.(*approvals.LogNotifier); !ok {
		t.Errorf("expected LogNotifier for nil config, got %T", n)
	}
}
