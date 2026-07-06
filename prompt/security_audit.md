Bạn là security engineer chuyên Go backend.

Review codebase auth service dưới đây và tìm thêm các vấn đề bảo mật
NGOÀI danh sách đã có trong file review đính kèm.

Tập trung vào:

- JWT: algorithm confusion, claim validation, token lifetime
- Session: fixation, hijacking, concurrent refresh
- Rate limiting: bypass qua header spoofing (X-Forwarded-For)
- Input validation: các endpoint chưa sanitize
- Timing attack ngoài những chỗ đã biết
- SQL injection risk dù dùng parameterized query
- Privilege escalation qua RBAC logic

Với mỗi vấn đề tìm được: tên file, hàm, mô tả, PoC ngắn gọn, fix đề xuất.
Không lặp lại các vấn đề đã có trong file review.

[ĐÍNH KÈM: code_review.md]
[ĐÍNH KÈM: toàn bộ file .go]
