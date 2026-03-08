package helpers

import "math/rand"

func GenerateCode() int {
	code := rand.Intn(9000) + 1000

	return code
}
