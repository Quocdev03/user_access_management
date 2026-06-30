# Quy tắc Code & Quy trình phát triển

## 1. Quy tắc đặt tên (Naming Conventions)

### Package

- Tên ngắn, viết thường, không dấu gạch dưới: `handler`, `service`, `repository`, `model`, `dto`.
- Không dùng tên quá chung chung: `utils`, `helpers`, `common` (ngoại trừ `pkg/`).

### File

- Viết thường, dùng dấu gạch dưới: `auth_handler.go`, `user_service.go`, `audit_repository.go`.
- File trong thư mục model luôn kết thúc bằng `_model.go`: `user_model.go`, `role_model.go`.
- Không lồng ghép quá sâu tên file (ưu tiên `password_repository.go` thay vì `password_reset_repository.go`).
- File test: thêm hậu tố `_test.go`: `auth_service_test.go`.

### Struct & Interface

- PascalCase: `UserService`, `AuthHandler`, `AuditLog`.
- **YAGNI (You Aren't Gonna Need It)**: KHÔNG tạo Interface cho Service hay Repository trừ khi có ít nhất 2 implementation thực tế. Sử dụng con trỏ Struct (Concrete Type) để truyền dependency.
- Nếu cần viết mock cho Unit Test, hãy định nghĩa Interface tại nơi sử dụng (Consumer-driven contract) bên trong file test.

### Hàm & Method

- PascalCase cho exported: `CreateUser`, `FindByEmail`.
- camelCase cho unexported: `hashPassword`, `validateInput`.
- Tên hàm mô tả hành động: động từ + danh từ.

### Biến & Hằng số

- camelCase cho biến: `userID`, `accessToken`, `maxRetries`.
- PascalCase cho hằng số exported: `MaxFailedAttempts`, `DefaultPageSize`.
- Viết hoa toàn bộ cho viết tắt: `userID` (không phải `userId`), `httpClient`.

### Database

- Tên bảng: số nhiều, snake_case: `users`, `user_roles`, `audit_logs`.
- Tên cột: snake_case: `created_at`, `email_verified`, `password_hash`.

---

## 2. Cấu trúc code theo tầng

### Handler (Controller)

```go
// Chỉ xử lý HTTP: bind input, validate, gọi service, trả response
func (h *AuthHandler) Login(c *gin.Context) {
    var req dto.LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.ValidationError(c, err)
        return
    }

    result, err := h.authService.Login(c.Request.Context(), req)
    if err != nil {
        response.Error(c, err)
        return
    }

    response.Success(c, http.StatusOK, "Đăng nhập thành công", result)
}
```

**Quy tắc Handler:**
- Không chứa business logic.
- Không truy cập database trực tiếp.
- Chỉ gọi 1 service method cho mỗi handler.
- Dùng DTO cho request/response, không dùng model trực tiếp.

### Service (Business Logic)

```go
// Chứa toàn bộ business logic, gọi repository qua con trỏ struct
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
    user, err := s.userRepo.FindByEmail(ctx, req.Email)
    if err != nil {
        return nil, ErrInvalidCredentials
    }

    if user.Status == model.StatusLocked {
        return nil, ErrAccountLocked
    }

    if !hash.CheckPassword(req.Password, user.PasswordHash) {
        s.handleFailedLogin(ctx, user)
        return nil, ErrInvalidCredentials
    }

    // Tạo token, lưu session...
    return response, nil
}
```

**Quy tắc Service:**
- Chứa toàn bộ nghiệp vụ.
- Bắt buộc dùng `database.TxManager` bọc trong `RunInTx` nếu nghiệp vụ ghi/sửa nhiều bảng hoặc nhiều dòng dữ liệu.
- Gọi repository qua con trỏ struct (không tạo interface thừa).
- Nhận context.Context ở tham số đầu tiên.
- Trả error có ý nghĩa (custom error, không dùng string).

### Repository (Data Access)

```go

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
    var user model.User
    query := "SELECT id, username, email, password_hash, status FROM users WHERE email = ?"
    err := r.db.QueryRowContext(ctx, query, email).Scan(
        &user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Status,
    )
    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    return &user, err
}
```

**Quy tắc Repository:**
- Không định nghĩa interface tại tầng Repository. Chỉ dùng struct.
- Implementation chỉ xử lý data access, không chứa business logic.
- Dùng `database.GetDB(ctx, r.db)` thay vì `r.db` trực tiếp để hỗ trợ Transaction (Unit of Work).
- Dùng parameterized queries (chống SQL injection).
- Trả model/entity, không trả DTO.

---

## 3. Xử lý lỗi (Error Handling)

### Định nghĩa lỗi tập trung

```go
// pkg/apperror/errors.go
var (
    ErrInvalidCredentials = NewAppError("INVALID_CREDENTIALS", "Sai email hoặc mật khẩu", http.StatusUnauthorized)
    ErrAccountLocked      = NewAppError("ACCOUNT_LOCKED", "Tài khoản đã bị khóa", http.StatusLocked)
    ErrNotFound           = NewAppError("NOT_FOUND", "Không tìm thấy tài nguyên", http.StatusNotFound)
    ErrConflict           = NewAppError("CONFLICT", "Dữ liệu đã tồn tại", http.StatusConflict)
    ErrForbidden          = NewAppError("FORBIDDEN", "Không có quyền truy cập", http.StatusForbidden)
)
```

### Quy tắc

- Dùng custom error type `AppError` với mã lỗi, message, HTTP status.
- Không dùng `fmt.Errorf` cho business error → dùng sentinel error hoặc `AppError`.
- Wrap error khi cần thêm context: `fmt.Errorf("tìm user theo email: %w", err)`.
- Handler chuyển đổi error thành HTTP response thông qua helper `response.Error()`.

---

## 4. Logging

### Quy tắc

- Dùng **Zap Logger** (structured JSON logging).
- Log levels: `DEBUG`, `INFO`, `WARN`, `ERROR`.
- Không dùng `fmt.Println` hay `log.Println`.
- Luôn kèm context: `request_id`, `user_id`, `action`.

```go
logger.Info("Đăng nhập thành công",
    zap.Int64("user_id", user.ID),
    zap.String("ip", clientIP),
)

logger.Error("Lỗi truy vấn database",
    zap.Error(err),
    zap.String("query", "FindByEmail"),
)
```

### Không log

- Mật khẩu (kể cả hash).
- Token (access/refresh).
- Thông tin nhạy cảm.

---

## 5. Validation

- Validate ở tầng Handler bằng **binding tags** của Gin.
- Validate business rules ở tầng Service.

```go
type RegisterRequest struct {
    Username string `json:"username" binding:"required,min=3,max=50,alphanum"`
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
    FullName string `json:"full_name" binding:"required,min=2,max=100"`
}
```

---

## 6. Testing

### Cấu trúc test

| Loại | Tầng | Cách test |
|------|------|-----------|
| Unit Test | Service | Định nghĩa Interface tại file test để mock, hoặc dùng Testcontainer |
| Unit Test | Handler | Định nghĩa Interface tại file test để mock, dùng `httptest` |
| Integration Test | Repository | Database thật (test container) |
| API Test | Toàn bộ | HTTP client gọi API thật |

### Quy tắc test

- File test đặt cùng package với code được test.
- Tên test: `Test_<Function>_<Scenario>` hoặc `TestFunction_Scenario`.
- Dùng **table-driven tests** khi có nhiều case.
- Mock bằng interface, không dùng framework mock phức tạp.

```go
func TestAuthService_Login_Success(t *testing.T) {
    // Để mock, định nghĩa interface cục bộ tại đây nếu cần thiết
    // type mockUserRepo interface { ... }
    
    // Arrange
    mockRepo := &mockUserRepository{...}
    service := NewAuthService(mockRepo)

    // Act
    result, err := service.Login(context.Background(), dto.LoginRequest{
        Email:    "test@example.com",
        Password: "password123",
    })

    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, result.AccessToken)
}
```

---

## 7. Quy trình Git

### Branching Strategy

```
main          ← Production-ready code
  └── develop ← Tích hợp các feature
        ├── feature/uc01-registration    ← Phát triển tính năng
        ├── feature/uc03-login
        ├── bugfix/fix-token-expiry      ← Sửa lỗi
        └── hotfix/security-patch        ← Sửa lỗi khẩn cấp
```

### Quy tắc branch

- `feature/<uc-id>-<tên-ngắn>`: tính năng mới.
- `bugfix/<mô-tả>`: sửa lỗi thường.
- `hotfix/<mô-tả>`: sửa lỗi khẩn cấp trên production.
- Luôn tạo branch từ `develop` (trừ hotfix từ `main`).

### Commit Message Convention

```
<type>(<scope>): <mô tả ngắn>

<mô tả chi tiết (tùy chọn)>
```

**Type:**

| Type | Mô tả |
|------|-------|
| `feat` | Tính năng mới |
| `fix` | Sửa lỗi |
| `docs` | Thay đổi tài liệu |
| `refactor` | Tái cấu trúc code |
| `test` | Thêm/sửa test |
| `chore` | Cấu hình, build, CI |

**Ví dụ:**

```
feat(auth): thêm API đăng ký tài khoản

- Tạo RegisterHandler, RegisterService
- Validate input, hash password bằng bcrypt
- Gán role mặc định "user"
- Gửi OTP xác thực email

Closes #UC-01
```

### Pull Request

- Tạo PR từ `feature/*` → `develop`.
- Tiêu đề: `[UC-XX] Mô tả ngắn`.
- Mô tả: liệt kê thay đổi, screenshots (nếu có).
- Cần ít nhất 1 review trước khi merge.
- Squash merge để giữ history gọn.

### Code Review Checklist

- [ ] Code tuân thủ Clean Architecture (không vi phạm dependency rule).
- [ ] Có unit test cho service layer.
- [ ] Không hardcode giá trị (dùng config/constant).
- [ ] Error handling đúng cách (không swallow error).
- [ ] Không có sensitive data trong log.
- [ ] API response đúng format chuẩn.
- [ ] Database query có parameterized (chống SQL injection).
- [ ] Tài liệu được cập nhật (nếu cần).
