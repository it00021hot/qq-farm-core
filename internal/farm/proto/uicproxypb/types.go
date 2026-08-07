// Package uicproxypb provides hand-written UIC proxy protobuf types.
// Source: qq-farm-bot/core/src/proto/uicproxypb.proto
package uicproxypb

// TextResult is one text moderation result.
type TextResult struct {
	Status int32
	Text   string
	UID    string
}

// TextItem is one text moderation input.
type TextItem struct {
	UID  string
	Text string
}

// BatchModerateTextRequest batch-moderates texts.
type BatchModerateTextRequest struct {
	Items []*TextItem
}

// BatchModerateTextReply lists moderation results.
type BatchModerateTextReply struct {
	Results []*TextResult
}
