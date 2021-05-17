package main

import (
	"errors"
	"fmt"
	"log"
	"reflect"

	jwt "github.com/dgrijalva/jwt-go"
)

func GetUserByToken(tokenString string) (*UserAccount, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {

		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected JWT signing method: %v", reflect.TypeOf(token.Method))
		}
		return []byte(config.AuthTokenKey), nil
	})

	if err != nil {
		return nil, err
	}

	if token == nil || !token.Valid {
		return nil, nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)

	if !ok {
		err = errors.New("Couldn't get claims for JWT token")
		log.Println(err)
		return nil, err
	}

	uname, ok := claims["username"]

	if !ok {
		err = errors.New("Couldn't get username from JWT claims.")
		log.Println(err)
		return nil, err
	}

	var username string

	if username, ok = uname.(string); !ok {
		err = errors.New("JWT contained empty username.")
		log.Println(err)
		return nil, err
	}

	return GetUserAccount(username)
}
