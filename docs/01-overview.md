> [!IMPORTANT]
> **LƯU Ý DÀNH CHO DEVELOPER (AI & HUMAN):**
> Các tài liệu thiết kế này mang tính chất là **KHUNG ĐỊNH HƯỚNG (Framework / Guidelines)**.
> KHÔNG ĐƯỢC áp dụng một cách rập khuôn, máy móc hoặc sao chép hoàn toàn 100%.
> Tùy thuộc vào bối cảnh thực tế của task, bạn phải linh hoạt tùy biến (ví dụ: dùng Atomic Query, Pessimistic Locking FOR UPDATE cho Concurrency, hoặc cấu trúc lại struct).

# Project Overview: User Access Management (UAM)

## 1. Giới thiệu dự án

**User Access Management (UAM)** là hệ thống backend được xây dựng bằng **Golang** nhằm quản lý tài khoản, xác thực, phân quyền, hồ sơ người dùng và monitoring cho các ứng dụng doanh nghiệp.

Dự án được thiết kế như một **Golang Backend Training Project** theo chuẩn doanh nghiệp, phù hợp cho việc học tập và áp dụng thực tế, bao gồm các tiêu chuẩn và quy trình phát triển chuyên nghiệp:

- Xây dựng RESTful API bằng **Gin Framework**.
- Áp dụng **JWT Authentication**, Refresh Token, RBAC và Permission-Based Authorization.
- Tổ chức source code theo **Clean Architecture**, Repository Pattern và Dependency Injection.
- Tích hợp **MySQL**, **Redis**, **golang-migrate** và **Docker Compose**.
- Xử lý validation, middleware, centralized error handling và **Swagger API Documentation**.
- Triển khai logging, configuration management, health check và monitoring theo chuẩn production.
- Viết unit test, integration test và API test.
- Làm việc theo quy trình task, sprint, code review và evidence.

---

## 2. Mục tiêu sản phẩm

Hệ thống giúp tổ chức thực hiện các nghiệp vụ sau:

- Cho phép user đăng ký, xác thực email/OTP và đăng nhập an toàn.
- Quản lý JWT access token, refresh token và session bằng Redis.
- Quản lý profile người dùng và avatar.
- Cho phép admin tìm kiếm, cập nhật, khóa/mở khóa và reset password cho user.
- Phân quyền bằng Role và Permission.
- Bảo vệ hệ thống trước brute force, rate limiting và password yếu.
- Gửi email/OTP cho các luồng xác thực.
- Ghi audit log cho login và các hành động quan trọng.
- Expose Health Check và Swagger API Documentation phục vụ vận hành.
- Cung cấp môi trường phát triển hoàn chỉnh bằng Docker Compose.

---

## 3. Đối tượng người dùng

### 3.1 End User

Hệ thống đáp ứng các nhu cầu:

- Đăng ký tài khoản.
- Xác thực Email/OTP.
- Đăng nhập và duy trì phiên làm việc.
- Quản lý thông tin cá nhân.
- Thay đổi avatar.
- Đổi email hoặc mật khẩu.
- Đăng xuất khỏi tất cả thiết bị.

### 3.2 Admin / Moderator

Hệ thống cung cấp các công cụ quản trị:

- Tìm kiếm và quản lý user.
- Cập nhật thông tin user.
- Enable, Disable hoặc Lock account.
- Reset password cho user.
- Gửi thông báo bắt buộc.
- Xem hoặc export audit logs.

---

## 4. Danh sách Tính năng & Use Cases (40 UC)

### Nhóm 1 — Authentication & Authorization

| Mã     | Tên Use Case                          |
| ------ | ------------------------------------- |
| UC-01  | User Registration                     |
| UC-02  | Email/OTP Verification                |
| UC-03  | User Login                            |
| UC-04  | Refresh Token                         |
| UC-05  | Logout                                |
| UC-06  | Lock Account After N Failed Attempts  |
| UC-07  | Forgot Password                       |
| UC-08  | Change Password                       |
| UC-09  | Role-Based Authorization (RBAC)       |
| UC-10  | Permission-Based Authorization        |
| UC-32  | Roles & Permissions Management        |
| UC-40  | Force Change Password                 |

### Nhóm 2 — User Profile & User Service

| Mã     | Tên Use Case                          |
| ------ | ------------------------------------- |
| UC-11  | View Profile                          |
| UC-12  | Update Profile                        |
| UC-13  | Change Email                          |
| UC-14  | Upload Avatar                         |
| UC-15  | Admin Search Users                    |
| UC-16  | Admin Update User Info                |
| UC-17  | Admin Enable / Disable / Lock User    |
| UC-18  | Admin Reset Password                  |

### Nhóm 3 — Security Extension

| Mã     | Tên Use Case                          |
| ------ | ------------------------------------- |
| UC-19  | Device Tracking                       |
| UC-20  | Session Management                    |
| UC-21  | Revoke All Sessions                   |
| UC-22  | Brute Force Protection                |
| UC-23  | Rate Limiting (Redis)                 |
| UC-24  | Password Policy                       |

### Nhóm 4 — Notification Service

| Mã     | Tên Use Case                          |
| ------ | ------------------------------------- |
| UC-25  | Send Registration Verification Email  |
| UC-26  | Send Forgot Password Email            |
| UC-27  | Admin Force Email Notifications       |

### Nhóm 5 — Audit & Monitoring

| Mã     | Tên Use Case                          |
| ------ | ------------------------------------- |
| UC-28  | Audit Login Events                    |
| UC-29  | Audit User Actions                    |
| UC-30  | Export Audit Logs                     |
| UC-31  | Health Check                          |

### Nhóm 6 — System & Developer Tools

| Mã     | Tên Use Case                          |
| ------ | ------------------------------------- |
| UC-33  | Swagger API Documentation             |
| UC-34  | Database Migration                    |
| UC-35  | Seed Data                             |
| UC-36  | Environment Configuration             |
| UC-37  | Docker Compose Setup                  |
| UC-38  | Unit & Integration Testing            |
| UC-39  | Structured Logging                    |

---

## 5. Công nghệ sử dụng

### Backend

| Công nghệ             | Mô tả                                      |
| ---------------------- | ------------------------------------------- |
| Golang 1.24+           | Ngôn ngữ chính                              |
| Gin Framework          | HTTP web framework                          |
| RESTful API            | Kiến trúc API                               |
| Clean Architecture     | Tổ chức source code                         |
| Repository Pattern     | Tầng truy xuất dữ liệu                     |
| Dependency Injection   | Quản lý dependencies                        |

### Database & Cache

| Công nghệ             | Mô tả                                      |
| ---------------------- | ------------------------------------------- |
| MySQL 8                | Cơ sở dữ liệu quan hệ                     |
| Redis                  | Cache & session store                       |
| golang-migrate         | Database migration tool                     |

### Authentication & Authorization

| Công nghệ                    | Mô tả                                |
| ----------------------------- | ------------------------------------- |
| JWT                           | Access token                          |
| Refresh Token                 | Gia hạn phiên đăng nhập              |
| RBAC                          | Phân quyền theo vai trò              |
| Permission-Based Authorization| Phân quyền theo permission            |
| bcrypt Password Hashing       | Mã hóa mật khẩu                     |

### Monitoring & Logging

| Công nghệ             | Mô tả                                      |
| ---------------------- | ------------------------------------------- |
| Health Check           | Kiểm tra trạng thái hệ thống               |
| Zap Logger             | Structured logging                          |

### Documentation & Development

| Công nghệ             | Mô tả                                      |
| ---------------------- | ------------------------------------------- |
| Swagger (Swaggo)       | API documentation tự động                   |
| Docker Compose         | Môi trường containerized                    |
| Air (Hot Reload)       | Live reload khi phát triển                  |
| Makefile               | Task runner                                 |
| GitHub Actions         | CI/CD (Optional)                            |
