// Package utils provides small standalone helpers (hashing, slice
// operations) used across lxgo-based applications.
package utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// Md5 returns s's hex-encoded MD5 hash.
func Md5(s string) string {
	hash := md5.New()
	io.WriteString(hash, s)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// GenRandomHash returns a URL-safe base64 encoding of n cryptographically
// random bytes - panics if the system's random source fails.
func GenRandomHash(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %s", err))
	}
	return base64.URLEncoding.EncodeToString(b)
}

// GenHash returns str's bcrypt hash - suitable for storing a password. See CheckHash.
func GenHash(str string) (string, error) {
	hashedStr, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("string hashing error: %s", err)
	}

	return string(hashedStr), err
}

// CheckHash reports whether str matches the bcrypt hash hashedStr - see GenHash.
func CheckHash(str string, hashedStr string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedStr), []byte(str))
	return err == nil
}
