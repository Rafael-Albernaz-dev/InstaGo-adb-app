<script lang="ts">
  import {
    Archive,
    Check,
    Copy,
    ExternalLink,
    FileText,
    FolderOpen,
    History,
    Image as ImageIcon,
    Pencil,
    RefreshCw,
    Search,
    Tag,
    Type
  } from '@lucide/svelte';
  import {
    filterAssets,
    formatBRL,
    formatBytes,
    getClassificationBadge,
    groupOffersByCategory
  } from '../lib/libraryHelpers';
  import type { LibraryAsset, LibraryOverview } from '../lib/types';

  interface Props {
    overview: LibraryOverview | null;
    loading: boolean;
    onRefresh: () => Promise<void>;
    onOpenFolder: (scope: 'library' | 'asset_dir' | 'creative_base', assetId?: string) => Promise<void>;
    onOpenInEditor: (documentId: string) => Promise<void>;
    onOpenAssetInEditor: (assetId: string, format: 'feed' | 'story') => Promise<void>;
    onNotify: (msg: string) => void;
  }

  let { overview, loading, onRefresh, onOpenFolder, onOpenInEditor, onOpenAssetInEditor, onNotify }: Props = $props();

  type SubTab = 'assets' | 'offers' | 'history';
  let subTab = $state<SubTab>('assets');
  let query = $state('');
  let classification = $state('all');
  let format = $state('all');
  let selectedAssetId = $state<string | null>(null);
  let copiedId = $state<string | null>(null);

  const filteredAssets = $derived.by(() => {
    if (!overview) return [];
    return filterAssets(overview.assets, { query, classification, format });
  });

  const selectedAsset = $derived.by<LibraryAsset | null>(() => {
    if (!overview || overview.assets.length === 0) return null;
    if (selectedAssetId) {
      const found = overview.assets.find((a) => a.id === selectedAssetId);
      if (found) return found;
    }
    return filteredAssets[0] ?? overview.assets[0] ?? null;
  });

  const offerGroups = $derived.by(() => {
    if (!overview) return [];
    return groupOffersByCategory(overview.offers);
  });

  async function copyText(id: string, text: string) {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      copiedId = id;
      onNotify('Texto copiado para a área de transferência.');
      setTimeout(() => {
        if (copiedId === id) copiedId = null;
      }, 1800);
    } catch {
      onNotify('Não foi possível copiar automaticamente.');
    }
  }
</script>

<section class="library-shell">
  <div class="library-header">
    <div class="library-tabs">
      <button class:primary={subTab === 'assets'} onclick={() => (subTab = 'assets')}>
        <ImageIcon size={14} />
        Acervo ({overview?.assets.length ?? 0})
      </button>
      <button class:primary={subTab === 'offers'} onclick={() => (subTab = 'offers')}>
        <Tag size={14} />
        Ofertas datadas ({overview?.offers.length ?? 0})
      </button>
      <button class:primary={subTab === 'history'} onclick={() => (subTab = 'history')}>
        <History size={14} />
        Histórico publicado ({overview?.history.length ?? 0})
      </button>
    </div>

    <div class="library-stats">
      {#if overview?.summary}
        <span class="stat-pill"><strong>{overview.summary.uniqueArtsCount}</strong> artes únicas</span>
        <span class="stat-pill"><strong>{overview.summary.auxiliaryCoversCount}</strong> capas extras</span>
        <span class="stat-pill"><strong>{overview.summary.editableModelsCount}</strong> modelo editável</span>
        <span class="stat-pill"><strong>{overview.summary.fontsCount}</strong> fontes locais</span>
      {/if}
      <button onclick={() => onOpenFolder('library')} title="Abrir pasta da biblioteca gerenciada">
        <FolderOpen size={14} />
        Pasta da biblioteca
      </button>
      <button onclick={onRefresh} disabled={loading} title="Atualizar listagem">
        <RefreshCw size={14} />
        Atualizar
      </button>
    </div>
  </div>

  {#if !overview}
    <div class="library-empty">
      <p>{loading ? 'Carregando biblioteca e importação da base...' : 'Nenhuma biblioteca carregada ainda.'}</p>
    </div>
  {:else if subTab === 'assets'}
    <div class="library-filters">
      <div class="search-wrap">
        <Search size={14} />
        <input
          type="search"
          placeholder="Buscar por título, arquivo, legenda ou peça..."
          bind:value={query}
          aria-label="Buscar na biblioteca"
        />
      </div>

      <select bind:value={classification} aria-label="Filtrar por categoria">
        <option value="all">Todas as categorias ({overview.countsByClass['all'] ?? overview.assets.length})</option>
        <option value="arte_unica">Oficial — artes únicas ({overview.countsByClass['arte_unica'] ?? 0})</option>
        <option value="capa_auxiliar">Capa extra — cópias feed ({overview.countsByClass['capa_auxiliar'] ?? 0})</option>
        <option value="cena">Cenas e fotos ({overview.countsByClass['cena'] ?? 0})</option>
        <option value="marca">Marcas e logos ({overview.countsByClass['marca'] ?? 0})</option>
        <option value="capa_editorial">Capas editoriais ({overview.countsByClass['capa_editorial'] ?? 0})</option>
        <option value="fonte">Fontes ({overview.countsByClass['fonte'] ?? 0})</option>
        <option value="historico">Rascunhos antigos ({overview.countsByClass['historico'] ?? 0})</option>
        <option value="referencia">Documentos e regras ({overview.countsByClass['referencia'] ?? 0})</option>
      </select>

      <select bind:value={format} aria-label="Filtrar por formato">
        <option value="all">Todos os formatos</option>
        <option value="feed">Feed (1080×1350)</option>
        <option value="story">Story (1080×1920)</option>
        <option value="cover">Capa destaque</option>
        <option value="other">Outros / Assets</option>
      </select>

      <span class="badge">{filteredAssets.length} exibidos</span>
    </div>

    <div class="library-workspace">
      <div class="asset-grid" role="list">
        {#if filteredAssets.length === 0}
          <div class="library-empty" style="grid-column:1 / -1">
            Nenhum item encontrado para os filtros selecionados.
          </div>
        {:else}
          {#each filteredAssets as asset (asset.id)}
            {@const badge = getClassificationBadge(asset.classification)}
            <button
              type="button"
              class="asset-card"
              class:selected={selectedAsset?.id === asset.id}
              onclick={() => (selectedAssetId = asset.id)}
            >
              <div class="asset-thumb">
                {#if asset.mediaKind === 'image' && asset.thumbUrl}
                  <img src={asset.thumbUrl} alt={asset.title} loading="lazy" />
                {:else if asset.mediaKind === 'font'}
                  <Type size={28} />
                {:else}
                  <FileText size={28} />
                {/if}
              </div>

              <div class="asset-card-body">
                <div class="asset-badges">
                  <span
                    class="class-badge"
                    style="background:{badge.bg};color:{badge.fg};border-color:{badge.border}"
                  >
                    {badge.label}
                  </span>
                  {#if asset.isEditableModel}
                    <span class="class-badge editable-badge">Editável</span>
                  {/if}
                </div>
                <div class="asset-title" title={asset.title}>{asset.title}</div>
                <div class="asset-meta">
                  <span>{asset.fileName}</span>
                  {#if asset.width > 0 && asset.height > 0}
                    <span>{asset.width}×{asset.height}</span>
                  {/if}
                </div>
              </div>
            </button>
          {/each}
        {/if}
      </div>

      <aside class="asset-inspector">
        {#if selectedAsset}
          {@const badge = getClassificationBadge(selectedAsset.classification)}
          <div class="panel-section">
            <div class="inspector-badges">
              <span
                class="class-badge"
                style="background:{badge.bg};color:{badge.fg};border-color:{badge.border}"
              >
                {badge.label}
              </span>
              {#if selectedAsset.isEditableModel}
                <span class="class-badge editable-badge">Modelo editável</span>
              {:else if selectedAsset.mediaKind === 'image'}
                <span class="badge">Referência pronta</span>
              {/if}
            </div>

            <h3 style="margin:4px 0 0">{selectedAsset.title}</h3>

            {#if selectedAsset.mediaKind === 'image' && selectedAsset.assetUrl}
              <div class="inspector-preview">
                <img src={selectedAsset.assetUrl} alt={selectedAsset.title} />
              </div>
            {/if}

            <div class="inspector-actions">
              <button onclick={() => onOpenFolder('asset_dir', selectedAsset.id)}>
                <FolderOpen size={14} />
                Abrir pasta
              </button>
              {#if selectedAsset.isEditableModel && selectedAsset.linkedDocumentId}
                <button class="primary" onclick={() => onOpenInEditor(selectedAsset.linkedDocumentId)}>
                  <Pencil size={14} />
                  Abrir modelo no Editor
                </button>
              {:else if selectedAsset.mediaKind === 'image'}
                <button
                  class="primary"
                  onclick={() =>
                    onOpenAssetInEditor(selectedAsset.id, selectedAsset.format === 'story' ? 'story' : 'feed')}
                >
                  <Pencil size={14} />
                  Abrir imagem no Editor
                </button>
              {/if}
            </div>

            {#if selectedAsset.caption}
              <div class="caption-box">
                <div class="caption-head">
                  <strong>Legenda / Texto associado</strong>
                  <button class="icon-btn" onclick={() => copyText(selectedAsset.id, selectedAsset.caption)}>
                    {#if copiedId === selectedAsset.id}<Check size={12} />{:else}<Copy size={12} />{/if}
                    Copiar
                  </button>
                </div>
                <pre class="caption-text">{selectedAsset.caption}</pre>
              </div>
            {/if}

            <div class="kv-list">
              <div><span>Arquivo</span><strong>{selectedAsset.fileName}</strong></div>
              <div><span>Formato</span><strong>{selectedAsset.format}</strong></div>
              <div><span>Subcategoria</span><strong>{selectedAsset.subCategory || '—'}</strong></div>
              {#if selectedAsset.width > 0}
                <div><span>Dimensões</span><strong>{selectedAsset.width}×{selectedAsset.height}px</strong></div>
              {/if}
              <div><span>Tamanho</span><strong>{formatBytes(selectedAsset.bytes)}</strong></div>
              <div><span>Caminho gerenciado</span><span class="mono">{selectedAsset.relPath}</span></div>
              <div><span>Origem preservada</span><span class="mono">{selectedAsset.sourceRelPath}</span></div>
              <div><span>SHA-256</span><span class="mono">{selectedAsset.sha256}</span></div>
            </div>
          </div>
        {:else}
          <div class="panel-section">
            <p style="margin:0;color:var(--text-muted)">Selecione um item na grade para ver detalhes, legenda e origem.</p>
          </div>
        {/if}
      </aside>
    </div>
  {:else if subTab === 'offers'}
    <div class="readonly-view">
      <div class="notice-banner">
        <Archive size={16} />
        <div>
          <strong>Referência comercial datada (11/11/2026) — somente leitura.</strong>
          Estes valores foram importados da base aprovada para consulta. Edição de campanhas, prazos e novas ofertas entra na etapa de Campanhas.
        </div>
      </div>

      {#each offerGroups as group (group.category)}
        <div class="panel-section">
          <h3>{group.label}</h3>
          <div class="table-wrap">
            <table class="data-table">
              <thead>
                <tr>
                  <th>Plano / Serviço</th>
                  <th>Promocional</th>
                  <th>Depois da promoção</th>
                  <th>Instalação</th>
                  <th>Detalhes e restrições</th>
                  <th>Conferido em</th>
                </tr>
              </thead>
              <tbody>
                {#each group.items as offer (offer.id)}
                  <tr>
                    <td>
                      <strong>{offer.title}</strong>
                      {#if offer.speedSummary}
                        <div class="muted-small">{offer.speedSummary}</div>
                      {/if}
                    </td>
                    <td>
                      <strong>{formatBRL(offer.promoPriceCents)}</strong>
                      <div class="muted-small">{offer.promoPeriod}</div>
                    </td>
                    <td>{formatBRL(offer.afterPriceCents)}/mês</td>
                    <td>{formatBRL(offer.installFeeCents)}</td>
                    <td>
                      {#if offer.servicesSummary}<div>{offer.servicesSummary}</div>{/if}
                      <div class="muted-small">{offer.restrictions}</div>
                    </td>
                    <td>
                      <div>{offer.checkedAt}</div>
                      <div class="muted-small">Somente leitura</div>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="readonly-view">
      <div class="notice-banner">
        <History size={16} />
        <div>
          <strong>Histórico operacional do Instagram (11/11/2026) — somente consulta.</strong>
          Estes registros representam o que já foi publicado anteriormente. A importação nunca cria fila de republicação.
        </div>
      </div>

      <div class="panel-section">
        <h3>Publicações e Destaques registrados ({overview.history.length})</h3>
        <div class="history-list">
          {#each overview.history as item (item.id)}
            <div class="history-card">
              <div class="history-head">
                <div>
                  <span class="badge">{item.kind === 'feed_post' ? 'Feed' : 'Destaque Stories'}</span>
                  {#if item.pinned}
                    <span class="class-badge" style="background:#e6f2df;color:#254a18;border-color:#9cc486">Fixado</span>
                  {/if}
                  <strong>{item.title}</strong>
                </div>
                <span class="muted-small">Posição #{item.position} · {item.pageCount} páginas · {item.observedAt}</span>
              </div>
              <div class="muted-small">Status registrado: <strong>{item.status}</strong></div>
              {#if item.instagramUrl}
                <div class="mono" style="display:flex;align-items:center;gap:6px">
                  <ExternalLink size={12} />
                  {item.instagramUrl}
                </div>
              {/if}
              {#if item.notes}
                <div class="muted-small">{item.notes}</div>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    </div>
  {/if}
</section>
