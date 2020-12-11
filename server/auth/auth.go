package auth

import (
	"context"
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
	"github.com/zlogic/nanorss-go/data"
)

// DB provides functions to read and write items in the database.
type DB interface {
	GetOrCreateConfigVariable(varName string, generator func() (string, error)) (string, error)
	GetUser(username string) (*data.User, error)
}

// generateRandomKey generates a random byte slice with the desired length.
func generateRandomKey(length int) []byte {
	// Based on Gorilla securecookie.
	k := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}

// getOrCreateKey returns the existing cookie encryption key from db.
// If the key doesn't exist, it will generate a new key and save it in db.
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
	db            DB
	jwtAuth       *jwtauth.JWTAuth
	cookieExpires time.Duration
}

// AuthenticationCookie is the name of the authentication cookie.
const authenticationCookie = "nanorss"

// usernameClaim is the JWT token claim containing the username.
const usernameClaim = "username"

// NewCookieHandler creates a new instance of CookieHandler, using db to read or write the encryption keys.
func NewCookieHandler(db DB) (*CookieHandler, error) {
	signKey, err := getOrCreateKey(db, "cookie-sign-key", 128)
	if err != nil {
		return nil, fmt.Errorf("cannot get the hash key: %w", err)
	}
	handler := &CookieHandler{db: db}
	handler.jwtAuth = jwtauth.New("HS256", signKey, nil)
	handler.cookieExpires = 14 * 24 * time.Hour
	return handler, nil
}

// getUsernameToken returns the JWT token which can be saved into a cookie.
// This token can be used to verify and authorize the user.
func (handler *CookieHandler) getUsernameToken(username string) (string, error) {
	cookieExpires := time.Now().Add(handler.cookieExpires)
	claims := jwt.MapClaims{usernameClaim: username}
	jwtauth.SetExpiry(claims, cookieExpires)
	_, value, err := handler.jwtAuth.Encode(claims)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt cookie: %w", err)
	}
	return value, nil
}

// SetCookieUsername writes the username cookie in the HTTP response.
// If username is empty, deletes the username cookie.
// If rememberMe is false, the cookie will expire when the browser session ends.
func (handler *CookieHandler) SetCookieUsername(w http.ResponseWriter, username string, rememberMe bool) error {
	cookie := http.Cookie{
		Name:    authenticationCookie,
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
		MaxAge:  0,

		HttpOnly: true,
	}

	if username != "" {
		cookieExpires := time.Now()
		value, err := handler.getUsernameToken(username)
		if err != nil {
			return err
		}

		cookie.Value = value
		if rememberMe {
			cookie.Expires = cookieExpires
			cookie.MaxAge = int(handler.cookieExpires / time.Second)
		} else {
			cookie.Expires = time.Time{}
			cookie.MaxAge = 0
		}
	}

	http.SetCookie(w, &cookie)
	return nil
}

// getUsername attempts to decrypt the username from the cookie.
// If not possible to authenticate the user, returns an empty string.
func (handler *CookieHandler) getUsername(w http.ResponseWriter, r *http.Request) (string, error) {
	token, err := jwtauth.VerifyRequest(handler.jwtAuth, r, getAuthenticationCookie)
	if err == jwtauth.ErrExpired {
		// Cookie has expired - remove it from client.
		handler.SetCookieUsername(w, "", false)
		return "", nil
	} else if err == jwtauth.ErrNoTokenFound {
		// Client doesn't have an authentication cookie.
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}
	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("cannot map claims %v", token.Claims)
	}
	username, ok := mapClaims[usernameClaim]
	if !ok {
		return "", fmt.Errorf("username claim not found %v", mapClaims)
	}
	usernameString, ok := username.(string)
	if !ok {
		return "", fmt.Errorf("username %v is not a string", username)
	}
	return usernameString, nil
}

// getAuthenticationCookie returns the value of the authentication cookie in the request.
func getAuthenticationCookie(r *http.Request) string {
	cookie, err := r.Cookie(authenticationCookie)
	if err == http.ErrNoCookie {
		// Cookie not set.
		return ""
	} else if err != nil {
		return ""
	}
	return cookie.Value
}

// userContextKey is the key used to identify the User value in the context.
type userContextKey struct{}

// UserContextKey is the context key which can be used to look up the User object in the context.
var UserContextKey = &userContextKey{}

// AuthHandlerFunc is the authentication middleware.
// It will set the UserContextKey value in the context.
func (handler *CookieHandler) AuthHandlerFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, err := handler.getUsername(w, r)

		authLogger := log.WithField("requestID", middleware.GetReqID(r.Context())).
			WithField("remoteAddr", r.RemoteAddr)

		if err != nil {
			authLogger.WithError(err).Error("Authentication failed")
			next.ServeHTTP(w, r)
			return
		}
		if username == "" {
			next.ServeHTTP(w, r)
			return
		}

		user, err := handler.db.GetUser(username)
		if err != nil {
			authLogger.WithError(err).Error("Cannot get user from database")
			next.ServeHTTP(w, r)
			return
		}
		if user == nil {
			authLogger.Errorf("user %v not found in database", username)
			next.ServeHTTP(w, r)
			return
		}
		// Token is authenticated, pass it through.
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUser returns the user data from the request context,
// or nil if the request is not associated with an authenticated user.
func GetUser(ctx context.Context) *data.User {
	user, ok := ctx.Value(UserContextKey).(*data.User)
	if ok {
		return user
	}
	return nil
}

// HasAuthenticationCookie returns true if request has a non-empty authentication cookie.
func (handler *CookieHandler) HasAuthenticationCookie(r *http.Request) bool {
	return getAuthenticationCookie(r) != ""
}
