<script lang="ts">
  import { ChevronLeft, ChevronRight, CopyPlus, FilePlus2, Trash2 } from '@lucide/svelte';
  import type { Page } from '../lib/types';

  interface Props {
    pages: Page[];
    activePageIndex: number;
    disabled?: boolean;
    onSelectPage: (index: number) => void;
    onAddPage: () => void;
    onDuplicatePage: () => void;
    onMovePage: (delta: -1 | 1) => void;
    onDeletePage: () => void;
  }

  let {
    pages,
    activePageIndex,
    disabled = false,
    onSelectPage,
    onAddPage,
    onDuplicatePage,
    onMovePage,
    onDeletePage
  }: Props = $props();
</script>

<div class="page-strip">
  <div class="page-strip-list">
    {#each pages as p, idx (p.id)}
      <button
        type="button"
        class="page-chip"
        class:active={idx === activePageIndex}
        onclick={() => onSelectPage(idx)}
        title={p.title}
      >
        <span class="page-swatch" style:background={p.background || '#F6F7EE'}></span>
        <span class="page-chip-label">
          <strong>{String(idx + 1).padStart(2, '0')}</strong>
          <span>{p.title || `Página ${idx + 1}`}</span>
        </span>
      </button>
    {/each}
  </div>

  <div class="page-strip-actions">
    <button type="button" onclick={onAddPage} {disabled} title="Adicionar página em branco">
      <FilePlus2 size={14} />
      + Página
    </button>
    <button type="button" onclick={onDuplicatePage} {disabled} title="Duplicar página atual mantendo padrão">
      <CopyPlus size={14} />
      Duplicar
    </button>
    <button
      type="button"
      onclick={() => onMovePage(-1)}
      disabled={disabled || activePageIndex <= 0}
      title="Mover página para a esquerda"
    >
      <ChevronLeft size={14} />
    </button>
    <button
      type="button"
      onclick={() => onMovePage(1)}
      disabled={disabled || activePageIndex >= pages.length - 1}
      title="Mover página para a direita"
    >
      <ChevronRight size={14} />
    </button>
    <button
      type="button"
      onclick={onDeletePage}
      disabled={disabled || pages.length <= 1}
      title="Excluir página atual"
    >
      <Trash2 size={14} />
    </button>
  </div>
</div>
