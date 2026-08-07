// Package seasonpb provides hand-written season service protobuf types.
package seasonpb

// SeasonItem is a reward item.
type SeasonItem struct {
	ItemID int64
	Count  int64
}

// SeasonRewardNode is one pass reward node.
type SeasonRewardNode struct {
	NodeID     int64
	Rewards    []SeasonItem
	IsKeyLevel bool
}

// SeasonActivity is a sub-activity in the current season.
type SeasonActivity struct {
	ActivityID int64
	Type       int64
	Name       []byte
	BeginTime  int64
	EndTime    int64
}

// SeasonPass is the battle pass state.
type SeasonPass struct {
	ActivityID          int64
	CurrentLevel        int64
	CurrentProgress     int64
	ProgressTarget      int64
	NodeCount           int64
	Nodes               []SeasonRewardNode
	ClaimedThroughLevel int64
	Title               []byte
	RulesJSON           []byte
}

// SeasonInfo is the current season snapshot.
type SeasonInfo struct {
	SeasonID   int64
	Name       []byte
	Status     int64
	BeginTime  int64
	EndTime    int64
	ServerTime int64
	Activities []SeasonActivity
	Pass       *SeasonPass
}

// GetSeasonInfoRequest fetches season info.
type GetSeasonInfoRequest struct{}

// GetSeasonInfoReply is the season info response.
type GetSeasonInfoReply struct {
	SeasonInfo *SeasonInfo
}

// ClaimBattlePassRewardsRequest claims pass rewards.
type ClaimBattlePassRewardsRequest struct{}

// ClaimBattlePassRewardsReply is the claim response.
type ClaimBattlePassRewardsReply struct {
	Rewards []SeasonItem
	Field2  []int64
	Pass    *SeasonPass
	Field4  bool
}

// BattlePassChangeNotify is a server push when battle pass changes.
type BattlePassChangeNotify struct {
	Pass *SeasonPass
}
