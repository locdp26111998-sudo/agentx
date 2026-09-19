# agentx — Recap context (Chat 4)

> File handoff mở chat mới. Cập nhật: **2026-09-19** — sau deploy VPS production (Nginx + SSL) và xác nhận Admin HTTPS public.

---

## 1. agentx là gì

- App Go (`module agentx`) — **quản đốc** điều phối agent CLI / GoClaw trả lời khách qua kênh (Messenger, sau này Zalo…).
- **Agent = config YAML**, không code lại từng khách.
- Hai loại engine cùng interface `Engine`:
  - **claude-code** — adapter dày (CLI `claude -p`, JSON `result` / `session_id`, `--resume`).
  - **goclaw** — adapter mỏng (`POST /v1/chat/completions`, model `goclaw:{agent_key}`).

**6 khối kiến trúc:** inbox → router → ticket → store → engine → guard.

---

## 2. Cấu trúc thư mục (repo)

```
agentx/
├── cmd/server/main.go          # HTTP server :8090, wire toàn bộ khối
├── configs/
│   ├── agents/
│   │   ├── *.yaml.example      # commit (mẫu)
│   │   └── *.yaml              # gitignore — secrets local/VPS
│   └── goclaw.yaml.example
│   └── goclaw.yaml             # gitignore
├── data/                       # gitignore — SQLite
├── internal/
│   ├── admin/                  # /admin SPA + API + SSE logs
│   │   └── web/                # index.html, app.js, style.css
│   ├── engine/                 # engine.go, claudecode.go, goclaw.go, selector.go
│   ├── goclaw/                 # GET /api/goclaw/agents (proxy dashboard)
│   ├── guard/
│   ├── inbox/                  # POST /webhook/messenger (Meta)
│   ├── router/                 # route theo channels.messenger.page_id
│   ├── store/                  # SQLite sessions (--resume)
│   └── ticket/
├── scripts/
│   ├── run-server.ps1
│   ├── stop-server.ps1
│   └── test-webhook.ps1
├── .cursor/rules/agentx.mdc
├── cursor-context-recap.md       # recap cũ (GĐ6 trước deploy)
├── recap-agentx-context-chat-4.md  # file này
├── go.mod / go.sum
└── .gitignore
```

**GitHub (public):** `https://github.com/locdp26111998-sudo/agentx`  
**Gitignore secrets:** `data/`, `*.db`, `.env*`, `configs/goclaw.yaml`, `configs/agents/*.yaml`

---

## 3. File / endpoint chính đã có

| Thành phần | File / route |
|------------|----------------|
| Server entry | `cmd/server/main.go` |
| Webhook test JSON | `POST /webhook` (body: `sender_id`, `text`, `page_id`) |
| Messenger Meta | `POST/GET /webhook/messenger` |
| Admin UI | `GET /admin`, SSE logs |
| GoClaw list (read-only) | `GET /api/goclaw/agents` |
| Engine Claude | `internal/engine/claudecode.go` |
| Engine GoClaw | `internal/engine/goclaw.go` |
| Chọn engine | `internal/engine/selector.go` + field `engine` trong YAML |
| Router multi-agent | `internal/router/router.go` |
| Tests | `*_test.go` (router, engine, goclaw, guard) |

---

## 4. Trạng thái giai đoạn

| Giai đoạn | Nội dung | Trạng thái |
|-----------|----------|------------|
| GĐ1 | Webhook → Claude Code → ticket | ✅ |
| GĐ2 | SQLite store, `--resume` session | ✅ |
| GĐ3 | Guard, Messenger inbox, Admin SPA (3 màn) | ✅ |
| Router | Multi-agent theo `page_id` | ✅ |
| GĐ5 | GoClaw Dashboard `/admin#/goclaw` | ✅ |
| GĐ6 | GoClaw engine adapter + selector | ✅ |
| **Deploy VPS** | Clone, build, systemd, Nginx, Let's Encrypt | ✅ **HTTP/HTTPS public** |
| **E2E Messenger production** | Token thật VPS + Meta webhook | ⏳ **Chưa** |
| GĐ4 (frontend thương hiệu) | — | Chưa |
| Codex / opencode engine | — | Chưa |
| Zalo OA | — | Chưa |

---

## 5. Luồng xử lý (runtime)

```
POST /webhook/messenger (Meta)
  → inbox parse → router.Resolve(messenger, page_id)
  → ticket.Compose → engine.Selector.Run (claude-code | goclaw)
  → guard.Check → gửi reply Messenger API
```

**Agent mẫu (local dev / example):**

| Agent | engine | page_id (example/dev) | Ghi chú |
|-------|--------|------------------------|---------|
| Mai (`demo`) | claude-code | `679362898589478` (local demo) | Fanpage Omnipick1 |
| Spa Hoa | claude-code | `111111111` | test script |
| Phòng khám Minh | goclaw + `goclaw_agent_key: ho-tro` | `222222222` | GoClaw trên senagent.io.vn |

**GoClaw config** (`configs/goclaw.yaml`):

- `base_url: https://senagent.io.vn`
- `user_id: system`
- API key dạng `goclaw_...` (lưu local/VPS, không commit)

Header: `Authorization: Bearer`, `X-GoClaw-User-Id`.

---

## 6. VPS / domain / SSL (production)

| Mục | Giá trị |
|-----|---------|
| Nhà cung cấp | 123HOST Cloud VPS |
| Hostname panel | Agentx |
| IP | `103.97.126.207` |
| OS | Ubuntu 22.04 LTS |
| SSH | `ssh -p 2018 root@103.97.126.207` |
| Hostname SSH | `srv-zl2rr` |
| App path | `/opt/agentx` |
| Binary | `/opt/agentx/bin/agentx` (build `go build -o bin/agentx ./cmd/server`) |
| Go trên VPS | 1.25+ (cần cho `modernc.org/sqlite`) |
| Service | `systemctl status agentx` — listen **127.0.0.1:8090** |
| Domain | **`agentx.senagent.io.vn`** (A → IP VPS) |
| Nginx | `/etc/nginx/sites-available/agentx` → proxy → `127.0.0.1:8090` |
| SSL | Let's Encrypt, cert name `agentx.senagent.io.vn` (certbot) |
| Admin public | **https://agentx.senagent.io.vn/admin** ✅ (đã xác nhận 2026-09-19) |
| Webhook URL (Meta) | **https://agentx.senagent.io.vn/webhook/messenger** (chưa cấu hình Meta) |

**Lưu ý deploy đã gặp:**

- Paste heredoc config Nginx trên SSH Windows dễ tạo file **rỗng** → dùng `base64 -d` một dòng hoặc kiểm tra `cat` sau khi ghi.
- Config SSL Nginx lỗi/thiếu file → Nginx không listen 80 → `curl` code `000`; sửa bằng config HTTP trước, rồi `certbot install --cert-name agentx.senagent.io.vn --nginx --redirect`.
- 123HOST trả lời ticket: **mặc định VPS không bật tường lửa cloud** — lỗi ngoài không vào ban đầu chủ yếu do **Nginx**, không phải ticket mở port.
- **Cloudflare quick tunnel** (`cloudflared --url http://127.0.0.1:8090 --protocol http2`) dùng được khi QUIC/UDP 7844 bị chặn; chỉ để test, không thay domain cố định.
- VPS **không có** `ufw` (`command not found`).

**systemd (tham khảo):** unit `agentx.service`, WorkingDirectory `/opt/agentx`, ExecStart `/opt/agentx/bin/agentx`.

---

## 7. Môi trường dev (Windows)

```powershell
& "G:\Agent X\agentx\scripts\run-server.ps1"

# Webhook test
& "G:\Agent X\agentx\scripts\test-webhook.ps1" -PageID "111111111" -Text "..."
& "G:\Agent X\agentx\scripts\test-webhook.ps1" -PageID "222222222" -Text "..."

# Admin local
# http://127.0.0.1:8090/admin
# http://127.0.0.1:8090/admin#/goclaw
```

Claude Code trên máy dev: OAuth Pro, **không** dùng `--bare`.

---

## 8. Nợ kỹ thuật / việc còn lại

### Bắt buộc cho Fanpage production

1. **Copy token Messenger thật** lên VPS: `configs/agents/demo.yaml` (và agent khác nếu dùng) — `page_access_token`, `page_id`, `verify_token`.
2. **`systemctl restart agentx`** trên VPS.
3. **Meta Developer Console:** Callback URL `https://agentx.senagent.io.vn/webhook/messenger`, verify token khớp YAML, subscribe events, gắn Page.
4. **E2E:** nhắn Fanpage → log SSE admin + reply.

### Bảo mật / vận hành

- Token Page / GoClaw API key chỉ trên VPS hoặc env, không commit.
- `agentx` bind localhost + Nginx reverse proxy — ổn; cân nhắc auth `/admin` sau.
- Gia hạn SSL: certbot timer (Let's Encrypt ~90 ngày).
- GitHub Actions auto-deploy: **chưa làm**.
- Script `deploy/` chuẩn hóa: **chưa làm**.
- Ghi lại Nginx SSL final trong repo (template): **chưa** (đang chỉ trên VPS).

### Sản phẩm (lộ trình rule)

- Guard nâng cao (regex, PII, rate limit).
- Session scope `sender_id` + `agent_id`.
- Adapter Codex / opencode.
- Kênh Zalo OA.
- Frontend thương hiệu GĐ4.

### Admin UI

- Trạng thái agent **Offline** trên dashboard = chưa phiên Messenger hoạt động, **không** có nghĩa VPS down.

---

## 9. Bước tiếp theo (chat mới — ưu tiên)

1. Hướng dẫn copy `demo.yaml` từ Windows → VPS (scp / nano / heredoc cẩn thận).
2. Cấu hình Meta webhook + verify.
3. Test tin thật Page **Omnipick1** (`page_id` demo local).
4. (Tuỳ chọn) Đồng bộ `spa-hoa` / `phong-kham-minh` với Page thật.
5. (Tuỳ chọn) CI deploy, backup `data/agentx.db`, monitoring.

---

## 10. Lệnh VPS hay dùng

```bash
cd /opt/agentx
systemctl status agentx
systemctl restart agentx
journalctl -u agentx -n 50 --no-pager

nginx -t && systemctl reload nginx
curl -I https://agentx.senagent.io.vn/admin
curl -I http://127.0.0.1:8090/admin

certbot certificates
# Gắn lại SSL vào nginx nếu cần:
# certbot install --cert-name agentx.senagent.io.vn --nginx --redirect --non-interactive
```

---

## 11. Quy tắc làm việc (Cursor)

- Go idiomatic, comment tiếng Việt khi cần.
- Không bịa flag Claude CLI — chỉ `-p`, `--output-format json`, `--resume`.
- Không lấy GoClaw làm trung tâm kiến trúc; cả hai engine qua `Engine`.
- Việc lớn: plan trước khi code; chỉ commit khi user yêu cầu.

---

*Chat 4 kết thúc: deploy HTTPS public OK; chờ E2E Messenger + token production trên VPS.*
