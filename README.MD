# UAM — User Access Management

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

Backend **training** quản lý user / auth / RBAC bằng **Go + Gin + MySQL + Redis**.  
Không phải product SaaS đầy đủ — mục tiêu là code theo Clean Architecture và các pattern auth phổ biến.

**Stack:** Gin · JWT (access + refresh) · MySQL (session, OTP hash, audit) · Redis (blacklist, revoke epoch, rate-limit) · SMTP (Mailpit local / Resend prod)

---

## Tài liệu (chi tiết nằm ở đây, không lặp trong README)

| File | Nội dung |
|------|----------|
| [docs/01-architecture.md](./docs/01-architecture.md) | Tầng Handler → Service → Repository, middleware |
| [docs/02-database-design.md](./docs/02-database-design.md) | Schema, Redis keys, migration |
| [docs/03-use-cases.md](./docs/03-use-cases.md) | Use case + endpoint thật |
| [docs/04-environment-setup.md](./docs/04-environment-setup.md) | Docker, env, mail, deploy |
| [AGENTS.md](./AGENTS.md) | Quy tắc code cho AI / contributor |

Swagger: generate bằng `make swagger` / `make.cmd swagger` → `docs/docs.go`, `swagger.json`, `swagger.yaml` (không sửa tay).  
UI test: `http://localhost:8080/` (chỉ dev; Swagger tắt khi `APP_ENV=production`).

---

## Chạy local (nhanh)

**App trên host + MySQL/Redis/Mailpit qua Docker** (cách thường dùng trên Windows):

```bash
# 1) Infra
docker compose up -d

# 2) Env
cp .env.local.example .env
# SMTP_PORT=1026 (map Mailpit; inbox http://localhost:8025)

# 3) Server (tự migrate khi start)
go run ./cmd/server
# hoặc: air  |  make.cmd run
```

| URL | |
|-----|--|
| API | `http://localhost:8080` |
| Health | `http://localhost:8080/health` |
| UI test | `http://localhost:8080/` |
| Mailpit | `http://localhost:8025` |
| Swagger | `http://localhost:8080/swagger/index.html` |

**Admin seed (sau migrate):**

| | |
|--|--|
| Email | `admin@localhost.local` |
| Password | `LocalDev@ChangeMe1` |
| Username | `admin_local` (login API dùng **email**, không dùng username) |

Env templates:

- `.env.local.example` — app host + `localhost` ports  
- `.env.example` — khi app cũng trong Docker network (`DB_HOST=mysql`)  
- `.env.production.example` — Render + Aiven + Resend (placeholder, không secret)

Lệnh tiện: `make.cmd help` (Windows) / `make help` (Unix).

---

## Phạm vi tính năng (tóm tắt)

- Auth: register, OTP email, login, refresh rotation + reuse detection, logout / logout-all, forgot-reset, force-change password  
- User: profile, avatar, đổi email (2 OTP), sessions, devices  
- Admin: list/update user, status, reset password, notify, roles/permissions, export audit CSV  
- Bảo vệ: rate-limit + IP ban, lock account sau N lần sai, bcrypt, OTP hash at rest, session invalidate (MySQL + Redis epoch)

Chi tiết endpoint → [docs/03-use-cases.md](./docs/03-use-cases.md).

---

## Cấu trúc repo

```text
cmd/server/          # main
internal/
  handler/           # HTTP
  service/           # business
  repository/        # MySQL + Redis
  middleware/        # auth, RBAC, rate-limit, CORS
  router/ dto/ model/ config/ constant/ worker/
pkg/                 # apperror, jwt, hash, database, response, logger
migrations/          # golang-migrate (single source of schema)
docs/                # tài liệu + swagger generated
ui_test/             # API console (static)
```

Luồng: **Handler → Service → Repository**. DI bằng concrete struct (xem `docs/01-architecture.md`).

---

## Deploy (test prod)

Ví dụ đã dùng: **Render** (Go + Redis) + **Aiven** (MySQL) + **Resend** (SMTP).

- `DB_TLS_SKIP_VERIFY=true` với Aiven  
- Resend: `SMTP_PORT=587` (không dùng 2525)  
- Domain `SMTP_FROM` phải verified trên Resend  

Chi tiết env → [docs/04-environment-setup.md](./docs/04-environment-setup.md) và `.env.production.example`.

---

## License

[MIT](./LICENSE)
