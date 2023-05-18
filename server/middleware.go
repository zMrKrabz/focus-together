package server

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"net/http"
)

type ErrorResponse struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

func UserTokenMiddleware(hmacSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := GetUserID(c, hmacSecret)
		if err != nil {
			c.JSON(400, ErrorResponse{
				Name:    "UserIDError",
				Message: fmt.Sprintf("unable to get user id: %v", err),
			})
			c.Abort()
			return
		}
		c.Set("id", userID)

		c.Next()
	}
}

func GetUserID(c *gin.Context, key []byte) (string, error) {
	cookie, err := c.Request.Cookie("user")
	if errors.Is(err, http.ErrNoCookie) {
		// generate cookie with jwt info
		token := jwt.New(jwt.SigningMethodHS256)
		claims := token.Claims.(jwt.MapClaims)
		id := uuid.New().String()
		claims["id"] = id

		tokenString, err := token.SignedString(key)
		if err != nil {
			return "", err
		}
		// TODO: this might be an issue with maxAge
		c.SetCookie("user", tokenString, -1, "/", "", true, true)
		return id, nil
	}

	if err != nil {
		return "", fmt.Errorf("unable to get \"user\" cookie: %v", err)
	}

	// decode jwt token and add info
	jwtString := cookie.Value
	token, err := jwt.Parse(jwtString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("signing method of jwt is not hmac")
		}

		return key, nil
	})

	if err != nil {
		return "", fmt.Errorf("error parsing jwt token: %v", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("jwt token is not valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("unable to map token claims")
	}

	return claims["id"].(string), nil
}

const sessionIDPath = "sessionID"

// OnlySessionOwner is a middleware where it checks the session that is being edited and the current user. If the user
// is not the owner, the middleware will return a 404 error. All paths must have :sessionID in the path
func OnlySessionOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := c.Get("id")
		if !ok {
			c.JSON(400, ErrorResponse{
				Name:    "MissingUserID",
				Message: "unable to identify user",
			})
			c.Abort()
			return
		}

		sessionID := c.Param("sessionID")
		if userID != sessionID {
			c.JSON(404, ErrorResponse{
				Name:    "UnauthorizedUser",
				Message: fmt.Sprintf("you are %v but are trying to edit %v", userID, sessionID),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
