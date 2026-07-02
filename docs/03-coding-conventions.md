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

### Handler (Presentation Layer)
- **Trách nhiệm**: Nhận HTTP request, bind & validate input DTO, gọi đúng 1 phương thức Service tương ứng, định dạng và trả về JSON response.
- **Quy tắc**: Không chứa logic nghiệp vụ hay truy xuất database trực tiếp.

```go
// Khung cấu trúc Handler mẫu
func (h *AuthHandler) Action(c *gin.Context) {
    var req dto.ActionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.ValidationError(c, err)
        return
    }
    res, err := h.service.Action(c.Request.Context(), req)
    if err != nil {
        response.Error(c, err)
        return
    }
    response.Success(c, http.StatusOK, "Thành công", res)
}
```

### Service (Business Logic Layer)
- **Trách nhiệm**: Xử lý toàn bộ logic nghiệp vụ (business rules), điều phối giao dịch (transactions), mã hóa dữ liệu, gửi email, tích hợp bên thứ ba.
- **Quy tắc**: 
  - Gọi Repository thông qua con trỏ struct trực tiếp (Concrete Type).
  - Bắt buộc dùng `database.TxManager` bọc trong `RunInTx` nếu nghiệp vụ ghi/sửa nhiều bảng hoặc nhiều dòng dữ liệu.
  - Trả về error có ý nghĩa (`AppError` hoặc sentinel errors).

```go
// Khung cấu trúc Service mẫu
func (s *Service) Action(ctx context.Context, req dto.ActionRequest) (*dto.ActionResponse, error) {
    // 1. Thực hiện logic nghiệp vụ...
    // 2. Gọi Repository...
    // 3. Đóng gói kết quả trả về...
    return response, nil
}
```

### Repository (Data Access Layer)
- **Trách nhiệm**: Chỉ thực hiện các thao tác đọc/ghi cơ sở dữ liệu (MySQL, Redis, v.v.).
- **Quy tắc**:
  - Không chứa logic nghiệp vụ.
  - Dùng `database.GetDB(ctx, r.db)` để tự động hỗ trợ transaction.
  - Luôn sử dụng parameterized queries đề phòng SQL Injection.

```go
// Khung cấu trúc Repository mẫu
func (r *Repository) FindByID(ctx context.Context, id uint64) (*model.Entity, error) {
    var entity model.Entity
    query := "SELECT * FROM table WHERE id = ? LIMIT 1"
    err := database.GetDB(ctx, r.db).GetContext(ctx, &entity, query, id)
    return &entity, err
}
```

---

## 3. Xử lý lỗi (Error Handling)

- Định nghĩa lỗi tập trung dưới dạng các `AppError` chứa mã lỗi (`Code`), thông báo dễ hiểu (`Message`) và HTTP status code.
- Luôn sử dụng sentinel errors / custom `AppError` được định nghĩa tập trung ở package `pkg/apperror`, không trả lỗi thô dạng string.
- Khi cần wrap lỗi cấp dưới, dùng `fmt.Errorf("context message: %w", err)` để giữ stack trace của lỗi gốc phục vụ cho debugging.
- Tầng Handler có trách nhiệm chuyển đổi error sang HTTP response bằng helper `response.Error()`.

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
// Khung viết Unit Test mẫu cho Service
func TestService_Action_Success(t *testing.T) {
    // Arrange (Thiết lập dependencies mock cục bộ hoặc struct cụ thể)
    mockRepo := &mockRepository{...}
    svc := NewService(mockRepo)

    // Act (Thực hiện hành động)
    res, err := svc.Action(context.Background(), req)

    // Assert (Xác minh kết quả)
    assert.NoError(t, err)
    assert.Equal(t, expected, res.Field)
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
