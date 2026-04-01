<script lang="ts">
  import { onMount } from 'svelte';
  import BirdCard from '../features/kiosk/components/BirdCard.svelte';
  import KioskSettings from '../features/kiosk/components/KioskSettings.svelte';
  import { t } from '$lib/i18n';

  interface Detection {
    commonName: string;
    scientificName: string;
    confidence: number;
    date: string;
    time: string;
  }

  interface CameraImages {
    [species: string]: string;
  }

  let detections = $state<Detection[]>([]);
  let cameraImages = $state<CameraImages>({});
  let thumbnailUrls = $state<Record<string, string>>({});
  let isOffline = $state(false);
  let showSettings = $state(false);
  let settingsTapCount = $state(0);
  let settingsTapTimer = $state<ReturnType<typeof setTimeout> | null>(null);

  const REFRESH_INTERVAL = 5000;
  const MAX_DISPLAY_BIRDS = 4;
  const TRIPLE_TAP_WINDOW = 1000;
  const TRIPLE_TAP_COUNT = 3;

  async function fetchDetections(): Promise<void> {
    try {
      const response = await fetch('/api/v2/detections/recent?limit=50');
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const data: Detection[] = await response.json();

      // Deduplicate by species name, keep most recent
      const seen = new Set<string>();
      const unique: Detection[] = [];
      for (const d of data) {
        if (!seen.has(d.commonName)) {
          seen.add(d.commonName);
          unique.push(d);
        }
        if (unique.length >= MAX_DISPLAY_BIRDS) break;
      }

      detections = unique;
      isOffline = false;

      // Fetch thumbnail URLs for displayed species
      const sciNames = unique.map(d => d.scientificName).filter(Boolean);
      if (sciNames.length > 0) {
        await fetchThumbnails(sciNames);
      }
    } catch {
      isOffline = true;
    }
  }

  async function fetchThumbnails(sciNames: string[]): Promise<void> {
    try {
      const params = sciNames.map(n => `species=${encodeURIComponent(n)}`).join('&');
      const response = await fetch(`/api/v2/analytics/species/thumbnails?${params}`);
      if (response.ok) {
        const data: Record<string, string> = await response.json();
        thumbnailUrls = data;
      }
    } catch {
      // Thumbnails are best-effort
    }
  }

  async function fetchCameraImages(): Promise<void> {
    try {
      const response = await fetch('/api/v2/vicohome/images');
      if (response.ok) {
        cameraImages = await response.json();
      }
    } catch {
      // Camera images are optional, don't set offline
    }
  }

  function handleSettingsTap(): void {
    settingsTapCount++;
    if (settingsTapTimer) clearTimeout(settingsTapTimer);
    if (settingsTapCount >= TRIPLE_TAP_COUNT) {
      showSettings = true;
      settingsTapCount = 0;
      return;
    }
    settingsTapTimer = setTimeout(() => {
      settingsTapCount = 0;
    }, TRIPLE_TAP_WINDOW);
  }

  function handleSettingsKeydown(e: KeyboardEvent): void {
    if (e.key === 'Enter') handleSettingsTap();
  }

  function formatTimeAgo(date: string, time: string): string {
    if (!date || !time) return '';
    const detectionDate = new Date(`${date}T${time}`);
    const diffMs = Date.now() - detectionDate.getTime();
    const diffSec = Math.floor(diffMs / 1000);
    if (diffSec < 60) return `${diffSec}s ago`;
    const diffMin = Math.floor(diffSec / 60);
    if (diffMin < 60) return `${diffMin}m ago`;
    const diffHr = Math.floor(diffMin / 60);
    if (diffHr < 24) return `${diffHr}h ago`;
    return `${Math.floor(diffHr / 24)}d ago`;
  }

  onMount(() => {
    fetchDetections();
    fetchCameraImages();

    const detectionInterval = setInterval(fetchDetections, REFRESH_INTERVAL);
    const cameraInterval = setInterval(fetchCameraImages, REFRESH_INTERVAL);

    return () => {
      clearInterval(detectionInterval);
      clearInterval(cameraInterval);
    };
  });
</script>

<div
  class="kiosk-container"
  role="button"
  tabindex="0"
  onclick={handleSettingsTap}
  onkeydown={handleSettingsKeydown}
>
  {#if isOffline}
    <div class="kiosk-status">
      {t('kiosk.offline')}
    </div>
  {:else if detections.length === 0}
    <div class="kiosk-status">
      {t('kiosk.noDetections')}
    </div>
  {:else}
    <div class="kiosk-grid">
      {#each detections as detection (detection.commonName)}
        <BirdCard
          species={detection.commonName}
          scientificName={detection.scientificName}
          confidence={Math.round(detection.confidence * 100)}
          timeAgo={formatTimeAgo(detection.date, detection.time)}
          cameraImageUrl={cameraImages[detection.commonName.toLowerCase()] ?? ''}
          thumbnailUrl={thumbnailUrls[detection.scientificName] ?? ''}
        />
      {/each}
    </div>
  {/if}

  {#if showSettings}
    <KioskSettings onClose={() => (showSettings = false)} />
  {/if}
</div>

<style>
  .kiosk-container {
    position: fixed;
    inset: 0;
    background-color: #111827;
    display: flex;
    flex-direction: column;
    cursor: default;
  }

  .kiosk-status {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #9ca3af;
    font-size: 1.5rem;
  }

  .kiosk-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    grid-template-rows: repeat(2, 1fr);
    gap: 0.5rem;
    padding: 0.5rem;
    height: 100%;
  }

  :global(body) {
    overflow: hidden;
  }
</style>
