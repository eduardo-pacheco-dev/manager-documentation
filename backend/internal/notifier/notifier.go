package notifier

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/config"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/models"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/store"
)

type Notifier struct {
	store  *store.Store
	config config.Config
}

func New(s *store.Store, cfg config.Config) *Notifier {
	return &Notifier{store: s, config: cfg}
}

var validLevels = map[string]bool{
	"info": true, "success": true, "warning": true, "danger": true,
}

// Send delivers a plain-text email using the configured SMTP server. It is a
// no-op when SMTP is not configured.
func (n *Notifier) Send(to, subject, body string) error {
	if n.config.SMTPHost == "" || strings.TrimSpace(to) == "" {
		return nil
	}
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		n.config.SMTPFrom, to, subject, body)
	address := net.JoinHostPort(n.config.SMTPHost, fmt.Sprintf("%d", n.config.SMTPPort))
	var auth smtp.Auth
	if n.config.SMTPUser != "" {
		auth = smtp.PlainAuth("", n.config.SMTPUser, n.config.SMTPPassword, n.config.SMTPHost)
	}
	if n.config.SMTPPort == 465 {
		return sendTLS(address, n.config.SMTPHost, auth, n.config.SMTPFrom, to, []byte(message))
	}
	return smtp.SendMail(address, auth, n.config.SMTPFrom, []string{to}, []byte(message))
}

// Notify stores an in-app notification and sends an e-mail when the account
// opted in and SMTP is configured.
func (n *Notifier) Notify(user models.User, title, body, link, level string) error {
	if !validLevels[level] {
		level = "info"
	}
	safeLink := SafeURL(link)
	notification := models.Notification{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		Title:     strings.TrimSpace(title),
		Body:      strings.TrimSpace(body),
		URL:       safeLink,
		Level:     level,
		CreatedAt: time.Now().UTC(),
	}
	if err := n.store.CreateNotification(&notification); err != nil {
		return err
	}
	if user.EmailNotifications {
		go n.sendEmail(user.Email, notification)
	}
	return nil
}

// NotifyOnce stores an in-app notification only once per dedupe key. It is
// used by scheduled cross-module reminders.
func (n *Notifier) NotifyOnce(user models.User, dedupeKey, title, body, link, level string) (bool, error) {
	if !validLevels[level] {
		level = "info"
	}
	key := strings.TrimSpace(dedupeKey)
	notification := models.Notification{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		Title:     strings.TrimSpace(title),
		Body:      strings.TrimSpace(body),
		URL:       SafeURL(link),
		Level:     level,
		CreatedAt: time.Now().UTC(),
	}
	if key != "" {
		notification.DedupeKey = &key
	}
	created, err := n.store.CreateNotificationOnce(&notification)
	if err != nil || !created {
		return false, err
	}
	if user.EmailNotifications {
		go n.sendEmail(user.Email, notification)
	}
	return true, nil
}

func (n *Notifier) sendEmail(to string, notification models.Notification) {
	if n.config.SMTPHost == "" {
		return
	}
	action := ""
	if notification.URL != nil {
		action = fmt.Sprintf("\n\nAbrir: %s%s", n.config.PublicURL, *notification.URL)
	}
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s%s\r\n",
		n.config.SMTPFrom, to, notification.Title, notification.Body, action)

	address := net.JoinHostPort(n.config.SMTPHost, fmt.Sprintf("%d", n.config.SMTPPort))
	var auth smtp.Auth
	if n.config.SMTPUser != "" {
		auth = smtp.PlainAuth("", n.config.SMTPUser, n.config.SMTPPassword, n.config.SMTPHost)
	}

	var err error
	if n.config.SMTPPort == 465 {
		err = sendTLS(address, n.config.SMTPHost, auth, n.config.SMTPFrom, to, []byte(message))
	} else {
		err = smtp.SendMail(address, auth, n.config.SMTPFrom, []string{to}, []byte(message))
	}
	if err != nil {
		log.Printf("notifier: failed to send email to %s: %v", to, err)
	}
}

func sendTLS(address, host string, auth smtp.Auth, from, to string, message []byte) error {
	conn, err := tls.Dial("tcp", address, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	if err != nil {
		return err
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// SafeURL allows relative paths (a single leading slash) and absolute http(s)
// URLs only, rejecting schemes such as javascript:.
func SafeURL(raw string) *string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return &raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil
	}
	return &raw
}
