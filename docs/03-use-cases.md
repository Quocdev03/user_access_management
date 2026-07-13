> [!IMPORTANT]
> **Nguồn sự thật:** `internal/router/*`, `internal/service/*`, config `.env`.
> Endpoint/path dưới đây khớp code hiện tại. Swagger UI (`/swagger/*`) **chỉ bật khi** `APP_ENV != production`.

# Use Cases — User Access Management

## Tổng quan

~40 UC theo nhóm nghiệp vụ. Auth token: **JWT** (access + refresh). Session active lưu **MySQL**; Redis dùng thu hồi token / rate-limit / pending email (xem `02-database-design.md`).

**Helper quan trọng:** `InvalidateUserSessions(userID)` = xóa sessions MySQL + set Redis revoke epoch → access JWT cũ chết ngay (không chờ hết hạn).

---

## Nhóm 1 — Xác thực & Phân quyền

### UC-01: Đăng ký — `POST /api/v1/auth/register`

1. Validate (binding + password policy).
2. Hash bcrypt → tạo user `inactive`, `email_verified=false`.
3. Gán role `user`.
4. Tạo OTP (lưu **hash**), gửi mail OTP plaintext chỉ qua email.
5. Rate limit theo IP.

### UC-02: Xác thực email OTP

| Method | Path |
|--------|------|
| POST | `/api/v1/auth/email/verify` |
| POST | `/api/v1/auth/email/resend` |

- So sánh OTP bằng hash + constant-time; max attempts / expiry từ config.
- Thành công → `email_verified=true`, `status=active`.
- Email không tồn tại: **không** lộ (trả lỗi generic giống OTP invalid).

### UC-03: Login — `POST /api/v1/auth/login`

- Đăng nhập bằng **email + password** (không username).
- Kiểm tra status (inactive / locked + `locked_until`).
- Sai mật khẩu → tăng `failed_login_attempts` (→ UC-06).
- `must_change_password` → `ERR_MUST_CHANGE_PASSWORD` (→ UC-40).
- Thành công: access + refresh JWT, session MySQL (có **`jti`**), device `FindOrCreate`, audit log.

### UC-04: Refresh — `POST /api/v1/auth/token/refresh`

- Rotation: session row update hash mới + jti mới.
- Refresh cũ → `revoked_rt:{hash}`; access jti cũ → blacklist.
- Reuse RT đã revoke → `InvalidateUserSessions` + lỗi token reuse.

### UC-05: Logout — `POST /api/v1/auth/logout`

- Blacklist access `jti` (Redis).
- Xóa session theo token hash (MySQL).

### UC-06: Lock sau N lần sai

- Ngưỡng / thời gian: `MAX_FAILED_ATTEMPTS`, `LOCK_DURATION` (mặc định 5 / 30m).
- Áp dụng login, change password (sai old), **force-change** (sai temp).

### UC-07 / UC-08: Forgot & Change password

| Method | Path |
|--------|------|
| POST | `/api/v1/auth/password/forgot` |
| POST | `/api/v1/auth/password/reset` |
| POST | `/api/v1/auth/password/change` |

- Forgot: luôn “success” nếu format OK (chống enumeration); token link hash trong MySQL.
- Reset/Change thành công → `InvalidateUserSessions`.
- Policy: `hash.ValidateNewPassword` (complexity + 72-byte bcrypt).

### UC-40: Force change password — `POST /api/v1/auth/password/force-change`

- Email + temp password + new password (public, rate-limit IP).
- Sai temp → cùng lockout như login.
- Thành công → clear `must_change_password`, invalidate sessions.

### UC-09 / UC-10: RBAC & Permission

- Auth middleware: Bearer JWT + blacklist jti + revoked epoch (fail-closed Redis).
- `RequireRole` / `PermissionMiddleware` (admin role có bypass permission check theo design hiện tại).
- Roles trong JWT claim — **đổi role phải invalidate session** để hết stale (→ UC-32).

### UC-32: Quản lý roles — `/api/v1/admin/roles/*`, gán role user

| Hành động | Path (rút gọn) |
|-----------|----------------|
| List/Create/Update/Delete role | `GET/POST/PUT/DELETE /admin/roles`… |
| Assign permissions | `PUT /admin/roles/{id}/permissions` |
| Assign/Remove user role | `POST/DELETE /admin/users/{id}/roles`… |

- Không xóa role hệ thống (`admin`/`moderator`/`user`) hoặc role đang được gán.
- Assign permission: validate mọi `permission_id` tồn tại.
- Gán/gỡ role user → **`InvalidateUserSessions`** (access JWT cũ chết).

---

## Nhóm 2 — Profile & Admin users

### UC-11 / UC-12 — Profile

| Method | Path |
|--------|------|
| GET | `/api/v1/users/me` |
| **PATCH** | `/api/v1/users/me` |

Không đổi username/email/password tại đây.

### UC-13: Đổi email

| Method | Path |
|--------|------|
| POST | `/api/v1/users/me/email/change-request` |
| POST | `/api/v1/users/me/email/verify` |
| POST | `/api/v1/users/me/email/resend` |

1. Verify current password → OTP old + new (hash DB).
2. Pending email mới trên Redis (`email_change_pending`) — **ngoài** MySQL tx.
3. Verify 2 OTP → update email → `InvalidateUserSessions`.

### UC-14: Avatar

| Method | Path |
|--------|------|
| POST | `/api/v1/users/me/avatar` |
| DELETE | `/api/v1/users/me/avatar` |

- ≤2MB; JPEG/PNG/WebP; static serve `/uploads`.

### UC-15–18: Admin users

| UC | Method | Path |
|----|--------|------|
| 15 Search | GET | `/admin/users` (page/`per_page`, filter) |
| Detail | GET | `/admin/users/{id}` |
| 16 Update | PUT | `/admin/users/{id}` |
| 17 Status | PATCH | `/admin/users/{id}/status` |
| 18 Reset PW | POST | `/admin/users/{id}/password/reset` |
| Notify | POST | `/admin/users/{id}/notify` |

- List: `LIMIT` = `per_page` (không dùng `page` làm limit).
- DTO admin dùng tag **`binding`** (Gin validate).
- Status `locked` **hoặc** `inactive` → `InvalidateUserSessions`.
- Update email unverified → invalidate sessions.
- Reset PW → temp mail + `must_change_password` + invalidate.
- Notify: HTML-escape subject/message.

---

## Nhóm 3 — Session / Device / Rate limit

### UC-19: Device tracking

- Login gọi `DeviceRepository.FindOrCreate`.
- `GET /api/v1/users/me/devices`.

### UC-20: Sessions

| Method | Path |
|--------|------|
| GET | `/users/me/sessions` |
| DELETE | `/users/me/sessions/{id}` |

- Revoke 1 session: blacklist **`jti`** access (nếu có) + xóa row MySQL.

### UC-21: Logout all — `POST /api/v1/auth/logout-all`

- `InvalidateUserSessions` (MySQL + epoch). **Không** “xóa session trong Redis” (session không nằm Redis).

### UC-22 / UC-23: Brute force & Rate limit

- Account lock (UC-06) + IP rate limit Redis (soft 429 + hard `ip_ban`).
- Config: `RATE_LIMIT_*`, per-route limits trên auth/user routes.
- `TRUSTED_PROXIES` nếu sau reverse proxy (tránh spoof `ClientIP`).

### UC-24: Password policy

- ≥8; hoa + thường + số + special; ≤72 bytes (bcrypt).
- `hash.ValidateNewPassword`.

---

## Nhóm 4 — Email

### UC-25 / UC-26

- Verify OTP email; forgot password = **link token** (không OTP).
- Dev: Mailpit / mock (mock **không** log body OTP).

### UC-27: Admin notify — `POST /admin/users/{id}/notify`

- Escape HTML; chỉ user không inactive.

---

## Nhóm 5 — Audit & Health

### UC-28 / UC-29 / UC-30

- Audit login + admin actions; export CSV `GET /admin/audit-logs/export`.

### UC-31: Health

| Path | Ý nghĩa |
|------|---------|
| `GET /health` | Liveness |
| `GET /health/ready` | MySQL + Redis ping |

---

## Nhóm 6 — Tooling

### UC-33: Swagger

- Generated: `docs/docs.go`, `swagger.json`, `swagger.yaml` (swag).
- Route mount **không** production.

### UC-34 / UC-35: Migration & seed

- `make migrate-up` / app auto-migrate lúc start (xem `main.go`).
- Seed roles/permissions + `admin_local` (`admin@localhost.local` / `LocalDev@ChangeMe1`).

### UC-36: Config

- Viper + `.env` / env vars. Quan trọng: JWT, DB, Redis, SMTP, security, `OTP_PEPPER`, `DB_TLS_SKIP_VERIFY`, `TRUSTED_PROXIES`.

### UC-37: Docker Compose

- Infra: mysql, redis, mailpit (app có thể chạy host). Ports DB/Redis chỉ local dev.

### UC-38: Tests

- Hiện có unit test tối thiểu (`pkg/jwt`, `pkg/hash`). Mở rộng khi cần.

### UC-39: Logging

- Zap structured; production release mode.

---

## Auth flow tóm tắt (thay file authentication-flow riêng)

```text
Register → OTP email (hash DB) → Verify → active
Login → JWT + session(jti) + device
Request → AuthMiddleware (jti blacklist + epoch) → RBAC/Permission → Handler
Logout → blacklist jti + delete session
LogoutAll / password / lock / role change → InvalidateUserSessions
Refresh → rotate RT + blacklist old access jti
Revoke one session → blacklist that jti + delete row
```
