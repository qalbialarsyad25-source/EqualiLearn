package bcrypt

import "golang.org/x/crypto/bcrypt"

type IBcrypt interface {
	GenerateHash(password string) (string, error)
	ValidatePassword(hashedPassword, password string) error
}

type Bcrypt struct {
	cost int
}

func NewBcrypt() IBcrypt {
	return &Bcrypt{cost: bcrypt.DefaultCost}
}

func (b *Bcrypt) GenerateHash(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (b *Bcrypt) ValidatePassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return err
	}
	return nil
}
