package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
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
			"wikipedia", "open-meteo", "hacker-news",
		}
	} else {
		candidates = []string{
			"wikipedia", "open-meteo", "hacker-news",
			"rest-countries", "exchange-rate",
			"public-holidays", "nominatim",
		}
	}
	return filterStarterServiceNames(candidates, func(name string) (*services.ServiceManifest, error) {
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

	if strings.EqualFold(strings.TrimSpace(rawServices), "all") {
		all, listErr := catalog.List()
		if listErr != nil {
			return initServiceSelection{}, fmt.Errorf("list catalog services: %w", listErr)
		}
		return initServiceSelection{Names: all}, nil
	}

	if strings.EqualFold(strings.TrimSpace(rawServices), "starter") {
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

	serviceCount := 0
	if all, err := catalog.List(); err == nil {
		serviceCount = len(all)
	}
	_, _ = fmt.Fprintf(os.Stderr, "Install all %d catalog services? [Y/n/select]: ", serviceCount)

	br := bufio.NewReader(reader)

	line, err := br.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return initServiceSelection{}, fmt.Errorf("read service selection: %w", err)
	}

	trimmed := strings.TrimSpace(line)

	if trimmed == "" && errors.Is(err, io.EOF) {
		return initServiceSelection{Skipped: true, Reason: "EOF"}, nil
	}

	if trimmed == "" || strings.EqualFold(trimmed, "y") || strings.EqualFold(trimmed, "yes") {
		all, listErr := catalog.List()
		if listErr != nil {
			return initServiceSelection{}, fmt.Errorf("list catalog services: %w", listErr)
		}
		return initServiceSelection{Names: all}, nil
	}

	if strings.EqualFold(trimmed, "n") || strings.EqualFold(trimmed, "no") {
		return initServiceSelection{Skipped: true, Reason: "user declined"}, nil
	}

	if strings.EqualFold(trimmed, "starter") {
		return initServiceSelection{Names: starterServiceNames()}, nil
	}

	if strings.EqualFold(trimmed, "select") {
		if printErr := printCatalogServiceCategories(); printErr != nil {
			return initServiceSelection{}, printErr
		}
		_, _ = fmt.Fprint(os.Stderr, "Enter comma-separated names, or 'all': ")

		line2, err2 := br.ReadString('\n')
		if err2 != nil && !errors.Is(err2, io.EOF) {
			return initServiceSelection{}, fmt.Errorf("read service selection: %w", err2)
		}

		trimmed2 := strings.TrimSpace(line2)
		if trimmed2 == "" {
			return initServiceSelection{Skipped: true, Reason: "empty selection"}, nil
		}

		if strings.EqualFold(trimmed2, "all") {
			all, listErr := catalog.List()
			if listErr != nil {
				return initServiceSelection{}, fmt.Errorf("list catalog services: %w", listErr)
			}
			return initServiceSelection{Names: all}, nil
		}

		if strings.EqualFold(trimmed2, "starter") {
			return initServiceSelection{Names: starterServiceNames()}, nil
		}

		selected2 := parseCSV(trimmed2)
		normalized2, normalizeErr2 := normalizeSelectedCatalogServices(selected2)
		if normalizeErr2 != nil {
			return initServiceSelection{}, normalizeErr2
		}
		if len(normalized2) == 0 {
			return initServiceSelection{Skipped: true, Reason: "empty selection"}, nil
		}
		return initServiceSelection{Names: normalized2}, nil
	}

	selected := parseCSV(trimmed)
	normalized, normalizeErr := normalizeSelectedCatalogServices(selected)
	if normalizeErr != nil {
		return initServiceSelection{}, normalizeErr
	}
	if len(normalized) == 0 {
		return initServiceSelection{Skipped: true, Reason: "empty selection"}, nil
	}
	return initServiceSelection{Names: normalized}, nil
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
