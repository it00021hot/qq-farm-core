// Package dogpb provides hand-written dog service protobuf types.
// Source: qq-farm-bot/core/src/proto/dogpb.proto
package dogpb

// DogInfo is one dog entry.
type DogInfo struct {
	ID     int64
	Name   string
	Price  int64
	Status int64
	Level  int64
}

// DogItem is a dog equipment/item.
type DogItem struct {
	ID       int64
	Duration int64
	Status   int64
}

// GetDogInfoRequest fetches dog info.
type GetDogInfoRequest struct{}

// GetDogInfoReply is the dog info result.
type GetDogInfoReply struct {
	Dogs             []*DogInfo
	ProtectDuration  int64
	Items            []*DogItem
}
