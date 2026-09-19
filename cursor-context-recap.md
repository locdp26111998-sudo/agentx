# agentx — Context recap (Cursor)



> Cập nhật sau GĐ5 GoClaw Dashboard + GĐ6 GoClaw Engine adapter.



---



## 1. Cấu trúc thư mục chính



```

agentx/

├── cmd/server/main.go

├── configs/

│   ├── agents/          # demo.yaml, spa-hoa.yaml, phong-kham-minh.yaml

│   └── goclaw.yaml      # base_url, api_key, user_id

├── internal/

│   ├── admin/           # frontend /admin + API admin

│   ├── engine/          # claudecode.go, goclaw.go, selector.go

│   ├── goclaw/          # dashboard proxy GET /api/goclaw/agents

│   ├── guard/

│   ├── inbox/           # messenger webhook

│   ├── router/          # multi-agent routing theo channels

│   ├── store/           # SQLite sessions

│   └── ticket/

└── scripts/             # run/stop/test-webhook.ps1

```



---



## 2. Đã xong



| Giai đoạn | Nội dung |

|-----------|----------|

| GĐ1 | Webhook giả → Claude Code → ticket YAML |

| GĐ2 | Memory `--resume`, SQLite store |

| GĐ3 | Guard, Messenger, Admin frontend (3 màn) |

| Router | Multi-agent theo `channels.messenger.page_id` |

| GĐ5 | GoClaw Dashboard `/admin#/goclaw` |

| GĐ6 | Engine adapter GoClaw (`POST /v1/chat/completions`, model `goclaw:{key}`) |



---



## 3. Luồng xử lý hiện tại



```

POST /webhook {sender_id, text, page_id}

  → router.Resolve(page_id) → ticket.Compose

  → engine.Selector (claude-code | goclaw theo YAML)

  → guard.Check → {"reply", "agent_id"}



Messenger POST → async → router → selector → guard → Send API

```



**Engine selector:**

- `engine: claude-code` (mặc định) → Claude Code CLI + `--resume`

- `engine: goclaw` + `goclaw_agent_key: ho-tro` → GoClaw HTTP API



**Agent dùng GoClaw:** `phong-kham-minh` (page_id `222222222`, key `ho-tro`)



---



## 4. Config GoClaw



`configs/goclaw.yaml`:

```yaml

goclaw:

  base_url: "https://senagent.io.vn"

  api_key: "goclaw_..."

  user_id: "system"

```



Header bắt buộc: `Authorization: Bearer {api_key}` + `X-GoClaw-User-Id: {user_id}`



---



## 5. Việc tiếp theo (chưa làm)



1. **E2E Messenger** — verify Meta webhook, tin thật Page Omnipick1

2. **Bảo mật token** — `page_access_token` ra env / file gitignored

3. **Adapter Codex** — engine thứ 3 qua interface `Engine`

4. **Kênh Zalo OA** — adapter inbox + routing `channels.zalo.oa_id`

5. **Guard nâng cao** — regex, PII, rate limit

6. **Session theo agent** — scope `sender_id` + `agent_id` trong SQLite



---



## 6. Lệnh test nhanh



```powershell

& "G:\Agent X\agentx\scripts\run-server.ps1"



# Claude Code (spa-hoa)

& "G:\Agent X\agentx\scripts\test-webhook.ps1" -PageID "111111111" -Text "Spa mo cua luc may gio?"



# GoClaw (phong-kham-minh)

& "G:\Agent X\agentx\scripts\test-webhook.ps1" -PageID "222222222" -Text "Phong kham mo cua luc may gio?"



# Admin + GoClaw dashboard

# http://127.0.0.1:8090/admin#/goclaw

```



---



*Cập nhật: 2026-09-18*


