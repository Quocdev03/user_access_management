# 🛡️ UAM (User Access Management) API

**UAM** là hệ thống backend hoàn chỉnh được xây dựng bằng **Golang** (Clean Architecture), chuyên quản lý tài khoản, xác thực đa lớp, phân quyền nâng cao (RBAC, Permission-based) và giám sát bảo mật tài khoản.

Dự án được thiết kế chuẩn doanh nghiệp (Production-Ready) với hiệu suất cao, khả năng chống Spam/Brute Force, hỗ trợ Live Testing Console và dễ dàng mở rộng cho các hệ thống thực tế.

---

## 🚀 Tính năng nổi bật

### 1. Kiến trúc & Công nghệ (Clean Architecture)
- Tổ chức mã nguồn chặt chẽ theo 3 lớp chuẩn: `Handler` -> `Service` -> `Repository`.
- **Dependency Injection**: Giảm thiểu độ kết dính, hỗ trợ tốt nhất cho việc mở rộng và Unit Testing.
- Hệ quản trị CSDL **MySQL 8** kết hợp với **Redis** cho caching và quản lý Session/Rate Limit.
- **Golang 1.24+**, sử dụng Framework **Gin HTTP** siêu tốc độ.

### 2. Xác thực đa lớp & Phân quyền nâng cao (RBAC)
- Xác thực bằng **JWT Access Token** (HS256) và cơ chế xoay vòng **Refresh Token** (Token Rotation).
- Phân quyền tinh tế: Quản lý cả **Role** (Vai trò) và **Permission** (Quyền hạn cụ thể).
- Cơ chế quản lý trạng thái tài khoản: *Active, Locked, Disabled, Pending.*

### 3. Bảo mật tối đa (Security First)
- Mã hoá mật khẩu một chiều với thuật toán `bcrypt`.
- **Thu hồi Token ngay lập tức** thông qua cơ chế Redis Blacklist khi có yêu cầu Logout hoặc Logout All Devices.
- **Anti-Spam & Brute Force Protection (Rate Limit - Hard Ban)**: Giới hạn request đến các endpoint nhạy cảm (Login, Register, OTP). Tự động chặn IP lên đến 15 phút nếu phát hiện spam.
- **Phòng chống các kỹ thuật tấn công:**
  - Chống *Timing Attack* khi dò tìm email.
  - Xử lý *Race Condition (TOCTOU)* bằng Database Transaction cấp phát `FOR UPDATE`.
  - Cắt ngắn payload cực đoan (IP, User-Agent) để chống DoS.

### 4. Các tiện ích & Công cụ phát triển
- Tích hợp HTTP API **Resend** (gửi OTP/Email) hoặc **Mailpit** (môi trường Local) vượt rào các hạn chế SMTP.
- **Giao diện API Live Testing**: Tích hợp sẵn tại `ui/index.html` cho phép test luồng API Local hoặc Production cực kỳ trực quan.
- Tự động hóa documentation với **Swagger**.
- Tooling đa dạng: `Makefile`, `Air` (Live Reload), `golang-migrate` (Quản lý Schema CSDL).

---

## 🛠️ Hướng dẫn cài đặt (Local)

### 1. Khởi động Docker Compose
Hệ thống sử dụng Docker để giả lập môi trường Local bao gồm MySQL, Redis và Mailpit:
```bash
docker-compose up -d
```

### 2. Thiết lập Environment
Sao chép `.env.example` thành `.env` và tùy chỉnh các thông số kết nối (đã được điền sẵn chuẩn cho môi trường Docker phía trên).
```bash
cp .env.example .env
```

### 3. Chạy Migrations (Tạo cấu trúc DB)
Đảm bảo bạn đã có sẵn thư mục `migrations/`. Chạy lệnh sau:
```bash
# Trên Windows
make.cmd migrate-up

# Trên macOS/Linux
make migrate-up
```

### 4. Khởi động API Server
Chạy ứng dụng với chế độ Live-reload (sử dụng Air) hoặc Go run:
```bash
# Dùng Air (nếu đã cài đặt)
air

# Hoặc dùng Go thuần
go run ./cmd/server/main.go
```
Ứng dụng sẽ mặc định chạy tại: `http://localhost:8080`.

### 5. Giao diện Test trực quan (UI Console)
Thay vì sử dụng Postman, bạn chỉ cần mở file `ui/index.html` trực tiếp trên trình duyệt. 
Giao diện cho phép tuỳ chọn gọi API đến **Local** hoặc **Production** để test mượt mà các luồng: *Đăng ký, Xác minh OTP, Đăng nhập và các luồng bảo mật khác.*

---

## 🌍 URL Vận Hành (Production)

Trong trường hợp ứng dụng đã được deploy lên môi trường Production, bạn có thể kiểm tra qua các Endpoints:
- **API Base URL**: `https://chiquocdev.me`
- **Swagger Documentation**: `https://chiquocdev.me/swagger/index.html` (Nếu được enable)
- **Health check**: `https://chiquocdev.me/health` (Light ping)
- **Ready check**: `https://chiquocdev.me/health/ready` (Deep ping: Checks DB & Redis)

---

## 📂 Cấu trúc dự án cơ bản

```text
├── cmd
│   └── server          # Entry point của ứng dụng (main.go)
├── docs                # Các tài liệu phân tích nghiệp vụ, kiến trúc, DB
├── internal
│   ├── config          # Cấu hình biến môi trường
│   ├── dto             # Data Transfer Objects
│   ├── entity          # Models tương tác trực tiếp với Database
│   ├── handler         # Tầng giao tiếp HTTP Request/Response
│   ├── middleware      # Auth, Rate Limit, CORS, Recovery
│   ├── repository      # Tầng tương tác với CSDL (MySQL, Redis)
│   ├── router          # Khai báo các Routes
│   └── service         # Tầng chứa lõi Logic nghiệp vụ
├── migrations          # Chứa các file SQL để chạy migration
├── pkg                 # Các thư viện/helper dùng chung (Hash, Logger, SMTP...)
├── ui                  # Mã nguồn UI API Tester
└── ...
```

---

*Dự án UAM được xây dựng với mục tiêu cung cấp giải pháp xác thực và phân quyền lõi bền vững, có thể tái sử dụng dễ dàng cho mọi ứng dụng Microservices hoặc Monolith trong tương lai.*
