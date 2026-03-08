package helpers

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) ([]byte, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	return hashedPassword, err
}

func ComparePassword(hash []byte, password string) error {
	err := bcrypt.CompareHashAndPassword(hash, []byte(password))

	return err
}
