package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := "123456"
	hash := "$2a$10$yQjCJwwcL7tvwEnJcMMz9ub5H.WURowkmdFkJasIzUZdMk28y1Tkm"

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	fmt.Printf("Password: %s\n", password)
	fmt.Printf("Hash: %s\n", hash)
	fmt.Printf("Valid: %v\n", err == nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
