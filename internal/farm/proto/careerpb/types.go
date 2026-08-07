// Package careerpb provides hand-written career service protobuf types.
// Source: qq-farm-bot/core/src/proto/careerpb.proto
package careerpb

// CareerInfo is one career entry.
type CareerInfo struct {
	CareerID int64
	Name     string
	Level    int32
	Exp      int64
	Status   int32
}

// CareerInfoGetRequest fetches career info.
type CareerInfoGetRequest struct{}

// CareerInfoGetReply lists careers.
type CareerInfoGetReply struct {
	Careers []*CareerInfo
}
