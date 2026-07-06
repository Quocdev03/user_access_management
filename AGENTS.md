# AI Project Instructions (Optimized)

> **MỤC TIÊU**: Tuân thủ tuyệt đối cấu trúc dự án, tối ưu số lượng token, dùng đúng tool, và đảm bảo chất lượng code (Clean Architecture, Concurrency, Security).

## 1. KHỞI TẠO & ĐỌC CONTEXT (BẮT BUỘC)

- **Đọc Skills (Bước 0)**: Đọc danh sách các `SKILL.md` trước. Chỉ dùng tool `view_file` đọc nội dung các skill thực sự liên quan đến task hiện tại. Chỉ đọc toàn bộ skills khi user yêu cầu audit hoặc thay đổi ảnh hưởng toàn project.
- **Đọc Docs (Ưu tiên theo task)**:
    - `01-overview.md`: Luôn đọc.
    - `02-architecture.md`: Khi tạo/sửa file trong `internal/`, `pkg/`, `cmd/`.
    - `03-coding-conventions.md`: Khi viết/refactor code.
    - `04-api-design.md`: Khi làm việc với endpoint, handler, response format.
    - `05-database-design.md`: Khi thao tác DB, model, repo.
    - `06-authentication-flow.md`: Auth, token, middleware, RBAC.
    - `07-use-cases.md`: Nghiệp vụ, user flow.
    - `08-environment-setup.md`: Docker, CI/CD, config.

## 2. QUY TRÌNH IMPLEMENT (WORKFLOW)

1. **Lên Plan**: Nếu task > 1 file hoặc tính năng mới, tạo outline ngắn (file cần sửa, Use Case, code tái sử dụng) & đợi user duyệt. Task nhỏ/fix bug thì làm luôn.
2. **Scan Codebase**: Trước khi tạo file, ưu tiên dùng search/symbol tool của IDE để tìm code tương tự. Chỉ dùng terminal search (như `grep_search`) khi IDE không có tool phù hợp, nhằm tránh duplicate:
    - Handler: Tìm `func.*Handler\b` (`internal/handler/`)
    - Service: Tìm `func New.*Service\b` (`internal/service/`)
    - Repo: Tìm `func New.*Repository\b` (`internal/repository/`)
    - Entity/DTO: Tìm Tên Entity, `type.*Request struct`, v.v.
3. **Verification**:
    - Sau khi thay đổi Go source (\*.go), chạy `cmd /c "go build ./..."`. Nếu chỉ sửa docs, config, comment thì không cần build.
    - Nếu có unit test cho phần bị sửa, hãy chạy các test liên quan. Không chạy toàn bộ test suite nếu chỉ thay đổi nhỏ, trừ khi được yêu cầu.
    - **Rollback/Fix**: Nếu build hoặc test fail, tự phân tích và fix. Nếu lỗi phức tạp vượt quá phạm vi task, báo lại cho user và xin ý kiến, KHÔNG tự ý sửa lan man.
4. **Cập nhật Docs**: BẮT BUỘC cập nhật `PROGRESS.md` khi:
    - Hoàn thành một Feature, Use Case mới.
    - Có thay đổi lớn về API, Database, Architecture.
    - Cập nhật tài liệu quy mô lớn, mang tính toàn cục (VD: Tích hợp Swagger, bổ sung comment/docs cho toàn bộ module).
    *Không cập nhật* nếu chỉ là refactor code nhỏ lẻ, fix bug lặt vặt, sửa lỗi typo hoặc format code thông thường.

## 3. QUY TẮC KIẾN TRÚC & CODE

- **Clean Architecture**: `Handler → Service → Repository`. Không nhảy cóc, không gọi ngược. Tuân thủ `02-architecture.md` và convention.
- **Backward Compatibility**: Không thay đổi public API, interface, response format hoặc database schema nếu user không yêu cầu rõ ràng.
- **Comment**: Khi sửa/xóa code, BẮT BUỘC cập nhật hoặc xóa comment cũ liên quan để tránh gây hiểu lầm.
- **Error Handling**:
    - _Lỗi Business_: Dùng `apperror` (VD: `return apperror.ErrNotFound`). Tuyệt đối KHÔNG `fmt.Errorf`.
    - _Lỗi System_: Dùng `fmt.Errorf("...: %w", err)` để bọc lỗi từ tầng dưới.
    - _Ignore Error_: Không bỏ qua lỗi security. Thao tác best-effort thì phải log (`logger.Warn`).
- **Thứ tự Validation & Anti-Spam**:
    1. Format Check (len, regex) -> 2. Rate Limit -> 3. AuthZ (Kiểm tra quyền) -> 4. DB Read -> 5. Bcrypt/Crypto -> 6. DB Write/Tx.
       _(Dùng Token Bucket. Chỉ tính rate limit/failed attempt nếu format hợp lệ)._
- **Security**: Mọi Middleware bảo mật (Auth, Rate limit) phải Fail-Closed (trả về 503 nếu service phụ trợ như Redis sập).

## 4. DATABASE & TRANSACTION

- **Database Migration**: Không tạo migration mới hoặc thay đổi schema hiện có nếu user không yêu cầu rõ ràng.
- **Chống Lost Update**: BẮT BUỘC dùng `FOR UPDATE` (VD: `FindByIDForUpdate(txCtx)`) khi load record để sửa. HOẶC dùng Atomic Update query trực tiếp nếu chỉ đổi trạng thái đơn giản.
- **TxManager**: Mọi thao tác gồm nhiều bước phải nằm trong `s.txManager.RunInTx(ctx, func(txCtx) {...})`.
- **Scope Context**:
    - Truyền `txCtx` vào các repo call bên trong transaction.
    - KHÔNG gọi repo bằng `ctx` gốc khi đang ở trong `RunInTx`.
    - Redis call đặt ngoài MySQL transaction. KHÔNG lồng các `RunInTx` chồng chéo.

## 5. HÀNH VI CỦA AI (TỐI ƯU OUTPUT)

- **Sử dụng Tool**: Ưu tiên tool chuyên biệt (`grep_search`, `list_dir`, `view_file`). Không lạm dụng lệnh bash (`grep`, `cat`, `ls`) khi đã có tool. Không dùng terminal để đọc file nếu IDE đã có file viewer/search tool. Dùng terminal chủ yếu cho build, test và git.
- **Sửa Code (Targeted Edits & Scope Control)**:
    - **Scope Control**: Chỉ sửa những file bắt buộc cho task. Tuyệt đối không "tiện tay" refactor, rename, move, reformat hoặc đụng chạm tới file/code không liên quan (No opportunistic refactoring).
    - Không tạo file mới nếu có thể mở rộng file hiện có. Ưu tiên sửa code hiện tại thay vì tạo abstraction mới.
    - Chỉ thay đổi chính xác đoạn code cần thiết. TUYỆT ĐỐI KHÔNG rewrite toàn bộ file nếu chỉ sửa 1 đoạn nhỏ.
    - Giữ nguyên định dạng: Không thay đổi formatting của phần code không đụng tới. Không reorder imports trừ khi bắt buộc.
- **Chống Truncate**: Output phải đầy đủ 100%. KHÔNG để lại `...` hoặc `// rest of code`. Code dài phải ghi thẳng vào file, không paste toàn bộ lên chat.
- **Giao tiếp**: Nói ngắn gọn. Không giải thích code hiển nhiên. Với quyết định kỹ thuật quan trọng, thêm comment ngắn vào code. Khi không chắc chắn về nghiệp vụ/cấu trúc, BẮT BUỘC hỏi lại user trước khi tự quyết định.

## 6. SIMPLICITY FIRST (TRÁNH OVER-ENGINEERING)

- **Consistency First**: Ưu tiên làm giống pattern đã tồn tại trong project hơn là áp dụng một pattern mới dù pattern mới có thể "đẹp" hơn. Giữ code nhất quán với codebase.
- **Prefer the simplest implementation**: Luôn ưu tiên giải pháp code đơn giản, trực quan nhất.
- **No premature abstraction**: Không tạo Interface/Abstraction trừ khi logic đó được tái sử dụng ít nhất 2 lần.
- **No one-time helpers**: Không tạo Helper function nếu nó chỉ được dùng đúng 1 lần (hãy viết thẳng logic inline).
- **Keep files grouped**: Không tách logic lắt nhắt ra nhiều file nhỏ trừ khi nó thực sự cải thiện khả năng bảo trì.
- **Avoid premature optimization**: Tránh tối ưu hóa hiệu suất quá sớm khi chưa có bottleneck thực tế.
- **Minimize changes**: Giữ lượng code thay đổi ở mức tối thiểu nhất có thể.
- **Reuse first**: BẮT BUỘC tìm và tái sử dụng code/utils/struct hiện có trước khi định tự viết mới.
- **No new dependencies**: Không thêm dependency/package mới nếu thư viện hiện có đã giải quyết được.
