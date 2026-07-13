package service

import (
	"crypto/tls"
	"fmt"
	"html"
	"net"
	"net/smtp"
	"strings"
	"time"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"

	"github.com/quocdev03/user-access-management/internal/config"
)

// MailSender abstraction: mock (no host) | plain local (Mailpit) | gomail (Resend…).
type MailSender interface {
	SendEmail(to, subject, body string) error
}

type MailService struct {
	sender MailSender
	cfg    *config.Config
}

func NewMailService(cfg *config.Config, logger *zap.Logger) *MailService {
	host := strings.TrimSpace(cfg.Mail.Host)
	from := strings.TrimSpace(cfg.Mail.From)
	if from == "" {
		from = "noreply@localhost"
	}

	var sender MailSender
	switch {
	case host == "":
		logger.Warn("SMTP_HOST trống — mock mail (không deliver)")
		sender = &mockMail{logger: logger}
	case isLocalMail(host, cfg.Mail.Port):
		logger.Info("Mail local/Mailpit",
			zap.String("host", host),
			zap.Int("port", cfg.Mail.Port),
			zap.String("inbox", "http://localhost:8025"),
		)
		sender = &localMail{host: host, port: cfg.Mail.Port, from: from, logger: logger}
	default:
		d := gomail.NewDialer(host, cfg.Mail.Port, cfg.Mail.User, cfg.Mail.Password)
		if cfg.Mail.Port == 465 || cfg.Mail.Port == 2465 {
			d.SSL = true
		}
		d.TLSConfig = &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
		logger.Info("Mail SMTP remote", zap.String("host", host), zap.Int("port", cfg.Mail.Port))
		sender = &remoteMail{dialer: d, from: from, logger: logger}
	}

	return &MailService{sender: sender, cfg: cfg}
}

func (s *MailService) SendEmail(to, subject, body string) error {
	return s.sender.SendEmail(to, subject, body)
}

func (s *MailService) SendVerificationEmail(to, otp string) error {
	body := fmt.Sprintf(`
		<h2>Xác thực tài khoản</h2>
		<p>Chào bạn,</p>
		<p>Mã OTP để xác thực tài khoản của bạn là: <strong>%s</strong></p>
		<p>Mã này có hiệu lực trong vòng 5 phút.</p>
		<p>Nếu bạn không thực hiện yêu cầu này, vui lòng bỏ qua email này.</p>
	`, otp)
	return s.SendEmail(to, "Xác thực tài khoản UAM", body)
}

func (s *MailService) SendPasswordResetEmail(to, token string) error {
	frontendURL := s.cfg.App.FrontendURL
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, token)
	body := fmt.Sprintf(`
		<h2>Khôi phục mật khẩu</h2>
		<p>Chào bạn,</p>
		<p>Bạn đã yêu cầu đặt lại mật khẩu. Vui lòng click vào link dưới đây để tiếp tục:</p>
		<p><a href="%s">Đặt lại mật khẩu</a></p>
		<p>Hoặc copy link này dán vào trình duyệt: <br/> %s</p>
		<p>Link này có hiệu lực trong vòng 1 giờ.</p>
		<p>Nếu bạn không yêu cầu, vui lòng bỏ qua email này.</p>
	`, resetURL, resetURL)
	return s.SendEmail(to, "Khôi phục mật khẩu UAM", body)
}

func (s *MailService) SendEmailChangeNotification(oldEmail, newEmail string) error {
	body := fmt.Sprintf(`
		<h2>Thông báo thay đổi Email</h2>
		<p>Chào bạn,</p>
		<p>Chúng tôi thông báo rằng địa chỉ email liên kết với tài khoản UAM của bạn đã được thay đổi thành công.</p>
		<p><strong>Email cũ:</strong> %s</p>
		<p><strong>Email mới:</strong> %s</p>
		<p>Nếu bạn không thực hiện thay đổi này, tài khoản của bạn có thể đã bị xâm nhập trái phép. Vui lòng liên hệ với bộ phận hỗ trợ khách hàng của chúng tôi ngay lập tức để được hỗ trợ bảo mật.</p>
	`, oldEmail, newEmail)
	subject := "Cảnh báo bảo mật: Địa chỉ Email của tài khoản đã thay đổi"
	if err := s.SendEmail(oldEmail, subject, body); err != nil {
		return err
	}
	return s.SendEmail(newEmail, subject, body)
}

func (s *MailService) SendAdminResetPasswordEmail(to, tempPassword string) error {
	body := fmt.Sprintf(`
		<h2>Đặt lại mật khẩu</h2>
		<p>Chào bạn,</p>
		<p>Quản trị viên hệ thống đã đặt lại mật khẩu cho tài khoản của bạn.</p>
		<p>Mật khẩu tạm thời của bạn là: <strong>%s</strong></p>
		<p>Vui lòng đăng nhập bằng mật khẩu này. Bạn sẽ được yêu cầu đổi mật khẩu ngay trong lần đăng nhập tiếp theo.</p>
	`, tempPassword)
	return s.SendEmail(to, "Thông báo đặt lại mật khẩu từ Quản trị viên", body)
}

func (s *MailService) SendAdminNotification(to, subject, message string) error {
	body := fmt.Sprintf(`
		<h2>Thông báo từ Hệ thống</h2>
		<p>Chào bạn,</p>
		<p>%s</p>
	`, html.EscapeString(message))
	return s.SendEmail(to, html.EscapeString(subject), body)
}

// --- transport ---

type mockMail struct{ logger *zap.Logger }

func (s *mockMail) SendEmail(to, subject, body string) error {
	s.logger.Info("[MOCK MAIL]", zap.String("to", to), zap.String("subject", subject), zap.Int("body_len", len(body)))
	return nil
}

// localMail: plain SMTP (no STARTTLS) for Mailpit. Host port 1026 on Windows Docker.
type localMail struct {
	host, from string
	port       int
	logger     *zap.Logger
}

func (s *localMail) SendEmail(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	conn, err := net.DialTimeout("tcp4", addr, 5*time.Second)
	if err != nil {
		s.logger.Error("SMTP local dial failed", zap.String("addr", addr), zap.Error(err))
		return fmt.Errorf("dial smtp %s: %w", addr, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))

	c, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if err := c.Mail(s.from); err != nil {
		return fmt.Errorf("smtp MAIL: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body)
	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	_ = c.Quit()

	s.logger.Info("Mail sent (local)", zap.String("to", to), zap.String("subject", subject))
	return nil
}

type remoteMail struct {
	dialer *gomail.Dialer
	from   string
	logger *zap.Logger
}

func (s *remoteMail) SendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	if err := s.dialer.DialAndSend(m); err != nil {
		s.logger.Error("SMTP remote failed", zap.String("to", to), zap.Error(err))
		return fmt.Errorf("failed to send email: %w", err)
	}
	s.logger.Info("Mail sent (remote)", zap.String("to", to), zap.String("subject", subject))
	return nil
}

func isLocalMail(host string, port int) bool {
	if port == 1025 || port == 1026 {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "127.0.0.1", "localhost", "mailpit", "::1":
		return true
	default:
		return false
	}
}
