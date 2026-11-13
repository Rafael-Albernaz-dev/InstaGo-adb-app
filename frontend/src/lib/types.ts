export type DocumentFormat = 'feed' | 'story';

export type LayerKind = 'rect' | 'text' | 'image';

export interface Layer {
  id: string;
  name: string;
  kind: LayerKind;
  x: number;
  y: number;
  width: number;
  height: number;
  scaleX: number;
  scaleY: number;
  angle: number;
  opacity: number;
  visible: boolean;
  locked: boolean;
  zIndex: number;
  fill?: string;
  stroke?: string;
  strokeWidth?: number;
  radius?: number;
  text?: string;
  fontFamily?: string;
  fontSize?: number;
  fontWeight?: string;
  lineHeight?: number;
  letterSpacing?: number;
  textAlign?: string;
  assetKey?: string;
  assetUrl?: string;
  objectFit?: 'cover' | 'contain' | string;
  positionX?: number;
  positionY?: number;
}

export interface Page {
  id: string;
  index: number;
  title: string;
  background: string;
  layers: Layer[];
}

export interface DocumentModel {
  id: string;
  schemaVersion: number;
  title: string;
  companyId: string;
  offerRef: string;
  format: DocumentFormat;
  workspaceWidth: number;
  workspaceHeight: number;
  exportWidth: number;
  exportHeight: number;
  exportScale: number;
  referenceImageUrl?: string;
  revision: number;
  status: string;
  pages: Page[];
  canvasJson?: string;
  createdAt: string;
  updatedAt: string;
}

export interface WorkspaceInfo {
  appName: string;
  appVersion: string;
  goVersion: string;
  os: string;
  arch: string;
  creativeBaseDir: string;
  creativeBaseOk: boolean;
  dbPath: string;
  exportsDir: string;
  heapAllocMb: number;
  processRssMb: number;
  numGoroutine: number;
}

export interface ExportRecord {
  id: string;
  documentId: string;
  revision: number;
  format: string;
  filePath: string;
  sha256: string;
  width: number;
  height: number;
  bytes: number;
  createdAt: string;
}
