> [!IMPORTANT]
> **LƯU Ý DÀNH CHO DEVELOPER (AI & HUMAN):**
> Các tài liệu thiết kế này mang tính chất là **KHUNG ĐỊNH HƯỚNG (Framework / Guidelines)**.
> KHÔNG ĐƯỢC áp dụng một cách rập khuôn, máy móc hoặc sao chép hoàn toàn 100%.
> Tùy thuộc vào bối cảnh thực tế của task, bạn phải linh hoạt tùy biến (ví dụ: dùng Atomic Query, Pessimistic Locking FOR UPDATE cho Concurrency, hoặc cấu trúc lại struct).

# Hướng dẫn cài đặt & Chạy dự án

## 1. Yêu cầu hệ thống

| Công cụ | Phiên bản | Mô tả |
|---------|-----------|-------|
| Go | latest | Ngôn ngữ lập trình |
| MySQL | latest | Cơ sở dữ liệu |
| Redis | latest | Cache & session store |
| Docker | latest | Container runtime |
| Docker Compose | latest | Quản lý multi-container |
| Make | latest | Task runner |
| Git | latest | Quản lý mã nguồn |

---

## 2. Cài đặt nhanh với Docker Compose (Khuyến nghị)

### Bước 1: Clone dự án

```bash
git clone <repository-url>
cd user_access_management
```

### Bước 2: Tạo file cấu hình

```bash
cp .env.example .env
```

Chỉnh sửa file `.env` theo môi trường:

```env
# Server
APP_ENV=development
APP_PORT=8080

# Database
DB_HOST=mysql
DB_PORT=3306
DB_NAME=uam_db
DB_USER=uam_user
DB_PASSWORD=uam_password
DB_ROOT_PASSWORD=root_password

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT
JWT_SECRET=your-secret-key-change-this
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# Mail (SMTP) - Mặc định dùng Mailpit (127.0.0.1:1025) cho dev
# Để dùng Resend gửi mail thật, hãy thay thế bằng:
# SMTP_HOST=smtp.resend.com
# SMTP_PORT=587
# SMTP_USER=resend
# SMTP_PASSWORD=YOUR_API_KEY
# SMTP_FROM=noreply@YOUR_DOMAIN.com
SMTP_HOST=127.0.0.1
SMTP_PORT=1025
SMTP_USER=
SMTP_PASSWORD=
SMTP_FROM=noreply@localhost

# Rate Limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m

# Account Lock
MAX_FAILED_ATTEMPTS=5
LOCK_DURATION=30m

# OTP
OTP_EXPIRY=5m
OTP_MAX_ATTEMPTS=5
```

### Bước 3: Khởi chạy

```bash
# Khởi chạy hạ tầng (MySQL, Redis, Mailpit, v.v) bằng Docker
docker compose up -d

# Cài đặt các công cụ phát triển (chỉ cần chạy lần đầu)
go install github.com/air-verse/air@latest
go install github.com/swaggo/swag/cmd/swag@latest
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Chạy migration để nạp cấu trúc database
make migrate-up
# Lưu ý: Nếu dùng Windows (CMD/PowerShell), thay chữ `make` bằng `.\make.cmd`
# Ví dụ: .\make.cmd migrate-up

# Chạy server ở máy Host (hỗ trợ Hot-reload)
make dev
```

> **Lưu ý CSDL:** Adminer và Redis-Commander đã được gỡ khỏi cấu hình mặc định để tối ưu tài nguyên. Bạn nên sử dụng các Desktop App như **DBeaver** (MySQL) và **Redis Insight** (Redis) để quản lý trực tiếp qua port 3306 / 6379.

### Bước 4: Kiểm tra

```bash
# Kiểm tra health check
curl http://localhost:8080/health

# Xem Swagger UI
# Mở trình duyệt: http://localhost:8080/swagger/index.html
```

> **Tài khoản Super Admin:** Sau khi chạy `migrate-up` thành công, hệ thống đã tự động tạo tài khoản có quyền Admin cao nhất để bạn test ngay.
> - Username: `admin_quocdev`
> - Mật khẩu: `Quocdev@2026`

---

## 3. Docker Compose — Chi tiết

### Cấu trúc file `docker-compose.yml`

```yaml
services:
  mysql:
    image: mysql:8
    ports:
      - "3306:3306"
    environment:
      MYSQL_ROOT_PASSWORD: ${DB_ROOT_PASSWORD}
      MYSQL_DATABASE: ${DB_NAME}
      MYSQL_USER: ${DB_USER}
      MYSQL_PASSWORD: ${DB_PASSWORD}
    volumes:
      - mysql_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  mailpit:
    image: axllent/mailpit
    ports:
FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Run stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/.env.example .env
EXPOSE 8080
CMD ["./server"]
```

### Các lệnh Docker Compose thường dùng

```bash
# Khởi chạy
docker compose up -d

# Dừng
docker compose down

# Dừng và xóa dữ liệu (volumes)
docker compose down -v

# Build lại image
docker compose up -d --build

# Xem logs realtime
docker compose logs -f app

# Truy cập MySQL CLI
docker compose exec mysql mysql -u root -p uam_db

# Truy cập Redis CLI
docker compose exec redis redis-cli
```

---

## 4. Cài đặt thủ công (Không dùng Docker)

### Bước 1: Cài đặt Go

Tải và cài đặt Go từ [https://go.dev/dl/](https://go.dev/dl/)

```bash
go version   # Kiểm tra: bản mới nhất (latest)
```

### Bước 2: Cài đặt MySQL

Cài MySQL bản mới nhất (latest) và tạo database:

```sql
CREATE DATABASE uam_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'uam_user'@'localhost' IDENTIFIED BY 'uam_password';
GRANT ALL PRIVILEGES ON uam_db.* TO 'uam_user'@'localhost';
FLUSH PRIVILEGES;
```

### Bước 3: Cài đặt Redis

Cài Redis bản mới nhất (latest) và khởi chạy:

```bash
redis-server
redis-cli ping   # Kiểm tra: PONG
```

### Bước 4: Cấu hình & Chạy

```bash
# Clone dự án
git clone <repository-url>
cd user_access_management

# Tạo file cấu hình
cp .env.example .env
# Sửa .env: đổi DB_HOST=localhost, REDIS_HOST=localhost

# Cài đặt dependencies
go mod download

# Cài công cụ phát triển
go install github.com/air-verse/air@latest           # Hot reload
go install github.com/swaggo/swag/cmd/swag@latest    # Swagger
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Chạy migration
make migrate-up

# Tạo dữ liệu mẫu
make seed

# Sinh Swagger docs
make swagger

# Chạy server (hot reload)
make dev
# Hoặc: air
```

---

## 5. Makefile — Các lệnh thường dùng

```makefile
# Chạy server development (hot reload)
make dev

# Chạy server production
make run

# Chạy migration
make migrate-up
make migrate-down
make migrate-create name=create_xxx_table

# Tạo dữ liệu mẫu
make seed

# Sinh Swagger docs
make swagger

# Chạy test
make test
make test-coverage

# Build binary
make build

# Docker
make docker-up
make docker-down
```

---

## 6. Kiểm tra sau cài đặt

| Kiểm tra | Lệnh/URL | Kết quả mong đợi |
|----------|-----------|-------------------|
| Server chạy | `curl http://localhost:8080/health` | `{"status": "UP"}` |
| MySQL kết nối | `curl http://localhost:8080/health/ready` | MySQL: UP |
| Redis kết nối | `curl http://localhost:8080/health/ready` | Redis: UP |
| Swagger UI | Mở `http://localhost:8080/swagger/index.html` | Trang Swagger hiển thị |
| Metrics | `curl http://localhost:8080/metrics` | Prometheus metrics |

---

## 7. Cấu hình gửi Email thật (Production / Testing) với Resend

Mặc định, môi trường phát triển (development) sử dụng **Mailpit** (chạy trong Docker qua port `1025` và giao diện web port `8025`) để bắt tất cả email gửi đi mà không làm phiền người dùng thật.

Khi bạn cần kiểm thử tính năng với email thật hoặc đưa ứng dụng lên Production, chúng tôi khuyên dùng [Resend](https://resend.com) làm SMTP Server.

### Các bước cấu hình:

1. Đăng ký tài khoản và tạo API Key tại [Resend Dashboard](https://resend.com/api-keys).
2. Xác thực tên miền (Verify Domain) của bạn trên Resend (ví dụ: `yourdomain.com`).
3. Mở file `.env` và cập nhật cấu hình **Mail (SMTP)** như sau:

```env
SMTP_HOST=smtp.resend.com
SMTP_PORT=587
SMTP_USER=resend
SMTP_PASSWORD=re_api_key_cua_ban_o_day
SMTP_FROM=noreply@yourdomain.com
```

**Lưu ý:**
- Username của Resend luôn là `resend`.
- `SMTP_FROM` phải là email kết thúc bằng tên miền bạn đã verify (nếu bạn chưa verify tên miền, Resend Sandbox Mode chỉ cho phép gửi từ `onboarding@resend.dev` tới địa chỉ email đăng ký tài khoản của bạn).
- Khởi động lại server (`make dev` hoặc restart docker) để ứng dụng nhận file cấu hình mới.
