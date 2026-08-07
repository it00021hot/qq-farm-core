package runtime

import (
	"fmt"
	"sync"
	"time"

	"github.com/MQEnergy/go-skeleton/internal/farm/logic"
	"github.com/MQEnergy/go-skeleton/internal/farm/proto/plantpb"
)

// friendHelpState mirrors bot friend/scheduler operation-limit + help-exp flags.
type friendHelpState struct {
	mu sync.Mutex

	limits                   map[int64]opLimitEntry
	canGetHelp               bool
	autoDisabled             bool
	badOperationLimitReached bool
	lastResetDay             string
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

// beijingDateKey returns YYYY-MM-DD in UTC+8 using synced server time when available.
func beijingDateKey() string {
	nowSec := logic.GetServerTimeSec()
	var t time.Time
	if nowSec > 0 {
		t = time.Unix(nowSec, 0).UTC()
	} else {
		t = time.Now().UTC()
	}
	bj := t.Add(8 * time.Hour)
	return fmt.Sprintf("%04d-%02d-%02d", bj.Year(), int(bj.Month()), bj.Day())
}

func (s *friendHelpState) checkDailyReset() {
	today := beijingDateKey()
	if s.lastResetDay == today {
		return
	}
	if s.lastResetDay != "" {
		s.limits = make(map[int64]opLimitEntry)
		s.canGetHelp = true
		s.autoDisabled = false
		s.badOperationLimitReached = false
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

func (s *friendHelpState) isBadOperationLimitReached() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkDailyReset()
	return s.badOperationLimitReached
}

func (s *friendHelpState) markBadOperationLimitReached() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkDailyReset()
	if s.badOperationLimitReached {
		return false
	}
	s.badOperationLimitReached = true
	return true
}

// canOperate reports whether opID still has day times left (bot canOperate).
func (s *friendHelpState) canOperate(opID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkDailyReset()
	if (opID == friendOpPutWeed || opID == friendOpPutBug) && s.badOperationLimitReached {
		return false
	}
	limit, ok := s.limits[opID]
	if !ok {
		return true
	}
	if limit.dayTimesLimit <= 0 {
		return true
	}
	return limit.dayTimes < limit.dayTimesLimit
}

// getRemainingTimes mirrors bot getRemainingTimes (999 when unknown / unlimited).
func (s *friendHelpState) getRemainingTimes(opID int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkDailyReset()
	if (opID == friendOpPutWeed || opID == friendOpPutBug) && s.badOperationLimitReached {
		return 0
	}
	limit, ok := s.limits[opID]
	if !ok || limit.dayTimesLimit <= 0 {
		return 999
	}
	left := limit.dayTimesLimit - limit.dayTimes
	if left < 0 {
		return 0
	}
	return int(left)
}
