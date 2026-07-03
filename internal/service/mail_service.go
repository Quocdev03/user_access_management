package service

import (
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"

	"github.com/quocdev03/user-access-management/internal/config"
)

type MailSender interface {
	SendEmail(to, subject, body string) error
}

type MockSender struct {
	logger *zap.Logger
}

func (s *MockSender) SendEmail(to, subject, body string) error {
	s.logger.Info("[MOCK MAIL] Gửi email giả lập thành công", zap.String("to", to), zap.String("subject", subject), zap.String("body", body))
	return nil
}


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

type MailService struct {
	sender MailSender
	cfg    *config.Config
}

func NewMailService(cfg *config.Config, logger *zap.Logger) *MailService {
	var sender MailSender

	if cfg.Mail.Host == "" {
		sender = &MockSender{logger: logger}
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

	return s.SendEmail(to, subject, body)
}

func (s *MailService) SendEmailChangeNotification(oldEmail, newEmail string) error {
	subject := "Cảnh báo bảo mật: Địa chỉ Email của tài khoản đã thay đổi"
	body := fmt.Sprintf(`
		<h2>Thông báo thay đổi Email</h2>
		<p>Chào bạn,</p>
		<p>Chúng tôi thông báo rằng địa chỉ email liên kết với tài khoản UAM của bạn đã được thay đổi thành công.</p>
		<p><strong>Email cũ:</strong> %s</p>
		<p><strong>Email mới:</strong> %s</p>
		<p>Nếu bạn không thực hiện thay đổi này, tài khoản của bạn có thể đã bị xâm nhập trái phép. Vui lòng liên hệ với bộ phận hỗ trợ khách hàng của chúng tôi ngay lập tức để được hỗ trợ bảo mật.</p>
	`, oldEmail, newEmail)

	if err := s.SendEmail(oldEmail, subject, body); err != nil {
		return err
	}
	if err := s.SendEmail(newEmail, subject, body); err != nil {
		return err
	}
	return nil
}
