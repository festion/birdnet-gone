<script lang="ts">
  import { X } from '@lucide/svelte';
  import { t } from '$lib/i18n';

  interface Props {
    onClose: () => void;
  }

  let { onClose }: Props = $props();

  let actionPending = $state(false);

  async function systemAction(action: string): Promise<void> {
    actionPending = true;
    try {
      await fetch(`/api/v2/control/system/${action}`, { method: 'POST' });
    } catch {
      // Best effort - system may be rebooting
    }
  }
</script>

<div class="kiosk-settings-overlay">
  <div class="kiosk-settings-dialog" role="dialog" aria-label={t('kiosk.settings')}>
    <div class="kiosk-settings-header">
      <h2>{t('kiosk.settings')}</h2>
      <button onclick={onClose} class="kiosk-settings-close" aria-label="Close">
        <X class="h-5 w-5" />
      </button>
    </div>

    <div class="kiosk-settings-actions">
      <a href="/ui/dashboard" class="kiosk-btn kiosk-btn-primary">
        {t('kiosk.openFullUI')}
      </a>

      <button
        onclick={() => systemAction('reboot')}
        disabled={actionPending}
        class="kiosk-btn kiosk-btn-warning"
      >
        {t('kiosk.reboot')}
      </button>

      <button
        onclick={() => systemAction('poweroff')}
        disabled={actionPending}
        class="kiosk-btn kiosk-btn-danger"
      >
        {t('kiosk.poweroff')}
      </button>
    </div>
  </div>
</div>

<style>
  .kiosk-settings-overlay {
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.8);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 50;
  }

  .kiosk-settings-dialog {
    background-color: #1f2937;
    border-radius: 0.75rem;
    padding: 1.5rem;
    width: 20rem;
    max-width: 90vw;
  }

  .kiosk-settings-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
  }

  .kiosk-settings-header h2 {
    color: white;
    font-size: 1.125rem;
    font-weight: 600;
  }

  .kiosk-settings-close {
    color: #9ca3af;
    background: none;
    border: none;
    cursor: pointer;
    padding: 0.25rem;
  }

  .kiosk-settings-close:hover {
    color: white;
  }

  .kiosk-settings-actions {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .kiosk-btn {
    display: block;
    width: 100%;
    text-align: center;
    padding: 0.75rem;
    color: white;
    border-radius: 0.5rem;
    border: none;
    cursor: pointer;
    font-size: 1rem;
    text-decoration: none;
    transition: background-color 0.15s;
  }

  .kiosk-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .kiosk-btn-primary {
    background-color: #2563eb;
  }

  .kiosk-btn-primary:hover {
    background-color: #1d4ed8;
  }

  .kiosk-btn-warning {
    background-color: #ca8a04;
  }

  .kiosk-btn-warning:hover:not(:disabled) {
    background-color: #a16207;
  }

  .kiosk-btn-danger {
    background-color: #dc2626;
  }

  .kiosk-btn-danger:hover:not(:disabled) {
    background-color: #b91c1c;
  }
</style>
