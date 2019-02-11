package data

import (
	"encoding/json"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Password    string
	Opml        string
	Pagemonitor string
}

type UserService struct {
	db *badger.DB
}

func newUserService(db *badger.DB) *UserService {
	return &UserService{
		db: db,
	}
}

func (s *UserService) Get(username string) (*User, error) {
	var user *User
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(CreateKey(&User{}, username))
		if err == badger.ErrKeyNotFound {
			return nil
		}

		value, err := item.Value()
		if err != nil {
			return err
		}
		user, err = decodeValue(value)
		return err
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read User %v", username)
	}
	return user, nil
}

func (s *UserService) Save(user *User) (err error) {
	value, err := user.encode()
	if err != nil {
		return err
	}
	err = s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(CreateKey(user, "default")), value)
	})
	return err
}

func (user *User) SetPassword(newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hash)
	return nil
}

func (user *User) ValidatePassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
}

func (user *User) encode() (encoded []byte, err error) {
	encoded, err = json.Marshal(user)
	if err != nil {
		err = errors.Wrap(err, "Cannot encode User")
	}
	return
}

func decodeValue(data []byte) (user *User, err error) {
	user = &User{}
	err = json.Unmarshal(data, user)
	if err != nil {
		err = errors.Wrap(err, "Cannot decode User")
		return nil, err
	}
	return
}
