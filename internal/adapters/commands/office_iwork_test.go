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

func TestKeynoteSetSlideTextUsesAssignment(t *testing.T) {
	cmds := IWorkCommands()
	cmd, ok := cmds["keynote-set-slide-text"]
	if !ok {
		t.Fatal("keynote-set-slide-text not found")
	}
	if strings.Contains(cmd.Script, "objectText().set(") {
		t.Error("keynote-set-slide-text must not use objectText().set() — invalid JXA; use objectText = value assignment")
	}
	if !strings.Contains(cmd.Script, "objectText = input.text") {
		t.Error("keynote-set-slide-text must assign objectText = input.text")
	}
}

func TestNumbersCellAccessUsesRanges(t *testing.T) {
	cmds := IWorkCommands()
	for _, name := range []string{"numbers-read-cell", "numbers-write-cell"} {
		cmd, ok := cmds[name]
		if !ok {
			t.Fatalf("%s not found", name)
		}
		if strings.Contains(cmd.Script, "cells.whose({name:") {
			t.Errorf("%s must not use cells.whose({name:...}) — use table.ranges[cell] instead", name)
		}
		if !strings.Contains(cmd.Script, "table.ranges[input.cell]") {
			t.Errorf("%s must access cells via table.ranges[input.cell]", name)
		}
	}
}

func TestMSOfficeExportCommandsUseResilientPDFPNGStrategies(t *testing.T) {
	cmds := MSOfficeCommands()

	wordPDF := cmds["word-save-as-pdf"].Script
	if !strings.Contains(wordPDF, "exportAsFixedFormat") {
		t.Error("word-save-as-pdf should try exportAsFixedFormat first")
	}
	if !strings.Contains(wordPDF, "fileFormat: 17") {
		t.Error("word-save-as-pdf should include numeric PDF fallback (17)")
	}

	excelPDF := cmds["excel-save-as-pdf"].Script
	if !strings.Contains(excelPDF, "exportAsFixedFormat") {
		t.Error("excel-save-as-pdf should use exportAsFixedFormat")
	}
	if !strings.Contains(excelPDF, "fileFormat: 57") {
		t.Error("excel-save-as-pdf should include saveAs PDF fallback (57)")
	}

	pptPDF := cmds["ppt-save-as-pdf"].Script
	if !strings.Contains(pptPDF, "fileFormat: 32") {
		t.Error("ppt-save-as-pdf should include numeric PDF fallback (32)")
	}

	pptPNG := cmds["ppt-save-as-png"].Script
	if !strings.Contains(pptPNG, "pres.export") {
		t.Error("ppt-save-as-png should attempt export() with filterName PNG")
	}
	if !strings.Contains(pptPNG, "fileFormat: \"PNG\"") {
		t.Error("ppt-save-as-png should keep saveAs PNG fallback")
	}
}
