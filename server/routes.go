package server

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
)

func AddRoutes(r *gin.Engine, hmacSecret []byte, db Database) {
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	apiGroup := r.Group("/api")
	apiGroup.Use(UserTokenMiddleware(hmacSecret))

	api := API{DB: db}
	apiGroup.GET("/", api.rootEndpoint)
	apiGroup.POST("/createSession", api.createSessionEndpoint)
	apiGroup.GET("/get", api.getSessionEndpoint)
	apiGroup.GET("/get/:id", api.getSessionByIDEndpoint)
	apiGroup.GET("/ping/:sessionID", api.pingSessionEndpoint)

	editGroup := apiGroup.Group("/edit")
	editGroup.Use(OnlySessionOwner())
	editGroup.GET("/start/:sessionID", api.startSessionEndpoint)
	editGroup.GET("/pause/:sessionID", api.pauseSessionEndpoint)
	editGroup.GET("/resume/:sessionID", api.resumeSessionEndpoint)
	editGroup.DELETE("/delete")
}

type API struct {
	DB Database
}

type Message struct {
	Value string
}

// rootEndpoint returns context about the user, such as the userID of cookie
func (api *API) rootEndpoint(c *gin.Context) {
	userID, ok := c.Get("id")
	if !ok {
		c.JSON(400, ErrorResponse{
			Name:    "MissingUserID",
			Message: "id key is not in context",
		})
		return
	}

	c.JSON(200, Message{Value: fmt.Sprintf("%s", userID)})
}

type SessionSettings struct {
	FocusDuration        int64 `json:"focus_duration"` // all durations are in ms, because unix
	BreakDuration        int64 `json:"break_duration"`
	LongBreakDuration    int64 `json:"long_break_duration"`
	NumFocusPerLongBreak int   `json:"num_focus_per_long_break"`
}

func (api *API) createSessionEndpoint(c *gin.Context) {
	var body SessionSettings
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		log.Printf("error on decoding json request body: %v\n", err)
		c.JSON(400, ErrorResponse{
			Name:    "JSONDecoding",
			Message: "unable to decode response body, make sure it is valid json",
		})
		return
	}

	if body.FocusDuration == 0 || body.BreakDuration == 0 || body.LongBreakDuration == 0 || body.NumFocusPerLongBreak == 0 {
		c.JSON(400, ErrorResponse{
			Name:    "FieldIsZero",
			Message: "fields cannot be 0",
		})
		return
	}

	userID, ok := c.Get("id")
	if !ok {
		c.JSON(500, ErrorResponse{
			Name:    "MissingUserID",
			Message: "id key is not in context",
		})
		return
	}
	session := CreateSession(body.FocusDuration, body.BreakDuration, body.LongBreakDuration, body.NumFocusPerLongBreak, userID.(string))
	if err := api.DB.CreateSession(&session); err != nil {
		log.Printf("db error: %v\n", err)
		c.JSON(500, ErrorResponse{
			Name:    "DatabaseError",
			Message: "Internal database error",
		})
		return
	}

	c.JSON(200, Message{
		Value: fmt.Sprintf("created session for %s", userID),
	})
}

type SessionBody struct {
	Settings      SessionSettings `json:"settings"`
	ActivityState ActivityState   `json:"activity_state"`
	PomodoroState PomodoroState   `json:"pomodoro_state"`
	PomodoroTime  int64           `json:"pomodoro_time"` // time last pomodoro start started
}

var MissingUserIDResponse = ErrorResponse{
	Name:    "MissingUserID",
	Message: "id key is not in context",
}

func (api *API) getSessionEndpoint(c *gin.Context) {
	userID, ok := c.Get("id")
	if !ok {
		c.JSON(500, ErrorResponse{
			Name:    "MissingUserID",
			Message: "id key is not in context",
		})
		return
	}

	id := userID.(string)
	session, err := api.DB.GetSession(id)
	if err != nil {
		c.JSON(400, ErrorResponse{
			Name:    "DatabaseErr",
			Message: fmt.Sprintf("unable to find session for user %s: %v", id, err),
		})
		return
	}

	c.JSON(200, SessionBody{
		Settings: SessionSettings{
			FocusDuration:        session.FocusDuration,
			BreakDuration:        session.BreakDuration,
			LongBreakDuration:    session.LongBreakDuration,
			NumFocusPerLongBreak: session.NumFocusPerLongBreak,
		},
		ActivityState: session.ActivityState,
		PomodoroState: session.PomodoroState,
		PomodoroTime:  session.PomodoroTime,
	})
}

func (api *API) getSessionByIDEndpoint(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, ErrorResponse{
			Name:    "IDFieldMissing",
			Message: "id parameter in path can not be empty",
		})
		return
	}

	session, err := api.DB.GetSession(id)
	if err != nil {
		c.JSON(400, ErrorResponse{
			Name:    "DatabaseErr",
			Message: fmt.Sprintf("unable to find session for user %s: %v", id, err),
		})
		return
	}
	c.JSON(200, SessionBody{
		Settings: SessionSettings{
			FocusDuration:        session.FocusDuration,
			BreakDuration:        session.BreakDuration,
			LongBreakDuration:    session.LongBreakDuration,
			NumFocusPerLongBreak: session.NumFocusPerLongBreak,
		},
		ActivityState: session.ActivityState,
		PomodoroState: session.PomodoroState,
		PomodoroTime:  session.PomodoroTime,
	})
}

func (api *API) startSessionEndpoint(c *gin.Context) {
	userID, ok := c.Get("id")
	if !ok {
		c.JSON(400, MissingUserIDResponse)
		return
	}

	session, err := api.DB.GetSession(userID.(string))
	if err != nil {
		c.JSON(400, ErrorResponse{
			Name:    "DatabaseErr",
			Message: fmt.Sprintf("unable to find session for user %s: %v", userID.(string), err),
		})
		return
	}
	if err := session.StartSession(); err != nil {
		c.JSON(400, ErrorResponse{
			Name:    "StartSessionError",
			Message: fmt.Sprintf("unable to start session: %v", err),
		})
		return
	}

	c.JSON(200, formatSessionBody(session))
}

func formatSessionBody(session *Session) SessionBody {
	return SessionBody{
		Settings: SessionSettings{
			FocusDuration:        session.FocusDuration,
			BreakDuration:        session.BreakDuration,
			LongBreakDuration:    session.LongBreakDuration,
			NumFocusPerLongBreak: session.NumFocusPerLongBreak,
		},
		ActivityState: session.ActivityState,
		PomodoroState: session.PomodoroState,
		PomodoroTime:  session.PomodoroTime,
	}
}

func (api *API) pauseSessionEndpoint(c *gin.Context) {
	userID, ok := c.Get("id")
	if !ok {
		c.JSON(400, MissingUserIDResponse)
		return
	}

	session, err := api.DB.GetSession(userID.(string))
	if err != nil {
		c.JSON(400, ErrorResponse{
			Name:    "DatabaseErr",
			Message: fmt.Sprintf("unable to find session for user %s: %v", userID.(string), err),
		})
		return
	}
	if err := session.PauseSession(); err != nil {
		c.JSON(400, ErrorResponse{
			Name:    "PauseSessionError",
			Message: fmt.Sprintf("unable to pause session: %v", err),
		})
		return
	}

	c.JSON(200, formatSessionBody(session))
}

func (api *API) resumeSessionEndpoint(c *gin.Context) {
	userID, ok := c.Get("id")
	if !ok {
		c.JSON(400, MissingUserIDResponse)
		return
	}

	session, err := api.DB.GetSession(userID.(string))
	if err != nil {
		c.JSON(400, ErrorResponse{
			Name:    "DatabaseErr",
			Message: fmt.Sprintf("unable to find session for user %s: %v", userID.(string), err),
		})
		return
	}
	if err := session.PauseSession(); err != nil {
		c.JSON(400, ErrorResponse{
			Name:    "ResumeSessionError",
			Message: fmt.Sprintf("unable to resume session: %v", err),
		})
		return
	}

	c.JSON(200, formatSessionBody(session))
}

func (api *API) pingSessionEndpoint(c *gin.Context) {
	id := c.Param("sessionID")

	session, err := api.DB.GetSession(id)
	if err != nil {
		c.JSON(400, ErrorResponse{
			Name:    "DatabaseError",
			Message: fmt.Sprintf("unable to get session from database: %v", err),
		})
		return
	}

	if err := session.PingSession(); err != nil {
		c.JSON(400, ErrorResponse{
			Name:    "PingSessionError",
			Message: fmt.Sprintf("unable to ping session: %v", err),
		})
		return
	}

	if err := session.UpdateSessionPomodoroState(); err != nil {
		c.JSON(400, ErrorResponse{
			Name:    "UpdateSessionPomodoroStateError",
			Message: fmt.Sprintf("unable to update pomodoro state of session: %v", err),
		})
		return
	}

	c.JSON(200, formatSessionBody(session))
}

func (api *API) deleteSessionEndpoint(c *gin.Context) {
	userID, ok := c.Get("id")
	if !ok {
		c.JSON(400, MissingUserIDResponse)
		return
	}

	if err := api.DB.DeleteSession(userID.(string)); err != nil {
		c.JSON(400, ErrorResponse{
			Name:    "DBDeleteError",
			Message: fmt.Sprintf("unable to delete from database: %v", err),
		})
		return
	}
	c.JSON(200, Message{Value: "success"})
}
