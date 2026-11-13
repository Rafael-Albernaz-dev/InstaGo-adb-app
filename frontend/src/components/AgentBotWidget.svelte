<script lang="ts">
  import { tick } from 'svelte';
  import {
    AlertTriangle,
    Bot,
    CheckCircle2,
    MapPin,
    Minimize2,
    RefreshCw,
    RotateCcw,
    Send,
    Square,
    Trash2
  } from '@lucide/svelte';
  import type {
    AgentAnnotation,
    AgentDeliveryMode,
    AgentLogEntry,
    AgentTask,
    CLIHealth
  } from '../lib/types';

  interface Props {
    open: boolean;
    health: CLIHealth | null;
    healthLoading: boolean;
    activeTask: AgentTask | null;
    recentTasks: AgentTask[];
    logs: AgentLogEntry[];
    annotations: AgentAnnotation[];
    selectedAnnotationId: string | null;
    markingMode: 'off' | 'pin' | 'rect';
    promptDraft: string;
    deliveryMode: AgentDeliveryMode;
    focusTrigger: number;
    disabled?: boolean;
    onToggleOpen: () => void;
    onRefreshHealth: () => void;
    onSetMarkingMode: (mode: 'off' | 'pin' | 'rect') => void;
    onSelectAnnotation: (id: string) => void;
    onUpdateAnnotation: (id: string, patch: Partial<AgentAnnotation>) => void;
    onDeleteAnnotation: (id: string) => void;
    onPromptChange: (val: string) => void;
    onDeliveryModeChange: (mode: AgentDeliveryMode) => void;
    onSubmit: () => void;
    onConfirmTask: (taskId: string) => void;
    onCancelTask: (taskId: string) => void;
    onRecoverArtifact: (taskId: string) => void;
    onResumeTask: (taskId: string) => void;
    onSelectHistoryTask: (taskId: string) => void;
  }

  let {
    open,
    health,
    healthLoading,
    activeTask,
    recentTasks,
    logs,
    annotations,
    selectedAnnotationId,
    markingMode,
    promptDraft,
    deliveryMode,
    focusTrigger,
    disabled = false,
    onToggleOpen,
    onRefreshHealth,
    onSetMarkingMode,
    onSelectAnnotation,
    onUpdateAnnotation,
    onDeleteAnnotation,
    onPromptChange,
    onDeliveryModeChange,
    onSubmit,
    onConfirmTask,
    onCancelTask,
    onRecoverArtifact,
    onResumeTask,
    onSelectHistoryTask
  }: Props = $props();

  let terminalEl = $state<HTMLPreElement | null>(null);
  let promptEl = $state<HTMLTextAreaElement | null>(null);

  const unresolvedMarks = $derived(annotations.filter((a) => !a.resolved));
  const isRunning = $derived(activeTask?.status === 'running' || activeTask?.status === 'pending');

  $effect(() => {
    // Scroll live terminal output cleanly to latest line whenever logs grow.
    void logs.length;
    if (open && terminalEl) {
      void tick().then(() => {
        if (terminalEl) {
          terminalEl.scrollTop = terminalEl.scrollHeight;
        }
      });
    }
  });

  $effect(() => {
    if (open && focusTrigger > 0) {
      void tick().then(() => {
        if (selectedAnnotationId) {
          const markInput = document.getElementById(`mark-note-${selectedAnnotationId}`) as HTMLTextAreaElement | null;
          if (markInput) {
            markInput.focus();
            return;
          }
        }
        promptEl?.focus();
      });
    }
  });

  function formatStatusBadge(status?: string): string {
    switch (status) {
      case 'running':
      case 'pending':
        return 'Executando no Codex...';
      case 'needs_confirmation':
        return 'Aguardando confirmação';
      case 'completed':
        return 'Concluído e aplicado';
      case 'canceled':
        return 'Cancelado';
      case 'interrupted_uncertain':
        return 'Interrompido';
      case 'failed':
        return 'Falhou';
      default:
        return '';
    }
  }
</script>

<div class="agent-floating-root">
  {#if open}
    <section class="agent-drawer" aria-label="Assistente Codex IA">
      <header class="agent-drawer-head">
        <div class="row-gap">
          <Bot size={16} />
          <strong>Agente Codex</strong>
          {#if healthLoading}
            <span class="badge">Verificando...</span>
          {:else if health?.available && health?.authenticated}
            <span class="agent-chip ok" title={health.authStatusText}>ChatGPT ativo</span>
          {:else}
            <span class="agent-chip err" title={health?.errorMessage || 'Codex indisponível'}>Indisponível</span>
          {/if}
        </div>
        <div class="row-gap">
          <button type="button" class="icon-btn" onclick={onRefreshHealth} title="Verificar status do Codex CLI">
            <RefreshCw size={13} />
          </button>
          <button type="button" class="icon-btn" onclick={onToggleOpen} title="Minimizar balão do agente">
            <Minimize2 size={14} />
          </button>
        </div>
      </header>

      {#if health && (!health.available || !health.authenticated)}
        <p class="error-banner" style="font-size:11px;">
          {health.errorMessage || 'Codex CLI não encontrado ou sem login ChatGPT ativo.'}
        </p>
      {/if}

      <div class="agent-toolbar-row">
        <button
          type="button"
          class:primary={markingMode === 'rect'}
          onclick={() => onSetMarkingMode(markingMode === 'rect' ? 'off' : 'rect')}
          disabled={isRunning || disabled}
          title="Circule direto na arte a área que o agente deve mudar"
        >
          <MapPin size={13} />
          {markingMode === 'rect' ? 'Desenhando área...' : 'Circular área na arte'}
        </button>
        <button
          type="button"
          class:primary={markingMode === 'pin'}
          onclick={() => onSetMarkingMode(markingMode === 'pin' ? 'off' : 'pin')}
          disabled={isRunning || disabled}
          title="Clique em um ponto exato da arte"
        >
          {markingMode === 'pin' ? 'Clicando ponto...' : 'Marcar ponto'}
        </button>
      </div>

      {#if unresolvedMarks.length > 0}
        <div class="agent-marks-list">
          {#each unresolvedMarks as mark, idx (mark.id)}
            <div
              class="agent-mark-card"
              class:selected={selectedAnnotationId === mark.id}
              onclick={() => onSelectAnnotation(mark.id)}
              role="presentation"
            >
              <div class="row-between">
                <strong>Marcação #{idx + 1} · {mark.kind === 'rect' ? 'Área' : 'Ponto'}</strong>
                <button
                  type="button"
                  class="icon-btn"
                  title="Remover marcação"
                  onclick={(e) => {
                    e.stopPropagation();
                    onDeleteAnnotation(mark.id);
                  }}
                >
                  <Trash2 size={12} />
                </button>
              </div>
              <textarea
                id={`mark-note-${mark.id}`}
                rows={2}
                value={mark.note}
                placeholder="Ex.: trocar por família tomando café na varanda rural..."
                oninput={(e) => onUpdateAnnotation(mark.id, { note: (e.currentTarget as HTMLTextAreaElement).value })}
              ></textarea>
            </div>
          {/each}
        </div>
      {/if}

      <div class="agent-compose">
        <div class="row-between">
          <span style="font-size:11px; font-weight:600; color:var(--text-muted);">Como aplicar o resultado:</span>
          <select
            value={deliveryMode}
            onchange={(e) => onDeliveryModeChange((e.currentTarget as HTMLSelectElement).value as AgentDeliveryMode)}
            disabled={isRunning}
            style="font-size:11px; padding:3px 6px; border-radius:6px; border:1px solid var(--border); background:var(--surface);"
          >
            <option value="auto">Automático (inteligente)</option>
            <option value="background_layer">Só foto de fundo (preserva textos)</option>
            <option value="full_piece">Editar arte inteira</option>
          </select>
        </div>

        <textarea
          bind:this={promptEl}
          rows={3}
          value={promptDraft}
          disabled={isRunning || disabled}
          placeholder="Descreva o que o agente deve criar ou alterar nesta página... (Ctrl+Enter para enviar)"
          oninput={(e) => onPromptChange((e.currentTarget as HTMLTextAreaElement).value)}
          onkeydown={(e) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
              e.preventDefault();
              onSubmit();
            }
          }}
        ></textarea>

        <div class="row-between">
          <span class="muted" style="font-size:11px;">
            {isRunning ? 'Uma execução por vez ativa.' : 'Usa login ChatGPT do Codex CLI.'}
          </span>
          {#if isRunning && activeTask}
            <button type="button" class="danger-btn" onclick={() => onCancelTask(activeTask.id)}>
              <Square size={12} /> Cancelar
            </button>
          {:else}
            <button
              type="button"
              class="primary"
              onclick={onSubmit}
              disabled={disabled || !health?.available || !health?.authenticated}
            >
              <Send size={13} /> Enviar ao Agente
            </button>
          {/if}
        </div>
      </div>

      {#if activeTask?.status === 'needs_confirmation'}
        <div class="agent-confirm-box">
          <p style="margin:0 0 8px; font-size:12px;">
            <strong><AlertTriangle size={13} style="vertical-align:-2px;" /> Confirmação antes de gerar:</strong>
            {activeTask.confirmationReason}
          </p>
          <div class="row-gap">
            <button type="button" class="primary" onclick={() => activeTask && onConfirmTask(activeTask.id)}>
              Confirmar e gerar assim mesmo
            </button>
            <button type="button" onclick={() => activeTask && onCancelTask(activeTask.id)}>
              Cancelar
            </button>
          </div>
        </div>
      {/if}

      {#if activeTask || logs.length > 0}
        <div class="agent-terminal-section">
          <div class="row-between" style="margin-bottom:4px;">
            <strong style="font-size:11px;">
              Terminal Codex {activeTask ? `· ${formatStatusBadge(activeTask.status)}` : ''}
            </strong>
            {#if recentTasks.length > 1}
              <select
                value={activeTask?.id ?? ''}
                onchange={(e) => onSelectHistoryTask((e.currentTarget as HTMLSelectElement).value)}
                style="font-size:10px; max-width:170px; padding:2px 4px; border-radius:4px; border:1px solid var(--border);"
              >
                {#each recentTasks as t (t.id)}
                  <option value={t.id}>{t.contextSummary || t.id} ({t.status})</option>
                {/each}
              </select>
            {/if}
          </div>

          <pre bind:this={terminalEl} class="agent-terminal">{#if logs.length === 0}Aguardando eventos do Codex CLI...{:else}{#each logs as entry (entry.id)}[{entry.stream}] {entry.text}
{/each}{/if}</pre>

          {#if activeTask?.status === 'completed' && activeTask.outputAssetUrl}
            <div class="agent-result-bar">
              <img src={activeTask.outputAssetUrl} alt="Resultado do agente" />
              <div style="min-width:0; flex:1;">
                <strong style="font-size:11px; color:#23441b;">
                  <CheckCircle2 size={12} style="vertical-align:-2px;" /> Aplicado na v{activeTask.resultRevision}
                </strong>
                <p class="muted" style="margin:2px 0 0; font-size:11px; white-space:normal;">
                  {activeTask.summaryText || 'Imagem salva na Biblioteca e aplicada na peça.'}
                </p>
              </div>
            </div>
          {:else if activeTask && (activeTask.status === 'canceled' || activeTask.status === 'failed' || activeTask.status === 'interrupted_uncertain')}
            <div class="agent-recovery-bar">
              {#if activeTask.errorMessage}
                <p class="muted" style="margin:0 0 6px; font-size:11px; color:#7d221d;">{activeTask.errorMessage}</p>
              {/if}
              <div class="row-gap" style="flex-wrap:wrap;">
                {#if activeTask.artifactRecoverable}
                  <button type="button" class="primary" onclick={() => activeTask && onRecoverArtifact(activeTask.id)}>
                    Recuperar imagem gerada
                  </button>
                {/if}
                {#if activeTask.threadId}
                  <button type="button" onclick={() => activeTask && onResumeTask(activeTask.id)}>
                    <RotateCcw size={12} /> Retomar sessão
                  </button>
                {/if}
              </div>
            </div>
          {/if}
        </div>
      {/if}
    </section>
  {/if}

  <button
    type="button"
    class="agent-fab"
    class:running={isRunning}
    class:has-open={open}
    onclick={onToggleOpen}
    title={open ? 'Minimizar Agente Codex' : 'Abrir Agente Codex para gerar ou editar na arte'}
  >
    <Bot size={18} />
    <span>{isRunning ? 'Agente gerando...' : 'Agente IA'}</span>
    {#if unresolvedMarks.length > 0}
      <span class="agent-fab-badge">{unresolvedMarks.length}</span>
    {/if}
  </button>
</div>
