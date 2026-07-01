# UAM (User Access Management) API

Hệ thống quản lý người dùng, phân quyền nâng cao (RBAC, Permission-based) và bảo mật tài khoản viết bằng ngôn ngữ Go.

## 🚀 Tính năng nổi bật
- **Kiến trúc sạch (Clean Architecture)**: Tổ chức code theo 3 lớp chuẩn: Handler -> Service -> Repository.
- **Xác thực đa lớp**: JWT Access Token (HS256) và cơ chế xoay vòng Refresh Token (Token Rotation) chống chiếm quyền.
- **Bảo mật tối đa**:
  - Hash mật khẩu bằng `bcrypt`.
  - Hệ thống thu hồi token lập tức qua Redis Blacklist.
  - Chống Timing Attack dò tìm email, chống Race Condition (TOCTOU) khi nhập sai OTP/Login bằng Database Transaction.
  - Chống DoS bằng cách cắt ngắn độ dài IP và User-Agent trước khi lưu DB.
- **Gửi Email linh hoạt**: Tích hợp gửi mã OTP và link khôi phục mật khẩu thông qua Resend HTTP API (né triệt để mọi lỗi chặn cổng SMTP của PaaS như Render) hoặc Mailpit ở Local.
- **API Live Testing Console**: Giao diện UI tích hợp sẵn hỗ trợ chuyển đổi linh hoạt giữa Local và Production để test nhanh.

## 🛠️ Cấu hình chạy nhanh (Local)

### 1. Khởi động Docker Compose
Để bật cơ sở dữ liệu MySQL, Redis và hòm thư ảo Mailpit test local:
```bash
docker-compose up -d
```

### 2. Thiết lập Environment
Sao chép `.env.example` thành `.env` và tùy chỉnh các thông số kết nối.
```bash
cp .env.example .env
```

### 3. Chạy migrations
```bash
# Trên Windows dùng make.cmd
make.cmd migrate-up

# Trên macOS/Linux
make migrate-up
```

### 4. Khởi động server
```bash
go run ./cmd/server/main.go
```
Ứng dụng sẽ chạy tại cổng `http://localhost:8080`.

### 5. Test API qua UI Console
Mở file `ui_test/index.html` trực tiếp trên trình duyệt của bạn. Tùy chọn **local** hoặc **production** để test các luồng đăng ký, đăng nhập, quên mật khẩu cực kỳ trực quan.

## 🌍 URL Vận Hành (Production)
- **API Base URL**: `https://chiquocdev.me`
- **Health check**: `https://chiquocdev.me/health`
- **Ready check**: `https://chiquocdev.me/health/ready`
