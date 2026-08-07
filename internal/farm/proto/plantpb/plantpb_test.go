package plantpb_test

import (
	"bytes"
	"testing"

	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/plantpb"
	"google.golang.org/protobuf/encoding/protowire"
)

func roundTripRequest[T interface {
	Marshal() []byte
	Unmarshal([]byte) error
}](t *testing.T, name string, orig T, clone T) {
	t.Helper()
	raw := orig.Marshal()
	if err := clone.Unmarshal(raw); err != nil {
		t.Fatalf("%s unmarshal: %v", name, err)
	}
	raw2 := clone.Marshal()
	if !bytes.Equal(raw, raw2) {
		t.Fatalf("%s round-trip bytes differ:\n  orig=% x\n  back=% x", name, raw, raw2)
	}
}

func TestAllLandsRequestRoundTrip(t *testing.T) {
	roundTripRequest(t, "AllLandsRequest(empty)", &plantpb.AllLandsRequest{}, &plantpb.AllLandsRequest{})
	roundTripRequest(t, "AllLandsRequest", &plantpb.AllLandsRequest{HostGID: 12345}, &plantpb.AllLandsRequest{})
}

func TestHarvestRequestRoundTrip(t *testing.T) {
	req := &plantpb.HarvestRequest{
		LandIDs: []int64{1, 2, 3},
		HostGID: 999,
		IsAll:   true,
	}
	roundTripRequest(t, "HarvestRequest", req, &plantpb.HarvestRequest{})

	body := req.Marshal()
	if len(body) == 0 || body[0] != 0x0a {
		t.Fatalf("expected packed land_ids tag 0x0a, got % x", body)
	}
}

func TestFarmingRequestRoundTrip(t *testing.T) {
	req := &plantpb.FarmingRequest{
		LandIDs: []int64{5, 6},
		HostGID: 888,
		Field3:  0,
		Field4:  2,
	}
	roundTripRequest(t, "FarmingRequest", req, &plantpb.FarmingRequest{})
}

func TestFertilizeRequestRoundTrip(t *testing.T) {
	req := &plantpb.FertilizeRequest{
		LandIDs:      []int64{7},
		FertilizerID: 1011,
	}
	roundTripRequest(t, "FertilizeRequest", req, &plantpb.FertilizeRequest{})
}

func TestPlantRequestRoundTripAndBotEncoding(t *testing.T) {
	req := &plantpb.PlantRequest{
		Items: []plantpb.PlantItem{
			{SeedID: 2001, LandIDs: []int64{10, 11, 12}},
		},
	}
	raw := req.Marshal()
	if len(raw) == 0 || raw[0] != 0x12 {
		t.Fatalf("PlantRequest should start with field 2 tag 0x12, got % x", raw)
	}

	var back plantpb.PlantRequest
	if err := back.Unmarshal(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back.Items) != 1 || back.Items[0].SeedID != 2001 || len(back.Items[0].LandIDs) != 3 {
		t.Fatalf("decoded=%+v", back)
	}
	raw2 := back.Marshal()
	if !bytes.Equal(raw, raw2) {
		t.Fatalf("round-trip bytes differ:\n  orig=% x\n  back=% x", raw, raw2)
	}
}

func TestRemoveUnlockUpgradeRequestsRoundTrip(t *testing.T) {
	roundTripRequest(t, "RemovePlantRequest", &plantpb.RemovePlantRequest{LandIDs: []int64{1, 2}}, &plantpb.RemovePlantRequest{})
	roundTripRequest(t, "UnlockLandRequest", &plantpb.UnlockLandRequest{LandID: 4, DoShared: true}, &plantpb.UnlockLandRequest{})
	roundTripRequest(t, "UpgradeLandRequest", &plantpb.UpgradeLandRequest{LandID: 5}, &plantpb.UpgradeLandRequest{})
	roundTripRequest(t, "WaterLandRequest", &plantpb.WaterLandRequest{LandIDs: []int64{3}, HostGID: 1}, &plantpb.WaterLandRequest{})
}

func TestPutInsectsPutWeedsCheckCanOperateRoundTrip(t *testing.T) {
	roundTripRequest(t, "PutInsectsRequest", &plantpb.PutInsectsRequest{HostGID: 100, LandIDs: []int64{1, 2}}, &plantpb.PutInsectsRequest{})
	roundTripRequest(t, "PutWeedsRequest", &plantpb.PutWeedsRequest{HostGID: 200, LandIDs: []int64{3}}, &plantpb.PutWeedsRequest{})
	roundTripRequest(t, "CheckCanOperateRequest", &plantpb.CheckCanOperateRequest{HostGID: 300, OperationID: 10008}, &plantpb.CheckCanOperateRequest{})
}

func sampleAllLandsReply() *plantpb.AllLandsReply {
	fert := int64(2)
	return &plantpb.AllLandsReply{
		Lands: []*plantpb.LandInfo{
			{ID: 1, Unlocked: true, Level: 1, MaxLevel: 3},
			{
				ID: 2, Unlocked: true, Level: 2, MaxLevel: 4,
				CouldUpgrade: true,
				MasterLandID: 2,
				SlaveLandIDs: []int64{3, 4},
				LandSize:     2,
				LandsLevel:   2,
				Plant: &plantpb.PlantInfo{
					ID: 1001, Name: "小麦",
					Season: 1, DryNum: 0,
					WeedOwners:         []int64{99},
					Stealable:          true,
					LeftInorcFertTimes: &fert,
					Activity: &plantpb.PlantActivityInfo{
						ActivityID: 1019, Param1: 240, Param2: 191, Date: 2026060100,
					},
					Phases: []plantpb.PlantPhaseInfo{
						{Phase: 1, BeginTime: 1_600_000_000},
						{Phase: 6, BeginTime: 1_600_001_000, DryTime: 1_600_002_000, WeedsTime: 1_600_003_000, InsectTime: 1_600_004_000},
					},
				},
			},
			{ID: 3, Unlocked: false, CouldUnlock: true},
		},
		OperationLimits: []*plantpb.OperationLimit{
			{ID: 10001, DayTimes: 5, DayTimesLt: 100},
		},
	}
}

func TestAllLandsReplyDecodeAndMapping(t *testing.T) {
	orig := sampleAllLandsReply()
	raw := orig.Marshal()

	var reply plantpb.AllLandsReply
	if err := reply.Unmarshal(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(reply.Lands) != 3 {
		t.Fatalf("lands=%d", len(reply.Lands))
	}
	if reply.Lands[1].Plant == nil || len(reply.Lands[1].Plant.Phases) != 2 {
		t.Fatalf("plant phases missing: %+v", reply.Lands[1].Plant)
	}
	if reply.Lands[1].Plant.LeftInorcFertTimes == nil || *reply.Lands[1].Plant.LeftInorcFertTimes != 2 {
		t.Fatalf("left_inorc_fert_times=%v", reply.Lands[1].Plant.LeftInorcFertTimes)
	}
	if len(reply.OperationLimits) != 1 || reply.OperationLimits[0].ID != 10001 {
		t.Fatalf("limits=%+v", reply.OperationLimits)
	}

	logic.SyncServerTime(1_700_000_000_000)
	lands := plantpb.LandsToLogic(reply.Lands)
	if len(lands) != 3 {
		t.Fatalf("mapped lands=%d", len(lands))
	}
	if lands[1].Plant == nil || lands[1].Plant.Name != "小麦" {
		t.Fatalf("mapped plant=%+v", lands[1].Plant)
	}
	if lands[1].Plant.Activity == nil || lands[1].Plant.Activity.Param1 != 240 {
		t.Fatalf("mapped activity=%+v", lands[1].Plant.Activity)
	}
	if !logic.IsActivityPlant(lands[1].Plant) {
		t.Fatal("expected activity plant")
	}
	if len(lands[1].SlaveLandIDs) != 2 || lands[1].LandSize != 2 {
		t.Fatalf("mapped land meta=%+v", lands[1])
	}

	result := logic.AnalyzeLands(lands)
	if len(result.Empty) != 1 || result.Empty[0] != 1 {
		t.Fatalf("empty=%v", result.Empty)
	}
	if len(result.Harvestable) != 1 || result.Harvestable[0] != 2 {
		t.Fatalf("harvestable=%v", result.Harvestable)
	}
	if len(result.Unlockable) != 1 || result.Unlockable[0] != 3 {
		t.Fatalf("unlockable=%v", result.Unlockable)
	}
	if len(result.Upgradable) != 1 || result.Upgradable[0] != 2 {
		t.Fatalf("upgradable=%v", result.Upgradable)
	}
}

func TestLandsNotifyRoundTrip(t *testing.T) {
	notify := &plantpb.LandsNotify{
		HostGID: 42,
		Lands:   sampleAllLandsReply().Lands[:1],
	}
	raw := notify.Marshal()
	var back plantpb.LandsNotify
	if err := back.Unmarshal(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.HostGID != 42 || len(back.Lands) != 1 || back.Lands[0].ID != 1 {
		t.Fatalf("notify=%+v", back)
	}
}

func TestAllLandsReplySkipsUnknownFields(t *testing.T) {
	var b []byte
	b = protowire.AppendTag(b, 99, protowire.VarintType)
	b = protowire.AppendVarint(b, 42)

	land := &plantpb.LandInfo{ID: 7, Unlocked: true}
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendBytes(b, land.Marshal())

	var reply plantpb.AllLandsReply
	if err := reply.Unmarshal(b); err != nil {
		t.Fatalf("unmarshal with unknown field: %v", err)
	}
	if len(reply.Lands) != 1 || reply.Lands[0].ID != 7 {
		t.Fatalf("lands=%+v", reply.Lands)
	}
}

func TestPlantInfoStealFieldsRoundTrip(t *testing.T) {
	orig := &plantpb.PlantInfo{
		ID:           1001,
		Name:         "tomato",
		FruitID:      2001,
		FruitNum:     5,
		Stealers:     []byte{0x01, 0x02, 0x03},
		Stealable:    true,
		LeftFruitNum: 3,
		StoleNum:     2,
		GrowSec:      3600,
	}
	raw := orig.Marshal()
	var back plantpb.PlantInfo
	if err := unmarshalPlantInfoForTest(&back, raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.FruitID != 2001 || back.FruitNum != 5 || back.LeftFruitNum != 3 {
		t.Fatalf("fruit fields: id=%d num=%d left=%d", back.FruitID, back.FruitNum, back.LeftFruitNum)
	}
	if !bytes.Equal(back.Stealers, orig.Stealers) {
		t.Fatalf("stealers=%x", back.Stealers)
	}
	if !back.Stealable || back.StoleNum != 2 {
		t.Fatalf("stealable=%v stole=%d", back.Stealable, back.StoleNum)
	}
	mapped := plantpb.LandsToLogic([]*plantpb.LandInfo{{ID: 1, Plant: &back}})
	if len(mapped) != 1 || mapped[0].Plant == nil {
		t.Fatal("mapping failed")
	}
	p := mapped[0].Plant
	if p.FruitID != 2001 || p.FruitNum != 5 || p.LeftFruitNum != 3 || !bytes.Equal(p.Stealers, orig.Stealers) {
		t.Fatalf("logic mapping: %+v", p)
	}
}

// unmarshalPlantInfoForTest exercises LandInfo plant decode path.
func unmarshalPlantInfoForTest(m *plantpb.PlantInfo, data []byte) error {
	land := &plantpb.LandInfo{}
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, 1)
	b = protowire.AppendTag(b, 10, protowire.BytesType)
	b = protowire.AppendBytes(b, data)
	if err := land.Unmarshal(b); err != nil {
		return err
	}
	if land.Plant == nil {
		*m = plantpb.PlantInfo{}
		return nil
	}
	*m = *land.Plant
	return nil
}

func TestPutSocialItemRequestRoundTrip(t *testing.T) {
	roundTripRequest(t, "PutSocialItemRequest", &plantpb.PutSocialItemRequest{
		HostGID: 99,
		LandID:  7,
		ItemID:  301101,
	}, &plantpb.PutSocialItemRequest{})
}
