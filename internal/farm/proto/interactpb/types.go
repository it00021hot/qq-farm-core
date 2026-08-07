// Package interactpb provides hand-written interact service protobuf types.
// Source: qq-farm-bot/core/src/proto/interactpb.proto
package interactpb

// InteractRecordExtra is InteractRecord.Extra.
type InteractRecordExtra struct {
	LandID int32
	Flag1  int32
	Flag2  int32
}

// InteractRecord is one visitor interaction record.
type InteractRecord struct {
	ServerTime int64
	ActionType int32
	VisitorGID int64
	Nick       string
	AvatarURL  string
	CropID     int32
	CropCount  int32
	Times      int32
	FromType   int32
	Level      int32
	Extra      *InteractRecordExtra
}

// InteractRecordsRequest fetches interaction records.
type InteractRecordsRequest struct{}

// InteractRecordsReply lists interaction records.
type InteractRecordsReply struct {
	Records []*InteractRecord
}

// InteractInfo is one interaction info entry.
type InteractInfo struct {
	VisitorGID int64
	Nick       string
	AvatarURL  string
	ActionType int32
	ServerTime int64
	CropID     int32
	CropCount  int32
	Level      int32
}

// GetInteractInfoRequest fetches interaction info.
type GetInteractInfoRequest struct{}

// GetInteractInfoReply lists interaction info.
type GetInteractInfoReply struct {
	Infos []*InteractInfo
}

// InteractSummary aggregates interaction stats.
type InteractSummary struct {
	TotalWater       int64
	TotalInsecticide int64
	TotalWeed        int64
	TotalSteal       int64
	TotalStolen      int64
	VisitorCount     int64
}

// GetInteractSummaryRequest fetches interaction summary.
type GetInteractSummaryRequest struct{}

// GetInteractSummaryReply is the summary result.
type GetInteractSummaryReply struct {
	Summary *InteractSummary
}

// DismissInteractPopupRequest dismisses the interact popup.
type DismissInteractPopupRequest struct{}

// DismissInteractPopupReply is an empty dismiss result.
type DismissInteractPopupReply struct{}
