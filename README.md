# AI Subtitle Pro

A Windows desktop app (**Go + Wails**) that translates a video's subtitles from a
source language into a target language of your choice — either **locally on the
GPU** (llama.cpp, CUDA build) or via the **Gemini API**.

> A Go rewrite of an older Python/PyQt tool, with a redesigned UI, **batch
> translation with context** (coherent dialogue), faithful **ASS** handling
> (positioning and styles preserved), and a single-executable distribution.

## How it works

1. Drag & drop a video (`.mkv`, `.mp4`, `.avi`) — or browse.
2. Pick the **subtitle track**, the **source** and **target** languages, and the
   **engine** (Local GPU or Gemini).
3. Subtitles are translated in batches with surrounding context, then reassembled.
4. A new subtitle track (default-flagged) is muxed back in →
   `{name}_PRO_FR.mkv` (or `{name}_TEST_20s.mkv` in Test mode).

For **ASS/SubStation** tracks the original file is preserved byte-for-byte
(header, styles, positioning) and only the dialogue text is replaced, so the
rendering matches the original exactly. SRT/VTT are handled via `go-astisub`.

## Distribution

- A **single `.exe`** plus a **`models/`** folder (your `.gguf` files) alongside it.
- `mkvmerge` / `mkvextract` (MKVToolNix) are **embedded** and auto-extracted to
  `%LOCALAPPDATA%\AISubtitlePro\bin\`.
- On the first Local run, the app **auto-downloads** (into that cache / `models/`):
  - the `llama-server` **CUDA** runtime (~620 MB), and
  - a default model **Gemma 3 12B (Q4, ~7 GB)** if no `.gguf` is present.

  An internet connection is needed that first time; everything is local afterwards.

## Development

**Prerequisites:** Go 1.23+, Node 18+, and the Wails CLI:
```
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

**Prepare the embedded binaries (before building):** copy `mkvmerge.exe` and
`mkvextract.exe` (from https://mkvtoolnix.download/) into
`internal/runtime/binaries/`. They are not versioned (see `.gitignore`).

**Run / build:**
```
wails dev      # development (hot reload)
wails build    # produces build/bin/AISubtitlePro.exe
```

**Tests:**
```
go test ./...
# optional MKV integration tests, pointing at a local MKVToolNix install:
AISUBTITLE_MKVTOOLNIX_DIR="C:\path\to\mkvtoolnix" go test ./internal/mkv/
```

## Architecture

| Package | Responsibility |
|---|---|
| `app.go` / `main.go` | Wails bindings + window |
| `internal/config` | load/save settings |
| `internal/runtime` | embed mkvtoolnix + download llama-server (CUDA) and the default model |
| `internal/mkv` | scan / extract / mux (MKVToolNix) |
| `internal/subs` | read/write SRT·VTT (go-astisub) and **raw ASS** + batching with context |
| `internal/engine` | `Translator` interface + `Local` (llama-server) + `Gemini` |
| `internal/pipeline` | orchestration: extract → translate → remux |
| `internal/winproc` | hide console windows (Windows) |

## Licenses of bundled components

- **MKVToolNix** (mkvmerge, mkvextract) — GPL-2.0
- **llama.cpp** (llama-server) — MIT
- **Gemma** model — Google Gemma Terms of Use
