package server

type Database interface {
	CreateSession(*Session) error
	GetSession(string) (*Session, error)
	DeleteSession(string) error
	// PingSession(string, int64) error
	// AddSessionParticipant(string, string) error
}
