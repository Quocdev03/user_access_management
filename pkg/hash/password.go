package hash

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength = 8

	lowerChars   = "abcdefghijklmnopqrstuvwxyz"
	upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberChars  = "0123456789"
	specialChars = "!@#$%^&*()-_=+[]{}<>?"
)

const allChars = lowerChars + upperChars + numberChars + specialChars

func SHA256(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func HashPassword(password string) (string, error) {
	if len(password) > bcrypt.MaxCost { // giữ nguyên check 72 bytes phía dưới
	}

	if len([]byte(password)) > 72 {
		return "", errors.New("mật khẩu vượt quá giới hạn 72 bytes của bcrypt")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func ValidatePasswordComplexity(password string) error {
	if len(password) < minPasswordLength {
		return errors.New("mật khẩu phải có tối thiểu 8 ký tự")
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, r := range password {
		switch {
		case 'a' <= r && r <= 'z':
			hasLower = true
		case 'A' <= r && r <= 'Z':
			hasUpper = true
		case '0' <= r && r <= '9':
			hasNumber = true
		case strings.ContainsRune(specialChars, r):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return errors.New("mật khẩu phải chứa ít nhất 1 chữ hoa, 1 chữ thường, 1 số và 1 ký tự đặc biệt")
	}

	return nil
}

func GenerateTempPassword(length int, oldHash string) (string, error) {
	if length < minPasswordLength {
		length = minPasswordLength
	}

	for {
		password := make([]byte, 0, length)

		for _, chars := range []string{
			lowerChars,
			upperChars,
			numberChars,
			specialChars,
		} {
			c, err := randomChar(chars)
			if err != nil {
				return "", err
			}
			password = append(password, c)
		}

		for len(password) < length {
			c, err := randomChar(allChars)
			if err != nil {
				return "", err
			}
			password = append(password, c)
		}

		if err := shuffle(password); err != nil {
			return "", err
		}

		if oldHash != "" && CheckPassword(string(password), oldHash) {
			continue
		}

		return string(password), nil
	}
}

func randomChar(chars string) (byte, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
	if err != nil {
		return 0, err
	}

	return chars[n.Int64()], nil
}

func shuffle(data []byte) error {
	for i := len(data) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}

		j := int(n.Int64())
		data[i], data[j] = data[j], data[i]
	}

	return nil
}
