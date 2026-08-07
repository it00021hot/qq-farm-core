package runtime

import (
	"sync"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/farm/proto/plantpb"
)

// friendHelpState mirrors bot friend/scheduler operation-limit + help-exp flags (WeChat path).
type friendHelpState struct {
	mu sync.Mutex

	limits       map[int64]opLimitEntry
	canGetHelp   bool
	autoDisabled bool
	lastResetDay string
}

type opLimitEntry struct {
	dayTimes         int64
	dayTimesLimit    int64
	dayExpTimes      int64
	dayExpTimesLimit int64
}

func newFriendHelpState() *friendHelpState {
	return &friendHelpState{
		limits:     make(map[int64]opLimitEntry),
		canGetHelp: true,
	}
}

func (s *friendHelpState) checkDailyReset() {
	today := time.Now().Format("2006-01-02")
	if s.lastResetDay == today {
		return
	}
	if s.lastResetDay != "" {
		s.limits = make(map[int64]opLimitEntry)
		s.canGetHelp = true
		s.autoDisabled = false
	}
	s.lastResetDay = today
}

func (s *friendHelpState) updateLimits(limits []*plantpb.OperationLimit) {
	if len(limits) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkDailyReset()
	for _, limit := range limits {
		if limit == nil || limit.Id <= 0 {
			continue
		}
		s.limits[limit.Id] = opLimitEntry{
			dayTimes:         limit.DayTimes,
			dayTimesLimit:    limit.DayTimesLt,
			dayExpTimes:      limit.DayExpTimes,
			dayExpTimesLimit: limit.DayExTimesLt,
		}
	}
}

func (s *friendHelpState) canGetExp(opID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkDailyReset()
	limit, ok := s.limits[opID]
	if !ok {
		// Bot: no limit info → conservative false until data arrives.
		return false
	}
	if limit.dayExpTimesLimit <= 0 {
		return true
	}
	return limit.dayExpTimes < limit.dayExpTimesLimit
}

func (s *friendHelpState) canGetExpByCandidates(opIDs []int64) bool {
	for _, id := range opIDs {
		if s.canGetExp(id) {
			return true
		}
	}
	return false
}

func (s *friendHelpState) getCanGetHelpExp() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkDailyReset()
	return s.canGetHelp
}

func (s *friendHelpState) setCanGetHelpExp(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkDailyReset()
	s.canGetHelp = v
}

func (s *friendHelpState) autoDisableHelpByExpLimit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkDailyReset()
	if !s.canGetHelp {
		return
	}
	s.canGetHelp = false
	s.autoDisabled = true
}
