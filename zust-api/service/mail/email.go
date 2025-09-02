package mail

import (
	"fmt"
	"net/smtp"
	"strings"
	"text/template"

	"zust/service/security"
)

// Email service struct, which holds configurations related to email sending
type EmailService struct {
	Host  string
	Port  string
	Email string
	Auth  smtp.Auth
}

// Constructing method for email service struct
func NewEmailService(config *security.Config) *EmailService {
	// Try simple authentication
	smtpAuth := smtp.PlainAuth("", config.Email, config.AppPassword, config.SMTPHost)

	return &EmailService{
		Host:  config.SMTPHost,
		Port:  config.SMTPPort,
		Email: config.Email,
		Auth:  smtpAuth,
	}
}

// Verification (account activation) email payload
type VerificationEmailPayload struct {
	Username string
	Link     string
}

// Method to prepare email payload.
// 'templ' is the path to where the HTML email located
// Note that this method won't do any type checking whether templ and payload actually match before processing
func (service *EmailService) PrepareEmail(templ string, payload any) (string, error) {
	// Create buffer
	tmpl, err := template.ParseFiles(templ)
	if err != nil {
		return "", err
	}

	// Execute template into buffer
	var sb strings.Builder
	err = tmpl.Execute(&sb, payload)
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}

// Method to send email
func (service *EmailService) SendEmail(to, subject, body string) error {
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
		service.Auth,
		service.Email,
		[]string{to},
		[]byte(message.String()),
	)
}
