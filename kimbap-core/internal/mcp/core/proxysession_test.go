package core

import (
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestInjectJSONRPCRequestIDsToMetaPreservesLargeNumericID(t *testing.T) {
	payload := []byte(`{"jsonrpc":"2.0","id":9007199254740993,"method":"tools/list","params":{}}`)

	updated, changed, err := injectJSONRPCRequestIDsToMeta(payload)
	if err != nil {
		t.Fatalf("injectJSONRPCRequestIDsToMeta returned error: %v", err)
	}
	if !changed {
		t.Fatal("expected payload to be changed")
	}

	var decoded map[string]any
	if err := decodeJSONUseNumber(updated, &decoded); err != nil {
		t.Fatalf("failed to decode updated payload: %v", err)
	}
	params, ok := decoded["params"].(map[string]any)
	if !ok {
		t.Fatal("expected params object")
	}
	meta, ok := params["_meta"].(map[string]any)
	if !ok {
		t.Fatal("expected _meta object")
	}
	idNum, ok := meta[upstreamRequestIDMetaKey].(json.Number)
	if !ok {
		t.Fatalf("expected injected request id as json.Number, got %T", meta[upstreamRequestIDMetaKey])
	}
	if idNum.String() != "9007199254740993" {
		t.Fatalf("expected exact numeric id preserved, got %s", idNum.String())
	}
}

func TestNormalizeCallToolParamsPreservesLargeNumericArguments(t *testing.T) {
	raw := &mcp.CallToolParamsRaw{
		Name:      "server::tool",
		Arguments: json.RawMessage(`{"count":9007199254740993}`),
	}

	p := &ProxySession{}
	params, err := p.normalizeCallToolParams(raw)
	if err != nil {
		t.Fatalf("normalizeCallToolParams returned error: %v", err)
	}

	args, ok := params.Arguments.(map[string]any)
	if !ok {
		t.Fatalf("expected arguments map, got %T", params.Arguments)
	}
	count, ok := args["count"].(json.Number)
	if !ok {
		t.Fatalf("expected count as json.Number, got %T", args["count"])
	}
	if count.String() != "9007199254740993" {
		t.Fatalf("expected exact numeric argument preserved, got %s", count.String())
	}
}
