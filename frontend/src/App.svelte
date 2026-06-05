<script>
  import { onMount } from "svelte";
  import { AppInfo } from "../wailsjs/go/main/App.js";

  let title = "AI Subtitle Pro";
  onMount(async () => {
    try {
      title = await AppInfo();
    } catch (e) {
      // pont indisponible (hors Wails) : on garde le titre par défaut
    }
  });
</script>

<main>
  <h1>{title}</h1>
  <p class="subtitle">Fondations prêtes — l'interface arrive au Plan 5.</p>
</main>

<style>
  main {
    height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
  }
  h1 {
    font-size: 2rem;
    font-weight: 700;
    letter-spacing: -0.02em;
  }
  .subtitle {
    color: #8a8a8a;
    font-size: 0.95rem;
  }
</style>
