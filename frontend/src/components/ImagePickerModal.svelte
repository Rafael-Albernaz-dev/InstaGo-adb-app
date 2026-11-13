<script lang="ts">
  import { Image as ImageIcon, Search, Upload, X } from '@lucide/svelte';
  import type { LibraryAsset } from '../lib/types';

  interface Props {
    open: boolean;
    mode: 'add' | 'replace';
    assets: LibraryAsset[];
    importing: boolean;
    onClose: () => void;
    onSelectAsset: (asset: LibraryAsset) => void;
    onUploadFile: (file: File) => Promise<void>;
  }

  let { open, mode, assets, importing, onClose, onSelectAsset, onUploadFile }: Props = $props();

  let query = $state('');
  let category = $state<'all' | 'cena' | 'marca' | 'arte_unica'>('all');

  const imageAssets = $derived.by(() => {
    const q = query.trim().toLowerCase();
    return assets
      .filter((a) => a.mediaKind === 'image')
      .filter((a) => category === 'all' || a.classification === category)
      .filter((a) => {
        if (!q) return true;
        return (
          a.title.toLowerCase().includes(q) ||
          a.fileName.toLowerCase().includes(q) ||
          a.subCategory.toLowerCase().includes(q)
        );
      })
      .slice(0, 60);
  });

  async function handleFileInput(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    await onUploadFile(file);
    input.value = '';
    onClose();
  }
</script>

{#if open}
  <div class="modal-backdrop" role="presentation" onclick={onClose}>
    <div
      class="modal-card modal-wide"
      role="dialog"
      aria-modal="true"
      aria-label="Escolher foto ou logo"
      tabindex="-1"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.key === 'Escape' && onClose()}
    >
      <div class="modal-header">
        <div>
          <h2>{mode === 'replace' ? 'Trocar Imagem ou Logo' : 'Inserir Foto ou Logo da Biblioteca'}</h2>
          <p class="muted">Escolha uma foto/marca já gerenciada ou importe um arquivo do computador.</p>
        </div>
        <button type="button" class="icon-btn" onclick={onClose} title="Fechar">
          <X size={16} />
        </button>
      </div>

      <div class="picker-toolbar">
        <div class="search-wrap" style="flex:1;">
          <Search size={14} />
          <input type="text" bind:value={query} placeholder="Buscar foto, logo ou cena..." />
        </div>
        <select bind:value={category}>
          <option value="all">Todas as imagens</option>
          <option value="cena">Fotos e cenas</option>
          <option value="marca">Logos da marca</option>
          <option value="arte_unica">Artes prontas</option>
        </select>

        <label class="upload-btn">
          <Upload size={14} />
          <span>{importing ? 'Importando...' : 'Importar do computador'}</span>
          <input type="file" accept="image/png,image/jpeg" onchange={handleFileInput} disabled={importing} hidden />
        </label>
      </div>

      <div class="picker-grid">
        {#each imageAssets as asset (asset.id)}
          <button
            type="button"
            class="picker-tile"
            onclick={() => {
              onSelectAsset(asset);
              onClose();
            }}
          >
            {#if asset.thumbUrl || asset.assetUrl}
              <img src={asset.thumbUrl || asset.assetUrl} alt={asset.title} loading="lazy" />
            {:else}
              <div class="picker-placeholder"><ImageIcon size={18} /></div>
            {/if}
            <div class="picker-caption">
              <strong>{asset.title}</strong>
              <span class="muted">{asset.classification} · {asset.width}×{asset.height}</span>
            </div>
          </button>
        {/each}
      </div>
    </div>
  </div>
{/if}
