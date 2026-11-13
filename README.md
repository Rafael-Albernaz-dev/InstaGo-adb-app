# InstaGo

Local-first desktop workspace for planning, designing, versioning, and automating Instagram campaigns — built with **Go**, **Wails 3**, **Svelte + TypeScript**, **Fabric.js**, **SQLite**, local **AI CLI** integration, and a planned **Android ADB** publishing bridge.

> **Status:** Milestone **R0** (`0.0.0-dev`) — native **Wails 3** + **Svelte 5** + **Fabric.js 7** + **SQLite** canvas editor scaffold with versioned revisions, undo/redo history, and verified `1080×1350` PNG/JPG export. Future milestones (**R1–R8**) cover full campaign planning, Codex CLI orchestration, and optional Android ADB automation.

---

## Overview

**InstaGo** centralizes the end-to-end social media production workflow in a lightweight desktop app (Ubuntu/Linux first, Windows planned later):

- **Editorial calendar & date research:** Track national, state, municipal, and commercial dates with explicit source attribution and timezone-safe scheduling (`America/Sao_Paulo`).
- **Offer & campaign catalog:** Structured, versioned commercial offers so price/date changes flag affected creative pieces instead of silently drifting.
- **Interactive visual canvas editor:** Free-form canvas powered by **Fabric.js** (text, photos, logos, shapes, multi-page carousels, Stories `1080×1920`, Feed `1080×1350`, undo/redo, and atomic local persistence).
- **Local AI CLI orchestration:** Go executes the local Codex CLI over `os/exec` with structured JSONL event streaming, explicit task states, cancellation, and artifact verification — no silent paid-API fallback.
- **Approval, versioning & organized export (MVP):** Versioned compositions, captions/CTAs, manifests, and manual-publishing export bundles.
- **Android ADB & scheduler bridge (Post-MVP):** Replaceable publishing connector for device discovery, media transfer, assisted Story/Feed workflows, and crash-safe scheduling recovery.

---

## Planned Stack & Architecture

| Layer | Technology | Role |
|---|---|---|
| Desktop shell | Wails 3 (pinned stable release) | Native window, frontend/Go bridge, file dialogs |
| Backend core | Go | Domain logic, campaign state, task queue, CLI & ADB adapters |
| Frontend UI | Svelte + TypeScript + Vite | Wireframe-first operator UI and editor controls |
| Visual canvas | Fabric.js | Editable layered compositions and raster export |
| Persistence | SQLite (`database/sql` + `modernc.org/sqlite`) | Relational metadata, migrations, audit trail |
| Styling & icons | Plain CSS + Lucide | Lightweight, high-contrast light-mode interface |

Architectural boundaries follow pragmatic Clean Architecture: domain rules and use cases stay independent from Wails, SQLite drivers, AI CLIs, and future ADB/API connectors.

---

## Roadmap

- **R0 — Foundation & native editor proof:** Go/Wails 3 + Svelte + Fabric.js + SQLite scaffold, single-composition conversion, native Ubuntu window, save/reopen/export baseline.
- **R1 — Profiles, asset library & base import:** Company profiles, SHA-256 asset indexing, thumbnails, filters, backup/restore.
- **R2 — Free-form editor, multi-page & versioning:** Full layer editing, carousel/Story pagination, version diffs, and approval locks.
- **R3 — Local AI CLI integration:** `os/exec` JSONL runner, native image generation, cancellation, quota/error handling, and artifact recovery.
- **R4 — Campaigns, date research & monthly planner:** Source-backed calendar research, configurable cadence, batch queue after approval.
- **R5 — Usable MVP delivery (Ubuntu):** Batch export bundles, operational tracking, backup/restore verification, and packaging.
- **R6 — Experimental Android ADB connector:** Device checks, media transfer, single-execution locking, and observed-result verification.
- **R7 — Automated scheduling & recovery:** Persistent worker queue, missed-schedule policy, and official API capability evaluation.
- **R8 — Windows support & multi-company evolution:** WebView2 packaging, cross-platform adapters, and multi-brand workspaces.

---

## Build & Checks

Core checks (repo hygiene, `gofmt`, `tsc --noEmit`, Vite build, Vitest, Go `vet` and `-race` tests on domain/sqlite/assets/editor):

```bash
make check-core
```

Native Linux build & full test suite (requires `libgtk-4-dev` and `libwebkitgtk-6.0-dev` on Ubuntu 24.04):

```bash
make check-native build
```

Alternative GTK3/WebKit2GTK 4.1 fallback targets (when `gtk+-3.0` and `webkit2gtk-4.1` headers are used):

```bash
make check-gtk3 build-gtk3
```

Local Git hooks (optional, recommended for contributors):

```bash
git config core.hooksPath .githooks
```

---

## License

[MIT](LICENSE)
