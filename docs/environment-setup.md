# Hướng dẫn cài đặt & Chạy dự án

## 1. Yêu cầu hệ thống

| Công cụ | Phiên bản | Mô tả |
|---------|-----------|-------|
| Go | 1.24+ | Ngôn ngữ lập trình |
| MySQL | 8.0+ | Cơ sở dữ liệu |
| Redis | 7.0+ | Cache & session store |
| Docker | 24.0+ | Container runtime |
| Docker Compose | 2.20+ | Quản lý multi-container |
| Make | 4.0+ | Task runner |
| Git | 2.40+ | Quản lý mã nguồn |

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

# Mail (SMTP)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=noreply@yourdomain.com

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
# Khởi chạy tất cả services (app + mysql + redis)
docker compose up -d

# Xem logs
docker compose logs -f app

# Chạy migration
docker compose exec app make migrate-up

# Tạo dữ liệu mẫu
docker compose exec app make seed
```

### Bước 4: Kiểm tra

```bash
# Kiểm tra health check
curl http://localhost:8080/health

# Xem Swagger UI
# Mở trình duyệt: http://localhost:8080/swagger/index.html
```

---

## 3. Docker Compose — Chi tiết

### Cấu trúc file `docker-compose.yml`

```yaml
services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "${APP_PORT:-8080}:8080"
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
    env_file:
      - .env
    volumes:
      - ./:/app                    # Mount source code (dev mode)
      - uploads:/app/uploads       # Persistent uploads
    restart: unless-stopped

  mysql:
    image: mysql:8.0
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
    image: redis:7-alpine
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

volumes:
  mysql_data:
  redis_data:
  uploads:
```

### Cấu trúc file `Dockerfile`

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder
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
go version   # Kiểm tra: go1.24+
```

### Bước 2: Cài đặt MySQL

Cài MySQL 8.0+ và tạo database:

```sql
CREATE DATABASE uam_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'uam_user'@'localhost' IDENTIFIED BY 'uam_password';
GRANT ALL PRIVILEGES ON uam_db.* TO 'uam_user'@'localhost';
FLUSH PRIVILEGES;
```

### Bước 3: Cài đặt Redis

Cài Redis 7.0+ và khởi chạy:

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
