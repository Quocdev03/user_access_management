# Thiết kế Database

## 1. Tổng quan

Hệ thống sử dụng **MySQL 8** làm cơ sở dữ liệu chính (bao gồm lưu trữ session phiên đăng nhập), **Redis** làm cache và lưu trữ danh sách thu hồi token (blacklist). Quản lý migration bằng **golang-migrate**.

---

## 2. Sơ đồ quan hệ (ERD)

```mermaid
erDiagram
    users ||--o{ user_roles : "có"
    roles ||--o{ user_roles : "được gán"
    roles ||--o{ role_permissions : "có"
    permissions ||--o{ role_permissions : "được gán"
    users ||--o{ sessions : "có"
    users ||--o{ devices : "đăng nhập từ"
    users ||--o{ audit_logs : "thực hiện"
    users ||--o{ otp_codes : "nhận"
    users ||--o{ password_reset_tokens : "yêu cầu"

    users {
        bigint id PK
        varchar username UK
        varchar email UK
        varchar password_hash
        varchar full_name
        varchar phone
        varchar avatar_url
        date date_of_birth
        enum status "active/inactive/locked"
        boolean email_verified
        int failed_login_attempts
        datetime locked_until
        datetime last_login_at
        datetime created_at
        datetime updated_at
    }

    roles {
        bigint id PK
        varchar name UK
        varchar description
        datetime created_at
        datetime updated_at
    }

    permissions {
        bigint id PK
        varchar name UK
        varchar description
        varchar resource
        varchar action
        datetime created_at
    }

    user_roles {
        bigint id PK
        bigint user_id FK
        bigint unsigned role_id FK
        timestamp assigned_at
    }

    role_permissions {
        bigint unsigned id PK
        bigint unsigned role_id FK
        bigint unsigned permission_id FK
        timestamp assigned_at
    }

    sessions {
        bigint unsigned id PK
        bigint unsigned user_id FK
        varchar token_hash UK
        varchar refresh_token_hash UK
        varchar ip_address
        varchar user_agent
        bigint unsigned device_id FK
        timestamp expires_at
        timestamp created_at
    }

    devices {
        bigint unsigned id PK
        bigint unsigned user_id FK
        varchar device_name
        varchar device_type
        varchar os
        varchar browser
        varchar ip_address
        timestamp last_active_at
        timestamp created_at
    }

    otp_codes {
        bigint unsigned id PK
        bigint unsigned user_id FK
        varchar code
        enum type "email_verification/forgot_password/change_email"
        int attempts
        boolean is_used
        timestamp expires_at
        timestamp created_at
    }

    password_reset_tokens {
        bigint unsigned id PK
        bigint unsigned user_id FK
        varchar token_hash UK
        boolean is_used
        timestamp expires_at
        timestamp created_at
    }

    audit_logs {
        bigint unsigned id PK
        bigint unsigned user_id FK
        varchar action
        varchar resource
        varchar resource_id
        varchar ip_address
        varchar user_agent
        json old_values
        json new_values
        enum status "success/failure"
        timestamp created_at
    }
```

---

## 3. Chi tiết từng bảng

### 3.1 `users` — Người dùng

| Cột | Kiểu | Ràng buộc | Mô tả |
|-----|------|-----------|-------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | Khóa chính |
| `username` | VARCHAR(50) | UNIQUE, NOT NULL | Tên đăng nhập |
| `email` | VARCHAR(255) | UNIQUE, NOT NULL | Email |
| `password_hash` | VARCHAR(255) | NOT NULL | Mật khẩu đã mã hóa (bcrypt) |
| `full_name` | VARCHAR(100) | | Họ tên đầy đủ |
| `phone` | VARCHAR(20) | NOT NULL | Số điện thoại |
| `avatar_url` | VARCHAR(500) | | Đường dẫn ảnh đại diện |
| `date_of_birth` | DATE | NOT NULL | Ngày tháng năm sinh |
| `status` | ENUM('active','inactive','locked') | DEFAULT 'inactive' | Trạng thái tài khoản |
| `email_verified` | BOOLEAN | DEFAULT FALSE | Đã xác thực email chưa |
| `failed_login_attempts` | TINYINT UNSIGNED | DEFAULT 0 | Số lần đăng nhập thất bại liên tiếp |
| `locked_until` | TIMESTAMP | NULL | Thời điểm mở khóa (nếu bị lock) |
| `last_login_at` | TIMESTAMP | NULL | Lần đăng nhập gần nhất |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Ngày tạo |
| `updated_at` | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | Ngày cập nhật |

### 3.2 `roles` — Vai trò

| Cột | Kiểu | Ràng buộc | Mô tả |
|-----|------|-----------|-------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | Khóa chính |
| `name` | VARCHAR(50) | UNIQUE, NOT NULL | Tên vai trò (`admin`, `moderator`, `user`) |
| `description` | VARCHAR(255) | | Mô tả vai trò |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Ngày tạo |
| `updated_at` | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | Ngày cập nhật |

### 3.3 `permissions` — Quyền hạn

| Cột | Kiểu | Ràng buộc | Mô tả |
|-----|------|-----------|-------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | Khóa chính |
| `name` | VARCHAR(100) | UNIQUE, NOT NULL | Tên quyền (VD: `users.read`, `users.update`) |
| `description` | VARCHAR(255) | | Mô tả quyền |
| `resource` | VARCHAR(50) | NOT NULL | Tài nguyên (VD: `users`, `roles`, `audit_logs`) |
| `action` | VARCHAR(50) | NOT NULL | Hành động (VD: `read`, `create`, `update`, `delete`) |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Ngày tạo |

### 3.4 `user_roles` — Gán vai trò cho người dùng

| Cột | Kiểu | Ràng buộc | Mô tả |
|-----|------|-----------|-------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | Khóa chính |
| `user_id` | BIGINT UNSIGNED | FK → users.id, NOT NULL | Người dùng |
| `role_id` | BIGINT UNSIGNED | FK → roles.id, NOT NULL | Vai trò |
| `assigned_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Ngày gán |

> **Ràng buộc**: UNIQUE(user_id, role_id) — mỗi user chỉ gán 1 lần cho mỗi role.

### 3.5 `role_permissions` — Gán quyền cho vai trò

| Cột | Kiểu | Ràng buộc | Mô tả |
|-----|------|-----------|-------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | Khóa chính |
| `role_id` | BIGINT UNSIGNED | FK → roles.id, NOT NULL | Vai trò |
| `permission_id` | BIGINT UNSIGNED | FK → permissions.id, NOT NULL | Quyền hạn |
| `assigned_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Ngày gán |

> **Ràng buộc**: UNIQUE(role_id, permission_id)

### 3.6 `sessions` — Phiên đăng nhập

| Cột | Kiểu | Ràng buộc | Mô tả |
|-----|------|-----------|-------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | Khóa chính |
| `user_id` | BIGINT UNSIGNED | FK → users.id, NOT NULL | Người dùng |
| `token_hash` | VARCHAR(255) | UNIQUE, NOT NULL | Hash của access token |
| `refresh_token_hash` | VARCHAR(255) | UNIQUE, NOT NULL | Hash của refresh token |
| `ip_address` | VARCHAR(45) | | Địa chỉ IP |
| `user_agent` | VARCHAR(500) | | Thông tin trình duyệt |
| `device_id` | BIGINT UNSIGNED | FK → devices.id, NULL | Thiết bị liên kết |
| `expires_at` | TIMESTAMP | NOT NULL | Thời điểm hết hạn |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Ngày tạo |

### 3.7 `devices` — Thiết bị đăng nhập

| Cột | Kiểu | Ràng buộc | Mô tả |
|-----|------|-----------|-------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | Khóa chính |
| `user_id` | BIGINT UNSIGNED | FK → users.id, NOT NULL | Người dùng |
| `device_name` | VARCHAR(100) | | Tên thiết bị |
| `device_type` | VARCHAR(50) | | Loại thiết bị (mobile, desktop, tablet) |
| `os` | VARCHAR(50) | | Hệ điều hành |
| `browser` | VARCHAR(50) | | Trình duyệt |
| `ip_address` | VARCHAR(45) | | Địa chỉ IP |
| `last_active_at` | TIMESTAMP | | Lần hoạt động gần nhất |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Ngày tạo |

### 3.8 `otp_codes` — Mã OTP

| Cột | Kiểu | Ràng buộc | Mô tả |
|-----|------|-----------|-------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | Khóa chính |
| `user_id` | BIGINT UNSIGNED | FK → users.id, NOT NULL | Người dùng |
| `code` | VARCHAR(10) | NOT NULL | Mã OTP (6 ký tự) |
| `type` | ENUM('email_verification', 'forgot_password', 'change_email') | NOT NULL | Loại OTP |
| `attempts` | TINYINT UNSIGNED | DEFAULT 0 | Số lần nhập sai |
| `is_used` | BOOLEAN | DEFAULT FALSE | Đã sử dụng chưa |
| `expires_at` | TIMESTAMP | NOT NULL | Thời điểm hết hạn |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Ngày tạo |

### 3.9 `password_reset_tokens` — Token đặt lại mật khẩu

| Cột | Kiểu | Ràng buộc | Mô tả |
|-----|------|-----------|-------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | Khóa chính |
| `user_id` | BIGINT UNSIGNED | FK → users.id, NOT NULL | Người dùng |
| `token_hash` | VARCHAR(255) | UNIQUE, NOT NULL | Hash của token |
| `is_used` | BOOLEAN | DEFAULT FALSE | Đã sử dụng chưa |
| `expires_at` | TIMESTAMP | NOT NULL | Thời điểm hết hạn |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Ngày tạo |

### 3.10 `audit_logs` — Nhật ký kiểm toán

| Cột | Kiểu | Ràng buộc | Mô tả |
|-----|------|-----------|-------|
| `id` | BIGINT UNSIGNED | PK, AUTO_INCREMENT | Khóa chính |
| `user_id` | BIGINT UNSIGNED | FK → users.id, NULL | Người thực hiện (NULL nếu system) |
| `action` | VARCHAR(50) | NOT NULL | Hành động (VD: `login`, `update_profile`, `lock_user`) |
| `resource` | VARCHAR(50) | | Tài nguyên bị tác động |
| `resource_id` | VARCHAR(50) | | ID tài nguyên bị tác động |
| `ip_address` | VARCHAR(45) | | Địa chỉ IP |
| `user_agent` | VARCHAR(500) | | Thông tin trình duyệt |
| `old_values` | JSON | NULL | Giá trị cũ (trước khi thay đổi) |
| `new_values` | JSON | NULL | Giá trị mới (sau khi thay đổi) |
| `status` | ENUM('success', 'failure') | NOT NULL | Kết quả |
| `created_at` | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Thời điểm |

---

## 4. Indexes

```sql
-- users (composite indexes thay vì single low-cardinality column)
CREATE INDEX idx_users_status_created ON users(status, created_at);
CREATE INDEX idx_users_email_verified_status ON users(email_verified, status);

-- user_roles
-- Lưu ý: idx trên user_id không cần vì UNIQUE KEY (user_id, role_id) đã cover prefix.
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);

-- role_permissions
-- Lưu ý: idx trên role_id không cần vì UNIQUE KEY (role_id, permission_id) đã cover prefix.

-- sessions
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- devices
CREATE INDEX idx_devices_user_id ON devices(user_id);

-- otp_codes
CREATE INDEX idx_otp_codes_user_id_type ON otp_codes(user_id, type);
CREATE INDEX idx_otp_codes_expires_at ON otp_codes(expires_at);

-- password_reset_tokens
CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);

-- audit_logs
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource, resource_id);
```

---

## 5. Redis — Dữ liệu lưu trữ

Redis không lưu dữ liệu lâu dài. Chỉ dùng cho việc lưu trữ tạm thời các token đã bị thu hồi (blacklist):

| Key Pattern | Kiểu | TTL | Mô tả |
|-------------|------|-----|-------|
| `blacklist:{jti}` | String | Bằng TTL còn lại của access token gốc | Access token ID (JTI) đã bị revoke khi người dùng logout |

---

## 6. Chiến lược Migration

Sử dụng **golang-migrate** với file SQL:

```

### Quy tắc migration

- Mỗi migration có file `.up.sql` (tạo) và `.down.sql` (rollback).
- Đánh số tuần tự: `000001`, `000002`, ...
- Tên file mô tả rõ hành động: `init_schema`, `add_xxx_column`, `seed_xxx`.
- **Không** sửa migration đã chạy trên môi trường khác. Tạo migration mới để thay đổi.

Hiện tại chúng ta gom toàn bộ thiết kế cơ sở vào 1 file duy nhất cho giai đoạn init:
```
migrations/
├── 000001_init_schema.up.sql
└── 000001_init_schema.down.sql
```
### Lưu ý thứ tự FK dependency

- `sessions` (000006) tạo trước `devices` (000007), nhưng `sessions.device_id` reference `devices.id`.
- Giải pháp: migration 000006 tạo bảng `sessions` **không có FK** cho `device_id`. Migration 000007 tạo bảng `devices` rồi **ALTER TABLE** thêm FK constraint cho `sessions.device_id → devices.id`.
- Rollback 000007 sẽ DROP FK trên `sessions` trước khi DROP TABLE `devices`.
