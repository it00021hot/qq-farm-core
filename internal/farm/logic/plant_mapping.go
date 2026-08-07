package logic

import "github.com/MQEnergy/go-skeleton/internal/farm/proto/plantpb"

// LandsFromPlantPB maps plantpb land snapshots into logic.LandInfo values.
func LandsFromPlantPB(lands []*plantpb.LandInfo) []LandInfo {
	out := make([]LandInfo, 0, len(lands))
	for _, l := range lands {
		if l == nil {
			continue
		}
		li := LandInfo{
			ID:           l.Id,
			Unlocked:     l.Unlocked,
			Level:        l.Level,
			MaxLevel:     l.MaxLevel,
			CouldUnlock:  l.CouldUnlock,
			CouldUpgrade: l.CouldUpgrade,
			MasterLandID: l.MasterLandId,
			SlaveLandIDs: append([]int64(nil), l.SlaveLandIds...),
			LandSize:     l.LandSize,
			LandsLevel:   l.LandsLevel,
		}
		if l.Plant != nil {
			p := l.Plant
			pi := PlantInfo{
				ID:              p.Id,
				Name:            p.Name,
				Season:          p.Season,
				DryNum:          p.DryNum,
				FruitID:         p.FruitId,
				FruitNum:        p.FruitNum,
				WeedOwners:      append([]int64(nil), p.WeedOwners...),
				InsectOwners:    append([]int64(nil), p.InsectOwners...),
				Stealers:        append([]byte(nil), p.Stealers...),
				Stealable:       p.Stealable,
				LeftFruitNum:    p.LeftFruitNum,
				MutantConfigIDs: append([]int64(nil), p.MutantConfigIds...),
			}
			// Bot only enforces left_inorc when the field is present. Proto3 omits zero on the
			// wire, so a zero value means "absent" → leave nil (eligible). Positive values set.
			if p.LeftInorcFertTimes > 0 {
				v := p.LeftInorcFertTimes
				pi.LeftInorcFertTimes = &v
			}
			if p.Field_36 != nil {
				pi.Activity = &PlantActivityInfo{
					ActivityID: p.Field_36.ActivityId,
					Param1:     p.Field_36.Param1,
					Param2:     p.Field_36.Param2,
					Date:       p.Field_36.Date,
				}
			}
			for _, ph := range p.Phases {
				if ph == nil {
					continue
				}
				for _, mu := range ph.Mutants {
					if mu != nil && mu.MutantConfigId > 0 {
						pi.MutantConfigIDs = append(pi.MutantConfigIDs, mu.MutantConfigId)
					}
				}
				pi.Phases = append(pi.Phases, PlantPhaseInfo{
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
