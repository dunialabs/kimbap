package controller

import "testing"

func TestIsInitializeRequestSingleObject(t *testing.T) {
	ok, valid := isInitializeRequest([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	if !valid {
		t.Fatal("expected valid JSON")
	}
	if !ok {
		t.Fatal("expected initialize request")
	}
}

func TestIsInitializeRequestBatchFirstInitialize(t *testing.T) {
	ok, valid := isInitializeRequest([]byte(`[{"jsonrpc":"2.0","id":1,"method":"initialize"}]`))
	if !valid {
		t.Fatal("expected valid JSON")
	}
	if !ok {
		t.Fatal("expected batch initialize request")
	}
}

func TestIsInitializeRequestInvalidJSON(t *testing.T) {
	ok, valid := isInitializeRequest([]byte(`{"jsonrpc"`))
	if valid {
		t.Fatal("expected invalid JSON")
	}
	if ok {
		t.Fatal("expected non-initialize for invalid JSON")
	}
}
