/**
 * Design Tokens — Task Orchestrator Dashboard
 *
 * Fuente de verdad para colores, tipografía, espaciado y radios.
 * Usar estas constantes en lugar de valores hardcodeados.
 *
 * Generado a partir del diseño en Stitch (Google) + definiciones del equipo.
 */

// ─── Tipografía ────────────────────────────────────────────────────────────

export const fonts = {
  /** UI principal: navegación, títulos, etiquetas, botones */
  sans: "'Satoshi', sans-serif",
  /** Código, logs, IDs de tareas, timestamps, números técnicos */
  mono: "'JetBrains Mono', monospace",
} as const;

// ─── Paleta de colores ─────────────────────────────────────────────────────

export const colors = {
  // Fondos
  bg: {
    base: "#0e0e10",      // fondo global
    surface: "#19191d",   // tarjetas, paneles
    surfaceHigh: "#1f1f24", // hover, elementos elevados
    bright: "#2b2c32",    // tooltips, dropdowns
  },

  // Bordes
  border: {
    subtle: "#75757c",    // bordes suaves
    default: "#3d3b3e",   // bordes de componentes
  },

  // Texto
  text: {
    primary: "#e7e4ec",   // texto principal
    secondary: "#a09da1", // texto secundario, placeholders
    muted: "#75757c",     // texto desactivado
  },

  // Acento — violeta suave
  violet: {
    DEFAULT: "#d0bcff",   // acento principal
    container: "#5516be", // fondos activos
    strong: "#4e03b8",    // hover sobre acento
  },

  // Estados semánticos
  status: {
    success: "#4ade80",   // verde-400 — SUCCESS
    failed: "#f87171",    // rojo-400 — FAILED
    running: "#d0bcff",   // violeta — RUNNING
    queued: "#fbbf24",    // ámbar-400 — QUEUED
    pending: "#a09da1",   // zinc — PENDING
    timeout: "#fb923c",   // naranja-400 — TIMEOUT
    cancelled: "#6b7280", // gris-500 — CANCELLED
  },

  // Runtime badges
  runtime: {
    python: "#3b82f6",    // azul
    nodejs: "#22c55e",    // verde
    typescript: "#60a5fa",// azul claro
    go: "#06b6d4",        // cyan
    java: "#f97316",      // naranja
  },
} as const;

// ─── Border radius ─────────────────────────────────────────────────────────

export const radius = {
  sm: "0.125rem",   // 2px  — detalles mínimos
  md: "0.25rem",    // 4px  — tarjetas, inputs
  lg: "0.5rem",     // 8px  — modales, dropdowns
  full: "0.75rem",  // 12px — badges, pills
} as const;

// ─── Espaciado sidebar ─────────────────────────────────────────────────────

export const sidebar = {
  collapsed: "48px",  // solo íconos
  expanded: "200px",  // íconos + labels (hover)
} as const;

// ─── CSS custom properties (para uso en app.css) ──────────────────────────
/**
 * Equivalencias en Tailwind CSS v4 / CSS variables:
 *
 * --color-bg-base:        #0e0e10
 * --color-bg-surface:     #19191d
 * --color-accent:         #d0bcff
 * --color-text:           #e7e4ec
 * --color-text-muted:     #a09da1
 * --font-sans:            'Satoshi', sans-serif
 * --font-mono:            'JetBrains Mono', monospace
 */
