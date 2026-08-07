// Package emailpb provides hand-written email service protobuf types.
// Source: qq-farm-bot/core/src/proto/emailpb.proto
package emailpb

import "github.com/MQEnergy/go-skeleton/internal/farm/proto/corepb"

// GetEmailListRequest lists emails in a box.
type GetEmailListRequest struct {
	BoxType int32
}

// EmailItem is one email entry.
type EmailItem struct {
	ID        string
	MailType  int32
	Title     string
	Claimed   bool
	HasReward bool
	Subtitle  string
}

// GetEmailListReply lists emails.
type GetEmailListReply struct {
	Emails []EmailItem
}

// ReadEmailRequest marks an email as read.
type ReadEmailRequest struct {
	BoxType int32
	EmailID string
}

// ReadEmailReply is the read result.
type ReadEmailReply struct{}

// ClaimEmailRequest claims one email.
type ClaimEmailRequest struct {
	BoxType int32
	EmailID string
}

// ClaimEmailReply is the claim result.
type ClaimEmailReply struct {
	Items []*corepb.Item
}

// BatchClaimEmailRequest batch-claims emails.
type BatchClaimEmailRequest struct {
	BoxType int32
	EmailID string
}

// BatchClaimEmailReply is the batch claim result.
type BatchClaimEmailReply struct {
	Items []*corepb.Item
}

// BatchDeleteEmailRequest deletes emails in batch.
type BatchDeleteEmailRequest struct {
	BoxType  int32
	EmailIDs []string
}

// BatchDeleteEmailReply is the delete result.
type BatchDeleteEmailReply struct {
	Success bool
}
