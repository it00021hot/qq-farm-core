package plantpb

import "github.com/MQEnergy/go-skeleton/internal/farm/logic"

// LandsToLogic maps plantpb land snapshots into logic.LandInfo values.
func LandsToLogic(lands []*LandInfo) []logic.LandInfo {
	out := make([]logic.LandInfo, 0, len(lands))
	for _, l := range lands {
		if l == nil {
			continue
		}
		li := logic.LandInfo{
			ID:           l.ID,
			Unlocked:     l.Unlocked,
			Level:        l.Level,
			MaxLevel:     l.MaxLevel,
			CouldUnlock:  l.CouldUnlock,
			CouldUpgrade: l.CouldUpgrade,
			MasterLandID: l.MasterLandID,
			SlaveLandIDs: append([]int64(nil), l.SlaveLandIDs...),
			LandSize:     l.LandSize,
			LandsLevel:   l.LandsLevel,
		}
		if l.Plant != nil {
			p := l.Plant
			pi := logic.PlantInfo{
				ID:              p.ID,
				Name:            p.Name,
				Season:          p.Season,
				DryNum:          p.DryNum,
				FruitID:         p.FruitID,
				FruitNum:        p.FruitNum,
				WeedOwners:      append([]int64(nil), p.WeedOwners...),
				InsectOwners:    append([]int64(nil), p.InsectOwners...),
				Stealers:        append([]byte(nil), p.Stealers...),
				Stealable:       p.Stealable,
				LeftFruitNum:    p.LeftFruitNum,
				MutantConfigIDs: append([]int64(nil), p.MutantConfigIDs...),
			}
			if p.LeftInorcFertTimes != nil {
				v := *p.LeftInorcFertTimes
				pi.LeftInorcFertTimes = &v
			}
			if p.Activity != nil {
				pi.Activity = &logic.PlantActivityInfo{
					ActivityID: p.Activity.ActivityID,
					Param1:     p.Activity.Param1,
					Param2:     p.Activity.Param2,
					Date:       p.Activity.Date,
				}
			}
			for _, ph := range p.Phases {
				for _, mu := range ph.Mutants {
					if mu.MutantConfigID > 0 {
						pi.MutantConfigIDs = append(pi.MutantConfigIDs, mu.MutantConfigID)
					}
				}
				pi.Phases = append(pi.Phases, logic.PlantPhaseInfo{
					Phase:      int(ph.Phase),
					BeginTime:  ph.BeginTime,
					DryTime:    ph.DryTime,
					WeedsTime:  ph.WeedsTime,
					InsectTime: ph.InsectTime,
				})
			}
			pi.MutantConfigIDs = uniquePositiveIDs(pi.MutantConfigIDs)
			li.Plant = &pi
		}
		out = append(out, li)
	}
	return out
}

func uniquePositiveIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
