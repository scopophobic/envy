package services

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"github.com/envo/backend/internal/config"
)

// EmailSender sends transactional emails.
type EmailSender interface {
	SendInvite(toEmail, orgName, inviterName, roleName, inviteURL string) error
}

// LogEmailSender is a safe fallback for dev/local environments.
type LogEmailSender struct{}

func (s *LogEmailSender) SendInvite(toEmail, orgName, inviterName, roleName, inviteURL string) error {
	log.Printf("[email] invite to=%s org=%q inviter=%q role=%q url=%s", toEmail, orgName, inviterName, roleName, inviteURL)
	return nil
}

// SMTPEmailSender sends invitations through SMTP.
type SMTPEmailSender struct {
	host      string
	port      string
	username  string
	password  string
	fromEmail string
	fromName  string
}

func NewSMTPEmailSender(cfg *config.Config) (*SMTPEmailSender, error) {
	if cfg.SMTPHost == "" || cfg.SMTPUsername == "" || cfg.SMTPPassword == "" || cfg.SMTPFromEmail == "" {
		return nil, fmt.Errorf("smtp configuration incomplete")
	}
	return &SMTPEmailSender{
		host:      cfg.SMTPHost,
		port:      cfg.SMTPPort,
		username:  cfg.SMTPUsername,
		password:  cfg.SMTPPassword,
		fromEmail: cfg.SMTPFromEmail,
		fromName:  cfg.SMTPFromName,
	}, nil
}

func (s *SMTPEmailSender) SendInvite(toEmail, orgName, inviterName, roleName, inviteURL string) error {
	subject := "You're invited to join an Envo workspace"
	body := fmt.Sprintf(
		"Hello,\n\n%s invited you to join \"%s\" in Envo as %s.\n\nAccept invitation:\n%s\n\nIf you do not have an account yet, sign in with this email first and then accept.\n\n- Envo\n",
		inviterName, orgName, roleName, inviteURL,
	)

	message := strings.Builder{}
	message.WriteString(fmt.Sprintf("From: %s <%s>\r\n", s.fromName, s.fromEmail))
	message.WriteString(fmt.Sprintf("To: %s\r\n", toEmail))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	message.WriteString("\r\n")
	message.WriteString(body)

	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	return smtp.SendMail(addr, auth, s.fromEmail, []string{toEmail}, []byte(message.String()))
}

