package notifier

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
)

type EmailConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	FromName string
}

type EmailNotifier struct {
	cfg EmailConfig
}

func NewEmailNotifier(cfg EmailConfig) *EmailNotifier {
	return &EmailNotifier{cfg: cfg}
}

func (n *EmailNotifier) Send(ctx context.Context, identifier, code string) error {
	msg, err := n.message(identifier, code)
	if err != nil {
		return err
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(n.cfg.Host, n.cfg.Port))
	if err != nil {
		return fmt.Errorf("notifier: email: dial: %w", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return fmt.Errorf("notifier: email: set deadline: %w", err)
		}
	}

	c, err := smtp.NewClient(conn, n.cfg.Host)
	if err != nil {
		return fmt.Errorf("notifier: email: new client: %w", err)
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: n.cfg.Host}); err != nil {
			return fmt.Errorf("notifier: email: starttls: %w", err)
		}
	}

	if n.cfg.Username != "" {
		auth := smtp.PlainAuth("", n.cfg.Username, n.cfg.Password, n.cfg.Host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("notifier: email: auth: %w", err)
		}
	}

	if err := c.Mail(n.cfg.From); err != nil {
		return fmt.Errorf("notifier: email: mail from: %w", err)
	}
	if err := c.Rcpt(identifier); err != nil {
		return fmt.Errorf("notifier: email: rcpt to: %w", err)
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("notifier: email: data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("notifier: email: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("notifier: email: close data: %w", err)
	}

	return c.Quit()
}

func (n *EmailNotifier) message(to, code string) ([]byte, error) {
	if strings.ContainsAny(to, "\r\n") {
		return nil, fmt.Errorf("notifier: email: recipient contains a line break")
	}

	var parts bytes.Buffer
	mw := multipart.NewWriter(&parts)

	if err := writeTextPart(mw, "text/plain", textBody(code)); err != nil {
		return nil, err
	}
	if err := writeTextPart(mw, "text/html", htmlBody(code)); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("notifier: email: close multipart: %w", err)
	}

	var b strings.Builder
	b.WriteString("From: " + n.fromHeader() + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: Tasdiqlash kodi\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: multipart/alternative; boundary=\"" + mw.Boundary() + "\"\r\n")
	b.WriteString("\r\n")

	return append([]byte(b.String()), parts.Bytes()...), nil
}

func writeTextPart(mw *multipart.Writer, contentType, content string) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", contentType+`; charset="utf-8"`)
	h.Set("Content-Transfer-Encoding", "quoted-printable")

	w, err := mw.CreatePart(h)
	if err != nil {
		return fmt.Errorf("notifier: email: create %s part: %w", contentType, err)
	}

	qp := quotedprintable.NewWriter(w)
	if _, err := io.WriteString(qp, content); err != nil {
		return fmt.Errorf("notifier: email: write %s part: %w", contentType, err)
	}
	if err := qp.Close(); err != nil {
		return fmt.Errorf("notifier: email: close %s part: %w", contentType, err)
	}
	return nil
}

func textBody(code string) string {
	return "Tasdiqlash kodingiz: " + code + "\n" +
		"Kod 5 daqiqa amal qiladi.\n" +
		"Kodni hech kimga bermang.\n\n" +
		"Agar bu so'rovni siz yubormagan bo'lsangiz, xatni e'tiborsiz qoldiring.\n"
}

func htmlBody(code string) string {
	safe := html.EscapeString(code)

	return `<!DOCTYPE html>
<html lang="uz">
<body style="margin:0;padding:0;background-color:#f1f3f7;">
<div style="display:none;max-height:0;overflow:hidden;opacity:0;">Tasdiqlash kodingiz: ` + safe + `</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f1f3f7;">
<tr>
<td align="center" style="padding:40px 16px;">

<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="max-width:460px;background-color:#ffffff;border-radius:14px;border:1px solid #e3e6ec;">
<tr>
<td style="padding:36px 36px 8px 36px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Arial,sans-serif;">
<div style="font-size:13px;font-weight:600;letter-spacing:1.4px;text-transform:uppercase;color:#5b6478;">Dehqon Bobo</div>
</td>
</tr>
<tr>
<td style="padding:16px 36px 0 36px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Arial,sans-serif;">
<h1 style="margin:0;font-size:21px;line-height:1.35;font-weight:600;color:#141821;">Tasdiqlash kodi</h1>
<p style="margin:10px 0 0 0;font-size:15px;line-height:1.6;color:#5b6478;">Hisobingizga kirish uchun quyidagi kodni kiriting.</p>
</td>
</tr>
<tr>
<td style="padding:24px 36px 0 36px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f5f6fa;border-radius:10px;">
<tr>
<td align="center" style="padding:22px 12px;font-family:'SF Mono',Menlo,Consolas,monospace;font-size:32px;font-weight:700;letter-spacing:9px;color:#141821;">` + safe + `</td>
</tr>
</table>
</td>
</tr>
<tr>
<td style="padding:18px 36px 0 36px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Arial,sans-serif;">
<p style="margin:0;font-size:14px;line-height:1.6;color:#5b6478;">Kod <strong style="color:#141821;">5 daqiqa</strong> amal qiladi. Uni hech kimga bermang &mdash; xodimlarimiz ham hech qachon so'ramaydi.</p>
</td>
</tr>
<tr>
<td style="padding:24px 36px 32px 36px;">
<div style="height:1px;background-color:#e9ebf0;font-size:0;line-height:0;">&nbsp;</div>
<p style="margin:18px 0 0 0;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Arial,sans-serif;font-size:13px;line-height:1.6;color:#8a92a6;">Agar bu so'rovni siz yubormagan bo'lsangiz, xatni e'tiborsiz qoldiring.</p>
</td>
</tr>
</table>

</td>
</tr>
</table>
</body>
</html>
`
}

func (n *EmailNotifier) fromHeader() string {
	name := strings.TrimSpace(n.cfg.FromName)
	if name == "" {
		return n.cfg.From
	}
	return mime.QEncoding.Encode("utf-8", name) + " <" + n.cfg.From + ">"
}
