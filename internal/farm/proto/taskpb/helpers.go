package taskpb

// ClaimablePointIDs returns active reward point IDs that are ready to claim (status=2).
func (x *Active) ClaimablePointIDs() []int64 {
	if x == nil {
		return nil
	}
	var out []int64
	for _, point := range x.Rewards {
		if point != nil && point.Status == 2 {
			out = append(out, point.PointId)
		}
	}
	return out
}
