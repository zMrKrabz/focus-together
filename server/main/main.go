package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/zMrKrabz/focus-together/server"
	"github.com/zMrKrabz/focus-together/server/mock"
	"log"
	"net/http"
)

var hmacSecret = []byte("helloworld")

func main() {
	db := &mock.MockDatabase{
		Sessions: map[string]*server.Session{},
	}
	r := gin.New()
	server.AddRoutes(r, hmacSecret, db)

	if err := r.Run(":8080"); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server closed")
		}

		log.Fatal(err)
	}
}
