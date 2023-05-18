package server

import (
	"fmt"
)

type ActivityState string

const (
	NOT_STARTED ActivityState = "NOT_STARTED"
	IN_PROGRESS ActivityState = "IN_PROGRESS"
	PAUSED      ActivityState = "PAUSED"
	STOPPED     ActivityState = "STOPPED"
)

type PomodoroState string

const (
	FOCUS      PomodoroState = "FOCUS"
	BREAK      PomodoroState = "BREAK"
	LONG_BREAK PomodoroState = "LONG_BREAK"
)

func CreateSession(focusDuration, breakDuration, longBreakDuration int64, numFocusPerLongBreak int, owner string) Session {
	return Session{
		FocusDuration:        focusDuration,
		BreakDuration:        breakDuration,
		LongBreakDuration:    longBreakDuration,
		Owner:                owner,
		LastPing:             Now(),
		Participants:         map[string]int64{},
		ActivityState:        NOT_STARTED,
		PomodoroState:        FOCUS,
		Counter:              0,
		NumFocusPerLongBreak: numFocusPerLongBreak,
	}
}

type Session struct {
	FocusDuration        int64 // in minutes
	BreakDuration        int64
	LongBreakDuration    int64
	Counter              int
	NumFocusPerLongBreak int
	Owner                string           // uuid, is also uuid of session
	LastPing             int64            // last time owner was present in session
	Participants         map[string]int64 // present uuids, with their last ping date
	ActivityState        ActivityState
	PomodoroState        PomodoroState
	PomodoroTime         int64 // time last pomodoro start started
	PauseDelta           int64 // difference between paused and when session was started
}

func (s *Session) StartSession() error {
	if s.PomodoroTime > 0 {
		return fmt.Errorf("session already started")
	}

	s.ActivityState = IN_PROGRESS
	s.PomodoroState = FOCUS
	s.PomodoroTime = Now()
	return nil
}

func (s *Session) PauseSession() error {
	if s.ActivityState == PAUSED {
		return fmt.Errorf("activity already paused")
	}
	s.ActivityState = PAUSED
	s.PauseDelta = Now() - s.PomodoroTime
	return nil
}

func (s *Session) ResumeSession() error {
	if s.ActivityState == IN_PROGRESS {
		return fmt.Errorf("session already in progress")
	}
	s.ActivityState = IN_PROGRESS
	s.PomodoroTime = Now() - s.PauseDelta
	return nil
}

func (s *Session) UpdateSessionPomodoroState() error {
	if s.PomodoroState == FOCUS {
		if Now() < s.PomodoroTime+s.FocusDuration {
			return nil
		}

		s.Counter++
		if s.Counter%s.NumFocusPerLongBreak == 0 {
			s.PomodoroState = LONG_BREAK
		} else {
			s.PomodoroState = BREAK
		}
	} else if s.PomodoroState == BREAK {
		if Now() < s.PomodoroTime+s.BreakDuration {
			return nil
		}

		s.PomodoroState = FOCUS
	} else {
		if Now() < s.PomodoroTime+s.LongBreakDuration {
			return nil
		}

		s.PomodoroState = FOCUS
	}
	s.PomodoroTime = Now()
	return nil
}

// StopSessionInactivity stops the session if it is over 5 minutes since the session was last pinged
func (s *Session) StopSessionInactivity() (bool, error) {
	if Now() < s.LastPing+300000 {
		return false, nil
	}
	return true, nil
}

func (s *Session) StopSession() error {
	if s.ActivityState == STOPPED {
		return fmt.Errorf("activity already stopped")
	}
	s.ActivityState = STOPPED
	return nil
}

func (s *Session) PingSession() error {
	s.LastPing = Now()
	return nil
}

func (s *Session) PingParticipant(participant string) error {
	if _, ok := s.Participants[participant]; !ok {
		return fmt.Errorf("participant not in session")
	}
	s.Participants[participant] = Now()
	return nil
}

// DeleteParticipantInactivity only deletes participant only if it has been over 5 minutes since the participant has joined
func (s *Session) DeleteParticipantInactivity(participant string) (bool, error) {
	lastPing, exists := s.Participants[participant]
	if !exists {
		return false, fmt.Errorf("participant not in session")
	}

	if Now() < lastPing+300000 {
		return false, nil
	}
	return s.DeleteParticipant(participant)
}

func (s *Session) DeleteParticipant(participant string) (bool, error) {
	if _, exists := s.Participants[participant]; !exists {
		return false, fmt.Errorf("participant not in session")
	}

	delete(s.Participants, participant)
	return true, nil
}

func (s *Session) JoinSession(participant string) error {
	if _, ok := s.Participants[participant]; ok {
		return fmt.Errorf("participant already in session")
	}

	s.Participants[participant] = Now()
	return nil
}
