import { Call } from '@wailsio/runtime';
import type {
  BackupResult,
  CompanyProfile,
  DocumentModel,
  ExportRecord,
  ImportSummary,
  IntegrityReport,
  LibraryFilter,
  LibraryOverview,
  WorkspaceInfo
} from './types';

const EDITOR_SERVICE_FQN = 'github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/wailsapp.EditorBridge';
const LIBRARY_SERVICE_FQN = 'github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/wailsapp.LibraryBridge';

export type Caller = <T>(method: string, ...args: unknown[]) => Promise<T>;

const defaultEditorCaller: Caller = <T>(method: string, ...args: unknown[]) =>
  Call.ByName(`${EDITOR_SERVICE_FQN}.${method}`, ...args) as Promise<T>;

const defaultLibraryCaller: Caller = <T>(method: string, ...args: unknown[]) =>
  Call.ByName(`${LIBRARY_SERVICE_FQN}.${method}`, ...args) as Promise<T>;

export interface EditorBridgeClient {
  getWorkspaceInfo(): Promise<WorkspaceInfo>;
  ensureSeedDocument(): Promise<DocumentModel>;
  getDocument(id: string): Promise<DocumentModel>;
  saveDocument(doc: DocumentModel, note: string): Promise<DocumentModel>;
  resetSeedDocument(): Promise<DocumentModel>;
  listExports(documentId: string): Promise<ExportRecord[]>;
  exportAsset(documentId: string, format: 'png' | 'jpg', dataBase64: string, fileName?: string): Promise<ExportRecord>;
  getLibraryOverview(filter?: LibraryFilter): Promise<LibraryOverview>;
  importBase(sourceDir: string): Promise<ImportSummary>;
  updateProfile(profile: CompanyProfile): Promise<CompanyProfile>;
  verifyIntegrity(): Promise<IntegrityReport>;
  createBackup(): Promise<BackupResult>;
  openFolder(scope: 'library' | 'asset_dir' | 'exports' | 'backups' | 'creative_base', assetId?: string): Promise<string>;
}

export function createBridge(
  editorCaller: Caller = defaultEditorCaller,
  libraryCaller: Caller = defaultLibraryCaller
): EditorBridgeClient {
  return {
    getWorkspaceInfo: () => editorCaller<WorkspaceInfo>('GetWorkspaceInfo'),
    ensureSeedDocument: () => editorCaller<DocumentModel>('EnsureSeedDocument'),
    getDocument: (id: string) => editorCaller<DocumentModel>('GetDocument', id),
    saveDocument: (doc: DocumentModel, note: string) => editorCaller<DocumentModel>('SaveDocument', doc, note),
    resetSeedDocument: () => editorCaller<DocumentModel>('ResetSeedDocument'),
    listExports: (documentId: string) => editorCaller<ExportRecord[]>('ListExports', documentId),
    exportAsset: (documentId: string, format: 'png' | 'jpg', dataBase64: string, fileName = '') =>
      editorCaller<ExportRecord>('ExportAsset', {
        documentId,
        format,
        dataBase64,
        fileName
      }),
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
    openFolder: (scope, assetId = '') => libraryCaller<string>('OpenFolder', scope, assetId)
  };
}

export const bridge = createBridge();
