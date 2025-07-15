package handlers

import (
	"fmt"
	"net/smtp"
)

// SendEmailNotification sends email using Gmail SMTP
func SendEmailNotification(to, subject, body string) error {
	from := "ranawatapplication@gmail.com"
	password := "" // Use Gmail App Password

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	// Construct the email message
	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" + body + "\r\n")

	// Set up authentication info
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Send the email
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}
