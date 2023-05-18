package mock

import (
	"fmt"
	"github.com/zMrKrabz/focus-together/server"
)

type MockDatabase struct {
	Sessions map[string]*server.Session
}

func (db *MockDatabase) CreateSession(session *server.Session) error {
	db.Sessions[session.Owner] = session
	return nil
}

func (db *MockDatabase) GetSession(id string) (*server.Session, error) {
	session, ok := db.Sessions[id]
	if !ok {
		return nil, fmt.Errorf("unable to find session with id: %s", id)
	}

	return session, nil
}

func (db *MockDatabase) DeleteSession(id string) error {
	delete(db.Sessions, id)
	return nil
}
