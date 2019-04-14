package server

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func getOrCreateKey(db DB, name string, length int) ([]byte, error) {
	hashKeyString, err := db.GetOrCreateConfigVariable(name, func() (string, error) {
		key := securecookie.GenerateRandomKey(length)
		return base64.StdEncoding.EncodeToString(key), nil
	})
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(hashKeyString)
}

// CookieHandler sets and validates secure authentication cookies.
type CookieHandler struct {
	secureCookie  *securecookie.SecureCookie
	cookieExpires time.Duration
}

// AuthenticationCookie is the name of the authentication cookie.
const AuthenticationCookie = "nanorss"

// NewCookieHandler creates a new instance of CookieHandler, using db to read or write the encryption keys.
func NewCookieHandler(db DB) (*CookieHandler, error) {
	hashKey, err := getOrCreateKey(db, "cookie-hash-key", 64)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot get the hash key")
	}
	blockKey, err := getOrCreateKey(db, "cookie-block-key", 32)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot get the block key")
	}
	handler := &CookieHandler{}
	handler.secureCookie = securecookie.New(hashKey, blockKey)
	handler.cookieExpires = 14 * 24 * time.Hour
	return handler, nil
}

// UserCookie contains data stored in the secure cookie.
type UserCookie struct {
	Username   string
	Authorized time.Time
}

// NewCookie creates a new cookie.
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
		encryptCookie := UserCookie{
			Username:   username,
			Authorized: currentTime,
		}
		value, err := handler.secureCookie.Encode(AuthenticationCookie, &encryptCookie)
		if err != nil {
			return errors.Wrapf(err, "Failed to encrypt cookie %v", err)
		}
		cookie.Expires = currentTime.Add(handler.cookieExpires)
		cookie.MaxAge = int(handler.cookieExpires / time.Second)
		cookie.Value = value
	}
	return nil
}

// GetUsername attempts to decrypt the username from the cookie.
// If not possible to authenticate the user, returns an empty string.
func (handler *CookieHandler) GetUsername(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie(AuthenticationCookie)
	if err == http.ErrNoCookie {
		// Cookie not set
		return ""
	} else if err != nil {
		log.WithField("cookie", AuthenticationCookie).WithError(err).Error("Failed to read cookie")
		return ""
	}
	value := UserCookie{}
	err = handler.secureCookie.Decode(AuthenticationCookie, cookie.Value, &value)
	if err != nil {
		log.WithField("cookie", cookie).WithError(err).Error("Failed to decrypt cookie")
		return ""
	}
	if value.Authorized.Add(handler.cookieExpires).Before(time.Now()) {
		// Cookie is valid but has expired
		c := handler.NewCookie()
		http.SetCookie(w, c)
		return ""
	}
	return value.Username
}
