package helpers

import (
	"fmt"
	"net/smtp"
	"os"
)

func SendVerifyMail(username, receiver string, code int) error {
	from := os.Getenv("MAIL_SENDER")
	pass := os.Getenv("MAIL_PASS")
	to := receiver
	host := os.Getenv("MAIL_HOST")
	port := os.Getenv("MAIL_PORT")

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: Blogger verification link\n\n" +
		fmt.Sprintf("Hello, %v\nClick the link: http://localhost:8080/api/v1/verify?code=%v to verify yourself", username, code)

	auth := smtp.PlainAuth("", from, pass, host)
	err := smtp.SendMail(host+":"+port, auth, from, []string{to}, []byte(msg))
	return err
}
