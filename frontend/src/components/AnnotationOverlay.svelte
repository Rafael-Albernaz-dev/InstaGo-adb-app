<script lang="ts">
  import type { AgentAnnotation } from '../lib/types';

  interface Props {
    annotations: AgentAnnotation[];
    workspaceWidth: number;
    workspaceHeight: number;
    markingMode: 'off' | 'pin' | 'rect';
    showMarks: boolean;
    selectedAnnotationId: string | null;
    onAddAnnotation: (kind: 'pin' | 'rect', x: number, y: number, w?: number, h?: number) => void;
    onSelectAnnotation: (id: string) => void;
  }

  let {
    annotations,
    workspaceWidth,
    workspaceHeight,
    markingMode,
    showMarks,
    selectedAnnotationId,
    onAddAnnotation,
    onSelectAnnotation
  }: Props = $props();

  let svgEl = $state<SVGSVGElement | null>(null);
  let dragStart = $state<{ x: number; y: number } | null>(null);
  let dragCurrent = $state<{ x: number; y: number } | null>(null);

  function toWorkspaceCoords(clientX: number, clientY: number): { x: number; y: number } {
    if (!svgEl) return { x: 0, y: 0 };
    const rect = svgEl.getBoundingClientRect();
    if (rect.width <= 0 || rect.height <= 0) return { x: 0, y: 0 };
    const x = ((clientX - rect.left) / rect.width) * workspaceWidth;
    const y = ((clientY - rect.top) / rect.height) * workspaceHeight;
    return {
      x: Math.max(0, Math.min(workspaceWidth, x)),
      y: Math.max(0, Math.min(workspaceHeight, y))
    };
  }

  function handlePointerDown(e: PointerEvent) {
    if (markingMode === 'off') return;
    e.preventDefault();
    const pt = toWorkspaceCoords(e.clientX, e.clientY);
    if (markingMode === 'pin') {
      onAddAnnotation('pin', pt.x, pt.y);
      return;
    }
    dragStart = pt;
    dragCurrent = pt;
    svgEl?.setPointerCapture(e.pointerId);
  }

  function handlePointerMove(e: PointerEvent) {
    if (!dragStart) return;
    dragCurrent = toWorkspaceCoords(e.clientX, e.clientY);
  }

  function handlePointerUp(e: PointerEvent) {
    if (!dragStart || !dragCurrent) return;
    if (svgEl?.hasPointerCapture(e.pointerId)) {
      svgEl.releasePointerCapture(e.pointerId);
    }
    const minX = Math.min(dragStart.x, dragCurrent.x);
    const minY = Math.min(dragStart.y, dragCurrent.y);
    const w = Math.abs(dragCurrent.x - dragStart.x);
    const h = Math.abs(dragCurrent.y - dragStart.y);
    const startPt = dragStart;
    dragStart = null;
    dragCurrent = null;

    if (w < 14 && h < 14) {
      onAddAnnotation('pin', startPt.x, startPt.y);
    } else {
      onAddAnnotation('rect', minX, minY, w, h);
    }
  }

  const previewRect = $derived.by(() => {
    if (!dragStart || !dragCurrent) return null;
    return {
      x: Math.min(dragStart.x, dragCurrent.x),
      y: Math.min(dragStart.y, dragCurrent.y),
      w: Math.abs(dragCurrent.x - dragStart.x),
      h: Math.abs(dragCurrent.y - dragStart.y)
    };
  });
</script>

{#if showMarks || markingMode !== 'off'}
  <svg
    bind:this={svgEl}
    class="annotation-overlay"
    class:capturing={markingMode !== 'off'}
    viewBox={`0 0 ${workspaceWidth} ${workspaceHeight}`}
    onpointerdown={handlePointerDown}
    onpointermove={handlePointerMove}
    onpointerup={handlePointerUp}
    role="region"
    aria-label="Marcações visuais para o agente"
  >
    {#each annotations as mark, idx (mark.id)}
      {@const isSelected = selectedAnnotationId === mark.id}
      {@const strokeColor = mark.resolved ? '#3D6A2F' : (mark.color || '#D9531E')}
      {#if mark.kind === 'rect'}
        <g
          class="annotation-item"
          class:selected={isSelected}
          onclick={(e) => {
            e.stopPropagation();
            onSelectAnnotation(mark.id);
          }}
          role="button"
          tabindex="0"
          onkeydown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault();
              onSelectAnnotation(mark.id);
            }
          }}
        >
          <rect
            x={mark.x}
            y={mark.y}
            width={mark.width ?? 60}
            height={mark.height ?? 60}
            rx="8"
            fill={mark.resolved ? 'rgba(61,106,47,0.10)' : 'rgba(217,83,30,0.12)'}
            stroke={strokeColor}
            stroke-width={isSelected ? 4 : 2.5}
            stroke-dasharray={mark.resolved ? undefined : '8 5'}
          />
          <circle cx={mark.x} cy={mark.y} r="13" fill={strokeColor} stroke="#FFFFFF" stroke-width="2" />
          <text
            x={mark.x}
            y={mark.y + 4.5}
            text-anchor="middle"
            fill="#FFFFFF"
            font-size="12"
            font-weight="700"
          >
            {idx + 1}
          </text>
        </g>
      {:else}
        <g
          class="annotation-item"
          class:selected={isSelected}
          onclick={(e) => {
            e.stopPropagation();
            onSelectAnnotation(mark.id);
          }}
          role="button"
          tabindex="0"
          onkeydown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault();
              onSelectAnnotation(mark.id);
            }
          }}
        >
          <circle
            cx={mark.x}
            cy={mark.y}
            r={isSelected ? 17 : 14}
            fill={strokeColor}
            stroke="#FFFFFF"
            stroke-width="2.5"
          />
          <text
            x={mark.x}
            y={mark.y + 4.5}
            text-anchor="middle"
            fill="#FFFFFF"
            font-size="12"
            font-weight="700"
          >
            {idx + 1}
          </text>
        </g>
      {/if}
    {/each}

    {#if previewRect}
      <rect
        x={previewRect.x}
        y={previewRect.y}
        width={previewRect.w}
        height={previewRect.h}
        rx="8"
        fill="rgba(217,83,30,0.16)"
        stroke="#D9531E"
        stroke-width="2.5"
        stroke-dasharray="6 4"
      />
    {/if}
  </svg>
{/if}
