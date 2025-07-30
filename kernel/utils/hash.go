package utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"

	"golang.org/x/crypto/bcrypt"
)

func Md5(s string) string {
	hash := md5.New()
	io.WriteString(hash, s)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func GenRandomHash(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		log.Printf("Failed to generate random bytes: %s", err)
	}
	return base64.URLEncoding.EncodeToString(b)
}

func GenHash(str string) (string, error) {
	hashedStr, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("string hashing error: %s", err)
	}

	return string(hashedStr), err
}

func CheckHash(str string, hashedStr string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedStr), []byte(str))
	return err == nil
}
