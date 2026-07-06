package service

import (
	"sync"
	"time"
)

const defaultSchedulingLogCapacity = 200

// SchedulingLogEvent 是调度面板展示的最近调度事件。
// 仅记录排障必要元数据，不包含请求正文、密钥或 token。
type SchedulingLogEvent struct {
	CreatedAt          time.Time         `json:"created_at"`
	Platform           string            `json:"platform,omitempty"`
	Model              string            `json:"model,omitempty"`
	GroupID            int64             `json:"group_id,omitempty"`
	CandidateCount     int               `json:"candidate_count"`
	AvailableCount     int               `json:"available_count"`
	AccountID          int64             `json:"account_id,omitempty"`
	AccountName        string            `json:"account_name,omitempty"`
	PreferredAccountID int64             `json:"preferred_account_id,omitempty"`
	PreferredHit       bool              `json:"preferred_hit"`
	StickyStatus              string         `json:"sticky_status,omitempty"`
	CredentialStrategy        string         `json:"credential_strategy,omitempty"`
	CredentialFallbackEnabled bool           `json:"credential_fallback_enabled"`
	SelectedCredentialType    string         `json:"selected_credential_type,omitempty"`
	Reason                    string         `json:"reason"`
	FilterSummary      map[string]int    `json:"filter_summary,omitempty"`
	RequestID          string            `json:"request_id,omitempty"`
	ClientRequestID    string            `json:"client_request_id,omitempty"`
}

// SchedulingLogService 保存最近调度事件的内存环形日志。
type SchedulingLogService struct {
	mu       sync.Mutex
	capacity int
	events   []SchedulingLogEvent
}

var DefaultSchedulingLogService = NewSchedulingLogService(defaultSchedulingLogCapacity)

func NewSchedulingLogService(capacity int) *SchedulingLogService {
	if capacity <= 0 {
		capacity = defaultSchedulingLogCapacity
	}
	return &SchedulingLogService{capacity: capacity, events: make([]SchedulingLogEvent, 0, capacity)}
}

func (s *SchedulingLogService) Record(event SchedulingLogEvent) {
	if s == nil {
		return
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.events) >= s.capacity {
		copy(s.events, s.events[1:])
		s.events[len(s.events)-1] = event
		return
	}
	s.events = append(s.events, event)
}

func (s *SchedulingLogService) List(limit int) []SchedulingLogEvent {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit > len(s.events) {
		limit = len(s.events)
	}
	result := make([]SchedulingLogEvent, 0, limit)
	for i := len(s.events) - 1; i >= 0 && len(result) < limit; i-- {
		item := s.events[i]
		if item.FilterSummary != nil {
			copyMap := make(map[string]int, len(item.FilterSummary))
			for key, value := range item.FilterSummary {
				copyMap[key] = value
			}
			item.FilterSummary = copyMap
		}
		result = append(result, item)
	}
	return result
}
