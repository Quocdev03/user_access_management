# AI Project Instructions

## Bước 1: Load Skills Index (BẮT BUỘC, chạy đầu mỗi phiên)

Không có skills nào được tự động inject. Bạn **phải đọc index** bằng cách:

```
view_file .agents/skills/INDEX.md
```

File này chứa toàn bộ danh sách skills và trigger tương ứng. **Chỉ đọc 1 file duy nhất này** — không `list_directory`, không đọc từng SKILL.md trước.

**Quy trình bắt buộc:**

1. `view_file .agents/skills/INDEX.md` → đọc bảng mapping.
2. Phân tích task hiện tại → đối chiếu với cột "Đọc khi task liên quan đến...".
3. Với mỗi skill match → `view_file <path>` SKILL.md tương ứng **trước khi làm bất cứ việc gì**.

> **QUY TẮC**: Không được bắt đầu viết code hoặc sửa file nếu chưa đọc SKILL.md của domain liên quan. Nếu không có skill phù hợp → tiếp tục bình thường, không đọc skill không liên quan để tiết kiệm token.

---

## Bước 2: Đọc Docs theo ngữ cảnh

Sau khi đọc skills, đọc docs theo thứ tự ưu tiên — **chỉ đọc file liên quan đến task**.

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

---

## Quy tắc tối ưu token (áp dụng xuyên suốt)

1. Dùng `grep`/`search` tìm đúng đoạn cần đọc — không đọc toàn bộ file lớn.
2. Khi sửa file → dùng replace chính xác đoạn cần sửa, không rewrite toàn file.
3. Gom nhiều thay đổi nhỏ trong cùng 1 file thành 1 lần sửa duy nhất.
4. Không liệt kê lại nội dung docs trong response — dùng làm context nội bộ.
5. Không giải thích code khi không được hỏi — chỉ viết code + comment ngắn inline.

---

## Quy tắc documentation

- **Cập nhật docs** khi: thêm endpoint mới, thêm bảng DB, đổi luồng xử lý chính.
- **Không cập nhật docs** khi: fix bug nhỏ, refactor nội bộ không đổi interface.

---

## Khi không chắc chắn

Hỏi thay vì giả định. Đặc biệt khi:

- Không rõ UC nào áp dụng.
- Có 2+ cách implement và không có convention sẵn.
- Task yêu cầu thay đổi file docs hoặc cấu trúc thư mục.
