// Package solartermspb provides hand-written solar terms protobuf types.
package solartermspb

type SolarTermReward struct {
	ItemID int64
	Count  int64
}

type SolarTermInfo struct {
	TermID    int64
	Status    int64
	BeginTime int64
	EndTime   int64
	Rewards   []SolarTermReward
	Name      []byte
}

type SolarTermsConfig struct {
	ConfigID   int64
	ActivityID int64
	RulesJSON  []byte
	Field4     []byte
}

type GetSolarTermsRequest struct{}

type GetSolarTermsReply struct {
	Terms         []SolarTermInfo
	ServerTime    int64
	CurrentConfig *SolarTermsConfig
	Configs       []SolarTermsConfig
}

type ClaimSolarTermsRequest struct {
	TermID int64
}

type ClaimSolarTermsReply struct {
	Rewards []SolarTermReward
	Term    *SolarTermInfo
}
