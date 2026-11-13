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

export type AssetClassification =
  | 'arte_unica'
  | 'capa_auxiliar'
  | 'cena'
  | 'marca'
  | 'capa_editorial'
  | 'fonte'
  | 'historico'
  | 'referencia';

export interface CompanyProfile {
  id: string;
  brand: string;
  profileName: string;
  instagramHandle: string;
  instagramUrl: string;
  siteUrl: string;
  camerasUrl: string;
  phoneDisplay: string;
  phoneE164: string;
  whatsappLink: string;
  region: string;
  clientAudience: string;
  approvedVisualVersion: string;
  approvalDate: string;
  creativeBaseDir: string;
  paletteJson: string;
  bioText: string;
  customEdited: boolean;
  updatedAt: string;
}

export interface LibraryAsset {
  id: string;
  companyId: string;
  relPath: string;
  thumbRelPath: string;
  sourceRelPath: string;
  fileName: string;
  mimeType: string;
  mediaKind: 'image' | 'font' | 'document' | string;
  classification: AssetClassification | string;
  subCategory: string;
  format: 'feed' | 'story' | 'cover' | 'square' | 'other' | string;
  title: string;
  caption: string;
  pieceId: string;
  pageNumber: number;
  width: number;
  height: number;
  bytes: number;
  sha256: string;
  isUniqueApproved: boolean;
  isEditableModel: boolean;
  linkedDocumentId: string;
  metadataJson: string;
  importedAt: string;
  assetUrl: string;
  thumbUrl: string;
}

export interface DatedOffer {
  id: string;
  companyId: string;
  category: string;
  title: string;
  promoPriceCents: number;
  promoPeriod: string;
  afterPriceCents: number;
  installFeeCents: number;
  speedSummary: string;
  servicesSummary: string;
  restrictions: string;
  campaignName: string;
  validUntil: string;
  checkedAt: string;
  sourceRef: string;
  readOnly: boolean;
}

export interface PublicationHistoryItem {
  id: string;
  companyId: string;
  kind: 'feed_post' | 'story_highlight' | string;
  pieceKey: string;
  title: string;
  position: number;
  pinned: boolean;
  status: string;
  instagramUrl: string;
  caption: string;
  pageCount: number;
  assetIdsJson: string;
  notes: string;
  observedAt: string;
  sourceRef: string;
  readOnly: boolean;
}

export interface ImportSummary {
  runId: string;
  companyId: string;
  sourceDir: string;
  uniqueArtsCount: number;
  auxiliaryCoversCount: number;
  totalAssetsCount: number;
  offersCount: number;
  historyCount: number;
  editableModelsCount: number;
  referenceOnlyCount: number;
  fontsCount: number;
  skillLoaded: boolean;
  status: string;
  createdAt: string;
}

export interface LibraryFilter {
  query?: string;
  classification?: string;
  format?: string;
  subCategory?: string;
}

export interface LibraryOverview {
  profile: CompanyProfile;
  summary: ImportSummary | null;
  assets: LibraryAsset[];
  offers: DatedOffer[];
  history: PublicationHistoryItem[];
  countsByClass: Record<string, number>;
  libraryDir: string;
  backupsDir: string;
}

export interface IntegrityIssue {
  assetId: string;
  title: string;
  relPath: string;
  expectedSha256: string;
  actualSha256: string;
  reason: string;
}

export interface IntegrityReport {
  checkedAt: string;
  totalAssets: number;
  okCount: number;
  missingCount: number;
  corruptedCount: number;
  issues: IntegrityIssue[];
}

export interface BackupResult {
  filePath: string;
  fileName: string;
  bytes: number;
  sha256: string;
  archivedFiles: number;
  createdAt: string;
}
