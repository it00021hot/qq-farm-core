package protocol

import (
	"sync"
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/gatepb"
	"google.golang.org/protobuf/proto"
)

func TestClientOnNotify(t *testing.T) {
	var (
		mu       sync.Mutex
		gotSvc   string
		gotMeth  string
		gotBody  []byte
		notified bool
	)
	client := NewClient(Options{
		OnNotify: func(service, method string, body []byte) {
			mu.Lock()
			defer mu.Unlock()
			gotSvc = service
			gotMeth = method
			gotBody = append([]byte(nil), body...)
			notified = true
		},
	})

	frame, err := proto.Marshal(&gatepb.Message{
		Meta: &gatepb.Meta{
			ServiceName: "gamepb.plantpb.PlantService",
			MethodName:  "LandsNotify",
			MessageType: int32(gatepb.MessageType_Notify),
			ServerSeq:   1,
		},
		Body: []byte("push-body"),
	})
	if err != nil {
		t.Fatal(err)
	}

	client.handleFrame(frame)

	mu.Lock()
	defer mu.Unlock()
	if !notified {
		t.Fatal("expected OnNotify to be invoked")
	}
	if gotSvc != "gamepb.plantpb.PlantService" || gotMeth != "LandsNotify" {
		t.Fatalf("unexpected notify target: %s.%s", gotSvc, gotMeth)
	}
	if string(gotBody) != "push-body" {
		t.Fatalf("unexpected body: %q", gotBody)
	}
}

func TestClientResponseStillHandled(t *testing.T) {
	client := NewClient(Options{})
	ch := make(chan rpcResult, 1)
	client.mu.Lock()
	client.pending[7] = ch
	client.mu.Unlock()

	frame, err := proto.Marshal(&gatepb.Message{
		Meta: &gatepb.Meta{
			ServiceName: "gamepb.userpb.UserService",
			MethodName:  "Login",
			MessageType: int32(gatepb.MessageType_Response),
			ClientSeq:   7,
		},
		Body: []byte("ok"),
	})
	if err != nil {
		t.Fatal(err)
	}
	client.handleFrame(frame)

	select {
	case res := <-ch:
		if res.err != nil {
			t.Fatalf("unexpected error: %v", res.err)
		}
		if string(res.body) != "ok" {
			t.Fatalf("unexpected body: %q", res.body)
		}
	default:
		t.Fatal("expected response on pending channel")
	}
}
