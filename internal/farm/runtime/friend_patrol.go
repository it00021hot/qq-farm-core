package runtime

import (
	"sort"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/friendpb"
)

const badFriendTopN = 20

// getPatrolBatchSize mirrors bot getPatrolBatchSize: ceil(n/4), at least 1 when n > 0.
func getPatrolBatchSize(friendCount int) int {
	if friendCount <= 0 {
		return 0
	}
	n := (friendCount + 3) / 4
	if n < 1 {
		return 1
	}
	return n
}

// selectUnvisitedPatrol picks up to budget unvisited GIDs from candidates.
// When all candidates are visited, clears visited and restarts (bot selectUnvisitedPatrol).
func selectUnvisitedPatrol(candidates []int64, budget int, visited map[int64]struct{}) []int64 {
	if len(candidates) == 0 || budget <= 0 {
		return nil
	}
	if visited == nil {
		visited = make(map[int64]struct{})
	}
	unmarked := filterUnvisited(candidates, visited)
	if len(unmarked) == 0 {
		for gid := range visited {
			delete(visited, gid)
		}
		unmarked = filterUnvisited(candidates, nil)
	}
	if len(unmarked) > budget {
		unmarked = unmarked[:budget]
	}
	return unmarked
}

func filterUnvisited(candidates []int64, visited map[int64]struct{}) []int64 {
	out := make([]int64, 0, len(candidates))
	for _, gid := range candidates {
		if gid <= 0 {
			continue
		}
		if visited != nil {
			if _, ok := visited[gid]; ok {
				continue
			}
		}
		out = append(out, gid)
	}
	return out
}

func markPatrolVisited(visited map[int64]struct{}, gid int64) {
	if visited == nil || gid <= 0 {
		return
	}
	visited[gid] = struct{}{}
}

type friendBubble struct {
	GID   int64
	Level int64
	Steal int64
	Help  int64
}

func collectEligibleFriends(friends []friendpb.GameFriend, myGID int64, blacklist map[int64]struct{}) []friendBubble {
	seen := make(map[int64]struct{}, len(friends))
	out := make([]friendBubble, 0, len(friends))
	for _, f := range friends {
		if f.GID <= 0 || f.GID == myGID || hasID(blacklist, f.GID) {
			continue
		}
		if _, ok := seen[f.GID]; ok {
			continue
		}
		seen[f.GID] = struct{}{}
		b := friendBubble{GID: f.GID, Level: f.Level}
		if f.Plant != nil {
			b.Steal = f.Plant.StealPlantNum
			b.Help = f.Plant.DryNum + f.Plant.WeedNum + f.Plant.InsectNum
		}
		out = append(out, b)
	}
	return out
}

// buildStealPatrolTargets: bubble friends first (steal desc), then ceil(n/4) unvisited zero-bubble probes.
func buildStealPatrolTargets(friends []friendpb.GameFriend, myGID int64, blacklist map[int64]struct{}, visited map[int64]struct{}) []int64 {
	eligible := collectEligibleFriends(friends, myGID, blacklist)
	var bubble, probe []friendBubble
	for _, f := range eligible {
		if f.Steal > 0 {
			bubble = append(bubble, f)
		} else {
			probe = append(probe, f)
		}
	}
	sort.Slice(bubble, func(i, j int) bool { return bubble[i].Steal > bubble[j].Steal })

	probeGIDs := make([]int64, len(probe))
	for i, f := range probe {
		probeGIDs[i] = f.GID
	}
	selected := selectUnvisitedPatrol(probeGIDs, getPatrolBatchSize(len(eligible)), visited)

	targets := make([]int64, 0, len(bubble)+len(selected))
	for _, f := range bubble {
		targets = append(targets, f.GID)
	}
	return append(targets, selected...)
}

// buildHelpPatrolTargets: help-bubble friends first (help desc), then ceil(n/4) unvisited zero-bubble probes.
func buildHelpPatrolTargets(friends []friendpb.GameFriend, myGID int64, blacklist map[int64]struct{}, visited map[int64]struct{}) []int64 {
	eligible := collectEligibleFriends(friends, myGID, blacklist)
	var bubble, probe []friendBubble
	for _, f := range eligible {
		if f.Help > 0 {
			bubble = append(bubble, f)
		} else {
			probe = append(probe, f)
		}
	}
	sort.Slice(bubble, func(i, j int) bool { return bubble[i].Help > bubble[j].Help })

	probeGIDs := make([]int64, len(probe))
	for i, f := range probe {
		probeGIDs[i] = f.GID
	}
	selected := selectUnvisitedPatrol(probeGIDs, getPatrolBatchSize(len(eligible)), visited)

	targets := make([]int64, 0, len(bubble)+len(selected))
	for _, f := range bubble {
		targets = append(targets, f.GID)
	}
	return append(targets, selected...)
}

// buildBadFriendTargets: zero steal/help bubbles, top N by level desc.
func buildBadFriendTargets(friends []friendpb.GameFriend, myGID int64, blacklist map[int64]struct{}, limit int) []int64 {
	if limit <= 0 {
		limit = badFriendTopN
	}
	eligible := collectEligibleFriends(friends, myGID, blacklist)
	var candidates []friendBubble
	for _, f := range eligible {
		if f.Steal == 0 && f.Help == 0 {
			candidates = append(candidates, f)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Level > candidates[j].Level })
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	out := make([]int64, len(candidates))
	for i, f := range candidates {
		out[i] = f.GID
	}
	return out
}
