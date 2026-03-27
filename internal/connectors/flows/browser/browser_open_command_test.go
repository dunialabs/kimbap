package browser

import (
	"errors"
	"reflect"
	"testing"
)

func TestBrowserOpenCommandForOS(t *testing.T) {
	authURL := "https://example.com/auth"

	tests := []struct {
		name       string
		goos       string
		lookPath   func(string) (string, error)
		browserEnv string
		wantCmd    string
		wantArgs   []string
		wantErr    bool
	}{
		{
			name:     "darwin uses open",
			goos:     "darwin",
			wantCmd:  "open",
			wantArgs: []string{authURL},
		},
		{
			name: "linux uses BROWSER before xdg-open",
			goos: "linux",
			lookPath: func(bin string) (string, error) {
				if bin == "xdg-open" {
					return "/usr/bin/xdg-open", nil
				}
				return "", errors.New("not found")
			},
			browserEnv: "firefox",
			wantCmd:    "firefox",
			wantArgs:   []string{authURL},
		},
		{
			name: "linux falls back to xdg-open",
			goos: "linux",
			lookPath: func(bin string) (string, error) {
				if bin == "xdg-open" {
					return "/usr/bin/xdg-open", nil
				}
				return "", errors.New("not found")
			},
			wantCmd:  "xdg-open",
			wantArgs: []string{authURL},
		},
		{
			name: "linux BROWSER supports %s placeholder",
			goos: "linux",
			lookPath: func(string) (string, error) {
				return "", errors.New("not found")
			},
			browserEnv: "firefox --new-tab=%s",
			wantCmd:    "firefox",
			wantArgs:   []string{"--new-tab=" + authURL},
		},
		{
			name: "linux BROWSER supports colon choices",
			goos: "linux",
			lookPath: func(string) (string, error) {
				return "", errors.New("not found")
			},
			browserEnv: "w3m %s:lynx %s",
			wantCmd:    "w3m",
			wantArgs:   []string{authURL},
		},
		{
			name: "linux BROWSER keeps quoted executable path",
			goos: "linux",
			lookPath: func(string) (string, error) {
				return "", errors.New("not found")
			},
			browserEnv: "\"/opt/custom browser\" --new-window",
			wantCmd:    "/opt/custom browser",
			wantArgs:   []string{"--new-window", authURL},
		},
		{
			name: "linux BROWSER colon inside quotes is not split",
			goos: "linux",
			lookPath: func(string) (string, error) {
				return "", errors.New("not found")
			},
			browserEnv: "\"w3m:custom\" %s:lynx %s",
			wantCmd:    "w3m:custom",
			wantArgs:   []string{authURL},
		},
		{
			name: "linux errors when no browser available",
			goos: "linux",
			lookPath: func(string) (string, error) {
				return "", errors.New("not found")
			},
			wantErr: true,
		},
		{
			name:     "windows uses rundll32",
			goos:     "windows",
			wantCmd:  "rundll32",
			wantArgs: []string{"url.dll,FileProtocolHandler", authURL},
		},
		{
			name:    "unsupported platform returns error",
			goos:    "plan9",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, args, err := browserOpenCommandForOS(tc.goos, authURL, tc.lookPath, tc.browserEnv)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got cmd=%q args=%v", cmd, args)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cmd != tc.wantCmd {
				t.Fatalf("cmd=%q want=%q", cmd, tc.wantCmd)
			}
			if !reflect.DeepEqual(args, tc.wantArgs) {
				t.Fatalf("args=%v want=%v", args, tc.wantArgs)
			}
		})
	}
}
