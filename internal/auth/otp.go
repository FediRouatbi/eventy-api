package auth

import (
	"crypto/rand"
	"math/big"
	"strings"
)

func generateOTPCode() (string, error) {
	var builder strings.Builder

	for range 4 {
		number, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}

		builder.WriteByte(byte('0') + byte(number.Int64()))
	}

	return builder.String(), nil
}
