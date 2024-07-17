package id

import "math/rand"

const IDChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
const IDCharsLen = len(IDChars)

func New(len int) string {
	current := ""

	for i := 0; i < len; i++ {
		current += string(IDChars[rand.Intn(IDCharsLen)])
	}

	return current
}
