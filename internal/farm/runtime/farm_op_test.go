package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/MQEnergy/go-skeleton/internal/farm/game"
	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/gatepb"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/plantpb"
	"google.golang.org/protobuf/proto"
)

type farmOpSender struct {
	mu           sync.Mutex
	methods      []string
	bodies       map[string][]byte
	lands        []byte
	harvestReply []byte
	plantReply   []byte
	allLandsN    int
	failAllLands bool
	failAfter    map[string]int // method → succeed N times then fail
	callCount    map[string]int
}

func (s *farmOpSender) Send(_ context.Context, _ string, method string, body []byte) ([]byte, *gatepb.Meta, error) {
	s.mu.Lock()
	s.methods = append(s.methods, method)
	if s.bodies == nil {
		s.bodies = make(map[string][]byte)
	}
	if s.callCount == nil {
		s.callCount = make(map[string]int)
	}
	s.bodies[method] = append([]byte(nil), body...)
	s.callCount[method]++
	n := s.callCount[method]
	failAfter := 0
	if s.failAfter != nil {
		failAfter = s.failAfter[method]
	}
	s.mu.Unlock()

	if failAfter > 0 && n > failAfter {
		return nil, nil, errors.New("simulated failure")
	}

	switch method {
	case "AllLands":
		s.mu.Lock()
		s.allLandsN++
		fail := s.failAllLands && s.allLandsN > 1
		lands := s.lands
		s.mu.Unlock()
		if fail {
			return nil, nil, errors.New("refresh failed")
		}
		return lands, nil, nil
	case "Harvest":
		if s.harvestReply != nil {
			return s.harvestReply, nil, nil
		}
		return nil, nil, nil
	case "Plant":
		if s.plantReply != nil {
			return s.plantReply, nil, nil
		}
		return nil, nil, nil
	}
	return nil, nil, nil
}

func (s *farmOpSender) called(method string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, got := range s.methods {
		if got == method {
			return true
		}
	}
	return false
}

func (s *farmOpSender) body(method string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.bodies[method]...)
}

func (s *farmOpSender) methodCount(method string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.callCount[method]
}

func TestRunFarmOperationCallsFarmingAndHarvest(t *testing.T) {
	reply, err := proto.Marshal(&plantpb.AllLandsReply{Lands: []*plantpb.LandInfo{
		{
			Id:       1,
			Unlocked: true,
			Plant: &plantpb.PlantInfo{Phases: []*plantpb.PlantPhaseInfo{
				{Phase: 1, BeginTime: 1},
				{Phase: 6, BeginTime: 2},
			}},
		},
		{
			Id:       2,
			Unlocked: true,
			Plant: &plantpb.PlantInfo{
				WeedOwners: []int64{99},
				Phases:     []*plantpb.PlantPhaseInfo{{Phase: 1, BeginTime: 1}},
			},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	sender := &farmOpSender{lands: reply}
	api := &game.API{Sender: sender, GID: 42}
	cfg := logic.DefaultAccountConfig()
	cfg.Automation.Fertilizer = "none"
	cfg.Automation.LandUpgrade = false

	hadWork, actions, _, err := RunFarmOperation(context.Background(), api, cfg, "all")
	if err != nil {
		t.Fatalf("RunFarmOperation: %v", err)
	}
	if !hadWork || len(actions) != 2 {
		t.Fatalf("hadWork=%v actions=%v", hadWork, actions)
	}
	if !sender.called("Farming") || !sender.called("Harvest") {
		t.Fatalf("expected Farming and Harvest, methods=%v", sender.methods)
	}
	var farming plantpb.FarmingRequest
	if err := proto.Unmarshal(sender.body("Farming"), &farming); err != nil {
		t.Fatalf("decode Farming request: %v", err)
	}
	if farming.HostGid != 42 || len(farming.LandIds) != 1 || farming.LandIds[0] != 2 {
		t.Fatalf("unexpected Farming request: %+v", farming)
	}
	var harvest plantpb.HarvestRequest
	if err := proto.Unmarshal(sender.body("Harvest"), &harvest); err != nil {
		t.Fatalf("decode Harvest request: %v", err)
	}
	if harvest.HostGid != 42 || len(harvest.LandIds) != 1 || harvest.LandIds[0] != 1 {
		t.Fatalf("unexpected Harvest request: %+v", harvest)
	}
}

func TestResolveRemovableUsesReplyOnRefreshFail(t *testing.T) {
	// Growing land in harvest reply must not be shoveled when refresh fails.
	harvested := []int64{1, 2}
	replyLands := []logic.LandInfo{
		{ID: 1, Unlocked: true, Plant: &logic.PlantInfo{Phases: []logic.PlantPhaseInfo{{Phase: logic.PhaseSeed, BeginTime: 1}}}},
		{ID: 2, Unlocked: true, Plant: &logic.PlantInfo{Phases: []logic.PlantPhaseInfo{{Phase: logic.PhaseDead, BeginTime: 1}}}},
	}
	resolved := logic.ResolveRemovableHarvestedLandsPure(harvested, replyLands, nil)
	if len(resolved.Growing) != 1 || resolved.Growing[0] != 1 {
		t.Fatalf("growing=%v want [1]", resolved.Growing)
	}
	if len(resolved.Removable) != 1 || resolved.Removable[0] != 2 {
		t.Fatalf("removable=%v want [2]", resolved.Removable)
	}
}

func TestHarvestDecodesReplyLands(t *testing.T) {
	sender := &harvestDecodeSender{reply: mustMarshalHarvestReply(t, []*plantpb.LandInfo{
		{Id: 7, Unlocked: true, Plant: &plantpb.PlantInfo{Phases: []*plantpb.PlantPhaseInfo{{Phase: 1, BeginTime: 1}}}},
	})}
	api := &game.API{Sender: sender, GID: 1}
	lands, err := api.Harvest(context.Background(), []int64{7})
	if err != nil {
		t.Fatalf("Harvest: %v", err)
	}
	if len(lands) != 1 || lands[0].ID != 7 || lands[0].Plant == nil {
		t.Fatalf("unexpected lands: %+v", lands)
	}
}

type harvestDecodeSender struct {
	reply []byte
}

func (s *harvestDecodeSender) Send(_ context.Context, _ string, method string, _ []byte) ([]byte, *gatepb.Meta, error) {
	if method != "Harvest" {
		return nil, nil, errors.New("unexpected method " + method)
	}
	return s.reply, nil, nil
}

func mustMarshalHarvestReply(t *testing.T, lands []*plantpb.LandInfo) []byte {
	t.Helper()
	body, err := proto.Marshal(&plantpb.HarvestReply{Land: lands})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestConfirmsPlantedFootprintHelper(t *testing.T) {
	lands := []logic.LandInfo{
		{ID: 1, Unlocked: true, Plant: &logic.PlantInfo{ID: 99, Phases: []logic.PlantPhaseInfo{{Phase: 1, BeginTime: 1}}}},
	}
	if !logic.ConfirmsPlantedFootprint([]int64{1}, 1, []int64{1}, lands) {
		t.Fatal("expected confirmed footprint")
	}
	if logic.ConfirmsPlantedFootprint([]int64{1, 2}, 1, []int64{1}, lands) {
		t.Fatal("expected unconfirmed when occupied missing expected id")
	}
}
