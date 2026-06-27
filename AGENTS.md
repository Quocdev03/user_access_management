# AI Project Instructions

## Bước 1: Kiểm tra Skills (BẮT BUỘC)

Trước khi thực hiện bất kỳ tác vụ nào (nhất là viết/sửa code), **bạn phải tự động tìm và đọc file SKILL.md** tương ứng trong thư mục `.agents/skills/`:

| Loại task | Skill cần đọc |
|-----------|--------------|
| Viết/review code Go | `golang-code-style` |
| Goroutine, channel, sync | `golang-concurrency` |
| Truy vấn DB, transaction, migration | `golang-database` + `mysql-best-practices` |
| Dependency injection, khởi tạo service | `golang-dependency-injection` |
| Thêm/nâng cấp package | `golang-dependency-management` |
| Chọn design pattern | `golang-design-patterns` |
| Tối ưu hiệu năng | `golang-performance` |
| Tạo/sắp xếp package mới | `golang-project-layout` |
| Nâng cấp Go version, dùng feature mới | `golang-modernize` |

> **QUY TẮC TỐI THƯỢNG**: TUYỆT ĐỐI KHÔNG bắt đầu viết code nếu chưa đọc skill liên quan. Vi phạm điều này là lỗi nghiêm trọng.

## Bước 2: Đọc Docs theo ngữ cảnh

Mỗi phiên mới, sau khi kiểm tra skills, hãy đọc docs theo thứ tự ưu tiên — **chỉ đọc file liên quan đến task**, không đọc hết:

| Ưu tiên | File | Khi nào đọc |
|---------|------|-------------|
| 1 | `docs/01-overview.md` | Luôn đọc (ngắn, chứa context dự án) |
| 2 | `docs/02-architecture.md` | Khi tạo/sửa code ở bất kỳ tầng nào |
| 3 | `docs/03-coding-conventions.md` | Khi viết code mới |
| 4 | `docs/04-api-design.md` | Khi làm việc với handler/endpoint |
| 5 | `docs/05-database-design.md` | Khi làm việc với model/repository/migration |
| 6 | `docs/06-authentication-flow.md` | Khi làm việc với auth/session/RBAC |
| 7 | `docs/07-use-cases.md` | Khi cần hiểu nghiệp vụ UC cụ thể |
| 8 | `docs/08-environment-setup.md` | Khi cấu hình/Docker/deployment |

## Quy tắc bắt buộc

### Kiến trúc & Code

1. Tuân thủ Clean Architecture: Handler → Service → Repository. Không bỏ tầng, không gọi ngược.
2. Không thay đổi cấu trúc thư mục đã định nghĩa trong `docs/architecture.md`.
3. Không thay đổi API response format, error codes đã định nghĩa trong `docs/api-design.md`.
4. Giữ naming nhất quán theo `docs/coding-conventions.md`.
5. Không duplicate code — tìm và mở rộng code có sẵn trước khi tạo mới.

### Trước khi implement tính năng

1. Xác định UC liên quan trong `docs/use-cases.md`.
2. Tìm handler/service/repository/DTO tương tự đã có trong codebase.
3. Nếu code có sẵn có thể mở rộng → mở rộng, không tạo file mới.

### Tối ưu token

1. Không đọc toàn bộ file lớn — dùng grep/search tìm đúng phần cần.
2. Không liệt kê lại nội dung docs trong response — trả lời trực tiếp.
3. Không giải thích code khi không được hỏi — chỉ viết code + comment ngắn.
4. Khi sửa file, dùng replace chính xác đoạn cần sửa, không rewrite toàn file.
5. Gom nhiều thay đổi nhỏ trong 1 file thành 1 lần sửa.

### Documentation

- Cập nhật docs khi thay đổi: thêm endpoint, thêm bảng, đổi luồng xử lý.
- Không cập nhật docs cho thay đổi nhỏ (fix bug, refactor nội bộ).

## Khi không chắc chắn

Hỏi thay vì giả định.
