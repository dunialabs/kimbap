package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/actions"
	"github.com/dunialabs/kimbap-core/internal/runtime"
)

type RuntimeAdapter struct {
	rt        *runtime.Runtime
	tenantID  string
	agentName string
}

func NewRuntimeAdapter(rt *runtime.Runtime, tenantID, agentName string) *RuntimeAdapter {
	if rt == nil {
		return nil
	}
	if strings.TrimSpace(tenantID) == "" {
		tenantID = "default"
	}
	if strings.TrimSpace(agentName) == "" {
		agentName = "mcp-agent"
	}
	return &RuntimeAdapter{rt: rt, tenantID: tenantID, agentName: agentName}
}

type ToolCallRequest struct {
	ToolName  string
	Arguments map[string]any
	RequestID string
	SessionID string
}

type ToolCallResult struct {
	Content        []ToolResultContent `json:"content"`
	IsError        bool                `json:"isError,omitempty"`
	RequestID      string              `json:"_requestId,omitempty"`
	AuditRef       string              `json:"_auditRef,omitempty"`
	PolicyDecision string              `json:"_policyDecision,omitempty"`
}

type ToolResultContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func (a *RuntimeAdapter) ExecuteToolCall(ctx context.Context, req ToolCallRequest) (*ToolCallResult, error) {
	if a == nil || a.rt == nil {
		return nil, nil
	}

	actionName := normalizeToolName(req.ToolName)
	if actionName == "" {
		return nil, nil
	}

	execReq := actions.ExecutionRequest{
		RequestID: req.RequestID,
		TraceID:   req.SessionID,
		TenantID:  a.tenantID,
		Principal: actions.Principal{
			ID:        a.agentName,
			TenantID:  a.tenantID,
			AgentName: a.agentName,
			Type:      "agent",
		},
		Action: actions.ActionDefinition{
			Name: actionName,
		},
		Input: req.Arguments,
		Mode:  actions.ModeCall,
		Session: &actions.SessionContext{
			SessionID: req.SessionID,
			Mode:      actions.ModeCall,
			Channel:   "mcp",
			StartedAt: time.Now(),
		},
	}

	if strings.TrimSpace(execReq.RequestID) == "" {
		execReq.RequestID = fmt.Sprintf("mcp_%d", time.Now().UTC().UnixNano())
	}
	if strings.TrimSpace(execReq.TraceID) == "" {
		execReq.TraceID = execReq.RequestID
	}
	if execReq.Input == nil {
		execReq.Input = map[string]any{}
	}

	result := a.rt.Execute(ctx, execReq)
	return mapExecutionResult(result), nil
}

func (a *RuntimeAdapter) CanHandle(ctx context.Context, toolName string) bool {
	if a == nil || a.rt == nil {
		return false
	}
	actionName := normalizeToolName(toolName)
	if actionName == "" {
		return false
	}
	if a.rt.ActionRegistry == nil {
		return false
	}
	def, err := a.rt.ActionRegistry.Lookup(ctx, actionName)
	return err == nil && def != nil
}

func normalizeToolName(toolName string) string {
	name := strings.TrimSpace(toolName)
	if name == "" {
		return ""
	}
	if strings.Contains(name, ".") {
		idx := strings.Index(name, ".")
		if idx <= 0 || idx >= len(name)-1 {
			return ""
		}
		service := strings.TrimSpace(name[:idx])
		action := strings.TrimSpace(name[idx+1:])
		if service == "" || action == "" {
			return ""
		}
		return name
	}

	splitIdx := strings.Index(name, "_")
	if splitIdx <= 0 || splitIdx >= len(name)-1 {
		return ""
	}

	service := strings.TrimSpace(name[:splitIdx])
	action := strings.TrimSpace(name[splitIdx+1:])
	if service == "" || action == "" {
		return ""
	}
	return service + "." + action
}

func mapExecutionResult(result actions.ExecutionResult) *ToolCallResult {
	tcr := &ToolCallResult{
		RequestID:      result.RequestID,
		AuditRef:       result.AuditRef,
		PolicyDecision: result.PolicyDecision,
	}

	switch result.Status {
	case actions.StatusSuccess:
		output, err := json.MarshalIndent(result.Output, "", "  ")
		if err != nil {
			tcr.IsError = true
			tcr.Content = []ToolResultContent{{Type: "text", Text: fmt.Sprintf("failed to encode action output: %v", err)}}
			return tcr
		}
		tcr.Content = []ToolResultContent{{Type: "text", Text: string(output)}}

	case actions.StatusApprovalRequired:
		tcr.IsError = false
		msg := "Action requires approval."
		if result.Error != nil && strings.TrimSpace(result.Error.Message) != "" {
			msg = result.Error.Message
		}
		if approvalID, ok := result.Meta["approval_request_id"].(string); ok && strings.TrimSpace(approvalID) != "" {
			msg += fmt.Sprintf(" Approval ID: %s", approvalID)
		}
		tcr.Content = []ToolResultContent{{Type: "text", Text: msg}}

	case actions.StatusError, actions.StatusTimeout, actions.StatusCancelled:
		tcr.IsError = true
		msg := fmt.Sprintf("Action failed: %s", result.Status)
		if result.Error != nil {
			msg = fmt.Sprintf("[%s] %s", result.Error.Code, result.Error.Message)
		}
		tcr.Content = []ToolResultContent{{Type: "text", Text: msg}}

	default:
		tcr.IsError = true
		tcr.Content = []ToolResultContent{{Type: "text", Text: fmt.Sprintf("Unhandled action status: %s", result.Status)}}
	}

	return tcr
}
