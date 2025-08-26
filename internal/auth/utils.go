package auth

import (
	"golang.org/x/crypto/bcrypt"
)

func HashString(originalString string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(originalString), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func VerifyHashedString(originalString, hashedString string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedString), []byte(originalString))

	return err == nil
}
