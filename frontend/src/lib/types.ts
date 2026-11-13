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

export type DocumentKind = 'piece' | 'template';
export type PieceType = 'feed_single' | 'carousel' | 'story';

export interface AgentAnnotation {
  id: string;
  kind: 'pin' | 'rect';
  x: number;
  y: number;
  width?: number;
  height?: number;
  layerId?: string;
  note: string;
  color?: string;
  resolved?: boolean;
  createdAt?: string;
}

export interface Page {
  id: string;
  index: number;
  title: string;
  background: string;
  layers: Layer[];
  annotations?: AgentAnnotation[];
}

export interface DocumentModel {
  id: string;
  schemaVersion: number;
  kind?: DocumentKind;
  pieceType?: PieceType;
  title: string;
  companyId: string;
  offerId?: string;
  offerRef: string;
  format: DocumentFormat;
  workspaceWidth: number;
  workspaceHeight: number;
  exportWidth: number;
  exportHeight: number;
  exportScale: number;
  referenceImageUrl?: string;
  revision: number;
  approvedRevision?: number;
  status: string;
  caption?: string;
  ctaText?: string;
  ctaDestination?: string;
  nativeInstruction?: string;
  publicationInstructions?: string;
  pages: Page[];
  canvasJson?: string;
  createdAt: string;
  updatedAt: string;
}

export interface RevisionSummary {
  documentId: string;
  revision: number;
  note: string;
  status: string;
  createdAt: string;
}

export interface RevisionRecord extends RevisionSummary {
  document: DocumentModel;
}

export interface OfferCheckResult {
  hasOffer: boolean;
  offerId?: string;
  offerTitle?: string;
  validUntil?: string;
  checkedAt?: string;
  status: 'none' | 'ok' | 'warning' | 'conflict';
  detectedPrices: string[];
  expectedPromoBrl?: string;
  expectedAfterBrl?: string;
  messages: string[];
}

export interface ApproveResult {
  approved: boolean;
  document?: DocumentModel;
  offerCheck: OfferCheckResult;
}

export interface ExportBundlePageInput {
  pageIndex: number;
  title: string;
  dataBase64: string;
}

export interface ExportBundleRecord {
  id: string;
  documentId: string;
  revision: number;
  documentStatus: string;
  format: string;
  pieceType: string;
  bundleDir: string;
  captionPath: string;
  manifestPath: string;
  pageCount: number;
  filePaths: string[];
  isDraftWarning: boolean;
  createdAt: string;
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

export type AgentTaskStatus =
  | 'pending'
  | 'running'
  | 'needs_confirmation'
  | 'completed'
  | 'failed'
  | 'canceled'
  | 'interrupted_uncertain';

export type AgentDeliveryMode = 'auto' | 'background_layer' | 'full_piece';

export interface CLIHealth {
  available: boolean;
  binaryPath: string;
  version: string;
  authenticated: boolean;
  authStatusText: string;
  errorMessage?: string;
}

export interface AgentTask {
  id: string;
  documentId: string;
  pageIndex: number;
  requestedMode: AgentDeliveryMode;
  resolvedMode: AgentDeliveryMode;
  prompt: string;
  confirmationReason?: string;
  status: AgentTaskStatus;
  threadId?: string;
  workDir?: string;
  outputImagePath?: string;
  outputAssetId?: string;
  outputAssetUrl?: string;
  artifactRecoverable?: boolean;
  instructionsVersion?: string;
  contextSummary?: string;
  resultDocumentId?: string;
  resultRevision?: number;
  summaryText?: string;
  errorCode?: string;
  errorMessage?: string;
  startedAt: string;
  updatedAt: string;
  finishedAt?: string;
}

export interface AgentLogEntry {
  id: string;
  taskId: string;
  seq: number;
  stream: string;
  eventType?: string;
  text: string;
  createdAt: string;
}

export interface AgentTaskDetail {
  task: AgentTask;
  logs: AgentLogEntry[];
}

export interface StartAgentTaskInput {
  documentId: string;
  pageIndex: number;
  mode: AgentDeliveryMode;
  prompt: string;
  snapshotBase64?: string;
  confirmBeforeRun?: boolean;
  includeAnnotations: boolean;
}
