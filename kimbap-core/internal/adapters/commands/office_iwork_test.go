package commands

import (
	"strings"
	"testing"
)

func TestMSOfficeCommandsBasicCoverage(t *testing.T) {
	cmds := MSOfficeCommands()
	if len(cmds) != 21 {
		t.Fatalf("got %d commands, want 21", len(cmds))
	}

	expected := []string{
		"word-create-document", "word-open-document", "word-get-text", "word-set-text", "word-find-replace", "word-save-as-pdf", "word-close-document",
		"excel-create-workbook", "excel-open-workbook", "excel-read-cell", "excel-write-cell", "excel-read-range", "excel-save-as-pdf", "excel-close-workbook",
		"ppt-create-presentation", "ppt-open-presentation", "ppt-add-slide", "ppt-set-slide-text", "ppt-save-as-pdf", "ppt-save-as-png", "ppt-close-presentation",
	}
	for _, name := range expected {
		if _, ok := cmds[name]; !ok {
			t.Errorf("missing command %q", name)
		}
	}

	for name, cmd := range cmds {
		if !strings.Contains(cmd.Script, stdinReader) {
			t.Errorf("%s: missing stdin reader", name)
		}
		switch {
		case strings.HasPrefix(name, "word-"):
			if cmd.TargetApp != "Microsoft Word" {
				t.Errorf("%s: TargetApp = %q, want Microsoft Word", name, cmd.TargetApp)
			}
		case strings.HasPrefix(name, "excel-"):
			if cmd.TargetApp != "Microsoft Excel" {
				t.Errorf("%s: TargetApp = %q, want Microsoft Excel", name, cmd.TargetApp)
			}
		case strings.HasPrefix(name, "ppt-"):
			if cmd.TargetApp != "Microsoft PowerPoint" {
				t.Errorf("%s: TargetApp = %q, want Microsoft PowerPoint", name, cmd.TargetApp)
			}
		}
	}
}

func TestIWorkCommandsBasicCoverage(t *testing.T) {
	cmds := IWorkCommands()
	if len(cmds) != 20 {
		t.Fatalf("got %d commands, want 20", len(cmds))
	}

	expected := []string{
		"keynote-create-presentation", "keynote-open-presentation", "keynote-add-slide", "keynote-set-slide-text", "keynote-export-pdf", "keynote-start-slideshow", "keynote-close-presentation",
		"numbers-create-spreadsheet", "numbers-open-spreadsheet", "numbers-read-cell", "numbers-write-cell", "numbers-export-pdf", "numbers-close-spreadsheet",
		"pages-create-document", "pages-open-document", "pages-get-text", "pages-set-text", "pages-export-pdf", "pages-export-word", "pages-close-document",
	}
	for _, name := range expected {
		if _, ok := cmds[name]; !ok {
			t.Errorf("missing command %q", name)
		}
	}

	for name, cmd := range cmds {
		if !strings.Contains(cmd.Script, stdinReader) {
			t.Errorf("%s: missing stdin reader", name)
		}
		switch {
		case strings.HasPrefix(name, "keynote-"):
			if cmd.TargetApp != "Keynote" {
				t.Errorf("%s: TargetApp = %q, want Keynote", name, cmd.TargetApp)
			}
		case strings.HasPrefix(name, "numbers-"):
			if cmd.TargetApp != "Numbers" {
				t.Errorf("%s: TargetApp = %q, want Numbers", name, cmd.TargetApp)
			}
		case strings.HasPrefix(name, "pages-"):
			if cmd.TargetApp != "Pages" {
				t.Errorf("%s: TargetApp = %q, want Pages", name, cmd.TargetApp)
			}
		}
	}
}
