// Package skinpb provides hand-written skin service protobuf types.
// Source: qq-farm-bot/core/src/proto/skinpb.proto
package skinpb

// SkinItem is one skin entry.
type SkinItem struct {
	SkinID   int64
	SlotType int64
}

// SkinsOwnedRequest fetches owned skins.
type SkinsOwnedRequest struct{}

// SkinsOwnedReply lists owned skins.
type SkinsOwnedReply struct {
	Skins []*SkinItem
}

// SkinsEquippedRequest fetches equipped skins.
type SkinsEquippedRequest struct{}

// SkinsEquippedReply lists equipped skins.
type SkinsEquippedReply struct {
	Skins []*SkinItem
}

// EquipRequest equips a skin.
type EquipRequest struct {
	SkinID   int64
	SlotType int64
}

// EquipReply is the equip result.
type EquipReply struct {
	Skins []*SkinItem
}

// MarkAsViewedRequest marks skins as viewed.
type MarkAsViewedRequest struct {
	SkinIDs []int64
}

// MarkAsViewedReply is an empty mark result.
type MarkAsViewedReply struct{}

// SkinChangeNotify is a skin change push.
type SkinChangeNotify struct {
	Skins []*SkinItem
}
