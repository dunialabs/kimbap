package core

import (
	"strings"
)

type startupDiagnostics struct {
	stderrWriter            *StderrTailWriter
	runnerStartupFailureMsg string
}

func newStartupDiagnostics(stderrWriter *StderrTailWriter) *startupDiagnostics {
	return &startupDiagnostics{
		stderrWriter: stderrWriter,
	}
}

func (d *startupDiagnostics) SetRunnerStartupFailureMessage(msg string) {
	d.runnerStartupFailureMsg = msg
}

func (d *startupDiagnostics) GetPreferredMessage(finalErr error) string {
	if d.runnerStartupFailureMsg != "" {
		return d.runnerStartupFailureMsg
	}

	if d.stderrWriter != nil {
		stderrSummary := d.stderrWriter.SummarizeTail(300)
		if stderrSummary != "" && !isGenericErrorMessage(stderrSummary) {
			return stderrSummary
		}
	}

	if finalErr != nil {
		return finalErr.Error()
	}

	return ""
}

func (d *startupDiagnostics) GetStderrSummary() string {
	if d.stderrWriter == nil {
		return ""
	}
	return d.stderrWriter.SummarizeTail(300)
}

var genericConnectionMessages = map[string]struct{}{
	"request timed out":           {},
	"connection closed":           {},
	"transport closed by server":  {},
	"fetch failed":                {},
	"error: request timed out":    {},
	"error: connection closed":    {},
	"error: fetch failed":         {},
	"mcperror: request timed out": {},
	"mcperror: connection closed": {},
	"typeerror: fetch failed":     {},
}

func isGenericErrorMessage(msg string) bool {
	normalized := strings.ToLower(strings.TrimSpace(msg))
	baseMessage := normalized
	if idx := strings.LastIndex(baseMessage, " ("); idx > 0 {
		baseMessage = strings.TrimSpace(baseMessage[:idx])
	}
	_, generic := genericConnectionMessages[baseMessage]
	return generic
}

func recordServerStartupError(sc *ServerContext, primaryMessage string, preferredMessage string) {
	nextMessage := preferredMessage
	if nextMessage == "" {
		nextMessage = primaryMessage
	}
	if strings.TrimSpace(nextMessage) == "" {
		return
	}
	sc.mu.RLock()
	current := sc.LastError
	sc.mu.RUnlock()
	if current == nextMessage {
		return
	}
	sc.RecordError(nextMessage)
}
