import { Call } from '@wailsio/runtime';
import type { DocumentModel, ExportRecord, WorkspaceInfo } from './types';

const SERVICE_FQN = 'github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/wailsapp.EditorBridge';

export type Caller = <T>(method: string, ...args: unknown[]) => Promise<T>;

const defaultCaller: Caller = <T>(method: string, ...args: unknown[]) =>
  Call.ByName(`${SERVICE_FQN}.${method}`, ...args) as Promise<T>;

export interface EditorBridgeClient {
  getWorkspaceInfo(): Promise<WorkspaceInfo>;
  ensureSeedDocument(): Promise<DocumentModel>;
  getDocument(id: string): Promise<DocumentModel>;
  saveDocument(doc: DocumentModel, note: string): Promise<DocumentModel>;
  resetSeedDocument(): Promise<DocumentModel>;
  listExports(documentId: string): Promise<ExportRecord[]>;
  exportAsset(documentId: string, format: 'png' | 'jpg', dataBase64: string, fileName?: string): Promise<ExportRecord>;
}

export function createBridge(caller: Caller = defaultCaller): EditorBridgeClient {
  return {
    getWorkspaceInfo: () => caller<WorkspaceInfo>('GetWorkspaceInfo'),
    ensureSeedDocument: () => caller<DocumentModel>('EnsureSeedDocument'),
    getDocument: (id: string) => caller<DocumentModel>('GetDocument', id),
    saveDocument: (doc: DocumentModel, note: string) => caller<DocumentModel>('SaveDocument', doc, note),
    resetSeedDocument: () => caller<DocumentModel>('ResetSeedDocument'),
    listExports: (documentId: string) => caller<ExportRecord[]>('ListExports', documentId),
    exportAsset: (documentId: string, format: 'png' | 'jpg', dataBase64: string, fileName = '') =>
      caller<ExportRecord>('ExportAsset', {
        documentId,
        format,
        dataBase64,
        fileName
      })
  };
}

export const bridge = createBridge();
