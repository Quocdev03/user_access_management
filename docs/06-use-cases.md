> [!IMPORTANT]
> **LƯU Ý DÀNH CHO DEVELOPER (AI & HUMAN):**
> Các tài liệu thiết kế này mang tính chất là **KHUNG ĐỊNH HƯỚNG (Framework / Guidelines)**.
> KHÔNG ĐƯỢC áp dụng một cách rập khuôn, máy móc hoặc sao chép hoàn toàn 100%.
> Tùy thuộc vào bối cảnh thực tế của task, bạn phải linh hoạt tùy biến (ví dụ: dùng Atomic Query, Pessimistic Locking FOR UPDATE cho Concurrency, hoặc cấu trúc lại struct).

# Danh sách Use Cases

## Tổng quan

Hệ thống UAM bao gồm **39 Use Cases** chia thành **6 nhóm** chức năng. Tài liệu này mô tả từng UC với thông tin: mô tả, đối tượng sử dụng, endpoint liên quan và quy tắc nghiệp vụ chính.

---

## Nhóm 1 — Xác thực & Phân quyền (Authentication & Authorization)

### UC-01: Đăng ký tài khoản (User Registration)

- **Đối tượng**: Người dùng mới
- **Mô tả**: Người dùng tạo tài khoản bằng username, email và mật khẩu.
- **Endpoint**: `POST /api/v1/auth/register`
- **Luồng chính**:
  1. Người dùng nhập username, email, mật khẩu, họ tên, số điện thoại, ngày sinh.
  2. Hệ thống validate dữ liệu (email hợp lệ, mật khẩu đủ mạnh, username/email chưa tồn tại).
  3. Hệ thống tạo tài khoản với trạng thái `inactive`.
  4. Gán role mặc định `user`.
  5. Gửi OTP qua email để xác thực (→ UC-25).
- **Quy tắc nghiệp vụ**:
  - Mật khẩu phải tuân thủ password policy (→ UC-24).
  - Email và username phải là duy nhất.
  - Mật khẩu được mã hóa bằng bcrypt trước khi lưu.

### UC-02: Xác thực Email/OTP (Email/OTP Verification)

- **Đối tượng**: Người dùng đã đăng ký
- **Mô tả**: Xác thực email bằng mã OTP gửi qua email.
- **Endpoints**: 
  - `POST /api/v1/auth/email/verify` (Xác thực mã OTP)
  - `POST /api/v1/auth/email/resend` (Gửi lại mã OTP xác thực)
- **Luồng chính**:
  1. Người dùng nhập email và mã OTP.
  2. Hệ thống kiểm tra OTP: đúng, chưa hết hạn, chưa sử dụng.
  3. Cập nhật `email_verified = true`, chuyển trạng thái sang `active`.
- **Quy tắc nghiệp vụ**:
  - OTP có thời hạn 5 phút.
  - Tối đa 5 lần nhập sai, sau đó OTP bị vô hiệu.
  - Có thể gửi lại OTP mới (giới hạn rate limit).

### UC-03: Đăng nhập (User Login)

- **Đối tượng**: Người dùng đã xác thực email
- **Mô tả**: Đăng nhập bằng email/username và mật khẩu, nhận JWT token.
- **Endpoint**: `POST /api/v1/auth/login`
- **Luồng chính**:
  1. Người dùng nhập email (hoặc username) và mật khẩu.
  2. Hệ thống kiểm tra tài khoản tồn tại, trạng thái active, mật khẩu đúng.
  3. Tạo access token (JWT, 15 phút) và refresh token (7 ngày).
  4. Lưu session vào Redis và database.
  5. Ghi audit log (→ UC-28).
  6. Cập nhật `last_login_at`, reset `failed_login_attempts = 0`.
- **Quy tắc nghiệp vụ**:
  - Nếu mật khẩu sai → tăng `failed_login_attempts` (→ UC-06).
  - Nếu tài khoản bị khóa → trả lỗi `ACCOUNT_LOCKED`.
  - Nếu email chưa xác thực → trả lỗi yêu cầu xác thực trước.
  - Ghi nhận thông tin thiết bị (→ UC-19).

### UC-04: Làm mới Token (Refresh Token)

- **Đối tượng**: Người dùng đã đăng nhập
- **Mô tả**: Dùng refresh token để lấy access token mới khi access token hết hạn.
- **Endpoint**: `POST /api/v1/auth/token/refresh`
- **Luồng chính**:
  1. Client gửi refresh token.
  2. Hệ thống validate refresh token (đúng, chưa hết hạn, chưa bị revoke).
  3. Tạo cặp access token + refresh token mới.
  4. Vô hiệu refresh token cũ (rotation).
- **Quy tắc nghiệp vụ**:
  - Áp dụng **refresh token rotation**: mỗi lần refresh tạo refresh token mới, token cũ bị vô hiệu.
  - Nếu phát hiện refresh token cũ đã bị sử dụng → revoke toàn bộ session của user (phòng token bị đánh cắp).

### UC-05: Đăng xuất (Logout)

- **Đối tượng**: Người dùng đã đăng nhập
- **Mô tả**: Đăng xuất khỏi phiên hiện tại.
- **Endpoint**: `POST /api/v1/auth/logout`
- **Luồng chính**:
  1. Client gửi access token trong header.
  2. Hệ thống thêm token vào blacklist Redis.
  3. Xóa session tương ứng.
- **Quy tắc nghiệp vụ**:
  - Token bị blacklist đến khi hết hạn tự nhiên.

### UC-06: Khóa tài khoản sau N lần đăng nhập thất bại

- **Đối tượng**: Hệ thống (tự động)
- **Mô tả**: Tự động khóa tài khoản khi đăng nhập sai quá số lần cho phép.
- **Luồng chính**:
  1. Mỗi lần đăng nhập sai → tăng `failed_login_attempts`.
  2. Khi đạt ngưỡng (mặc định 10 lần) → chuyển trạng thái sang `locked`, set `locked_until`.
  3. Ghi audit log.
- **Quy tắc nghiệp vụ**:
  - Ngưỡng mặc định: 10 lần.
  - Thời gian khóa: 15 phút (tự mở khóa sau khi hết thời gian).
  - Đăng nhập thành công → reset bộ đếm về 0.

### UC-07: Quên mật khẩu (Forgot Password)

- **Đối tượng**: Người dùng
- **Mô tả**: Yêu cầu đặt lại mật khẩu qua email.
- **Endpoint**: `POST /api/v1/auth/password/forgot`
- **Luồng chính**:
  1. Người dùng nhập email.
  2. Hệ thống tạo token đặt lại mật khẩu, lưu hash vào database.
  3. Gửi email chứa link/token (→ UC-26).
- **Quy tắc nghiệp vụ**:
  - Luôn trả thành công dù email có tồn tại hay không (chống enumeration).
  - Token có thời hạn 1 giờ.
  - Chỉ token mới nhất có hiệu lực (token cũ bị vô hiệu).

### UC-08: Đổi mật khẩu (Change Password)

- **Đối tượng**: Người dùng đã đăng nhập
- **Mô tả**: Đổi mật khẩu khi đã biết mật khẩu cũ.
- **Endpoint**: `POST /api/v1/auth/password/change`
- **Luồng chính**:
  1. Người dùng nhập mật khẩu cũ và mật khẩu mới.
  2. Hệ thống xác minh mật khẩu cũ đúng.
  3. Validate mật khẩu mới theo policy (→ UC-24).
  4. Cập nhật mật khẩu, revoke tất cả session khác.
- **Quy tắc nghiệp vụ**:
  - Mật khẩu mới không được trùng mật khẩu cũ.

### UC-09: Phân quyền theo vai trò (RBAC)

- **Đối tượng**: Hệ thống
- **Mô tả**: Kiểm soát quyền truy cập dựa trên vai trò (Role) được gán cho người dùng.
- **Cơ chế**:
  1. Middleware RBAC chạy sau middleware Auth.
  2. Lấy danh sách roles của user từ context/database.
  3. Kiểm tra user có role yêu cầu hay không.
  4. Nếu không → trả 403 Forbidden.
- **Vai trò mặc định**:
  - `admin`: Toàn quyền quản trị.
  - `moderator`: Quản lý user, xem audit logs.
  - `user`: Chỉ quản lý thông tin cá nhân.

### UC-10: Phân quyền theo Permission

- **Đối tượng**: Hệ thống
- **Mô tả**: Kiểm soát quyền truy cập chi tiết hơn RBAC, dựa trên permission cụ thể.
- **Cơ chế**:
  1. Mỗi role được gán danh sách permissions (VD: `users.read`, `users.update`).
  2. Middleware Permission kiểm tra user có permission yêu cầu cho endpoint.
  3. Permission được định nghĩa theo format: `{resource}.{action}`.

### UC-32: Quản lý Vai trò và Quyền (Roles & Permissions Management)

- **Đối tượng**: Admin
- **Mô tả**: Admin có quyền quản lý các vai trò (roles) và gán danh sách quyền hạn (permissions) tương ứng cho từng vai trò, cũng như gán hoặc gỡ vai trò của người dùng.
- **Endpoints**:
  - `GET /api/v1/admin/roles` (Xem danh sách vai trò)
  - `POST /api/v1/admin/roles` (Tạo vai trò mới)
  - `PUT /api/v1/admin/roles/{id}` (Cập nhật vai trò)
  - `DELETE /api/v1/admin/roles/{id}` (Xóa vai trò)
  - `GET /api/v1/admin/roles/{id}/permissions` (Xem quyền của vai trò)
  - `PUT /api/v1/admin/roles/{id}/permissions` (Gán quyền cho vai trò)
  - `POST /api/v1/admin/users/{id}/roles` (Gán vai trò cho người dùng)
  - `DELETE /api/v1/admin/users/{id}/roles/{roleId}` (Gỡ vai trò khỏi người dùng)
- **Quy tắc nghiệp vụ**:
  - Chỉ admin mới có quyền thực hiện.
  - Sau khi gán/gỡ vai trò hoặc thay đổi quyền của vai trò, hệ thống sẽ thực hiện thu hồi phiên hoạt động của các user bị ảnh hưởng để các thay đổi phân quyền có hiệu lực ngay lập tức.
  - Ghi audit log cho tất cả các thao tác thay đổi cấu hình phân quyền (→ UC-29).

---

## Nhóm 2 — Hồ sơ & Quản lý người dùng (User Profile & User Service)

### UC-11: Xem hồ sơ cá nhân (View Profile)

- **Đối tượng**: Người dùng đã đăng nhập
- **Mô tả**: Xem thông tin cá nhân của mình.
- **Endpoint**: `GET /api/v1/users/me`
- **Dữ liệu trả về**: username, email, full_name, phone, avatar_url, status, created_at, roles.

### UC-12: Cập nhật hồ sơ (Update Profile)

- **Đối tượng**: Người dùng đã đăng nhập
- **Mô tả**: Cập nhật thông tin cá nhân (họ tên, số điện thoại).
- **Endpoint**: `PUT /api/v1/users/me`
- **Quy tắc**: Không thể đổi username, email (dùng UC-13), mật khẩu (dùng UC-08).

### UC-13: Đổi Email (Change Email)

- **Đối tượng**: Người dùng đã đăng nhập.
- **Mô tả**: Đổi địa chỉ email của tài khoản. Đơn giản hóa quy trình thành 2 bước: yêu cầu đổi email (gửi OTP đồng thời) và xác thực (nhập 2 OTP cùng lúc).
- **Endpoints**: 
  - `POST /api/v1/users/me/email/change-request` (Xác thực mật khẩu -> Gửi OTP đến cả email cũ và mới)
  - `POST /api/v1/users/me/email/verify` (Xác thực OTP)
  - `POST /api/v1/users/me/email/resend` (Gửi lại mã OTP đổi email)
- **Luồng chính**:
  1. Người dùng gửi yêu cầu đổi email kèm mật khẩu hiện tại và email mới.
  2. Hệ thống xác thực mật khẩu và sinh mã OTP gửi đồng thời đến **email cũ** và **email mới**.
  3. Người dùng nhập cả 2 mã OTP cùng lúc để xác nhận.
  4. Hệ thống kiểm tra trùng email mới, xác thực 2 mã OTP (Atomic). Nếu hợp lệ, cập nhật email và gửi thư thông báo bảo mật đến cả 2 email.
- **Quy tắc & Luồng ngoại lệ**:
  - **Re-authentication**: Bắt buộc nhập đúng mật khẩu hiện tại ở bước 1 để chặn truy cập trái phép.
  - **Race Condition Check**: Kiểm tra tính duy nhất của email mới ở cả bước yêu cầu (bước 1) và bước cập nhật cuối cùng (bước 5).
  - **Mất quyền truy cập email cũ**: Đây là luồng tự động tự phục vụ (self-service). Nếu người dùng mất quyền truy cập email cũ, họ bắt buộc phải liên hệ bộ phận hỗ trợ (Customer Support) để xác minh danh tính thủ công.
  - **Brute-Force OTP**: Mỗi mã OTP chỉ được thử sai tối đa 5 lần. Quá 5 lần, mã OTP sẽ bị vô hiệu hóa.
  - **Security Notification**: Sau khi đổi thành công, hệ thống bắt buộc gửi email thông báo bảo mật đến email cũ cảnh báo về thay đổi này để chủ tài khoản thực tế kịp thời phát hiện nếu bị hack.

### UC-14: Upload ảnh đại diện (Upload Avatar)

- **Đối tượng**: Người dùng đã đăng nhập
- **Mô tả**: Upload hoặc thay đổi ảnh đại diện.
- **Endpoint**: `POST /api/v1/users/me/avatar`
- **Quy tắc**:
  - Định dạng cho phép: JPEG, PNG, WebP.
  - Kích thước tối đa: 2MB.
  - Lưu trữ local hoặc cloud storage.

### UC-15: Admin tìm kiếm người dùng (Admin Search Users)

- **Đối tượng**: Admin, Moderator
- **Mô tả**: Tìm kiếm và lọc danh sách người dùng.
- **Endpoint**: `GET /api/v1/admin/users`
- **Bộ lọc**: username, email, status, role, ngày tạo.
- **Hỗ trợ phân trang, sắp xếp**.

### UC-16: Admin cập nhật thông tin người dùng

- **Đối tượng**: Admin
- **Mô tả**: Admin cập nhật thông tin (họ tên, số điện thoại, email, trạng thái xác thực email, ngày sinh, avatar) của người dùng.
- **Endpoint**: `PUT /api/v1/admin/users/{id}`
- **Quy tắc**: Ghi audit log (→ UC-29).

### UC-17: Admin bật/tắt/khóa tài khoản

- **Đối tượng**: Admin
- **Mô tả**: Thay đổi trạng thái tài khoản: active, inactive, locked.
- **Endpoint**: `PATCH /api/v1/admin/users/{id}/status`
- **Quy tắc**:
  - Khi khóa → revoke tất cả session của user.
  - Ghi audit log (→ UC-29).

### UC-18: Admin đặt lại mật khẩu

- **Đối tượng**: Admin
- **Mô tả**: Admin đặt lại mật khẩu cho người dùng, hệ thống gửi mật khẩu tạm qua email.
- **Endpoint**: `POST /api/v1/admin/users/{id}/password/reset`
- **Quy tắc**:
  - Mật khẩu tạm được sinh tự động.
  - Revoke tất cả session của user.
  - User bắt buộc đổi mật khẩu sau khi đăng nhập.

---

## Nhóm 3 — Mở rộng bảo mật (Security Extension)

### UC-19: Theo dõi thiết bị (Device Tracking)

- **Đối tượng**: Hệ thống
- **Mô tả**: Ghi nhận thông tin thiết bị (browser, OS, IP) mỗi lần đăng nhập.
- **Cơ chế**: Parse User-Agent header, lưu vào bảng `devices`.
- **Liên kết**: UC-03 (Login), UC-20 (Session).

### UC-20: Quản lý phiên đăng nhập (Session Management)

- **Đối tượng**: Người dùng đã đăng nhập
- **Mô tả**: Xem danh sách phiên đăng nhập đang hoạt động.
- **Endpoints**: 
  - `GET /api/v1/users/me/sessions` (Xem danh sách phiên đăng nhập đang hoạt động)
  - `DELETE /api/v1/users/me/sessions/{id}` (Hủy một phiên đăng nhập cụ thể)
  - `GET /api/v1/users/me/devices` (Xem danh sách thiết bị đã từng đăng nhập)

### UC-21: Hủy tất cả phiên (Revoke All Sessions)

- **Đối tượng**: Người dùng đã đăng nhập
- **Mô tả**: Đăng xuất khỏi tất cả thiết bị (trừ phiên hiện tại).
- **Endpoint**: `POST /api/v1/auth/logout-all`
- **Cơ chế**: Xóa tất cả session trong Redis, blacklist tất cả token.

### UC-22: Bảo vệ chống Brute Force

- **Đối tượng**: Hệ thống
- **Mô tả**: Kết hợp UC-06 (khóa tài khoản) và UC-23 (rate limiting) để chống tấn công brute force.
- **Cơ chế**:
  - Giới hạn số lần đăng nhập thất bại trên mỗi tài khoản.
  - Giới hạn số lần request trên mỗi IP.
  - Tăng thời gian chờ theo cấp số nhân (progressive delay).

### UC-23: Giới hạn tốc độ request (Rate Limiting)

- **Đối tượng**: Hệ thống
- **Mô tả**: Giới hạn số request trên mỗi IP/user trong khoảng thời gian, dùng Redis.
- **Cơ chế**: Sliding window counter lưu trong Redis.
- **Cấu hình mặc định**:
  - Cơ chế 2 tầng: Tầng Soft Limit (báo lỗi 429) và Hard Ban (Khóa IP 15 phút nếu spam quá lớn).
  - API chung: 100 request/phút (ban 300).
  - Auth (Login): 15 request/phút (ban 50).
  - Auth (Register/Reset PW): 10 request/phút (ban 30).
  - Bổ sung: Validate Data ở Service layer LUÔN thực thi trước khi đánh Rate Limit Token Bucket.

### UC-24: Chính sách mật khẩu (Password Policy)

- **Đối tượng**: Hệ thống
- **Mô tả**: Áp dụng quy tắc mật khẩu mạnh.
- **Quy tắc mặc định**:
  - Tối thiểu 8 ký tự.
  - Có ít nhất 1 chữ hoa, 1 chữ thường, 1 số, 1 ký tự đặc biệt.
  - Không trùng mật khẩu cũ.
- **Áp dụng tại**: UC-01 (Register), UC-08 (Change Password), UC-18 (Admin Reset).

---

## Nhóm 4 — Dịch vụ thông báo (Notification Service)

### UC-25: Gửi email xác thực đăng ký

- **Đối tượng**: Hệ thống
- **Mô tả**: Gửi OTP qua email sau khi đăng ký.
- **Kích hoạt bởi**: UC-01 (Register).
- **Nội dung**: Mã OTP 6 chữ số, hết hạn sau 5 phút.

### UC-26: Gửi email quên mật khẩu

- **Đối tượng**: Hệ thống
- **Mô tả**: Gửi link/token đặt lại mật khẩu qua email.
- **Kích hoạt bởi**: UC-07 (Forgot Password).
- **Nội dung**: Link chứa token, hết hạn sau 1 giờ.

### UC-27: Admin gửi thông báo bắt buộc

- **Đối tượng**: Admin
- **Mô tả**: Admin gửi email thông báo bắt buộc đến một hoặc nhiều người dùng.
- **Endpoint**: `POST /api/v1/admin/users/{id}/notify`
- **Quy tắc**: Ghi audit log.

---

## Nhóm 5 — Kiểm toán & Giám sát (Audit & Monitoring)

### UC-28: Ghi nhật ký đăng nhập (Audit Login Events)

- **Đối tượng**: Hệ thống
- **Mô tả**: Tự động ghi log mỗi lần đăng nhập (thành công/thất bại).
- **Dữ liệu ghi**: user_id, IP, user_agent, thời gian, kết quả.
- **Kích hoạt bởi**: UC-03 (Login).

### UC-29: Ghi nhật ký hành động (Audit User Actions)

- **Đối tượng**: Hệ thống
- **Mô tả**: Ghi log các hành động quan trọng: cập nhật profile, đổi mật khẩu, thay đổi trạng thái, gán role.
- **Dữ liệu ghi**: user_id, action, resource, old_values, new_values.

### UC-30: Export nhật ký (Export Audit Logs)

- **Đối tượng**: Admin
- **Mô tả**: Export nhật ký kiểm toán ra file CSV.
- **Endpoint**: `GET /api/v1/admin/audit-logs/export`
- **Hỗ trợ lọc**: theo user, action, khoảng thời gian.

### UC-31: Kiểm tra sức khỏe hệ thống (Health Check)

- **Đối tượng**: Hệ thống vận hành
- **Mô tả**: Endpoint kiểm tra trạng thái ứng dụng và kết nối.
- **Endpoints**:
  - `GET /health` → kiểm tra ứng dụng đang chạy.
  - `GET /health/ready` → kiểm tra kết nối MySQL, Redis.
- **Response**: trạng thái `UP`/`DOWN` cho từng dependency.



---

## Nhóm 6 — Hệ thống & Công cụ phát triển (System & Developer Tools)

### UC-33: Swagger API Documentation

- **Mô tả**: Tự động sinh API documentation từ code bằng Swaggo.
- **Endpoint**: `GET /swagger/*`
- **Công cụ**: `swag init` sinh file swagger.json.

### UC-34: Database Migration

- **Mô tả**: Quản lý schema database bằng golang-migrate.
- **Lệnh**: `make migrate-up`, `make migrate-down`, `make migrate-create`.

### UC-35: Seed Data

- **Mô tả**: Tạo dữ liệu mẫu ban đầu: roles, permissions, admin account.
- **Lệnh**: `make seed`
- **Dữ liệu mặc định**:
  - 3 roles: `admin`, `moderator`, `user`.
  - Permissions cho từng resource.
  - 1 tài khoản admin mặc định.

### UC-36: Cấu hình môi trường (Environment Configuration)

- **Mô tả**: Quản lý cấu hình qua file `.env` và biến môi trường.
- **File mẫu**: `.env.example`
- **Thư viện**: Viper hoặc godotenv.

### UC-37: Docker Compose Setup

- **Mô tả**: Cấu hình Docker Compose để chạy toàn bộ hệ thống.
- **Services**: app (Go), mysql, redis.
- **File**: `docker-compose.yml`, `Dockerfile`.

### UC-38: Unit & Integration Testing

- **Mô tả**: Viết test cho từng tầng.
- **Unit test**: Service layer (mock repository).
- **Integration test**: Repository layer (test database thật).
- **API test**: Handler layer (httptest).
- **Lệnh**: `make test`, `make test-coverage`.

### UC-39: Structured Logging

- **Mô tả**: Sử dụng Zap Logger ghi log có cấu trúc JSON.
- **Log levels**: DEBUG, INFO, WARN, ERROR, FATAL.
- **Thông tin log**: timestamp, level, message, request_id, user_id, error stack.
