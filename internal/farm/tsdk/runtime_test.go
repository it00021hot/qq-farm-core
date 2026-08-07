package tsdk_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/tsdk"
)

func TestInitAndEncryptRoundTrip(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	// internal/farm/tsdk → repo root
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	wasmPath := filepath.Join(root, "resource", "farm", "tsdk.wasm")

	dir := t.TempDir()
	rt, err := tsdk.New(tsdk.Config{
		WASMPath:  wasmPath,
		AccountID: "test",
		DataRoot:  dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Destroy()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := rt.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}

	in := []byte("abc")
	enc, err := rt.Encrypt(in)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if fmt.Sprintf("%x", enc) != "6776c3" {
		t.Fatalf("encrypt got %x want 6776c3", enc)
	}
	dec, err := rt.Decrypt(enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(dec) != string(in) {
		t.Fatalf("decrypt got %q want %q", dec, in)
	}
}
