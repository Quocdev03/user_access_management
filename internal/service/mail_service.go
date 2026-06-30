package service

import (
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"

	"github.com/quocdev03/user-access-management/internal/config"
)

type MailService interface {
	SendEmail(to, subject, body string) error
	SendVerificationEmail(to, otp string) error
	SendPasswordResetEmail(to, token string) error
}

type mailService struct {
	cfg    *config.Config
	logger *zap.Logger
}

func NewMailService(cfg *config.Config, logger *zap.Logger) MailService {
	return &mailService{
		cfg:    cfg,
		logger: logger,
	}
}

func (s *mailService) SendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.cfg.Mail.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(s.cfg.Mail.Host, s.cfg.Mail.Port, s.cfg.Mail.User, s.cfg.Mail.Password)

	// Nếu cấu hình host trống thì ta có thể mock để không bị lỗi trên môi trường dev chưa có SMTP
	if s.cfg.Mail.Host == "" {
		s.logger.Info("[MOCK MAIL] Bỏ qua gửi email do chưa cấu hình SMTP", zap.String("to", to), zap.String("subject", subject))
		return nil
	}

	if err := d.DialAndSend(m); err != nil {
		s.logger.Error("Lỗi gửi email", zap.String("to", to), zap.Error(err))
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("Đã gửi email thành công", zap.String("to", to), zap.String("subject", subject))
	return nil
}

func (s *mailService) SendVerificationEmail(to, otp string) error {
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

func (s *mailService) SendPasswordResetEmail(to, token string) error {
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
