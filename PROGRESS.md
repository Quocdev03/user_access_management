# UAM — Tiến độ triển khai

> Cập nhật lần cuối: 2026-06-28

---

## ✅ Đã hoàn thành

### 🏗️ Nền móng (Base Infrastructure)
| File | Mô tả |
|------|-------|
| `cmd/server/main.go` | Entry point: load config, init logger, kết nối MySQL/Redis, graceful shutdown |
| `internal/router/router.go` | Gin router, health check `/health`, `/health/ready`, group `/api/v1` |
| `pkg/response/response.go` | Chuẩn hóa response thành công/lỗi |
| `pkg/apperror/errors.go` | Custom error với HTTP status code |
| `pkg/logger/logger.go` | Zap structured logger (dev/prod mode) |
| `pkg/hash/password.go` | bcrypt hash + verify password |
| `migrations/` | 11 file migration đầy đủ (users, roles, permissions, sessions, devices, otp_codes, audit_logs, seed data) |

---

### 🔐 Auth Module — UC-01, 02, 03 (Phase 1)

#### Models
| File | Nội dung |
|------|---------|
| `internal/model/user.go` | Struct `User`, constants `StatusActive/Inactive/Locked` |
| `internal/model/role.go` | Struct `Role` |
| `internal/model/otp.go` | Struct `OTPCode` (có `attempts`, `is_used`) |

#### DTOs
| File | Nội dung |
|------|---------|
| `internal/dto/auth.go` | `RegisterRequest/Response`, `VerifyEmailRequest`, `LoginRequest/Response`, `UserInfoResponse` |

#### Repository Layer (MySQL)
| File | Methods |
|------|---------|
| `internal/repository/user_repository.go` | `Create`, `FindByEmail`, `FindByUsername`, `UpdateStatus`, `UpdateLastLogin`, `IncrementFailedLogins`, `LockAccount` |
| `internal/repository/otp_repository.go` | `Create`, `GetLatestValidCode`, `IncrementAttempts`, `MarkAsUsed` |
| `internal/repository/role_repository.go` | `FindByName`, `AssignRoleToUser` |

#### Service Layer (Business Logic)
| File | Logic đã implement |
|------|------------------|
| `internal/service/auth_service.go` | **Register**: hash pass → check email/username trùng → tạo user inactive → gán role `user` → sinh OTP (crypto/rand) → log mock mail |
| | **VerifyEmail**: validate OTP → đếm lần sai → block sau 5 lần → mark as used → active account |
| | **Login**: kiểm tra status/lock → verify pass → tự unlock nếu hết hạn → khóa 30p sau 5 lần sai → update last_login |

#### Handler & Routes
| File | Routes |
|------|-------|
| `internal/handler/auth_handler.go` | `Register`, `VerifyEmail`, `Login` |
| `internal/router/router.go` | `POST /api/v1/auth/register`, `POST /api/v1/auth/verify-email`, `POST /api/v1/auth/login` |

---

## 🐛 Bugs đã fix (Self-Review)
| # | Vấn đề | Fix |
|---|--------|-----|
| A1 | Logic off-by-one khi kiểm tra ngưỡng khóa | Kiểm tra trước, increment sau |
| A2 | Thời gian khóa sai spec (15p → 30p) | Constant lockDuration = 30 * time.Minute |
| A3 | Thiếu kiểm tra username trùng | Thêm FindByUsername + check trong Register |
| A4 | math/rand không đủ bảo mật cho OTP | Đổi sang crypto/rand |
| B1 | fmt.Printf vi phạm convention | Inject Zap Logger |
| B2 | Thiếu error wrapping | fmt.Errorf("context: %w", err) |
| B3 | Anonymous struct trong DTO | Tách thành UserInfoResponse |

---

## 🚧 Chưa làm / TODO

| UC | Tính năng | Ghi chú |
|----|----------|---------|
| UC-03 | JWT Token thật (access + refresh) | Đang mock string |
| UC-04 | Refresh Token endpoint | Chưa có |
| UC-05 | Logout (blacklist Redis) | Chưa có |
| UC-07 | Forgot Password | Chưa có |
| UC-08 | Change Password | Chưa có |
| UC-20 | Lưu session vào DB & Redis | Chưa có |
| UC-25 | Gửi email thật (SMTP) | Đang log console |
| UC-28 | Audit log cho Login/Register | Chưa có |
| UC-11~14 | User Profile (xem, sửa, avatar, đổi email) | Chưa có |
| UC-15~18 | Admin quản lý user | Chưa có |
| UC-38 | Unit & Integration Test | Chưa có |

---

## 🔑 Bước tiếp theo đề xuất

1. **Implement JWT Token** (`pkg/jwt/jwt.go`) → hoàn thiện Login
2. **UC-04 Refresh Token** + **UC-05 Logout** → hoàn chỉnh Auth flow
3. **UC-25 Email Service** (SMTP) → thay mock bằng mail thật
