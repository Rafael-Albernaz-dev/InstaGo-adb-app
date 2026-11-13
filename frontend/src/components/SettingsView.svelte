<script lang="ts">
  import {
    Archive,
    CheckCircle2,
    FolderOpen,
    RefreshCw,
    Save,
    ShieldCheck,
    TriangleAlert
  } from '@lucide/svelte';
  import { formatBytes } from '../lib/libraryHelpers';
  import type { BackupResult, CompanyProfile, IntegrityReport, LibraryOverview, WorkspaceInfo } from '../lib/types';

  interface Props {
    overview: LibraryOverview | null;
    workspaceInfo: WorkspaceInfo | null;
    fontsSummary: string;
    busy: boolean;
    onSaveProfile: (profile: CompanyProfile) => Promise<void>;
    onReimport: (sourceDir: string) => Promise<void>;
    onVerifyIntegrity: () => Promise<IntegrityReport | null>;
    onCreateBackup: () => Promise<BackupResult | null>;
    onOpenFolder: (scope: 'library' | 'backups' | 'exports' | 'creative_base') => Promise<void>;
  }

  let {
    overview,
    workspaceInfo,
    fontsSummary,
    busy,
    onSaveProfile,
    onReimport,
    onVerifyIntegrity,
    onCreateBackup,
    onOpenFolder
  }: Props = $props();

  let draft = $state<CompanyProfile | null>(null);
  let integrityReport = $state<IntegrityReport | null>(null);
  let lastBackup = $state<BackupResult | null>(null);

  $effect(() => {
    if (overview?.profile) {
      draft = { ...overview.profile };
    }
  });

  async function handleSave() {
    if (!draft) return;
    await onSaveProfile(draft);
  }

  async function handleReimport() {
    if (!draft) return;
    await onReimport(draft.creativeBaseDir);
  }

  async function handleVerify() {
    const rep = await onVerifyIntegrity();
    if (rep) integrityReport = rep;
  }

  async function handleBackup() {
    const res = await onCreateBackup();
    if (res) lastBackup = res;
  }
</script>

<section class="settings-shell">
  {#if draft}
    <div class="settings-grid">
      <div class="panel-section">
        <h3>Perfil da empresa ({draft.brand})</h3>
        <p class="muted-small" style="margin:0">
          Identificador interno: <span class="mono">{draft.id}</span> · Versão visual aprovada:
          <strong>{draft.approvedVisualVersion || 'V5'}</strong> ({draft.approvalDate || '2026-11-11'})
        </p>

        <div class="grid-2">
          <div class="field">
            <label for="prof-brand">Marca</label>
            <input id="prof-brand" bind:value={draft.brand} />
          </div>
          <div class="field">
            <label for="prof-handle">Instagram (@)</label>
            <input id="prof-handle" bind:value={draft.instagramHandle} />
          </div>
        </div>

        <div class="field">
          <label for="prof-name">Nome exibido no perfil</label>
          <input id="prof-name" bind:value={draft.profileName} />
        </div>

        <div class="grid-2">
          <div class="field">
            <label for="prof-phone">Telefone / WhatsApp</label>
            <input id="prof-phone" bind:value={draft.phoneDisplay} />
          </div>
          <div class="field">
            <label for="prof-site">Site oficial</label>
            <input id="prof-site" bind:value={draft.siteUrl} />
          </div>
        </div>

        <div class="field">
          <label for="prof-region">Região de atendimento</label>
          <input id="prof-region" bind:value={draft.region} />
        </div>

        <div class="field">
          <label for="prof-bio">Bio do Instagram</label>
          <textarea id="prof-bio" rows={4} bind:value={draft.bioText}></textarea>
        </div>

        <div class="field">
          <label for="prof-base">Pasta da base criativa (origem somente leitura)</label>
          <input id="prof-base" bind:value={draft.creativeBaseDir} />
        </div>

        <div class="settings-actions">
          <button class="primary" onclick={handleSave} disabled={busy}>
            <Save size={14} />
            Salvar perfil
          </button>
          <button onclick={handleReimport} disabled={busy}>
            <RefreshCw size={14} />
            Reimportar base
          </button>
          <button onclick={() => onOpenFolder('creative_base')} disabled={busy}>
            <FolderOpen size={14} />
            Abrir pasta de origem
          </button>
        </div>
      </div>

      <div class="settings-side">
        <div class="panel-section">
          <h3>Status da base, fontes e regras</h3>
          <div class="kv-list">
            <div><span>Fontes locais</span><strong>{fontsSummary}</strong></div>
            <div>
              <span>Skill Rural Conecta</span>
              <strong>{overview?.summary?.skillLoaded ? 'Carregada' : 'Não localizada'}</strong>
            </div>
            <div>
              <span>Artes únicas importadas</span>
              <strong>{overview?.summary?.uniqueArtsCount ?? 0}</strong>
            </div>
            <div>
              <span>Capas auxiliares separadas</span>
              <strong>{overview?.summary?.auxiliaryCoversCount ?? 0}</strong>
            </div>
            <div>
              <span>V5 original</span>
              <strong>Preservada intacta (somente referência)</strong>
            </div>
            <div>
              <span>Biblioteca gerenciada</span>
              <span class="mono">{overview?.libraryDir ?? ''}</span>
            </div>
          </div>

          <div class="settings-actions">
            <button onclick={handleVerify} disabled={busy}>
              <ShieldCheck size={14} />
              Verificar integridade (SHA-256)
            </button>
            <button onclick={() => onOpenFolder('library')} disabled={busy}>
              <FolderOpen size={14} />
              Abrir biblioteca
            </button>
          </div>

          {#if integrityReport}
            <div class="report-box" class:report-error={integrityReport.missingCount > 0 || integrityReport.corruptedCount > 0}>
              {#if integrityReport.missingCount === 0 && integrityReport.corruptedCount === 0}
                <CheckCircle2 size={16} />
                <span>
                  Integridade confirmada: <strong>{integrityReport.okCount}/{integrityReport.totalAssets}</strong> arquivos conferidos sem erros.
                </span>
              {:else}
                <TriangleAlert size={16} />
                <span>
                  Atenção: {integrityReport.okCount} OK · {integrityReport.missingCount} ausentes · {integrityReport.corruptedCount} divergentes.
                </span>
              {/if}
            </div>
          {/if}
        </div>

        <div class="panel-section">
          <h3>Cópia de segurança portátil (.zip)</h3>
          <p class="muted-small" style="margin:0">
            Gera um arquivo único contendo cópia do banco SQLite, toda a biblioteca gerenciada, exportações e manifesto de conferência SHA-256.
            A restauração completa em outra pasta será habilitada na entrega final do MVP (R5).
          </p>

          <div class="settings-actions">
            <button class="accent" onclick={handleBackup} disabled={busy}>
              <Archive size={14} />
              Gerar backup (.zip)
            </button>
            <button onclick={() => onOpenFolder('backups')} disabled={busy}>
              <FolderOpen size={14} />
              Abrir pasta de backups
            </button>
          </div>

          {#if lastBackup}
            <div class="report-box">
              <CheckCircle2 size={16} />
              <div>
                <div><strong>{lastBackup.fileName}</strong> ({formatBytes(lastBackup.bytes)} · {lastBackup.archivedFiles} arquivos)</div>
                <div class="mono">{lastBackup.filePath}</div>
                <div class="mono">sha256: {lastBackup.sha256}</div>
              </div>
            </div>
          {/if}

          {#if workspaceInfo}
            <div class="muted-small">
              DB: <span class="mono">{workspaceInfo.dbPath}</span>
            </div>
          {/if}
        </div>
      </div>
    </div>
  {:else}
    <div class="panel-section" style="max-width:520px">
      <h3>Carregando configurações do perfil...</h3>
    </div>
  {/if}
</section>
