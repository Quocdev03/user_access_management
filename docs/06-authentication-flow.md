> [!IMPORTANT]
> **LƯU Ý DÀNH CHO DEVELOPER (AI & HUMAN):**
> Các tài liệu thiết kế này mang tính chất là **KHUNG ĐỊNH HƯỚNG (Framework / Guidelines)**.
> KHÔNG ĐƯỢC áp dụng một cách rập khuôn, máy móc hoặc sao chép hoàn toàn 100%.
> Tùy thuộc vào bối cảnh thực tế của task, bạn phải linh hoạt tùy biến (ví dụ: dùng Atomic Query, Pessimistic Locking FOR UPDATE cho Concurrency, hoặc cấu trúc lại struct).

# Luồng xác thực & Phân quyền

## 1. Tổng quan

Hệ thống sử dụng **JWT (JSON Web Token)** cho xác thực, kết hợp **Refresh Token Rotation** để duy trì phiên đăng nhập an toàn. Phân quyền theo mô hình **RBAC (Role-Based Access Control)** kết hợp **Permission-Based Authorization**.

---

## 2. Luồng đăng ký & Xác thực email

```mermaid
sequenceDiagram
    participant U as Người dùng
    participant S as Server
    participant DB as MySQL
    participant R as Redis
    participant M as Mail Service

    U->>S: POST /auth/register (username, email, password)
    S->>S: Validate dữ liệu
    S->>DB: Kiểm tra email/username trùng
    DB-->>S: Không trùng
    S->>S: Hash mật khẩu (bcrypt)
    S->>DB: Tạo user (status=inactive)
    S->>DB: Gán role "user"
    S->>S: Sinh OTP 6 số
    S->>R: Lưu OTP (TTL 5 phút)
    S->>DB: Lưu OTP vào bảng otp_codes
    S->>M: Gửi email chứa OTP
    S-->>U: 201 - Đăng ký thành công

    Note over U,M: Người dùng kiểm tra email

    U->>S: POST /auth/verify-email (email, otp)
    S->>R: Kiểm tra OTP
    R-->>S: OTP hợp lệ
    S->>DB: Cập nhật email_verified=true, status=active
    S->>R: Xóa OTP
    S-->>U: 200 - Xác thực thành công
```

---

## 3. Luồng đăng nhập

```mermaid
sequenceDiagram
    participant U as Người dùng
    participant S as Server
    participant DB as MySQL
    participant R as Redis

    U->>S: POST /auth/login (email, password)
    S->>DB: Tìm user theo email
    DB-->>S: User tồn tại

    alt Tài khoản bị khóa
        S-->>U: 423 - ACCOUNT_LOCKED
    end

    alt Email chưa xác thực
        S-->>U: 401 - Yêu cầu xác thực email
    end

    S->>S: So sánh mật khẩu (bcrypt)

    alt Mật khẩu sai
        S->>DB: Tăng failed_login_attempts
        alt Đạt ngưỡng (5 lần)
            S->>DB: Khóa tài khoản (locked_until = now + 30 phút)
        end
        S-->>U: 401 - INVALID_CREDENTIALS
    end

    S->>S: Sinh Access Token (JWT, 15 phút)
    S->>S: Sinh Refresh Token (random, 7 ngày)
    S->>R: Lưu session (access token hash + refresh token hash)
    S->>DB: Lưu session, cập nhật last_login_at, reset failed_attempts
    S->>DB: Ghi nhận thiết bị (device tracking)
    S->>DB: Ghi audit log
    S-->>U: 200 - {access_token, refresh_token, expires_in}
```

---

## 4. Cấu trúc JWT Access Token

### Header

```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

### Payload (Claims)

```json
{
  "sub": 1,
  "username": "nguyenvana",
  "email": "nguyenvana@example.com",
  "roles": ["user"],
  "iat": 1700000000,
  "exp": 1700000900
}
```

| Claim | Mô tả |
|-------|-------|
| `sub` | User ID (uint64) |
| `username` | Tên đăng nhập |
| `email` | Email |
| `roles` | Danh sách vai trò |
| `iat` | Thời điểm tạo (Unix timestamp) |
| `exp` | Thời điểm hết hạn (Unix timestamp) |

### Cấu hình token

| Loại | Thời hạn | Nơi lưu |
|------|----------|---------|
| Access Token | 15 phút | Client (header Authorization) |
| Refresh Token | 7 ngày | Client (httpOnly cookie hoặc body) |

---

## 5. Luồng Refresh Token

```mermaid
sequenceDiagram
    participant U as Client
    participant S as Server
    participant DB as MySQL

    U->>S: POST /auth/refresh-token (refresh_token)
    S->>S: Parse & Validate JWT signature
    S->>DB: Tìm session theo hash của refresh_token
    DB-->>S: Session hợp lệ, chưa hết hạn

    S->>S: Sinh Access Token mới
    S->>S: Sinh Refresh Token mới (rotation)
    S->>DB: UPDATE session hiện tại với các hash mới và expires_at mới
    S-->>U: 200 - {access_token, refresh_token}
```

### Refresh Token Rotation

- Mỗi lần refresh → tạo access & refresh token mới, cập nhật trực tiếp session cũ trong MySQL bằng query UPDATE (chuyển đổi nguyên tử).
- Nếu refresh token bị sử dụng trái phép hoặc bị thu hồi (không tìm thấy session nào khớp), server trả về lỗi `ERR_REFRESH_INVALID` (401).
- Đảm bảo tính toàn vẹn dữ liệu (ACID) bằng cơ chế cập nhật trực tiếp thay vì xóa rồi tạo mới.

---

## 6. Luồng đăng xuất

```mermaid
sequenceDiagram
    participant U as Client
    participant S as Server
    participant R as Redis

    Note over U,R: Đăng xuất phiên hiện tại
    U->>S: POST /auth/logout (Authorization: Bearer {token})
    S->>R: Thêm access token vào blacklist
    S->>R: Xóa session tương ứng
    S-->>U: 200 - Đăng xuất thành công

    Note over U,R: Đăng xuất tất cả thiết bị
    U->>S: POST /auth/logout-all (Authorization: Bearer {token})
    S->>R: Lấy tất cả session của user
    S->>R: Blacklist tất cả token + xóa tất cả session
    S-->>U: 200 - Đã đăng xuất tất cả thiết bị
```

---

## 7. Luồng quên mật khẩu

```mermaid
sequenceDiagram
    participant U as Người dùng
    participant S as Server
    participant DB as MySQL
    participant M as Mail Service

    U->>S: POST /auth/forgot-password (email)
    S->>DB: Tìm user theo email

    alt Email không tồn tại
        S-->>U: 200 - Nếu email tồn tại, đã gửi hướng dẫn
        Note over S: Trả thành công dù không tìm thấy (chống enumeration)
    end

    S->>S: Sinh reset token (random + hash)
    S->>DB: Lưu token hash (TTL 1 giờ)
    S->>M: Gửi email chứa link reset
    S-->>U: 200 - Nếu email tồn tại, đã gửi hướng dẫn

    Note over U,M: Người dùng click link trong email

    U->>S: POST /auth/reset-password (token, new_password)
    S->>DB: Kiểm tra token (hợp lệ, chưa hết hạn, chưa dùng)
    S->>S: Validate mật khẩu mới (password policy)
    S->>S: Hash mật khẩu mới (bcrypt)
    S->>DB: Cập nhật mật khẩu, đánh dấu token đã dùng
    S->>DB: Revoke tất cả session
    S-->>U: 200 - Đặt lại mật khẩu thành công
```

---

## 8. Mô hình phân quyền RBAC

### Quan hệ

```mermaid
graph LR
    U[User] -->|có nhiều| UR[User Roles]
    UR -->|thuộc về| R[Role]
    R -->|có nhiều| RP[Role Permissions]
    RP -->|thuộc về| P[Permission]
```

**User → Role → Permission**: Người dùng được gán vai trò, mỗi vai trò có danh sách quyền cụ thể.

### Vai trò mặc định

| Vai trò | Mô tả | Quyền chính |
|---------|-------|-------------|
| `admin` | Quản trị viên cao nhất | Toàn quyền |
| `moderator` | Quản lý nội dung | Xem/sửa user, xem audit logs |
| `user` | Người dùng thường | Chỉ quản lý thông tin cá nhân |

### Ma trận quyền (Permission Matrix)

| Permission | Admin | Moderator | User |
|-----------|-------|-----------|------|
| `users.read` (xem danh sách user) | ✅ | ✅ | ❌ |
| `users.create` (tạo user) | ✅ | ❌ | ❌ |
| `users.update` (sửa user) | ✅ | ✅ | ❌ |
| `users.delete` (xóa user) | ✅ | ❌ | ❌ |
| `users.notify` (gửi thông báo) | ✅ | ❌ | ❌ |
| `roles.read` (xem vai trò) | ✅ | ✅ | ❌ |
| `roles.create` (tạo vai trò) | ✅ | ❌ | ❌ |
| `roles.update` (sửa vai trò) | ✅ | ❌ | ❌ |
| `roles.delete` (xóa vai trò) | ✅ | ❌ | ❌ |
| `audit_logs.read` (xem logs) | ✅ | ✅ | ❌ |
| `audit_logs.export` (export logs) | ✅ | ❌ | ❌ |
| `profile.read` (xem hồ sơ cá nhân) | ✅ | ✅ | ✅ |
| `profile.update` (sửa hồ sơ cá nhân) | ✅ | ✅ | ✅ |

### Luồng kiểm tra quyền (Middleware)

```mermaid
flowchart TD
    A[Request đến] --> B{Có Authorization header?}
    B -->|Không| C[401 Unauthorized]
    B -->|Có| D[Parse JWT Token]
    D --> E{Token hợp lệ?}
    E -->|Không| F[401 Token Invalid/Expired]
    E -->|Có| G{Token bị blacklist?}
    G -->|Có| H[401 Token Revoked]
    G -->|Không| I[Inject user info vào context]
    I --> J{Endpoint yêu cầu role?}
    J -->|Không| K[Tiếp tục → Handler]
    J -->|Có| L{User có role yêu cầu?}
    L -->|Không| M[403 Forbidden]
    L -->|Có| K
```

---

## 9. Bảo mật bổ sung

### Chống Brute Force

| Cơ chế | Chi tiết |
|--------|---------|
| Khóa tài khoản | Sau 10 lần sai → khóa 15 phút |
| Rate Limiting | Login: 15 req/phút/IP (Ban IP 15p nếu > 50 req) |
| Rate Limit Cứng | Cho phép sai 3-5 lần (Token Bucket) thay vì block ngay lần đầu |

### Password Policy

| Quy tắc | Giá trị |
|---------|---------|
| Độ dài tối thiểu | 8 ký tự |
| Chữ hoa | Ít nhất 1 |
| Chữ thường | Ít nhất 1 |
| Chữ số | Ít nhất 1 |
| Ký tự đặc biệt | Ít nhất 1 (`@#$%^&*!`) |
| Không trùng mật khẩu cũ | Có |

### Token Security

| Biện pháp | Mô tả |
|-----------|-------|
| Hash token trước khi lưu DB | Dùng SHA-256, không lưu token plaintext |
| Refresh Token Rotation | Token cũ bị vô hiệu ngay sau khi dùng bằng câu lệnh UPDATE |
| Blacklist khi logout | Token bị blacklist trong Redis đến khi hết hạn |
| Redis Fail-Open | Nếu Redis sập, hệ thống ghi log warning/error và bỏ qua kiểm tra blacklist (vẫn cho phép request đi qua nếu chữ ký JWT hợp lệ) |
