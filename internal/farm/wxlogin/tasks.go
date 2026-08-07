package wxlogin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

const (
	TaskTTL = 110 * time.Second

	StatusWaiting      = "waiting"
	StatusScanned      = "scanned"
	StatusAuthorized   = "authorized"
	StatusReadyForCode = "ready_for_code"
	StatusCancelled    = "cancelled"
	StatusExpired      = "expired"
	StatusFailed       = "failed"
)

// Task is an in-memory WeChat QR login task owned by one admin user.
type Task struct {
	ID        string
	Owner     string
	CreatedAt time.Time
	Status    string
	Session   *Session
	QR        []byte
	Code      string

	mu      sync.Mutex
	pending *sync.Mutex // serializes poll/confirm/code
}

// TaskStore keeps active login tasks.
type TaskStore struct {
	mu    sync.Mutex
	tasks map[string]*Task
	svc   *WxLoginService
}

func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks: make(map[string]*Task),
		svc:   NewWxLoginService(),
	}
}

func (s *TaskStore) Create(owner string) (*Task, error) {
	session, qr, err := s.svc.CreateQrSession(context.Background())
	if err != nil {
		return nil, err
	}
	id, err := randomTaskID()
	if err != nil {
		return nil, err
	}
	task := &Task{
		ID:        id,
		Owner:     owner,
		CreatedAt: time.Now(),
		Status:    StatusWaiting,
		Session:   session,
		QR:        qr,
		pending:   &sync.Mutex{},
	}
	s.mu.Lock()
	s.tasks[id] = task
	s.mu.Unlock()
	return task, nil
}

func (s *TaskStore) Find(owner, taskID string) (*Task, error) {
	s.mu.Lock()
	task := s.tasks[taskID]
	s.mu.Unlock()
	if task == nil || task.Owner != owner {
		return nil, fmt.Errorf("Login task not found or expired")
	}
	if time.Since(task.CreatedAt) > TaskTTL {
		s.Destroy(task)
		return nil, fmt.Errorf("Login task not found or expired")
	}
	return task, nil
}

func (s *TaskStore) Destroy(task *Task) {
	if task == nil {
		return
	}
	s.svc.Destroy(task.Session)
	task.Code = ""
	s.mu.Lock()
	delete(s.tasks, task.ID)
	s.mu.Unlock()
}

func (s *TaskStore) Poll(task *Task) error {
	task.pending.Lock()
	defer task.pending.Unlock()
	if task.Status == StatusAuthorized || task.Status == StatusReadyForCode {
		return nil
	}
	st, err := s.svc.Poll(context.Background(), task.Session)
	if err != nil {
		return err
	}
	task.Status = string(st)
	return nil
}

func (s *TaskStore) Confirm(task *Task) error {
	task.pending.Lock()
	defer task.pending.Unlock()
	if task.Status != StatusAuthorized {
		return fmt.Errorf("Waiting for scan authorization")
	}
	if _, _, err := s.svc.Confirm(context.Background(), task.Session); err != nil {
		return err
	}
	task.Status = StatusReadyForCode
	return nil
}

func (s *TaskStore) IssueCode(task *Task) (string, error) {
	task.pending.Lock()
	defer task.pending.Unlock()
	if task.Status != StatusReadyForCode {
		return "", fmt.Errorf("Login code is not ready")
	}
	code, err := s.svc.IssueCode(context.Background(), task.Session, TargetMiniProgramID)
	if err != nil {
		return "", err
	}
	task.Code = code
	return code, nil
}

// PublicView is the API-facing task summary.
func (t *Task) PublicView() map[string]any {
	return map[string]any{
		"task_id":    t.ID,
		"app_id":     TargetMiniProgramID,
		"status":     t.Status,
		"expires_at": t.CreatedAt.Add(TaskTTL).Unix(),
	}
}

func randomTaskID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
