# AI Project Instructions

## Bước 1: Load TOÀN BỘ Skills (BẮT BUỘC — KHÔNG ĐƯỢC BỎ QUA)

> ⛔ **HARD BLOCK**: Đây là bước **số 0**, thực hiện **trước tất cả mọi thứ** — kể cả trước khi đọc docs, trước khi nghĩ về plan, trước khi gọi bất kỳ tool nào khác. Vi phạm quy tắc này → output sẽ sai domain và phải làm lại từ đầu.

Đọc lần lượt **TOÀN BỘ** các file SKILL.md dưới đây bằng `view_file`, **không được bỏ sót bất kỳ file nào**, kể cả khi thấy task hiện tại "có vẻ" không liên quan đến skill đó:

```
.agents/skills/brainstorming/SKILL.md
.agents/skills/devops-engineer/SKILL.md
.agents/skills/golang-concurrency/SKILL.md
.agents/skills/golang-code-style/SKILL.md
.agents/skills/golang-database/SKILL.md
.agents/skills/golang-dependency-injection/SKILL.md
.agents/skills/golang-dependency-management/SKILL.md
.agents/skills/golang-design-patterns/SKILL.md
.agents/skills/golang-documentation/SKILL.md
.agents/skills/golang-modernize/SKILL.md
.agents/skills/golang-performance/SKILL.md
.agents/skills/golang-project-layout/SKILL.md
.agents/skills/golang-security/SKILL.md
.agents/skills/mysql-best-practices/SKILL.md
.agents/skills/security-review/SKILL.md
.agents/skills/code-comments/SKILL.md
```

**Quy tắc:**

- Đọc đủ **15/15 file** trước khi chuyển sang Bước 2. Không dừng sớm dù task có vẻ đơn giản hay chỉ chạm 1 domain.
- Không cần liệt kê, giải thích hay note lại nội dung đã đọc trong response — chỉ dùng làm context nội bộ để áp dụng khi code.
- Nếu một skill mới được thêm vào thư mục `.agents/skills/` mà chưa có trong danh sách trên → vẫn phải đọc, và báo cho user biết để cập nhật danh sách này.

> **QUY TẮC CỨNG**: Không có khái niệm "skill không liên quan nên bỏ qua". Đọc hết, áp dụng cái nào phù hợp.

---

## Bước 2: Đọc Docs theo ngữ cảnh

Sau khi đọc skills, đọc docs theo thứ tự ưu tiên — **chỉ đọc file liên quan đến task**.
Lưu ý đây chỉ là cấu trúc để định hình, nếu có thay đổi nào tốt hơn thì phải đề xuất trong kế hoạch.

| Ưu tiên | File                             | Trigger cụ thể                                                         |
| ------- | -------------------------------- | ---------------------------------------------------------------------- |
| 1       | `docs/01-overview.md`            | Luôn đọc (bắt buộc, file ngắn)                                         |
| 2       | `docs/02-architecture.md`        | Task tạo/sửa file bất kỳ trong `internal/`, `pkg/`, `cmd/`             |
| 3       | `docs/03-coding-conventions.md`  | Task viết code mới hoặc refactor                                       |
| 4       | `docs/04-api-design.md`          | File path chứa `/handler/` hoặc liên quan đến endpoint/response format |
| 5       | `docs/05-database-design.md`     | File path chứa `/model/`, `/repository/`, hoặc migration               |
| 6       | `docs/06-authentication-flow.md` | Task liên quan đến login, token, middleware auth, RBAC                 |
| 7       | `docs/07-use-cases.md`           | Task mô tả nghiệp vụ (UC), cần hiểu flow người dùng                    |
| 8       | `docs/08-environment-setup.md`   | Task liên quan đến Docker, `.env`, CI/CD, deployment                   |

---

## Bước 3: Plan trước khi implement

Với task liên quan đến **nhiều hơn 1 file** hoặc **tính năng mới**:

1. **Outline plan ngắn** gồm:
    - Các file sẽ tạo mới / sửa (kèm lý do).
    - UC liên quan trong `docs/07-use-cases.md`.
    - Có code tương tự trong codebase không? (xem Bước 4)
2. **Trình bày plan** cho user, chờ confirm trước khi code.

Với task nhỏ (fix bug, sửa 1 chỗ trong 1 file) → không cần plan, làm luôn.

---

## Bước 4: Scan Codebase trước khi tạo file mới

Trước khi tạo handler/service/repository/DTO mới, bắt buộc chạy:

```bash
# Tìm pattern tương tự
grep -r "func.*Handler" internal/handler/ --include="*.go" -l
grep -r "type.*Request struct" internal/dto/ --include="*.go" -l

# Tìm theo tên entity
grep -r "<TênEntity>" internal/ --include="*.go" -l
```

**Quy tắc:**

- Nếu tìm thấy code tương tự → **mở rộng**, không tạo file mới.
- Nếu chưa có → tạo file mới, đặt tên theo convention trong `docs/03-coding-conventions.md`.
- Không duplicate logic — extract thành shared function nếu dùng lại từ 2 nơi trở lên.

---

## Quy tắc kiến trúc & code

1. Tuân thủ Clean Architecture: `Handler → Service → Repository`. Không bỏ tầng, không gọi ngược.
2. Không thay đổi cấu trúc thư mục đã định nghĩa trong `docs/02-architecture.md`.
3. Không thay đổi API response format, error codes đã định nghĩa trong `docs/04-api-design.md`.
4. Giữ naming nhất quán theo `docs/03-coding-conventions.md`.
5. **Dọn dẹp comment (BẮT BUỘC)**: Khi sửa đổi hoặc xóa code, PHẢI xóa hoặc cập nhật các comment (docstrings, inline comments) liên quan đến đoạn code đó. Tuyệt đối không để lại comment cũ gây hiểu lầm (stale comments) khiến các AI Agent sau này đọc nhầm và sinh code sai logic.

---

## Quy tắc tối ưu token (áp dụng xuyên suốt)

1. Dùng `grep`/`search` tìm đúng đoạn cần đọc trong **docs và code** — không đọc toàn bộ file lớn (quy tắc này KHÔNG áp dụng cho Bước 1, skill luôn đọc full).
2. Khi sửa file → dùng replace chính xác đoạn cần sửa, không rewrite toàn file.
3. Gom nhiều thay đổi nhỏ trong cùng 1 file thành 1 lần sửa duy nhất.
4. Không liệt kê lại nội dung docs/skills trong response — dùng làm context nội bộ.
5. Không giải thích code khi không được hỏi — chỉ viết code + comment ngắn inline.
6. **Không được ngắt nội dung giữa chừng**. Nếu code block hoặc nội dung response quá dài, phải dùng tool `write_to_file` / `replace_file_content` để áp thẳng vào file — không cắt rồi đẩy lên chat từng phần.

---

## Quy tắc documentation

- **Cập nhật docs** khi: thêm endpoint mới, thêm bảng DB, đổi luồng xử lý chính.
- **Không cập nhật docs** khi: fix bug nhỏ, refactor nội bộ không đổi interface.

### Cập nhật PROGRESS.md (BẮT BUỘC)

File `PROGRESS.md` là nhật ký tiến độ dự án, **phải được cập nhật sau mỗi task hoàn thành**:

- **Cập nhật khi**: hoàn thành UC mới, tạo file mới, sửa API/DB, fix bug quan trọng.
- **Không cập nhật khi**: fix typo, sửa comment, thay đổi không ảnh hưởng behavior.

**Format cập nhật:**

1. Chuyển UC từ bảng `🚧 Chưa làm` → `✅ Đã hoàn thành`.
2. Thêm file mới vào bảng tương ứng trong phần đã hoàn thành.
3. Ghi ngắn gọn: tên file, method/logic chính — **không viết dài**.

> **QUY TẮC CỨNG**: Nếu một task tạo hoặc sửa ít nhất 1 file Go/SQL, bắt buộc cập nhật `PROGRESS.md` trước khi kết thúc turn.

---

## Quy tắc output đầy đủ — chống truncate

> **ÁP DỤNG BẮT BUỘC cho mọi model.**

1. **Không được viết code rồi để "..." hoặc "// rest of code"** trong bất kỳ hoàn cảnh nào.
2. Nếu nội dung code dài → **dùng tool ghi thẳng vào file**, không paste lên chat.
3. Nếu cần giải thích + code trong cùng 1 response → giải thích ngắn trước, ghi code vào file sau bằng tool.
4. Không chia code thành nhiều phần "Phần 1... Phần 2..." — gom thành 1 lần ghi file duy nhất.
5. Sau mỗi lần ghi file, chạy `go build ./...` để xác nhận không có lỗi compile — báo kết quả cho user.

---

## Khi không chắc chắn

Hỏi thay vì giả định. Đặc biệt khi:

- Không rõ UC nào áp dụng.
- Có 2+ cách implement và không có convention sẵn.
- Task yêu cầu thay đổi file docs hoặc cấu trúc thư mục.

---

## Quy tắc thiết kế Validation & Anti-Spam (BẮT BUỘC)

1. **Luôn đặt các kiểm tra Validation (CPU-bound) ở ĐẦU hàm**: Các kiểm tra rẻ (như độ dài `len`, format, regex, `ValidatePasswordComplexity`, so sánh chuỗi) phải được chạy TRƯỚC khi gọi Database, TRƯỚC khi tính toán nặng (Bcrypt), và TRƯỚC khi lưu Rate Limit.
2. **KHÔNG khóa user vì lỗi nhập liệu**: Không được phép gọi các hàm RateLimit, LockAccount hoặc trừ quota của người dùng nếu request của họ bị lỗi do nhập liệu sai (Validation Error). Chỉ áp dụng Rate Limit/Quota cho các request đã qua vòng kiểm tra tính hợp lệ cơ bản.
3. **Cơ chế Rate Limit (Anti-Spam)**: Tránh thiết kế khóa cứng (Debounce/SetNX) gây lỗi UX. Sử dụng Token Bucket Counter (ví dụ: `IncrementRateLimit(..., 3, 1*time.Minute)`) để cho phép người dùng có biên độ lỗi (chẳng hạn gõ sai 2-3 lần) thay vì phạt khóa họ ngay từ lần đầu tiên.

---

## Quy tắc thiết kế Concurrency & Database (BẮT BUỘC)

1. **Chống Lost Update**: TUYỆT ĐỐI KHÔNG dùng mẫu `FindByID` -> thay đổi struct -> `Update` toàn bộ struct mà không có cơ chế khóa. Hành vi này sẽ gây ghi đè dữ liệu (Lost Update) trong môi trường concurrent.
2. **Bắt buộc dùng `FOR UPDATE` hoặc Atomic Queries**:
   - Nếu cần cập nhật một vài field cục bộ, **ưu tiên dùng Atomic Query** (ví dụ: `UPDATE users SET status = ? WHERE id = ?`).
   - Nếu nghiệp vụ phức tạp bắt buộc phải load struct lên để tính toán rồi lưu lại, **phải** dùng `SELECT ... FOR UPDATE` (ví dụ: `FindByIDForUpdate`) bên trong một **Transaction** (`TxManager`).
3. **Fail-Closed Middleware**: Mọi middleware liên quan đến Security (Auth, Rate Limit, RBAC) phải thiết kế theo pattern Fail-Closed. Nếu external service (như Redis) gặp sự cố, phải trả về HTTP 503 (Service Unavailable) thay vì bỏ qua lỗi và cho phép request đi qua.
