package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/quocdev03/user-access-management/internal/config"
)

// MailSender abstraction: plain local (Mailpit) | Resend HTTP API.
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
		logger.Warn("SMTP_HOST trống — mail sẽ chỉ được log, không gửi thật")
		sender = &logOnlyMail{logger: logger}
	case host == "smtp.resend.com":
		logger.Info("Mail via Resend HTTP API", zap.String("from", from))
		sender = &resendHTTPMail{apiKey: cfg.Mail.Password, from: from, logger: logger}
	default:
		// Mailpit hoặc bất kỳ plain SMTP local nào
		logger.Info("Mail local/Mailpit",
			zap.String("host", host),
			zap.Int("port", cfg.Mail.Port),
		)
		sender = &localMail{host: host, port: cfg.Mail.Port, from: from, logger: logger}
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

// logOnlyMail: fallback khi SMTP_HOST trống — chỉ log, không gửi thật.
type logOnlyMail struct{ logger *zap.Logger }

func (s *logOnlyMail) SendEmail(to, subject, _ string) error {
	s.logger.Info("[LOG ONLY MAIL — không gửi thật]", zap.String("to", to), zap.String("subject", subject))
	return nil
}

// localMail: plain SMTP (no STARTTLS) cho Mailpit. Host port 1026 trên Windows Docker.
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

// resendHTTPMail gửi email qua Resend REST API (HTTPS port 443).
// Dùng cho production trên PaaS (Render, Railway…) nơi outbound SMTP bị chặn.
type resendHTTPMail struct {
	apiKey string
	from   string
	logger *zap.Logger
}

func (s *resendHTTPMail) SendEmail(to, subject, body string) error {
	payload := map[string]interface{}{
		"from":    s.from,
		"to":     []string{to},
		"subject": subject,
		"html":    body,
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal resend payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("Resend HTTP API failed", zap.String("to", to), zap.Error(err))
		return fmt.Errorf("resend API call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		s.logger.Error("Resend API error",
			zap.Int("status", resp.StatusCode),
			zap.String("to", to),
			zap.String("response", string(respBody)),
		)
		return fmt.Errorf("resend API status %d: %s", resp.StatusCode, string(respBody))
	}

	s.logger.Info("Mail sent (Resend API)", zap.String("to", to), zap.String("subject", subject))
	return nil
}

