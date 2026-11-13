<script lang="ts">
  import {
    AlertTriangle,
    AlignCenter,
    AlignLeft,
    AlignRight,
    CheckCircle2,
    Copy,
    FolderOpen,
    History,
    Image as ImageIcon,
    Lock,
    RotateCcw,
    Save,
    Sparkles,
    Trash2
  } from '@lucide/svelte';
  import type { LayerAlignment } from '../lib/editorHelpers';
  import { formatDocStatusLabel } from '../lib/editorHelpers';
  import type {
    DatedOffer,
    DocumentModel,
    ExportBundleRecord,
    ExportRecord,
    Layer,
    OfferCheckResult,
    Page,
    RevisionSummary
  } from '../lib/types';

  type RightTab = 'estilo' | 'legenda' | 'versoes';

  interface Props {
    doc: DocumentModel;
    page: Page;
    selectedLayer: Layer | null;
    offers: DatedOffer[];
    offerCheck: OfferCheckResult | null;
    revisions: RevisionSummary[];
    exportsList: ExportRecord[];
    bundlesList: ExportBundleRecord[];
    busy: boolean;
    onUpdateLayer: (patch: Partial<Layer>, commitHistory?: boolean) => void;
    onAlignLayer: (alignment: LayerAlignment) => void;
    onDeleteSelectedLayer: () => void;
    onOpenReplaceImage: () => void;
    onUpdatePageMeta: (patch: Partial<Page>) => void;
    onUpdateDocMeta: (patch: Partial<DocumentModel>, recheckOffer?: boolean) => void;
    onSaveVersion: (note: string) => Promise<void>;
    onApprove: () => Promise<void>;
    onUnlockDraft: () => Promise<void>;
    onOpenCompareVersions: () => void;
    onRestoreRevision: (rev: number) => Promise<void>;
    onOpenExportPath: (path: string) => Promise<void>;
    onAskAgentForLayer: (layer: Layer) => void;
    onNotify: (msg: string) => void;
  }

  let {
    doc,
    page,
    selectedLayer,
    offers,
    offerCheck,
    revisions,
    exportsList,
    bundlesList,
    busy,
    onUpdateLayer,
    onAlignLayer,
    onDeleteSelectedLayer,
    onOpenReplaceImage,
    onUpdatePageMeta,
    onUpdateDocMeta,
    onSaveVersion,
    onApprove,
    onUnlockDraft,
    onOpenCompareVersions,
    onRestoreRevision,
    onOpenExportPath,
    onAskAgentForLayer,
    onNotify
  }: Props = $props();

  let activeTab = $state<RightTab>('estilo');
  let versionNote = $state('');
  let copiedCaption = $state(false);

  const isLockedApproved = $derived(doc.status === 'approved');

  async function copyCaptionText() {
    const text = doc.caption ?? '';
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      copiedCaption = true;
      onNotify('Legenda copiada preservando quebras de linha.');
      setTimeout(() => (copiedCaption = false), 1800);
    } catch {
      onNotify('Não foi possível copiar automaticamente.');
    }
  }
</script>

<aside class="panel right">
  <div class="right-tabs">
    <button type="button" class:active={activeTab === 'estilo'} onclick={() => (activeTab = 'estilo')}>
      Estilo
    </button>
    <button type="button" class:active={activeTab === 'legenda'} onclick={() => (activeTab = 'legenda')}>
      Legenda{#if offerCheck?.status === 'conflict'} !{/if}
    </button>
    <button type="button" class:active={activeTab === 'versoes'} onclick={() => (activeTab = 'versoes')}>
      Versões ({revisions.length})
    </button>
  </div>

  {#if isLockedApproved}
    <div class="approved-lock-card">
      <div>
        <strong><Lock size={13} style="vertical-align:-2px;" /> Versão v{doc.revision} Aprovada</strong>
        <p class="muted">Protegida contra alterações acidentais. Para editar, abra um novo rascunho.</p>
      </div>
      <button type="button" class="primary" onclick={onUnlockDraft} disabled={busy}>
        Editar como nova versão
      </button>
    </div>
  {/if}

  {#if activeTab === 'estilo'}
    <div class="panel-section">
      <h3>Página {page.index + 1}</h3>
      <label class="field">
        <span>Nome da página</span>
        <input
          type="text"
          value={page.title}
          disabled={isLockedApproved}
          onchange={(e) => onUpdatePageMeta({ title: (e.currentTarget as HTMLInputElement).value })}
        />
      </label>
      <label class="field">
        <span>Cor de fundo</span>
        <div class="row-gap">
          <input
            type="color"
            value={page.background || '#F6F7EE'}
            disabled={isLockedApproved}
            onchange={(e) => onUpdatePageMeta({ background: (e.currentTarget as HTMLInputElement).value })}
          />
          <input
            type="text"
            value={page.background || '#F6F7EE'}
            disabled={isLockedApproved}
            onchange={(e) => onUpdatePageMeta({ background: (e.currentTarget as HTMLInputElement).value })}
          />
        </div>
      </label>
    </div>

    <div class="panel-section">
      <div class="row-between">
        <h3>Elemento selecionado</h3>
        {#if selectedLayer && !isLockedApproved}
          <button type="button" class="icon-btn" onclick={onDeleteSelectedLayer} title="Remover camada">
            <Trash2 size={14} />
          </button>
        {/if}
      </div>

      {#if selectedLayer}
        <label class="field">
          <span>Nome</span>
          <input
            type="text"
            value={selectedLayer.name}
            disabled={isLockedApproved}
            onchange={(e) => onUpdateLayer({ name: (e.currentTarget as HTMLInputElement).value }, true)}
          />
        </label>

        <div class="field">
          <span>Alinhar na página</span>
          <div class="align-bar">
            <button type="button" disabled={isLockedApproved || selectedLayer.locked} onclick={() => onAlignLayer('left')} title="Esquerda">Esq</button>
            <button type="button" disabled={isLockedApproved || selectedLayer.locked} onclick={() => onAlignLayer('center')} title="Centro horizontal">Centro</button>
            <button type="button" disabled={isLockedApproved || selectedLayer.locked} onclick={() => onAlignLayer('right')} title="Direita">Dir</button>
            <button type="button" disabled={isLockedApproved || selectedLayer.locked} onclick={() => onAlignLayer('top')} title="Topo">Topo</button>
            <button type="button" disabled={isLockedApproved || selectedLayer.locked} onclick={() => onAlignLayer('middle')} title="Meio vertical">Meio</button>
            <button type="button" disabled={isLockedApproved || selectedLayer.locked} onclick={() => onAlignLayer('bottom')} title="Base">Base</button>
          </div>
        </div>

        {#if selectedLayer.kind === 'text'}
          <label class="field">
            <span>Texto</span>
            <textarea
              rows={3}
              value={selectedLayer.text ?? ''}
              disabled={isLockedApproved}
              oninput={(e) => onUpdateLayer({ text: (e.currentTarget as HTMLTextAreaElement).value }, false)}
              onchange={() => onUpdateLayer({}, true)}
            ></textarea>
          </label>
          <div class="row-gap">
            <label class="field" style="flex:1;">
              <span>Fonte</span>
              <select
                value={selectedLayer.fontFamily ?? 'Inter'}
                disabled={isLockedApproved}
                onchange={(e) => onUpdateLayer({ fontFamily: (e.currentTarget as HTMLSelectElement).value }, true)}
              >
                <option value="Sora">Sora</option>
                <option value="Inter">Inter</option>
              </select>
            </label>
            <label class="field" style="width:74px;">
              <span>Tam.</span>
              <input
                type="number"
                min="8"
                max="200"
                value={Math.round(selectedLayer.fontSize ?? 20)}
                disabled={isLockedApproved}
                onchange={(e) => onUpdateLayer({ fontSize: Number((e.currentTarget as HTMLInputElement).value) || 20 }, true)}
              />
            </label>
          </div>
          <div class="row-gap">
            <label class="field" style="flex:1;">
              <span>Peso</span>
              <select
                value={selectedLayer.fontWeight ?? '600'}
                disabled={isLockedApproved}
                onchange={(e) => onUpdateLayer({ fontWeight: (e.currentTarget as HTMLSelectElement).value }, true)}
              >
                <option value="400">Regular (400)</option>
                <option value="500">Médio (500)</option>
                <option value="600">Semibold (600)</option>
                <option value="700">Bold (700)</option>
                <option value="800">ExtraBold (800)</option>
              </select>
            </label>
            <div class="field">
              <span>Alinhamento</span>
              <div class="row-gap">
                <button
                  type="button"
                  class:primary={selectedLayer.textAlign === 'left' || !selectedLayer.textAlign}
                  disabled={isLockedApproved}
                  onclick={() => onUpdateLayer({ textAlign: 'left' }, true)}
                  title="Esquerda"
                >
                  <AlignLeft size={13} />
                </button>
                <button
                  type="button"
                  class:primary={selectedLayer.textAlign === 'center'}
                  disabled={isLockedApproved}
                  onclick={() => onUpdateLayer({ textAlign: 'center' }, true)}
                  title="Centralizado"
                >
                  <AlignCenter size={13} />
                </button>
                <button
                  type="button"
                  class:primary={selectedLayer.textAlign === 'right'}
                  disabled={isLockedApproved}
                  onclick={() => onUpdateLayer({ textAlign: 'right' }, true)}
                  title="Direita"
                >
                  <AlignRight size={13} />
                </button>
              </div>
            </div>
          </div>
        {/if}

        {#if selectedLayer.kind === 'rect'}
          <div class="row-gap">
            <label class="field" style="flex:1;">
              <span>Arredondamento</span>
              <input
                type="number"
                min="0"
                max="120"
                value={Math.round(selectedLayer.radius ?? 0)}
                disabled={isLockedApproved}
                onchange={(e) => onUpdateLayer({ radius: Number((e.currentTarget as HTMLInputElement).value) || 0 }, true)}
              />
            </label>
            <label class="field" style="flex:1;">
              <span>Borda (px)</span>
              <input
                type="number"
                min="0"
                max="20"
                value={Math.round(selectedLayer.strokeWidth ?? 0)}
                disabled={isLockedApproved}
                onchange={(e) => onUpdateLayer({ strokeWidth: Number((e.currentTarget as HTMLInputElement).value) || 0 }, true)}
              />
            </label>
          </div>
        {/if}

        {#if selectedLayer.kind === 'image'}
          <button type="button" onclick={onOpenReplaceImage} disabled={isLockedApproved}>
            <ImageIcon size={14} />
            Trocar foto ou logo
          </button>
          <label class="field">
            <span>Enquadramento</span>
            <select
              value={selectedLayer.objectFit ?? 'cover'}
              disabled={isLockedApproved}
              onchange={(e) => onUpdateLayer({ objectFit: (e.currentTarget as HTMLSelectElement).value }, true)}
            >
              <option value="cover">Preencher moldura (recorte proporcional)</option>
              <option value="contain">Mostrar inteiro (ideal para logos)</option>
            </select>
          </label>
          {#if (selectedLayer.objectFit ?? 'cover') === 'cover'}
            <div class="row-gap">
              <label class="field" style="flex:1;">
                <span>Foco X ({Math.round((selectedLayer.positionX ?? 0.5) * 100)}%)</span>
                <input
                  type="range"
                  min="0"
                  max="1"
                  step="0.05"
                  value={selectedLayer.positionX ?? 0.5}
                  disabled={isLockedApproved}
                  oninput={(e) => onUpdateLayer({ positionX: Number((e.currentTarget as HTMLInputElement).value) }, false)}
                  onchange={() => onUpdateLayer({}, true)}
                />
              </label>
              <label class="field" style="flex:1;">
                <span>Foco Y ({Math.round((selectedLayer.positionY ?? 0.5) * 100)}%)</span>
                <input
                  type="range"
                  min="0"
                  max="1"
                  step="0.05"
                  value={selectedLayer.positionY ?? 0.5}
                  disabled={isLockedApproved}
                  oninput={(e) => onUpdateLayer({ positionY: Number((e.currentTarget as HTMLInputElement).value) }, false)}
                  onchange={() => onUpdateLayer({}, true)}
                />
              </label>
            </div>
          {/if}
        {/if}

        {#if selectedLayer.kind !== 'image'}
          <label class="field">
            <span>Cor</span>
            <div class="row-gap">
              <input
                type="color"
                value={selectedLayer.fill || '#374F1E'}
                disabled={isLockedApproved}
                onchange={(e) => onUpdateLayer({ fill: (e.currentTarget as HTMLInputElement).value }, true)}
              />
              <input
                type="text"
                value={selectedLayer.fill || '#374F1E'}
                disabled={isLockedApproved}
                onchange={(e) => onUpdateLayer({ fill: (e.currentTarget as HTMLInputElement).value }, true)}
              />
            </div>
          </label>
        {/if}

        <label class="field">
          <span>Opacidade ({Math.round((selectedLayer.opacity ?? 1) * 100)}%)</span>
          <input
            type="range"
            min="0.05"
            max="1"
            step="0.05"
            value={selectedLayer.opacity ?? 1}
            disabled={isLockedApproved}
            oninput={(e) => onUpdateLayer({ opacity: Number((e.currentTarget as HTMLInputElement).value) }, false)}
            onchange={() => onUpdateLayer({}, true)}
          />
        </label>

        <div class="meta-grid muted">
          <span>X: {Math.round(selectedLayer.x)}px</span>
          <span>Y: {Math.round(selectedLayer.y)}px</span>
          <span>L: {Math.round(selectedLayer.width)}px</span>
          <span>A: {Math.round(selectedLayer.height)}px</span>
        </div>

        <button
          type="button"
          onclick={() => selectedLayer && onAskAgentForLayer(selectedLayer)}
          title="Marca este elemento e abre o balão do Agente IA para você pedir ajustes"
          style="width:100%;"
        >
          <Sparkles size={13} />
          Pedir ajuste ao Agente IA
        </button>
      {:else}
        <p class="muted">Clique em qualquer elemento do quadro ou na lista de camadas para ajustar propriedades.</p>
      {/if}
    </div>
  {:else if activeTab === 'legenda'}
    <div class="panel-section">
      <h3>Oferta vinculada e conferência</h3>
      <label class="field">
        <span>Oferta comercial de referência</span>
        <select
          value={doc.offerId ?? ''}
          disabled={isLockedApproved}
          onchange={(e) => {
            const id = (e.currentTarget as HTMLSelectElement).value;
            const found = offers.find((o) => o.id === id);
            onUpdateDocMeta(
              {
                offerId: id,
                offerRef: found ? `${found.sourceRef}#${found.id}` : ''
              },
              true
            );
          }}
        >
          <option value="">Sem oferta vinculada</option>
          {#each offers as offer (offer.id)}
            <option value={offer.id}>{offer.title} ({offer.validUntil || offer.checkedAt})</option>
          {/each}
        </select>
      </label>

      {#if offerCheck && offerCheck.hasOffer}
        <div
          class="offer-check-box"
          class:conflict={offerCheck.status === 'conflict'}
          class:warning={offerCheck.status === 'warning'}
          class:ok={offerCheck.status === 'ok'}
        >
          <div class="row-gap">
            {#if offerCheck.status === 'ok'}
              <CheckCircle2 size={14} />
              <strong>Preços conferidos</strong>
            {:else}
              <AlertTriangle size={14} />
              <strong>{offerCheck.status === 'conflict' ? 'Divergência de preço detectada' : 'Atenção às condições'}</strong>
            {/if}
          </div>
          {#each offerCheck.messages as msg}
            <p>{msg}</p>
          {/each}
        </div>
      {/if}
    </div>

    <div class="panel-section">
      <div class="row-between">
        <h3>Legenda e chamada (CTA)</h3>
        <button type="button" onclick={copyCaptionText} disabled={!doc.caption}>
          {#if copiedCaption}<Check size={13} /> Copiada{:else}<Copy size={13} /> Copiar{/if}
        </button>
      </div>
      <label class="field">
        <span>Texto da legenda (quebras de linha preservadas)</span>
        <textarea
          rows={6}
          value={doc.caption ?? ''}
          disabled={isLockedApproved}
          oninput={(e) => onUpdateDocMeta({ caption: (e.currentTarget as HTMLTextAreaElement).value })}
          placeholder="Escreva a legenda da publicação..."
        ></textarea>
      </label>
      <label class="field">
        <span>Texto da chamada (CTA)</span>
        <input
          type="text"
          value={doc.ctaText ?? ''}
          disabled={isLockedApproved}
          oninput={(e) => onUpdateDocMeta({ ctaText: (e.currentTarget as HTMLInputElement).value })}
          placeholder="Ex.: Chamar no WhatsApp"
        />
      </label>
      <label class="field">
        <span>Destino (WhatsApp ou link)</span>
        <input
          type="text"
          value={doc.ctaDestination ?? ''}
          disabled={isLockedApproved}
          oninput={(e) => onUpdateDocMeta({ ctaDestination: (e.currentTarget as HTMLInputElement).value })}
          placeholder="https://wa.me/5538997402599"
        />
      </label>
      <label class="field">
        <span>Lembrete de recurso nativo no Instagram</span>
        <textarea
          rows={2}
          value={doc.nativeInstruction ?? ''}
          disabled={isLockedApproved}
          oninput={(e) => onUpdateDocMeta({ nativeInstruction: (e.currentTarget as HTMLTextAreaElement).value })}
          placeholder="Ex.: Inserir adesivo de link no Story; botão desenhado não cria link sozinho."
        ></textarea>
      </label>
    </div>
  {:else}
    <div class="panel-section">
      <h3>Versões e Aprovação</h3>
      <label class="field">
        <span>Nota opcional da versão</span>
        <input
          type="text"
          bind:value={versionNote}
          disabled={isLockedApproved}
          placeholder="Ex.: ajustado preço Black Friday"
        />
      </label>
      <div class="row-gap">
        <button
          type="button"
          onclick={async () => {
            await onSaveVersion(versionNote);
            versionNote = '';
          }}
          disabled={busy || isLockedApproved}
          style="flex:1;"
        >
          <Save size={13} />
          Salvar versão
        </button>
        <button
          type="button"
          class="primary"
          onclick={onApprove}
          disabled={busy || isLockedApproved}
          style="flex:1;"
        >
          <CheckCircle2 size={13} />
          Aprovar
        </button>
      </div>

      <button type="button" onclick={onOpenCompareVersions} disabled={revisions.length === 0}>
        <History size={13} />
        Comparar versões lado a lado ({revisions.length})
      </button>

      <div class="revision-list">
        {#each revisions.slice(0, 8) as r (r.revision)}
          <div class="revision-row">
            <div>
              <strong>v{r.revision} · {formatDocStatusLabel(r.status)}</strong>
              <span class="muted">{r.note || 'Snapshot salvo'}</span>
            </div>
            {#if !isLockedApproved && r.revision !== doc.revision}
              <button type="button" class="icon-btn" onclick={() => onRestoreRevision(r.revision)} title="Restaurar como novo rascunho">
                <RotateCcw size={13} />
              </button>
            {/if}
          </div>
        {/each}
      </div>
    </div>

    <div class="panel-section">
      <h3>Exportações</h3>
      {#if bundlesList.length === 0 && exportsList.length === 0}
        <p class="muted">Nenhuma exportação verificada ainda.</p>
      {:else}
        {#each bundlesList.slice(0, 4) as b (b.id)}
          <div class="export-item">
            <div class="row-between">
              <strong>Pacote v{b.revision} ({b.pageCount} pág · {b.format.toUpperCase()})</strong>
              <button type="button" class="icon-btn" onclick={() => onOpenExportPath(b.bundleDir)} title="Abrir pasta exportada">
                <FolderOpen size={13} />
              </button>
            </div>
            {#if b.isDraftWarning}
              <span class="warn-tag">Rascunho — não aprovado</span>
            {/if}
            <span class="export-path">{b.bundleDir}</span>
          </div>
        {/each}
        {#each exportsList.slice(0, 4) as item (item.id)}
          <div class="export-item">
            <strong>rev {item.revision} · {item.format.toUpperCase()} {item.width}×{item.height}</strong>
            <span class="export-path">{item.filePath}</span>
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</aside>
