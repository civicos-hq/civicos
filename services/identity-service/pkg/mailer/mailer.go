package mailer

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Mailer is the only contract callers depend on. Swap implementations
// (SMTP, console, future SES/Postmark) without touching the service layer.
type Mailer interface {
	Send(to, subject, htmlBody, textBody string) error
}

// SMTPMailer speaks to any RFC-compliant SMTP relay (Mailpit, Gmail, Resend,
// SendGrid, Postmark, SES). Behaviour by port:
//
//	465  → implicit TLS from the first byte
//	587  → plain TCP, then STARTTLS upgrade (required by most modern providers)
//	other (25, 1025, …) → plain TCP, STARTTLS used only if the server advertises it
//
// PlainAuth runs only when a username is configured — Mailpit and other dev
// relays accept unauthenticated SMTP.
type SMTPMailer struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

func NewSMTPMailer(host string, port int, user, password, from string) *SMTPMailer {
	return &SMTPMailer{host: host, port: port, user: user, password: password, from: from}
}

func (m *SMTPMailer) Send(to, subject, htmlBody, textBody string) error {
	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	msg := buildMime(m.from, to, subject, htmlBody, textBody)

	conn, err := dial(addr, m.host, m.port == 465)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	// Upgrade to TLS via STARTTLS when the server supports it. Required for
	// Gmail/Resend/SES on 587; harmless when the relay (Mailpit) doesn't
	// advertise the extension. On 587 specifically we treat STARTTLS as
	// mandatory so a misconfigured relay can't silently leak credentials.
	if m.port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
				return fmt.Errorf("starttls: %w", err)
			}
		} else if m.port == 587 {
			return fmt.Errorf("smtp server on port 587 did not advertise STARTTLS — refusing to send credentials over plaintext")
		}
	}

	if m.user != "" {
		auth := smtp.PlainAuth("", m.user, m.password, m.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}

	if err := client.Mail(addrOnly(m.from)); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close body: %w", err)
	}
	return client.Quit()
}

// dial returns a connection ready for smtp.NewClient. Implicit TLS (port 465)
// wraps the socket in TLS immediately; everything else is plain TCP that may
// be upgraded later via STARTTLS.
func dial(addr, host string, implicitTLS bool) (net.Conn, error) {
	d := &net.Dialer{Timeout: 10 * time.Second}
	if implicitTLS {
		return tls.DialWithDialer(d, "tcp", addr, &tls.Config{ServerName: host})
	}
	return d.Dial("tcp", addr)
}

// ConsoleMailer prints the message to stdout. Useful when no SMTP is
// configured — verification links land in the service log so a developer
// can copy them straight into the browser.
type ConsoleMailer struct {
	from string
}

func NewConsoleMailer(from string) *ConsoleMailer {
	return &ConsoleMailer{from: from}
}

func (m *ConsoleMailer) Send(to, subject, htmlBody, textBody string) error {
	log.Printf(
		"\n────── 📨 mail (console) ──────\nFrom: %s\nTo:   %s\nSubj: %s\n\n%s\n────────────────────────────────\n",
		m.from, to, subject, textBody,
	)
	return nil
}

func buildMime(from, to, subject, htmlBody, textBody string) []byte {
	boundary := "civicos-mime-boundary"
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	b.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary)

	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(textBody)
	b.WriteString("\r\n\r\n")

	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	b.WriteString(htmlBody)
	b.WriteString("\r\n\r\n")

	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return []byte(b.String())
}

// addrOnly extracts the bare "user@host" from a "Name <user@host>" string,
// because smtp.SendMail rejects the friendly-name form as a MAIL FROM.
func addrOnly(s string) string {
	if i := strings.LastIndex(s, "<"); i != -1 {
		if j := strings.Index(s[i:], ">"); j != -1 {
			return s[i+1 : i+j]
		}
	}
	return s
}
