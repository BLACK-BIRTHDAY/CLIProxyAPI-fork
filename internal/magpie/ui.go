package magpie

// LogoDataURI is the SVG data URI icon for MAGPIE Google AI Studio integration.
const LogoDataURI = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyNCIgaGVpZ2h0PSIyNCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSJub25lIj48ZGVmcz48bGluZWFyR3JhZGllbnQgaWQ9ImEiIHgxPSIyIiB5MT0iMyIgeDI9IjIwIiB5Mj0iMjEiIGdyYWRpZW50VW5pdHM9InVzZXJTcGFjZU9uVXNlIj48c3RvcCBvZmZzZXQ9IjAlIiBzdG9wLWNvbG9yPSIjMDZCNkQ0Ii8+PHN0b3Agb2Zmc2V0PSI1MCUiIHN0b3AtY29sb3I9IiMzQjgyRjYiLz48c3RvcCBvZmZzZXQ9IjEwMCUiIHN0b3AtY29sb3I9IiM4QjVDRjYiLz48L2xpbmVhckdyYWRpZW50PjxsaW5lYXJHcmFkaWVudCBpZD0iYiIgeDE9IjE0IiB5MT0iMiIgeDI9IjIyIiB5Mj0iMTAiIGdyYWRpZW50VW5pdHM9InVzZXJTcGFjZU9uVXNlIj48c3RvcCBvZmZzZXQ9IjAlIiBzdG9wLWNvbG9yPSIjOEI1Q0Y2Ii8+PHN0b3Agb2Zmc2V0PSIxMDAlIiBzdG9wLWNvbG9yPSIjRUM0ODk5Ii8+PC9saW5lYXJHcmFkaWVudD48L2RlZnM+PHBhdGggZD0iTTEwIDMuMkMxMC4zNSAzLjIgMTAuNjIgMy40NSAxMC43MiAzLjhDMTEuNDUgNi44NSAxMy4xNSA4LjU1IDE2LjIgOS4yOEMxNi41NSA5LjM4IDE2LjggOS42NSAxNi44IDEwQzE2LjggMTAuMzUgMTYuNTUgMTAuNjIgMTYuMiAxMC43MkMxMy4xNSAxMS40NSAxMS40NSAxMy4xNSAxMC43MiAxNi4yQzEwLjYyIDE2LjU1IDEwLjM1IDE2LjggMTAgMTYuOEM5LjY1IDE2LjggOS4zOCAxNi41NSA5LjI4IDE2LjJDOC41NSAxMy4xNSA2Ljg1IDExLjQ1IDMuOCAxMC43MkMzLjQ1IDEwLjYyIDMuMiAxMC4zNSAzLjIgMTBDMy4yIDkuNjUgMy40NSA5LjM4IDMuOCA5LjI4QzYuODUgOC41NSA4LjU1IDYuODUgOS4yOCAzLjhDOS4zOCAzLjQ1IDkuNjUgMy4yIDEwIDMuMloiIHRyYW5zZm9ybT0idHJhbnNsYXRlKC0wLjUgMi41KSBzY2FsZSgxLjEyKSIgZmlsbD0idXJsKCNhKSIvPjxwYXRoIGQ9Ik0xOC4yIDEuOEMxOC40MiAxLjggMTguNiAxLjk1IDE4LjY2IDIuMTZDMTkuMDggMy44MiAyMC4wMiA0Ljc2IDIxLjY4IDUuMThDMjEuODkgNS4yNCAyMi4wNCA1LjQyIDIyLjA0IDUuNjRDMjIuMDQgNS44NiAyMS44OSA2LjA0IDIxLjY4IDYuMUMyMC4wMiA2LjUyIDE5LjA4IDcuNDYgMTguNjYgOS4xMkMxOC42IDkuMzMgMTguNDIgOS40OCAxOC4yIDkuNDhDMTcuOTggOS40OCAxNy44IDkuMzMgMTcuNzQgOS4xMkMxNy4zMiA3LjQ2IDE2LjM4IDYuNTIgMTQuNzIgNi4xQzE0LjUxIDYuMDQgMTQuMzYgNS44NiAxNC4zNiA1LjY0QzE0LjM2IDUuNDIgMTQuNTEgNS4yNCAxNC43MiA1LjE4QzE2LjM4IDQuNzYgMTcuMzIgMy44MiAxNy43NCAyLjE2QzE3LjggMS45NSAxNy45OCAxLjggMTguMiAxLjhaIiBmaWxsPSJ1cmwoI2IpIi8+PGNpcmNsZSBjeD0iMTcuOCIgY3k9IjE4LjIiIHI9IjEuMzUiIGZpbGw9IiM2MzY2RjEiLz48L3N2Zz4="

// QuotaPageHTML returns the self-contained HTML document for the MAGPIE Quota and Accounts page.
func QuotaPageHTML() []byte {
	return []byte(quotaPageHTML)
}

// SettingsPageHTML returns the self-contained HTML document for the MAGPIE Settings page.
func SettingsPageHTML() []byte {
	return []byte(settingsPageHTML)
}

const quotaPageHTML = `<!DOCTYPE html>
<html lang="zh-CN" data-theme="light">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>MAGPIE 配额与账号 - CLIProxyAPI</title>
  <style>
    :root, [data-theme="light"] {
      --bg: #faf9f5;
      --card: #f0eee8;
      --card-hover: #e8e5de;
      --border: #e3e1db;
      --border-subtle: #eceae4;
      --text: #2d2a26;
      --text-muted: #76726b;
      --text-faint: #a09a90;
      --accent: #7c3aed;
      --accent-hover: #6d28d9;
      --accent-light: #ede9fe;
      --accent-border: #ddd6fe;
      --success: #10b981;
      --success-bg: #ecfdf5;
      --success-border: #a7f3d0;
      --warning: #f59e0b;
      --warning-bg: #fffbeb;
      --warning-border: #fde68a;
      --danger: #ef4444;
      --danger-bg: #fef2f2;
      --danger-border: #fecaca;
      --input-bg: #ffffff;
      --shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.05);
    }
    [data-theme="white"] {
      --bg: #ffffff;
      --card: #ffffff;
      --card-hover: #f9fafb;
      --border: #e5e5e5;
      --border-subtle: #f3f4f6;
      --text: #111827;
      --text-muted: #6b7280;
      --text-faint: #9ca3af;
      --accent: #7c3aed;
      --accent-hover: #6d28d9;
      --accent-light: #f5f3ff;
      --accent-border: #e9d5ff;
      --success: #10b981;
      --success-bg: #ecfdf5;
      --success-border: #a7f3d0;
      --warning: #f59e0b;
      --warning-bg: #fffbeb;
      --warning-border: #fde68a;
      --danger: #ef4444;
      --danger-bg: #fef2f2;
      --danger-border: #fecaca;
      --input-bg: #ffffff;
      --shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.06);
    }
    [data-theme="dark"] {
      --bg: #151412;
      --card: #1d1b18;
      --card-hover: #262320;
      --border: #3a3530;
      --border-subtle: #2d2925;
      --text: #f6f4f1;
      --text-muted: #a8a29e;
      --text-faint: #78716c;
      --accent: #8b5cf6;
      --accent-hover: #7c3aed;
      --accent-light: #2e1065;
      --accent-border: #5b21b6;
      --success: #34d399;
      --success-bg: #064e3b;
      --success-border: #065f46;
      --warning: #fbbf24;
      --warning-bg: #451a03;
      --warning-border: #78350f;
      --danger: #f87171;
      --danger-bg: #450a0a;
      --danger-border: #7f1d1d;
      --input-bg: #1d1b18;
      --shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.4);
    }
    * {
      box-sizing: border-box;
      margin: 0;
      padding: 0;
    }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      background-color: var(--bg);
      color: var(--text);
      line-height: 1.5;
      padding: 24px 16px;
      transition: background-color 0.2s ease, color 0.2s ease;
    }
    .container {
      max-width: 1200px;
      margin: 0 auto;
      display: flex;
      flex-direction: column;
      gap: 20px;
    }
    .header {
      display: flex;
      flex-direction: row;
      justify-content: space-between;
      align-items: center;
      flex-wrap: wrap;
      gap: 16px;
    }
    .header-left {
      display: flex;
      align-items: center;
      gap: 12px;
    }
    .logo {
      width: 38px;
      height: 38px;
      border-radius: 10px;
      padding: 4px;
      background: var(--accent-light);
      border: 1px solid var(--accent-border);
      flex-shrink: 0;
    }
    .title-group {
      display: flex;
      flex-direction: column;
    }
    .title-row {
      display: flex;
      align-items: center;
      gap: 10px;
    }
    .title-row h1 {
      font-size: 22px;
      font-weight: 700;
      color: var(--text);
    }
    .subtitle {
      font-size: 13px;
      color: var(--text-muted);
      margin-top: 2px;
    }
    .toolbar {
      display: flex;
      align-items: center;
      gap: 8px;
      flex-wrap: wrap;
    }
    .btn {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 8px 14px;
      border-radius: 10px;
      font-size: 13px;
      font-weight: 500;
      cursor: pointer;
      border: 1px solid var(--border);
      background: var(--card);
      color: var(--text);
      transition: all 0.15s ease;
      text-decoration: none;
      user-select: none;
    }
    .btn:hover {
      background: var(--card-hover);
    }
    .btn:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
    .btn-primary {
      background: var(--accent);
      color: #ffffff;
      border-color: var(--accent);
    }
    .btn-primary:hover {
      background: var(--accent-hover);
      border-color: var(--accent-hover);
    }
    .btn-danger {
      background: var(--danger-bg);
      color: var(--danger);
      border-color: var(--danger-border);
    }
    .btn-danger:hover {
      background: var(--danger);
      color: #ffffff;
    }
    .btn-sm {
      padding: 4px 8px;
      font-size: 12px;
      border-radius: 6px;
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 3px 10px;
      border-radius: 9999px;
      font-size: 12px;
      font-weight: 600;
    }
    .dot {
      width: 6px;
      height: 6px;
      border-radius: 50%;
    }
    .badge-success {
      background: var(--success-bg);
      color: var(--success);
      border: 1px solid var(--success-border);
    }
    .badge-success .dot { background: var(--success); }
    .badge-warning {
      background: var(--warning-bg);
      color: var(--warning);
      border: 1px solid var(--warning-border);
    }
    .badge-warning .dot { background: var(--warning); }
    .badge-danger {
      background: var(--danger-bg);
      color: var(--danger);
      border: 1px solid var(--danger-border);
    }
    .badge-danger .dot { background: var(--danger); }
    .badge-info {
      background: var(--accent-light);
      color: var(--accent);
      border: 1px solid var(--accent-border);
    }
    .badge-info .dot { background: var(--accent); }
    .badge-neutral {
      background: var(--card);
      color: var(--text-muted);
      border: 1px solid var(--border);
    }
    .badge-neutral .dot { background: var(--text-faint); }
    .banner {
      background: var(--card);
      border: 1px solid var(--border);
      border-left: 4px solid var(--accent);
      border-radius: 12px;
      padding: 16px 20px;
      display: flex;
      flex-direction: column;
      gap: 8px;
      box-shadow: var(--shadow);
    }
    .banner-title {
      font-size: 14px;
      font-weight: 600;
      color: var(--text);
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .banner-content {
      font-size: 13px;
      color: var(--text-muted);
      line-height: 1.6;
    }
    .code-row {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-top: 4px;
    }
    .code-box {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      font-size: 12px;
      background: var(--bg);
      border: 1px solid var(--border);
      padding: 6px 12px;
      border-radius: 6px;
      color: var(--accent);
      user-select: all;
    }
    .stats-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 12px;
    }
    .stat-card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 16px;
      box-shadow: var(--shadow);
      display: flex;
      flex-direction: column;
      justify-content: space-between;
    }
    .stat-label {
      font-size: 12px;
      font-weight: 500;
      color: var(--text-muted);
      margin-bottom: 6px;
    }
    .stat-val {
      font-size: 24px;
      font-weight: 700;
      color: var(--text);
    }
    .stat-meta {
      font-size: 11px;
      color: var(--text-faint);
      margin-top: 4px;
    }
    .panel {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 20px;
      box-shadow: var(--shadow);
      display: flex;
      flex-direction: column;
      gap: 16px;
    }
    .panel-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      border-bottom: 1px solid var(--border-subtle);
      padding-bottom: 12px;
    }
    .panel-title {
      font-size: 16px;
      font-weight: 600;
      color: var(--text);
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .cooldown-list {
      display: flex;
      flex-direction: column;
      gap: 10px;
    }
    .cooldown-item {
      background: var(--bg);
      border: 1px solid var(--border-subtle);
      border-radius: 10px;
      padding: 12px 16px;
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    .cooldown-row {
      display: flex;
      justify-content: space-between;
      align-items: center;
      font-size: 13px;
    }
    .cooldown-model {
      font-weight: 600;
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      color: var(--text);
    }
    .cooldown-stats {
      display: flex;
      align-items: center;
      gap: 10px;
      font-size: 12px;
      color: var(--text-muted);
    }
    .progress-bar {
      height: 6px;
      width: 100%;
      background: var(--border);
      border-radius: 3px;
      overflow: hidden;
    }
    .progress-fill {
      height: 100%;
      background: var(--warning);
      border-radius: 3px;
      transition: width 0.3s ease;
    }
    .progress-fill.high {
      background: var(--danger);
    }
    .empty-state {
      padding: 24px;
      text-align: center;
      color: var(--text-muted);
      font-size: 13px;
      background: var(--bg);
      border: 1px dashed var(--border);
      border-radius: 10px;
    }
    .accounts-toolbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 12px;
      flex-wrap: wrap;
    }
    .search-input {
      padding: 8px 12px;
      border-radius: 8px;
      border: 1px solid var(--border);
      background: var(--input-bg);
      color: var(--text);
      font-size: 13px;
      width: 260px;
      max-width: 100%;
      outline: none;
    }
    .search-input:focus {
      border-color: var(--accent);
    }
    .table-container {
      overflow-x: auto;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 13px;
      text-align: left;
    }
    th {
      padding: 10px 12px;
      border-bottom: 1px solid var(--border);
      font-weight: 600;
      color: var(--text-muted);
      background: var(--bg);
    }
    td {
      padding: 12px;
      border-bottom: 1px solid var(--border-subtle);
      color: var(--text);
      vertical-align: middle;
    }
    tr:last-child td {
      border-bottom: none;
    }
    tr:hover td {
      background: var(--bg);
    }
    .switch {
      position: relative;
      display: inline-block;
      width: 38px;
      height: 22px;
      flex-shrink: 0;
    }
    .switch input {
      opacity: 0;
      width: 0;
      height: 0;
    }
    .slider {
      position: absolute;
      cursor: pointer;
      top: 0; left: 0; right: 0; bottom: 0;
      background-color: var(--border);
      transition: .2s;
      border-radius: 22px;
    }
    .slider:before {
      position: absolute;
      content: "";
      height: 16px;
      width: 16px;
      left: 3px;
      bottom: 3px;
      background-color: white;
      transition: .2s;
      border-radius: 50%;
    }
    input:checked + .slider {
      background-color: var(--accent);
    }
    input:checked + .slider:before {
      transform: translateX(16px);
    }
    .modal-backdrop {
      position: fixed;
      inset: 0;
      background: rgba(0, 0, 0, 0.45);
      z-index: 1000;
      display: none;
      align-items: center;
      justify-content: center;
      padding: 16px;
      backdrop-filter: blur(2px);
    }
    .modal-backdrop.active {
      display: flex;
    }
    .modal-card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 14px;
      width: 100%;
      max-width: 520px;
      padding: 20px;
      box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.25);
      display: flex;
      flex-direction: column;
      gap: 16px;
    }
    .modal-title {
      font-size: 16px;
      font-weight: 700;
      color: var(--text);
    }
    .modal-body {
      font-size: 13px;
      color: var(--text-muted);
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    .modal-footer {
      display: flex;
      justify-content: flex-end;
      gap: 10px;
      margin-top: 8px;
    }
    .textarea {
      width: 100%;
      height: 140px;
      padding: 10px;
      border-radius: 8px;
      border: 1px solid var(--border);
      background: var(--input-bg);
      color: var(--text);
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      font-size: 12px;
      outline: none;
      resize: vertical;
    }
    .textarea:focus {
      border-color: var(--accent);
    }
    .file-drop {
      border: 2px dashed var(--border);
      border-radius: 10px;
      padding: 16px;
      text-align: center;
      background: var(--bg);
      cursor: pointer;
      transition: border-color 0.15s ease;
    }
    .file-drop:hover {
      border-color: var(--accent);
    }
    #toast-container {
      position: fixed;
      top: 20px;
      right: 20px;
      z-index: 9999;
      display: flex;
      flex-direction: column;
      gap: 8px;
      pointer-events: none;
    }
    .toast {
      padding: 10px 16px;
      border-radius: 8px;
      font-size: 13px;
      font-weight: 500;
      box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
      pointer-events: auto;
      transition: all 0.2s ease;
      display: flex;
      align-items: center;
      gap: 8px;
      max-width: 360px;
    }
    .toast-success {
      background: var(--success);
      color: #ffffff;
    }
    .toast-error {
      background: var(--danger);
      color: #ffffff;
    }
    .toast-info {
      background: var(--accent);
      color: #ffffff;
    }
    .model-tag {
      font-size: 11px;
      padding: 2px 6px;
      border-radius: 4px;
      background: var(--bg);
      border: 1px solid var(--border);
      color: var(--text-muted);
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      display: inline-block;
      margin: 2px 2px 2px 0;
    }
    .cooldown-tag {
      font-size: 11px;
      padding: 2px 6px;
      border-radius: 4px;
      background: var(--warning-bg);
      color: var(--warning);
      border: 1px solid var(--warning-border);
      font-weight: 600;
      display: inline-block;
      margin: 2px 2px 2px 0;
    }
  </style>
</head>
<body>
  <div id="toast-container"></div>

  <div class="container">
    <!-- Header -->
    <div class="header">
      <div class="header-left">
        <img class="logo" id="magpie-logo" alt="MAGPIE Logo" />
        <div class="title-group">
          <div class="title-row">
            <h1>MAGPIE 配额与账号</h1>
            <span id="service-badge" class="badge badge-neutral"><span class="dot"></span> 加载中...</span>
          </div>
          <p class="subtitle">管理 Google AI Studio 多账号池与各模型冷却窗口，支持 Headless 账号包一键导入导出。</p>
        </div>
      </div>
      <div class="toolbar">
        <button id="btn-refresh" class="btn" onclick="fetchData(true)">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"/></svg>
          刷新状态
        </button>
        <button id="btn-restart" class="btn" onclick="handleStartOrRestart()">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="5 3 19 12 5 21 5 3"/></svg>
          启动 / 重启服务
        </button>
        <button id="btn-import" class="btn" onclick="openImportModal()">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
          导入账号包
        </button>
        <button id="btn-export" class="btn" onclick="exportAccounts()">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
          导出账号包
        </button>
        <a href="/v0/resource/plugins/magpie/settings" class="btn btn-primary">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" y1="21" x2="4" y2="14"/><line x1="4" y1="10" x2="4" y2="3"/><line x1="12" y1="21" x2="12" y2="12"/><line x1="12" y1="8" x2="12" y2="3"/><line x1="20" y1="21" x2="20" y2="16"/><line x1="20" y1="12" x2="20" y2="3"/><line x1="1" y1="14" x2="7" y2="14"/><line x1="9" y1="8" x2="15" y2="8"/><line x1="17" y1="16" x2="23" y2="16"/></svg>
          MAGPIE 设置
        </a>
      </div>
    </div>

    <!-- Cloud Run Headless Notice Banner -->
    <div class="banner">
      <div class="banner-title">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
        Cloud Run 无头运行提示 (Headless Gateway)
      </div>
      <div class="banner-content">
        Google Cloud Run 容器无图形桌面环境，不支持弹出 Camoufox 浏览器登录。如需在桌面端导出已有 Google AI Studio 账号并同步至本网关，请在本地终端运行：
        <div class="code-row">
          <code class="code-box">node scripts/export-magpie-bundle.mjs</code>
          <button class="btn btn-sm" onclick="copyText('node scripts/export-magpie-bundle.mjs')">复制命令</button>
        </div>
        导出后点击上方「导入账号包」上传生成的 <code>magpie-accounts-bundle.json</code> 即可一键热加载入池！
      </div>
    </div>

    <!-- Summary Stats Grid -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-label">服务状态</div>
        <div class="stat-val" id="stat-service">--</div>
        <div class="stat-meta" id="stat-service-meta">端口: --</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">启用账号 (Serving)</div>
        <div class="stat-val" id="stat-enabled">0</div>
        <div class="stat-meta" id="stat-total">总账号: 0</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">就绪账号 (Ready)</div>
        <div class="stat-val" style="color: var(--success);" id="stat-ready">0</div>
        <div class="stat-meta">即时可用</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">冷却中 (Cooling Down)</div>
        <div class="stat-val" style="color: var(--warning);" id="stat-cooling">0</div>
        <div class="stat-meta" id="stat-cooling-meta">启用账号冷却数量</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">需重新授权 (Auth Required)</div>
        <div class="stat-val" style="color: var(--danger);" id="stat-auth">0</div>
        <div class="stat-meta">需要更新凭据</div>
      </div>
    </div>

    <!-- Model Cooldown Windows Panel -->
    <div class="panel">
      <div class="panel-header">
        <div class="panel-title">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
          模型冷却窗口 (Model Cooldown Windows)
        </div>
        <span id="cooldown-summary-badge" class="badge badge-success"><span class="dot"></span> 全模型就绪</span>
      </div>
      <div id="cooldown-windows-list" class="cooldown-list">
        <div class="empty-state">所有模型当前均处于就绪状态，无活跃冷却限制。</div>
      </div>
    </div>

    <!-- Account Pool Table Section -->
    <div class="panel">
      <div class="panel-header">
        <div class="panel-title">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          账号池状态 (Account Pool)
        </div>
        <div class="accounts-toolbar">
          <input type="text" id="search-input" class="search-input" placeholder="搜索账号 ID、邮箱或标签..." oninput="handleSearch()" />
          <span id="account-filter-count" style="font-size: 12px; color: var(--text-muted);">共 0 个账号</span>
        </div>
      </div>
      <div class="table-container">
        <table>
          <thead>
            <tr>
              <th style="width: 50px;">状态</th>
              <th>账号 ID / 标识</th>
              <th>邮箱 / 标签</th>
              <th>活跃冷却与支持模型</th>
              <th style="width: 80px; text-align: center;">启用</th>
              <th style="width: 80px; text-align: center;">操作</th>
            </tr>
          </thead>
          <tbody id="accounts-tbody">
            <tr>
              <td colspan="6" class="empty-state">暂无账号数据</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>

  <!-- Import Modal -->
  <div id="modal-import" class="modal-backdrop">
    <div class="modal-card">
      <div class="modal-title">导入 MAGPIE 账号凭据包 (Account Bundle)</div>
      <div class="modal-body">
        <p>支持上传 <code>scripts/export-magpie-bundle.mjs</code> 生成的 <code>magpie-accounts-bundle.json</code> 文件或直接粘贴 JSON 内容：</p>
        <div class="file-drop" onclick="document.getElementById('import-file-input').click()">
          <input type="file" id="import-file-input" accept=".json" style="display:none;" onchange="handleFileSelected(event)" />
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="margin: 0 auto 6px; color: var(--accent);"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
          <div>点击选择文件或拖拽 <strong>.json</strong> 文件至此</div>
        </div>
        <textarea id="import-json-text" class="textarea" placeholder="在此粘贴 MagpieAccountBundle JSON 内容..." oninput="validateImportInput()"></textarea>
        <div id="import-preview-meta" style="font-size: 12px; color: var(--text-muted);">未选择或输入文件</div>
      </div>
      <div class="modal-footer">
        <button class="btn" onclick="closeImportModal()">取消</button>
        <button id="btn-submit-import" class="btn btn-primary" onclick="submitImportAccounts()" disabled>开始导入</button>
      </div>
    </div>
  </div>

  <!-- Delete Modal -->
  <div id="modal-delete" class="modal-backdrop">
    <div class="modal-card">
      <div class="modal-title" style="color: var(--danger);">确认删除账号</div>
      <div class="modal-body">
        <p>确定要从 MAGPIE 账号池中永久删除账号 <strong id="delete-account-id-label" style="color: var(--text);"></strong> 吗？</p>
        <p style="font-size: 12px; color: var(--text-faint);">此操作将永久移除该账号在持久化存储中的所有凭证与浏览器会话数据，无法撤销。</p>
      </div>
      <div class="modal-footer">
        <button class="btn" onclick="closeDeleteModal()">取消</button>
        <button id="btn-confirm-delete" class="btn btn-danger" onclick="confirmDeleteAccount()">确认删除</button>
      </div>
    </div>
  </div>

  <!-- Auth Key Modal -->
  <div id="modal-auth" class="modal-backdrop">
    <div class="modal-card">
      <div class="modal-title">管理鉴权验证</div>
      <div class="modal-body">
        <p>检测到当前请求未附带有效管理权限。请输入 Management Password 或 API Key：</p>
        <input type="password" id="auth-key-input" class="search-input" style="width: 100%;" placeholder="Management Password / API Key" />
      </div>
      <div class="modal-footer">
        <button class="btn" onclick="closeAuthModal()">取消</button>
        <button class="btn btn-primary" onclick="saveAuthKey()">保存并重试</button>
      </div>
    </div>
  </div>

  <script>
    // Injected MAGPIE SVG Logo
    var LOGO_DATA_URI = "PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyNCIgaGVpZ2h0PSIyNCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSJub25lIj48ZGVmcz48bGluZWFyR3JhZGllbnQgaWQ9ImEiIHgxPSIyIiB5MT0iMyIgeDI9IjIwIiB5Mj0iMjEiIGdyYWRpZW50VW5pdHM9InVzZXJTcGFjZU9uVXNlIj48c3RvcCBvZmZzZXQ9IjAlIiBzdG9wLWNvbG9yPSIjMDZCNkQ0Ii8+PHN0b3Agb2Zmc2V0PSI1MCUiIHN0b3AtY29sb3I9IiMzQjgyRjYiLz48c3RvcCBvZmZzZXQ9IjEwMCUiIHN0b3AtY29sb3I9IiM4QjVDRjYiLz48L2xpbmVhckdyYWRpZW50PjxsaW5lYXJHcmFkaWVudCBpZD0iYiIgeDE9IjE0IiB5MT0iMiIgeDI9IjIyIiB5Mj0iMTAiIGdyYWRpZW50VW5pdHM9InVzZXJTcGFjZU9uVXNlIj48c3RvcCBvZmZzZXQ9IjAlIiBzdG9wLWNvbG9yPSIjOEI1Q0Y2Ii8+PHN0b3Agb2Zmc2V0PSIxMDAlIiBzdG9wLWNvbG9yPSIjRUM0ODk5Ii8+PC9saW5lYXJHcmFkaWVudD48L2RlZnM+PHBhdGggZD0iTTEwIDMuMkMxMC4zNSAzLjIgMTAuNjIgMy40NSAxMC43MiAzLjhDMTEuNDUgNi44NSAxMy4xNSA4LjU1IDE2LjIgOS4yOEMxNi41NSA5LjM4IDE2LjggOS42NSAxNi44IDEwQzE2LjggMTAuMzUgMTYuNTUgMTAuNjIgMTYuMiAxMC43MkMxMy4xNSAxMS40NSAxMS40NSAxMy4xNSAxMC43MiAxNi4yQzEwLjYyIDE2LjU1IDEwLjM1IDE2LjggMTAgMTYuOEM5LjY1IDE2LjggOS4zOCAxNi41NSA5LjI4IDE2LjJDOC41NSAxMy4xNSA2Ljg1IDExLjQ1IDMuOCAxMC43MkMzLjQ1IDEwLjYyIDMuMiAxMC4zNSAzLjIgMTBDMy4yIDkuNjUgMy40NSA5LjM4IDMuOCA5LjI4QzYuODUgOC41NSA4LjU1IDYuODUgOS4yOCAzLjhDOS4zOCAzLjQ1IDkuNjUgMy4yIDEwIDMuMloiIHRyYW5zZm9ybT0idHJhbnNsYXRlKC0wLjUgMi41KSBzY2FsZSgxLjEyKSIgZmlsbD0idXJsKCNhKSIvPjxwYXRoIGQ9Ik0xOC4yIDEuOEMxOC40MiAxLjggMTguNiAxLjk1IDE4LjY2IDIuMTZDMTkuMDggMy44MiAyMC4wMiA0Ljc2IDIxLjY4IDUuMThDMjEuODkgNS4yNCAyMi4wNCA1LjQyIDIyLjA0IDUuNjRDMjIuMDQgNS44NiAyMS44OSA2LjA0IDIxLjY4IDYuMUMyMC4wMiA2LjUyIDE5LjA4IDcuNDYgMTguNjYgOS4xMkMxOC42IDkuMzMgMTguNDIgOS40OCAxOC4yIDkuNDhDMTcuOTggOS40OCAxNy44IDkuMzMgMTcuNzQgOS4xMkMxNy4zMiA3LjQ2IDE2LjM4IDYuNTIgMTQuNzIgNi4xQzE0LjUxIDYuMDQgMTQuMzYgNS44NiAxNC4zNiA1LjY0QzE0LjM2IDUuNDIgMTQuNTEgNS4yNCAxNC43MiA1LjE4QzE2LjM4IDQuNzYgMTcuMzIgMy44MiAxNy43NCAyLjE2QzE3LjggMS45NSAxNy45OCAxLjggMTguMiAxLjhaIiBmaWxsPSJ1cmwoI2IpIi8+PGNpcmNsZSBjeD0iMTcuOCIgY3k9IjE4LjIiIHI9IjEuMzUiIGZpbGw9IiM2MzY2RjEiLz48L3N2Zz4=";
    var logoImg = document.getElementById('magpie-logo');
    if (logoImg) logoImg.src = "data:image/svg+xml;base64," + LOGO_DATA_URI;

    // State
    var rawStatus = null;
    var rawQuota = null;
    var allAccounts = [];
    var searchQuery = '';
    var accountToDelete = null;
    var pendingAuthRetry = null;

    // Theme sync
    function initThemeSync() {
      function getTheme() {
        try {
          if (window.parent && window.parent.document && window.parent.document.documentElement) {
            var t = window.parent.document.documentElement.getAttribute('data-theme');
            if (t) return t;
          }
        } catch (e) {}
        try {
          var local = localStorage.getItem('cli-proxy-theme');
          if (local) return local;
        } catch (e) {}
        if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
          return 'dark';
        }
        return 'light';
      }
      function applyTheme(t) {
        document.documentElement.setAttribute('data-theme', t);
      }
      applyTheme(getTheme());
      try {
        if (window.parent && window.parent.document && window.parent.document.documentElement) {
          var obs = new MutationObserver(function() {
            var t = window.parent.document.documentElement.getAttribute('data-theme');
            if (t) applyTheme(t);
          });
          obs.observe(window.parent.document.documentElement, {
            attributes: true,
            attributeFilter: ['data-theme']
          });
        }
      } catch (e) {}
      window.addEventListener('storage', function(e) {
        if (e.key === 'cli-proxy-theme' && e.newValue) {
          applyTheme(e.newValue);
        }
      });
      if (window.matchMedia) {
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function(e) {
          if (!localStorage.getItem('cli-proxy-theme')) {
            applyTheme(e.matches ? 'dark' : 'light');
          }
        });
      }
    }
    initThemeSync();

    // Toast
    function showToast(msg, type) {
      type = type || 'info';
      var container = document.getElementById('toast-container');
      if (!container) return;
      var el = document.createElement('div');
      el.className = 'toast toast-' + type;
      el.textContent = msg;
      container.appendChild(el);
      setTimeout(function() {
        el.style.opacity = '0';
        el.style.transform = 'translateY(-10px)';
        setTimeout(function() {
          if (el.parentNode) el.parentNode.removeChild(el);
        }, 200);
      }, 3000);
    }

    // Clipboard
    function copyText(text) {
      if (!text) return;
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(function() {
          showToast('已复制到剪贴板', 'success');
        }).catch(function() {
          fallbackCopy(text);
        });
      } else {
        fallbackCopy(text);
      }
    }
    function fallbackCopy(text) {
      var ta = document.createElement('textarea');
      ta.value = text;
      document.body.appendChild(ta);
      ta.select();
      try {
        document.execCommand('copy');
        showToast('已复制到剪贴板', 'success');
      } catch (e) {
        showToast('复制失败', 'error');
      }
      document.body.removeChild(ta);
    }

    // API fetcher
    function fetchApi(endpoint, options) {
      options = options || {};
      var headers = Object.assign({
        'Content-Type': 'application/json'
      }, options.headers || {});

      try {
        var key = localStorage.getItem('cli-proxy-auth') || sessionStorage.getItem('cli-proxy-auth');
        if (key && !headers['Authorization']) {
          headers['Authorization'] = 'Bearer ' + key;
          headers['X-Management-Key'] = key;
        }
      } catch (e) {}

      options.headers = headers;
      options.credentials = 'same-origin';
      options.cache = 'no-store';

      return fetch(endpoint, options).then(function(res) {
        if (res.status === 401 || res.status === 403) {
          pendingAuthRetry = { endpoint: endpoint, options: options };
          openAuthModal();
          throw new Error('需要管理权限认证 (HTTP ' + res.status + ')');
        }
        if (!res.ok) {
          return res.text().then(function(t) {
            var msg = t;
            try {
              var j = JSON.parse(t);
              msg = j.error || j.message || t;
            } catch (err) {}
            throw new Error(msg || ('HTTP ' + res.status));
          });
        }
        var ct = res.headers.get('content-type') || '';
        if (ct.indexOf('application/json') !== -1) {
          return res.json();
        }
        return res.text();
      });
    }

    function openAuthModal() {
      var m = document.getElementById('modal-auth');
      if (m) m.classList.add('active');
    }
    function closeAuthModal() {
      var m = document.getElementById('modal-auth');
      if (m) m.classList.remove('active');
    }
    function saveAuthKey() {
      var input = document.getElementById('auth-key-input');
      if (!input) return;
      var val = input.value.trim();
      if (!val) return;
      try {
        localStorage.setItem('cli-proxy-auth', val);
      } catch (e) {}
      closeAuthModal();
      showToast('管理密钥已保存，正在重试请求...', 'info');
      if (pendingAuthRetry) {
        fetchApi(pendingAuthRetry.endpoint, pendingAuthRetry.options).then(function() {
          fetchData(false);
        }).catch(function(err) {
          showToast(err.message, 'error');
        });
        pendingAuthRetry = null;
      }
    }

    // Cooldown countdown helpers
    function formatRemaining(untilStr) {
      if (!untilStr) return '';
      var ms = Date.parse(untilStr) - Date.now();
      if (isNaN(ms) || ms <= 0) return '即将恢复';
      var totalSecs = Math.floor(ms / 1000);
      var mins = Math.floor(totalSecs / 60);
      var secs = totalSecs % 60;
      if (mins > 0) {
        return mins + 'm ' + secs + 's';
      }
      return secs + 's';
    }

    function updateLiveCountdowns() {
      var els = document.querySelectorAll('[data-countdown-until]');
      els.forEach(function(el) {
        var u = el.getAttribute('data-countdown-until');
        if (u) {
          var rem = formatRemaining(u);
          if (el.getAttribute('data-prefix')) {
            el.textContent = el.getAttribute('data-prefix') + rem;
          } else {
            el.textContent = rem;
          }
        }
      });
    }
    setInterval(updateLiveCountdowns, 1000);

    // Compute cooldown windows strictly over enabled accounts
    function computeCooldownWindows(accounts, rawCooldowns) {
      var enabledAccounts = accounts.filter(function(a) {
        return a.enabled !== false && a.state !== 'disabled';
      });
      var enabledIds = new Set(enabledAccounts.map(function(a) { return a.id; }));

      var list = [];
      if (Array.isArray(rawCooldowns)) {
        list = rawCooldowns;
      } else if (rawCooldowns && Array.isArray(rawCooldowns.cooldowns)) {
        list = rawCooldowns.cooldowns;
      }

      var now = Date.now();
      var active = list.filter(function(c) {
        if (!c || !c.until) return false;
        var u = Date.parse(c.until);
        return !isNaN(u) && u > now && enabledIds.has(c.account_id);
      });

      if (active.length === 0) return [];

      var byModel = new Map();
      active.forEach(function(c) {
        var model = c.model_id || '所有模型';
        var entry = byModel.get(model);
        if (!entry) {
          entry = { accounts: new Set(), until: Infinity, reasons: new Set() };
          byModel.set(model, entry);
        }
        entry.accounts.add(c.account_id);
        var u = Date.parse(c.until);
        if (u < entry.until) entry.until = u;
        if (c.reason && c.reason.trim()) entry.reasons.add(c.reason.trim());
      });

      var windows = [];
      byModel.forEach(function(m, model) {
        var serves = enabledAccounts.filter(function(a) {
          return !model || model === '所有模型' || !a.models || !a.models.length || a.models.indexOf(model) !== -1;
        }).length;
        var total = Math.max(serves, m.accounts.size, 1);
        var used = Math.round((m.accounts.size / total) * 100);
        var reasonsArr = Array.from(m.reasons);
        windows.push({
          name: model,
          used: used,
          resetsAt: new Date(m.until).toISOString(),
          coolingCount: m.accounts.size,
          totalServing: total,
          reason: reasonsArr.length > 0 ? reasonsArr[0] : ''
        });
      });

      windows.sort(function(a, b) {
        return b.used - a.used || Date.parse(a.resetsAt) - Date.parse(b.resetsAt);
      });
      return windows;
    }

    // Render Cooldown Windows
    function renderCooldownWindows(windows) {
      var container = document.getElementById('cooldown-windows-list');
      var badge = document.getElementById('cooldown-summary-badge');
      if (!container || !badge) return;

      if (!windows || windows.length === 0) {
        badge.className = 'badge badge-success';
        badge.innerHTML = '<span class="dot"></span> 全模型就绪';
        container.innerHTML = '<div class="empty-state">所有模型当前均处于就绪状态，无活跃冷却限制。</div>';
        return;
      }

      var totalCooling = 0;
      windows.forEach(function(w) { totalCooling += w.coolingCount; });
      badge.className = 'badge badge-warning';
      badge.innerHTML = '<span class="dot"></span> ' + windows.length + ' 个模型冷却中 (' + totalCooling + ' 账号)';

      var html = '';
      windows.forEach(function(w) {
        var isHigh = w.used >= 80;
        var reasonHtml = w.reason ? '<span class="model-tag" style="color:var(--warning);">' + escapeHtml(w.reason) + '</span>' : '';
        html += '<div class="cooldown-item">' +
          '<div class="cooldown-row">' +
            '<div class="cooldown-model">' + escapeHtml(w.name) + ' ' + reasonHtml + '</div>' +
            '<div class="cooldown-stats">' +
              '<span>' + w.coolingCount + ' / ' + w.totalServing + ' 账号冷却中 (' + w.used + '%)</span>' +
              '<span class="cooldown-tag" data-countdown-until="' + w.resetsAt + '">' + formatRemaining(w.resetsAt) + '</span>' +
            '</div>' +
          '</div>' +
          '<div class="progress-bar">' +
            '<div class="progress-fill ' + (isHigh ? 'high' : '') + '" style="width:' + w.used + '%;"></div>' +
          '</div>' +
        '</div>';
      });
      container.innerHTML = html;
    }

    // Render Accounts Table
    function renderAccounts(accounts, cooldowns) {
      var tbody = document.getElementById('accounts-tbody');
      var countEl = document.getElementById('account-filter-count');
      if (!tbody) return;

      var q = (searchQuery || '').trim().toLowerCase();
      var filtered = accounts.filter(function(a) {
        if (!q) return true;
        var id = (a.id || '').toLowerCase();
        var label = (a.label || a.email || a.name || '').toLowerCase();
        return id.indexOf(q) !== -1 || label.indexOf(q) !== -1;
      });

      if (countEl) {
        countEl.textContent = '显示 ' + filtered.length + ' / ' + accounts.length + ' 个账号';
      }

      if (filtered.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="empty-state">未找到匹配的账号</td></tr>';
        return;
      }

      // Map cooldowns per account
      var rawList = Array.isArray(cooldowns) ? cooldowns : (cooldowns && Array.isArray(cooldowns.cooldowns) ? cooldowns.cooldowns : []);
      var now = Date.now();
      var cdMap = new Map();
      rawList.forEach(function(c) {
        if (!c || !c.until) return;
        var u = Date.parse(c.until);
        if (!isNaN(u) && u > now) {
          var arr = cdMap.get(c.account_id) || [];
          arr.push(c);
          cdMap.set(c.account_id, arr);
        }
      });

      var html = '';
      filtered.forEach(function(acc) {
        var isEnabled = acc.enabled !== false && acc.state !== 'disabled';
        var state = (acc.state || 'ready').toLowerCase();
        var myCooldowns = cdMap.get(acc.id) || [];

        // State Badge
        var badgeHtml = '';
        if (!isEnabled) {
          badgeHtml = '<span class="badge badge-neutral"><span class="dot"></span> 已禁用</span>';
        } else if (state === 'auth_required') {
          badgeHtml = '<span class="badge badge-danger"><span class="dot"></span> 授权失效</span>';
        } else if (state === 'busy') {
          badgeHtml = '<span class="badge badge-info"><span class="dot"></span> 忙碌</span>';
        } else if (state === 'cooldown' || myCooldowns.length > 0) {
          var earliest = Infinity;
          myCooldowns.forEach(function(c) {
            var u = Date.parse(c.until);
            if (u < earliest) earliest = u;
          });
          var untilIso = earliest !== Infinity ? new Date(earliest).toISOString() : '';
          badgeHtml = '<span class="badge badge-warning"><span class="dot"></span> 冷却中</span>';
          if (untilIso) {
            badgeHtml += ' <span style="font-size:11px;color:var(--warning);" data-countdown-until="' + untilIso + '">' + formatRemaining(untilIso) + '</span>';
          }
        } else {
          badgeHtml = '<span class="badge badge-success"><span class="dot"></span> 就绪</span>';
        }

        // Active model cooldown pills
        var cdPills = '';
        myCooldowns.forEach(function(c) {
          var m = c.model_id || 'all';
          cdPills += '<span class="cooldown-tag">' + escapeHtml(m) + ': <span data-countdown-until="' + c.until + '">' + formatRemaining(c.until) + '</span></span> ';
        });

        // Models list preview
        var modelsHtml = '';
        if (acc.models && acc.models.length > 0) {
          acc.models.slice(0, 3).forEach(function(m) {
            modelsHtml += '<span class="model-tag">' + escapeHtml(m) + '</span> ';
          });
          if (acc.models.length > 3) {
            modelsHtml += '<span class="model-tag">+' + (acc.models.length - 3) + '</span>';
          }
        }

        html += '<tr>' +
          '<td>' + badgeHtml + '</td>' +
          '<td>' +
            '<div style="font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-weight:600;display:flex;align-items:center;gap:6px;">' +
              '<span>' + escapeHtml(acc.id) + '</span>' +
              '<button class="btn btn-sm" style="padding:2px 5px;" onclick="copyText(\'' + escapeJsString(acc.id) + '\')">' +
                '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>' +
              '</button>' +
            '</div>' +
          '</td>' +
          '<td>' +
            '<div>' + escapeHtml(acc.email || acc.label || acc.name || '--') + '</div>' +
          '</td>' +
          '<td>' +
            (cdPills ? '<div style="margin-bottom:4px;">' + cdPills + '</div>' : '') +
            (modelsHtml ? '<div>' + modelsHtml + '</div>' : '<span style="color:var(--text-faint);font-size:11px;">全部模型通用</span>') +
          '</td>' +
          '<td style="text-align:center;">' +
            '<label class="switch">' +
              '<input type="checkbox" ' + (isEnabled ? 'checked' : '') + ' onchange="toggleAccount(\'' + escapeJsString(acc.id) + '\', this.checked)">' +
              '<span class="slider"></span>' +
            '</label>' +
          '</td>' +
          '<td style="text-align:center;">' +
            '<button class="btn btn-sm btn-danger" onclick="openDeleteModal(\'' + escapeJsString(acc.id) + '\', \'' + escapeJsString(acc.email || acc.label || acc.id) + '\')">' +
              '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>' +
            '</button>' +
          '</td>' +
        '</tr>';
      });

      tbody.innerHTML = html;
    }

    // Render Stats
    function renderStats(statusData, quotaData) {
      var isRunning = false;
      var port = 2048;
      if (statusData && statusData.process) {
        isRunning = Boolean(statusData.process.running);
        if (statusData.process.port) port = statusData.process.port;
      } else if (quotaData && quotaData.status) {
        isRunning = Boolean(quotaData.status.running);
      }

      var svcBadge = document.getElementById('service-badge');
      var statSvc = document.getElementById('stat-service');
      var statSvcMeta = document.getElementById('stat-service-meta');

      if (isRunning) {
        if (svcBadge) {
          svcBadge.className = 'badge badge-success';
          svcBadge.innerHTML = '<span class="dot"></span> 运行中 (' + port + ')';
        }
        if (statSvc) {
          statSvc.textContent = '运行中';
          statSvc.style.color = 'var(--success)';
        }
      } else {
        if (svcBadge) {
          svcBadge.className = 'badge badge-warning';
          svcBadge.innerHTML = '<span class="dot"></span> 已停止';
        }
        if (statSvc) {
          statSvc.textContent = '已停止';
          statSvc.style.color = 'var(--warning)';
        }
      }
      if (statSvcMeta) {
        statSvcMeta.textContent = '内部端口: ' + port;
      }

      var accounts = allAccounts || [];
      var total = accounts.length;
      var enabledAccounts = accounts.filter(function(a) {
        return a.enabled !== false && a.state !== 'disabled';
      });
      var enabledCount = enabledAccounts.length;

      // Active cooldowns map
      var rawCooldowns = quotaData ? quotaData.cooldowns : [];
      var rawList = Array.isArray(rawCooldowns) ? rawCooldowns : (rawCooldowns && Array.isArray(rawCooldowns.cooldowns) ? rawCooldowns.cooldowns : []);
      var now = Date.now();
      var activeCooldownAccountIds = new Set();
      rawList.forEach(function(c) {
        if (!c || !c.until) return;
        var u = Date.parse(c.until);
        if (!isNaN(u) && u > now) {
          activeCooldownAccountIds.add(c.account_id);
        }
      });

      var coolingCount = enabledAccounts.filter(function(a) {
        return a.state === 'cooldown' || activeCooldownAccountIds.has(a.id);
      }).length;

      var readyCount = enabledAccounts.filter(function(a) {
        return (a.state === 'ready' || !a.state) && !activeCooldownAccountIds.has(a.id);
      }).length;

      var authCount = accounts.filter(function(a) {
        return a.state === 'auth_required';
      }).length;

      var statEnabled = document.getElementById('stat-enabled');
      var statTotal = document.getElementById('stat-total');
      var statReady = document.getElementById('stat-ready');
      var statCooling = document.getElementById('stat-cooling');
      var statAuth = document.getElementById('stat-auth');

      if (statEnabled) statEnabled.textContent = enabledCount;
      if (statTotal) statTotal.textContent = '总账号池: ' + total;
      if (statReady) statReady.textContent = readyCount;
      if (statCooling) statCooling.textContent = coolingCount;
      if (statAuth) statAuth.textContent = authCount;
    }

    // Main Fetch
    function fetchData(isManual) {
      var btn = document.getElementById('btn-refresh');
      if (btn && isManual) btn.disabled = true;

      Promise.all([
        fetchApi('/v8/management/magpie/status').catch(function() { return null; }),
        fetchApi('/v8/management/magpie/quota').catch(function() { return null; })
      ]).then(function(results) {
        rawStatus = results[0];
        rawQuota = results[1];

        if (rawQuota && Array.isArray(rawQuota.accounts)) {
          allAccounts = rawQuota.accounts;
        } else if (rawQuota && rawQuota.data && Array.isArray(rawQuota.data.accounts)) {
          allAccounts = rawQuota.data.accounts;
        } else {
          allAccounts = [];
        }

        renderStats(rawStatus, rawQuota);

        var cooldowns = (rawQuota && rawQuota.cooldowns) ? rawQuota.cooldowns : [];
        var windows = computeCooldownWindows(allAccounts, cooldowns);
        renderCooldownWindows(windows);

        renderAccounts(allAccounts, cooldowns);

        if (isManual) {
          showToast('配额与状态已刷新', 'success');
        }
      }).catch(function(err) {
        console.error('Fetch error:', err);
        if (isManual) {
          showToast('刷新失败: ' + err.message, 'error');
        }
      }).finally(function() {
        if (btn) btn.disabled = false;
      });
    }

    // Search handler
    function handleSearch() {
      var input = document.getElementById('search-input');
      if (input) {
        searchQuery = input.value;
        var cooldowns = (rawQuota && rawQuota.cooldowns) ? rawQuota.cooldowns : [];
        renderAccounts(allAccounts, cooldowns);
      }
    }

    // Toggle Account
    function toggleAccount(accountId, targetState) {
      fetchApi('/v8/management/magpie/accounts/' + encodeURIComponent(accountId) + '/toggle', {
        method: 'PATCH',
        body: JSON.stringify({ enabled: targetState })
      }).then(function() {
        showToast(targetState ? '已启用账号' : '已禁用账号', 'success');
        fetchData(false);
      }).catch(function(err) {
        showToast('切换账号状态失败: ' + err.message, 'error');
        fetchData(false);
      });
    }

    // Delete Account
    function openDeleteModal(id, label) {
      accountToDelete = id;
      var lbl = document.getElementById('delete-account-id-label');
      if (lbl) lbl.textContent = id + (label && label !== id ? ' (' + label + ')' : '');
      var m = document.getElementById('modal-delete');
      if (m) m.classList.add('active');
    }
    function closeDeleteModal() {
      accountToDelete = null;
      var m = document.getElementById('modal-delete');
      if (m) m.classList.remove('active');
    }
    function confirmDeleteAccount() {
      if (!accountToDelete) return;
      var btn = document.getElementById('btn-confirm-delete');
      if (btn) btn.disabled = true;

      fetchApi('/v8/management/magpie/accounts/' + encodeURIComponent(accountToDelete), {
        method: 'DELETE'
      }).then(function() {
        showToast('已删除账号: ' + accountToDelete, 'success');
        closeDeleteModal();
        fetchData(false);
      }).catch(function(err) {
        showToast('删除失败: ' + err.message, 'error');
      }).finally(function() {
        if (btn) btn.disabled = false;
      });
    }

    // Start / Restart Service
    function handleStartOrRestart() {
      var isRunning = rawStatus && rawStatus.process && rawStatus.process.running;
      var endpoint = isRunning ? '/v8/management/magpie/restart' : '/v8/management/magpie/start';
      var text = isRunning ? '重启' : '启动';

      showToast('正在' + text + ' MAGPIE 服务...', 'info');
      fetchApi(endpoint, {
        method: 'POST'
      }).then(function() {
        showToast('MAGPIE 服务已' + text + '成功', 'success');
        fetchData(false);
      }).catch(function(err) {
        showToast(text + '服务失败: ' + err.message, 'error');
      });
    }

    // Export Accounts Bundle
    function exportAccounts() {
      var btn = document.getElementById('btn-export');
      if (btn) btn.disabled = true;

      fetchApi('/v8/management/magpie/accounts/export').then(function(res) {
        var bundle = (res && res.data) ? res.data : res;
        if (!bundle || !bundle.accounts || bundle.accounts.length === 0) {
          showToast('当前没有可导出的账号凭证', 'warning');
          return;
        }

        var jsonStr = JSON.stringify(bundle, null, 2);
        try {
          var blob = new Blob([jsonStr], { type: 'application/json' });
          var url = URL.createObjectURL(blob);
          var a = document.createElement('a');
          var dateStr = new Date().toISOString().slice(0, 10);
          a.href = url;
          a.download = 'magpie-accounts-bundle-' + dateStr + '.json';
          document.body.appendChild(a);
          a.click();
          document.body.removeChild(a);
          URL.revokeObjectURL(url);
          showToast('成功导出 ' + bundle.accounts.length + ' 个账号凭据', 'success');
        } catch (e) {
          copyText(jsonStr);
          showToast('已复制账号凭证 JSON 到剪贴板', 'success');
        }
      }).catch(function(err) {
        showToast('导出失败: ' + err.message, 'error');
      }).finally(function() {
        if (btn) btn.disabled = false;
      });
    }

    // Import Accounts Bundle
    function openImportModal() {
      var ta = document.getElementById('import-json-text');
      var meta = document.getElementById('import-preview-meta');
      var btn = document.getElementById('btn-submit-import');
      if (ta) ta.value = '';
      if (meta) meta.textContent = '请选择文件或粘贴 JSON 内容';
      if (btn) btn.disabled = true;
      var m = document.getElementById('modal-import');
      if (m) m.classList.add('active');
    }
    function closeImportModal() {
      var m = document.getElementById('modal-import');
      if (m) m.classList.remove('active');
    }
    function handleFileSelected(e) {
      var file = e.target.files && e.target.files[0];
      if (!file) return;
      var reader = new FileReader();
      reader.onload = function(evt) {
        var content = evt.target && evt.target.result;
        if (typeof content === 'string') {
          var ta = document.getElementById('import-json-text');
          if (ta) ta.value = content;
          validateImportInput();
        }
      };
      reader.readAsText(file);
      e.target.value = '';
    }
    function validateImportInput() {
      var ta = document.getElementById('import-json-text');
      var meta = document.getElementById('import-preview-meta');
      var btn = document.getElementById('btn-submit-import');
      if (!ta || !meta || !btn) return;

      var text = ta.value.trim();
      if (!text) {
        meta.textContent = '请输入或上传有效 JSON 数据';
        meta.style.color = 'var(--text-muted)';
        btn.disabled = true;
        return;
      }

      try {
        var parsed = JSON.parse(text);
        var bundle = (parsed && parsed.bundle && Array.isArray(parsed.bundle.accounts)) ? parsed.bundle : parsed;
        if (bundle && Array.isArray(bundle.accounts) && bundle.accounts.length > 0) {
          meta.textContent = '已解析有效凭证包: ' + bundle.accounts.length + ' 个账号' + (bundle.exported_at ? ' (导出于 ' + bundle.exported_at + ')' : '');
          meta.style.color = 'var(--success)';
          btn.disabled = false;
        } else {
          meta.textContent = 'JSON 格式正确，但未找到有效 accounts 账号数组';
          meta.style.color = 'var(--danger)';
          btn.disabled = true;
        }
      } catch (err) {
        meta.textContent = 'JSON 解析错误: ' + err.message;
        meta.style.color = 'var(--danger)';
        btn.disabled = true;
      }
    }
    function submitImportAccounts() {
      var ta = document.getElementById('import-json-text');
      var btn = document.getElementById('btn-submit-import');
      if (!ta || !btn) return;

      var text = ta.value.trim();
      if (!text) return;

      var parsed = null;
      try {
        parsed = JSON.parse(text);
      } catch (e) {
        showToast('无效的 JSON 格式', 'error');
        return;
      }

      var bundle = (parsed && parsed.bundle && Array.isArray(parsed.bundle.accounts)) ? parsed.bundle : parsed;
      btn.disabled = true;
      btn.textContent = '正在导入...';

      fetchApi('/v8/management/magpie/accounts/import', {
        method: 'POST',
        body: JSON.stringify({ bundle: bundle })
      }).then(function(res) {
        var result = (res && res.data) ? res.data : res;
        var count = (result && result.imported_count !== undefined) ? result.imported_count : bundle.accounts.length;
        showToast('成功导入 ' + count + ' 个账号凭据！', 'success');
        closeImportModal();
        fetchData(false);
      }).catch(function(err) {
        showToast('导入失败: ' + err.message, 'error');
      }).finally(function() {
        btn.disabled = false;
        btn.textContent = '开始导入';
      });
    }

    // Helpers
    function escapeHtml(str) {
      if (!str) return '';
      return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
    }
    function escapeJsString(str) {
      if (!str) return '';
      return String(str).replace(/\\/g, '\\\\').replace(/'/g, "\\'");
    }

    // Initial load
    fetchData(false);
    setInterval(function() {
      fetchData(false);
    }, 15000);
  </script>
</body>
</html>`

const settingsPageHTML = `<!DOCTYPE html>
<html lang="zh-CN" data-theme="light">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>MAGPIE 设置 - CLIProxyAPI</title>
  <style>
    :root, [data-theme="light"] {
      --bg: #faf9f5;
      --card: #f0eee8;
      --card-hover: #e8e5de;
      --border: #e3e1db;
      --border-subtle: #eceae4;
      --text: #2d2a26;
      --text-muted: #76726b;
      --text-faint: #a09a90;
      --accent: #7c3aed;
      --accent-hover: #6d28d9;
      --accent-light: #ede9fe;
      --accent-border: #ddd6fe;
      --success: #10b981;
      --success-bg: #ecfdf5;
      --success-border: #a7f3d0;
      --warning: #f59e0b;
      --warning-bg: #fffbeb;
      --warning-border: #fde68a;
      --danger: #ef4444;
      --danger-bg: #fef2f2;
      --danger-border: #fecaca;
      --input-bg: #ffffff;
      --shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.05);
    }
    [data-theme="white"] {
      --bg: #ffffff;
      --card: #ffffff;
      --card-hover: #f9fafb;
      --border: #e5e5e5;
      --border-subtle: #f3f4f6;
      --text: #111827;
      --text-muted: #6b7280;
      --text-faint: #9ca3af;
      --accent: #7c3aed;
      --accent-hover: #6d28d9;
      --accent-light: #f5f3ff;
      --accent-border: #e9d5ff;
      --success: #10b981;
      --success-bg: #ecfdf5;
      --success-border: #a7f3d0;
      --warning: #f59e0b;
      --warning-bg: #fffbeb;
      --warning-border: #fde68a;
      --danger: #ef4444;
      --danger-bg: #fef2f2;
      --danger-border: #fecaca;
      --input-bg: #ffffff;
      --shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.06);
    }
    [data-theme="dark"] {
      --bg: #151412;
      --card: #1d1b18;
      --card-hover: #262320;
      --border: #3a3530;
      --border-subtle: #2d2925;
      --text: #f6f4f1;
      --text-muted: #a8a29e;
      --text-faint: #78716c;
      --accent: #8b5cf6;
      --accent-hover: #7c3aed;
      --accent-light: #2e1065;
      --accent-border: #5b21b6;
      --success: #34d399;
      --success-bg: #064e3b;
      --success-border: #065f46;
      --warning: #fbbf24;
      --warning-bg: #451a03;
      --warning-border: #78350f;
      --danger: #f87171;
      --danger-bg: #450a0a;
      --danger-border: #7f1d1d;
      --input-bg: #1d1b18;
      --shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.4);
    }
    * {
      box-sizing: border-box;
      margin: 0;
      padding: 0;
    }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      background-color: var(--bg);
      color: var(--text);
      line-height: 1.5;
      padding: 24px 16px;
      transition: background-color 0.2s ease, color 0.2s ease;
    }
    .container {
      max-width: 1200px;
      margin: 0 auto;
      display: flex;
      flex-direction: column;
      gap: 20px;
    }
    .header {
      display: flex;
      flex-direction: row;
      justify-content: space-between;
      align-items: center;
      flex-wrap: wrap;
      gap: 16px;
    }
    .header-left {
      display: flex;
      align-items: center;
      gap: 12px;
    }
    .logo {
      width: 38px;
      height: 38px;
      border-radius: 10px;
      padding: 4px;
      background: var(--accent-light);
      border: 1px solid var(--accent-border);
      flex-shrink: 0;
    }
    .title-group {
      display: flex;
      flex-direction: column;
    }
    .title-row {
      display: flex;
      align-items: center;
      gap: 10px;
    }
    .title-row h1 {
      font-size: 22px;
      font-weight: 700;
      color: var(--text);
    }
    .subtitle {
      font-size: 13px;
      color: var(--text-muted);
      margin-top: 2px;
    }
    .toolbar {
      display: flex;
      align-items: center;
      gap: 8px;
      flex-wrap: wrap;
    }
    .btn {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 8px 14px;
      border-radius: 10px;
      font-size: 13px;
      font-weight: 500;
      cursor: pointer;
      border: 1px solid var(--border);
      background: var(--card);
      color: var(--text);
      transition: all 0.15s ease;
      text-decoration: none;
      user-select: none;
    }
    .btn:hover {
      background: var(--card-hover);
    }
    .btn:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
    .btn-primary {
      background: var(--accent);
      color: #ffffff;
      border-color: var(--accent);
    }
    .btn-primary:hover {
      background: var(--accent-hover);
      border-color: var(--accent-hover);
    }
    .btn-danger {
      background: var(--danger-bg);
      color: var(--danger);
      border-color: var(--danger-border);
    }
    .btn-danger:hover {
      background: var(--danger);
      color: #ffffff;
    }
    .btn-success {
      background: var(--success);
      color: #ffffff;
      border-color: var(--success);
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 3px 10px;
      border-radius: 9999px;
      font-size: 12px;
      font-weight: 600;
    }
    .dot {
      width: 6px;
      height: 6px;
      border-radius: 50%;
    }
    .badge-success {
      background: var(--success-bg);
      color: var(--success);
      border: 1px solid var(--success-border);
    }
    .badge-success .dot { background: var(--success); }
    .badge-warning {
      background: var(--warning-bg);
      color: var(--warning);
      border: 1px solid var(--warning-border);
    }
    .badge-warning .dot { background: var(--warning); }
    .badge-danger {
      background: var(--danger-bg);
      color: var(--danger);
      border: 1px solid var(--danger-border);
    }
    .badge-danger .dot { background: var(--danger); }
    .badge-info {
      background: var(--accent-light);
      color: var(--accent);
      border: 1px solid var(--accent-border);
    }
    .badge-info .dot { background: var(--accent); }
    .badge-neutral {
      background: var(--card);
      color: var(--text-muted);
      border: 1px solid var(--border);
    }
    .badge-neutral .dot { background: var(--text-faint); }
    .panel {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 20px;
      box-shadow: var(--shadow);
      display: flex;
      flex-direction: column;
      gap: 16px;
    }
    .panel-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      border-bottom: 1px solid var(--border-subtle);
      padding-bottom: 12px;
    }
    .panel-title {
      font-size: 16px;
      font-weight: 600;
      color: var(--text);
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .banner {
      background: var(--card);
      border: 1px solid var(--accent-border);
      background: linear-gradient(135deg, var(--accent-light) 0%, var(--card) 100%);
      border-radius: 14px;
      padding: 20px;
      display: flex;
      flex-direction: column;
      gap: 14px;
      box-shadow: var(--shadow);
    }
    .banner-title {
      font-size: 15px;
      font-weight: 700;
      color: var(--text);
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .banner-desc {
      font-size: 13px;
      color: var(--text-muted);
      line-height: 1.6;
    }
    .quick-info-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
      gap: 12px;
    }
    .quick-card {
      background: var(--bg);
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 12px 14px;
      display: flex;
      flex-direction: column;
      gap: 6px;
    }
    .quick-label {
      font-size: 11px;
      font-weight: 600;
      color: var(--text-muted);
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }
    .quick-val-row {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 8px;
    }
    .quick-val {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      font-size: 12px;
      font-weight: 600;
      color: var(--accent);
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .status-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 12px;
    }
    .status-tile {
      background: var(--bg);
      border: 1px solid var(--border-subtle);
      border-radius: 10px;
      padding: 14px;
      display: flex;
      flex-direction: column;
      gap: 6px;
    }
    .status-tile-label {
      font-size: 12px;
      color: var(--text-muted);
    }
    .status-tile-val {
      font-size: 14px;
      font-weight: 600;
      color: var(--text);
      display: flex;
      align-items: center;
      gap: 6px;
    }
    .status-tile-meta {
      font-size: 11px;
      color: var(--text-faint);
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .form-section {
      display: flex;
      flex-direction: column;
      gap: 14px;
      padding-top: 6px;
    }
    .section-title {
      font-size: 14px;
      font-weight: 600;
      color: var(--text);
      border-bottom: 1px solid var(--border-subtle);
      padding-bottom: 8px;
    }
    .toggles-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 12px;
    }
    .toggle-card {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 14px 16px;
      background: var(--bg);
      border: 1px solid var(--border-subtle);
      border-radius: 10px;
      gap: 12px;
    }
    .toggle-info {
      display: flex;
      flex-direction: column;
      gap: 3px;
    }
    .toggle-title {
      font-size: 13px;
      font-weight: 600;
      color: var(--text);
    }
    .toggle-desc {
      font-size: 11px;
      color: var(--text-muted);
      line-height: 1.4;
    }
    .switch {
      position: relative;
      display: inline-block;
      width: 40px;
      height: 22px;
      flex-shrink: 0;
    }
    .switch input {
      opacity: 0;
      width: 0;
      height: 0;
    }
    .slider {
      position: absolute;
      cursor: pointer;
      top: 0; left: 0; right: 0; bottom: 0;
      background-color: var(--border);
      transition: .2s;
      border-radius: 22px;
    }
    .slider:before {
      position: absolute;
      content: "";
      height: 16px;
      width: 16px;
      left: 3px;
      bottom: 3px;
      background-color: white;
      transition: .2s;
      border-radius: 50%;
    }
    input:checked + .slider {
      background-color: var(--accent);
    }
    input:checked + .slider:before {
      transform: translateX(18px);
    }
    .fields-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
      gap: 16px;
    }
    .form-group {
      display: flex;
      flex-direction: column;
      gap: 6px;
    }
    .form-label {
      font-size: 12px;
      font-weight: 600;
      color: var(--text);
    }
    .form-desc {
      font-size: 11px;
      color: var(--text-faint);
    }
    .form-input, .form-select {
      padding: 8px 12px;
      border-radius: 8px;
      border: 1px solid var(--border);
      background: var(--input-bg);
      color: var(--text);
      font-size: 13px;
      outline: none;
      font-family: inherit;
    }
    .form-input:focus, .form-select:focus {
      border-color: var(--accent);
    }
    .form-input.mono {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      font-size: 12px;
    }
    .form-actions {
      display: flex;
      justify-content: flex-end;
      gap: 12px;
      padding-top: 14px;
      border-top: 1px solid var(--border-subtle);
    }
    .modal-backdrop {
      position: fixed;
      inset: 0;
      background: rgba(0, 0, 0, 0.45);
      z-index: 1000;
      display: none;
      align-items: center;
      justify-content: center;
      padding: 16px;
      backdrop-filter: blur(2px);
    }
    .modal-backdrop.active {
      display: flex;
    }
    .modal-card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 14px;
      width: 100%;
      max-width: 480px;
      padding: 20px;
      box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.25);
      display: flex;
      flex-direction: column;
      gap: 16px;
    }
    .modal-title {
      font-size: 16px;
      font-weight: 700;
      color: var(--text);
    }
    .modal-body {
      font-size: 13px;
      color: var(--text-muted);
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    .modal-footer {
      display: flex;
      justify-content: flex-end;
      gap: 10px;
      margin-top: 8px;
    }
    #toast-container {
      position: fixed;
      top: 20px;
      right: 20px;
      z-index: 9999;
      display: flex;
      flex-direction: column;
      gap: 8px;
      pointer-events: none;
    }
    .toast {
      padding: 10px 16px;
      border-radius: 8px;
      font-size: 13px;
      font-weight: 500;
      box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
      pointer-events: auto;
      transition: all 0.2s ease;
      display: flex;
      align-items: center;
      gap: 8px;
      max-width: 360px;
    }
    .toast-success {
      background: var(--success);
      color: #ffffff;
    }
    .toast-error {
      background: var(--danger);
      color: #ffffff;
    }
    .toast-info {
      background: var(--accent);
      color: #ffffff;
    }
  </style>
</head>
<body>
  <div id="toast-container"></div>

  <div class="container">
    <!-- Header -->
    <div class="header">
      <div class="header-left">
        <img class="logo" id="magpie-logo" alt="MAGPIE Logo" />
        <div class="title-group">
          <div class="title-row">
            <h1>MAGPIE 服务设置</h1>
            <span id="service-badge" class="badge badge-neutral"><span class="dot"></span> 加载中...</span>
          </div>
          <p class="subtitle">管理 AIStudio2API 本地守护进程、模型路由、调度策略与网络代理。</p>
        </div>
      </div>
      <div class="toolbar">
        <button id="btn-refresh" class="btn" onclick="fetchStatus(true)">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"/></svg>
          刷新状态
        </button>
        <a href="/v0/resource/plugins/magpie/quota" class="btn btn-primary">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z"/></svg>
          MAGPIE 配额与账号
        </a>
      </div>
    </div>

    <!-- Unified Gateway Banner -->
    <div class="banner">
      <div class="banner-title">
        <span class="badge badge-info">(MAGPIE) 命名空间</span>
        统一 Antigravity / CLIProxyAPI 网关路由
      </div>
      <div class="banner-desc">
        MAGPIE 后端已无缝整合至统一网关。所有来自 Google AI Studio 的模型均自动带有 <code>(MAGPIE)/</code> 前缀（例如：<code>(MAGPIE)/gemini-4-argon</code>）。客户端仅需配置统一 BaseURL 与统一 API Key 即可透明路由访问。
      </div>
      <div class="quick-info-grid">
        <div class="quick-card">
          <div class="quick-label">统一 BaseURL (OpenAI & Gemini)</div>
          <div class="quick-val-row">
            <span class="quick-val" id="val-base-url">http://127.0.0.1:8080</span>
            <button class="btn btn-sm" onclick="copyBaseUrl()">复制</button>
          </div>
        </div>
        <div class="quick-card">
          <div class="quick-label">模型命名空间 (Client Model ID)</div>
          <div class="quick-val-row">
            <span class="quick-val">(MAGPIE)/&lt;upstream_model&gt;</span>
            <span style="font-size: 11px; color: var(--text-muted);">自动透明代理</span>
          </div>
        </div>
        <div class="quick-card">
          <div class="quick-label">预装 Sidecar 核心</div>
          <div class="quick-val-row">
            <span class="quick-val">/app/magpie/aistudio2api</span>
            <span class="badge badge-success" style="padding: 1px 6px; font-size: 10px;">v0.2.19-magpie.1</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Backend Lifecycle Panel -->
    <div class="panel">
      <div class="panel-header">
        <div class="panel-title">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="2" width="20" height="8" rx="2" ry="2"/><rect x="2" y="14" width="20" height="8" rx="2" ry="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>
          后端守护进程状态与生命周期控制
        </div>
        <div class="toolbar">
          <button id="btn-start" class="btn btn-success" onclick="startService()">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="5 3 19 12 5 21 5 3"/></svg>
            启动服务
          </button>
          <button id="btn-stop" class="btn btn-danger" onclick="stopService()">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/></svg>
            停止服务
          </button>
          <button id="btn-restart" class="btn" onclick="restartService()">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"/></svg>
            重启服务
          </button>
        </div>
      </div>

      <div class="status-grid">
        <div class="status-tile">
          <div class="status-tile-label">运行状态</div>
          <div class="status-tile-val" id="tile-status">--</div>
          <div class="status-tile-meta" id="tile-port">内部端口: 2048</div>
        </div>
        <div class="status-tile">
          <div class="status-tile-label">AIStudio2API 核心版本</div>
          <div class="status-tile-val" id="tile-version">v0.2.19-magpie.1</div>
          <div class="status-tile-meta" id="tile-bin-path">/app/magpie/aistudio2api</div>
        </div>
        <div class="status-tile">
          <div class="status-tile-label">Camoufox 环境</div>
          <div class="status-tile-val" id="tile-camoufox">已就绪 (Headless)</div>
          <div class="status-tile-meta">锁定版本: 152.0.4-beta.29</div>
        </div>
        <div class="status-tile">
          <div class="status-tile-label">数据持久化目录</div>
          <div class="status-tile-val" id="tile-data-dir">/data/magpie</div>
          <div class="status-tile-meta">GCS FUSE 分离存储</div>
        </div>
      </div>
    </div>

    <!-- Configuration Form -->
    <form id="config-form" class="panel" onsubmit="handleFormSubmit(event)">
      <div class="panel-header">
        <div class="panel-title">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
          运行参数与模型路由配置 (Runtime & Model Settings)
        </div>
        <span id="dirty-indicator" style="font-size: 12px; color: var(--text-faint); display: none;">有未保存的修改</span>
      </div>

      <!-- Section 1: Features & Toggles -->
      <div class="form-section">
        <div class="section-title">1. 核心路由与特性开关</div>
        <div class="toggles-grid">
          <div class="toggle-card">
            <div class="toggle-info">
              <div class="toggle-title">启用 MAGPIE 整合 (enabled)</div>
              <div class="toggle-desc">在统一网关中注册 (MAGPIE) 系列模型并按需透明代理。</div>
            </div>
            <label class="switch">
              <input type="checkbox" name="enabled" />
              <span class="slider"></span>
            </label>
          </div>

          <div class="toggle-card">
            <div class="toggle-info">
              <div class="toggle-title">随网关自动启动服务 (auto_start)</div>
              <div class="toggle-desc">当网关启动或收到 (MAGPIE) 请求时自动唤醒后台进程。</div>
            </div>
            <label class="switch">
              <input type="checkbox" name="auto_start" />
              <span class="slider"></span>
            </label>
          </div>

          <div class="toggle-card">
            <div class="toggle-info">
              <div class="toggle-title">允许原生工具调用 (allow_tools)</div>
              <div class="toggle-desc">支持 tools/functions 参数，含 >63 字符超长工具名自动哈希双向映射。</div>
            </div>
            <label class="switch">
              <input type="checkbox" name="allow_tools" />
              <span class="slider"></span>
            </label>
          </div>

          <div class="toggle-card">
            <div class="toggle-info">
              <div class="toggle-title">允许多模态图片输入 (allow_images)</div>
              <div class="toggle-desc">允许向 Google AI Studio 发送多模态内联图像或文件数据。</div>
            </div>
            <label class="switch">
              <input type="checkbox" name="allow_images" />
              <span class="slider"></span>
            </label>
          </div>

          <div class="toggle-card" style="grid-column: 1 / -1;">
            <div class="toggle-info">
              <div class="toggle-title">暴露画图生成模型 (image_models)</div>
              <div class="toggle-desc">在模型列表及路由中暴露 Nano Banana 等 AI Studio 生图绘图模型。</div>
            </div>
            <label class="switch">
              <input type="checkbox" name="image_models" />
              <span class="slider"></span>
            </label>
          </div>
        </div>
      </div>

      <!-- Section 2: Concurrency & Worker Strategy -->
      <div class="form-section">
        <div class="section-title">2. 账号调度与 Worker 并发策略</div>
        <div class="fields-grid">
          <div class="form-group">
            <label class="form-label">账号池调度策略 (routing_strategy)</label>
            <select name="routing_strategy" class="form-select">
              <option value="fill-first">fill-first (优先用满单账号配额，耗尽再轮替 - 推荐)</option>
              <option value="round-robin">round-robin (轮询平摊请求至所有就绪账号)</option>
            </select>
            <span class="form-desc">建议使用 fill-first 以最大化利用单一账号配额。</span>
          </div>

          <div class="form-group">
            <label class="form-label">单账号并发限制 (per_account_concurrency)</label>
            <input type="number" name="per_account_concurrency" min="1" class="form-input mono" />
            <span class="form-desc">限制单个 Google 账号同时处理的最大并发调用量 (默认: 1)。</span>
          </div>

          <div class="form-group">
            <label class="form-label">常驻 Worker 进程数 (warm_worker_limit)</label>
            <input type="number" name="warm_worker_limit" min="1" class="form-input mono" />
            <span class="form-desc">保持常驻运行的无头 Camoufox 进程数量 (默认: 1)。</span>
          </div>

          <div class="form-group">
            <label class="form-label">峰值 Worker 上限 (max_active_workers)</label>
            <input type="number" name="max_active_workers" min="1" class="form-input mono" />
            <span class="form-desc">高负载下允许动态唤醒的最大活跃 Worker 进程上限 (&ge; 常驻数)。</span>
          </div>

          <div class="form-group">
            <label class="form-label">Camoufox 启动并发 (warm_startup_concurrency)</label>
            <input type="number" name="warm_startup_concurrency" min="1" class="form-input mono" />
            <span class="form-desc">初始化或扩容 Worker 时允许同时拉起的浏览器进程并发数。</span>
          </div>

          <div class="form-group">
            <label class="form-label">请求超时时间毫秒 (request_timeout_ms)</label>
            <input type="number" name="request_timeout_ms" min="1000" step="1000" class="form-input mono" />
            <span class="form-desc">AI Studio 上游请求的最长挂起超时 (默认 1800000 即 30 分钟)。</span>
          </div>
        </div>
      </div>

      <!-- Section 3: Hedging & Network -->
      <div class="form-section">
        <div class="section-title">3. 首事件对冲与网络代理设置</div>
        <div class="fields-grid">
          <div class="form-group">
            <label class="form-label">首事件对冲策略 (first_event_hedge)</label>
            <select name="first_event_hedge" class="form-select">
              <option value="off">off (关闭对冲)</option>
              <option value="auto">auto (仅指定模型启用对冲 - 推荐)</option>
              <option value="on">on (所有模型强制对冲)</option>
            </select>
            <span class="form-desc">对冲延迟首包以降低首字响应时间 (TTFT)。</span>
          </div>

          <div class="form-group">
            <label class="form-label">对冲目标模型列表 (first_event_hedge_models)</label>
            <input type="text" name="first_event_hedge_models" class="form-input mono" placeholder="gemini-4-argon" />
            <span class="form-desc">指定需启用对冲的模型名称，多个以逗号分隔，默认 gemini-4-argon。</span>
          </div>

          <div class="form-group">
            <label class="form-label">访问 Google 的上游代理 (proxy)</label>
            <input type="text" name="proxy" class="form-input mono" placeholder="http://127.0.0.1:7890 (留空继承系统全局代理)" />
            <span class="form-desc">留空则自动继承网关全局代理配置。云上海外直连无需填写。</span>
          </div>

          <div class="form-group">
            <label class="form-label">守护进程内部监听端口 (internal_port)</label>
            <input type="number" name="internal_port" min="1024" max="65535" class="form-input mono" />
            <span class="form-desc">AIStudio2API 绑定的本地回环端口，默认 2048。</span>
          </div>

          <div class="form-group" style="grid-column: 1 / -1;">
            <label class="form-label">数据持久化存储路径 (data_dir)</label>
            <input type="text" name="data_dir" class="form-input mono" placeholder="/data/magpie" />
            <span class="form-desc">保存 accounts 凭证与会话状态的目录路径。</span>
          </div>
        </div>
      </div>

      <div class="form-actions">
        <button type="submit" id="btn-save" class="btn btn-primary">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>
          保存配置
        </button>
      </div>
    </form>
  </div>

  <!-- Auth Key Modal -->
  <div id="modal-auth" class="modal-backdrop">
    <div class="modal-card">
      <div class="modal-title">管理鉴权验证</div>
      <div class="modal-body">
        <p>检测到当前请求未附带有效管理权限。请输入 Management Password 或 API Key：</p>
        <input type="password" id="auth-key-input" class="form-input" style="width: 100%;" placeholder="Management Password / API Key" />
      </div>
      <div class="modal-footer">
        <button class="btn" onclick="closeAuthModal()">取消</button>
        <button class="btn btn-primary" onclick="saveAuthKey()">保存并重试</button>
      </div>
    </div>
  </div>

  <script>
    // Injected MAGPIE SVG Logo
    var LOGO_DATA_URI = "PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyNCIgaGVpZ2h0PSIyNCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSJub25lIj48ZGVmcz48bGluZWFyR3JhZGllbnQgaWQ9ImEiIHgxPSIyIiB5MT0iMyIgeDI9IjIwIiB5Mj0iMjEiIGdyYWRpZW50VW5pdHM9InVzZXJTcGFjZU9uVXNlIj48c3RvcCBvZmZzZXQ9IjAlIiBzdG9wLWNvbG9yPSIjMDZCNkQ0Ii8+PHN0b3Agb2Zmc2V0PSI1MCUiIHN0b3AtY29sb3I9IiMzQjgyRjYiLz48c3RvcCBvZmZzZXQ9IjEwMCUiIHN0b3AtY29sb3I9IiM4QjVDRjYiLz48L2xpbmVhckdyYWRpZW50PjxsaW5lYXJHcmFkaWVudCBpZD0iYiIgeDE9IjE0IiB5MT0iMiIgeDI9IjIyIiB5Mj0iMTAiIGdyYWRpZW50VW5pdHM9InVzZXJTcGFjZU9uVXNlIj48c3RvcCBvZmZzZXQ9IjAlIiBzdG9wLWNvbG9yPSIjOEI1Q0Y2Ii8+PHN0b3Agb2Zmc2V0PSIxMDAlIiBzdG9wLWNvbG9yPSIjRUM0ODk5Ii8+PC9saW5lYXJHcmFkaWVudD48L2RlZnM+PHBhdGggZD0iTTEwIDMuMkMxMC4zNSAzLjIgMTAuNjIgMy40NSAxMC43MiAzLjhDMTEuNDUgNi44NSAxMy4xNSA4LjU1IDE2LjIgOS4yOEMxNi41NSA5LjM4IDE2LjggOS42NSAxNi44IDEwQzE2LjggMTAuMzUgMTYuNTUgMTAuNjIgMTYuMiAxMC43MkMxMy4xNSAxMS40NSAxMS40NSAxMy4xNSAxMC43MiAxNi4yQzEwLjYyIDE2LjU1IDEwLjM1IDE2LjggMTAgMTYuOEM5LjY1IDE2LjggOS4zOCAxNi41NSA5LjI4IDE2LjJDOC41NSAxMy4xNSA2Ljg1IDExLjQ1IDMuOCAxMC43MkMzLjQ1IDEwLjYyIDMuMiAxMC4zNSAzLjIgMTBDMy4yIDkuNjUgMy40NSA5LjM4IDMuOCA5LjI4QzYuODUgOC41NSA4LjU1IDYuODUgOS4yOCAzLjhDOS4zOCAzLjQ1IDkuNjUgMy4yIDEwIDMuMloiIHRyYW5zZm9ybT0idHJhbnNsYXRlKC0wLjUgMi41KSBzY2FsZSgxLjEyKSIgZmlsbD0idXJsKCNhKSIvPjxwYXRoIGQ9Ik0xOC4yIDEuOEMxOC40MiAxLjggMTguNiAxLjk1IDE4LjY2IDIuMTZDMTkuMDggMy44MiAyMC4wMiA0Ljc2IDIxLjY4IDUuMThDMjEuODkgNS4yNCAyMi4wNCA1LjQyIDIyLjA0IDUuNjRDMjIuMDQgNS44NiAyMS44OSA2LjA0IDIxLjY4IDYuMUMyMC4wMiA2LjUyIDE5LjA4IDcuNDYgMTguNjYgOS4xMkMxOC42IDkuMzMgMTguNDIgOS40OCAxOC4yIDkuNDhDMTcuOTggOS40OCAxNy44IDkuMzMgMTcuNzQgOS4xMkMxNy4zMiA3LjQ2IDE2LjM4IDYuNTIgMTQuNzIgNi4xQzE0LjUxIDYuMDQgMTQuMzYgNS44NiAxNC4zNiA1LjY0QzE0LjM2IDUuNDIgMTQuNTEgNS4yNCAxNC43MiA1LjE4QzE2LjM4IDQuNzYgMTcuMzIgMy44MiAxNy43NCAyLjE2QzE3LjggMS45NSAxNy45OCAxLjggMTguMiAxLjhaIiBmaWxsPSJ1cmwoI2IpIi8+PGNpcmNsZSBjeD0iMTcuOCIgY3k9IjE4LjIiIHI9IjEuMzUiIGZpbGw9IiM2MzY2RjEiLz48L3N2Zz4=";
    var logoImg = document.getElementById('magpie-logo');
    if (logoImg) logoImg.src = "data:image/svg+xml;base64," + LOGO_DATA_URI;

    // Theme sync
    function initThemeSync() {
      function getTheme() {
        try {
          if (window.parent && window.parent.document && window.parent.document.documentElement) {
            var t = window.parent.document.documentElement.getAttribute('data-theme');
            if (t) return t;
          }
        } catch (e) {}
        try {
          var local = localStorage.getItem('cli-proxy-theme');
          if (local) return local;
        } catch (e) {}
        if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
          return 'dark';
        }
        return 'light';
      }
      function applyTheme(t) {
        document.documentElement.setAttribute('data-theme', t);
      }
      applyTheme(getTheme());
      try {
        if (window.parent && window.parent.document && window.parent.document.documentElement) {
          var obs = new MutationObserver(function() {
            var t = window.parent.document.documentElement.getAttribute('data-theme');
            if (t) applyTheme(t);
          });
          obs.observe(window.parent.document.documentElement, {
            attributes: true,
            attributeFilter: ['data-theme']
          });
        }
      } catch (e) {}
      window.addEventListener('storage', function(e) {
        if (e.key === 'cli-proxy-theme' && e.newValue) {
          applyTheme(e.newValue);
        }
      });
      if (window.matchMedia) {
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function(e) {
          if (!localStorage.getItem('cli-proxy-theme')) {
            applyTheme(e.matches ? 'dark' : 'light');
          }
        });
      }
    }
    initThemeSync();

    // Toast
    function showToast(msg, type) {
      type = type || 'info';
      var container = document.getElementById('toast-container');
      if (!container) return;
      var el = document.createElement('div');
      el.className = 'toast toast-' + type;
      el.textContent = msg;
      container.appendChild(el);
      setTimeout(function() {
        el.style.opacity = '0';
        el.style.transform = 'translateY(-10px)';
        setTimeout(function() {
          if (el.parentNode) el.parentNode.removeChild(el);
        }, 200);
      }, 3000);
    }

    // Copy helper
    function copyText(text) {
      if (!text) return;
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(function() {
          showToast('已复制到剪贴板', 'success');
        }).catch(function() {
          fallbackCopy(text);
        });
      } else {
        fallbackCopy(text);
      }
    }
    function fallbackCopy(text) {
      var ta = document.createElement('textarea');
      ta.value = text;
      document.body.appendChild(ta);
      ta.select();
      try {
        document.execCommand('copy');
        showToast('已复制到剪贴板', 'success');
      } catch (e) {
        showToast('复制失败', 'error');
      }
      document.body.removeChild(ta);
    }

    function copyBaseUrl() {
      var el = document.getElementById('val-base-url');
      if (el) copyText(el.textContent);
    }

    // Set BaseURL from origin
    try {
      var origin = window.location.origin;
      var urlEl = document.getElementById('val-base-url');
      if (urlEl && origin) urlEl.textContent = origin;
    } catch (e) {}

    // API fetcher
    var pendingAuthRetry = null;
    function fetchApi(endpoint, options) {
      options = options || {};
      var headers = Object.assign({
        'Content-Type': 'application/json'
      }, options.headers || {});

      try {
        var key = localStorage.getItem('cli-proxy-auth') || sessionStorage.getItem('cli-proxy-auth');
        if (key && !headers['Authorization']) {
          headers['Authorization'] = 'Bearer ' + key;
          headers['X-Management-Key'] = key;
        }
      } catch (e) {}

      options.headers = headers;
      options.credentials = 'same-origin';
      options.cache = 'no-store';

      return fetch(endpoint, options).then(function(res) {
        if (res.status === 401 || res.status === 403) {
          pendingAuthRetry = { endpoint: endpoint, options: options };
          openAuthModal();
          throw new Error('需要管理权限认证 (HTTP ' + res.status + ')');
        }
        if (!res.ok) {
          return res.text().then(function(t) {
            var msg = t;
            try {
              var j = JSON.parse(t);
              msg = j.error || j.message || t;
            } catch (err) {}
            throw new Error(msg || ('HTTP ' + res.status));
          });
        }
        var ct = res.headers.get('content-type') || '';
        if (ct.indexOf('application/json') !== -1) {
          return res.json();
        }
        return res.text();
      });
    }

    function openAuthModal() {
      var m = document.getElementById('modal-auth');
      if (m) m.classList.add('active');
    }
    function closeAuthModal() {
      var m = document.getElementById('modal-auth');
      if (m) m.classList.remove('active');
    }
    function saveAuthKey() {
      var input = document.getElementById('auth-key-input');
      if (!input) return;
      var val = input.value.trim();
      if (!val) return;
      try {
        localStorage.setItem('cli-proxy-auth', val);
      } catch (e) {}
      closeAuthModal();
      showToast('管理密钥已保存，正在重试请求...', 'info');
      if (pendingAuthRetry) {
        fetchApi(pendingAuthRetry.endpoint, pendingAuthRetry.options).then(function() {
          fetchStatus(false);
        }).catch(function(err) {
          showToast(err.message, 'error');
        });
        pendingAuthRetry = null;
      }
    }

    // State & Dirty Field Tracking
    var isDirty = false;
    var focusedField = null;
    var currentStatus = null;

    function setupDirtyTracking() {
      var form = document.getElementById('config-form');
      if (!form) return;
      var inputs = form.querySelectorAll('input, select');
      inputs.forEach(function(el) {
        el.addEventListener('focus', function() {
          focusedField = el.name;
        });
        el.addEventListener('blur', function() {
          focusedField = null;
          if (el.type === 'number' || el.type === 'text') {
            triggerAutoSave();
          }
        });
        el.addEventListener('input', function() {
          setDirty(true);
        });
        el.addEventListener('change', function() {
          setDirty(true);
          if (el.type === 'checkbox' || el.tagName === 'SELECT') {
            triggerAutoSave();
          }
        });
      });
    }
    setupDirtyTracking();

    function setDirty(val) {
      isDirty = val;
      var ind = document.getElementById('dirty-indicator');
      if (ind) {
        ind.style.display = val ? 'inline' : 'none';
      }
    }

    // Update form values with dirty protection
    function updateFormValues(cfg) {
      if (!cfg) return;
      var form = document.getElementById('config-form');
      if (!form) return;

      for (var key in cfg) {
        var el = form.elements[key];
        if (!el) continue;

        // Skip fields that are currently focused or if user has unsaved edits in active focus
        if (focusedField === key || (isDirty && focusedField !== null)) {
          continue;
        }

        if (el.type === 'checkbox') {
          el.checked = Boolean(cfg[key]);
        } else {
          el.value = cfg[key] !== undefined && cfg[key] !== null ? cfg[key] : '';
        }
      }
    }

    // Read form values with client-side normalization
    function getFormData() {
      var form = document.getElementById('config-form');
      var concurrency = Math.max(1, parseInt(form.elements.per_account_concurrency.value, 10) || 1);
      var warmLimit = Math.max(1, parseInt(form.elements.warm_worker_limit.value, 10) || 1);
      var maxWorkers = Math.max(warmLimit, parseInt(form.elements.max_active_workers.value, 10) || 1);
      var warmStartup = Math.min(maxWorkers, Math.max(1, parseInt(form.elements.warm_startup_concurrency.value, 10) || 1));

      return {
        enabled: form.elements.enabled.checked,
        auto_start: form.elements.auto_start.checked,
        allow_tools: form.elements.allow_tools.checked,
        allow_images: form.elements.allow_images.checked,
        image_models: form.elements.image_models.checked,
        routing_strategy: form.elements.routing_strategy.value || 'fill-first',
        per_account_concurrency: concurrency,
        warm_worker_limit: warmLimit,
        max_active_workers: maxWorkers,
        warm_startup_concurrency: warmStartup,
        first_event_hedge: form.elements.first_event_hedge.value || 'off',
        first_event_hedge_models: form.elements.first_event_hedge_models.value || 'gemini-4-argon',
        request_timeout_ms: Math.max(1000, parseInt(form.elements.request_timeout_ms.value, 10) || 1800000),
        proxy: (form.elements.proxy.value || '').trim(),
        internal_port: parseInt(form.elements.internal_port.value, 10) || 2048,
        data_dir: (form.elements.data_dir.value || '').trim()
      };
    }

    // Auto-save debounce
    var autoSaveTimer = null;
    function triggerAutoSave() {
      if (autoSaveTimer) clearTimeout(autoSaveTimer);
      autoSaveTimer = setTimeout(function() {
        saveConfig(false);
      }, 500);
    }

    function saveConfig(isManual) {
      var d = getFormData();
      var btn = document.getElementById('btn-save');
      if (btn && isManual) {
        btn.disabled = true;
        btn.textContent = '保存中...';
      }

      fetchApi('/v8/management/magpie/config', {
        method: 'POST',
        body: JSON.stringify({ config: d, enabled: d.enabled, auto_start: d.auto_start, allow_tools: d.allow_tools, allow_images: d.allow_images, image_models: d.image_models, routing_strategy: d.routing_strategy, per_account_concurrency: d.per_account_concurrency, warm_worker_limit: d.warm_worker_limit, max_active_workers: d.max_active_workers, warm_startup_concurrency: d.warm_startup_concurrency, first_event_hedge: d.first_event_hedge, first_event_hedge_models: d.first_event_hedge_models, request_timeout_ms: d.request_timeout_ms, proxy: d.proxy, internal_port: d.internal_port, data_dir: d.data_dir })
      }).then(function(res) {
        setDirty(false);
        showToast('MAGPIE 配置保存成功', 'success');
        var saved = (res && res.data) ? res.data : res;
        if (saved && (saved.config || saved.enabled !== undefined)) {
          updateFormValues(saved.config || saved);
        }
      }).catch(function(err) {
        showToast('保存配置失败: ' + err.message, 'error');
      }).finally(function() {
        if (btn && isManual) {
          btn.disabled = false;
          btn.textContent = '保存配置';
        }
      });
    }

    function handleFormSubmit(e) {
      if (e) e.preventDefault();
      saveConfig(true);
    }

    // Render Status Tiles
    function renderStatusTiles(statusData) {
      var proc = statusData ? statusData.process : null;
      var install = statusData ? statusData.install : null;

      var isRunning = proc && proc.running;
      var port = (proc && proc.port) ? proc.port : 2048;

      var badge = document.getElementById('service-badge');
      var tileStatus = document.getElementById('tile-status');
      var tilePort = document.getElementById('tile-port');
      var tileVer = document.getElementById('tile-version');
      var tileBin = document.getElementById('tile-bin-path');
      var tileData = document.getElementById('tile-data-dir');
      var btnStart = document.getElementById('btn-start');
      var btnStop = document.getElementById('btn-stop');

      if (isRunning) {
        if (badge) {
          badge.className = 'badge badge-success';
          badge.innerHTML = '<span class="dot"></span> 运行中 (' + port + ')';
        }
        if (tileStatus) {
          tileStatus.innerHTML = '<span class="dot" style="background:var(--success);"></span> 运行中 (' + (proc.state || 'RUNNING') + ')';
          tileStatus.style.color = 'var(--success)';
        }
        if (btnStart) btnStart.style.display = 'none';
        if (btnStop) btnStop.style.display = 'inline-flex';
      } else {
        if (badge) {
          badge.className = 'badge badge-warning';
          badge.innerHTML = '<span class="dot"></span> 已停止';
        }
        if (tileStatus) {
          tileStatus.innerHTML = '<span class="dot" style="background:var(--warning);"></span> 已停止';
          tileStatus.style.color = 'var(--warning)';
        }
        if (btnStart) btnStart.style.display = 'inline-flex';
        if (btnStop) btnStop.style.display = 'none';
      }

      if (tilePort) tilePort.textContent = '内部端口: ' + port;
      if (tileVer && install && install.version) tileVer.textContent = 'v' + install.version;
      if (tileBin && install && install.binary_path) tileBin.textContent = install.binary_path;
      if (tileData && install && install.data_dir) tileData.textContent = install.data_dir;
    }

    // Fetch Status & Config
    function fetchStatus(isManual) {
      var btn = document.getElementById('btn-refresh');
      if (btn && isManual) btn.disabled = true;

      fetchApi('/v8/management/magpie/status').then(function(res) {
        var data = (res && res.data) ? res.data : res;
        currentStatus = data;
        renderStatusTiles(data);
        if (data && data.config) {
          updateFormValues(data.config);
        }
        if (isManual) {
          showToast('MAGPIE 状态已刷新', 'success');
        }
      }).catch(function(err) {
        console.error('Fetch status error:', err);
        if (isManual) {
          showToast('刷新状态失败: ' + err.message, 'error');
        }
      }).finally(function() {
        if (btn) btn.disabled = false;
      });
    }

    // Service Controls
    function startService() {
      var btn = document.getElementById('btn-start');
      if (btn) btn.disabled = true;
      showToast('正在启动 MAGPIE 服务...', 'info');

      fetchApi('/v8/management/magpie/start', { method: 'POST' }).then(function() {
        showToast('MAGPIE 服务启动成功', 'success');
        fetchStatus(false);
      }).catch(function(err) {
        showToast('启动失败: ' + err.message, 'error');
      }).finally(function() {
        if (btn) btn.disabled = false;
      });
    }

    function stopService() {
      var btn = document.getElementById('btn-stop');
      if (btn) btn.disabled = true;
      showToast('正在停止 MAGPIE 服务...', 'info');

      fetchApi('/v8/management/magpie/stop', { method: 'POST' }).then(function() {
        showToast('MAGPIE 服务已停止', 'success');
        fetchStatus(false);
      }).catch(function(err) {
        showToast('停止失败: ' + err.message, 'error');
      }).finally(function() {
        if (btn) btn.disabled = false;
      });
    }

    function restartService() {
      var btn = document.getElementById('btn-restart');
      if (btn) btn.disabled = true;
      showToast('正在重启 MAGPIE 服务...', 'info');

      fetchApi('/v8/management/magpie/restart', { method: 'POST' }).then(function() {
        showToast('MAGPIE 服务重启成功', 'success');
        fetchStatus(false);
      }).catch(function(err) {
        showToast('重启失败: ' + err.message, 'error');
      }).finally(function() {
        if (btn) btn.disabled = false;
      });
    }

    // Initial load
    fetchStatus(false);
    setInterval(function() {
      fetchStatus(false);
    }, 15000);
  </script>
</body>
</html>`
