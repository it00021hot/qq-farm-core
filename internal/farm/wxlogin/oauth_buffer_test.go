package wxlogin

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLoginBufferPayloadAndSignature(t *testing.T) {
	payload, err := loginBufferPayload("oid", "tok")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, `"unionid"`) || !strings.Contains(payload, "oid") || !strings.Contains(payload, "tok") {
		t.Fatalf("payload=%s", payload)
	}
	timestamp := "1710000000000"
	nonce := "1234"
	got := loginBufferSignature(payload, timestamp, nonce)
	sum := md5.Sum([]byte(payload + timestamp + loginBufferAccessKey + nonce))
	want := hex.EncodeToString(sum[:])
	if got != want {
		t.Fatalf("sig=%s want=%s", got, want)
	}
}

func TestParseLoginBufferJSON(t *testing.T) {
	buf, err := parseLoginBufferJSON([]byte(`{"code":0,"ext_info":{"list_s":{"login_buffer":{"value":["abc"]}}}}`))
	if err != nil || buf != "abc" {
		t.Fatalf("buf=%q err=%v", buf, err)
	}
	if _, err := parseLoginBufferJSON([]byte(`{"code":1,"ext_info":{}}`)); err == nil {
		t.Fatal("invalid response should fail")
	}
	if _, err := parseLoginBufferJSON([]byte(`{"code":0,"ext_info":{"list_s":{"login_buffer":{"value":[""]}}}}`)); err == nil {
		t.Fatal("empty buffer should fail")
	}
}

func TestRequestAllowsNilCookieJar(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	t.Cleanup(srv.Close)
	status, body, err := request(context.Background(), nil, srv.URL, http.MethodPost, []byte(`{}`), nil, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || string(body) != `{"code":0}` {
		t.Fatalf("status=%d body=%s", status, body)
	}
}
