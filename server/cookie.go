package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/jwtauth"
	log "github.com/sirupsen/logrus"
)

func generateRandomKey(length int) []byte {
	// Based on Gorilla securecookie
	k := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}

func getOrCreateKey(db DB, name string, length int) ([]byte, error) {
	hashKeyString, err := db.GetOrCreateConfigVariable(name, func() (string, error) {
		key := generateRandomKey(length)
		return base64.StdEncoding.EncodeToString(key), nil
	})
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(hashKeyString)
}

// CookieHandler sets and validates secure authentication cookies.
type CookieHandler struct {
	jwtAuth       *jwtauth.JWTAuth
	cookieExpires time.Duration
}

// AuthenticationCookie is the name of the authentication cookie.
const AuthenticationCookie = "nanorss"

// usernameClaim is the JWT token claim containing the username.
const usernameClaim = "username"

// NewCookieHandler creates a new instance of CookieHandler, using db to read or write the encryption keys.
func NewCookieHandler(db DB) (*CookieHandler, error) {
	signKey, err := getOrCreateKey(db, "cookie-sign-key", 128)
	if err != nil {
		return nil, fmt.Errorf("Cannot get the hash key because of %w", err)
	}
	handler := &CookieHandler{}
	handler.jwtAuth = jwtauth.New("HS256", signKey, nil)
	handler.cookieExpires = 14 * 24 * time.Hour
	return handler, nil
}

// NewCookie creates a new authentication cookie.
func (handler *CookieHandler) NewCookie() *http.Cookie {
	return &http.Cookie{
		Name:    AuthenticationCookie,
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
		MaxAge:  0,

		HttpOnly: true,
	}
}

// SetCookieUsername encrypts and sets the cookie to contain the username.
func (handler *CookieHandler) SetCookieUsername(cookie *http.Cookie, username string) error {
	currentTime := time.Now()
	if username != "" {
		cookieExpires := currentTime.Add(handler.cookieExpires)
		claims := jwt.MapClaims{usernameClaim: username}
		jwtauth.SetExpiry(claims, cookieExpires)
		_, value, err := handler.jwtAuth.Encode(claims)
		if err != nil {
			return fmt.Errorf("Failed to encrypt cookie because of %w", err)
		}
		cookie.Expires = cookieExpires
		cookie.MaxAge = int(handler.cookieExpires / time.Second)
		cookie.Value = value
	}
	return nil
}

// GetUsername attempts to decrypt the username from the cookie.
// If not possible to authenticate the user, returns an empty string.
func (handler *CookieHandler) GetUsername(w http.ResponseWriter, r *http.Request) string {
	token, err := jwtauth.VerifyRequest(handler.jwtAuth, r, getAuthenticationCookie)
	if err == jwtauth.ErrExpired {
		// Cookie has expired - remove it from client.
		c := handler.NewCookie()
		http.SetCookie(w, c)
		return ""
	}
	if err != nil {
		log.WithField("requestID", middleware.GetReqID(r.Context())).
			WithField("remoteAddr", r.RemoteAddr).
			WithError(err).Error("Authentication failed")
		return ""
	}
	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.WithField("claims", token.Claims).Error("Cannot map claims")
		return ""
	}
	username, ok := mapClaims[usernameClaim]
	if !ok {
		log.WithField("claims", mapClaims).Error("Username claim not found")
		return ""
	}
	usernameString, ok := username.(string)
	if !ok {
		log.WithField("claims", mapClaims).Error("Username is not a string")
		return ""
	}
	return usernameString
}

func getAuthenticationCookie(r *http.Request) string {
	cookie, err := r.Cookie(AuthenticationCookie)
	if err == http.ErrNoCookie {
		// Cookie not set
		return ""
	} else if err != nil {
		log.WithField("cookie", AuthenticationCookie).WithError(err).Error("Failed to read cookie")
		return ""
	}
	return cookie.Value
}
