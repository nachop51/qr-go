// Retro ink chart: a fixed, quantized palette standing in for the native
// color input, in a shared popover. Quantized like the codes themselves:
// 12 hues x 6 shades plus a 12-step gray ramp, every cell a real button.
// One chart on the page, retargeted to whichever swatch opened it; free-form
// colors still come in through the hex field next to each swatch.

export interface SwatchPickerOptions {
  button: HTMLButtonElement;
  label: string;
  onPick: (hex: string) => void;
}

export interface SwatchPicker {
  /** Current color; setting repaints the swatch and the open chart. */
  value: string;
}

const COLS = 12;
const HUES = [0, 30, 60, 90, 120, 150, 180, 210, 240, 270, 300, 330];
const SHADES = [86, 72, 58, 46, 34, 22]; // lightness rows, light to dark

function hslToHex(h: number, s: number, l: number): string {
  const f = (n: number) => {
    const k = (n + h / 30) % 12;
    const a = (s / 100) * Math.min(l / 100, 1 - l / 100);
    const c = l / 100 - a * Math.max(-1, Math.min(k - 3, 9 - k, 1));
    return Math.round(255 * c)
      .toString(16)
      .padStart(2, "0");
  };
  return `#${f(0)}${f(8)}${f(4)}`;
}

const PALETTE: string[] = [];
for (const l of SHADES) for (const h of HUES) PALETTE.push(hslToHex(h, 78, l));
const GRAYS = Array.from({ length: COLS }, (_, i) => hslToHex(0, 0, 100 - (i * 100) / (COLS - 1)));
// The chart's black is the studio ink, not #000000: it's the page's ink, the
// QR default, and it means both default swatches land on a real cell, so the
// chart always opens with the current color marked.
GRAYS[GRAYS.length - 1] = "#1a1a1e";

interface ActiveSwatch {
  button: HTMLButtonElement;
  label: string;
  readonly value: string;
  apply(hex: string): void;
}

let chartEl: HTMLDivElement | null = null;
let labelEl: HTMLSpanElement;
let hexEl: HTMLSpanElement;
let active: ActiveSwatch | null = null;
const cells: HTMLButtonElement[] = [];

function makeGrid(colors: string[]): HTMLDivElement {
  const grid = document.createElement("div");
  grid.className = "ink-grid";
  for (const hex of colors) {
    const cell = document.createElement("button");
    cell.type = "button";
    cell.className = "ink-cell";
    cell.dataset.hex = hex;
    cell.tabIndex = -1;
    cell.style.setProperty("--c", hex);
    cell.setAttribute("aria-label", hex);
    cell.setAttribute("aria-pressed", "false");
    cell.addEventListener("click", () => active?.apply(hex));
    grid.append(cell);
    cells.push(cell);
  }
  return grid;
}

function mark(hex: string): void {
  for (const cell of cells) {
    const on = cell.dataset.hex === hex;
    cell.classList.toggle("on", on);
    cell.setAttribute("aria-pressed", String(on));
  }
}

function focusCell(cell: HTMLButtonElement): void {
  for (const c of cells) c.tabIndex = -1;
  cell.tabIndex = 0;
  cell.focus({ preventScroll: true });
}

function place(anchor: HTMLElement): void {
  if (!chartEl) return;
  const r = anchor.getBoundingClientRect();
  const w = chartEl.offsetWidth;
  const h = chartEl.offsetHeight;
  const left = Math.max(8, Math.min(r.left, window.innerWidth - w - 8));
  let top = r.bottom + 8;
  if (top + h > window.innerHeight - 8) top = Math.max(8, r.top - h - 8);
  chartEl.style.left = `${left}px`;
  chartEl.style.top = `${top}px`;
}

function ensureChart(): HTMLDivElement {
  if (chartEl) return chartEl;

  chartEl = document.createElement("div");
  chartEl.className = "ink-chart";
  chartEl.setAttribute("popover", "");
  chartEl.setAttribute("role", "dialog");

  const head = document.createElement("div");
  head.className = "ink-chart-head";
  labelEl = document.createElement("span");
  labelEl.className = "ink-chart-label";
  hexEl = document.createElement("span");
  hexEl.className = "ink-chart-hex";
  head.append(labelEl, hexEl);

  chartEl.append(head, makeGrid(PALETTE), makeGrid(GRAYS));
  document.body.append(chartEl);

  // Both grids share the 12-column pitch, so one flat index walks the whole
  // chart: left/right step a cell, up/down step a row across the gray ramp.
  chartEl.addEventListener("keydown", (e) => {
    const moves: Record<string, number> = {
      ArrowRight: 1,
      ArrowLeft: -1,
      ArrowDown: COLS,
      ArrowUp: -COLS,
    };
    const move = moves[e.key];
    if (move === undefined) return;
    const idx = cells.indexOf(document.activeElement as HTMLButtonElement);
    if (idx < 0) return;
    const next = cells[idx + move];
    if (next) {
      e.preventDefault();
      focusCell(next);
    }
  });

  chartEl.addEventListener("toggle", (e) => {
    if ((e as ToggleEvent).newState === "closed" && active) {
      active.button.setAttribute("aria-expanded", "false");
      // Esc leaves focus on a cell inside the hidden popover; hand it back.
      if (chartEl?.contains(document.activeElement)) active.button.focus();
      active = null;
    }
  });

  // Fixed positioning: track the anchor while the page moves under the chart.
  const follow = () => {
    if (active && chartEl?.matches(":popover-open")) place(active.button);
  };
  window.addEventListener("resize", follow);
  window.addEventListener("scroll", follow, true);

  return chartEl;
}

function open(swatch: ActiveSwatch): void {
  const chart = ensureChart();
  active = swatch;
  labelEl.textContent = swatch.label;
  hexEl.textContent = swatch.value;
  mark(swatch.value);
  chart.setAttribute("aria-label", `${swatch.label} ink chart`);
  chart.showPopover();
  place(swatch.button);
  swatch.button.setAttribute("aria-expanded", "true");
  focusCell(cells.find((c) => c.dataset.hex === swatch.value) ?? cells[0]);
}

export function createSwatchPicker(opts: SwatchPickerOptions): SwatchPicker {
  const { button, label, onPick } = opts;
  let value = (button.dataset.value ?? "#000000").toLowerCase();

  const paint = () => {
    button.style.setProperty("--swatch", value);
    button.dataset.value = value;
  };
  const repaintChart = () => {
    if (active === self) {
      hexEl.textContent = value;
      mark(value);
    }
  };
  paint();
  button.setAttribute("aria-haspopup", "dialog");
  button.setAttribute("aria-expanded", "false");

  const self: ActiveSwatch = {
    button,
    label,
    get value() {
      return value;
    },
    apply(hex: string) {
      value = hex.toLowerCase();
      paint();
      repaintChart();
      onPick(value);
    },
  };

  // Caret at the hex field's right edge: a second, conspicuous trigger for
  // the chart; the bare swatch reads as a static preview to first-time users.
  // Redundant for keyboard/SR (the swatch button is the accessible trigger),
  // so it's skipped by both.
  const caret = document.createElement("button");
  caret.type = "button";
  caret.className = "swatch-caret";
  caret.tabIndex = -1;
  caret.setAttribute("aria-hidden", "true");
  button.closest(".swatch-combo")?.append(caret);

  // Clicking an owner trigger while its chart is open should close, not
  // reopen: light dismiss already fired on pointerdown, so flag that case.
  let dismissedBySelf = false;
  for (const trigger of [button, caret]) {
    trigger.addEventListener("pointerdown", () => {
      dismissedBySelf = active === self;
    });
    trigger.addEventListener("click", () => {
      if (dismissedBySelf) {
        dismissedBySelf = false;
        return;
      }
      open(self);
    });
  }

  return {
    get value() {
      return value;
    },
    set value(hex: string) {
      value = hex.toLowerCase();
      paint();
      repaintChart();
    },
  };
}
