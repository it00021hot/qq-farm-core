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
	req := &queuedRequest{method: "Login", ch: ch}
	client.mu.Lock()
	client.pending[7] = req
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

// TestClientMissingTypeSettlesPendingReply 对齐 rust gateway.rs dispatch 的回包
// 容错：部分大包（如 FriendService.GetAll）不带标准 Response type，MessageType
// 缺失但 client_seq 命中 pending 且 method 名一致时按回包完成。
func TestClientMissingTypeSettlesPendingReply(t *testing.T) {
	client := NewClient(Options{})
	ch := make(chan rpcResult, 1)
	req := &queuedRequest{method: "FriendService.GetAll", ch: ch}
	client.mu.Lock()
	client.pending[9] = req
	client.mu.Unlock()

	frame, err := proto.Marshal(&gatepb.Message{
		Meta: &gatepb.Meta{
			ServiceName: "gamepb.friendpb.FriendService",
			MethodName:  "FriendService.GetAll",
			ClientSeq:   9,
			// MessageType 缺失（proto3 缺省 = MessageType_None）
		},
		Body: []byte("big-payload"),
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
		if string(res.body) != "big-payload" {
			t.Fatalf("unexpected body: %q", res.body)
		}
	default:
		t.Fatal("expected missing-type frame to settle the pending request")
	}
}

// TestClientMissingTypeMethodMismatchDropped：method 名与 pending 请求不一致时
// 不容错完成（对齐 rust is_pending_reply 的 method 校验）。
func TestClientMissingTypeMethodMismatchDropped(t *testing.T) {
	client := NewClient(Options{})
	ch := make(chan rpcResult, 1)
	req := &queuedRequest{method: "Login", ch: ch}
	client.mu.Lock()
	client.pending[3] = req
	client.mu.Unlock()

	frame, err := proto.Marshal(&gatepb.Message{
		Meta: &gatepb.Meta{
			MethodName: "SomeOther.Method",
			ClientSeq:  3,
		},
		Body: []byte("x"),
	})
	if err != nil {
		t.Fatal(err)
	}
	client.handleFrame(frame)

	select {
	case res := <-ch:
		t.Fatalf("frame should not settle pending request, got %+v", res)
	default:
	}
	if got := client.PendingCount(); got != 1 {
		t.Fatalf("pending should be untouched, got %d", got)
	}
}

// TestClientMissingTypeUnknownSeqDropped：client_seq == 0 或未命中 pending 的
// 帧（对齐 rust 的 client_seq != 0 前置条件）不得影响任何请求。
func TestClientMissingTypeUnknownSeqDropped(t *testing.T) {
	client := NewClient(Options{})
	ch := make(chan rpcResult, 1)
	req := &queuedRequest{method: "Login", ch: ch}
	client.mu.Lock()
	client.pending[5] = req
	client.mu.Unlock()

	frame, err := proto.Marshal(&gatepb.Message{
		Meta: &gatepb.Meta{ClientSeq: 42}, // 无 pending 的 seq
		Body: []byte("x"),
	})
	if err != nil {
		t.Fatal(err)
	}
	client.handleFrame(frame)

	select {
	case res := <-ch:
		t.Fatalf("unknown-seq frame should not settle pending request, got %+v", res)
	default:
	}
	if got := client.PendingCount(); got != 1 {
		t.Fatalf("pending should be untouched, got %d", got)
	}
}

// TestClientMissingTypeErrorCodeFails：容错完成的帧带非 0 错误码时按失败交付
// （对齐 rust handle_response 的 error_code 分支）。
func TestClientMissingTypeErrorCodeFails(t *testing.T) {
	client := NewClient(Options{})
	ch := make(chan rpcResult, 1)
	req := &queuedRequest{method: "Login", ch: ch}
	client.mu.Lock()
	client.pending[6] = req
	client.mu.Unlock()

	frame, err := proto.Marshal(&gatepb.Message{
		Meta: &gatepb.Meta{
			ServiceName:  "gamepb.userpb.UserService",
			ErrorMessage: "denied",
			ErrorCode:    1001,
			ClientSeq:    6,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	client.handleFrame(frame)

	select {
	case res := <-ch:
		if res.err == nil {
			t.Fatal("expected error for non-zero error code")
		}
	default:
		t.Fatal("expected the pending request to settle with an error")
	}
}
