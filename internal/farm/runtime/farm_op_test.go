package runtime

import (
	"context"
	"errors"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/it00021hot/qq-farm-core/internal/farm/game"
	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/corepb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/gatepb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/itempb"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/plantpb"
	"google.golang.org/protobuf/proto"
)

type farmOpSender struct {
	mu             sync.Mutex
	methods        []string
	bodies         map[string][]byte
	history        map[string][][]byte
	lands          []byte
	harvestReply   []byte
	plantReply     []byte
	bag            []byte
	allLandsN      int
	failAllLands   bool
	failAfter      map[string]int // method → succeed N times then fail
	callCount      map[string]int
	autoPlantReply bool // synthesize a PlantReply that confirms the request footprint
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
	if s.history == nil {
		s.history = make(map[string][][]byte)
	}
	s.bodies[method] = append([]byte(nil), body...)
	s.history[method] = append(s.history[method], append([]byte(nil), body...))
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
	case "Bag":
		s.mu.Lock()
		bag := s.bag
		s.mu.Unlock()
		return bag, nil, nil
	case "Harvest":
		if s.harvestReply != nil {
			return s.harvestReply, nil, nil
		}
		return nil, nil, nil
	case "Plant":
		if s.plantReply != nil {
			return s.plantReply, nil, nil
		}
		if s.autoPlantReply {
			var req plantpb.PlantRequest
			if err := proto.Unmarshal(body, &req); err == nil {
				var planted []*plantpb.LandInfo
				for _, item := range req.Items {
					for _, id := range item.LandIds {
						planted = append(planted, &plantpb.LandInfo{
							Id:       id,
							Unlocked: true,
							Plant: &plantpb.PlantInfo{Phases: []*plantpb.PlantPhaseInfo{
								{Phase: 1, BeginTime: 1},
							}},
						})
					}
				}
				if out, mErr := proto.Marshal(&plantpb.PlantReply{Land: planted}); mErr == nil {
					return out, nil, nil
				}
			}
		}
		return nil, nil, nil
	}
	return nil, nil, nil
}

func (s *farmOpSender) SendNoReply(_ context.Context, _ string, method string, body []byte) error {
	_, _, err := s.Send(context.Background(), "", method, body)
	return err
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

// historyFor returns every request body sent for a method, in order.
func (s *farmOpSender) historyFor(method string) [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([][]byte(nil), s.history[method]...)
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
	cfg.Automation.SkipOwnWeedBug = false // 默认 true 会跳过一键务农，这里显式验证务农+收获路径

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

func (s *harvestDecodeSender) SendNoReply(context.Context, string, string, []byte) error {
	return errors.New("unexpected SendNoReply")
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

func TestRunFarmOperationCleansInteractionItems(t *testing.T) {
	reply, err := proto.Marshal(&plantpb.AllLandsReply{
		Lands: []*plantpb.LandInfo{
			{
				Id: 1, Unlocked: true,
				Plant: &plantpb.PlantInfo{
					InteractionUses: []*plantpb.PlantInteractionUseInfo{{ItemId: 301101}},
					Phases:          []*plantpb.PlantPhaseInfo{{Phase: 1, BeginTime: 1}},
				},
			},
			{
				Id: 2, Unlocked: true,
				Plant: &plantpb.PlantInfo{
					InteractionTargets: []*plantpb.PlantInteractionTargetInfo{{ItemId: 5006}},
					Phases:             []*plantpb.PlantPhaseInfo{{Phase: 1, BeginTime: 1}},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	sender := &farmOpSender{lands: reply}
	api := &game.API{Sender: sender, GID: 7}
	cfg := logic.DefaultAccountConfig()
	cfg.Automation.Fertilizer = "none"
	cfg.Automation.LandUpgrade = false
	cfg.Automation.SkipOwnWeedBug = false

	_, actions, _, err := RunFarmOperation(context.Background(), api, cfg, "all")
	if err != nil {
		t.Fatalf("RunFarmOperation: %v", err)
	}
	var farming plantpb.FarmingRequest
	if err := proto.Unmarshal(sender.body("Farming"), &farming); err != nil {
		t.Fatalf("decode Farming request: %v", err)
	}
	if len(farming.LandIds) != 2 || farming.LandIds[0] != 1 || farming.LandIds[1] != 2 {
		t.Fatalf("unexpected Farming land ids: %+v", farming.LandIds)
	}
	if len(farming.SocialEventItemIds) != 0 {
		t.Fatalf("expected no social event ids, got %+v", farming.SocialEventItemIds)
	}
	found := false
	for _, action := range actions {
		if strings.Contains(action, "道具2") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 务农 action with 道具 part, actions=%v", actions)
	}
}

func loadRuntimeGameConfig(t *testing.T) {
	t.Helper()
	_, thisFile, _, ok := goruntime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	if err := logic.LoadGameConfig(filepath.Join(root, "resource", "farm", "gameConfig")); err != nil {
		t.Fatal(err)
	}
}

// pickSizeOneSeedIDs returns two distinct seed ids whose plant size is 1
// (single-land footprint keeps layout planning deterministic in tests).
func pickSizeOneSeedIDs(t *testing.T) (int64, int64) {
	t.Helper()
	var a, b int64
	for _, seed := range logic.GetAllSeeds() {
		if seed.SeedID <= 0 || logic.GetPlantSizeBySeedID(seed.SeedID) != 1 {
			continue
		}
		if a == 0 {
			a = seed.SeedID
			continue
		}
		if seed.SeedID != a {
			b = seed.SeedID
			break
		}
	}
	if a == 0 || b == 0 {
		t.Fatal("game config lacks two size-1 seeds")
	}
	return a, b
}

func mustMarshalAllLands(t *testing.T, lands []*plantpb.LandInfo) []byte {
	t.Helper()
	raw, err := proto.Marshal(&plantpb.AllLandsReply{Lands: lands})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func mustMarshalBag(t *testing.T, items []*corepb.Item) []byte {
	t.Helper()
	raw, err := proto.Marshal(&itempb.BagReply{ItemBag: &corepb.ItemBag{Items: items}})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// decodePlantRequests returns decoded Plant requests in send order.
func decodePlantRequests(t *testing.T, sender *farmOpSender) []plantpb.PlantRequest {
	t.Helper()
	var out []plantpb.PlantRequest
	for _, body := range sender.historyFor("Plant") {
		var req plantpb.PlantRequest
		if err := proto.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode Plant request: %v", err)
		}
		out = append(out, req)
	}
	return out
}

// TestBagPlantingRespectsLandTypes 对齐 rust plant_from_bag_seeds：受限种子先种
// 且只落在命中类型的空地；不限种子用剩余地块。
func TestBagPlantingRespectsLandTypes(t *testing.T) {
	loadRuntimeGameConfig(t)
	unrestrictedSeed, restrictedSeed := pickSizeOneSeedIDs(t)

	lands := mustMarshalAllLands(t, []*plantpb.LandInfo{
		{Id: 1, Unlocked: true, Level: 2}, // 红土地
		{Id: 2, Unlocked: true, Level: 1}, // 普通土地
	})
	bag := mustMarshalBag(t, []*corepb.Item{
		{Id: unrestrictedSeed, Count: 5},
		{Id: restrictedSeed, Count: 1},
	})
	sender := &farmOpSender{lands: lands, bag: bag, autoPlantReply: true}
	api := &game.API{Sender: sender, GID: 42}
	cfg := logic.DefaultAccountConfig()
	cfg.Automation.Fertilizer = "none"
	cfg.Automation.LandUpgrade = false
	cfg.BagSeedLandTypes = map[string][]string{
		strconv.FormatInt(restrictedSeed, 10): {logic.LandTypeRed},
	}

	hadWork, actions, _, err := RunFarmOperation(context.Background(), api, cfg, "plant")
	if err != nil {
		t.Fatalf("RunFarmOperation plant: %v", err)
	}
	if !hadWork {
		t.Fatalf("expected work, actions=%v", actions)
	}
	requests := decodePlantRequests(t, sender)
	if len(requests) != 2 {
		t.Fatalf("expected 2 Plant requests, got %d", len(requests))
	}
	first, second := requests[0], requests[1]
	if len(first.Items) != 1 || first.Items[0].SeedId != restrictedSeed {
		t.Fatalf("restricted seed must plant first, got %+v", first)
	}
	if got := first.Items[0].LandIds; len(got) != 1 || got[0] != 1 {
		t.Fatalf("restricted seed must land on red land 1 only, got %v", got)
	}
	if len(second.Items) != 1 || second.Items[0].SeedId != unrestrictedSeed {
		t.Fatalf("unrestricted seed must plant second, got %+v", second)
	}
	if got := second.Items[0].LandIds; len(got) != 1 || got[0] != 2 {
		t.Fatalf("unrestricted seed must take remaining land 2, got %v", got)
	}
}

// TestBagPlantingLandTypeResolveFailFallsBackUnrestricted 对齐 rust：土地类型
// 解析失败仅 warn，本轮按不限制处理，不能导致整轮种不下去。
func TestBagPlantingLandTypeResolveFailFallsBackUnrestricted(t *testing.T) {
	loadRuntimeGameConfig(t)
	unrestrictedSeed, restrictedSeed := pickSizeOneSeedIDs(t)

	lands := mustMarshalAllLands(t, []*plantpb.LandInfo{
		{Id: 1, Unlocked: true, Level: 2},
		{Id: 2, Unlocked: true, Level: 1},
	})
	bag := mustMarshalBag(t, []*corepb.Item{
		{Id: unrestrictedSeed, Count: 5},
		{Id: restrictedSeed, Count: 1},
	})
	sender := &farmOpSender{lands: lands, bag: bag, autoPlantReply: true, failAllLands: true}
	api := &game.API{Sender: sender, GID: 42}
	cfg := logic.DefaultAccountConfig()
	cfg.Automation.Fertilizer = "none"
	cfg.Automation.LandUpgrade = false
	cfg.BagSeedLandTypes = map[string][]string{
		strconv.FormatInt(restrictedSeed, 10): {logic.LandTypeRed},
	}

	if _, _, _, err := RunFarmOperation(context.Background(), api, cfg, "plant"); err != nil {
		t.Fatalf("resolve failure must not fail the round: %v", err)
	}
	planted := map[int64]bool{}
	for _, req := range decodePlantRequests(t, sender) {
		for _, item := range req.Items {
			for _, id := range item.LandIds {
				planted[id] = true
			}
		}
	}
	if !planted[1] || !planted[2] {
		t.Fatalf("expected unrestricted planting on both lands, got %v", planted)
	}
}

func TestRunFarmOperationWaterOp(t *testing.T) {
	lands := mustMarshalAllLands(t, []*plantpb.LandInfo{
		{Id: 1, Unlocked: true, Plant: &plantpb.PlantInfo{
			DryNum: 2,
			Phases: []*plantpb.PlantPhaseInfo{{Phase: 3, BeginTime: 1}},
		}},
		{Id: 2, Unlocked: true, Plant: &plantpb.PlantInfo{
			Phases: []*plantpb.PlantPhaseInfo{{Phase: 3, BeginTime: 1}},
		}},
	})
	sender := &farmOpSender{lands: lands}
	api := &game.API{Sender: sender, GID: 42}
	cfg := logic.DefaultAccountConfig()

	hadWork, actions, _, err := RunFarmOperation(context.Background(), api, cfg, "water")
	if err != nil {
		t.Fatalf("RunFarmOperation water: %v", err)
	}
	if !hadWork || len(actions) != 1 || actions[0] != "浇水1" {
		t.Fatalf("hadWork=%v actions=%v", hadWork, actions)
	}
	if !sender.called("WaterLand") {
		t.Fatalf("expected WaterLand, methods=%v", sender.methods)
	}
	var req plantpb.WaterLandRequest
	if err := proto.Unmarshal(sender.body("WaterLand"), &req); err != nil {
		t.Fatal(err)
	}
	if len(req.LandIds) != 1 || req.LandIds[0] != 1 || req.HostGid != 42 {
		t.Fatalf("unexpected WaterLand request: %+v", req)
	}
}

// TestRunFarmOperationWeedBugOps 对齐 rust op_weed/op_insecticide：两者数据源
// 相同（长草或生虫的地块），复用一键务农。
func TestRunFarmOperationWeedBugOps(t *testing.T) {
	lands := mustMarshalAllLands(t, []*plantpb.LandInfo{
		{Id: 1, Unlocked: true, Plant: &plantpb.PlantInfo{
			WeedOwners: []int64{9},
			Phases:     []*plantpb.PlantPhaseInfo{{Phase: 3, BeginTime: 1}},
		}},
		{Id: 2, Unlocked: true, Plant: &plantpb.PlantInfo{
			InsectOwners: []int64{8},
			Phases:       []*plantpb.PlantPhaseInfo{{Phase: 3, BeginTime: 1}},
		}},
		{Id: 3, Unlocked: true, Plant: &plantpb.PlantInfo{
			Phases: []*plantpb.PlantPhaseInfo{{Phase: 3, BeginTime: 1}},
		}},
	})
	for _, tc := range []struct {
		op    string
		label string
	}{{"weed", "除草2"}, {"bug", "除虫2"}, {"insecticide", "除虫2"}} {
		sender := &farmOpSender{lands: lands}
		api := &game.API{Sender: sender, GID: 42}
		hadWork, actions, _, err := RunFarmOperation(context.Background(), api, logic.DefaultAccountConfig(), tc.op)
		if err != nil {
			t.Fatalf("%s: %v", tc.op, err)
		}
		if !hadWork || len(actions) != 1 || actions[0] != tc.label {
			t.Fatalf("%s: hadWork=%v actions=%v", tc.op, hadWork, actions)
		}
		var req plantpb.FarmingRequest
		if err := proto.Unmarshal(sender.body("Farming"), &req); err != nil {
			t.Fatal(err)
		}
		if len(req.LandIds) != 2 || req.LandIds[0] != 1 || req.LandIds[1] != 2 {
			t.Fatalf("%s: unexpected Farming request: %+v", tc.op, req)
		}
	}
}

func TestRunFarmOperationRemoveOp(t *testing.T) {
	lands := mustMarshalAllLands(t, []*plantpb.LandInfo{
		{Id: 1, Unlocked: true, Plant: &plantpb.PlantInfo{
			Phases: []*plantpb.PlantPhaseInfo{{Phase: 7, BeginTime: 1}}, // dead
		}},
		{Id: 2, Unlocked: true, Plant: &plantpb.PlantInfo{
			Phases: []*plantpb.PlantPhaseInfo{{Phase: 3, BeginTime: 1}},
		}},
	})
	sender := &farmOpSender{lands: lands}
	api := &game.API{Sender: sender, GID: 42}

	hadWork, actions, _, err := RunFarmOperation(context.Background(), api, logic.DefaultAccountConfig(), "remove")
	if err != nil {
		t.Fatalf("RunFarmOperation remove: %v", err)
	}
	if !hadWork || len(actions) != 1 || actions[0] != "铲除1" {
		t.Fatalf("hadWork=%v actions=%v", hadWork, actions)
	}
	var req plantpb.RemovePlantRequest
	if err := proto.Unmarshal(sender.body("RemovePlant"), &req); err != nil {
		t.Fatal(err)
	}
	if len(req.LandIds) != 1 || req.LandIds[0] != 1 {
		t.Fatalf("unexpected RemovePlant request: %+v", req)
	}
}

func TestRunFarmOperationUnlockOp(t *testing.T) {
	lands := mustMarshalAllLands(t, []*plantpb.LandInfo{
		{Id: 4, Unlocked: false, CouldUnlock: true},
		{Id: 5, Unlocked: false, CouldUnlock: true},
	})
	sender := &farmOpSender{lands: lands}
	api := &game.API{Sender: sender, GID: 42}

	hadWork, actions, _, err := RunFarmOperation(context.Background(), api, logic.DefaultAccountConfig(), "unlock")
	if err != nil {
		t.Fatalf("RunFarmOperation unlock: %v", err)
	}
	// rust op_unlock：只解锁第一块可解锁土地并返回其 landId。
	if !hadWork || len(actions) != 1 || actions[0] != "解锁4" {
		t.Fatalf("hadWork=%v actions=%v", hadWork, actions)
	}
	var req plantpb.UnlockLandRequest
	if err := proto.Unmarshal(sender.body("UnlockLand"), &req); err != nil {
		t.Fatal(err)
	}
	if req.LandId != 4 || req.DoShared {
		t.Fatalf("unexpected UnlockLand request: %+v", req)
	}

	// 无可解锁土地 → 无动作。
	sender2 := &farmOpSender{lands: mustMarshalAllLands(t, []*plantpb.LandInfo{
		{Id: 1, Unlocked: true},
	})}
	api2 := &game.API{Sender: sender2, GID: 42}
	hadWork, _, _, err = RunFarmOperation(context.Background(), api2, logic.DefaultAccountConfig(), "unlock")
	if err != nil {
		t.Fatalf("unlock without target: %v", err)
	}
	if hadWork || sender2.called("UnlockLand") {
		t.Fatal("must not unlock without unlockable land")
	}
}

// TestRunFarmOperationFertilizeOp 对齐 rust op_fertilize：按化肥配置跑一轮并
// 返回 normal/organic 计数。
func TestRunFarmOperationFertilizeOp(t *testing.T) {
	lands := mustMarshalAllLands(t, []*plantpb.LandInfo{
		{Id: 1, Unlocked: true, Plant: &plantpb.PlantInfo{
			Phases: []*plantpb.PlantPhaseInfo{{Phase: 3, BeginTime: 1}},
		}},
	})
	sender := &farmOpSender{lands: lands}
	api := &game.API{Sender: sender, GID: 42}
	cfg := logic.DefaultAccountConfig()
	cfg.Automation.Fertilizer = logic.FertilizerNormal

	hadWork, actions, _, err := RunFarmOperation(context.Background(), api, cfg, "fertilize")
	if err != nil {
		t.Fatalf("RunFarmOperation fertilize: %v", err)
	}
	if !hadWork {
		t.Fatalf("expected work, actions=%v", actions)
	}
	if last := actions[len(actions)-1]; last != "施肥(普通肥1/有机肥0)" {
		t.Fatalf("expected fertilize count summary, actions=%v", actions)
	}
	if !sender.called("Fertilize") {
		t.Fatalf("expected Fertilize, methods=%v", sender.methods)
	}
	var req plantpb.FertilizeRequest
	if err := proto.Unmarshal(sender.body("Fertilize"), &req); err != nil {
		t.Fatal(err)
	}
	if len(req.LandIds) != 1 || req.LandIds[0] != 1 || req.FertilizerId != game.NormalFertilizerID {
		t.Fatalf("unexpected Fertilize request: %+v", req)
	}

	// 无种植地块 → 无动作。
	sender2 := &farmOpSender{lands: mustMarshalAllLands(t, []*plantpb.LandInfo{
		{Id: 1, Unlocked: true},
	})}
	api2 := &game.API{Sender: sender2, GID: 42}
	hadWork, _, _, err = RunFarmOperation(context.Background(), api2, cfg, "fertilize")
	if err != nil {
		t.Fatalf("fertilize without crops: %v", err)
	}
	if hadWork || sender2.called("Fertilize") {
		t.Fatal("must not fertilize without crops")
	}
}

// TestRunFarmOperationCycleAliasAll 对齐 rust operate："cycle" 是完整一轮的别名。
func TestRunFarmOperationCycleAliasAll(t *testing.T) {
	lands := mustMarshalAllLands(t, []*plantpb.LandInfo{
		{Id: 1, Unlocked: true, Plant: &plantpb.PlantInfo{
			Phases: []*plantpb.PlantPhaseInfo{
				{Phase: 1, BeginTime: 1},
				{Phase: 6, BeginTime: 2},
			},
		}},
	})
	sender := &farmOpSender{lands: lands}
	api := &game.API{Sender: sender, GID: 42}
	cfg := logic.DefaultAccountConfig()
	cfg.Automation.Fertilizer = "none"
	cfg.Automation.LandUpgrade = false
	cfg.Automation.Sell = false

	hadWork, actions, _, err := RunFarmOperation(context.Background(), api, cfg, "cycle")
	if err != nil {
		t.Fatalf("RunFarmOperation cycle: %v", err)
	}
	if !hadWork || !sender.called("Harvest") {
		t.Fatalf("cycle must run the full round, actions=%v", actions)
	}
}
