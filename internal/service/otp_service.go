package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"math/big"
	"time"

	"go.uber.org/zap"

	"github.com/quocdev03/user-access-management/internal/repository"
	"github.com/quocdev03/user-access-management/pkg/apperror"
)

const maxOTPAttempts = 5

type OTPService struct {
	otpRepo     *repository.OTPRepository
	mailService *MailService
	logger      *zap.Logger
}

func NewOTPService(otpRepo *repository.OTPRepository, mailService *MailService, logger *zap.Logger) *OTPService {
	return &OTPService{
		otpRepo:     otpRepo,
		mailService: mailService,
		logger:      logger,
	}
}

func (s *OTPService) GenerateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("generateOTP: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s *OTPService) CreateAndSendOTP(ctx context.Context, userID uint64, email string, otpType string, expiresAt time.Time) (func(), error) {
	otp, err := s.GenerateOTP()
	if err != nil {
		return nil, err
	}

	if err := s.otpRepo.Create(ctx, userID, otp, otpType, expiresAt); err != nil {
		return nil, fmt.Errorf("otpRepo.Create: %w", err)
	}

	sendFunc := func() {
		if err := s.mailService.SendVerificationEmail(email, otp); err != nil {
			s.logger.Error("Không thể gửi email OTP", zap.String("email", email), zap.Error(err))
		}
	}

	return sendFunc, nil
}

func (s *OTPService) VerifyOTP(ctx context.Context, userID uint64, otpType string, inputCode string) error {
	otpCode, err := s.otpRepo.GetLatestValidCodeForUpdate(ctx, userID, otpType)
	if err != nil {
		return fmt.Errorf("otpRepo.GetLatestValidCodeForUpdate: %w", err)
	}
	if otpCode == nil {
		return apperror.ErrOTPExpired
	}

	if otpCode.Attempts >= maxOTPAttempts {
		return apperror.ErrOTPMaxAttempts
	}

	if subtle.ConstantTimeCompare([]byte(otpCode.Code), []byte(inputCode)) != 1 {
		attempts, err := s.otpRepo.IncrementAttempts(ctx, otpCode.ID)
		if err != nil {
			return fmt.Errorf("otpRepo.IncrementAttempts: %w", err)
		}
		if attempts >= maxOTPAttempts {
			return apperror.ErrOTPMaxAttempts
		}
		return apperror.ErrOTPInvalid
	}

	if err := s.otpRepo.MarkAsUsed(ctx, otpCode.ID); err != nil {
		return fmt.Errorf("otpRepo.MarkAsUsed: %w", err)
	}

	return nil
}
