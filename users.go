package main

import (
	"errors"
	"log"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	jwt "github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

type UserAccount struct {
	Username     string
	PasswordHash []byte
	Email        []string
	LastLogin    int64
	Created      int64
}

var ErrUsernameTaken = errors.New("username taken")
var ErrUserDoesNotExist = errors.New("user does not exist")

func (a *UserAccount) Save() error {
	userBytes, err := encodeForDB(*a)

	if err != nil {
		log.Printf("Error encoding user account for DB: %v", err)
		return err
	}

	return DB.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte("user-"+a.Username), userBytes)
		err := txn.SetEntry(entry)
		log.Printf("Updated user record for %s", a.Username)
		return err
	})
}

func (a *UserAccount) GenerateToken() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": a.Username,
		"expires":  time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString([]byte(config.AuthTokenKey))
}

func NewUserAccount(username string, password string, email string) (*UserAccount, error) {
	existingUser, err := GetUserAccount(username)

	if err != nil {
		return nil, err
	}

	if existingUser != nil {
		return nil, ErrUsernameTaken
	}

	account := &UserAccount{
		Username:     username,
		PasswordHash: hashAndSalt(password),
		Created:      time.Now().Unix(),
	}
	err = account.Save()

	if err != nil {
		log.Printf("Error saving user account to DB for user %s", username)
		return nil, err
	}

	return account, nil
}

func Login(username string, password string) (*UserAccount, error) {
	user, err := GetUserAccount(username)

	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, nil
	}

	if checkPassword(password, user.PasswordHash) {
		return user, nil
	}

	return nil, nil
}

func GetUserAccount(username string) (*UserAccount, error) {
	var user *UserAccount

	err := GetDBItem("user-"+username, user)

	return user, err
}

func hashAndSalt(pass string) []byte {
	// Use GenerateFromPassword to hash & salt pwd.
	// MinCost is just an integer constant provided by the bcrypt
	// package along with DefaultCost & MaxCost.
	// The cost can be any value you want provided it isn't lower
	// than the MinCost (4)
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.MinCost)
	if err != nil {
		log.Println(err)
	}

	return hash
}

func checkPassword(pass string, hash []byte) bool {
	err := bcrypt.CompareHashAndPassword(hash, []byte(pass))

	if err != nil {
		log.Println(err)
		return false
	}

	return true
}
