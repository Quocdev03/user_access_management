# UAM — Tiến độ triển khai

> Cập nhật lần cuối: 2026-06-29

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
| `pkg/jwt/jwt.go` | JWT sinh, parse và xác thực token (HS256) |
| `migrations/` | 11 file migration đầy đủ (users, roles, permissions, sessions, devices, otp_codes, audit_logs, seed data) |

---

### 🔐 Auth Module — UC-01, 02, 03, 04, 05 (Phase 1 & 2)

#### Models
| File | Nội dung |
|------|---------|
| `internal/model/user.go` | Struct `User`, constants `StatusActive/Inactive/Locked` |
| `internal/model/role.go` | Struct `Role` |
| `internal/model/otp.go` | Struct `OTPCode` (có `attempts`, `is_used`) |
| `internal/model/session.go` | Struct `Session` (tương ứng bảng database `sessions`) |

#### DTOs
| File | Nội dung |
|------|---------|
| `internal/dto/auth.go` | `RegisterRequest/Response`, `VerifyEmailRequest`, `LoginRequest/Response`, `UserInfoResponse`, `RefreshTokenRequest/Response` |

#### Repository Layer (MySQL & Redis)
| File | Methods |
|------|---------|
| `internal/repository/user_repository.go` | `Create`, `FindByEmail`, `FindByUsername`, `UpdateStatus`, `UpdateLastLogin`, `IncrementFailedLogins`, `LockAccount`, `FindByID`, `UpdatePassword` |
| `internal/repository/otp_repository.go` | `Create`, `GetLatestValidCode`, `IncrementAttempts`, `MarkAsUsed` |
| `internal/repository/password_reset_repo.go` | `Create`, `FindByTokenHash`, `MarkAsUsed`, `InvalidateAllUserTokens` |
| `internal/repository/role_repository.go` | `FindByName`, `AssignRoleToUser` |
| `internal/repository/session_repository.go` | `Create`, `Update`, `FindByRefreshTokenHash`, `DeleteByRefreshTokenHash`, `DeleteByTokenHash`, `DeleteByUserID`, `AddToBlacklist` (Redis), `IsBlacklisted` (Redis) |

#### Service Layer (Business Logic)
| File | Logic đã implement |
|------|------------------|
| `internal/service/auth_service.go` | **Register**: hash pass → check email/username trùng → tạo user inactive → gán role `user` → sinh OTP (crypto/rand) → gửi email thực tế |
| | **VerifyEmail**: validate OTP → đếm lần sai → block sau 5 lần → mark as used → active account |
| | **ResendVerificationEmail**: tìm user → kiểm tra trạng thái → sinh OTP mới → lưu DB → gửi lại OTP (UC-02) |
| | **Login**: kiểm tra status/lock → verify pass → tự unlock nếu hết hạn → khóa 30p sau 5 lần sai → update last_login → sinh JWT thật (access & refresh) → tạo session trong DB |
| | **Refresh**: validate refresh token → kiểm tra session DB → rotation (tạo mới cả 2 token) → thu hồi session cũ và tạo session mới |
| | **Logout**: tính TTL còn lại của access token → thêm JTI vào Redis blacklist → xóa session trong MySQL bằng token_hash |
| | **Forgot/Reset Password**: sinh mã token băm, gửi email link, đổi mật khẩu và vô hiệu hóa session |
| | **Change Password**: verify mật khẩu cũ, cập nhật và vô hiệu hóa các thiết bị khác |
| `internal/service/mail_service.go` | **SendEmail**: tích hợp SMTP qua `gomail.v2` gửi email xác thực, khôi phục mật khẩu |

#### Handler & Routes
| File | Routes |
|------|-------|
| `internal/middleware/auth_middleware.go` | Middleware xác thực JWT access token, kiểm tra Redis blacklist, inject user context |
| `internal/handler/auth_handler.go` | `Register`, `VerifyEmail`, `Login`, `RefreshToken`, `Logout`, `ForgotPassword`, `ResetPassword`, `ChangePassword` |
| `internal/router/router.go` | `POST /api/v1/auth/register`, `POST /api/v1/auth/verify-email`, `POST /api/v1/auth/login`, `POST /api/v1/auth/refresh-token`, `POST /api/v1/auth/logout`, `POST /api/v1/auth/forgot-password`, `POST /api/v1/auth/reset-password`, `POST /api/v1/auth/change-password` |

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
| B4 | Xung đột package name `jwt` | Alias thư viện `golang-jwt` bên ngoài thành `golangjwt` để tránh đè namespace |
| B5 | Kiểu dữ liệu User ID không đồng bộ | Đồng bộ Claims.UserID thành uint64 |
| B6 | Rủi ro mất session khi Refresh | Chuyển sang UPDATE session thay vì Delete+Create |
| B7 | Lỗi sập Redis gây Fail-Closed | Chuyển sang cơ chế Fail-Open cho Redis blacklist check |
| B8 | Comment tiếng Anh và thiếu comment | Chuẩn hóa toàn bộ comment tiếng Anh sang tiếng Việt, bổ sung đầy đủ comment tiếng Việt cho toàn bộ struct/interface/method/logic trong internal/ và pkg/ |
| B9 | Lỗi parse struct do kiểu dữ liệu NULL (LastLoginAt, DateOfBirth, Phone, AvatarURL) | Sửa thành dùng con trỏ (pointer) trong struct User |
| B10 | Nguy cơ lỗi 500 do Database Data Truncation và Type Mismatch (DateOfBirth string vs time.Time) | Bổ sung tag `max=`, `datetime=` vào DTO, đổi `DateOfBirth` thành `time.Time` và dùng `time.Parse` |
| B11 | Lỗi `duplicate migration file` khi chạy `make.cmd migrate-up` do có 2 file version 000001 | Xóa các file `000001_init_schema`, di chuyển nội dung ra file `database.sql` bên ngoài root |
| M1 | Yêu cầu nghiệp vụ bắt buộc khai báo Phone, DateOfBirth lúc đăng ký | Cập nhật file migration sang NOT NULL, DTO sang required, code service/repository tương ứng |
| M2 | Thiếu công cụ chạy Makefile trên Windows | Bổ sung file `make.cmd` hỗ trợ chạy các lệnh tương đương trên CMD/PowerShell |
| M3 | Tối ưu tài nguyên hệ thống Docker | Gỡ bỏ `adminer` và `redis-commander` khỏi `docker-compose.yml`, chuyển sang dùng Desktop App (DBeaver, Redis Insight) |
| M4 | Cung cấp tài khoản Super Admin sẵn để dễ dàng test | Tạo migration `000012_seed_superadmin` sinh sẵn user `admin_quocdev` với đầy đủ các field và role `admin` |
| B12 | Rủi ro lỗi panic khi query database nếu dữ liệu user thiếu họ tên (`full_name` = NULL) | Sửa file migration `000001` cập nhật `full_name` sang `NOT NULL`, reset lại database. Đồng thời cập nhật lại schema trong Docs và file `database.sql` |
| B13 | VULN-001: Lỗ hổng TOCTOU (Race Condition) khi Rate Limit đăng nhập | Sử dụng Transaction (Pessimistic Lock) kết hợp nguyên tử hóa cập nhật `failed_login_attempts` trong `UserRepository` |
| B14 | VULN-002: Không phát hiện Token Reuse khi kẻ xấu dùng lại Refresh Token cũ | Thêm cơ chế lưu Refresh Token cũ vào danh sách thu hồi (Redis). Xóa toàn bộ session của User nếu phát hiện reuse |
| B15 | VULN-003: Lỗ hổng Timing Attack dò tìm email | Bổ sung hàm băm mật khẩu giả (Dummy bcrypt hash) khi không tìm thấy user, cân bằng thời gian phản hồi |
| B16 | VULN-004: Lỗ hổng TOCTOU (Race Condition) khi kiểm tra giới hạn nhập sai OTP | Sử dụng Transaction (Pessimistic Lock) nguyên tử hóa cập nhật số lần nhập sai OTP trong `OTPRepository` |
| R1 | Tối ưu code theo chuẩn Golang (Code Style & Generics) | Tối ưu `auth_handler.go` với Generic function `bindAndValidate[T any]`. Tối ưu `auth_service.go` với `generateTokenPair`, loại bỏ `else` lồng ghép thừa thãi và tách method `handleFailedLogin`. |
| M5 | Tích hợp gửi email thực tế (UC-25, UC-26) | Thay thế mock bằng cấu hình Resend SMTP, cho phép gửi OTP và Link thật tới người dùng; bổ sung tài liệu vận hành. |
| B17 | Lỗ hổng Logic & Bảo mật trong Forgot/Reset/Change Password | Fix lỗi dò rỉ email (che giấu lỗi SMTP 500), chống spam Rate Limit cho Forgot Password, chống Brute-Force mật khẩu cũ, khắc phục Race Condition khi Reset bằng Atomic Query, và đảm bảo thứ tự thu hồi Session trước khi update DB. |
| R2 | Loại bỏ code dư thừa (Golang Performance) | Refactor `auth_service.go` và các Repository, loại bỏ Query thừa (TOCTOU, N+1), gỡ bỏ Transaction không cần thiết (Lock Contention) cho các hàm tăng biến đếm, gộp các Query cập nhật Database. |
---

## 🚧 Chưa làm / TODO

| UC | Tính năng | Ghi chú |
|----|----------|---------|
| UC-28 | Audit log cho Login/Register | Chưa có |
| UC-28 | Audit log cho Login/Register | Chưa có |
| UC-11~14 | User Profile (xem, sửa, avatar, đổi email) | Chưa có |
| UC-15~18 | Admin quản lý user | Chưa có |
| UC-38 | Unit & Integration Test | Đã viết unit test cho `pkg/jwt`, các module khác chưa có |

---

## 🔑 Bước tiếp theo đề xuất

1. **UC-07 Forgot Password** + **UC-08 Change Password** → hoàn thiện các luồng khôi phục và thay đổi mật khẩu.
2. **UC-28 Audit Logging** → ghi lại nhật ký đăng nhập và hành động của user để theo dõi bảo mật.
3. **UC-11~14 User Profile** → Quản lý hồ sơ, cập nhật thông tin và avatar.
