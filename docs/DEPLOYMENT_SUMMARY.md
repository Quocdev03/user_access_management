> [!IMPORTANT]
> **LƯU Ý DÀNH CHO DEVELOPER (AI & HUMAN):**
> Các tài liệu thiết kế này mang tính chất là **KHUNG ĐỊNH HƯỚNG (Framework / Guidelines)**.
> KHÔNG ĐƯỢC áp dụng một cách rập khuôn, máy móc hoặc sao chép hoàn toàn 100%.
> Tùy thuộc vào bối cảnh thực tế của task, bạn phải linh hoạt tùy biến (ví dụ: dùng Atomic Query, Pessimistic Locking FOR UPDATE cho Concurrency, hoặc cấu trúc lại struct).

# Kiến trúc Hệ thống & Báo cáo Triển khai
*(User Access Management - UAM)*

Tài liệu này giới thiệu tổng quan về bức tranh toàn cảnh hạ tầng (Infrastructure) và cấu trúc triển khai thực tế (Production Deployment) của dự án **User Access Management (UAM)**. 

Dự án được xây dựng dựa trên nguyên tắc **Clean Architecture**, tối ưu hóa hiệu suất (High Performance), tính bảo mật (Security), và hoàn toàn sẵn sàng cho môi trường đám mây (Cloud-Native).

---

## 1. Công nghệ Lõi (Tech Stack)
- **Ngôn ngữ:** Golang (Go 1.22+)
- **Mô hình thiết kế:** Clean Architecture (Handler -> Service -> Repository)
- **Bảo mật:** 
  - Mã hóa mật khẩu một chiều (Bcrypt).
  - Quản lý phiên đăng nhập kép: Access Token (thời gian sống ngắn) & Refresh Token (thời gian sống dài) bằng JWT chuẩn hóa.
  - Thu hồi Token (Blacklisting) thông qua Redis.
  - Rate Limiting chống Brute-Force & Spam OTP.

## 2. Kiến trúc Hạ tầng Đám mây (Cloud Infrastructure)

Hệ thống được thiết kế theo mô hình Micro-services phân tán, sử dụng các dịch vụ Managed Services hiện đại nhằm đảm bảo độ tin cậy và khả năng mở rộng tự động.

### 2.1. API Service (Golang)
- **Nền tảng vận hành:** **Render (Web Service)**
- **CI/CD:** Pipeline tự động (Automated CI/CD). Mỗi thay đổi trên nhánh `main` của GitHub được Render tự động nhận diện, đóng gói (Containerize) qua Dockerfile, và triển khai không gián đoạn (Zero-Downtime Deployment).
- **Cấu hình (Configuration):** Toàn bộ Secret Keys, JWT Signatures và Database Credentials được tiêm qua **Environment Variables** để đảm bảo tuyệt đối an toàn bảo mật, thay thế hoàn toàn file `.env`.

### 2.2. Cơ sở dữ liệu Quan hệ (Primary Database)
- **Nền tảng cung cấp:** **Aiven (Managed MySQL Cloud)**
- **Đặc tính kỹ thuật:**
  - Dữ liệu được lưu trữ an toàn trên đám mây với cơ chế sao lưu tự động.
  - Chuẩn mã hóa truyền tải bắt buộc **SSL/TLS**. 
  - Mã nguồn (Go) được refactor chuyên biệt sử dụng `mysql.Config` để tự động đàm phán kết nối bảo mật (skip-verify) mà không phá vỡ cấu trúc Clean Code.

### 2.3. Caching & Session Store (In-Memory Database)
- **Nền tảng cung cấp:** **Render (Managed Redis)**
- **Đặc tính kỹ thuật:**
  - Tách biệt hoàn toàn với MySQL để giảm tải (Offloading) cho DB chính.
  - Vận hành hoàn toàn qua mạng nội bộ (Internal Network) tốc độ cực cao, không bị lộ cổng (Port) ra ngoài Internet.
  - Đảm nhiệm: Lưu trữ trạng thái Rate Limit, phiên đăng nhập, và OTP Tạm thời.

### 2.4. Hệ thống Gửi Email (Mailing Service)
- **Nền tảng cung cấp:** **Resend (SMTP)**
- **Đặc tính kỹ thuật:**
  - Cung cấp dịch vụ gửi Email Transactional (Mã OTP Xác thực, Quên Mật khẩu) với độ trễ thấp và tỷ lệ vào Inbox cao.
  - Quá trình phát triển cục bộ (Local) sử dụng **Mailpit** qua Docker Compose để giả lập (Mocking), tránh rò rỉ email khi test.

### 2.5. Domain & Routing
- **Tên miền:** Cấp phát và quản lý qua **Namecheap**.
- **DNS & SSL:** Được trỏ thông qua bản ghi CNAME về Load Balancer của Render. Chuyển hướng HTTPS (SSL/TLS Certificate) được tự động gia hạn và mã hóa 2 chiều ngay từ phía máy khách (Client).

---

## 3. Sơ đồ Luồng Dữ liệu (Data Flow)

```mermaid
graph LR
    Client((Client/Browser)) -->|HTTPS / CNAME Namecheap| RenderAPI[Golang API Service (Render)]
    RenderAPI -->|Read/Write (TLS)| AivenMySQL[(Aiven MySQL)]
    RenderAPI -->|Cache/RateLimit (Internal)| RenderRedis[(Redis Cluster)]
    RenderAPI -->|Send OTP/Verification| Resend[Resend SMTP]
    Resend -->|Delivery| UserEmail((User Inbox))
```

*Đây là một dự án chứng minh năng lực thiết kế hệ thống chuyên nghiệp, khả năng làm việc với đa dịch vụ đám mây (Cloud Providers) và tuân thủ chặt chẽ các chuẩn mực bảo mật tiên tiến nhất của Golang.*
