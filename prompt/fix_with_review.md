Bạn là senior Go engineer chuyên về authentication system.

Tôi có một Go backend xử lý auth/user management với stack: Gin, sqlx (MySQL), Redis, JWT (golang-jwt/v5), Zap logger, Viper config.

Dưới đây là toàn bộ source code và file review liệt kê 22 vấn đề đã được phân tích.

Yêu cầu:

1. Đọc file review trước để hiểu context
2. Fix theo thứ tự: 🔴 Critical → 🟠 Security → 🟡 Quality
3. Với mỗi file được sửa, output file Go hoàn chỉnh có thể dùng ngay
4. Thêm comment ngắn tại dòng thay đổi, format: // [FIX] lý do
5. Không thay đổi business logic, không đổi tên hàm/interface
6. Không giải thích dài dòng — chỉ plan 3 dòng rồi đưa code

[ĐÍNH KÈM: code_review.md]
[ĐÍNH KÈM: toàn bộ file .go]
