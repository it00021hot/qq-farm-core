package logic

// 多格作物的未来布局预留。
//
// 1:1 翻译 qq-farm-bot/core/src/services/farm/layout-reservation.ts
// （bot 96fdb39，2026-09-11），与 rust
// crates/qq-farm-core/src/services/farm/layout_reservation.rs 保持同构。
//
// 用途：高优先级的多格背包种子暂时凑不齐完整布局时，选一组「未来布局」，
// 只预留其中当前已经空出的土地，避免这些空地被后续低优先种子占掉；
// 始终选锚点最小且已部分空出的布局，使后续轮次只会向更早布局收敛，不来回切换。

import "slices"

// FutureLayoutReservation 预留结果：目标布局 + 其中当前已空出、应被预留的土地。
type FutureLayoutReservation struct {
	Layout         PlantingLayout
	ReservedLandIDs []int64
}

// SelectFutureLayoutReservation 为暂时凑不齐的多格作物选择一组未来布局，
// 并只预留其中当前已空出的土地。
//
//   - currentEmptyLandIDs：本轮该种子可用的空地（已按土地类型过滤）
//   - allEligibleLandIDs：全部合格土地（未解锁的除外）
//   - plantSize：作物占地边长
//
// 第二个返回值为 false 表示无需/无法预留：单格作物、空地已能连成完整布局
// （直接种）、或没有任何「部分空出」的候选布局。
func SelectFutureLayoutReservation(currentEmptyLandIDs, allEligibleLandIDs []int64, plantSize int64) (*FutureLayoutReservation, bool) {
	size := max(plantSize, 1)
	if size <= 1 {
		return nil, false
	}

	emptySet := make(map[int64]struct{}, len(currentEmptyLandIDs))
	for _, id := range currentEmptyLandIDs {
		if id > 0 {
			emptySet[id] = struct{}{}
		}
	}
	if len(emptySet) == 0 {
		return nil, false
	}
	// 空地已能连成完整布局：直接种，无需预留
	emptySlice := make([]int64, 0, len(emptySet))
	for id := range emptySet {
		emptySlice = append(emptySlice, id)
	}
	if layouts := BuildPlantingLayouts(emptySlice, size); len(layouts) > 0 {
		return nil, false
	}

	eligible := make([]int64, 0, len(allEligibleLandIDs))
	for _, id := range allEligibleLandIDs {
		if id > 0 {
			eligible = append(eligible, id)
		}
	}
	var candidates []*FutureLayoutReservation
	for _, layout := range BuildPlantingLayouts(eligible, size) {
		reserved := make([]int64, 0, len(layout.LandIDs))
		for _, id := range layout.LandIDs {
			if _, ok := emptySet[id]; ok {
				reserved = append(reserved, id)
			}
		}
		// 只保留「部分空出」的布局：全空说明能直接种；全不空无法帮助收敛
		if len(reserved) == 0 || len(reserved) >= len(layout.LandIDs) {
			continue
		}
		candidates = append(candidates, &FutureLayoutReservation{Layout: layout, ReservedLandIDs: reserved})
	}
	if len(candidates) == 0 {
		return nil, false
	}
	// 锚点最小 => 收敛到最早的布局，避免布局来回切换
	slices.SortStableFunc(candidates, func(a, b *FutureLayoutReservation) int {
		switch {
		case a.Layout.AnchorLandID < b.Layout.AnchorLandID:
			return -1
		case a.Layout.AnchorLandID > b.Layout.AnchorLandID:
			return 1
		default:
			return 0
		}
	})
	return candidates[0], true
}
