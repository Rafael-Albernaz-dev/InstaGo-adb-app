<script lang="ts">
  import { FileText, FolderOpen, Layers, Plus, Smartphone, X } from '@lucide/svelte';
  import { formatDocStatusLabel } from '../lib/editorHelpers';
  import type { DatedOffer, DocumentModel } from '../lib/types';

  interface Props {
    open: boolean;
    documents: DocumentModel[];
    offers: DatedOffer[];
    currentDocId: string;
    onClose: () => void;
    onSelectDocument: (id: string) => void;
    onCreateFromTemplate: (templateKind: string, offerId: string, title: string) => void;
  }

  let {
    open,
    documents,
    offers,
    currentDocId,
    onClose,
    onSelectDocument,
    onCreateFromTemplate
  }: Props = $props();

  let selectedKind = $state<'feed' | 'story' | 'carousel' | 'blank_feed' | 'blank_story'>('story');
  let selectedOfferId = $state('');
  let customTitle = $state('');

  const pieces = $derived(documents.filter((d) => d.kind !== 'template'));
  const starters = [
    {
      id: 'feed',
      title: 'Feed Único (1080×1350)',
      desc: 'Modelo de plano/combo com foto, preço destacado e condições.',
      icon: FileText
    },
    {
      id: 'story',
      title: 'Story (1080×1920)',
      desc: 'Formato vertical em pé com espaço reservado para adesivo de link.',
      icon: Smartphone
    },
    {
      id: 'carousel',
      title: 'Carrossel (2 páginas base)',
      desc: 'Capa + página interna pronta para duplicar quantas vezes precisar.',
      icon: Layers
    },
    {
      id: 'blank_feed',
      title: 'Feed em Branco (1080×1350)',
      desc: 'Tela limpa na proporção de postagem do Feed.',
      icon: Plus
    },
    {
      id: 'blank_story',
      title: 'Story em Branco (1080×1920)',
      desc: 'Tela limpa vertical na proporção de Story.',
      icon: Plus
    }
  ] as const;

  function submitCreate() {
    onCreateFromTemplate(selectedKind, selectedOfferId, customTitle.trim());
    customTitle = '';
    onClose();
  }
</script>

{#if open}
  <div class="modal-backdrop" role="presentation" onclick={onClose}>
    <div
      class="modal-card"
      role="dialog"
      aria-modal="true"
      aria-label="Peças e novos modelos"
      tabindex="-1"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.key === 'Escape' && onClose()}
    >
      <div class="modal-header">
        <div>
          <h2>Peças do Estúdio e Criação</h2>
          <p class="muted">Abra uma peça salva ou comece a partir de um modelo base.</p>
        </div>
        <button type="button" class="icon-btn" onclick={onClose} title="Fechar">
          <X size={16} />
        </button>
      </div>

      <div class="modal-two-col">
        <div class="modal-col">
          <h3>Criar nova peça</h3>
          <div class="starter-grid">
            {#each starters as st}
              {@const Icon = st.icon}
              <button
                type="button"
                class="starter-card"
                class:selected={selectedKind === st.id}
                onclick={() => (selectedKind = st.id)}
              >
                <div class="starter-head">
                  <Icon size={15} />
                  <strong>{st.title}</strong>
                </div>
                <span>{st.desc}</span>
              </button>
            {/each}
          </div>

          <label class="field">
            <span>Título da nova peça (opcional)</span>
            <input type="text" bind:value={customTitle} placeholder="Ex.: Carrossel Black Friday Fibra" />
          </label>

          <label class="field">
            <span>Vincular a uma oferta importada (opcional)</span>
            <select bind:value={selectedOfferId}>
              <option value="">Sem oferta vinculada</option>
              {#each offers as offer (offer.id)}
                <option value={offer.id}>{offer.title} ({offer.validUntil || offer.checkedAt})</option>
              {/each}
            </select>
          </label>

          <button type="button" class="primary" onclick={submitCreate}>
            <Plus size={14} />
            Criar peça editável
          </button>
        </div>

        <div class="modal-col">
          <h3>Peças existentes ({pieces.length})</h3>
          <div class="doc-list">
            {#each pieces as item (item.id)}
              <div class="doc-row" class:selected={item.id === currentDocId}>
                <div class="doc-row-meta">
                  <strong>{item.title}</strong>
                  <span class="muted">
                    {item.format === 'story' ? 'Story 1080×1920' : 'Feed 1080×1350'} · {item.pages.length} pág. · v{item.revision} · {formatDocStatusLabel(item.status)}
                  </span>
                </div>
                <button
                  type="button"
                  onclick={() => {
                    onSelectDocument(item.id);
                    onClose();
                  }}
                >
                  <FolderOpen size={13} />
                  {item.id === currentDocId ? 'Aberta' : 'Abrir'}
                </button>
              </div>
            {/each}
          </div>
          <p class="muted" style="font-size:11px; margin-top:6px;">
            Dica: você também pode abrir qualquer arte antiga diretamente da aba <strong>Biblioteca</strong> clicando em “Abrir no Editor”.
          </p>
        </div>
      </div>
    </div>
  </div>
{/if}
