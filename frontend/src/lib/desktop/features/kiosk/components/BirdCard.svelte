<script lang="ts">
  interface Props {
    species: string;
    scientificName: string;
    confidence: number;
    timeAgo: string;
    cameraImageUrl?: string;
  }

  let { species, scientificName, confidence, timeAgo, cameraImageUrl = '' }: Props = $props();

  let imageError = $state(false);

  // Image priority: camera still > API thumbnail
  let imageUrl = $derived(
    cameraImageUrl && !imageError
      ? cameraImageUrl
      : `/api/v2/analytics/species/thumbnails?species=${encodeURIComponent(scientificName)}`
  );

  function handleImageError(): void {
    imageError = true;
  }
</script>

<div class="bird-card">
  <div class="bird-card-image">
    <img src={imageUrl} alt={species} onerror={handleImageError} />
    <div class="bird-card-confidence">
      {confidence}%
    </div>
  </div>
  <div class="bird-card-info">
    <div class="bird-card-name">{species}</div>
    <div class="bird-card-meta">
      <span class="bird-card-scientific">{scientificName}</span>
      <span class="bird-card-time">{timeAgo}</span>
    </div>
  </div>
</div>

<style>
  .bird-card {
    position: relative;
    background-color: #1f2937;
    border-radius: 0.5rem;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .bird-card-image {
    flex: 1;
    position: relative;
    min-height: 0;
  }

  .bird-card-image img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .bird-card-confidence {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    background-color: rgba(0, 0, 0, 0.6);
    color: white;
    font-size: 0.875rem;
    padding: 0.25rem 0.5rem;
    border-radius: 0.25rem;
  }

  .bird-card-info {
    padding: 0.5rem;
    background-color: rgba(31, 41, 55, 0.9);
  }

  .bird-card-name {
    color: white;
    font-weight: 600;
    font-size: 0.875rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .bird-card-meta {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .bird-card-scientific {
    color: #9ca3af;
    font-size: 0.75rem;
    font-style: italic;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .bird-card-time {
    color: #9ca3af;
    font-size: 0.75rem;
    margin-left: 0.5rem;
    flex-shrink: 0;
  }
</style>
