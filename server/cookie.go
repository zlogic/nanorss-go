package server

import (
	"encoding/base64"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/pkg/errors"
	"github.com/zlogic/nanorss-go/data"
)

func getOrCreateKey(db *data.DBService, name string, length int) ([]byte, error) {
	hashKeyString, err := db.GetOrCreateConfigVariable(name, func() (string, error) {
		key := securecookie.GenerateRandomKey(length)
		return base64.StdEncoding.EncodeToString(key), nil
	})
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(hashKeyString)
}

type CookieHandler struct {
	secureCookie  *securecookie.SecureCookie
	cookieExpires time.Duration
}

const AuthorizationCookie = "nanorss"

func NewCookieHandler(db *data.DBService) (*CookieHandler, error) {
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

type UserCookie struct {
	Username   string
	Authorized time.Time
}

func (handler *CookieHandler) newCookie() *http.Cookie {
	return &http.Cookie{
		Name:    AuthorizationCookie,
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
		MaxAge:  0,

		HttpOnly: true,
	}
}

func (handler *CookieHandler) setCookieUsername(cookie *http.Cookie, username string) error {
	currentTime := time.Now()
	if username != "" {
		encryptCookie := UserCookie{
			Username:   username,
			Authorized: currentTime,
		}
		value, err := handler.secureCookie.Encode(AuthorizationCookie, &encryptCookie)
		if err != nil {
			return errors.Wrapf(err, "Failed to encrypt cookie %v", err)
		}
		cookie.Expires = currentTime.Add(handler.cookieExpires)
		cookie.MaxAge = int(handler.cookieExpires / time.Second)
		cookie.Value = value
	}
	return nil
}

func (handler *CookieHandler) GetUsername(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie(AuthorizationCookie)
	if err != nil {
		log.Printf("Failed to read cookie %v %v", cookie, err)
		return ""
	}
	value := UserCookie{}
	err = handler.secureCookie.Decode(AuthorizationCookie, cookie.Value, &value)
	if err != nil {
		log.Printf("Failed to decrypt cookie %v %v", cookie, err)
		return ""
	}
	if value.Authorized.Add(handler.cookieExpires).Before(time.Now()) {
		// Cookie is valid but has expired
		c := handler.newCookie()
		http.SetCookie(w, c)
		return ""
	}
	return value.Username
}
