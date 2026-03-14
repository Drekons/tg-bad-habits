package bot

import (
	"sync"
	
	"github.com/drek/tg-bad-habbits/internal/models"
)

// State represents the current FSM state of a user session.
type State int

const (
	StateIdle                 State = iota
	StateWaitStart                  // registered but waiting for "Start" tap
	StateWaitConfirmRelapse         // waiting for Yes/No on relapse confirmation
	StateHabitName                  // step 1: entering habit name
	StateHabitLastRelapse           // step 2: entering last relapse date
	StateHabitCost                  // step 3: entering cost per relapse
	StateHabitAvgCount              // step 4: entering avg relapses count
	StateHabitAvgPeriod             // step 5: choosing period (reply buttons)
	StateViewingHabitStats          // viewing stats for one habit; Back → habit list
	StateViewingStatsList           // viewing "choose habit for stats"; Back → main
)

// session stores per-user FSM data.
type session struct {
	State           State
	HabitDraft      habitDraft  // used during habit creation steps
	PendingHabitID  int64       // habit ID awaiting relapse confirmation
	MainMessageID   int         // message ID of the main screen (for edits)
}

// habitDraft accumulates user inputs during the habit creation flow.
type habitDraft struct {
	Name              string
	OriginAtRaw       string  // raw string before parsing
	OriginAt          interface{} // time.Time after parsing
	CostPerRelapse    float64
	AvgRelapsesCount  float64
	AvgRelapsesPeriod models.AvgPeriod
}

// StateManager is a thread-safe in-memory FSM store.
type StateManager struct {
	mu       sync.RWMutex
	sessions map[int64]*session
}

func NewStateManager() *StateManager {
	return &StateManager{
		sessions: make(map[int64]*session),
	}
}

func (sm *StateManager) get(userID int64) *session {
	sm.mu.RLock()
	s, ok := sm.sessions[userID]
	sm.mu.RUnlock()
	if !ok {
		s = &session{State: StateIdle}
		sm.mu.Lock()
		sm.sessions[userID] = s
		sm.mu.Unlock()
	}
	return s
}

func (sm *StateManager) GetState(userID int64) State {
	return sm.get(userID).State
}

func (sm *StateManager) SetState(userID int64, state State) {
	sm.get(userID).State = state
}

func (sm *StateManager) GetDraft(userID int64) *habitDraft {
	return &sm.get(userID).HabitDraft
}

func (sm *StateManager) ResetDraft(userID int64) {
	sm.get(userID).HabitDraft = habitDraft{}
}

func (sm *StateManager) SetPendingHabit(userID int64, habitID int64) {
	sm.get(userID).PendingHabitID = habitID
}

func (sm *StateManager) GetPendingHabit(userID int64) int64 {
	return sm.get(userID).PendingHabitID
}

func (sm *StateManager) SetMainMessageID(userID int64, msgID int) {
	sm.get(userID).MainMessageID = msgID
}

func (sm *StateManager) GetMainMessageID(userID int64) int {
	return sm.get(userID).MainMessageID
}

// ActiveMainUsers returns userIDs of users currently on the main screen with a tracked message.
func (sm *StateManager) ActiveMainUsers() []int64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	var ids []int64
	for id, s := range sm.sessions {
		if s.State == StateIdle && s.MainMessageID != 0 {
			ids = append(ids, id)
		}
	}
	return ids
}
