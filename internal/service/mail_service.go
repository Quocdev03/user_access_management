package service

import (
	"fmt"

	"github.com/resend/resend-go/v2"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"

	"github.com/quocdev03/user-access-management/internal/config"
)

// MailSender định nghĩa giao diện chuẩn cho mọi dịch vụ gửi mail (Resend, SMTP, Mock, SendGrid...)
type MailSender interface {
	SendEmail(to, subject, body string) error
}

// ----------------------------------------------------------------------
// 1. Mock Sender (Dùng khi chưa cấu hình host)
// ----------------------------------------------------------------------
type MockSender struct {
	logger *zap.Logger
}

func (s *MockSender) SendEmail(to, subject, body string) error {
	s.logger.Info("[MOCK MAIL] Bỏ qua gửi email thực tế. Nội dung email được in ra bên dưới:", zap.String("to", to), zap.String("subject", subject))
	
	// In thẳng nội dung ra Console để Dev có thể lấy OTP hoặc Link Đặt lại mật khẩu
	fmt.Println("\n================ MOCK EMAIL CONTENT ================")
	fmt.Println("TO      :", to)
	fmt.Println("SUBJECT :", subject)
	fmt.Println("BODY    :\n", body)
	fmt.Println("====================================================\n")
	
	return nil
}

// ----------------------------------------------------------------------
// 2. Resend Sender (Dùng qua SDK/HTTP API để né chặn cổng)
// ----------------------------------------------------------------------
type ResendSender struct {
	client *resend.Client
	from   string
	logger *zap.Logger
}

func (s *ResendSender) SendEmail(to, subject, body string) error {
	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: subject,
		Html:    body,
	}

	if _, err := s.client.Emails.Send(params); err != nil {
		s.logger.Error("Lỗi gửi email qua Resend SDK", zap.String("to", to), zap.Error(err))
		return fmt.Errorf("failed to send via resend sdk: %w", err)
	}

	s.logger.Info("Đã gửi email thành công qua Resend SDK", zap.String("to", to), zap.String("subject", subject))
	return nil
}

// ----------------------------------------------------------------------
// 3. SMTP Sender (Dùng cho gomail tiêu chuẩn)
// ----------------------------------------------------------------------
type SMTPSender struct {
	dialer *gomail.Dialer
	from   string
	logger *zap.Logger
}

func (s *SMTPSender) SendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	if err := s.dialer.DialAndSend(m); err != nil {
		s.logger.Error("Lỗi gửi email SMTP", zap.String("to", to), zap.Error(err))
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("Đã gửi email thành công qua SMTP", zap.String("to", to), zap.String("subject", subject))
	return nil
}

// ----------------------------------------------------------------------
// Core MailService
// ----------------------------------------------------------------------
type MailService struct {
	sender MailSender
	cfg    *config.Config
}

// NewMailService đóng vai trò là Factory nạp đúng Sender dựa trên Config
func NewMailService(cfg *config.Config, logger *zap.Logger) *MailService {
	var sender MailSender

	if cfg.Mail.Host == "" {
		sender = &MockSender{logger: logger}
	} else if cfg.Mail.Host == "smtp.resend.com" {
		sender = &ResendSender{
			client: resend.NewClient(cfg.Mail.Password),
			from:   cfg.Mail.From,
			logger: logger,
		}
	} else {
		sender = &SMTPSender{
			dialer: gomail.NewDialer(cfg.Mail.Host, cfg.Mail.Port, cfg.Mail.User, cfg.Mail.Password),
			from:   cfg.Mail.From,
			logger: logger,
		}
	}

	return &MailService{
		sender: sender,
		cfg:    cfg,
	}
}

// Gọi sender interface, không quan tâm nó là Resend hay SMTP
func (s *MailService) SendEmail(to, subject, body string) error {
	return s.sender.SendEmail(to, subject, body)
}

func (s *MailService) SendVerificationEmail(to, otp string) error {
	subject := "Xác thực tài khoản UAM"
	body := fmt.Sprintf(`
		<h2>Xác thực tài khoản</h2>
		<p>Chào bạn,</p>
		<p>Mã OTP để xác thực tài khoản của bạn là: <strong>%s</strong></p>
		<p>Mã này có hiệu lực trong vòng 5 phút.</p>
		<p>Nếu bạn không thực hiện yêu cầu này, vui lòng bỏ qua email này.</p>
	`, otp)

	return s.SendEmail(to, subject, body)
}

func (s *MailService) SendPasswordResetEmail(to, token string) error {
	subject := "Khôi phục mật khẩu UAM"

	// Sử dụng biến môi trường APP_FRONTEND_URL để ghép link
	frontendURL := s.cfg.App.FrontendURL
	if frontendURL == "" {
		frontendURL = "http://localhost:3000" // Fallback nếu quên cấu hình
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

	return s.SendEmail(to, subject, body)
}
