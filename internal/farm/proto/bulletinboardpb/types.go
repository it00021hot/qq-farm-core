// Package bulletinboardpb provides hand-written bulletin board protobuf types.
// Source: qq-farm-bot/core/src/proto/bulletinboardpb.proto
package bulletinboardpb

// BulletinItem is one bulletin list entry.
type BulletinItem struct {
	ID     int64
	Title  string
	Status int64
	Type   int64
}

// GetBulletinListRequest fetches bulletin list.
type GetBulletinListRequest struct {
	Count int64
}

// GetBulletinListReply lists bulletins.
type GetBulletinListReply struct {
	Bulletins []*BulletinItem
}

// GetBulletinDetailRequest fetches bulletin detail.
type GetBulletinDetailRequest struct {
	ID int64
}

// BulletinDetail is full bulletin content.
type BulletinDetail struct {
	ID      int64
	Title   string
	Content string
	Status  int64
	Type    int64
}

// GetBulletinDetailReply is the detail result.
type GetBulletinDetailReply struct {
	Detail *BulletinDetail
}

// BulletinListChangedNTF notifies bulletin list changes.
type BulletinListChangedNTF struct {
	Bulletins []*BulletinItem
}
