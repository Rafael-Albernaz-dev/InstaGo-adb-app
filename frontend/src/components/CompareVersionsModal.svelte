<script lang="ts">
  import { History, RotateCcw, X } from '@lucide/svelte';
  import { bridge } from '../lib/bridge';
  import { renderPageOffscreenDataUrl } from '../lib/canvasMapper';
  import { formatDocStatusLabel, summarizeVersionChanges } from '../lib/editorHelpers';
  import type { DocumentModel, RevisionSummary } from '../lib/types';

  interface Props {
    open: boolean;
    documentId: string;
    revisions: RevisionSummary[];
    activePageIndex: number;
    canRestore: boolean;
    onClose: () => void;
    onRestoreRevision: (revision: number) => Promise<void>;
  }

  let { open, documentId, revisions, activePageIndex, canRestore, onClose, onRestoreRevision }: Props = $props();

  let revA = $state(0);
  let revB = $state(0);
  let docA = $state<DocumentModel | null>(null);
  let docB = $state<DocumentModel | null>(null);
  let imgA = $state('');
  let imgB = $state('');
  let loading = $state(false);
  let errorMsg = $state('');

  $effect(() => {
    if (!open || revisions.length === 0) return;
    if (!revB || !revisions.some((r) => r.revision === revB)) {
      revB = revisions[0].revision;
    }
    const second = revisions.length > 1 ? revisions[1].revision : revisions[0].revision;
    if (!revA || !revisions.some((r) => r.revision === revA)) {
      revA = second;
    }
  });

  $effect(() => {
    if (!open || !documentId || revA <= 0 || revB <= 0) return;
    let cancelled = false;
    loading = true;
    errorMsg = '';

    Promise.all([bridge.getRevision(documentId, revA), bridge.getRevision(documentId, revB)])
      .then(async ([recA, recB]) => {
        if (cancelled) return;
        docA = recA.document;
        docB = recB.document;
        const pageA = recA.document.pages[Math.min(activePageIndex, recA.document.pages.length - 1)];
        const pageB = recB.document.pages[Math.min(activePageIndex, recB.document.pages.length - 1)];
        const [urlA, urlB] = await Promise.all([
          renderPageOffscreenDataUrl(pageA, recA.document.workspaceWidth, recA.document.workspaceHeight, 'png', 0.65),
          renderPageOffscreenDataUrl(pageB, recB.document.workspaceWidth, recB.document.workspaceHeight, 'png', 0.65)
        ]);
        if (!cancelled) {
          imgA = urlA;
          imgB = urlB;
        }
      })
      .catch((err) => {
        if (!cancelled) {
          errorMsg = err instanceof Error ? err.message : String(err);
        }
      })
      .finally(() => {
        if (!cancelled) loading = false;
      });

    return () => {
      cancelled = true;
    };
  });

  const diffs = $derived.by(() => {
    if (!docA || !docB) return [];
    return summarizeVersionChanges(docA, docB);
  });
</script>

{#if open}
  <div class="modal-backdrop" role="presentation" onclick={onClose}>
    <div
      class="modal-card modal-wide"
      role="dialog"
      aria-modal="true"
      aria-label="Comparar versões lado a lado"
      tabindex="-1"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.key === 'Escape' && onClose()}
    >
      <div class="modal-header">
        <div>
          <h2><History size={16} style="vertical-align:-3px;" /> Comparar versões salvas</h2>
          <p class="muted">Compare visualmente qualquer duas versões salvas antes de aprovar ou restaurar.</p>
        </div>
        <button type="button" class="icon-btn" onclick={onClose} title="Fechar">
          <X size={16} />
        </button>
      </div>

      {#if errorMsg}
        <p class="error-banner">{errorMsg}</p>
      {/if}

      <div class="compare-grid">
        <div class="compare-pane">
          <div class="compare-pane-head">
            <select bind:value={revA}>
              {#each revisions as r (r.revision)}
                <option value={r.revision}>
                  v{r.revision} ({formatDocStatusLabel(r.status)}) — {r.note || 'sem nota'}
                </option>
              {/each}
            </select>
            {#if canRestore && revA > 0}
              <button
                type="button"
                onclick={async () => {
                  await onRestoreRevision(revA);
                  onClose();
                }}
              >
                <RotateCcw size={13} />
                Restaurar v{revA}
              </button>
            {/if}
          </div>
          <div class="compare-img-box">
            {#if loading && !imgA}
              <span class="muted">Renderizando prévia...</span>
            {:else if imgA}
              <img src={imgA} alt={`Versão ${revA}`} />
            {/if}
          </div>
        </div>

        <div class="compare-pane">
          <div class="compare-pane-head">
            <select bind:value={revB}>
              {#each revisions as r (r.revision)}
                <option value={r.revision}>
                  v{r.revision} ({formatDocStatusLabel(r.status)}) — {r.note || 'sem nota'}
                </option>
              {/each}
            </select>
            {#if canRestore && revB > 0}
              <button
                type="button"
                onclick={async () => {
                  await onRestoreRevision(revB);
                  onClose();
                }}
              >
                <RotateCcw size={13} />
                Restaurar v{revB}
              </button>
            {/if}
          </div>
          <div class="compare-img-box">
            {#if loading && !imgB}
              <span class="muted">Renderizando prévia...</span>
            {:else if imgB}
              <img src={imgB} alt={`Versão ${revB}`} />
            {/if}
          </div>
        </div>
      </div>

      <div class="compare-diff-box">
        <strong>Resumo de diferenças (v{revA} → v{revB}):</strong>
        <ul>
          {#each diffs as line}
            <li>{line}</li>
          {/each}
        </ul>
      </div>
    </div>
  </div>
{/if}
