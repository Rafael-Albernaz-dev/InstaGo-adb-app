import { Call } from '@wailsio/runtime';
import type {
  AgentTask,
  AgentTaskDetail,
  ApproveResult,
  BackupResult,
  CLIHealth,
  CompanyProfile,
  DocumentModel,
  ExportBundlePageInput,
  ExportBundleRecord,
  ExportRecord,
  ImportSummary,
  IntegrityReport,
  LibraryAsset,
  LibraryFilter,
  LibraryOverview,
  OfferCheckResult,
  RevisionRecord,
  RevisionSummary,
  StartAgentTaskInput,
  WorkspaceInfo
} from './types';

const EDITOR_SERVICE_FQN = 'github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/wailsapp.EditorBridge';
const LIBRARY_SERVICE_FQN = 'github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/wailsapp.LibraryBridge';
const AGENT_SERVICE_FQN = 'github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/wailsapp.AgentBridge';

export type Caller = <T>(method: string, ...args: unknown[]) => Promise<T>;

const defaultEditorCaller: Caller = <T>(method: string, ...args: unknown[]) =>
  Call.ByName(`${EDITOR_SERVICE_FQN}.${method}`, ...args) as Promise<T>;

const defaultLibraryCaller: Caller = <T>(method: string, ...args: unknown[]) =>
  Call.ByName(`${LIBRARY_SERVICE_FQN}.${method}`, ...args) as Promise<T>;

const defaultAgentCaller: Caller = <T>(method: string, ...args: unknown[]) =>
  Call.ByName(`${AGENT_SERVICE_FQN}.${method}`, ...args) as Promise<T>;

export interface EditorBridgeClient {
  getWorkspaceInfo(): Promise<WorkspaceInfo>;
  ensureSeedDocument(): Promise<DocumentModel>;
  ensureStarterDocuments(): Promise<DocumentModel[]>;
  listDocuments(): Promise<DocumentModel[]>;
  getDocument(id: string): Promise<DocumentModel>;
  createPieceFromTemplate(templateKind: string, offerId?: string, title?: string): Promise<DocumentModel>;
  createPieceFromLibraryAsset(assetId: string, format?: string, title?: string): Promise<DocumentModel>;
  importEditorImage(fileName: string, dataBase64: string): Promise<LibraryAsset>;
  saveDraft(doc: DocumentModel): Promise<DocumentModel>;
  saveDocument(doc: DocumentModel, note: string): Promise<DocumentModel>;
  resetSeedDocument(): Promise<DocumentModel>;
  listRevisions(documentId: string): Promise<RevisionSummary[]>;
  getRevision(documentId: string, revision: number): Promise<RevisionRecord>;
  restoreRevision(documentId: string, revision: number): Promise<DocumentModel>;
  startDraftFromApproved(documentId: string): Promise<DocumentModel>;
  checkOfferConsistency(doc: DocumentModel): Promise<OfferCheckResult>;
  approveDocument(doc: DocumentModel, confirmConflicts?: boolean): Promise<ApproveResult>;
  listExports(documentId: string): Promise<ExportRecord[]>;
  exportAsset(documentId: string, format: 'png' | 'jpg', dataBase64: string, fileName?: string): Promise<ExportRecord>;
  exportBundle(documentId: string, format: 'png' | 'jpg', pages: ExportBundlePageInput[]): Promise<ExportBundleRecord>;
  listExportBundles(documentId: string): Promise<ExportBundleRecord[]>;
  openExportPath(path: string): Promise<void>;
  getLibraryOverview(filter?: LibraryFilter): Promise<LibraryOverview>;
  importBase(sourceDir: string): Promise<ImportSummary>;
  updateProfile(profile: CompanyProfile): Promise<CompanyProfile>;
  verifyIntegrity(): Promise<IntegrityReport>;
  createBackup(): Promise<BackupResult>;
  openFolder(scope: 'library' | 'asset_dir' | 'exports' | 'backups' | 'creative_base', assetId?: string): Promise<string>;
  checkAgentHealth(): Promise<CLIHealth>;
  listAgentTasks(documentId?: string): Promise<AgentTask[]>;
  getAgentTask(taskId: string): Promise<AgentTaskDetail>;
  startAgentTask(req: StartAgentTaskInput): Promise<AgentTask>;
  confirmAgentTask(taskId: string): Promise<AgentTask>;
  cancelAgentTask(taskId: string): Promise<void>;
  recoverAgentArtifact(taskId: string): Promise<AgentTask>;
  resumeAgentTask(taskId: string, followUp?: string): Promise<AgentTask>;
}

export function createBridge(
  editorCaller: Caller = defaultEditorCaller,
  libraryCaller: Caller = defaultLibraryCaller,
  agentCaller: Caller = defaultAgentCaller
): EditorBridgeClient {
  return {
    getWorkspaceInfo: () => editorCaller<WorkspaceInfo>('GetWorkspaceInfo'),
    ensureSeedDocument: () => editorCaller<DocumentModel>('EnsureSeedDocument'),
    ensureStarterDocuments: () => editorCaller<DocumentModel[]>('EnsureStarterDocuments'),
    listDocuments: () => editorCaller<DocumentModel[]>('ListDocuments'),
    getDocument: (id: string) => editorCaller<DocumentModel>('GetDocument', id),
    createPieceFromTemplate: (templateKind: string, offerId = '', title = '') =>
      editorCaller<DocumentModel>('CreatePieceFromTemplate', templateKind, offerId, title),
    createPieceFromLibraryAsset: (assetId: string, format = '', title = '') =>
      editorCaller<DocumentModel>('CreatePieceFromLibraryAsset', assetId, format, title),
    importEditorImage: (fileName: string, dataBase64: string) =>
      editorCaller<LibraryAsset>('ImportEditorImage', fileName, dataBase64),
    saveDraft: (doc: DocumentModel) => editorCaller<DocumentModel>('SaveDraft', doc),
    saveDocument: (doc: DocumentModel, note: string) => editorCaller<DocumentModel>('SaveDocument', doc, note),
    resetSeedDocument: () => editorCaller<DocumentModel>('ResetSeedDocument'),
    listRevisions: (documentId: string) => editorCaller<RevisionSummary[]>('ListRevisions', documentId),
    getRevision: (documentId: string, revision: number) =>
      editorCaller<RevisionRecord>('GetRevision', documentId, revision),
    restoreRevision: (documentId: string, revision: number) =>
      editorCaller<DocumentModel>('RestoreRevision', documentId, revision),
    startDraftFromApproved: (documentId: string) =>
      editorCaller<DocumentModel>('StartDraftFromApproved', documentId),
    checkOfferConsistency: (doc: DocumentModel) => editorCaller<OfferCheckResult>('CheckOfferConsistency', doc),
    approveDocument: (doc: DocumentModel, confirmConflicts = false) =>
      editorCaller<ApproveResult>('ApproveDocument', doc, confirmConflicts),
    listExports: (documentId: string) => editorCaller<ExportRecord[]>('ListExports', documentId),
    exportAsset: (documentId: string, format: 'png' | 'jpg', dataBase64: string, fileName = '') =>
      editorCaller<ExportRecord>('ExportAsset', {
        documentId,
        format,
        dataBase64,
        fileName
      }),
    exportBundle: (documentId: string, format: 'png' | 'jpg', pages: ExportBundlePageInput[]) =>
      editorCaller<ExportBundleRecord>('ExportBundle', {
        documentId,
        format,
        pages
      }),
    listExportBundles: (documentId: string) =>
      editorCaller<ExportBundleRecord[]>('ListExportBundles', documentId),
    openExportPath: (path: string) => editorCaller<void>('OpenExportPath', path),
    getLibraryOverview: (filter: LibraryFilter = {}) =>
      libraryCaller<LibraryOverview>('GetLibraryOverview', {
        query: filter.query ?? '',
        classification: filter.classification ?? '',
        format: filter.format ?? '',
        subCategory: filter.subCategory ?? ''
      }),
    importBase: (sourceDir: string) => libraryCaller<ImportSummary>('ImportBase', sourceDir),
    updateProfile: (profile: CompanyProfile) => libraryCaller<CompanyProfile>('UpdateProfile', profile),
    verifyIntegrity: () => libraryCaller<IntegrityReport>('VerifyIntegrity'),
    createBackup: () => libraryCaller<BackupResult>('CreateBackup'),
    openFolder: (scope, assetId = '') => libraryCaller<string>('OpenFolder', scope, assetId),
    checkAgentHealth: () => agentCaller<CLIHealth>('CheckAgentHealth'),
    listAgentTasks: (documentId = '') => agentCaller<AgentTask[]>('ListAgentTasks', documentId),
    getAgentTask: (taskId: string) => agentCaller<AgentTaskDetail>('GetAgentTask', taskId),
    startAgentTask: (req: StartAgentTaskInput) => agentCaller<AgentTask>('StartAgentTask', req),
    confirmAgentTask: (taskId: string) => agentCaller<AgentTask>('ConfirmAgentTask', taskId),
    cancelAgentTask: (taskId: string) => agentCaller<void>('CancelAgentTask', taskId),
    recoverAgentArtifact: (taskId: string) => agentCaller<AgentTask>('RecoverAgentArtifact', taskId),
    resumeAgentTask: (taskId: string, followUp = '') => agentCaller<AgentTask>('ResumeAgentTask', taskId, followUp)
  };
}

export const bridge = createBridge();
