# InstaGo

Local-first desktop workspace for planning, designing, versioning, and automating Instagram campaigns — built with **Go**, **Wails 3**, **Svelte + TypeScript**, **Fabric.js**, **SQLite**, local **AI CLI** integration, and a planned **Android ADB** publishing bridge.

> **Status:** Milestones **R0–R3** (`0.0.0-dev`) — native **Wails 3** + **Svelte 5** + **Fabric.js 7** + **SQLite** desktop workspace with managed SHA-256 asset library, portable `.zip` backups, multi-page free-form editor (Feed `1080×1350`, Story `1080×1920`, carousels), versioned revisions with approval locks and bundle export, local **Codex CLI** JSONL agent orchestration with visual canvas annotations, and Linux desktop/taskbar integration. Future milestones (**R4–R8**) cover monthly campaign planning, date research, packaging/restore, Windows, and optional Android ADB automation.

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

## Implemented Scope (R0–R3)

- **Foundation & native shell (R0):** Wails 3 desktop window (`GTK4/WebKitGTK 6.0` and `GTK3/WebKit2GTK 4.1` build targets), Go/Svelte bridge, SQLite migrations, and deterministic PNG/JPG raster export.
- **Asset library, profiles & backups (R1):** Managed `$INSTAGO_DATA_DIR/library` storage with SHA-256 indexing, concurrent thumbnail generation, category filters, company profile settings, integrity checks, and portable `.zip` backup export.
- **Free-form multi-page editor & versioning (R2):** Layer manipulation, alignment helpers, image crop/framing (`cover`/`contain`), Feed/Story/carousel pagination, continuous draft autosave, numbered snapshots with side-by-side comparison, approval immutability, commercial offer price-conflict warnings, and numbered export bundles (`01.png`, `legenda.txt`, `manifesto.json`).
- **Local AI CLI integration (R3):** `os/exec` Codex CLI runner with ChatGPT login detection, live JSONL log streaming, single-task queue, cancellation, partial artifact recovery, visual pin/rectangle annotations on canvas, and floating editor assistant widget.
- **Linux desktop integration:** Embedded SVG/PNG app icon and `make install-desktop` helper (`scripts/linux/install-desktop.sh`) registering the `.desktop` entry and hicolor icons.

---

## Architecture & Planned Stack

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

- **R0 — Foundation & native editor proof (Implemented):** Go/Wails 3 + Svelte + Fabric.js + SQLite scaffold, single-composition conversion, native Ubuntu window, save/reopen/export baseline.
- **R1 — Profiles, asset library & base import (Implemented):** Company profiles, SHA-256 asset indexing, thumbnails, filters, integrity check, and `.zip` backup export.
- **R2 — Free-form editor, multi-page & versioning (Implemented):** Full layer editing, carousel/Story pagination, version diffs, approval locks, and numbered export bundles.
- **R3 — Local AI CLI integration (Implemented):** `os/exec` JSONL runner, native image generation, visual canvas annotations, cancellation, error handling, and artifact recovery.
- **R4 — Campaigns, date research & monthly planner (Next):** Source-backed calendar research, configurable cadence, batch queue after approval.
- **R5 — Usable MVP delivery (Ubuntu):** Batch export bundles, operational tracking, backup restore verification, and packaging.
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

Install local Linux `.desktop` entry and hicolor icons for the GTK3 desktop binary:

```bash
make install-desktop
```

Local Git hooks (optional, recommended for contributors):

```bash
git config core.hooksPath .githooks
```

---

## License

[MIT](LICENSE)
