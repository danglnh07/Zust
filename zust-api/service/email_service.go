package service

import (
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"zust/util"
)

type EmailService struct {
	Host     string
	Port     string
	Email    string
	Password string
}

func NewEmailService() *EmailService {
	// Load email configuration from environment variables
	config := util.GetConfig()

	return &EmailService{
		Host:     config.SMTPHost,
		Port:     config.SMTPPort,
		Email:    config.Email,
		Password: config.AppPassword,
	}
}

type VerificationEmailData struct {
	Username string
	Link     string
}

func (service *EmailService) PrepareEmail(data VerificationEmailData) (string, error) {
	// Create buffer
	tmpl, err := template.ParseFiles("../template/verification.html")
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	err = tmpl.Execute(&sb, data)
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}

func (service *EmailService) SendEmail(to, subject, body string) error {
	smtpAuth := smtp.PlainAuth("", service.Email, service.Password, service.Host)

	// Set email headers with MIME version and content type
	headers := make(map[string]string)
	headers["From"] = service.Email
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// Build the message with headers
	var message strings.Builder
	for key, value := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	message.WriteString("\r\n")
	message.WriteString(body)

	addr := fmt.Sprintf("%s:%s", service.Host, service.Port)
	return smtp.SendMail(
		addr,
		smtpAuth,
		service.Email,
		[]string{to},
		[]byte(message.String()),
	)
}
