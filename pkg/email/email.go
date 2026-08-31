package email

import (
	"log"
)

func SendEmail(to, subject, body string) error {
	log.Printf("[EMAIL] To: %s | Subject: %s\nBody:\n%s\n", to, subject, body)
	return nil
}
