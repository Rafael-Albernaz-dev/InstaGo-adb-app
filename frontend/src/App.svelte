<script lang="ts">
  import { onDestroy, onMount, tick } from 'svelte';
  import { Canvas, Rect, Textbox } from 'fabric';
  import {
    ArrowDown,
    ArrowUp,
    Columns2,
    Download,
    Eye,
    EyeOff,
    FolderOpen,
    Layers,
    Lock,
    Plus,
    Redo2,
    RotateCcw,
    Save,
    Square,
    Type,
    Undo2,
    Unlock
  } from '@lucide/svelte';
  import { bridge } from './lib/bridge';
  import {
    exportCanvasDataUrl,
    getLayerId,
    renderPageToCanvas,
    syncCanvasToPage,
    tagLayerObject
  } from './lib/canvasMapper';
  import { loadBrandFonts } from './lib/fonts';
  import { HistoryStack } from './lib/history';
  import LibraryView from './components/LibraryView.svelte';
  import SettingsView from './components/SettingsView.svelte';
  import type {
    BackupResult,
    CompanyProfile,
    DocumentModel,
    ExportRecord,
    IntegrityReport,
    Layer,
    LibraryOverview,
    WorkspaceInfo
  } from './lib/types';

  type NavSection = 'editor' | 'overview' | 'calendar' | 'library' | 'agent' | 'settings';

  let activeSection = $state<NavSection>('editor');
  let workspaceInfo = $state<WorkspaceInfo | null>(null);
  let doc = $state<DocumentModel | null>(null);
  let exportsList = $state<ExportRecord[]>([]);
  let libraryOverview = $state<LibraryOverview | null>(null);
  let libraryLoading = $state(false);
  let selectedLayerId = $state<string | null>(null);
  let showReference = $state(false);
  let stageScale = $state(0.75);
  let statusMessage = $state('Inicializando InstaGo (R1)...');
  let errorMessage = $state('');
  let fontsSummary = $state('Carregando fontes...');
  let busy = $state(false);
  let canUndo = $state(false);
  let canRedo = $state(false);

  let canvasEl = $state<HTMLCanvasElement | null>(null);
  let fabricCanvas: Canvas | null = null;
  let history: HistoryStack<DocumentModel> | null = null;
  let suppressCanvasEvents = false;

  const selectedLayer = $derived.by<Layer | null>(() => {
    if (!doc || !selectedLayerId) return null;
    const page = doc.pages[0];
    return page?.layers.find((l) => l.id === selectedLayerId) ?? null;
  });

  const orderedLayers = $derived.by<Layer[]>(() => {
    if (!doc || !doc.pages[0]) return [];
    return [...doc.pages[0].layers].sort((a, b) => b.zIndex - a.zIndex);
  });

  function updateHistoryFlags() {
    canUndo = history?.canUndo ?? false;
    canRedo = history?.canRedo ?? false;
  }

  async function mountCanvas(targetDoc: DocumentModel) {
    await tick();
    if (!canvasEl) return;

    if (!fabricCanvas) {
      fabricCanvas = new Canvas(canvasEl, {
        width: targetDoc.workspaceWidth,
        height: targetDoc.workspaceHeight,
        preserveObjectStacking: true,
        selection: true
      });

      fabricCanvas.on('selection:created', (e) => {
        selectedLayerId = getLayerId(e.selected?.[0]);
      });
      fabricCanvas.on('selection:updated', (e) => {
        selectedLayerId = getLayerId(e.selected?.[0]);
      });
      fabricCanvas.on('selection:cleared', () => {
        selectedLayerId = null;
      });
      fabricCanvas.on('object:modified', () => {
        if (!suppressCanvasEvents) commitFromCanvas();
      });
      fabricCanvas.on('text:changed', () => {
        if (!suppressCanvasEvents) commitFromCanvas();
      });
    }

    suppressCanvasEvents = true;
    try {
      await renderPageToCanvas(fabricCanvas, targetDoc.pages[0]);
      applyStageZoom();
    } finally {
      suppressCanvasEvents = false;
    }
  }

  function applyStageZoom() {
    if (!fabricCanvas || !doc) return;
    fabricCanvas.setDimensions({
      width: Math.round(doc.workspaceWidth * stageScale),
      height: Math.round(doc.workspaceHeight * stageScale)
    });
    fabricCanvas.setZoom(stageScale);
    fabricCanvas.renderAll();
  }

  function commitFromCanvas() {
    if (!fabricCanvas || !doc || !history) return;
    const syncedPage = syncCanvasToPage(fabricCanvas, doc.pages[0]);
    const nextDoc: DocumentModel = {
      ...doc,
      pages: [syncedPage, ...doc.pages.slice(1)]
    };
    doc = nextDoc;
    history.push(nextDoc);
    updateHistoryFlags();
    statusMessage = 'Alteração registrada no histórico local (não salva no SQLite).';
  }

  async function applyHistorySnapshot(snapshot: DocumentModel | null) {
    if (!snapshot || !fabricCanvas) return;
    doc = snapshot;
    suppressCanvasEvents = true;
    try {
      await renderPageToCanvas(fabricCanvas, snapshot.pages[0]);
      applyStageZoom();
    } finally {
      suppressCanvasEvents = false;
    }
    updateHistoryFlags();
  }

  async function handleUndo() {
    if (!history) return;
    await applyHistorySnapshot(history.undo());
    statusMessage = 'Desfazer aplicado.';
  }

  async function handleRedo() {
    if (!history) return;
    await applyHistorySnapshot(history.redo());
    statusMessage = 'Refazer aplicado.';
  }

  async function handleSave() {
    if (!doc || !fabricCanvas) return;
    busy = true;
    errorMessage = '';
    try {
      const syncedPage = syncCanvasToPage(fabricCanvas, doc.pages[0]);
      const payload: DocumentModel = {
        ...doc,
        pages: [syncedPage, ...doc.pages.slice(1)],
        canvasJson: JSON.stringify(fabricCanvas.toJSON())
      };
      const saved = await bridge.saveDocument(payload, 'editor save');
      doc = saved;
      history?.reset(saved);
      updateHistoryFlags();
      workspaceInfo = await bridge.getWorkspaceInfo();
      statusMessage = `Salvo no SQLite (revisão ${saved.revision}).`;
    } catch (err) {
      errorMessage = `Falha ao salvar: ${String(err)}`;
    } finally {
      busy = false;
    }
  }

  async function handleReopen() {
    if (!doc) return;
    busy = true;
    errorMessage = '';
    try {
      const loaded = await bridge.getDocument(doc.id);
      doc = loaded;
      history = new HistoryStack(loaded);
      updateHistoryFlags();
      await mountCanvas(loaded);
      exportsList = (await bridge.listExports(loaded.id)) ?? [];
      workspaceInfo = await bridge.getWorkspaceInfo();
      statusMessage = `Reaberto do SQLite (revisão ${loaded.revision}).`;
    } catch (err) {
      errorMessage = `Falha ao reabrir: ${String(err)}`;
    } finally {
      busy = false;
    }
  }

  async function handleResetSeed() {
    busy = true;
    errorMessage = '';
    try {
      const resetDoc = await bridge.resetSeedDocument();
      doc = resetDoc;
      selectedLayerId = null;
      history = new HistoryStack(resetDoc);
      updateHistoryFlags();
      await mountCanvas(resetDoc);
      statusMessage = `Base V5 restaurada (revisão ${resetDoc.revision}).`;
    } catch (err) {
      errorMessage = `Falha ao restaurar base V5: ${String(err)}`;
    } finally {
      busy = false;
    }
  }

  async function handleExport(format: 'png' | 'jpg') {
    if (!doc || !fabricCanvas) return;
    busy = true;
    errorMessage = '';
    try {
      const dataUrl = exportCanvasDataUrl(
        fabricCanvas,
        format,
        doc.exportScale,
        doc.workspaceWidth,
        doc.workspaceHeight
      );
      const rec = await bridge.exportAsset(doc.id, format, dataUrl);
      exportsList = [rec, ...exportsList];
      workspaceInfo = await bridge.getWorkspaceInfo();
      statusMessage = `Exportado ${rec.width}×${rec.height} (${rec.format.toUpperCase()}) em ${rec.filePath}`;
    } catch (err) {
      errorMessage = `Falha na exportação: ${String(err)}`;
    } finally {
      busy = false;
    }
  }

  function selectLayerOnCanvas(layerId: string) {
    selectedLayerId = layerId;
    if (!fabricCanvas) return;
    const obj = fabricCanvas.getObjects().find((o) => getLayerId(o) === layerId);
    if (obj) {
      fabricCanvas.setActiveObject(obj);
      fabricCanvas.renderAll();
    }
  }

  async function mutateSelectedLayer(mutator: (layer: Layer) => Layer) {
    if (!doc || !selectedLayerId || !fabricCanvas) return;
    const page = doc.pages[0];
    const nextLayers = page.layers.map((l) => (l.id === selectedLayerId ? mutator(l) : l));
    const nextDoc: DocumentModel = { ...doc, pages: [{ ...page, layers: nextLayers }, ...doc.pages.slice(1)] };
    doc = nextDoc;
    history?.push(nextDoc);
    updateHistoryFlags();
    suppressCanvasEvents = true;
    try {
      await renderPageToCanvas(fabricCanvas, nextDoc.pages[0]);
      applyStageZoom();
      selectLayerOnCanvas(selectedLayerId);
    } finally {
      suppressCanvasEvents = false;
    }
  }

  async function toggleLayerProp(layerId: string, prop: 'visible' | 'locked') {
    if (!doc || !fabricCanvas) return;
    const page = doc.pages[0];
    const nextLayers = page.layers.map((l) => (l.id === layerId ? { ...l, [prop]: !l[prop] } : l));
    const nextDoc: DocumentModel = { ...doc, pages: [{ ...page, layers: nextLayers }, ...doc.pages.slice(1)] };
    doc = nextDoc;
    history?.push(nextDoc);
    updateHistoryFlags();
    suppressCanvasEvents = true;
    try {
      await renderPageToCanvas(fabricCanvas, nextDoc.pages[0]);
      applyStageZoom();
    } finally {
      suppressCanvasEvents = false;
    }
  }

  async function moveLayerOrder(layerId: string, delta: 1 | -1) {
    if (!doc || !fabricCanvas) return;
    const sorted = [...doc.pages[0].layers].sort((a, b) => a.zIndex - b.zIndex);
    const idx = sorted.findIndex((l) => l.id === layerId);
    const targetIdx = idx + delta;
    if (idx < 0 || targetIdx < 0 || targetIdx >= sorted.length) return;
    const [item] = sorted.splice(idx, 1);
    sorted.splice(targetIdx, 0, item);
    const reindexed = sorted.map((l, i) => ({ ...l, zIndex: i + 1 }));
    const nextDoc: DocumentModel = { ...doc, pages: [{ ...doc.pages[0], layers: reindexed }, ...doc.pages.slice(1)] };
    doc = nextDoc;
    history?.push(nextDoc);
    updateHistoryFlags();
    suppressCanvasEvents = true;
    try {
      await renderPageToCanvas(fabricCanvas, nextDoc.pages[0]);
      applyStageZoom();
      selectLayerOnCanvas(layerId);
    } finally {
      suppressCanvasEvents = false;
    }
  }

  function addQuickText() {
    if (!doc || !fabricCanvas) return;
    const id = `text-${Date.now()}`;
    const newLayer: Layer = {
      id,
      name: 'Novo texto',
      kind: 'text',
      x: 60,
      y: 230,
      width: 300,
      height: 40,
      scaleX: 1,
      scaleY: 1,
      angle: 0,
      opacity: 1,
      visible: true,
      locked: false,
      zIndex: doc.pages[0].layers.length + 1,
      fill: '#374F1E',
      text: 'Texto editável',
      fontFamily: 'Sora',
      fontSize: 28,
      fontWeight: '700',
      lineHeight: 1.15,
      textAlign: 'left'
    };
    const tb = new Textbox(newLayer.text!, {
      originX: 'left',
      originY: 'top',
      left: newLayer.x,
      top: newLayer.y,
      width: newLayer.width,
      fill: newLayer.fill,
      fontFamily: newLayer.fontFamily,
      fontSize: newLayer.fontSize,
      fontWeight: newLayer.fontWeight
    });
    tagLayerObject(tb, id);
    doc = {
      ...doc,
      pages: [{ ...doc.pages[0], layers: [...doc.pages[0].layers, newLayer] }, ...doc.pages.slice(1)]
    };
    fabricCanvas.add(tb);
    fabricCanvas.setActiveObject(tb);
    selectedLayerId = id;
    commitFromCanvas();
  }

  function addQuickRect() {
    if (!doc || !fabricCanvas) return;
    const id = `rect-${Date.now()}`;
    const newLayer: Layer = {
      id,
      name: 'Forma / destaque',
      kind: 'rect',
      x: 380,
      y: 230,
      width: 160,
      height: 48,
      scaleX: 1,
      scaleY: 1,
      angle: 0,
      opacity: 1,
      visible: true,
      locked: false,
      zIndex: doc.pages[0].layers.length + 1,
      fill: '#98B432',
      radius: 8
    };
    const rect = new Rect({
      originX: 'left',
      originY: 'top',
      left: newLayer.x,
      top: newLayer.y,
      width: newLayer.width,
      height: newLayer.height,
      fill: newLayer.fill,
      rx: newLayer.radius,
      ry: newLayer.radius
    });
    tagLayerObject(rect, id);
    doc = {
      ...doc,
      pages: [{ ...doc.pages[0], layers: [...doc.pages[0].layers, newLayer] }, ...doc.pages.slice(1)]
    };
    fabricCanvas.add(rect);
    fabricCanvas.setActiveObject(rect);
    selectedLayerId = id;
    commitFromCanvas();
  }

  async function refreshLibrary() {
    libraryLoading = true;
    errorMessage = '';
    try {
      libraryOverview = await bridge.getLibraryOverview({});
    } catch (err) {
      errorMessage = `Erro ao carregar biblioteca: ${String(err)}`;
    } finally {
      libraryLoading = false;
    }
  }

  async function switchSection(next: NavSection) {
    if (next === activeSection) return;
    if (activeSection === 'editor' && fabricCanvas) {
      fabricCanvas.dispose();
      fabricCanvas = null;
    }
    activeSection = next;
    if (next === 'editor' && doc) {
      await mountCanvas(doc);
    } else if ((next === 'library' || next === 'settings') && !libraryOverview) {
      await refreshLibrary();
    }
  }

  async function handleOpenInEditor(documentId: string) {
    busy = true;
    errorMessage = '';
    try {
      const loaded = await bridge.getDocument(documentId);
      doc = loaded;
      history = new HistoryStack(loaded);
      updateHistoryFlags();
      selectedLayerId = null;
      await switchSection('editor');
      statusMessage = `Documento "${loaded.title}" aberto no editor`;
    } catch (err) {
      errorMessage = `Erro ao abrir documento no editor: ${String(err)}`;
    } finally {
      busy = false;
    }
  }

  async function handleSaveProfile(profile: CompanyProfile) {
    busy = true;
    errorMessage = '';
    try {
      const saved = await bridge.updateProfile(profile);
      if (libraryOverview) {
        libraryOverview = { ...libraryOverview, profile: saved };
      }
      statusMessage = `Perfil "${saved.brand}" salvo no SQLite`;
    } catch (err) {
      errorMessage = `Erro ao salvar perfil: ${String(err)}`;
    } finally {
      busy = false;
    }
  }

  async function handleReimportBase(sourceDir: string) {
    busy = true;
    errorMessage = '';
    try {
      const sum = await bridge.importBase(sourceDir);
      await refreshLibrary();
      statusMessage = `Base reimportada: ${sum.uniqueArtsCount} artes únicas, ${sum.auxiliaryCoversCount} capas extras, ${sum.totalAssetsCount} itens`;
    } catch (err) {
      errorMessage = `Erro ao reimportar base: ${String(err)}`;
    } finally {
      busy = false;
    }
  }

  async function handleVerifyIntegrity(): Promise<IntegrityReport | null> {
    busy = true;
    errorMessage = '';
    try {
      const rep = await bridge.verifyIntegrity();
      statusMessage = `Integridade conferida: ${rep.okCount}/${rep.totalAssets} OK`;
      return rep;
    } catch (err) {
      errorMessage = `Erro na verificação de integridade: ${String(err)}`;
      return null;
    } finally {
      busy = false;
    }
  }

  async function handleCreateBackup(): Promise<BackupResult | null> {
    busy = true;
    errorMessage = '';
    try {
      const res = await bridge.createBackup();
      statusMessage = `Backup criado: ${res.fileName} (${res.archivedFiles} arquivos)`;
      return res;
    } catch (err) {
      errorMessage = `Erro ao gerar backup: ${String(err)}`;
      return null;
    } finally {
      busy = false;
    }
  }

  async function handleOpenFolder(
    scope: 'library' | 'asset_dir' | 'exports' | 'backups' | 'creative_base',
    assetId = ''
  ) {
    errorMessage = '';
    try {
      const opened = await bridge.openFolder(scope, assetId);
      statusMessage = `Pasta aberta: ${opened}`;
    } catch (err) {
      errorMessage = `Erro ao abrir pasta: ${String(err)}`;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    const tag = (e.target as HTMLElement | null)?.tagName;
    const isInput = tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT';
    if (activeSection !== 'editor') return;
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'z' && !e.shiftKey && !isInput) {
      e.preventDefault();
      void handleUndo();
    } else if ((e.ctrlKey || e.metaKey) && (e.key.toLowerCase() === 'y' || (e.key.toLowerCase() === 'z' && e.shiftKey)) && !isInput) {
      e.preventDefault();
      void handleRedo();
    } else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
      e.preventDefault();
      void handleSave();
    }
  }

  onMount(async () => {
    window.addEventListener('keydown', handleKeydown);
    try {
      const fontResult = await loadBrandFonts();
      fontsSummary = fontResult.loaded.length
        ? `Fontes locais: ${fontResult.loaded.join(', ')}`
        : 'Fontes locais indisponíveis (fallback do sistema)';

      workspaceInfo = await bridge.getWorkspaceInfo();
      const seeded = await bridge.ensureSeedDocument();
      doc = seeded;
      history = new HistoryStack(seeded);
      updateHistoryFlags();
      await mountCanvas(seeded);
      exportsList = (await bridge.listExports(seeded.id)) ?? [];
      await refreshLibrary();
      const artsCount = libraryOverview?.summary?.uniqueArtsCount ?? 0;
      statusMessage = `Pronto · R1 (${seeded.exportWidth}×${seeded.exportHeight} · ${artsCount} artes únicas na biblioteca)`;
    } catch (err) {
      errorMessage = `Erro ao iniciar ponte Go/Wails: ${String(err)}`;
      statusMessage = 'Falha na inicialização';
    }
  });

  onDestroy(() => {
    window.removeEventListener('keydown', handleKeydown);
    fabricCanvas?.dispose();
    fabricCanvas = null;
  });
</script>

<div class="app-shell" class:no-toolbar={activeSection !== 'editor'}>
  <header class="nav-bar">
    <div class="brand-title">
      <Layers size={16} />
      <span>InstaGo</span>
      <span class="badge">R1 · Biblioteca e Perfil</span>
    </div>

    <nav class="nav-tabs" aria-label="Áreas principais">
      <button class="nav-tab" class:active={activeSection === 'editor'} onclick={() => switchSection('editor')}>
        Editor
      </button>
      <button class="nav-tab" class:active={activeSection === 'overview'} onclick={() => switchSection('overview')}>
        Visão geral
      </button>
      <button class="nav-tab" class:active={activeSection === 'calendar'} onclick={() => switchSection('calendar')}>
        Calendário
      </button>
      <button class="nav-tab" class:active={activeSection === 'library'} onclick={() => switchSection('library')}>
        Biblioteca
      </button>
      <button class="nav-tab" class:active={activeSection === 'agent'} onclick={() => switchSection('agent')}>
        Agente
      </button>
      <button class="nav-tab" class:active={activeSection === 'settings'} onclick={() => switchSection('settings')}>
        Configurações
      </button>
    </nav>
  </header>

  {#if activeSection === 'editor'}
  <div class="toolbar">
    <div class="toolbar-group">
      <button onclick={handleUndo} disabled={!canUndo || busy} title="Desfazer (Ctrl+Z)">
        <Undo2 size={14} /> Desfazer
      </button>
      <button onclick={handleRedo} disabled={!canRedo || busy} title="Refazer (Ctrl+Shift+Z)">
        <Redo2 size={14} /> Refazer
      </button>
      <button onclick={addQuickText} disabled={!doc || busy}>
        <Type size={14} /> + Texto
      </button>
      <button onclick={addQuickRect} disabled={!doc || busy}>
        <Square size={14} /> + Forma
      </button>
      <button onclick={() => (showReference = !showReference)} disabled={!doc?.referenceImageUrl}>
        <Columns2 size={14} /> {showReference ? 'Ocultar V5' : 'Comparar V5'}
      </button>
      <label class="badge">
        Zoom:
        <select
          bind:value={stageScale}
          onchange={() => applyStageZoom()}
          aria-label="Zoom da prancheta"
          style="border:0;background:transparent;font-weight:600"
        >
          <option value={0.6}>60%</option>
          <option value={0.75}>75%</option>
          <option value={1}>100%</option>
        </select>
      </label>
    </div>

    <div class="toolbar-group">
      <button onclick={handleResetSeed} disabled={!doc || busy} title="Restaurar composição V5 original">
        <RotateCcw size={14} /> Restaurar V5
      </button>
      <button onclick={handleReopen} disabled={!doc || busy}>
        <FolderOpen size={14} /> Reabrir SQLite
      </button>
      <button class="primary" onclick={handleSave} disabled={!doc || busy}>
        <Save size={14} /> Salvar
      </button>
      <button class="accent" onclick={() => handleExport('png')} disabled={!doc || busy}>
        <Download size={14} /> Exportar PNG
      </button>
      <button onclick={() => handleExport('jpg')} disabled={!doc || busy}>
        <Download size={14} /> JPG
      </button>
    </div>
  </div>
  {/if}

  {#if activeSection === 'editor'}
    <main class="workspace">
      <aside class="panel">
        <div class="panel-section">
          <h3>Camadas ({orderedLayers.length})</h3>
          <div class="layer-list">
            {#each orderedLayers as layer (layer.id)}
              <div class="layer-row" class:selected={layer.id === selectedLayerId}>
                <button class="layer-name" onclick={() => selectLayerOnCanvas(layer.id)} title={layer.name}>
                  {layer.name}
                </button>
                <button class="icon-btn" onclick={() => moveLayerOrder(layer.id, 1)} title="Trazer para frente">
                  <ArrowUp size={12} />
                </button>
                <button class="icon-btn" onclick={() => moveLayerOrder(layer.id, -1)} title="Enviar para trás">
                  <ArrowDown size={12} />
                </button>
                <button class="icon-btn" onclick={() => toggleLayerProp(layer.id, 'locked')} title="Bloquear/Desbloquear">
                  {#if layer.locked}<Lock size={12} />{:else}<Unlock size={12} />{/if}
                </button>
                <button class="icon-btn" onclick={() => toggleLayerProp(layer.id, 'visible')} title="Mostrar/Ocultar">
                  {#if layer.visible}<Eye size={12} />{:else}<EyeOff size={12} />{/if}
                </button>
              </div>
            {/each}
          </div>
        </div>

        {#if workspaceInfo}
          <div class="panel-section">
            <h3>Baseline R0</h3>
            <div>App: <strong>{workspaceInfo.appName}</strong> ({workspaceInfo.appVersion})</div>
            <div>Go: {workspaceInfo.goVersion} ({workspaceInfo.os}/{workspaceInfo.arch})</div>
            <div>Heap Go: {workspaceInfo.heapAllocMb.toFixed(1)} MB · RSS: {workspaceInfo.processRssMb.toFixed(1)} MB</div>
            <div>{fontsSummary}</div>
            <div>
              Base criativa:
              <strong style="color:{workspaceInfo.creativeBaseOk ? '#3D6A2F' : '#A63329'}">
                {workspaceInfo.creativeBaseOk ? 'Localizada' : 'Não encontrada'}
              </strong>
            </div>
          </div>
        {/if}
      </aside>

      <section class="canvas-stage">
        {#if doc}
          <div class="artboard-frame" style="width:{Math.round(doc.workspaceWidth * stageScale) + 24}px">
            <div class="artboard-meta">
              <span title={doc.title}><strong>{doc.title}</strong></span>
              <span>Rev {doc.revision} · {doc.exportWidth}×{doc.exportHeight}px</span>
            </div>
            <div class="canvas-wrap">
              <canvas bind:this={canvasEl}></canvas>
            </div>
          </div>

          {#if showReference && doc.referenceImageUrl}
            <div class="artboard-frame">
              <div class="artboard-meta">
                <span><strong>Referência V5 original</strong></span>
                <span>02-combo-02.jpg (1080×1350)</span>
              </div>
              <img class="reference-img" src={doc.referenceImageUrl} alt="Referência V5 02-combo-02" />
            </div>
          {/if}
        {:else}
          <div class="panel-section" style="max-width:420px;margin-top:40px">
            <h3>Carregando documento R0</h3>
            <p style="margin:0">{errorMessage || statusMessage}</p>
          </div>
        {/if}
      </section>

      <aside class="panel right">
        <div class="panel-section">
          <h3>Propriedades</h3>
          {#if selectedLayer}
            <div class="field">
              <label for="layer-name">Camada</label>
              <input
                id="layer-name"
                value={selectedLayer.name}
                onchange={(e) => mutateSelectedLayer((l) => ({ ...l, name: e.currentTarget.value }))}
              />
            </div>

            <div class="grid-2">
              <div class="field">
                <label for="layer-x">X</label>
                <input
                  id="layer-x"
                  type="number"
                  value={Math.round(selectedLayer.x)}
                  onchange={(e) => mutateSelectedLayer((l) => ({ ...l, x: Number(e.currentTarget.value) }))}
                />
              </div>
              <div class="field">
                <label for="layer-y">Y</label>
                <input
                  id="layer-y"
                  type="number"
                  value={Math.round(selectedLayer.y)}
                  onchange={(e) => mutateSelectedLayer((l) => ({ ...l, y: Number(e.currentTarget.value) }))}
                />
              </div>
            </div>

            {#if selectedLayer.kind === 'text'}
              <div class="field">
                <label for="layer-text">Texto</label>
                <textarea
                  id="layer-text"
                  rows={3}
                  value={selectedLayer.text ?? ''}
                  onchange={(e) => mutateSelectedLayer((l) => ({ ...l, text: e.currentTarget.value }))}
                ></textarea>
              </div>
              <div class="grid-2">
                <div class="field">
                  <label for="layer-font">Fonte</label>
                  <select
                    id="layer-font"
                    value={selectedLayer.fontFamily ?? 'Inter'}
                    onchange={(e) => mutateSelectedLayer((l) => ({ ...l, fontFamily: e.currentTarget.value }))}
                  >
                    <option value="Sora">Sora</option>
                    <option value="Inter">Inter</option>
                  </select>
                </div>
                <div class="field">
                  <label for="layer-size">Tamanho</label>
                  <input
                    id="layer-size"
                    type="number"
                    min="8"
                    max="180"
                    value={selectedLayer.fontSize ?? 20}
                    onchange={(e) => mutateSelectedLayer((l) => ({ ...l, fontSize: Number(e.currentTarget.value) }))}
                  />
                </div>
              </div>
            {/if}

            {#if selectedLayer.kind !== 'image'}
              <div class="grid-2">
                <div class="field">
                  <label for="layer-fill">Cor</label>
                  <input
                    id="layer-fill"
                    type="color"
                    value={selectedLayer.fill ?? '#374F1E'}
                    onchange={(e) => mutateSelectedLayer((l) => ({ ...l, fill: e.currentTarget.value }))}
                  />
                </div>
                <div class="field">
                  <label for="layer-opacity">Opacidade</label>
                  <input
                    id="layer-opacity"
                    type="number"
                    min="0"
                    max="1"
                    step="0.05"
                    value={selectedLayer.opacity}
                    onchange={(e) => mutateSelectedLayer((l) => ({ ...l, opacity: Number(e.currentTarget.value) }))}
                  />
                </div>
              </div>
            {/if}
          {:else}
            <p style="margin:0;color:var(--text-muted)">Selecione um elemento no canvas ou na lista de camadas.</p>
          {/if}
        </div>

        <div class="panel-section">
          <h3>Exportações verificadas ({exportsList.length})</h3>
          {#if exportsList.length === 0}
            <p style="margin:0;color:var(--text-muted)">Nenhuma exportação registrada ainda nesta sessão.</p>
          {:else}
            {#each exportsList.slice(0, 5) as item (item.id)}
              <div style="border-top:1px solid var(--border);padding-top:6px">
                <div><strong>{item.format.toUpperCase()}</strong> · {item.width}×{item.height} · rev {item.revision}</div>
                <div class="mono">{item.filePath}</div>
                <div class="mono">sha256: {item.sha256.slice(0, 16)}…</div>
              </div>
            {/each}
          {/if}
        </div>
      </aside>
    </main>
  {:else if activeSection === 'library'}
    <LibraryView
      overview={libraryOverview}
      loading={libraryLoading}
      onRefresh={refreshLibrary}
      onOpenFolder={handleOpenFolder}
      onOpenInEditor={handleOpenInEditor}
      onNotify={(msg) => (statusMessage = msg)}
    />
  {:else if activeSection === 'settings'}
    <SettingsView
      overview={libraryOverview}
      {workspaceInfo}
      {fontsSummary}
      {busy}
      onSaveProfile={handleSaveProfile}
      onReimport={handleReimportBase}
      onVerifyIntegrity={handleVerifyIntegrity}
      onCreateBackup={handleCreateBackup}
      onOpenFolder={handleOpenFolder}
    />
  {:else}
    <main style="padding:24px;overflow:auto">
      <div class="panel-section" style="max-width:640px">
        <h3>Área prevista no Roadmap ({activeSection})</h3>
        <p style="margin:0">
          O shell wireframe já reserva as áreas previstas no DRS. Já estão ativos:
          <strong>Editor Nativo</strong> (R0) e <strong>Biblioteca, Perfil e Backup</strong> (R1). As áreas de
          <strong>Calendário</strong>, <strong>Agente</strong> e <strong>Visão geral</strong> entram nas etapas R2–R5.
        </p>
      </div>
    </main>
  {/if}

  <footer class="status-bar">
    <span class:error-text={Boolean(errorMessage)}>{errorMessage || statusMessage}</span>
    <span>{workspaceInfo ? `DB: ${workspaceInfo.dbPath}` : ''}</span>
  </footer>
</div>
