package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: token <secret> <username>")
		os.Exit(1)
	}

	secret := []byte(os.Args[1])
	user := os.Args[2]

	claims := jwt.MapClaims{
		"sub": user,
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	s, err := token.SignedString(secret)
	if err != nil {
		panic(err)
	}

	fmt.Println(s)
}
