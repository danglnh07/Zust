package util

import (
	"math/rand"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Utility method: generate a random number in interval [min, max]
func RandomInt(min, max int) int {
	return min + rand.Intn(max-min+1)
}

// Utility method: generate a random string with length n. The character possible is defined in the alphabet constant
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for range n {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}
