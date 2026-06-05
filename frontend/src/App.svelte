<script>
  import { onMount } from "svelte";
  import {
    SelectVideo, ScanTracks, ListModels, GetConfig, SaveConfig,
    StartTranslation, Cancel,
  } from "../wailsjs/go/main/App.js";
  import { EventsOn, OnFileDrop } from "../wailsjs/runtime/runtime.js";

  const LANGS = ["ANGLAIS", "FRANÇAIS", "JAPONAIS", "ESPAGNOL", "ALLEMAND", "ITALIEN", "PORTUGAIS", "RUSSE", "CHINOIS", "CORÉEN", "ARABE", "NÉERLANDAIS", "POLONAIS", "TURC"];
  const GEMINI_MODELS = ["gemini-2.0-flash", "gemini-1.5-flash", "gemini-1.5-pro"];

  let videoPath = "", videoName = "", dragging = false, scanning = false;
  let engine = "Local";
  let models = [], localModel = "", geminiModel = GEMINI_MODELS[0], apiKey = "", srcLang = "ANGLAIS", tgtLang = "FRANÇAIS";
  let tracks = [], selectedTrackId = null;
  let running = false, progress = { done: 0, total: 0 }, download = null;
  let logs = [], result = "", errorMsg = "", logEl;

  $: pct = progress.total > 0 ? Math.round((progress.done / progress.total) * 100) : 0;
  $: selectedTrack = tracks.find((t) => t.ID === selectedTrackId) || null;
  $: canStart =
    !!videoPath && selectedTrack && !selectedTrack.IsImageBased &&
    (engine === "Local" ? (models.length === 0 || !!localModel) : apiKey.trim().length > 0) && !running;

  onMount(async () => {
    try {
      const cfg = await GetConfig();
      engine = cfg.mode === "Gemini" ? "Gemini" : "Local";
      srcLang = cfg.source_lang || "ANGLAIS";
      tgtLang = cfg.target_lang || "FRANÇAIS";
      apiKey = cfg.api_key || "";
      if (engine === "Gemini" && cfg.model) geminiModel = cfg.model;
      else if (engine === "Local" && cfg.model) localModel = cfg.model;
    } catch (e) {}
    await refreshModels();

    EventsOn("progress", (d) => (progress = d));
    EventsOn("log", (m) => pushLog(m));
    EventsOn("download", (d) => (download = d));
    EventsOn("done", (out) => {
      running = false; download = null;
      result = "Vidéo créée : " + baseName(out);
    });
    EventsOn("error", (m) => {
      running = false; download = null;
      if (m && m !== "Annulé.") errorMsg = m;
    });

    OnFileDrop((x, y, paths) => {
      dragging = false;
      if (paths && paths.length) setVideo(paths[0]);
    }, false);
  });

  async function refreshModels() {
    try {
      models = (await ListModels()) || [];
      if (!localModel && models.length) localModel = models[0];
    } catch (e) { models = []; }
  }

  const baseName = (p) => (p || "").split(/[\\/]/).pop();

  function pushLog(m) {
    logs = [...logs, m];
    queueMicrotask(() => { if (logEl) logEl.scrollTop = logEl.scrollHeight; });
  }

  async function browse() {
    try { const p = await SelectVideo(); if (p) setVideo(p); }
    catch (e) { pushLog("❌ " + e); }
  }

  async function setVideo(path) {
    const lower = (path || "").toLowerCase();
    if (!(lower.endsWith(".mkv") || lower.endsWith(".mp4") || lower.endsWith(".avi"))) {
      pushLog("⚠️ Format non supporté (mkv, mp4, avi)."); return;
    }
    videoPath = path; videoName = baseName(path);
    tracks = []; selectedTrackId = null; result = ""; errorMsg = "";
    scanning = true;
    try {
      const t = (await ScanTracks(path)) || [];
      tracks = t;
      const firstText = t.find((x) => !x.IsImageBased);
      selectedTrackId = firstText ? firstText.ID : (t.length ? t[0].ID : null);
      if (!t.length) pushLog("⚠️ Aucune piste de sous-titres trouvée dans ce fichier.");
    } catch (e) { pushLog("❌ Scan : " + e); }
    scanning = false;
  }

  function trackLabel(t) {
    let s = `Piste ${t.ID} · ${t.Language || "und"}`;
    if (t.Codec) s += ` · ${t.Codec}`;
    if (t.IsImageBased) s += "  (image — non traduisible)";
    return s;
  }

  async function start(testMode) {
    if (!canStart) return;
    await persist();
    running = true; result = ""; errorMsg = "";
    progress = { done: 0, total: 0 }; download = null;
    StartTranslation({
      videoPath, engine,
      model: engine === "Local" ? localModel : geminiModel,
      apiKey, srcLang, targetLang: tgtLang, trackID: selectedTrackId, testMode,
    });
  }

  function cancel() { Cancel(); pushLog("⏹ Annulation demandée…"); }

  async function persist() {
    try {
      await SaveConfig({
        mode: engine,
        model: engine === "Local" ? localModel : geminiModel,
        api_key: apiKey, source_lang: srcLang, target_lang: tgtLang,
        batch_size: 12, context_size: 2,
      });
    } catch (e) {}
  }

  const fmtMB = (n) => (n / 1048576).toFixed(0);
</script>

<main>
  <header class="topbar">
    <div class="logo">AI</div>
    <div>
      <h1>AI Subtitle Pro</h1>
      <p class="tag">Traduction de sous-titres par IA — local (CUDA) ou Gemini</p>
    </div>
  </header>

  <button
    class="dropzone"
    class:drag={dragging}
    class:has={!!videoPath}
    on:click={browse}
    on:dragenter|preventDefault={() => (dragging = true)}
    on:dragover|preventDefault={() => (dragging = true)}
    on:dragleave|preventDefault={() => (dragging = false)}
    on:drop|preventDefault={() => (dragging = false)}
  >
    {#if videoPath}
      <div class="film">🎬</div>
      <div class="fname">{videoName}</div>
      <div class="hint">{scanning ? "Analyse des pistes…" : "Cliquer ou déposer pour changer"}</div>
    {:else}
      <div class="film">⬇</div>
      <div class="fname">Glissez une vidéo ici</div>
      <div class="hint">ou cliquez pour parcourir · mkv, mp4, avi</div>
    {/if}
  </button>

  <section class="card">
    <div class="seg">
      <button class:active={engine === "Local"} on:click={() => (engine = "Local")} disabled={running}>Local · GPU</button>
      <button class:active={engine === "Gemini"} on:click={() => (engine = "Gemini")} disabled={running}>Gemini · API</button>
    </div>

    {#if engine === "Local"}
      <label class="field">
        <span>Modèle (.gguf)</span>
        {#if models.length}
          <select bind:value={localModel} disabled={running}>
            {#each models as m}<option value={m}>{m}</option>{/each}
          </select>
        {:else}
          <div class="empty"><strong>Gemma 3 12B</strong> (~7 Go) sera téléchargé automatiquement au 1er lancement. (Ou placez vos <code>.gguf</code> dans <code>models/</code>.)</div>
        {/if}
      </label>
    {:else}
      <label class="field">
        <span>Clé API Gemini</span>
        <input type="password" bind:value={apiKey} placeholder="AIza…" disabled={running} />
      </label>
      <label class="field">
        <span>Modèle Gemini</span>
        <select bind:value={geminiModel} disabled={running}>
          {#each GEMINI_MODELS as m}<option value={m}>{m}</option>{/each}
        </select>
      </label>
    {/if}

    <div class="row">
      <label class="field">
        <span>Langue source</span>
        <select bind:value={srcLang} disabled={running}>
          {#each LANGS as l}<option value={l}>{l}</option>{/each}
        </select>
      </label>
      <label class="field">
        <span>Langue cible</span>
        <select bind:value={tgtLang} disabled={running}>
          {#each LANGS as l}<option value={l}>{l}</option>{/each}
        </select>
      </label>
    </div>

    <label class="field">
      <span>Piste de sous-titres</span>
      <select bind:value={selectedTrackId} disabled={running || !tracks.length}>
        {#if !tracks.length}
          <option value={null}>—</option>
        {:else}
          {#each tracks as t}
            <option value={t.ID} disabled={t.IsImageBased}>{trackLabel(t)}</option>
          {/each}
        {/if}
      </select>
    </label>
  </section>

  <div class="actions">
    {#if running}
      <button class="btn danger" on:click={cancel}>Annuler</button>
    {:else}
      <button class="btn ghost" on:click={() => start(true)} disabled={!canStart}>Test · 20 s</button>
      <button class="btn primary" on:click={() => start(false)} disabled={!canStart}>Démarrer la traduction</button>
    {/if}
  </div>

  {#if running || pct > 0 || download}
    <section class="card progress-card">
      {#if download}
        <div class="pline">{download.stage} · {fmtMB(download.done)}{download.total > 0 ? " / " + fmtMB(download.total) : ""} Mo</div>
        <div class="bar"><div class="fill" style="width:{download.total > 0 ? Math.round((download.done / download.total) * 100) : 25}%"></div></div>
      {:else}
        <div class="pline">{progress.total > 0 ? `${progress.done} / ${progress.total} lignes` : "Préparation…"} · {pct}%</div>
        <div class="bar"><div class="fill" style="width:{pct}%"></div></div>
      {/if}
    </section>
  {/if}

  {#if result}<div class="banner ok">✅ {result}</div>{/if}
  {#if errorMsg}<div class="banner err">⚠️ {errorMsg}</div>{/if}

  {#if logs.length}
    <section class="card log" bind:this={logEl}>
      {#each logs as l}<div class="logline">{l}</div>{/each}
    </section>
  {/if}
</main>

<style>
  main {
    max-width: 820px;
    margin: 0 auto;
    padding: 26px 26px 40px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .topbar { display: flex; align-items: center; gap: 14px; }
  .logo {
    width: 44px; height: 44px; border-radius: 12px;
    display: grid; place-items: center; font-weight: 800; font-size: 15px; color: #fff;
    background: linear-gradient(135deg, #7c6cff, #4d8bff);
    box-shadow: 0 6px 20px rgba(124, 108, 255, 0.35);
  }
  h1 { margin: 0; font-size: 20px; font-weight: 700; letter-spacing: -0.02em; }
  .tag { margin: 2px 0 0; color: #8a8a93; font-size: 12.5px; }

  .dropzone {
    width: 100%; box-sizing: border-box;
    border: 1.5px dashed #34343c; background: #141418; color: inherit;
    border-radius: 16px; padding: 28px; cursor: pointer; font: inherit;
    display: flex; flex-direction: column; align-items: center; gap: 6px;
    transition: 0.18s ease;
  }
  .dropzone:hover { border-color: #4a4a55; background: #16161b; }
  .dropzone.drag { border-color: #7c6cff; background: #191726; }
  .dropzone.has { border-style: solid; border-color: #2c2c34; }
  .film { font-size: 30px; }
  .fname { font-weight: 600; font-size: 15px; word-break: break-all; text-align: center; }
  .hint { color: #8a8a93; font-size: 12.5px; }

  .card {
    background: #141418; border: 1px solid #24242b; border-radius: 16px;
    padding: 18px; display: flex; flex-direction: column; gap: 14px;
  }

  .seg {
    display: flex; gap: 6px; background: #0f0f12;
    padding: 5px; border-radius: 12px; border: 1px solid #24242b;
  }
  .seg button {
    flex: 1; padding: 9px; border: none; background: transparent; color: #9a9aa3;
    border-radius: 9px; font: inherit; font-weight: 600; cursor: pointer; transition: 0.15s;
  }
  .seg button.active {
    background: linear-gradient(135deg, #7c6cff, #5b7cff); color: #fff;
    box-shadow: 0 4px 14px rgba(124, 108, 255, 0.3);
  }
  .seg button:disabled { cursor: default; opacity: 0.6; }

  .field { display: flex; flex-direction: column; gap: 6px; flex: 1; }
  .field > span {
    font-size: 11.5px; color: #9a9aa3; font-weight: 600;
    text-transform: uppercase; letter-spacing: 0.04em;
  }
  .row { display: flex; gap: 14px; }
  .empty { color: #8a8a93; font-size: 13px; line-height: 1.5; }
  code { background: #222229; padding: 1px 6px; border-radius: 6px; font-size: 12px; }

  select, input {
    width: 100%; box-sizing: border-box; background: #0f0f12; color: #e7e7ea;
    border: 1px solid #2a2a32; border-radius: 10px; padding: 10px 12px; font: inherit;
    outline: none; transition: 0.15s;
  }
  select:focus, input:focus { border-color: #7c6cff; box-shadow: 0 0 0 3px rgba(124, 108, 255, 0.15); }

  .actions { display: flex; gap: 12px; }
  .btn {
    flex: 1; padding: 13px; border: none; border-radius: 12px;
    font: inherit; font-weight: 700; cursor: pointer; transition: 0.15s; color: #fff;
  }
  .btn:disabled { opacity: 0.4; cursor: default; }
  .btn.primary {
    flex: 2; background: linear-gradient(135deg, #7c6cff, #5b7cff);
    box-shadow: 0 6px 18px rgba(124, 108, 255, 0.3);
  }
  .btn.primary:not(:disabled):hover { filter: brightness(1.08); }
  .btn.ghost { background: #1c1c22; color: #cfcfd6; border: 1px solid #2c2c34; }
  .btn.danger { background: #2a1620; color: #ff8a9b; border: 1px solid #5a2535; }

  .progress-card { gap: 10px; }
  .pline { font-size: 13px; color: #b9b9c2; }
  .bar {
    height: 10px; background: #0f0f12; border-radius: 99px;
    overflow: hidden; border: 1px solid #24242b;
  }
  .fill {
    height: 100%; background: linear-gradient(90deg, #7c6cff, #4d8bff);
    border-radius: 99px; transition: width 0.3s ease;
  }

  .banner { border-radius: 12px; padding: 12px 14px; font-size: 13.5px; font-weight: 600; }
  .banner.ok { background: #0f2018; border: 1px solid #1d4633; color: #7ee2a8; }
  .banner.err { background: #241318; border: 1px solid #5a2535; color: #ff9aa8; }

  .log {
    max-height: 200px; overflow: auto; gap: 2px; background: #0b0b0d;
    font-family: "Consolas", "Courier New", monospace; font-size: 12px;
    color: #9fe0b0; padding: 14px;
  }
  .logline { white-space: pre-wrap; line-height: 1.5; }
</style>
