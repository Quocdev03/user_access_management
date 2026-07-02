# Thiết kế API

## 1. Quy ước chung

### Base URL

```
http://localhost:8080/api/v1
```

### Định dạng Response thành công

```json
{
  "success": true,
  "message": "Thao tác thành công",
  "data": { ... }
}
```

### Định dạng Response có phân trang

```json
{
  "success": true,
  "message": "Lấy danh sách thành công",
  "data": [ ... ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

### Định dạng Response lỗi

```json
{
  "success": false,
  "message": "Mô tả lỗi",
  "error": {
    "code": "VALIDATION_ERROR",
    "details": [
      {
        "field": "email",
        "message": "Email không hợp lệ"
      }
    ]
  }
}
```

### Headers

| Header | Giá trị | Mô tả |
|--------|---------|-------|
| `Content-Type` | `application/json` | Định dạng body |
| `Authorization` | `Bearer {access_token}` | Token xác thực (endpoints cần auth) |

### Phân trang

Các endpoint trả danh sách hỗ trợ query params:

| Param | Mặc định | Mô tả |
|-------|----------|-------|
| `page` | 1 | Trang hiện tại |
| `per_page` | 20 | Số bản ghi/trang (tối đa 100) |
| `sort_by` | `created_at` | Cột sắp xếp |
| `sort_order` | `desc` | Thứ tự: `asc` hoặc `desc` |

---

## 2. Danh sách mã lỗi

| Mã lỗi | HTTP Status | Mô tả |
|---------|-------------|-------|
| `VALIDATION_ERROR` | 400 | Dữ liệu đầu vào không hợp lệ |
| `INVALID_CREDENTIALS` | 401 | Sai tên đăng nhập hoặc mật khẩu |
| `TOKEN_EXPIRED` | 401 | Token hết hạn |
| `TOKEN_INVALID` | 401 | Token không hợp lệ |
| `UNAUTHORIZED` | 401 | Chưa đăng nhập |
| `FORBIDDEN` | 403 | Không có quyền truy cập |
| `NOT_FOUND` | 404 | Không tìm thấy tài nguyên |
| `CONFLICT` | 409 | Dữ liệu bị trùng (email, username) |
| `ACCOUNT_LOCKED` | 423 | Tài khoản bị khóa |
| `RATE_LIMITED` | 429 | Vượt giới hạn request |
| `OTP_EXPIRED` | 400 | Mã OTP hết hạn |
| `OTP_INVALID` | 400 | Mã OTP không đúng |
| `OTP_MAX_ATTEMPTS` | 400 | Nhập sai OTP quá số lần cho phép |
| `INTERNAL_ERROR` | 500 | Lỗi hệ thống |

---

## 3. Danh sách Endpoints

### 3.1 Xác thực (Authentication)

| Phương thức | Endpoint | Mô tả | Auth |
|-------------|----------|-------|------|
| POST | `/auth/register` | Đăng ký tài khoản | Không |
| POST | `/auth/verify-email` | Xác thực email bằng OTP | Không |
| POST | `/auth/login` | Đăng nhập | Không |
| POST | `/auth/refresh-token` | Làm mới access token | Không (dùng refresh token) |
| POST | `/auth/logout` | Đăng xuất | Có |
| POST | `/auth/logout-all` | Đăng xuất tất cả thiết bị | Có |
| POST | `/auth/forgot-password` | Yêu cầu đặt lại mật khẩu | Không |
| POST | `/auth/reset-password` | Đặt lại mật khẩu bằng token | Không |
| POST | `/auth/change-password` | Đổi mật khẩu (khi đã đăng nhập) | Có |

### 3.2 Hồ sơ người dùng (User Profile)

| Phương thức | Endpoint | Mô tả | Auth |
|-------------|----------|-------|------|
| GET | `/users/me` | Xem thông tin cá nhân | Có |
| PUT | `/users/me` | Cập nhật thông tin cá nhân | Có |
| PUT | `/users/me/email` | Đổi email (gửi OTP xác thực) | Có |
| POST | `/users/me/email/verify` | Xác thực email mới bằng OTP | Có |
| POST | `/users/me/avatar` | Upload ảnh đại diện | Có |
| DELETE | `/users/me/avatar` | Xóa ảnh đại diện | Có |

### 3.3 Phiên đăng nhập & Thiết bị (Session & Device)

| Phương thức | Endpoint | Mô tả | Auth |
|-------------|----------|-------|------|
| GET | `/users/me/sessions` | Xem danh sách phiên đăng nhập | Có |
| DELETE | `/users/me/sessions/{id}` | Hủy một phiên đăng nhập | Có |
| GET | `/users/me/devices` | Xem danh sách thiết bị | Có |

### 3.4 Quản trị (Admin)

| Phương thức | Endpoint | Mô tả | Auth | Quyền |
|-------------|----------|-------|------|-------|
| GET | `/admin/users` | Tìm kiếm danh sách người dùng | Có | `users.read` |
| GET | `/admin/users/{id}` | Xem chi tiết một người dùng | Có | `users.read` |
| PUT | `/admin/users/{id}` | Cập nhật thông tin người dùng | Có | `users.update` |
| PATCH | `/admin/users/{id}/status` | Thay đổi trạng thái (active/inactive/locked) | Có | `users.update` |
| POST | `/admin/users/{id}/reset-password` | Đặt lại mật khẩu cho người dùng | Có | `users.update` |
| POST | `/admin/users/{id}/notify` | Gửi thông báo bắt buộc | Có | `users.notify` |

### 3.5 Vai trò & Quyền hạn (Roles & Permissions)

| Phương thức | Endpoint | Mô tả | Auth | Quyền |
|-------------|----------|-------|------|-------|
| GET | `/admin/roles` | Danh sách vai trò | Có | `roles.read` |
| POST | `/admin/roles` | Tạo vai trò mới | Có | `roles.create` |
| PUT | `/admin/roles/{id}` | Cập nhật vai trò | Có | `roles.update` |
| DELETE | `/admin/roles/{id}` | Xóa vai trò | Có | `roles.delete` |
| GET | `/admin/roles/{id}/permissions` | Xem quyền của vai trò | Có | `roles.read` |
| PUT | `/admin/roles/{id}/permissions` | Gán quyền cho vai trò | Có | `roles.update` |
| POST | `/admin/users/{id}/roles` | Gán vai trò cho người dùng | Có | `users.update` |
| DELETE | `/admin/users/{id}/roles/{roleId}` | Gỡ vai trò khỏi người dùng | Có | `users.update` |

### 3.6 Nhật ký kiểm toán (Audit Logs)

| Phương thức | Endpoint | Mô tả | Auth | Quyền |
|-------------|----------|-------|------|-------|
| GET | `/admin/audit-logs` | Xem nhật ký kiểm toán | Có | `audit_logs.read` |
| GET | `/admin/audit-logs/export` | Export nhật ký (CSV) | Có | `audit_logs.export` |

**Query params cho audit-logs:**

| Param | Mô tả |
|-------|-------|
| `user_id` | Lọc theo người dùng |
| `action` | Lọc theo hành động |
| `resource` | Lọc theo tài nguyên |
| `status` | Lọc theo kết quả (`success`/`failure`) |
| `from` | Từ ngày (ISO 8601) |
| `to` | Đến ngày (ISO 8601) |

### 3.7 Hệ thống (System)

| Phương thức | Endpoint | Mô tả | Auth |
|-------------|----------|-------|------|
| GET | `/health` | Kiểm tra trạng thái hệ thống | Không |
| GET | `/health/ready` | Kiểm tra kết nối database, Redis | Không |

---

## 4. Chi tiết API — Xác thực

### POST `/auth/register`

Đăng ký tài khoản mới. Sau khi đăng ký, hệ thống gửi OTP qua email để xác thực.

**Request:**
```json
{
  "username": "nguyenvana",
  "email": "nguyenvana@example.com",
  "password": "MatKhau@123",
  "full_name": "Nguyễn Văn A"
}
```

**Response (201):**
```json
{
  "success": true,
  "message": "Đăng ký thành công. Vui lòng kiểm tra email để xác thực tài khoản.",
  "data": {
    "user_id": 1,
    "username": "nguyenvana",
    "email": "nguyenvana@example.com"
  }
}
```

### POST `/auth/verify-email`

Xác thực email bằng mã OTP.

**Request:**
```json
{
  "email": "nguyenvana@example.com",
  "otp": "123456"
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "Xác thực email thành công. Bạn có thể đăng nhập."
}
```

### POST `/auth/login`

Đăng nhập và nhận JWT access token + refresh token.

**Request:**
```json
{
  "email": "nguyenvana@example.com",
  "password": "MatKhau@123"
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "Đăng nhập thành công",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "dGhpcyBpcyBhIHJlZnJl...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

### POST `/auth/refresh-token`

Làm mới access token bằng refresh token.

**Request:**
```json
{
  "refresh_token": "dGhpcyBpcyBhIHJlZnJl..."
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "Làm mới token thành công",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "bmV3IHJlZnJlc2ggdG9r...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

### POST `/auth/forgot-password`

Gửi email chứa link/token đặt lại mật khẩu.

**Request:**
```json
{
  "email": "nguyenvana@example.com"
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "Nếu email tồn tại, chúng tôi đã gửi hướng dẫn đặt lại mật khẩu."
}
```

> **Lưu ý**: Luôn trả thành công dù email có tồn tại hay không (chống enumeration).

### POST `/auth/reset-password`

Đặt lại mật khẩu bằng token từ email.

**Request:**
```json
{
  "token": "abc123def456...",
  "new_password": "MatKhauMoi@456"
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "Đặt lại mật khẩu thành công. Vui lòng đăng nhập lại."
}
```
