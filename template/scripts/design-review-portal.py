#!/usr/bin/env python3

import argparse
import html
import json
import os
import pathlib
import posixpath
import re
import secrets
import subprocess
import urllib.parse
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Dict, List, Optional


def now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def read_json(path: pathlib.Path) -> Dict:
    try:
        with path.open("r", encoding="utf-8") as f:
            return json.load(f)
    except Exception:
        return {}


def write_json(path: pathlib.Path, payload: Dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as f:
        json.dump(payload, f, indent=2)
        f.write("\n")


def read_text(path: pathlib.Path) -> str:
    try:
        return path.read_text(encoding="utf-8").strip()
    except Exception:
        return ""


class PortalState:
    def __init__(self, root: pathlib.Path, token_file: pathlib.Path):
        self.root = root
        self.state_dir = root / ".claude" / "state"
        self.review_input_dir = root / "docs" / "design" / "review-input"
        self.artifact_index_file = self.state_dir / "design-artifacts.json"
        self.maple_json = self.state_dir / "maple.json"
        self.pending_file = self.state_dir / "approval-pending.txt"
        self.feedback_file = self.state_dir / "design-feedback.json"
        self.feedback_log = self.state_dir / "design-feedback.log"
        self.panes_file = self.state_dir / "panes.json"
        self.token_file = token_file

    def token(self) -> str:
        tok = read_text(self.token_file)
        if tok:
            return tok
        tok = secrets.token_hex(24)
        self.token_file.parent.mkdir(parents=True, exist_ok=True)
        self.token_file.write_text(tok + "\n", encoding="utf-8")
        return tok

    def pipeline(self) -> Dict:
        p = read_json(self.maple_json)
        pending = read_text(self.pending_file)
        awaiting = str(p.get("awaiting_approval", "") or "").strip()
        approval_pending = pending or awaiting
        feedback = read_json(self.feedback_file)
        stage = p.get("stage", "")
        return {
            "taffy": p.get("taffy", ""),
            "stage": stage,
            "status": p.get("status", ""),
            "awaiting_approval": awaiting,
            "updated_at": p.get("updated_at", ""),
            "approval_pending": approval_pending,
            "feedback": feedback,
            "uploads": self.list_uploads(),
            "declared_artifacts": self.declared_artifacts(stage),
        }

    def pending_stage(self) -> str:
        pending = read_text(self.pending_file)
        if pending:
            return pending
        p = read_json(self.maple_json)
        return str(p.get("awaiting_approval", "") or "").strip()

    def mark_resumed(self, stage: str) -> None:
        p = read_json(self.maple_json)
        if not isinstance(p, dict):
            p = {}
        awaiting = str(p.get("awaiting_approval", "") or "").strip()
        if stage and awaiting and awaiting != stage:
            return
        if awaiting:
            p["awaiting_approval"] = ""
        status = str(p.get("status", "") or "").strip().upper()
        if status == "PAUSED":
            p["status"] = "RUNNING"
        p["updated_at"] = now_iso()
        write_json(self.maple_json, p)

    def notify_continue(self) -> int:
        panes = read_json(self.panes_file)
        if not isinstance(panes, dict):
            return 0
        notified = 0
        for pane in panes.values():
            if not isinstance(pane, dict):
                continue
            kind = str(pane.get("kind", "") or "").strip()
            target = str(pane.get("target", "") or "").strip()
            if not kind or not target:
                continue
            try:
                if kind == "tmux":
                    subprocess.run(
                        ["tmux", "send-keys", "-t", target, "continue", "Enter"],
                        check=True,
                        stdout=subprocess.DEVNULL,
                        stderr=subprocess.DEVNULL,
                    )
                    notified += 1
                    continue
                if kind == "zellij":
                    subprocess.run(
                        ["zellij", "action", "go-to-tab-name", target],
                        check=True,
                        stdout=subprocess.DEVNULL,
                        stderr=subprocess.DEVNULL,
                    )
                    subprocess.run(
                        ["zellij", "action", "write-chars", "continue"],
                        check=True,
                        stdout=subprocess.DEVNULL,
                        stderr=subprocess.DEVNULL,
                    )
                    subprocess.run(
                        ["zellij", "action", "write", "13"],
                        check=True,
                        stdout=subprocess.DEVNULL,
                        stderr=subprocess.DEVNULL,
                    )
                    notified += 1
            except Exception:
                continue
        return notified

    def request_changes(self, message: str, attachments: Optional[List[str]] = None) -> Dict:
        stage = self.pending_stage()
        payload = {
            "stage": stage,
            "message": message.strip(),
            "ts": now_iso(),
            "status": "requested_changes",
            "attachments": attachments or [],
        }
        write_json(self.feedback_file, payload)
        self.feedback_log.parent.mkdir(parents=True, exist_ok=True)
        with self.feedback_log.open("a", encoding="utf-8") as f:
            f.write(json.dumps(payload) + "\n")
        if self.pending_file.exists():
            self.pending_file.unlink()
        self.mark_resumed(stage)
        payload["signaled_panes"] = self.notify_continue()
        return payload

    def reject(self, message: str, attachments: Optional[List[str]] = None) -> Dict:
        stage = self.pending_stage()
        payload = {
            "stage": stage,
            "message": message.strip(),
            "ts": now_iso(),
            "status": "rejected",
            "attachments": attachments or [],
        }
        write_json(self.feedback_file, payload)
        self.feedback_log.parent.mkdir(parents=True, exist_ok=True)
        with self.feedback_log.open("a", encoding="utf-8") as f:
            f.write(json.dumps(payload) + "\n")
        if self.pending_file.exists():
            self.pending_file.unlink()
        self.mark_resumed(stage)
        payload["signaled_panes"] = self.notify_continue()
        return payload

    def approve(self) -> Dict:
        stage = self.pending_stage()
        if self.pending_file.exists():
            self.pending_file.unlink()
        self.mark_resumed(stage)
        return {"approved": True, "stage": stage, "ts": now_iso(), "signaled_panes": self.notify_continue()}

    def list_uploads(self) -> List[Dict]:
        out: List[Dict] = []
        if not self.review_input_dir.exists():
            return out
        for p in self.review_input_dir.iterdir():
            if not p.is_file():
                continue
            rel = p.relative_to(self.root).as_posix()
            out.append(
                {
                    "path": rel,
                    "name": p.name,
                    "mtime": int(p.stat().st_mtime),
                    "platform": infer_platform(rel),
                }
            )
        out.sort(key=lambda x: x["mtime"], reverse=True)
        return out[:200]

    def save_upload(self, filename: str, content: bytes) -> Dict:
        if len(content) == 0:
            raise ValueError("empty file")
        if len(content) > 10 * 1024 * 1024:
            raise ValueError("file too large (max 10MB)")
        safe = sanitize_filename(filename)
        ext = pathlib.Path(safe).suffix.lower()
        allowed = {
            ".excalidraw",
            ".json",
            ".jpeg",
            ".jpg",
            ".png",
            ".webp",
            ".gif",
            ".html",
            ".htm",
            ".txt",
            ".md",
            ".svg",
            ".css",
        }
        if ext not in allowed:
            raise ValueError("unsupported file type")
        self.review_input_dir.mkdir(parents=True, exist_ok=True)
        stamped = f"{now_iso().replace(':', '').replace('-', '')}-{safe}"
        target = self.review_input_dir / stamped
        target.write_bytes(content)
        rel = target.relative_to(self.root).as_posix()
        return {
            "path": rel,
            "name": target.name,
            "platform": infer_platform(rel),
            "size": len(content),
        }

    def declared_artifacts(self, stage: str) -> List[Dict]:
        raw = read_json(self.artifact_index_file)
        items = raw.get("items", [])
        if not isinstance(items, list):
            return []
        out: List[Dict] = []
        stage_l = (stage or "").lower()
        for item in items:
            if isinstance(item, str):
                rel = to_repo_rel(self.root, item)
                if rel and is_allowed_rel(rel):
                    mtime = 0
                    target = self.root / rel
                    if target.exists() and target.is_file():
                        mtime = int(target.stat().st_mtime)
                    out.append({"path": rel, "kind": "file", "platform": infer_platform(rel), "mtime": mtime, "source": "manifest"})
                continue
            if not isinstance(item, dict):
                continue
            rel = to_repo_rel(self.root, str(item.get("path", "")))
            if not rel or not is_allowed_rel(rel):
                continue
            declared_stage = str(item.get("stage", "")).strip().lower()
            if declared_stage and stage_l and declared_stage != stage_l:
                continue
            mtime = int(item.get("mtime", 0) or 0)
            target = self.root / rel
            if target.exists() and target.is_file():
                mtime = int(target.stat().st_mtime)
            out.append(
                {
                    "path": rel,
                    "kind": str(item.get("kind", "file")),
                    "platform": str(item.get("platform", infer_platform(rel))),
                    "mtime": mtime,
                    "source": "manifest",
                }
            )
        return out[:200]


def sanitize_filename(name: str) -> str:
    base = pathlib.Path(name).name
    base = re.sub(r"[^A-Za-z0-9._-]+", "-", base).strip("-")
    if not base:
        base = "upload"
    return base[:120]


def normalize_rel(path: str) -> str:
    norm = posixpath.normpath(path.replace("\\", "/"))
    if norm.startswith("../") or norm == "..":
        return ""
    return norm.lstrip("/")


def to_repo_rel(root: pathlib.Path, path: str) -> str:
    raw = str(path or "").strip()
    if not raw:
        return ""
    p = pathlib.Path(raw)
    if p.is_absolute():
        try:
            return p.resolve().relative_to(root.resolve()).as_posix()
        except Exception:
            return ""
    return normalize_rel(raw)


def is_allowed_rel(path: str) -> bool:
    allowed_roots = (
        "docs/design/",
        "docs/stories/",
        "docs/specs/",
        "artifacts/",
        "screenshots/",
        "previews/",
    )
    return any(path.startswith(root) for root in allowed_roots)


def infer_platform(rel: str) -> str:
    p = rel.lower()
    if any(x in p for x in ["mobile", "android", "ios", "react-native", "swiftui", "flutter"]):
        return "mobile"
    if any(x in p for x in ["desktop", "electron", "tauri", "macos", "windows", "linux"]):
        return "desktop"
    if any(x in p for x in ["tui", "terminal", "cli", "console", ".ansi", ".cast"]):
        return "tui"
    if any(x in p for x in ["web", ".html", ".htm", ".svg", ".tsx", ".jsx", ".css"]):
        return "web"
    return "general"


def list_artifacts(root: pathlib.Path, stage: str) -> List[Dict]:
    stage = (stage or "").lower()
    roots = [
        "docs/design",
        "docs/stories",
        "docs/specs",
        "artifacts",
        "screenshots",
        "previews",
    ]
    if "wireframe" in stage:
        roots = ["docs/design/wireframes", "docs/design", "docs/stories", "docs/specs", "previews", "screenshots"]
    elif "visual-identity" in stage or "design-tokens" in stage:
        roots = ["docs/design/identity", "docs/design", "docs/stories", "docs/specs", "previews"]
    elif "mockup" in stage:
        roots = [
            "docs/design/mockups",
            "docs/design/system/components",
            "docs/design/identity",
            "docs/stories",
            "docs/specs",
            "artifacts",
            "screenshots",
            "previews",
        ]

    reviewable_ext = {
        ".excalidraw",
        ".md",
        ".txt",
        ".html",
        ".htm",
        ".svg",
        ".png",
        ".jpg",
        ".jpeg",
        ".gif",
        ".webp",
        ".json",
        ".yaml",
        ".yml",
        ".css",
        ".mp4",
        ".webm",
    }
    items: List[Dict] = []
    for r in roots:
        base = root / r
        if not base.exists():
            continue
        for p in base.rglob("*"):
            if not p.is_file():
                continue
            if p.name.startswith("."):
                continue
            rel = p.relative_to(root).as_posix()
            ext = p.suffix.lower()
            if ext not in reviewable_ext:
                continue
            kind = "text"
            if ext in {".png", ".jpg", ".jpeg", ".gif", ".webp"}:
                kind = "image"
            elif ext in {".mp4", ".webm"}:
                kind = "video"
            elif ext in {".html", ".htm", ".svg"}:
                kind = "preview"
            elif ext in {".json", ".md", ".tsx", ".ts", ".css", ".html", ".yaml", ".yml"}:
                kind = "text"
            else:
                kind = "file"
            items.append(
                {
                    "path": rel,
                    "kind": kind,
                    "platform": infer_platform(rel),
                    "mtime": int(p.stat().st_mtime),
                }
            )
    items.sort(key=lambda x: x["mtime"], reverse=True)
    return items[:200]


def merge_artifacts(scanned: List[Dict], declared: List[Dict]) -> List[Dict]:
    merged: List[Dict] = []
    seen: Dict[str, bool] = {}
    declared_sorted = sorted(declared, key=lambda x: int(x.get("mtime", 0) or 0), reverse=True)
    for item in declared_sorted:
        path = str(item.get("path", "")).strip()
        if not path or seen.get(path):
            continue
        seen[path] = True
        merged.append(item)
    scanned_sorted = sorted(scanned, key=lambda x: int(x.get("mtime", 0) or 0), reverse=True)
    for item in scanned_sorted:
        path = str(item.get("path", "")).strip()
        if not path or seen.get(path):
            continue
        seen[path] = True
        merged.append(item)
    return merged[:300]


def content_type_for(path: pathlib.Path) -> str:
    ext = path.suffix.lower()
    if ext == ".html":
        return "text/html; charset=utf-8"
    if ext == ".json":
        return "application/json; charset=utf-8"
    if ext in {".md", ".txt", ".tsx", ".ts", ".css", ".yml", ".yaml"}:
        return "text/plain; charset=utf-8"
    if ext == ".svg":
        return "image/svg+xml"
    if ext == ".png":
        return "image/png"
    if ext in {".jpg", ".jpeg"}:
        return "image/jpeg"
    if ext == ".gif":
        return "image/gif"
    if ext == ".webp":
        return "image/webp"
    if ext == ".mp4":
        return "video/mp4"
    if ext == ".webm":
        return "video/webm"
    return "application/octet-stream"


def render_index(token: str) -> str:
    token_js = json.dumps(token)
    return f"""<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>MAPLE Design Review</title>
  <style>
    :root {{
      --bg: #0f1117;
      --card: #171b22;
      --text: #e5e7eb;
      --muted: #9ca3af;
      --ok: #22c55e;
      --warn: #f59e0b;
      --bad: #ef4444;
      --line: #2b313b;
      --accent: #60a5fa;
    }}
    body {{ margin:0; font-family: Inter, system-ui, -apple-system, sans-serif; background:var(--bg); color:var(--text); }}
    .wrap {{ max-width:1200px; margin:0 auto; padding:24px; }}
    .h1 {{ font-size:24px; font-weight:700; margin:0 0 16px; }}
    .sub {{ color:var(--muted); font-size:13px; margin-bottom:16px; }}
    .row {{ display:grid; grid-template-columns: 1fr 1fr; gap:16px; }}
    @media (max-width: 980px) {{ .row {{ grid-template-columns: 1fr; }} }}
    .card {{ background:var(--card); border:1px solid var(--line); border-radius:12px; padding:16px; }}
    .label {{ color:var(--muted); font-size:12px; text-transform:uppercase; letter-spacing:.05em; }}
    .value {{ font-size:15px; margin-top:4px; }}
    .btns {{ display:flex; gap:10px; flex-wrap:wrap; margin-top:14px; }}
    button {{ border:0; border-radius:10px; padding:10px 14px; color:#fff; font-weight:600; cursor:pointer; }}
    .approve {{ background:var(--ok); }}
    .changes {{ background:var(--warn); color:#111827; }}
    .reject {{ background:var(--bad); }}
    .refresh {{ background:#374151; }}
    .upload {{ background:#2563eb; }}
    textarea {{ width:100%; min-height:92px; margin-top:10px; background:#0b0d12; color:var(--text); border:1px solid var(--line); border-radius:10px; padding:10px; }}
    .list {{ margin-top:12px; max-height:520px; overflow:auto; border:1px solid var(--line); border-radius:10px; }}
    .item {{ display:flex; justify-content:space-between; gap:12px; padding:10px 12px; border-bottom:1px solid var(--line); }}
    .item:last-child {{ border-bottom:none; }}
    .item button {{ background:#1f2937; padding:5px 10px; border-radius:8px; font-size:12px; }}
    .item .path-btn {{ background:none; border:0; color:var(--accent); cursor:pointer; text-align:left; padding:0; font-size:13px; }}
    .preview {{ margin-top:12px; border:1px solid var(--line); border-radius:10px; background:#0b0d12; min-height:260px; }}
    .preview-head {{ display:flex; justify-content:space-between; gap:8px; border-bottom:1px solid var(--line); padding:10px 12px; color:var(--muted); font-size:12px; }}
    .preview-body {{ padding:12px; max-height:520px; overflow:auto; }}
    .preview-body img {{ max-width:100%; border-radius:8px; }}
    .preview-body iframe {{ width:100%; height:520px; border:0; border-radius:8px; background:#fff; }}
    .preview-body video {{ width:100%; max-height:520px; border-radius:8px; background:#000; }}
    .code {{ white-space:pre-wrap; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size:12px; }}
    .md {{ line-height:1.55; font-size:14px; }}
    .md h1, .md h2, .md h3, .md h4 {{ margin: 16px 0 8px; line-height:1.25; }}
    .md h1 {{ font-size: 22px; }}
    .md h2 {{ font-size: 19px; }}
    .md h3 {{ font-size: 16px; }}
    .md p {{ margin: 8px 0; }}
    .md ul, .md ol {{ margin: 8px 0 8px 22px; }}
    .md li {{ margin: 4px 0; }}
    .md pre {{ background:#0b0d12; border:1px solid var(--line); border-radius:8px; padding:10px; overflow:auto; }}
    .md code {{ background:#0b0d12; border:1px solid var(--line); border-radius:6px; padding:1px 5px; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size:12px; }}
    .md blockquote {{ margin:10px 0; padding:8px 12px; border-left:3px solid var(--line); color:var(--muted); }}
    a {{ color:var(--accent); text-decoration:none; }}
    .chip {{ font-size:11px; color:var(--muted); border:1px solid var(--line); border-radius:999px; padding:2px 8px; }}
    .status {{ font-weight:700; }}
    .upload-input {{
      width: 100%;
      margin-top: 10px;
      border: 1px dashed var(--line);
      border-radius: 10px;
      padding: 10px;
      color: var(--muted);
      background: #0b0d12;
    }}
    .uploads {{ margin-top: 10px; max-height: 180px; overflow: auto; border:1px solid var(--line); border-radius:10px; }}
    .upload-item {{ display:flex; justify-content:space-between; gap:10px; padding:8px 10px; border-bottom:1px solid var(--line); font-size:12px; }}
    .upload-item:last-child {{ border-bottom:none; }}
    .upload-left {{ display:flex; align-items:center; gap:8px; min-width:0; }}
    .upload-left label {{ white-space:nowrap; overflow:hidden; text-overflow:ellipsis; max-width:330px; }}
    .upload-actions button {{ background:#1f2937; padding:4px 8px; border-radius:7px; font-size:11px; }}
    .modal {{
      position: fixed;
      inset: 0;
      background: rgba(2, 6, 23, 0.75);
      display: none;
      align-items: center;
      justify-content: center;
      z-index: 9999;
      padding: 24px;
    }}
    .modal.open {{ display: flex; }}
    .modal-card {{
      width: min(1200px, 96vw);
      max-height: 92vh;
      background: var(--card);
      border: 1px solid var(--line);
      border-radius: 12px;
      overflow: hidden;
      display: grid;
      grid-template-rows: auto 1fr;
    }}
    .modal-head {{
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      padding: 12px 14px;
      border-bottom: 1px solid var(--line);
      color: var(--muted);
      font-size: 12px;
    }}
    .modal-close {{ background: #374151; }}
    .modal-body {{ padding: 14px; overflow: auto; min-height: 200px; }}
  </style>
</head>
<body>
  <div class="wrap">
    <div class="h1">MAPLE Design Review Portal</div>
    <div class="sub">Companion to TUI pipeline approvals. TUI remains the primary control surface.</div>
    <div class="row">
      <div class="card">
        <div class="label">Workflow</div><div id="wf" class="value">-</div>
        <div class="label" style="margin-top:10px">Stage</div><div id="stage" class="value">-</div>
        <div class="label" style="margin-top:10px">Status</div><div id="status" class="value status">-</div>
        <div class="label" style="margin-top:10px">Pending approval</div><div id="pending" class="value">-</div>
        <div class="label" style="margin-top:10px">Updated</div><div id="updated" class="value">-</div>
        <textarea id="feedback" placeholder="Feedback for the agent (required for reject/change)..."></textarea>
        <input id="uploadInput" class="upload-input" type="file" multiple accept=".excalidraw,.json,.jpg,.jpeg,.png,.webp,.gif,.html,.htm,.txt,.md,.svg,.css" />
        <div class="btns">
          <button class="approve" onclick="approveStage()">Approve stage</button>
          <button class="reject" onclick="rejectStage()">Reject</button>
          <button class="changes" onclick="requestChanges()">Request changes</button>
          <button class="upload" onclick="uploadFiles()">Upload files</button>
          <button class="refresh" onclick="refreshAll()">Refresh</button>
        </div>
        <div class="label" style="margin-top:10px">Uploaded review files</div>
        <div id="uploads" class="uploads"></div>
        <div id="msg" class="sub" style="margin-top:10px"></div>
      </div>
      <div class="card">
        <div class="label">Artifacts</div>
        <div id="artifacts" class="list"></div>
        <div class="preview">
          <div class="preview-head">
            <span>Visual preview</span>
            <span id="previewPath">No preview selected</span>
          </div>
          <div id="previewBody" class="preview-body">No preview available yet for this stage.</div>
        </div>
      </div>
    </div>
  </div>
  <div id="artifactModal" class="modal" onclick="onModalBackdrop(event)">
    <div class="modal-card">
      <div class="modal-head">
        <span id="modalPath">Artifact preview</span>
        <button class="modal-close" onclick="closeArtifactModal()">Close</button>
      </div>
      <div id="modalBody" class="modal-body">No artifact selected.</div>
    </div>
  </div>
  <script src="https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"></script>
  <script>
    let TOKEN = {token_js};
    const selectedUploads = new Set();

    function esc(s) {{
      return String(s || "")
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
    }}

    function looksLikeTerminalArtifact(txt) {{
      // box-drawing chars signal an ASCII/TUI wireframe or mockup — render raw monospace
      return /[┌┐└┘├┤┬┴┼─│]/.test(String(txt || ""));
    }}

    function renderInlineMarkdown(text) {{
      let out = esc(text || "");
      out = out.replace(/`([^`]+)`/g, '<code>$1</code>');
      out = out.replace(/\\*\\*([^*]+)\\*\\*/g, '<strong>$1</strong>');
      out = out.replace(/\\*([^*]+)\\*/g, '<em>$1</em>');
      out = out.replace(/\\[([^\\]]+)\\]\\(([^\\s)]+)\\)/g, '<a href="$2" class="md-link" rel="noreferrer noopener">$1</a>');
      return out;
    }}

    function renderMarkdown(md) {{
      const lines = String(md || "").replace(/\\r\\n/g, "\\n").split("\\n");
      let htmlOut = '<div class="md">';
      let inCode = false;
      let codeLang = "";
      let codeLines = [];
      let inUl = false;
      let inOl = false;

      const flushLists = () => {{
        if (inUl) {{ htmlOut += "</ul>"; inUl = false; }}
        if (inOl) {{ htmlOut += "</ol>"; inOl = false; }}
      }};

      const flushCode = () => {{
        if (!inCode) return;
        const codeText = codeLines.join("\\n");
        if (codeLang === "mermaid") {{
          htmlOut += `<pre class="mermaid-source">${{esc(codeText)}}</pre>`;
        }} else {{
          htmlOut += `<pre><code>${{esc(codeText)}}</code></pre>`;
        }}
        inCode = false;
        codeLang = "";
        codeLines = [];
      }};

      for (const rawLine of lines) {{
        const line = rawLine || "";
        const trimmed = line.trim();
        if (trimmed.startsWith("```")) {{
          if (inCode) {{
            flushCode();
          }} else {{
            flushLists();
            inCode = true;
            const fence = trimmed.match(/^```([A-Za-z0-9_-]+)?\\s*$/);
            codeLang = String((fence && fence[1]) || "").toLowerCase();
            codeLines = [];
          }}
          continue;
        }}
        if (inCode) {{
          codeLines.push(line);
          continue;
        }}
        if (!trimmed) {{
          flushLists();
          htmlOut += "<p></p>";
          continue;
        }}
        if (/^#{{1,4}}\\s+/.test(trimmed)) {{
          flushLists();
          const level = (trimmed.match(/^#+/) || ["#"])[0].length;
          const content = trimmed.replace(/^#{{1,4}}\\s+/, "");
          htmlOut += `<h${{level}}>${{renderInlineMarkdown(content)}}</h${{level}}>`;
          continue;
        }}
        if (/^>\\s?/.test(trimmed)) {{
          flushLists();
          htmlOut += `<blockquote>${{renderInlineMarkdown(trimmed.replace(/^>\\s?/, ""))}}</blockquote>`;
          continue;
        }}
        if (/^[-*]\\s+/.test(trimmed)) {{
          if (!inUl) {{ flushLists(); htmlOut += "<ul>"; inUl = true; }}
          htmlOut += `<li>${{renderInlineMarkdown(trimmed.replace(/^[-*]\\s+/, ""))}}</li>`;
          continue;
        }}
        if (/^\\d+\\.\\s+/.test(trimmed)) {{
          if (!inOl) {{ flushLists(); htmlOut += "<ol>"; inOl = true; }}
          htmlOut += `<li>${{renderInlineMarkdown(trimmed.replace(/^\\d+\\.\\s+/, ""))}}</li>`;
          continue;
        }}
        flushLists();
        htmlOut += `<p>${{renderInlineMarkdown(trimmed)}}</p>`;
      }}
      flushCode();
      flushLists();
      htmlOut += "</div>";
      return htmlOut;
    }}

    let mermaidInitialized = false;

    function ensureMermaid() {{
      if (!window.mermaid) return false;
      if (!mermaidInitialized) {{
        window.mermaid.initialize({{
          startOnLoad: false,
          securityLevel: "loose",
          theme: "default",
        }});
        mermaidInitialized = true;
      }}
      return true;
    }}

    async function renderMermaidIn(container) {{
      if (!container) return;
      const sources = Array.from(container.querySelectorAll("pre.mermaid-source"));
      if (!sources.length) return;
      if (!ensureMermaid()) return;
      for (const pre of sources) {{
        const holder = document.createElement("div");
        holder.className = "mermaid";
        holder.textContent = pre.textContent || "";
        pre.replaceWith(holder);
      }}
      try {{
        await window.mermaid.run({{ nodes: container.querySelectorAll(".mermaid") }});
      }} catch (_e) {{
      }}
    }}

    async function api(path, options={{}}) {{
      const headers = options.headers || {{}};
      headers["X-Maple-Token"] = TOKEN;
      const isFormData = typeof FormData !== "undefined" && options.body instanceof FormData;
      if (!headers["Content-Type"] && options.body && !isFormData) headers["Content-Type"] = "application/json";
      let res = await fetch(path, {{...options, headers}});
      let txt = await res.text();
      let data = {{}};
      try {{ data = JSON.parse(txt || "{{}}"); }} catch (e) {{ data = {{ raw: txt }}; }}
      if (res.status === 403 && String(data.error || "").toLowerCase() === "invalid token") {{
        const tokenRes = await fetch("/api/token");
        if (tokenRes.ok) {{
          const tokenPayload = await tokenRes.json();
          if (tokenPayload && tokenPayload.token) {{
            TOKEN = String(tokenPayload.token);
            headers["X-Maple-Token"] = TOKEN;
            res = await fetch(path, {{...options, headers}});
            txt = await res.text();
            try {{ data = JSON.parse(txt || "{{}}"); }} catch (e) {{ data = {{ raw: txt }}; }}
          }}
        }}
      }}
      if (!res.ok) throw new Error(data.error || `HTTP ${{res.status}}`);
      return data;
    }}

    async function refreshAll() {{
      const s = await api("/api/state");
      document.getElementById("wf").textContent = s.taffy || "-";
      document.getElementById("stage").textContent = s.stage || "-";
      document.getElementById("status").textContent = s.status || "-";
      document.getElementById("pending").textContent = s.approval_pending || "-";
      document.getElementById("updated").textContent = s.updated_at || "-";
      if (s.approval_pending && s.feedback && s.feedback.message) {{
        document.getElementById("feedback").value = s.feedback.message;
      }}
      if (s.approval_pending && s.feedback && Array.isArray(s.feedback.attachments)) {{
        selectedUploads.clear();
        for (const p of s.feedback.attachments) selectedUploads.add(String(p));
      }}

      const a = await api("/api/artifacts");
      const box = document.getElementById("artifacts");
      box.innerHTML = "";
      let selected = null;
      if (!a.items || a.items.length === 0) {{
        box.innerHTML = '<div class="item"><span>No artifacts found yet for this stage. Expected: previewable files (.excalidraw/.html/.svg/.png/.jpg/.md) and optional .claude/state/design-artifacts.json manifest.</span></div>';
        setPreview(null);
      }} else {{
        for (const item of a.items) {{
          const row = document.createElement("div");
          row.className = "item";
          const left = document.createElement("span");
          const pathBtn = document.createElement("button");
          pathBtn.className = "path-btn";
          pathBtn.textContent = item.path;
          pathBtn.addEventListener("click", () => openArtifactModal(item));
          left.appendChild(pathBtn);
          const right = document.createElement("span");
          const sourceChip = item.source ? ` <span class="chip">${{esc(item.source)}}</span>` : "";
          right.innerHTML = `<span class="chip">${{esc(item.platform || "general")}}</span> <span class="chip">${{esc(item.kind)}}</span>${{sourceChip}}`;
          const btn = document.createElement("button");
          btn.textContent = "preview";
          btn.addEventListener("click", () => setPreview(item));
          right.appendChild(document.createTextNode(" "));
          right.appendChild(btn);
          row.appendChild(left);
          row.appendChild(right);
          box.appendChild(row);
          if (!selected && isPreviewable(item)) {{
            selected = item;
          }}
        }}
      }}
      setPreview(selected);
      const up = await api("/api/uploads");
      renderUploads(up.items || []);
    }}

    function isPreviewable(item) {{
      if (!item) return false;
      if (item.kind === "image" || item.kind === "preview" || item.kind === "video") return true;
      const p = String(item.path || "").toLowerCase();
      return p.endsWith(".md") || p.endsWith(".tsx") || p.endsWith(".ts") || p.endsWith(".css") || p.endsWith(".json") || p.endsWith(".excalidraw") || isImagePath(p);
    }}

    function isImagePath(lowerPath) {{
      const p = String(lowerPath || "");
      return p.endsWith(".png") || p.endsWith(".jpg") || p.endsWith(".jpeg") || p.endsWith(".gif") || p.endsWith(".webp");
    }}

    function renderExcalidrawSvg(rawText, maxChars) {{
      let parsed = null;
      try {{ parsed = JSON.parse(String(rawText || "")); }} catch (_e) {{ parsed = null; }}
      if (!parsed || !Array.isArray(parsed.elements)) {{
        return `<div class="code">${{esc(String(rawText || "").slice(0, maxChars))}}</div>`;
      }}
      const elems = parsed.elements.filter(e => e && !e.isDeleted);
      if (!elems.length) {{
        return '<div class="code">Empty Excalidraw scene.</div>';
      }}
      let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
      for (const e of elems) {{
        const x = Number(e.x || 0);
        const y = Number(e.y || 0);
        const w = Math.abs(Number(e.width || 0));
        const h = Math.abs(Number(e.height || 0));
        minX = Math.min(minX, x);
        minY = Math.min(minY, y);
        maxX = Math.max(maxX, x + w);
        maxY = Math.max(maxY, y + h);
      }}
      if (!Number.isFinite(minX) || !Number.isFinite(minY) || !Number.isFinite(maxX) || !Number.isFinite(maxY)) {{
        minX = 0; minY = 0; maxX = 1200; maxY = 800;
      }}
      const pad = 24;
      const vbX = minX - pad;
      const vbY = minY - pad;
      const vbW = Math.max(200, (maxX - minX) + pad * 2);
      const vbH = Math.max(120, (maxY - minY) + pad * 2);
      const stroke = (e) => esc(e.strokeColor || "#1f2937");
      const fill = (e) => esc(e.backgroundColor && e.backgroundColor !== "transparent" ? e.backgroundColor : "none");
      const sw = (e) => Math.max(1, Number(e.strokeWidth || 1));
      const pieces = [];
      for (const e of elems.slice(0, 600)) {{
        const t = String(e.type || "");
        const x = Number(e.x || 0);
        const y = Number(e.y || 0);
        const w = Math.abs(Number(e.width || 0));
        const h = Math.abs(Number(e.height || 0));
        if (t === "rectangle") {{
          pieces.push(`<rect x="${{x}}" y="${{y}}" width="${{w}}" height="${{h}}" rx="6" ry="6" fill="${{fill(e)}}" stroke="${{stroke(e)}}" stroke-width="${{sw(e)}}"></rect>`);
          continue;
        }}
        if (t === "ellipse") {{
          pieces.push(`<ellipse cx="${{x + w / 2}}" cy="${{y + h / 2}}" rx="${{Math.max(1, w / 2)}}" ry="${{Math.max(1, h / 2)}}" fill="${{fill(e)}}" stroke="${{stroke(e)}}" stroke-width="${{sw(e)}}"></ellipse>`);
          continue;
        }}
        if (t === "diamond") {{
          const p = `${{x + w / 2}},${{y}} ${{x + w}},${{y + h / 2}} ${{x + w / 2}},${{y + h}} ${{x}},${{y + h / 2}}`;
          pieces.push(`<polygon points="${{p}}" fill="${{fill(e)}}" stroke="${{stroke(e)}}" stroke-width="${{sw(e)}}"></polygon>`);
          continue;
        }}
        if (t === "text") {{
          const fontSize = Math.max(12, Number(e.fontSize || 16));
          const text = esc(String(e.text || ""));
          pieces.push(`<text x="${{x}}" y="${{y + fontSize}}" font-size="${{fontSize}}" fill="${{stroke(e)}}" font-family="Inter, system-ui, sans-serif">${{text}}</text>`);
          continue;
        }}
        if (Array.isArray(e.points) && e.points.length > 0) {{
          const pts = e.points.map(p => `${{x + Number(p[0] || 0)}},${{y + Number(p[1] || 0)}}`).join(" ");
          pieces.push(`<polyline points="${{pts}}" fill="none" stroke="${{stroke(e)}}" stroke-width="${{sw(e)}}" stroke-linecap="round" stroke-linejoin="round"></polyline>`);
          continue;
        }}
        pieces.push(`<rect x="${{x}}" y="${{y}}" width="${{Math.max(12, w)}}" height="${{Math.max(12, h)}}" fill="${{fill(e)}}" stroke="${{stroke(e)}}" stroke-width="${{sw(e)}}"></rect>`);
      }}
      return `<div style="overflow:auto;max-height:78vh"><svg xmlns="http://www.w3.org/2000/svg" viewBox="${{vbX}} ${{vbY}} ${{vbW}} ${{vbH}}" style="width:100%;height:auto;background:#fff;border-radius:8px;border:1px solid #d9e2ec">${{pieces.join("")}}</svg></div>`;
    }}

    async function setPreview(item) {{
      const pathEl = document.getElementById("previewPath");
      const body = document.getElementById("previewBody");
      if (!item) {{
        pathEl.textContent = "No preview selected";
        body.textContent = "No preview available yet for this stage.";
        return;
      }}
      const path = String(item.path || "");
      const lower = path.toLowerCase();
      const href = "/artifact/" + encodeURIComponent(path);
      pathEl.textContent = path;
      if (item.kind === "image" || isImagePath(lower)) {{
        body.innerHTML = `<img src="${{href}}" alt="${{esc(path)}}"/>`;
        return;
      }}
      if (item.kind === "video" || lower.endsWith(".mp4") || lower.endsWith(".webm")) {{
        body.innerHTML = `<video controls src="${{href}}"></video>`;
        return;
      }}
      if (item.kind === "preview" || lower.endsWith(".html") || lower.endsWith(".htm") || lower.endsWith(".svg")) {{
        body.innerHTML = `<iframe src="${{href}}" title="${{esc(path)}}"></iframe>`;
        return;
      }}
      try {{
        const res = await fetch(href);
        const txt = await res.text();
        if (lower.endsWith(".md")) {{
          if (looksLikeTerminalArtifact(txt)) {{
            body.innerHTML = `<div class="code">${{esc(txt.slice(0, 200000))}}</div>`;
            return;
          }}
          body.innerHTML = renderMarkdown(txt.slice(0, 200000));
          await renderMermaidIn(body);
          return;
        }}
        if (lower.endsWith(".excalidraw")) {{
          body.innerHTML = renderExcalidrawSvg(txt, 120000);
          return;
        }}
        body.innerHTML = `<div class="code">${{esc(txt.slice(0, 120000))}}</div>`;
      }} catch (e) {{
        body.textContent = "Failed to load preview content.";
      }}
    }}

    function resetReviewInputs() {{
      document.getElementById("feedback").value = "";
      document.getElementById("uploadInput").value = "";
      selectedUploads.clear();
    }}

    function artifactPathFromHref(href) {{
      const raw = String(href || "").trim();
      if (!raw) return "";
      if (raw.startsWith("/artifact/")) {{
        return decodeURIComponent(raw.slice("/artifact/".length));
      }}
      try {{
        const u = new URL(raw, window.location.origin);
        if (u.origin !== window.location.origin) return "";
        if (!u.pathname.startsWith("/artifact/")) return "";
        return decodeURIComponent(u.pathname.slice("/artifact/".length));
      }} catch (_e) {{
        return "";
      }}
    }}

    function openExternalInModal(url) {{
      const modal = document.getElementById("artifactModal");
      const modalPath = document.getElementById("modalPath");
      const modalBody = document.getElementById("modalBody");
      const safe = String(url || "");
      modalPath.textContent = safe || "Link preview";
      modal.classList.add("open");
      document.body.style.overflow = "hidden";
      modalBody.innerHTML = `<iframe src="${{esc(safe)}}" title="${{esc(safe)}}" style="width:100%;height:78vh;border:0;border-radius:8px;background:#fff;"></iframe>`;
    }}

    function bindMarkdownLinks(containerId) {{
      const container = document.getElementById(containerId);
      if (!container) return;
      container.addEventListener("click", async (event) => {{
        const el = event.target;
        const link = el && el.closest ? el.closest("a.md-link") : null;
        if (!link) return;
        event.preventDefault();
        const href = String(link.getAttribute("href") || "");
        const artifactPath = artifactPathFromHref(href);
        if (artifactPath) {{
          await openArtifactModal({{ path: artifactPath, kind: "file" }});
          return;
        }}
        openExternalInModal(href);
      }});
    }}

    async function openArtifactModal(item) {{
      const modal = document.getElementById("artifactModal");
      const modalPath = document.getElementById("modalPath");
      const modalBody = document.getElementById("modalBody");
      const path = String(item?.path || "");
      const lower = path.toLowerCase();
      const href = "/artifact/" + encodeURIComponent(path);
      modalPath.textContent = path || "Artifact preview";
      modal.classList.add("open");
      document.body.style.overflow = "hidden";

      if (!item) {{
        modalBody.textContent = "No artifact selected.";
        return;
      }}
      if (item.kind === "image" || isImagePath(lower)) {{
        modalBody.innerHTML = `<img src="${{href}}" alt="${{esc(path)}}" style="max-width:100%;border-radius:8px;"/>`;
        return;
      }}
      if (item.kind === "video" || lower.endsWith(".mp4") || lower.endsWith(".webm")) {{
        modalBody.innerHTML = `<video controls src="${{href}}" style="width:100%;max-height:80vh;border-radius:8px;background:#000;"></video>`;
        return;
      }}
      if (item.kind === "preview" || lower.endsWith(".html") || lower.endsWith(".htm") || lower.endsWith(".svg")) {{
        modalBody.innerHTML = `<iframe src="${{href}}" title="${{esc(path)}}" style="width:100%;height:78vh;border:0;border-radius:8px;background:#fff;"></iframe>`;
        return;
      }}
      try {{
        const res = await fetch(href);
        const txt = await res.text();
        if (lower.endsWith(".md")) {{
          if (looksLikeTerminalArtifact(txt)) {{
            modalBody.innerHTML = `<div class="code">${{esc(txt.slice(0, 240000))}}</div>`;
            return;
          }}
          modalBody.innerHTML = renderMarkdown(txt.slice(0, 240000));
          await renderMermaidIn(modalBody);
          return;
        }}
        if (lower.endsWith(".excalidraw")) {{
          modalBody.innerHTML = renderExcalidrawSvg(txt, 180000);
          return;
        }}
        modalBody.innerHTML = `<div class="code">${{esc(txt.slice(0, 180000))}}</div>`;
      }} catch (e) {{
        modalBody.textContent = "Failed to load artifact content.";
      }}
    }}

    function renderUploads(items) {{
      const box = document.getElementById("uploads");
      box.innerHTML = "";
      if (!items || items.length === 0) {{
        box.innerHTML = '<div class="upload-item"><span class="sub">No uploaded review files yet.</span></div>';
        return;
      }}
      for (const item of items) {{
        const row = document.createElement("div");
        row.className = "upload-item";
        const left = document.createElement("div");
        left.className = "upload-left";
        const chk = document.createElement("input");
        chk.type = "checkbox";
        chk.checked = selectedUploads.has(item.path);
        chk.addEventListener("change", () => {{
          if (chk.checked) selectedUploads.add(item.path);
          else selectedUploads.delete(item.path);
        }});
        const lbl = document.createElement("label");
        lbl.textContent = item.path;
        left.appendChild(chk);
        left.appendChild(lbl);
        const right = document.createElement("div");
        right.className = "upload-actions";
        right.innerHTML = `<span class="chip">${{esc(item.platform || "general")}}</span> `;
        const pbtn = document.createElement("button");
        pbtn.textContent = "preview";
        pbtn.addEventListener("click", () => openArtifactModal({{ path: item.path, kind: "file" }}));
        right.appendChild(pbtn);
        row.appendChild(left);
        row.appendChild(right);
        box.appendChild(row);
      }}
    }}

    function closeArtifactModal() {{
      const modal = document.getElementById("artifactModal");
      modal.classList.remove("open");
      document.body.style.overflow = "";
    }}

    function onModalBackdrop(event) {{
      if (event.target && event.target.id === "artifactModal") {{
        closeArtifactModal();
      }}
    }}

    window.addEventListener("keydown", (event) => {{
      if (event.key === "Escape") {{
        closeArtifactModal();
      }}
    }});

    async function approveStage() {{
      try {{
        const r = await api("/api/approve", {{ method: "POST" }});
        const n = (r && r.signaled_panes) || 0;
        const note = n > 0
          ? `nudged ${{n}} agent pane(s)`
          : "no live agent pane — it resumes on its next poll, or paste 'continue' in its terminal";
        document.getElementById("msg").textContent = `Approved: ${{r.stage || "stage"}} — ${{note}}`;
        resetReviewInputs();
        await refreshAll();
      }} catch (e) {{
        document.getElementById("msg").textContent = e.message;
      }}
    }}

    async function uploadSelectedInputFiles() {{
      const input = document.getElementById("uploadInput");
      const files = input.files ? Array.from(input.files) : [];
      if (!files.length) return [];
      const fd = new FormData();
      for (const f of files) fd.append("files", f, f.name);
      const r = await api("/api/upload", {{ method: "POST", body: fd, headers: {{}} }});
      const uploaded = Array.isArray(r.items) ? r.items : [];
      for (const it of uploaded) selectedUploads.add(it.path);
      input.value = "";
      return uploaded.map(x => x.path);
    }}

    async function uploadFiles() {{
      try {{
        const uploaded = await uploadSelectedInputFiles();
        if (!uploaded.length) {{
          document.getElementById("msg").textContent = "Choose at least one file to upload.";
          return;
        }}
        document.getElementById("msg").textContent = `Uploaded ${{uploaded.length}} file(s).`;
        await refreshAll();
      }} catch (e) {{
        document.getElementById("msg").textContent = e.message;
      }}
    }}

    async function rejectStage() {{
      const message = document.getElementById("feedback").value.trim();
      if (!message) {{
        document.getElementById("msg").textContent = "Add feedback before rejecting.";
        return;
      }}
      try {{
        const uploaded = await uploadSelectedInputFiles();
        const attachments = Array.from(new Set([...(Array.from(selectedUploads)), ...uploaded]));
        const r = await api("/api/reject", {{
          method: "POST",
          body: JSON.stringify({{ message, attachments }})
        }});
        document.getElementById("msg").textContent = `Rejected: ${{r.stage || "stage"}}`;
        resetReviewInputs();
        await refreshAll();
      }} catch (e) {{
        document.getElementById("msg").textContent = e.message;
      }}
    }}

    async function requestChanges() {{
      const message = document.getElementById("feedback").value.trim();
      if (!message) {{
        document.getElementById("msg").textContent = "Add feedback before requesting changes.";
        return;
      }}
      try {{
        const uploaded = await uploadSelectedInputFiles();
        const attachments = Array.from(new Set([...(Array.from(selectedUploads)), ...uploaded]));
        const r = await api("/api/request-changes", {{
          method: "POST",
          body: JSON.stringify({{ message, attachments }})
        }});
        document.getElementById("msg").textContent = `Changes requested for: ${{r.stage || "stage"}}`;
        resetReviewInputs();
        await refreshAll();
      }} catch (e) {{
        document.getElementById("msg").textContent = e.message;
      }}
    }}

    refreshAll().catch(err => {{
      document.getElementById("msg").textContent = err.message;
    }});
    bindMarkdownLinks("previewBody");
    bindMarkdownLinks("modalBody");
    setInterval(() => refreshAll().catch(() => {{}}), 4000);
  </script>
</body>
</html>"""


class Handler(BaseHTTPRequestHandler):
    server_version = "MapleDesignPortal/1.0"

    def _json(self, payload: Dict, code: int = 200) -> None:
        body = json.dumps(payload).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _text(self, data: bytes, ctype: str, code: int = 200) -> None:
        self.send_response(code)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def _state(self) -> PortalState:
        return self.server.portal_state  # type: ignore[attr-defined]

    def _check_token(self) -> Optional[str]:
        token = self.headers.get("X-Maple-Token", "")
        expected = self._state().token()
        if not expected or token != expected:
            return "invalid token"
        return None

    def _read_json_body(self) -> Dict:
        try:
            length = int(self.headers.get("Content-Length", "0"))
        except Exception:
            length = 0
        if length <= 0:
            return {}
        raw = self.rfile.read(length)
        try:
            return json.loads(raw.decode("utf-8"))
        except Exception:
            return {}

    def _read_upload_files(self) -> List[Dict]:
        content_type = self.headers.get("Content-Type", "")
        boundary = None
        for part in content_type.split(";"):
            part = part.strip()
            if part.lower().startswith("boundary="):
                boundary = part[9:].strip().strip('"')
                break
        if not boundary:
            raise ValueError("no boundary in Content-Type")
        try:
            length = int(self.headers.get("Content-Length", "0"))
        except Exception:
            length = 0
        if length <= 0:
            raise ValueError("empty body")
        body = self.rfile.read(length)
        boundary_bytes = b"--" + boundary.encode()
        parts = body.split(boundary_bytes)
        uploaded: List[Dict] = []
        for raw_part in parts[1:]:
            if raw_part.lstrip(b"\r\n").startswith(b"--"):
                break
            if raw_part.startswith(b"\r\n"):
                raw_part = raw_part[2:]
            if b"\r\n\r\n" not in raw_part:
                continue
            raw_headers, raw_body = raw_part.split(b"\r\n\r\n", 1)
            if raw_body.endswith(b"\r\n"):
                raw_body = raw_body[:-2]
            headers_str = raw_headers.decode("utf-8", errors="replace")
            filename = None
            for line in headers_str.split("\r\n"):
                if line.lower().startswith("content-disposition:"):
                    for item in line.split(";"):
                        item = item.strip()
                        if item.lower().startswith("filename="):
                            filename = item[9:].strip().strip('"')
                            break
            if not filename:
                continue
            item = self._state().save_upload(filename, raw_body)
            uploaded.append(item)
        return uploaded


    def do_GET(self) -> None:
        if self.path == "/health":
            self._json({"ok": True})
            return
        if self.path == "/" or self.path.startswith("/index.html"):
            token = self._state().token()
            html_doc = render_index(token).encode("utf-8")
            self._text(html_doc, "text/html; charset=utf-8")
            return
        if self.path.startswith("/api/state"):
            self._json(self._state().pipeline())
            return
        if self.path.startswith("/api/token"):
            self._json({"token": self._state().token()})
            return
        if self.path.startswith("/api/artifacts"):
            stage = self._state().pipeline().get("approval_pending") or self._state().pipeline().get("stage", "")
            scanned = list_artifacts(self._state().root, stage)
            declared = self._state().declared_artifacts(stage)
            items = merge_artifacts(scanned, declared)
            self._json({"items": items, "stage": stage})
            return
        if self.path.startswith("/api/uploads"):
            self._json({"items": self._state().list_uploads()})
            return
        if self.path.startswith("/artifact/"):
            rel = urllib.parse.unquote(self.path[len("/artifact/"):])
            rel = normalize_rel(rel)
            if not rel or not is_allowed_rel(rel):
                self._json({"error": "forbidden path"}, 403)
                return
            file_path = self._state().root / rel
            if not file_path.exists() or not file_path.is_file():
                self._json({"error": "not found"}, 404)
                return
            try:
                data = file_path.read_bytes()
            except Exception:
                self._json({"error": "cannot read file"}, 500)
                return
            self._text(data, content_type_for(file_path))
            return
        self._json({"error": "not found"}, 404)

    def do_POST(self) -> None:
        if self.path == "/api/approve":
            err = self._check_token()
            if err:
                self._json({"error": err}, 403)
                return
            self._json(self._state().approve())
            return
        if self.path == "/api/request-changes":
            err = self._check_token()
            if err:
                self._json({"error": err}, 403)
                return
            body = self._read_json_body()
            message = str(body.get("message", "")).strip()
            attachments = body.get("attachments", [])
            if not isinstance(attachments, list):
                attachments = []
            if not message:
                self._json({"error": "message required"}, 400)
                return
            self._json(self._state().request_changes(message, [str(x) for x in attachments]))
            return
        if self.path == "/api/reject":
            err = self._check_token()
            if err:
                self._json({"error": err}, 403)
                return
            body = self._read_json_body()
            message = str(body.get("message", "")).strip()
            attachments = body.get("attachments", [])
            if not isinstance(attachments, list):
                attachments = []
            if not message:
                self._json({"error": "message required"}, 400)
                return
            self._json(self._state().reject(message, [str(x) for x in attachments]))
            return
        if self.path == "/api/upload":
            err = self._check_token()
            if err:
                self._json({"error": err}, 403)
                return
            ctype = self.headers.get("Content-Type", "")
            if "multipart/form-data" not in ctype:
                self._json({"error": "multipart/form-data required"}, 400)
                return
            try:
                items = self._read_upload_files()
            except ValueError as e:
                self._json({"error": str(e)}, 400)
                return
            except Exception:
                self._json({"error": "failed to parse upload"}, 400)
                return
            if not items:
                self._json({"error": "no files uploaded"}, 400)
                return
            self._json({"items": items})
            return
        self._json({"error": "not found"}, 404)

    def log_message(self, fmt: str, *args) -> None:
        return


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="MAPLE design review portal")
    parser.add_argument("--root", required=True, help="repo root path")
    parser.add_argument("--port", type=int, default=4173, help="localhost port")
    parser.add_argument("--token-file", required=True, help="file containing auth token")
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    root = pathlib.Path(args.root).resolve()
    token_file = pathlib.Path(args.token_file).resolve()
    state = PortalState(root=root, token_file=token_file)
    server = ThreadingHTTPServer(("127.0.0.1", args.port), Handler)
    server.portal_state = state  # type: ignore[attr-defined]
    print(f"[design-portal] listening on http://127.0.0.1:{args.port}", flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
