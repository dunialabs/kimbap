package services

import (
	"net/http"
	"net/url"
	"testing"
)

func TestRemoteFetchClientRejectsHTTPSDowngradeRedirect(t *testing.T) {
	client := newRemoteFetchClient("https", false)
	if client.CheckRedirect == nil {
		t.Fatal("expected redirect policy")
	}

	req := &http.Request{URL: &url.URL{Scheme: "http", Host: "example.com"}}
	err := client.CheckRedirect(req, []*http.Request{{URL: &url.URL{Scheme: "https", Host: "example.com"}}})
	if err == nil {
		t.Fatal("expected redirect downgrade from https to http to be rejected")
	}
}

func TestRemoteFetchClientAllowsHTTPRedirectWhenInitialSchemeIsHTTP(t *testing.T) {
	client := newRemoteFetchClient("http", false)
	if client.CheckRedirect == nil {
		t.Fatal("expected redirect policy")
	}

	req := &http.Request{URL: &url.URL{Scheme: "http", Host: "example.com"}}
	if err := client.CheckRedirect(req, []*http.Request{{URL: &url.URL{Scheme: "http", Host: "example.com"}}}); err != nil {
		t.Fatalf("expected http redirect to stay allowed for http source, got %v", err)
	}
}
