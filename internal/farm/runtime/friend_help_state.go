package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/it00021hot/qq-farm-core/internal/farm/logic"
	"github.com/it00021hot/qq-farm-core/internal/farm/proto/plantpb"
)

const badDailyStateVersion = 1

// friendHelpState mirrors bot friend/scheduler operation-limit + help-exp flags.
type friendHelpState struct {
	mu sync.Mutex

	accountID uint64
	dataRoot  string

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

type badDailyStateFile struct {
	Version int    `json:"version"`
	Date    string `json:"date"`
	Stopped bool   `json:"stopped"`
}

func newFriendHelpState(accountID uint64, dataRoot string) *friendHelpState {
	s := &friendHelpState{
		accountID:  accountID,
		dataRoot:   dataRoot,
		limits:     make(map[int64]opLimitEntry),
		canGetHelp: true,
	}
	today := beijingDateKey()
	s.badOperationLimitReached = s.loadBadDailyStop(today)
	s.lastResetDay = today
	return s
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

func (s *friendHelpState) badDailyStatePath() string {
	root := s.dataRoot
	if root == "" {
		root = "data"
	}
	token := sha256.Sum256([]byte(fmt.Sprintf("%d", s.accountID)))
	return filepath.Join(root, fmt.Sprintf("friend-bad-state-%s.json", hex.EncodeToString(token[:])))
}

func (s *friendHelpState) loadBadDailyStop(today string) bool {
	path := s.badDailyStatePath()
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var state badDailyStateFile
	if json.Unmarshal(raw, &state) != nil {
		return false
	}
	return state.Version == badDailyStateVersion && state.Date == today && state.Stopped
}

func (s *friendHelpState) persistBadDailyStop(today string) {
	path := s.badDailyStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		slog.Warn("friend bad daily state mkdir failed", "path", path, "err", err)
		return
	}
	raw, err := json.Marshal(badDailyStateFile{
		Version: badDailyStateVersion,
		Date:    today,
		Stopped: true,
	})
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		slog.Warn("friend bad daily state write failed", "path", tmp, "err", err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		slog.Warn("friend bad daily state rename failed", "path", path, "err", err)
	}
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
	}
	s.badOperationLimitReached = s.loadBadDailyStop(today)
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
		if limit.Id == friendOpBadShared && limit.DayTimesLt > 0 && limit.DayTimes >= limit.DayTimesLt {
			s.markBadOperationLimitReachedLocked("operation_limit")
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
	return s.markBadOperationLimitReachedLocked("")
}

func (s *friendHelpState) markBadOperationLimitReachedLocked(method string) bool {
	if s.badOperationLimitReached {
		return false
	}
	s.badOperationLimitReached = true
	s.persistBadDailyStop(s.lastResetDay)
	if method != "" {
		slog.Info("今日放虫/放草次数已达上限，停止两类操作", "account", s.accountID, "method", method)
	} else {
		slog.Info("今日放虫/放草次数已达上限，停止两类操作", "account", s.accountID)
	}
	return true
}

// canOperate reports whether opID still has day times left (bot canOperate).
func (s *friendHelpState) canOperate(opID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkDailyReset()
	if (opID == friendOpBadShared || opID == friendOpPutBug) && s.badOperationLimitReached {
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
	if (opID == friendOpBadShared || opID == friendOpPutBug) && s.badOperationLimitReached {
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

// getRemainingBadOperationTimes returns shared put-weed/put-insect quota (10003).
func (s *friendHelpState) getRemainingBadOperationTimes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkDailyReset()
	if s.badOperationLimitReached {
		return 0
	}
	limit, ok := s.limits[friendOpBadShared]
	if !ok || limit.dayTimesLimit <= 0 {
		return 999
	}
	left := limit.dayTimesLimit - limit.dayTimes
	if left < 0 {
		return 0
	}
	return int(left)
}
