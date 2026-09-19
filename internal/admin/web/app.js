(function () {
  "use strict";

  const app = document.getElementById("app");
  const pageTitle = document.getElementById("page-title");
  const navLinks = document.querySelectorAll(".nav-link");

  let logEventSource = null;

  // --- Router (hash, không reload trang) ---
  function parseRoute() {
    const hash = location.hash.slice(1) || "/";
    const parts = hash.split("/").filter(Boolean);
    if (parts[0] === "logs") return { view: "logs" };
    if (parts[0] === "goclaw") return { view: "goclaw" };
    if (parts[0] === "agent" && parts[1]) return { view: "agent", id: parts[1] };
    return { view: "list" };
  }

  function navigate(hash) {
    location.hash = hash;
  }

  function setActiveNav(route) {
    navLinks.forEach((link) => {
      const r = link.getAttribute("data-route");
      link.classList.toggle("active", r === route || (route.startsWith("/agent") && r === "/"));
    });
  }

  async function render() {
    const route = parseRoute();
    disconnectLogs();

    if (route.view === "list") {
      setActiveNav("/");
      pageTitle.textContent = "Danh sách agent";
      await renderAgentList();
    } else if (route.view === "agent") {
      setActiveNav("/");
      pageTitle.textContent = "Cấu hình agent";
      await renderAgentForm(route.id);
    } else if (route.view === "goclaw") {
      setActiveNav("/goclaw");
      pageTitle.textContent = "GoClaw";
      await renderGoclawAgents();
    } else if (route.view === "logs") {
      setActiveNav("/logs");
      pageTitle.textContent = "Log real-time";
      renderLogs();
    }
  }

  window.addEventListener("hashchange", render);
  window.addEventListener("load", render);

  const btnLogout = document.getElementById("btn-logout");
  if (btnLogout) {
    btnLogout.addEventListener("click", async function () {
      await fetch("/api/admin/logout", { method: "POST", credentials: "same-origin" });
      location.href = "/admin/login";
    });
  }

  function redirectLogin() {
    location.href = "/admin/login";
  }

  // --- API ---
  async function apiGet(path) {
    const res = await fetch(path, { credentials: "same-origin" });
    if (res.status === 401) {
      redirectLogin();
      throw new Error("unauthorized");
    }
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || res.statusText);
    }
    return res.json();
  }

  async function apiPut(path, body) {
    const res = await fetch(path, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      credentials: "same-origin",
      body: JSON.stringify(body),
    });
    if (res.status === 401) {
      redirectLogin();
      throw new Error("unauthorized");
    }
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || res.statusText);
    }
    return res.json();
  }

  // --- Màn 1: Danh sách agent ---
  async function renderAgentList() {
    app.innerHTML = '<div class="empty-state">Đang tải...</div>';
    try {
      const agents = await apiGet("/api/admin/agents");
      if (!agents.length) {
        app.innerHTML = '<div class="empty-state">Chưa có agent trong configs/agents/</div>';
        return;
      }
      const html = agents
        .map(
          (a) => `
        <div class="agent-card" data-id="${escapeAttr(a.id)}">
          <div class="agent-name">${escapeHtml(a.name || a.id)}</div>
          <div class="agent-meta">Engine: ${escapeHtml(a.engine)} · Kênh: ${escapeHtml(a.channel)} · ${a.session_count} phiên</div>
          <span class="status-badge ${a.status}">
            <span class="status-dot"></span>
            ${a.status === "online" ? "Online" : "Offline"}
          </span>
        </div>`
        )
        .join("");
      app.innerHTML = `<div class="agent-grid">${html}</div>`;
      app.querySelectorAll(".agent-card").forEach((card) => {
        card.addEventListener("click", () => navigate("#/agent/" + card.dataset.id));
      });
    } catch (e) {
      app.innerHTML = `<div class="error-banner">${escapeHtml(e.message)}</div>`;
    }
  }

  // --- Màn 2: Form cấu hình agent ---
  async function renderAgentForm(id) {
    app.innerHTML = '<div class="empty-state">Đang tải...</div>';
    try {
      const a = await apiGet("/api/admin/agents/" + encodeURIComponent(id));
      app.innerHTML = `
        <a href="#/" class="back-link">← Quay lại danh sách</a>
        <form class="form-panel" id="agent-form">
          <div class="form-row">
            <div class="form-group">
              <label>Tên agent</label>
              <input name="name" value="${escapeAttr(a.name)}" required>
            </div>
            <div class="form-group">
              <label>Engine</label>
              <select name="engine">
                <option value="claude-code" ${a.engine === "claude-code" ? "selected" : ""}>claude-code</option>
                <option value="codex" ${a.engine === "codex" ? "selected" : ""}>codex</option>
                <option value="goclaw" ${a.engine === "goclaw" ? "selected" : ""}>goclaw</option>
              </select>
            </div>
          </div>

          <div class="form-group">
            <label>Persona (soul)</label>
            <textarea name="soul" class="tall">${escapeHtml(a.soul)}</textarea>
          </div>

          <div class="form-group">
            <label>Kiến thức (knowledge)</label>
            <textarea name="knowledge" class="tall">${escapeHtml(a.knowledge)}</textarea>
          </div>

          <div class="form-section">
            <h3>Guard</h3>
            <div class="form-row">
              <div class="form-group">
                <label>Độ dài tối đa (ký tự)</label>
                <input name="max_reply_length" type="number" min="0" value="${a.guard.max_reply_length || 0}">
              </div>
              <div class="form-group">
                <label>Từ cấm (mỗi dòng một từ)</label>
                <textarea name="forbidden_words">${escapeHtml((a.guard.forbidden_words || []).join("\n"))}</textarea>
              </div>
            </div>
          </div>

          <div class="form-section">
            <h3>Channels (routing)</h3>
            <div class="form-group">
              <label>Messenger page_id (mỗi dòng một ID)</label>
              <textarea name="messenger_page_ids">${escapeHtml((a.channels?.messenger || []).map((c) => c.page_id).join("\n"))}</textarea>
            </div>
            <div class="form-group">
              <label>Zalo oa_id (mỗi dòng một ID)</label>
              <textarea name="zalo_oa_ids">${escapeHtml((a.channels?.zalo || []).map((c) => c.oa_id).join("\n"))}</textarea>
            </div>
          </div>

          <div class="form-section">
            <h3>Credential Messenger</h3>
            <div class="form-group">
              <label>Page access token</label>
              <input name="page_access_token" value="${escapeAttr(a.messenger?.page_access_token || "")}">
            </div>
            <div class="form-group">
              <label>Verify token</label>
              <input name="verify_token" value="${escapeAttr(a.messenger?.verify_token || "")}">
            </div>
          </div>

          <div class="form-actions">
            <button type="submit" class="btn btn-primary">Lưu</button>
            <span class="toast" id="save-toast">Đã lưu file YAML</span>
          </div>
        </form>`;

      document.getElementById("agent-form").addEventListener("submit", async (ev) => {
        ev.preventDefault();
        const fd = new FormData(ev.target);
        const forbiddenRaw = fd.get("forbidden_words") || "";
        const forbidden = forbiddenRaw
          .split("\n")
          .map((s) => s.trim())
          .filter(Boolean);

        const messengerIDs = (fd.get("messenger_page_ids") || "")
          .split("\n")
          .map((s) => s.trim())
          .filter(Boolean)
          .map((page_id) => ({ page_id }));
        const zaloIDs = (fd.get("zalo_oa_ids") || "")
          .split("\n")
          .map((s) => s.trim())
          .filter(Boolean)
          .map((oa_id) => ({ oa_id }));

        const payload = {
          name: fd.get("name"),
          engine: fd.get("engine"),
          soul: fd.get("soul"),
          knowledge: fd.get("knowledge"),
          channels: {
            messenger: messengerIDs,
            zalo: zaloIDs,
          },
          guard: {
            max_reply_length: parseInt(fd.get("max_reply_length"), 10) || 0,
            forbidden_words: forbidden,
          },
          messenger: {
            page_access_token: fd.get("page_access_token"),
            verify_token: fd.get("verify_token"),
          },
        };

        try {
          await apiPut("/api/admin/agents/" + encodeURIComponent(id), payload);
          const toast = document.getElementById("save-toast");
          toast.classList.add("show");
          setTimeout(() => toast.classList.remove("show"), 2500);
        } catch (e) {
          alert("Lưu thất bại: " + e.message);
        }
      });
    } catch (e) {
      app.innerHTML = `<div class="error-banner">${escapeHtml(e.message)}</div>`;
    }
  }

  // --- Màn GoClaw: danh sách agent từ GoClaw (chỉ đọc) ---
  async function renderGoclawAgents() {
    app.innerHTML = '<div class="empty-state">Đang tải từ GoClaw...</div>';
    try {
      const res = await fetch("/api/goclaw/agents");
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || res.statusText);
      }

      const agents = data.agents || [];
      if (!agents.length) {
        app.innerHTML = '<div class="empty-state">GoClaw chưa có agent nào</div>';
        return;
      }

      const html = agents
        .map(
          (a) => `
        <div class="goclaw-card">
          <div class="goclaw-header">
            <div class="agent-name">${escapeHtml(a.display_name || "—")}</div>
            <span class="status-badge ${escapeAttr(a.status || "offline")}">
              <span class="status-dot"></span>
              ${escapeHtml(a.status || "unknown")}
            </span>
          </div>
          <div class="agent-meta">Model: ${escapeHtml(a.model || "—")}</div>
          <div class="goclaw-desc">${escapeHtml(a.frontmatter || "")}</div>
        </div>`
        )
        .join("");
      app.innerHTML = `<div class="agent-grid">${html}</div>`;
    } catch (e) {
      app.innerHTML = `<div class="error-banner">${escapeHtml(e.message)}</div>`;
    }
  }

  // --- Màn 3: Log real-time (SSE) ---
  function renderLogs() {
    app.innerHTML = `
      <div class="logs-toolbar">
        <span class="logs-status" id="logs-status">Đang kết nối...</span>
        <button type="button" class="btn btn-secondary" id="clear-logs">Xóa màn hình</button>
      </div>
      <div class="log-viewer" id="log-viewer"></div>`;

    const viewer = document.getElementById("log-viewer");
    const status = document.getElementById("logs-status");

    document.getElementById("clear-logs").addEventListener("click", () => {
      viewer.innerHTML = "";
    });

    logEventSource = new EventSource("/api/admin/logs/stream");

    logEventSource.onopen = () => {
      status.textContent = "Đã kết nối — log real-time";
      status.classList.add("connected");
    };

    logEventSource.onmessage = (ev) => {
      appendLogLine(viewer, ev.data);
    };

    logEventSource.onerror = () => {
      status.textContent = "Mất kết nối — đang thử lại...";
      status.classList.remove("connected");
    };
  }

  function appendLogLine(viewer, text) {
    const line = document.createElement("div");
    line.className = "log-line";
    line.textContent = text;
    viewer.appendChild(line);
    viewer.scrollTop = viewer.scrollHeight;
  }

  function disconnectLogs() {
    if (logEventSource) {
      logEventSource.close();
      logEventSource = null;
    }
  }

  // --- Utils ---
  function escapeHtml(s) {
    if (s == null) return "";
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function escapeAttr(s) {
    return escapeHtml(s).replace(/'/g, "&#39;");
  }
})();
