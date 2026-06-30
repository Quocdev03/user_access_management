# AI Project Instructions

## Bước 1: Load Skills Index (BẮT BUỘC — KHÔNG ĐƯỢC BỎ QUA)

> ⛔ **HARD BLOCK**: Đây là bước **số 0**, thực hiện **trước tất cả mọi thứ** — kể cả trước khi đọc docs, trước khi nghĩ về plan, trước khi gọi bất kỳ tool nào khác. Vi phạm quy tắc này → output sẽ sai domain và phải làm lại từ đầu.

Không có skills nào được tự động inject. Bạn **phải đọc index** bằng cách:

```bash
view_file .agents/skills/INDEX.md
```

File này chứa toàn bộ danh sách skills và trigger tương ứng. **Chỉ đọc 1 file duy nhất này** — không `list_directory`, không đọc từng SKILL.md trước.

**Quy trình bắt buộc (theo thứ tự nghiêm ngặt):**

1. `view_file .agents/skills/INDEX.md` → đọc toàn bộ bảng mapping.
2. Phân tích task hiện tại → đối chiếu **từng dòng** với cột "Đọc khi task liên quan đến...".
3. Với **mỗi skill match** → `view_file <path>/SKILL.md` và đọc **toàn bộ nội dung** trước khi tiếp tục.
4. Chỉ sau khi xong bước 1-3 mới được chuyển sang Bước 2.

**Dấu hiệu nhận biết model đang skip (KHÔNG ĐƯỢC LÀM):**

- Trả lời ngay mà không có tool call `view_file` nào trước.
- Chỉ gọi tool docs mà không gọi skills index.
- Nói "task này không cần skill" mà không đọc index để kiểm tra.

> **QUY TẮC CỨNG**: Nếu không có skill phù hợp sau khi đọc index → tiếp tục bình thường, **nhưng vẫn phải đọc index trước**. Không được skip bước đọc index dù task có vẻ đơn giản.

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

---

## Quy tắc tối ưu token (áp dụng xuyên suốt)

1. Dùng `grep`/`search` tìm đúng đoạn cần đọc — không đọc toàn bộ file lớn.
2. Khi sửa file → dùng replace chính xác đoạn cần sửa, không rewrite toàn file.
3. Gom nhiều thay đổi nhỏ trong cùng 1 file thành 1 lần sửa duy nhất.
4. Không liệt kê lại nội dung docs trong response — dùng làm context nội bộ.
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
