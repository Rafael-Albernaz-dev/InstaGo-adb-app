<script lang="ts">
  import { onDestroy, onMount, tick } from 'svelte';
  import { Events } from '@wailsio/runtime';
  import { Canvas } from 'fabric';
  import {
    AlertTriangle,
    ArrowDown,
    ArrowUp,
    CheckCircle2,
    Columns2,
    Download,
    Eye,
    EyeOff,
    FolderOpen,
    Image as ImageIcon,
    Lock,
    MapPin,
    Plus,
    Redo2,
    Save,
    Square,
    Type,
    Undo2,
    Unlock
  } from '@lucide/svelte';
  import AgentBotWidget from './components/AgentBotWidget.svelte';
  import AnnotationOverlay from './components/AnnotationOverlay.svelte';
  import CompareVersionsModal from './components/CompareVersionsModal.svelte';
  import EditorRightPanel from './components/EditorRightPanel.svelte';
  import ImagePickerModal from './components/ImagePickerModal.svelte';
  import LibraryView from './components/LibraryView.svelte';
  import NewPieceModal from './components/NewPieceModal.svelte';
  import PageStrip from './components/PageStrip.svelte';
  import SettingsView from './components/SettingsView.svelte';
  import { bridge } from './lib/bridge';
  import {
    getLayerId,
    renderPageOffscreenDataUrl,
    renderPageSnapshotWithAnnotationsDataUrl,
    renderPageToCanvas,
    syncCanvasToPage,
    updateCanvasBackgroundInPlace,
    updateLayerObjectInPlace
  } from './lib/canvasMapper';
  import {
    alignLayerOnPage,
    createAnnotation,
    createBlankPage,
    createShortId,
    duplicatePage,
    formatDocStatusLabel,
    movePage,
    removePageAt,
    type LayerAlignment
  } from './lib/editorHelpers';
  import { loadBrandFonts } from './lib/fonts';
  import { HistoryStack } from './lib/history';
  import type {
    AgentAnnotation,
    AgentDeliveryMode,
    AgentLogEntry,
    AgentTask,
    BackupResult,
    CLIHealth,
    CompanyProfile,
    DocumentModel,
    ExportBundlePageInput,
    ExportBundleRecord,
    ExportRecord,
    IntegrityReport,
    Layer,
    LibraryAsset,
    LibraryOverview,
    OfferCheckResult,
    Page,
    RevisionSummary,
    WorkspaceInfo
  } from './lib/types';

  type NavSection = 'editor' | 'library' | 'settings';

  let activeSection = $state<NavSection>('editor');
  let workspaceInfo = $state<WorkspaceInfo | null>(null);
  let documentsList = $state<DocumentModel[]>([]);
  let doc = $state<DocumentModel | null>(null);
  let activePageIndex = $state(0);
  let exportsList = $state<ExportRecord[]>([]);
  let bundlesList = $state<ExportBundleRecord[]>([]);
  let revisions = $state<RevisionSummary[]>([]);
  let offerCheck = $state<OfferCheckResult | null>(null);
  let libraryOverview = $state<LibraryOverview | null>(null);
  let libraryLoading = $state(false);
  let selectedLayerId = $state<string | null>(null);
  let selectedAnnotationId = $state<string | null>(null);
  let markingMode = $state<'off' | 'pin' | 'rect'>('off');
  let showMarks = $state(true);
  let showReference = $state(false);

  // Floating Codex Agent state
  let agentOpen = $state(false);
  let agentHealth = $state<CLIHealth | null>(null);
  let agentHealthLoading = $state(false);
  let agentTasks = $state<AgentTask[]>([]);
  let activeAgentTask = $state<AgentTask | null>(null);
  let agentLogs = $state<AgentLogEntry[]>([]);
  let agentPromptDraft = $state('');
  let agentDeliveryMode = $state<AgentDeliveryMode>('auto');
  let agentFocusTrigger = $state(0);
  let unsubAgentLog: (() => void) | null = null;
  let unsubAgentTask: (() => void) | null = null;
  let stageScale = $state(0.75);
  let statusMessage = $state('Inicializando InstaGo...');
  let errorMessage = $state('');
  let fontsSummary = $state('Carregando fontes...');
  let busy = $state(false);
  let canUndo = $state(false);
  let canRedo = $state(false);

  // Modals & prompts
  let showNewPieceModal = $state(false);
  let showCompareModal = $state(false);
  let imagePickerOpen = $state(false);
  let imagePickerMode = $state<'add' | 'replace'>('add');
  let importingImage = $state(false);
  let pendingConflictApprove = $state<OfferCheckResult | null>(null);
  let lastExportBundle = $state<ExportBundleRecord | null>(null);

  let canvasEl = $state<HTMLCanvasElement | null>(null);
  let fabricCanvas: Canvas | null = null;
  let history: HistoryStack<DocumentModel> | null = null;
  let suppressCanvasEvents = false;
  let autosaveTimer: ReturnType<typeof setTimeout> | null = null;
  let autosaveToken = 0;

  const activePage = $derived.by<Page | null>(() => {
    if (!doc || doc.pages.length === 0) return null;
    const safeIdx = Math.min(Math.max(0, activePageIndex), doc.pages.length - 1);
    return doc.pages[safeIdx];
  });

  const selectedLayer = $derived.by<Layer | null>(() => {
    if (!activePage || !selectedLayerId) return null;
    return activePage.layers.find((l) => l.id === selectedLayerId) ?? null;
  });

  const orderedLayers = $derived.by<Layer[]>(() => {
    if (!activePage) return [];
    return [...activePage.layers].sort((a, b) => b.zIndex - a.zIndex);
  });

  function updateHistoryFlags() {
    canUndo = history?.canUndo ?? false;
    canRedo = history?.canRedo ?? false;
  }

  function clearAutosaveTimer() {
    if (autosaveTimer !== null) {
      clearTimeout(autosaveTimer);
      autosaveTimer = null;
    }
  }

  function syncLockOnCanvas() {
    if (!fabricCanvas || !doc) return;
    const isApproved = doc.status === 'approved';
    fabricCanvas.selection = !isApproved;
    for (const obj of fabricCanvas.getObjects()) {
      if (isApproved) {
        obj.set({ selectable: false, evented: false });
      } else {
        obj.set({ selectable: true, evented: true });
      }
    }
    fabricCanvas.renderAll();
  }

  async function renderActivePageToCanvas(targetDoc: DocumentModel, pageIdx: number) {
    await tick();
    if (!canvasEl) return;

    const safeIdx = Math.min(Math.max(0, pageIdx), targetDoc.pages.length - 1);
    const page = targetDoc.pages[safeIdx];

    if (!fabricCanvas) {
      fabricCanvas = new Canvas(canvasEl, {
        width: targetDoc.workspaceWidth,
        height: targetDoc.workspaceHeight,
        preserveObjectStacking: true,
        selection: targetDoc.status !== 'approved'
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
        if (!suppressCanvasEvents) syncInlineTextFromCanvas();
      });
      fabricCanvas.on('text:editing:exited', () => {
        if (!suppressCanvasEvents) commitFromCanvas();
      });
    }

    suppressCanvasEvents = true;
    try {
      await renderPageToCanvas(fabricCanvas, page);
      applyStageZoom(targetDoc);
      syncLockOnCanvas();
    } finally {
      suppressCanvasEvents = false;
    }
  }

  function applyStageZoom(targetDoc: DocumentModel | null = doc) {
    if (!fabricCanvas || !targetDoc) return;
    fabricCanvas.setDimensions({
      width: Math.round(targetDoc.workspaceWidth * stageScale),
      height: Math.round(targetDoc.workspaceHeight * stageScale)
    });
    fabricCanvas.setZoom(stageScale);
    fabricCanvas.renderAll();
  }

  function withUpdatedPage(targetDoc: DocumentModel, pageIdx: number, updater: (p: Page) => Page): DocumentModel {
    const pages = targetDoc.pages.map((p, idx) => (idx === pageIdx ? updater(p) : p));
    return { ...targetDoc, pages };
  }

  function scheduleAutosaveAndCheck(nextDoc: DocumentModel) {
    clearAutosaveTimer();
    const token = ++autosaveToken;
    autosaveTimer = setTimeout(async () => {
      if (token !== autosaveToken || nextDoc.status === 'approved') return;
      try {
        await bridge.saveDraft(nextDoc);
        if (token !== autosaveToken) return;
        const check = await bridge.checkOfferConsistency(nextDoc);
        if (token !== autosaveToken) return;
        offerCheck = check;
        statusMessage = 'Rascunho salvo automaticamente.';
      } catch (err) {
        if (token === autosaveToken) {
          errorMessage = `Erro no salvamento automático: ${String(err)}`;
        }
      }
    }, 850);
  }

  function commitDocChange(nextDoc: DocumentModel, pushHistory = true, triggerAutosave = true) {
    doc = nextDoc;
    if (pushHistory && history) {
      history.push(nextDoc);
      updateHistoryFlags();
    }
    if (triggerAutosave && nextDoc.status !== 'approved') {
      scheduleAutosaveAndCheck(nextDoc);
    }
  }

  function flushCanvasIntoDoc(): DocumentModel | null {
    if (!fabricCanvas || !doc || !activePage) return doc;
    const synced = syncCanvasToPage(fabricCanvas, activePage);
    const updated = withUpdatedPage(doc, activePage.index, () => synced);
    doc = updated;
    return updated;
  }

  function blurFocusedButton() {
    const el = document.activeElement;
    if (el instanceof HTMLElement && el.tagName === 'BUTTON') {
      el.blur();
    }
  }

  function syncInlineTextFromCanvas() {
    if (!fabricCanvas || !doc || !activePage || doc.status === 'approved') return;
    const syncedPage = syncCanvasToPage(fabricCanvas, activePage);
    const nextDoc = withUpdatedPage(doc, activePage.index, () => syncedPage);
    commitDocChange(nextDoc, false, true);
  }

  function commitFromCanvas() {
    if (!fabricCanvas || !doc || !activePage || doc.status === 'approved') return;
    const syncedPage = syncCanvasToPage(fabricCanvas, activePage);
    const nextDoc = withUpdatedPage(doc, activePage.index, () => syncedPage);
    commitDocChange(nextDoc, true, true);
  }

  async function refreshDocumentSideData(targetDoc: DocumentModel) {
    const [revs, exps, bundles, check, allDocs] = await Promise.all([
      bridge.listRevisions(targetDoc.id),
      bridge.listExports(targetDoc.id),
      bridge.listExportBundles(targetDoc.id),
      bridge.checkOfferConsistency(targetDoc),
      bridge.listDocuments()
    ]);
    revisions = revs ?? [];
    exportsList = exps ?? [];
    bundlesList = bundles ?? [];
    offerCheck = check;
    documentsList = allDocs ?? [];
  }

  async function refreshAgentTasks(documentId: string) {
    try {
      const list = await bridge.listAgentTasks(documentId);
      agentTasks = list ?? [];
      if (agentTasks.length > 0 && (!activeAgentTask || activeAgentTask.documentId !== documentId)) {
        const det = await bridge.getAgentTask(agentTasks[0].id);
        activeAgentTask = det.task;
        agentLogs = det.logs ?? [];
      } else if (agentTasks.length === 0) {
        activeAgentTask = null;
        agentLogs = [];
      }
    } catch {
      // keep UI clean if agent task list unavailable during early startup
    }
  }

  async function refreshAgentHealth() {
    agentHealthLoading = true;
    try {
      agentHealth = await bridge.checkAgentHealth();
    } catch (err) {
      agentHealth = {
        available: false,
        binaryPath: '',
        version: '',
        authenticated: false,
        authStatusText: '',
        errorMessage: String(err)
      };
    } finally {
      agentHealthLoading = false;
    }
  }

  async function loadDocumentIntoEditor(targetDoc: DocumentModel, pageIndex = 0) {
    clearAutosaveTimer();
    autosaveToken++;
    pendingConflictApprove = null;
    lastExportBundle = null;
    doc = targetDoc;
    activePageIndex = Math.min(Math.max(0, pageIndex), Math.max(0, targetDoc.pages.length - 1));
    selectedLayerId = null;
    selectedAnnotationId = null;
    markingMode = 'off';
    history = new HistoryStack(targetDoc);
    updateHistoryFlags();
    if (activeSection === 'editor') {
      await renderActivePageToCanvas(targetDoc, activePageIndex);
    }
    await Promise.all([refreshDocumentSideData(targetDoc), refreshAgentTasks(targetDoc.id)]);
  }

  async function selectPage(nextIdx: number) {
    if (!doc || nextIdx < 0 || nextIdx >= doc.pages.length || nextIdx === activePageIndex) return;
    const flushed = flushCanvasIntoDoc();
    activePageIndex = nextIdx;
    selectedLayerId = null;
    selectedAnnotationId = null;
    if (flushed) {
      await renderActivePageToCanvas(flushed, nextIdx);
    }
  }

  async function handleAddPage() {
    if (!doc || doc.status === 'approved') return;
    const flushed = flushCanvasIntoDoc() ?? doc;
    const newPage = createBlankPage(flushed.pages.length);
    const nextDoc: DocumentModel = {
      ...flushed,
      pieceType: flushed.pages.length + 1 > 1 && flushed.format === 'feed' ? 'carousel' : flushed.pieceType,
      pages: [...flushed.pages, newPage]
    };
    activePageIndex = nextDoc.pages.length - 1;
    selectedLayerId = null;
    commitDocChange(nextDoc, true, true);
    await renderActivePageToCanvas(nextDoc, activePageIndex);
    statusMessage = `Página ${nextDoc.pages.length} adicionada.`;
  }

  async function handleDuplicatePage() {
    if (!doc || !activePage || doc.status === 'approved') return;
    const flushed = flushCanvasIntoDoc() ?? doc;
    const current = flushed.pages[activePageIndex];
    const cloned = duplicatePage(current);
    const before = flushed.pages.slice(0, activePageIndex + 1);
    const after = flushed.pages.slice(activePageIndex + 1);
    const pages = [...before, cloned, ...after].map((p, idx) => ({ ...p, index: idx }));
    const nextDoc: DocumentModel = {
      ...flushed,
      pieceType: pages.length > 1 && flushed.format === 'feed' ? 'carousel' : flushed.pieceType,
      pages
    };
    activePageIndex = activePageIndex + 1;
    selectedLayerId = null;
    commitDocChange(nextDoc, true, true);
    await renderActivePageToCanvas(nextDoc, activePageIndex);
    statusMessage = `Página ${activePageIndex} duplicada para a posição ${activePageIndex + 1}.`;
  }

  async function handleMovePage(delta: -1 | 1) {
    if (!doc || doc.status === 'approved') return;
    const flushed = flushCanvasIntoDoc() ?? doc;
    const reordered = movePage(flushed.pages, activePageIndex, delta);
    if (reordered === flushed.pages) return;
    const nextIdx = activePageIndex + delta;
    const nextDoc: DocumentModel = { ...flushed, pages: reordered };
    activePageIndex = nextIdx;
    commitDocChange(nextDoc, true, true);
    await renderActivePageToCanvas(nextDoc, nextIdx);
  }

  async function handleDeletePage() {
    if (!doc || doc.pages.length <= 1 || doc.status === 'approved') return;
    const flushed = flushCanvasIntoDoc() ?? doc;
    const remaining = removePageAt(flushed.pages, activePageIndex);
    const nextIdx = Math.min(activePageIndex, remaining.length - 1);
    const nextDoc: DocumentModel = { ...flushed, pages: remaining };
    activePageIndex = nextIdx;
    selectedLayerId = null;
    commitDocChange(nextDoc, true, true);
    await renderActivePageToCanvas(nextDoc, nextIdx);
    statusMessage = 'Página removida.';
  }

  async function applyHistorySnapshot(snapshot: DocumentModel | null) {
    if (!snapshot || !fabricCanvas) return;
    doc = snapshot;
    const safeIdx = Math.min(activePageIndex, snapshot.pages.length - 1);
    activePageIndex = safeIdx;
    await renderActivePageToCanvas(snapshot, safeIdx);
    updateHistoryFlags();
    scheduleAutosaveAndCheck(snapshot);
  }

  async function handleUndo() {
    if (!history || doc?.status === 'approved') return;
    await applyHistorySnapshot(history.undo());
    statusMessage = 'Desfazer aplicado.';
  }

  async function handleRedo() {
    if (!history || doc?.status === 'approved') return;
    await applyHistorySnapshot(history.redo());
    statusMessage = 'Refazer aplicado.';
  }

  async function handleSaveVersion(note = '') {
    if (!doc || !fabricCanvas) return;
    if (doc.status === 'approved') {
      statusMessage = "Peça aprovada protegida. Clique em 'Editar como nova versão'.";
      return;
    }
    busy = true;
    errorMessage = '';
    clearAutosaveTimer();
    try {
      const payload = flushCanvasIntoDoc() ?? doc;
      payload.canvasJson = JSON.stringify(fabricCanvas.toJSON());
      const saved = await bridge.saveDocument(payload, note.trim() || 'versão salva no editor');
      await loadDocumentIntoEditor(saved, activePageIndex);
      workspaceInfo = await bridge.getWorkspaceInfo();
      statusMessage = `Versão v${saved.revision} salva no histórico.`;
    } catch (err) {
      errorMessage = `Falha ao salvar versão: ${String(err)}`;
    } finally {
      busy = false;
    }
  }

  async function handleApprove(confirmConflicts = false) {
    if (!doc || !fabricCanvas) return;
    busy = true;
    errorMessage = '';
    clearAutosaveTimer();
    try {
      const payload = flushCanvasIntoDoc() ?? doc;
      payload.canvasJson = JSON.stringify(fabricCanvas.toJSON());
      const res = await bridge.approveDocument(payload, confirmConflicts);
      offerCheck = res.offerCheck;
      if (!res.approved) {
        pendingConflictApprove = res.offerCheck;
        statusMessage = 'Aprovação aguardando confirmação da divergência de preço.';
        return;
      }
      pendingConflictApprove = null;
      if (res.document) {
        await loadDocumentIntoEditor(res.document, activePageIndex);
        statusMessage = `Versão v${res.document.revision} aprovada e protegida contra edições acidentais.`;
      }
    } catch (err) {
      errorMessage = `Erro ao aprovar peça: ${String(err)}`;
    } finally {
      busy = false;
    }
  }

  async function handleUnlockDraft() {
    if (!doc) return;
    busy = true;
    errorMessage = '';
    try {
      const unlocked = await bridge.startDraftFromApproved(doc.id);
      await loadDocumentIntoEditor(unlocked, activePageIndex);
      statusMessage = `Nova versão v${unlocked.revision} aberta em rascunho (v${unlocked.approvedRevision} aprovada preservada).`;
    } catch (err) {
      errorMessage = `Erro ao abrir novo rascunho: ${String(err)}`;
    } finally {
      busy = false;
    }
  }

  async function handleRestoreRevision(revision: number) {
    if (!doc) return;
    busy = true;
    errorMessage = '';
    try {
      const restored = await bridge.restoreRevision(doc.id, revision);
      await loadDocumentIntoEditor(restored, 0);
      statusMessage = `Versão v${revision} restaurada como rascunho v${restored.revision}.`;
    } catch (err) {
      errorMessage = `Erro ao restaurar versão: ${String(err)}`;
    } finally {
      busy = false;
    }
  }

  async function handleExport(format: 'png' | 'jpg') {
    if (!doc || !fabricCanvas) return;
    busy = true;
    errorMessage = '';
    clearAutosaveTimer();
    try {
      const payload = flushCanvasIntoDoc() ?? doc;
      if (payload.status !== 'approved') {
        await bridge.saveDraft(payload);
      }

      const pagesInput: ExportBundlePageInput[] = [];
      for (let i = 0; i < payload.pages.length; i++) {
        const p = payload.pages[i];
        const dataBase64 = await renderPageOffscreenDataUrl(
          p,
          payload.workspaceWidth,
          payload.workspaceHeight,
          format,
          payload.exportScale
        );
        pagesInput.push({
          pageIndex: i,
          title: p.title,
          dataBase64
        });
      }

      const bundle = await bridge.exportBundle(payload.id, format, pagesInput);
      lastExportBundle = bundle;
      await refreshDocumentSideData(payload);
      workspaceInfo = await bridge.getWorkspaceInfo();
      statusMessage = `Exportado pacote (${bundle.pageCount} página(s), legenda.txt e manifesto.json) em ${bundle.bundleDir}`;
    } catch (err) {
      errorMessage = `Falha na exportação: ${String(err)}`;
    } finally {
      busy = false;
    }
  }

  function selectLayerOnCanvas(layerId: string) {
    selectedLayerId = layerId;
    if (!fabricCanvas || doc?.status === 'approved') return;
    const obj = fabricCanvas.getObjects().find((o) => getLayerId(o) === layerId);
    if (obj) {
      fabricCanvas.setActiveObject(obj);
      fabricCanvas.renderAll();
    }
  }

  async function handleUpdateLayer(patch: Partial<Layer>, commitHistory = true) {
    if (!doc || !activePage || !selectedLayerId || doc.status === 'approved') return;
    const targetLayerId = selectedLayerId;
    const patchKeys = Object.keys(patch);
    if (patchKeys.length === 0) {
      if (commitHistory) {
        commitDocChange(doc, true, true);
      }
      return;
    }

    const baseDoc = commitHistory ? (flushCanvasIntoDoc() ?? doc) : doc;
    const page = baseDoc.pages[activePageIndex];
    let updatedLayer: Layer | null = null;
    const nextLayers = page.layers.map((l) => {
      if (l.id !== targetLayerId) return l;
      updatedLayer = { ...l, ...patch };
      return updatedLayer;
    });
    if (!updatedLayer) return;

    const nextDoc = withUpdatedPage(baseDoc, activePageIndex, (p) => ({ ...p, layers: nextLayers }));
    commitDocChange(nextDoc, commitHistory, true);

    const needsImageReload =
      'assetUrl' in patch || 'objectFit' in patch || 'positionX' in patch || 'positionY' in patch;
    if (!needsImageReload && fabricCanvas && updateLayerObjectInPlace(fabricCanvas, updatedLayer)) {
      return;
    }

    await renderActivePageToCanvas(nextDoc, activePageIndex);
    selectLayerOnCanvas(targetLayerId);
  }

  async function handleAlignLayer(alignment: LayerAlignment) {
    if (!doc || !activePage || !selectedLayer || doc.status === 'approved') return;
    const aligned = alignLayerOnPage(selectedLayer, doc.workspaceWidth, doc.workspaceHeight, alignment);
    await handleUpdateLayer({ x: aligned.x, y: aligned.y }, true);
  }

  async function handleDeleteSelectedLayer() {
    if (!doc || !activePage || !selectedLayerId || doc.status === 'approved') return;
    const flushed = flushCanvasIntoDoc() ?? doc;
    const page = flushed.pages[activePageIndex];
    const nextLayers = page.layers
      .filter((l) => l.id !== selectedLayerId)
      .map((l, idx) => ({ ...l, zIndex: idx + 1 }));
    const nextDoc = withUpdatedPage(flushed, activePageIndex, (p) => ({ ...p, layers: nextLayers }));
    selectedLayerId = null;
    commitDocChange(nextDoc, true, true);
    await renderActivePageToCanvas(nextDoc, activePageIndex);
  }

  async function toggleLayerProp(layerId: string, prop: 'visible' | 'locked') {
    if (!doc || !activePage || doc.status === 'approved') return;
    const flushed = flushCanvasIntoDoc() ?? doc;
    const page = flushed.pages[activePageIndex];
    const nextLayers = page.layers.map((l) => (l.id === layerId ? { ...l, [prop]: !l[prop] } : l));
    const nextDoc = withUpdatedPage(flushed, activePageIndex, (p) => ({ ...p, layers: nextLayers }));
    commitDocChange(nextDoc, true, true);
    await renderActivePageToCanvas(nextDoc, activePageIndex);
  }

  async function moveLayerOrder(layerId: string, delta: 1 | -1) {
    if (!doc || !activePage || doc.status === 'approved') return;
    const flushed = flushCanvasIntoDoc() ?? doc;
    const sorted = [...flushed.pages[activePageIndex].layers].sort((a, b) => a.zIndex - b.zIndex);
    const idx = sorted.findIndex((l) => l.id === layerId);
    const targetIdx = idx + delta;
    if (idx < 0 || targetIdx < 0 || targetIdx >= sorted.length) return;
    const [item] = sorted.splice(idx, 1);
    sorted.splice(targetIdx, 0, item);
    const reindexed = sorted.map((l, i) => ({ ...l, zIndex: i + 1 }));
    const nextDoc = withUpdatedPage(flushed, activePageIndex, (p) => ({ ...p, layers: reindexed }));
    commitDocChange(nextDoc, true, true);
    await renderActivePageToCanvas(nextDoc, activePageIndex);
    selectLayerOnCanvas(layerId);
  }

  async function addQuickText() {
    if (!doc || !activePage || doc.status === 'approved') return;
    blurFocusedButton();
    const flushed = flushCanvasIntoDoc() ?? doc;
    const id = createShortId('text');
    const page = flushed.pages[activePageIndex];
    const newLayer: Layer = {
      id,
      name: 'Novo texto',
      kind: 'text',
      x: 60,
      y: 220,
      width: 320,
      height: 44,
      scaleX: 1,
      scaleY: 1,
      angle: 0,
      opacity: 1,
      visible: true,
      locked: false,
      zIndex: page.layers.length + 1,
      fill: '#374F1E',
      text: 'Texto editável',
      fontFamily: 'Sora',
      fontSize: 28,
      fontWeight: '700',
      lineHeight: 1.15,
      textAlign: 'left'
    };
    const nextDoc = withUpdatedPage(flushed, activePageIndex, (p) => ({ ...p, layers: [...p.layers, newLayer] }));
    selectedLayerId = id;
    commitDocChange(nextDoc, true, true);
    await renderActivePageToCanvas(nextDoc, activePageIndex);
    selectLayerOnCanvas(id);
    if (fabricCanvas) {
      const obj = fabricCanvas.getObjects().find((o) => getLayerId(o) === id) as
        | { enterEditing?: () => void; selectAll?: () => void }
        | undefined;
      if (obj && typeof obj.enterEditing === 'function') {
        obj.enterEditing();
        if (typeof obj.selectAll === 'function') {
          obj.selectAll();
        }
        fabricCanvas.requestRenderAll();
      }
    }
    statusMessage = 'Texto adicionado. Digite direto na arte ou use o painel direito.';
  }

  async function addQuickRect() {
    if (!doc || !activePage || doc.status === 'approved') return;
    blurFocusedButton();
    const flushed = flushCanvasIntoDoc() ?? doc;
    const id = createShortId('rect');
    const page = flushed.pages[activePageIndex];
    const newLayer: Layer = {
      id,
      name: 'Forma / destaque',
      kind: 'rect',
      x: 340,
      y: 220,
      width: 180,
      height: 54,
      scaleX: 1,
      scaleY: 1,
      angle: 0,
      opacity: 1,
      visible: true,
      locked: false,
      zIndex: page.layers.length + 1,
      fill: '#98B432',
      radius: 10
    };
    const nextDoc = withUpdatedPage(flushed, activePageIndex, (p) => ({ ...p, layers: [...p.layers, newLayer] }));
    selectedLayerId = id;
    commitDocChange(nextDoc, true, true);
    await renderActivePageToCanvas(nextDoc, activePageIndex);
    selectLayerOnCanvas(id);
  }

  async function applyImageAssetToCanvas(asset: LibraryAsset) {
    if (!doc || !activePage || doc.status === 'approved') return;
    const flushed = flushCanvasIntoDoc() ?? doc;

    if (imagePickerMode === 'replace' && selectedLayerId) {
      await handleUpdateLayer(
        {
          assetKey: asset.id,
          assetUrl: asset.assetUrl,
          objectFit: asset.classification === 'marca' ? 'contain' : 'cover'
        },
        true
      );
      statusMessage = `Imagem da camada alterada para "${asset.title}".`;
      return;
    }

    const page = flushed.pages[activePageIndex];
    const id = createShortId('img');
    const defaultW = asset.classification === 'marca' ? 180 : Math.min(480, Math.round(flushed.workspaceWidth * 0.78));
    const ratio = asset.width > 0 && asset.height > 0 ? asset.height / asset.width : 0.7;
    const defaultH = asset.classification === 'marca' ? 60 : Math.min(420, Math.max(90, Math.round(defaultW * ratio)));
    const newLayer: Layer = {
      id,
      name: asset.title || asset.fileName,
      kind: 'image',
      x: Math.round((flushed.workspaceWidth - defaultW) / 2),
      y: Math.round((flushed.workspaceHeight - defaultH) / 2),
      width: defaultW,
      height: defaultH,
      scaleX: 1,
      scaleY: 1,
      angle: 0,
      opacity: 1,
      visible: true,
      locked: false,
      zIndex: page.layers.length + 1,
      assetKey: asset.id,
      assetUrl: asset.assetUrl,
      objectFit: asset.classification === 'marca' ? 'contain' : 'cover',
      positionX: 0.5,
      positionY: 0.5
    };
    const nextDoc = withUpdatedPage(flushed, activePageIndex, (p) => ({ ...p, layers: [...p.layers, newLayer] }));
    selectedLayerId = id;
    commitDocChange(nextDoc, true, true);
    await renderActivePageToCanvas(nextDoc, activePageIndex);
    selectLayerOnCanvas(id);
    statusMessage = `Imagem "${asset.title}" adicionada à página.`;
  }

  async function handleUploadImageFile(file: File) {
    importingImage = true;
    errorMessage = '';
    try {
      const dataUrl = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result || ''));
        reader.onerror = () => reject(new Error('Falha ao ler arquivo selecionado'));
        reader.readAsDataURL(file);
      });
      const asset = await bridge.importEditorImage(file.name, dataUrl);
      await refreshLibrary();
      await applyImageAssetToCanvas(asset);
    } catch (err) {
      errorMessage = `Falha ao importar imagem: ${String(err)}`;
    } finally {
      importingImage = false;
    }
  }

  async function handleUpdatePageMeta(patch: Partial<Page>) {
    if (!doc || !activePage || doc.status === 'approved') return;
    const flushed = flushCanvasIntoDoc() ?? doc;
    const nextDoc = withUpdatedPage(flushed, activePageIndex, (p) => ({ ...p, ...patch }));
    commitDocChange(nextDoc, true, true);
    if ('background' in patch && Object.keys(patch).length === 1 && fabricCanvas && patch.background !== undefined) {
      updateCanvasBackgroundInPlace(fabricCanvas, patch.background);
      return;
    }
    await renderActivePageToCanvas(nextDoc, activePageIndex);
  }

  async function handleUpdateDocMeta(patch: Partial<DocumentModel>, recheckOffer = false) {
    if (!doc || doc.status === 'approved') return;
    const flushed = flushCanvasIntoDoc() ?? doc;
    const nextDoc: DocumentModel = { ...flushed, ...patch };
    commitDocChange(nextDoc, false, true);
    if (recheckOffer) {
      try {
        offerCheck = await bridge.checkOfferConsistency(nextDoc);
      } catch {
        // surfaced cleanly during autosave/approve
      }
    }
  }

  function handleAddAnnotation(kind: 'pin' | 'rect', x: number, y: number, w?: number, h?: number) {
    if (!doc || !activePage) return;
    const mark = createAnnotation(kind, x, y, w, h, selectedLayerId ?? '');
    const nextDoc = withUpdatedPage(doc, activePageIndex, (p) => ({
      ...p,
      annotations: [...(p.annotations ?? []), mark]
    }));
    selectedAnnotationId = mark.id;
    markingMode = 'off';
    showMarks = true;
    commitDocChange(nextDoc, false, nextDoc.status !== 'approved');
    agentOpen = true;
    agentFocusTrigger++;
    statusMessage = 'Marcação adicionada. Descreva no balão do Agente o que deseja ajustar.';
  }

  function handleAskAgentForLayer(layer: Layer) {
    if (!doc || !activePage) return;
    const w = Math.max(24, (layer.width || 120) * (layer.scaleX || 1));
    const h = Math.max(24, (layer.height || 40) * (layer.scaleY || 1));
    const mark = createAnnotation('rect', layer.x, layer.y, w, h, layer.id);
    mark.note = `Ajustar "${layer.name}": `;
    const nextDoc = withUpdatedPage(doc, activePageIndex, (p) => ({
      ...p,
      annotations: [...(p.annotations ?? []), mark]
    }));
    selectedAnnotationId = mark.id;
    showMarks = true;
    markingMode = 'off';
    commitDocChange(nextDoc, false, nextDoc.status !== 'approved');
    agentOpen = true;
    agentFocusTrigger++;
  }

  function handleUpdateAnnotation(id: string, patch: Partial<AgentAnnotation>) {
    if (!doc || !activePage) return;
    const nextDoc = withUpdatedPage(doc, activePageIndex, (p) => ({
      ...p,
      annotations: (p.annotations ?? []).map((a) => (a.id === id ? { ...a, ...patch } : a))
    }));
    commitDocChange(nextDoc, false, nextDoc.status !== 'approved');
  }

  function handleDeleteAnnotation(id: string) {
    if (!doc || !activePage) return;
    const nextDoc = withUpdatedPage(doc, activePageIndex, (p) => ({
      ...p,
      annotations: (p.annotations ?? []).filter((a) => a.id !== id)
    }));
    if (selectedAnnotationId === id) selectedAnnotationId = null;
    commitDocChange(nextDoc, false, nextDoc.status !== 'approved');
  }

  async function handleSubmitAgentTask() {
    if (!doc || !activePage || !fabricCanvas) return;
    errorMessage = '';
    try {
      clearAutosaveTimer();
      const payload = flushCanvasIntoDoc() ?? doc;
      if (payload.status !== 'approved') {
        await bridge.saveDraft(payload);
      }
      const snapDataUrl = await renderPageSnapshotWithAnnotationsDataUrl(
        fabricCanvas,
        payload.workspaceWidth,
        payload.workspaceHeight,
        activePage.annotations ?? []
      );
      const started = await bridge.startAgentTask({
        documentId: payload.id,
        pageIndex: activePageIndex,
        mode: agentDeliveryMode,
        prompt: agentPromptDraft,
        snapshotBase64: snapDataUrl,
        includeAnnotations: true
      });
      activeAgentTask = started;
      agentTasks = [started, ...agentTasks.filter((t) => t.id !== started.id)];
      const det = await bridge.getAgentTask(started.id);
      agentLogs = det.logs ?? [];
      agentOpen = true;
      if (started.status === 'running') {
        agentPromptDraft = '';
        statusMessage = 'Pedido enviado ao Codex CLI. Acompanhe o log ao vivo no balão do agente.';
      }
    } catch (err) {
      errorMessage = `Agente IA: ${String(err)}`;
    }
  }

  async function handleConfirmAgentTask(taskId: string) {
    errorMessage = '';
    try {
      const updated = await bridge.confirmAgentTask(taskId);
      activeAgentTask = updated;
      agentTasks = [updated, ...agentTasks.filter((t) => t.id !== updated.id)];
      agentPromptDraft = '';
      statusMessage = 'Execução no Codex CLI confirmada e iniciada.';
    } catch (err) {
      errorMessage = `Erro ao confirmar tarefa do agente: ${String(err)}`;
    }
  }

  async function handleCancelAgentTask(taskId: string) {
    errorMessage = '';
    try {
      await bridge.cancelAgentTask(taskId);
      const det = await bridge.getAgentTask(taskId);
      activeAgentTask = det.task;
      agentLogs = det.logs ?? [];
      agentTasks = agentTasks.map((t) => (t.id === taskId ? det.task : t));
      statusMessage = 'Cancelamento enviado ao agente.';
    } catch (err) {
      errorMessage = `Erro ao cancelar agente: ${String(err)}`;
    }
  }

  async function handleRecoverAgentArtifact(taskId: string) {
    errorMessage = '';
    try {
      const recovered = await bridge.recoverAgentArtifact(taskId);
      activeAgentTask = recovered;
      agentTasks = agentTasks.map((t) => (t.id === taskId ? recovered : t));
      if (recovered.resultDocumentId && recovered.resultDocumentId === doc?.id) {
        const reloaded = await bridge.getDocument(recovered.resultDocumentId);
        await loadDocumentIntoEditor(reloaded, recovered.pageIndex);
        await refreshLibrary();
        statusMessage = `Imagem recuperada e aplicada na peça (v${recovered.resultRevision}).`;
      }
    } catch (err) {
      errorMessage = `Erro ao recuperar imagem: ${String(err)}`;
    }
  }

  async function handleResumeAgentTask(taskId: string) {
    errorMessage = '';
    try {
      const resumed = await bridge.resumeAgentTask(taskId, agentPromptDraft);
      activeAgentTask = resumed;
      agentTasks = [resumed, ...agentTasks.filter((t) => t.id !== resumed.id)];
      const det = await bridge.getAgentTask(resumed.id);
      agentLogs = det.logs ?? [];
      if (resumed.status === 'completed' && resumed.resultDocumentId && resumed.resultDocumentId === doc?.id) {
        const reloaded = await bridge.getDocument(resumed.resultDocumentId);
        await loadDocumentIntoEditor(reloaded, resumed.pageIndex);
        await refreshLibrary();
      }
    } catch (err) {
      errorMessage = `Erro ao retomar sessão no Codex: ${String(err)}`;
    }
  }

  async function handleSelectAgentHistoryTask(taskId: string) {
    try {
      const det = await bridge.getAgentTask(taskId);
      activeAgentTask = det.task;
      agentLogs = det.logs ?? [];
    } catch (err) {
      errorMessage = `Erro ao abrir histórico do agente: ${String(err)}`;
    }
  }

  async function handleCreatePiece(templateKind: string, offerId: string, title: string) {
    busy = true;
    errorMessage = '';
    try {
      const created = await bridge.createPieceFromTemplate(templateKind, offerId, title);
      await loadDocumentIntoEditor(created, 0);
      await switchSection('editor');
      statusMessage = `Peça "${created.title}" criada e pronta para editar.`;
    } catch (err) {
      errorMessage = `Erro ao criar peça: ${String(err)}`;
    } finally {
      busy = false;
    }
  }

  async function handleOpenAssetInEditor(assetId: string, format: 'feed' | 'story') {
    busy = true;
    errorMessage = '';
    try {
      const created = await bridge.createPieceFromLibraryAsset(assetId, format);
      await loadDocumentIntoEditor(created, 0);
      await switchSection('editor');
      statusMessage = `Imagem aberta no editor como nova peça ("${created.title}").`;
    } catch (err) {
      errorMessage = `Erro ao abrir imagem no editor: ${String(err)}`;
    } finally {
      busy = false;
    }
  }

  async function handleOpenInEditor(documentId: string) {
    busy = true;
    errorMessage = '';
    try {
      const loaded = await bridge.getDocument(documentId);
      await loadDocumentIntoEditor(loaded, 0);
      await switchSection('editor');
      statusMessage = `Documento "${loaded.title}" aberto no editor.`;
    } catch (err) {
      errorMessage = `Erro ao abrir documento no editor: ${String(err)}`;
    } finally {
      busy = false;
    }
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
      flushCanvasIntoDoc();
      fabricCanvas.dispose();
      fabricCanvas = null;
    }
    activeSection = next;
    if (next === 'editor' && doc) {
      await renderActivePageToCanvas(doc, activePageIndex);
    } else if ((next === 'library' || next === 'settings') && !libraryOverview) {
      await refreshLibrary();
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
      statusMessage = `Perfil "${saved.brand}" salvo no SQLite.`;
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
      statusMessage = `Base reimportada: ${sum.uniqueArtsCount} artes únicas, ${sum.auxiliaryCoversCount} capas extras.`;
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
      statusMessage = `Integridade conferida: ${rep.okCount}/${rep.totalAssets} OK.`;
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
      statusMessage = `Backup criado: ${res.fileName} (${res.archivedFiles} arquivos).`;
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

  async function handleOpenExportPath(path: string) {
    errorMessage = '';
    try {
      await bridge.openExportPath(path);
      statusMessage = `Pasta de exportação aberta: ${path}`;
    } catch (err) {
      errorMessage = `Erro ao abrir exportação: ${String(err)}`;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    const target = e.target as HTMLElement | null;
    const tag = target?.tagName;
    const isInput = tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || Boolean(target?.isContentEditable);
    const activeObj = fabricCanvas?.getActiveObject() as { isEditing?: boolean } | null | undefined;
    const isCanvasEditing = Boolean(activeObj?.isEditing);
    if (activeSection !== 'editor') return;

    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
      e.preventDefault();
      void handleSaveVersion();
      return;
    }

    if (e.key === 'Escape' && markingMode !== 'off') {
      markingMode = 'off';
      return;
    }

    if (isInput || isCanvasEditing) return;

    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'z' && !e.shiftKey) {
      e.preventDefault();
      void handleUndo();
    } else if ((e.ctrlKey || e.metaKey) && (e.key.toLowerCase() === 'y' || (e.key.toLowerCase() === 'z' && e.shiftKey))) {
      e.preventDefault();
      void handleRedo();
    } else if ((e.key === 'Delete' || e.key === 'Backspace') && selectedLayerId && !selectedLayer?.locked) {
      e.preventDefault();
      void handleDeleteSelectedLayer();
    }
  }

  onMount(async () => {
    window.addEventListener('keydown', handleKeydown);

    unsubAgentLog = Events.On('agent:log', (ev) => {
      const entry = ev.data as AgentLogEntry | undefined;
      if (!entry || !entry.taskId) return;
      if (activeAgentTask?.id === entry.taskId) {
        if (!agentLogs.some((l) => l.id === entry.id)) {
          agentLogs = [...agentLogs, entry];
        }
      }
    });

    unsubAgentTask = Events.On('agent:task', (ev) => {
      const updated = ev.data as AgentTask | undefined;
      if (!updated || !updated.id) return;
      agentTasks = [updated, ...agentTasks.filter((t) => t.id !== updated.id)];
      if (activeAgentTask?.id === updated.id) {
        activeAgentTask = updated;
      }
      if (updated.status === 'completed' && updated.resultDocumentId && updated.resultDocumentId === doc?.id) {
        void (async () => {
          try {
            const reloaded = await bridge.getDocument(updated.resultDocumentId!);
            await loadDocumentIntoEditor(reloaded, updated.pageIndex);
            await refreshLibrary();
            statusMessage = `Agente aplicou nova versão v${updated.resultRevision} na peça.`;
          } catch (err) {
            errorMessage = `Erro ao recarregar peça atualizada pelo agente: ${String(err)}`;
          }
        })();
      }
    });

    try {
      const fontResult = await loadBrandFonts();
      fontsSummary = fontResult.loaded.length
        ? `Fontes locais: ${fontResult.loaded.join(', ')}`
        : 'Fontes locais indisponíveis (fallback do sistema)';

      workspaceInfo = await bridge.getWorkspaceInfo();
      const starters = await bridge.ensureStarterDocuments();
      documentsList = starters;
      const initialDoc =
        starters.find((d) => d.kind !== 'template') ??
        starters[0] ??
        (await bridge.ensureSeedDocument());
      await loadDocumentIntoEditor(initialDoc, 0);
      await Promise.all([refreshLibrary(), refreshAgentHealth()]);
      statusMessage = `Pronto · Editor multi-página e Agente Codex ativos (${initialDoc.exportWidth}×${initialDoc.exportHeight}px)`;
    } catch (err) {
      errorMessage = `Erro ao iniciar ponte Go/Wails: ${String(err)}`;
      statusMessage = 'Falha na inicialização';
    }
  });

  onDestroy(() => {
    clearAutosaveTimer();
    window.removeEventListener('keydown', handleKeydown);
    unsubAgentLog?.();
    unsubAgentTask?.();
    fabricCanvas?.dispose();
    fabricCanvas = null;
  });
</script>

<div class="app-shell" class:no-toolbar={activeSection !== 'editor'}>
  <header class="nav-bar">
    <div class="brand-title">
      <img src="/favicon.svg" alt="InstaGo" class="brand-logo" width="22" height="22" />
      <span>InstaGo</span>
      <span class="badge">Estúdio · R3</span>
    </div>

    <nav class="nav-tabs" aria-label="Áreas principais">
      <button type="button" class="nav-tab" class:active={activeSection === 'editor'} onclick={() => switchSection('editor')}>
        Editor
      </button>
      <button type="button" class="nav-tab" class:active={activeSection === 'library'} onclick={() => switchSection('library')}>
        Biblioteca
      </button>
      <button type="button" class="nav-tab" class:active={activeSection === 'settings'} onclick={() => switchSection('settings')}>
        Configurações
      </button>
    </nav>
  </header>

  {#if activeSection === 'editor'}
    <div class="toolbar">
      <div class="toolbar-group">
        <button onclick={() => (showNewPieceModal = true)} disabled={busy}>
          <Plus size={14} /> Nova / Peças
        </button>
        <button onclick={handleUndo} disabled={!canUndo || busy || doc?.status === 'approved'} title="Desfazer (Ctrl+Z)">
          <Undo2 size={14} />
        </button>
        <button onclick={handleRedo} disabled={!canRedo || busy || doc?.status === 'approved'} title="Refazer (Ctrl+Shift+Z)">
          <Redo2 size={14} />
        </button>
        <button onclick={addQuickText} disabled={!doc || busy || doc.status === 'approved'}>
          <Type size={14} /> + Texto
        </button>
        <button onclick={addQuickRect} disabled={!doc || busy || doc.status === 'approved'}>
          <Square size={14} /> + Forma
        </button>
        <button
          type="button"
          onclick={() => {
            blurFocusedButton();
            imagePickerMode = 'add';
            imagePickerOpen = true;
          }}
          disabled={!doc || busy || doc.status === 'approved'}
        >
          <ImageIcon size={14} /> + Imagem
        </button>
        <button
          type="button"
          class:primary={markingMode !== 'off'}
          onclick={() => {
            blurFocusedButton();
            markingMode = markingMode === 'off' ? 'rect' : 'off';
          }}
          disabled={!doc}
          title="Circule ou clique na área que quer pedir ao agente para alterar"
        >
          <MapPin size={14} /> {markingMode !== 'off' ? 'Finalizar marcação' : 'Apontar p/ agente'}
        </button>
        <label class="badge">
          Zoom:
          <select
            bind:value={stageScale}
            onchange={() => applyStageZoom()}
            aria-label="Zoom da prancheta"
            style="border:0;background:transparent;font-weight:600"
          >
            <option value={0.5}>50%</option>
            <option value={0.6}>60%</option>
            <option value={0.75}>75%</option>
            <option value={1}>100%</option>
          </select>
        </label>
      </div>

      <div class="toolbar-group">
        {#if doc?.referenceImageUrl}
          <button onclick={() => (showReference = !showReference)}>
            <Columns2 size={14} /> {showReference ? 'Ocultar Ref.' : 'Comparar Ref.'}
          </button>
        {/if}
        <button
          class="primary"
          onclick={() => handleSaveVersion()}
          disabled={!doc || busy || doc.status === 'approved'}
          title="Salvar nova versão numerada no histórico (Ctrl+S)"
        >
          <Save size={14} /> Salvar versão
        </button>
        <button onclick={() => handleApprove(false)} disabled={!doc || busy || doc.status === 'approved'}>
          <CheckCircle2 size={14} /> {doc?.status === 'approved' ? 'Aprovada' : 'Aprovar'}
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
                <button
                  class="icon-btn"
                  disabled={doc?.status === 'approved'}
                  onclick={() => moveLayerOrder(layer.id, 1)}
                  title="Trazer para frente"
                >
                  <ArrowUp size={12} />
                </button>
                <button
                  class="icon-btn"
                  disabled={doc?.status === 'approved'}
                  onclick={() => moveLayerOrder(layer.id, -1)}
                  title="Enviar para trás"
                >
                  <ArrowDown size={12} />
                </button>
                <button
                  class="icon-btn"
                  disabled={doc?.status === 'approved'}
                  onclick={() => toggleLayerProp(layer.id, 'locked')}
                  title="Bloquear/Desbloquear"
                >
                  {#if layer.locked}<Lock size={12} />{:else}<Unlock size={12} />{/if}
                </button>
                <button
                  class="icon-btn"
                  disabled={doc?.status === 'approved'}
                  onclick={() => toggleLayerProp(layer.id, 'visible')}
                  title="Mostrar/Ocultar"
                >
                  {#if layer.visible}<Eye size={12} />{:else}<EyeOff size={12} />{/if}
                </button>
              </div>
            {/each}
          </div>
        </div>

        <div class="panel-section">
          <h3>Peça aberta</h3>
          {#if doc}
            <label class="field">
              <span>Título</span>
              <input
                type="text"
                value={doc.title}
                disabled={doc.status === 'approved'}
                oninput={(e) => handleUpdateDocMeta({ title: (e.currentTarget as HTMLInputElement).value })}
              />
            </label>
            <div class="muted">
              Formato: <strong>{doc.format === 'story' ? 'Story' : doc.pages.length > 1 ? 'Carrossel' : 'Feed'}</strong> ({doc.exportWidth}×{doc.exportHeight})
            </div>
            <div class="muted">
              Status: <strong>{formatDocStatusLabel(doc.status)} · v{doc.revision}</strong>
            </div>
            <label class="row-gap" style="font-size:11px; margin-top:2px;">
              <input type="checkbox" bind:checked={showMarks} />
              Mostrar marcações do agente ({(activePage?.annotations ?? []).filter((a) => !a.resolved).length})
            </label>
          {/if}
        </div>
      </aside>

      <div class="stage-column">
        {#if pendingConflictApprove}
          <div class="conflict-banner">
            <div>
              <strong><AlertTriangle size={14} style="vertical-align:-2px;" /> Divergência com oferta vinculada:</strong>
              {pendingConflictApprove.messages.join(' ')}
            </div>
            <div class="row-gap">
              <button type="button" onclick={() => (pendingConflictApprove = null)}>Corrigir antes</button>
              <button type="button" class="primary" onclick={() => handleApprove(true)}>Confirmar e aprovar mesmo assim</button>
            </div>
          </div>
        {/if}

        {#if lastExportBundle}
          <div class="conflict-banner" style="background:#edf5e4; border-color:#9cc486; color:#233e18;">
            <div>
              <strong>Exportação concluída:</strong> {lastExportBundle.pageCount} página(s) + legenda.txt
              {#if lastExportBundle.isDraftWarning}(Aviso: exportada como RASCUNHO — ainda não aprovada){/if}
            </div>
            <div class="row-gap">
              <button type="button" onclick={() => lastExportBundle && handleOpenExportPath(lastExportBundle.bundleDir)}>
                <FolderOpen size={13} /> Abrir pasta
              </button>
              <button type="button" onclick={() => (lastExportBundle = null)}>Fechar</button>
            </div>
          </div>
        {/if}

        <section class="canvas-stage">
          {#if doc && activePage}
            <div class="artboard-frame" style="width:{Math.round(doc.workspaceWidth * stageScale) + 24}px">
              <div class="artboard-meta">
                <span title={activePage.title}>
                  <strong>Pág. {activePageIndex + 1}/{doc.pages.length} · {activePage.title}</strong>
                </span>
                <span>v{doc.revision} · {doc.exportWidth}×{doc.exportHeight}px</span>
              </div>
              <div class="canvas-wrap" role="presentation" onpointerdown={blurFocusedButton}>
                <canvas bind:this={canvasEl}></canvas>
                <AnnotationOverlay
                  annotations={(activePage.annotations ?? []).filter((a) => !a.resolved)}
                  workspaceWidth={doc.workspaceWidth}
                  workspaceHeight={doc.workspaceHeight}
                  {markingMode}
                  {showMarks}
                  {selectedAnnotationId}
                  onAddAnnotation={handleAddAnnotation}
                  onSelectAnnotation={(id) => {
                    selectedAnnotationId = id;
                    agentOpen = true;
                    agentFocusTrigger++;
                  }}
                />
              </div>
            </div>

            {#if showReference && doc.referenceImageUrl}
              <div class="artboard-frame">
                <div class="artboard-meta">
                  <span><strong>Referência original</strong></span>
                  <span>{doc.exportWidth}×{doc.exportHeight}</span>
                </div>
                <img class="reference-img" src={doc.referenceImageUrl} alt="Referência original" />
              </div>
            {/if}
          {:else}
            <div class="panel-section" style="max-width:420px;margin-top:40px">
              <h3>Carregando documento...</h3>
              <p style="margin:0">{errorMessage || statusMessage}</p>
            </div>
          {/if}
        </section>

        {#if doc}
          <PageStrip
            pages={doc.pages}
            {activePageIndex}
            disabled={busy || doc.status === 'approved'}
            onSelectPage={selectPage}
            onAddPage={handleAddPage}
            onDuplicatePage={handleDuplicatePage}
            onMovePage={handleMovePage}
            onDeletePage={handleDeletePage}
          />
        {/if}
      </div>

      {#if doc && activePage}
        <EditorRightPanel
          {doc}
          page={activePage}
          {selectedLayer}
          offers={libraryOverview?.offers ?? []}
          {offerCheck}
          {revisions}
          {exportsList}
          {bundlesList}
          {busy}
          onUpdateLayer={handleUpdateLayer}
          onAlignLayer={handleAlignLayer}
          onDeleteSelectedLayer={handleDeleteSelectedLayer}
          onOpenReplaceImage={() => {
            imagePickerMode = 'replace';
            imagePickerOpen = true;
          }}
          onUpdatePageMeta={handleUpdatePageMeta}
          onUpdateDocMeta={handleUpdateDocMeta}
          onSaveVersion={handleSaveVersion}
          onApprove={() => handleApprove(false)}
          onUnlockDraft={handleUnlockDraft}
          onOpenCompareVersions={() => (showCompareModal = true)}
          onRestoreRevision={handleRestoreRevision}
          onOpenExportPath={handleOpenExportPath}
          onAskAgentForLayer={handleAskAgentForLayer}
          onNotify={(msg) => (statusMessage = msg)}
        />
      {/if}
    </main>
  {:else if activeSection === 'library'}
    <LibraryView
      overview={libraryOverview}
      loading={libraryLoading}
      onRefresh={refreshLibrary}
      onOpenFolder={handleOpenFolder}
      onOpenInEditor={handleOpenInEditor}
      onOpenAssetInEditor={handleOpenAssetInEditor}
      onNotify={(msg) => (statusMessage = msg)}
    />
  {:else}
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
  {/if}

  <footer class="status-bar">
    <span class:error-text={Boolean(errorMessage)}>{errorMessage || statusMessage}</span>
    <span>{workspaceInfo ? `DB: ${workspaceInfo.dbPath}` : ''}</span>
  </footer>
</div>

<NewPieceModal
  open={showNewPieceModal}
  documents={documentsList}
  offers={libraryOverview?.offers ?? []}
  currentDocId={doc?.id ?? ''}
  onClose={() => (showNewPieceModal = false)}
  onSelectDocument={handleOpenInEditor}
  onCreateFromTemplate={handleCreatePiece}
/>

<ImagePickerModal
  open={imagePickerOpen}
  mode={imagePickerMode}
  assets={libraryOverview?.assets ?? []}
  importing={importingImage}
  onClose={() => (imagePickerOpen = false)}
  onSelectAsset={applyImageAssetToCanvas}
  onUploadFile={handleUploadImageFile}
/>

{#if doc}
  <CompareVersionsModal
    open={showCompareModal}
    documentId={doc.id}
    {revisions}
    {activePageIndex}
    canRestore={doc.status !== 'approved'}
    onClose={() => (showCompareModal = false)}
    onRestoreRevision={handleRestoreRevision}
  />
{/if}

{#if activeSection === 'editor' && doc && activePage}
  <AgentBotWidget
    open={agentOpen}
    health={agentHealth}
    healthLoading={agentHealthLoading}
    activeTask={activeAgentTask}
    recentTasks={agentTasks}
    logs={agentLogs}
    annotations={activePage.annotations ?? []}
    {selectedAnnotationId}
    {markingMode}
    promptDraft={agentPromptDraft}
    deliveryMode={agentDeliveryMode}
    focusTrigger={agentFocusTrigger}
    disabled={busy}
    onToggleOpen={() => {
      agentOpen = !agentOpen;
      if (agentOpen && !agentHealth) {
        void refreshAgentHealth();
      }
    }}
    onRefreshHealth={() => void refreshAgentHealth()}
    onSetMarkingMode={(m) => (markingMode = m)}
    onSelectAnnotation={(id) => {
      selectedAnnotationId = id;
      agentFocusTrigger++;
    }}
    onUpdateAnnotation={handleUpdateAnnotation}
    onDeleteAnnotation={handleDeleteAnnotation}
    onPromptChange={(v) => (agentPromptDraft = v)}
    onDeliveryModeChange={(m) => (agentDeliveryMode = m)}
    onSubmit={() => void handleSubmitAgentTask()}
    onConfirmTask={(id) => void handleConfirmAgentTask(id)}
    onCancelTask={(id) => void handleCancelAgentTask(id)}
    onRecoverArtifact={(id) => void handleRecoverAgentArtifact(id)}
    onResumeTask={(id) => void handleResumeAgentTask(id)}
    onSelectHistoryTask={(id) => void handleSelectAgentHistoryTask(id)}
  />
{/if}
