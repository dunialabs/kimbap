package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/services/catalog"
)

type initServiceSelection struct {
	Names   []string
	Skipped bool
	Reason  string
}

func starterServiceNames() []string {
	candidates := []string{}
	if runtime.GOOS == "darwin" {
		candidates = []string{
			"apple-notes", "apple-calendar", "apple-reminders",
			"finder", "safari", "contacts",
			"wikipedia", "open-meteo", "open-meteo-geocoding", "hacker-news",
		}
	} else {
		candidates = []string{
			"wikipedia", "open-meteo", "open-meteo-geocoding", "hacker-news",
			"rest-countries", "exchange-rate",
			"public-holidays", "nominatim",
		}
	}
	candidateSet := make(map[string]struct{}, len(candidates))
	for _, name := range candidates {
		candidateSet[name] = struct{}{}
	}

	orderedCandidates := make([]string, 0, len(candidates))
	if allServices, listErr := catalog.List(); listErr == nil {
		for _, name := range allServices {
			if _, ok := candidateSet[name]; ok {
				orderedCandidates = append(orderedCandidates, name)
			}
		}
	}
	if len(orderedCandidates) == 0 {
		orderedCandidates = candidates
	}

	return filterStarterServiceNames(orderedCandidates, func(name string) (*services.ServiceManifest, error) {
		data, err := catalog.Get(name)
		if err != nil {
			return nil, err
		}
		return services.ParseManifest(data)
	})
}

func filterStarterServiceNames(candidates []string, resolveManifest func(name string) (*services.ServiceManifest, error)) []string {
	filtered := make([]string, 0, len(candidates))
	for _, name := range candidates {
		if resolveManifest != nil {
			manifest, err := resolveManifest(name)
			if err == nil && serviceManifestRequiresCredentials(manifest) {
				continue
			}
		}
		filtered = append(filtered, name)
	}
	return filtered
}

func resolveInitServiceSelectionFromReader(rawServices string, noServices bool, interactive bool, reader io.Reader) (initServiceSelection, error) {
	if noServices {
		return initServiceSelection{Skipped: true, Reason: "skipped by --no-services"}, nil
	}
	if strings.EqualFold(strings.TrimSpace(rawServices), "none") {
		return initServiceSelection{Skipped: true, Reason: "skipped by --services none"}, nil
	}

	if isChecklistSelectionKeyword(rawServices) {
		if !interactive {
			return initServiceSelection{}, fmt.Errorf("--services %q requires interactive stdin", strings.TrimSpace(rawServices))
		}
		return resolveChecklistServiceSelectionFromReader(reader)
	}

	if strings.EqualFold(strings.TrimSpace(rawServices), "all") {
		all, listErr := catalog.List()
		if listErr != nil {
			return initServiceSelection{}, fmt.Errorf("list catalog services: %w", listErr)
		}
		return initServiceSelection{Names: all}, nil
	}

	if strings.EqualFold(strings.TrimSpace(rawServices), "starter") || strings.EqualFold(strings.TrimSpace(rawServices), "recommended") {
		return initServiceSelection{Names: starterServiceNames()}, nil
	}

	if strings.TrimSpace(rawServices) != "" {
		selected := parseCSV(rawServices)
		if len(selected) == 0 {
			return initServiceSelection{}, fmt.Errorf("invalid --services value: %q", rawServices)
		}
		normalized, err := normalizeSelectedCatalogServices(selected)
		if err != nil {
			return initServiceSelection{}, err
		}
		return initServiceSelection{Names: normalized}, nil
	}

	if !interactive {
		return initServiceSelection{Skipped: true, Reason: "non-interactive stdin"}, nil
	}

	return resolveChecklistServiceSelectionFromReader(reader)
}

func isChecklistSelectionKeyword(raw string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	switch trimmed {
	case "select", "interactive", "checkbox", "checklist":
		return true
	default:
		return false
	}
}

func resolveChecklistServiceSelectionFromReader(reader io.Reader) (initServiceSelection, error) {
	all, err := catalog.List()
	if err != nil {
		return initServiceSelection{}, fmt.Errorf("list catalog services: %w", err)
	}
	if len(all) == 0 {
		return initServiceSelection{Skipped: true, Reason: "no catalog services available"}, nil
	}

	selected := make(map[string]bool, len(all))
	starterSet := make(map[string]struct{})
	for _, name := range starterServiceNames() {
		starterSet[name] = struct{}{}
	}
	for _, name := range all {
		if _, ok := starterSet[name]; ok {
			selected[name] = true
		}
	}

	canRedraw := isInteractiveTTY(os.Stderr) && os.Getenv("TERM") != "dumb"
	prevLines := 0
	statusMsg := ""

	br := bufio.NewReader(reader)
	for {
		if canRedraw && prevLines > 0 {
			_, _ = fmt.Fprintf(os.Stderr, "\033[%dA\033[J", prevLines+1)
		}

		lineCount := 0
		_, _ = fmt.Fprintln(os.Stderr)
		lineCount++
		_, _ = fmt.Fprintln(os.Stderr, "Service checklist (checkbox style)")
		lineCount++
		_, _ = fmt.Fprintln(os.Stderr, "Commands: <idx>[,<idx>...] toggle · a=all · n=none · r=recommended · d=done · q=skip")
		lineCount++
		for idx, name := range all {
			mark := " "
			if selected[name] {
				mark = "x"
			}
			_, _ = fmt.Fprintf(os.Stderr, " [%s] %2d) %s\n", mark, idx+1, name)
			lineCount++
		}
		if statusMsg != "" {
			_, _ = fmt.Fprintln(os.Stderr, statusMsg)
			lineCount++
		}
		_, _ = fmt.Fprint(os.Stderr, "Select> ")
		prevLines = lineCount
		statusMsg = ""

		line, readErr := br.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return initServiceSelection{}, fmt.Errorf("read service selection: %w", readErr)
		}
		trimmed := strings.ToLower(strings.TrimSpace(line))
		if trimmed == "" && errors.Is(readErr, io.EOF) {
			return initServiceSelection{Skipped: true, Reason: "EOF"}, nil
		}

		switch trimmed {
		case "", "d", "done":
			names := make([]string, 0, len(all))
			for _, name := range all {
				if selected[name] {
					names = append(names, name)
				}
			}
			if len(names) == 0 {
				return initServiceSelection{Skipped: true, Reason: "empty selection"}, nil
			}
			return initServiceSelection{Names: names}, nil
		case "q", "quit", "skip", "cancel":
			return initServiceSelection{Skipped: true, Reason: "user declined"}, nil
		case "a", "all":
			for _, name := range all {
				selected[name] = true
			}
		case "n", "none":
			for _, name := range all {
				selected[name] = false
			}
		case "r", "recommended", "starter":
			for _, name := range all {
				_, isStarter := starterSet[name]
				selected[name] = isStarter
			}
		default:
			indices, parseErr := parseChecklistIndices(trimmed, len(all))
			if parseErr != nil {
				if canRedraw {
					statusMsg = fmt.Sprintf("Invalid selection: %v", parseErr)
				} else {
					_, _ = fmt.Fprintf(os.Stderr, "Invalid selection: %v\n", parseErr)
				}
				continue
			}
			for _, idx := range indices {
				name := all[idx-1]
				selected[name] = !selected[name]
			}
		}
	}
}

func parseChecklistIndices(raw string, max int) ([]int, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("empty selection")
	}
	tokens := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty selection")
	}
	seen := map[int]struct{}{}
	out := make([]int, 0, len(tokens))
	addIndex := func(v int) error {
		if v < 1 || v > max {
			return fmt.Errorf("selection index %d out of range (1-%d)", v, max)
		}
		if _, ok := seen[v]; ok {
			return nil
		}
		seen[v] = struct{}{}
		out = append(out, v)
		return nil
	}

	for _, token := range tokens {
		part := strings.TrimSpace(token)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			start, err1 := strconv.Atoi(strings.TrimSpace(bounds[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("invalid range %q", part)
			}
			if start > end {
				start, end = end, start
			}
			for i := start; i <= end; i++ {
				if err := addIndex(i); err != nil {
					return nil, err
				}
			}
			continue
		}
		idx, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid selection token %q", part)
		}
		if err := addIndex(idx); err != nil {
			return nil, err
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty selection")
	}
	return out, nil
}

func resolveInitServiceSelection(rawServices string, noServices bool) (initServiceSelection, error) {
	return resolveInitServiceSelectionFromReader(rawServices, noServices, isInteractiveStdin(), os.Stdin)
}

func isInteractiveStdin() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
