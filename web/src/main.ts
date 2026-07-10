import {
  initQrgo,
  isError,
  type EyeShape,
  type GenerateOptions,
  type GenerateResult,
  type ModuleShape,
} from "./qrgo";
import { CONTENT_TYPES, FIELDS, type Field, type Values } from "./forms";
import { createSwatchPicker } from "./colorpicker";

// --- pixel wordmark ---------------------------------------------------------
// 5x7 bitmap glyphs rendered as SVG modules; the separator is a mini finder
// pattern in the second ink. Same visual grammar as the codes themselves.

const GLYPHS: Record<string, string[]> = {
  Q: [".###.", "#...#", "#...#", "#...#", "#.#.#", "#..#.", ".##.#"],
  R: ["####.", "#...#", "#...#", "####.", "#.#..", "#..#.", "#...#"],
  G: [".###.", "#...#", "#....", "#.###", "#...#", "#...#", ".###."],
  O: [".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."],
  "·": [".....", "#####", "#...#", "#.#.#", "#...#", "#####", "....."],
};

function drawWordmark(svg: SVGElement, word: string): void {
  const M = 6,
    GAP = 2; // module px, modules between glyphs
  let x = 0;
  const rects: string[] = [];
  let i = 0;
  for (const ch of word) {
    const glyph = GLYPHS[ch];
    if (!glyph) continue;
    const ink = ch === "·" ? "var(--blue)" : "var(--ink)";
    glyph.forEach((row, gy) => {
      [...row].forEach((cell, gx) => {
        if (cell !== "#") return;
        rects.push(
          `<rect x="${(x + gx) * M}" y="${gy * M}" width="${M}" height="${M}" fill="${ink}" style="--i:${i++}"/>`,
        );
      });
    });
    x += 5 + GAP;
  }
  const w = (x - GAP) * M;
  svg.setAttribute("viewBox", `0 0 ${w} ${7 * M}`);
  svg.setAttribute("width", String(w));
  svg.setAttribute("height", String(7 * M));
  svg.innerHTML = rects.join("");
}

// --- state ------------------------------------------------------------------

interface State {
  type: string;
  values: Record<string, Values>; // per content type, so switching tabs keeps input
  ecLevel: "L" | "M" | "Q" | "H";
  eciPolicy: "auto" | "disabled";
  version: number; // 0 = auto
  mask: number; // -1 = auto
  dark: string;
  light: string;
  quiet: number;
  pngSize: number;
  logo: Uint8Array | null;
  logoModules: number; // 0 = max the EC level allows
  logoScale: number; // % of the logo area the image fills
  moduleShape: ModuleShape;
  eyeFrameShape: EyeShape;
  eyeBallShape: EyeShape;
  eyeFrame: string; // "" = follow the module color
  eyeBall: string; // "" = follow the module color
  gradientKind: "none" | "linear" | "radial";
  gradientFrom: string;
  gradientTo: string;
  gradientAngle: number; // degrees, linear only
}

const state: State = {
  type: "text",
  values: Object.fromEntries(Object.keys(FIELDS).map((k) => [k, {}])),
  ecLevel: "M",
  eciPolicy: "auto",
  version: 0,
  mask: -1,
  dark: "#1a1a1e",
  light: "#ffffff",
  quiet: 4,
  pngSize: 1280,
  logo: null,
  logoModules: 0,
  logoScale: 80,
  moduleShape: "square",
  eyeFrameShape: "square",
  eyeBallShape: "square",
  eyeFrame: "",
  eyeBall: "",
  gradientKind: "none",
  gradientFrom: "#1a1a1e",
  gradientTo: "#4338ca",
  gradientAngle: 45,
};

let lastPayload = "";
let lastResult: GenerateResult | null = null;

const EC_HINTS: Record<string, string> = {
  L: "L recovers up to 7% damage (smallest code)",
  M: "M recovers up to 15% damage",
  Q: "Q recovers up to 25% damage",
  H: "H recovers up to 30% damage (best with a logo)",
};

// --- dom --------------------------------------------------------------------

const $ = <T extends HTMLElement>(sel: string): T => document.querySelector(sel) as T;

const tabsEl = $("#tabs");
const formEl = $<HTMLFormElement>("#content-form");
const panelEl = $("#content-panel");
const previewEl = $("#preview");
const ticketEl = $("#ticket");
const specEl = $("#spec");
const warningsEl = $("#warnings");
const payloadEl = $("#payload");
const logoExtrasEl = $("#logo-extras");
const logoSpanRowEl = $("#logo-span-row");
const logoModulesEl = $<HTMLInputElement>("#logo-modules");
const logoModulesOut = $("#logo-modules-out");
const logoScaleRowEl = $("#logo-scale-row");
const logoScaleEl = $<HTMLInputElement>("#logo-scale");
const logoScaleOut = $("#logo-scale-out");

// setEcLevel updates the level everywhere it shows: state, the radio group,
// and the hint. Used by the radios themselves and by the logo picker, which
// bumps the level to High so the code survives the overlay.
function setEcLevel(level: State["ecLevel"]): void {
  state.ecLevel = level;
  const radio = document.querySelector<HTMLInputElement>(
    `#ec-levels input[name="ecLevel"][value="${level}"]`,
  );
  if (radio) radio.checked = true;
  $("#ec-hint").textContent = EC_HINTS[level];
}

// The "QR shape" select is a shortcut over the three shape selects in More
// style, not state of its own: picking a preset writes through to all three,
// and the select itself only reflects whether the current combination still
// matches one. When it doesn't, it shows a display-only "Custom" option.
const QR_SHAPE_PRESETS: Record<
  string,
  { module: ModuleShape; frame: EyeShape; ball: EyeShape }
> = {
  square: { module: "square", frame: "square", ball: "square" },
  rounded: { module: "rounded", frame: "rounded", ball: "rounded" },
  circle: { module: "dot", frame: "circle", ball: "circle" },
};

function syncQrShapePreset(): void {
  const sel = $<HTMLSelectElement>("#qr-shape");
  const match = Object.entries(QR_SHAPE_PRESETS).find(
    ([, p]) =>
      p.module === state.moduleShape &&
      p.frame === state.eyeFrameShape &&
      p.ball === state.eyeBallShape,
  );
  const custom = sel.querySelector<HTMLOptionElement>('option[value="custom"]');
  if (match) {
    custom?.remove();
    sel.value = match[0];
  } else {
    if (!custom) {
      const o = document.createElement("option");
      o.value = "custom";
      o.textContent = "Custom";
      o.disabled = true; // shown as the current value, never user-pickable
      sel.append(o);
    }
    sel.value = "custom";
  }
}

// The slider's right end means "max the EC level allows" (sent as 0 so the
// span keeps tracking the budget when the code's size or EC level changes);
// anything left of it is an explicit span.
function readLogoSpan(): void {
  const v = Number(logoModulesEl.value);
  state.logoModules = v >= Number(logoModulesEl.max) ? 0 : v;
  logoModulesOut.textContent = state.logoModules === 0 ? "max" : `${state.logoModules} modules`;
}

function selectTab(btn: HTMLButtonElement, focus = false): void {
  state.type = btn.dataset.type!;
  for (const b of tabsEl.children as Iterable<HTMLButtonElement>) {
    const active = b === btn;
    b.ariaSelected = String(active);
    b.tabIndex = active ? 0 : -1;
  }
  panelEl.setAttribute("aria-labelledby", btn.id);
  if (focus) btn.focus();
  buildForm(!focus); // arrow-key browsing keeps focus on the tab
  regenerate();
}

function buildTabs(): void {
  tabsEl.innerHTML = ""; // replaces the static boot copy from index.html
  for (const t of CONTENT_TYPES) {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.textContent = t.label;
    btn.role = "tab";
    btn.id = `tab-${t.id}`;
    btn.dataset.type = t.id;
    btn.setAttribute("aria-controls", "content-panel");
    const active = t.id === state.type;
    btn.ariaSelected = String(active);
    btn.tabIndex = active ? 0 : -1;
    btn.addEventListener("click", () => selectTab(btn));
    tabsEl.append(btn);
  }
  panelEl.setAttribute("aria-labelledby", `tab-${state.type}`);

  // Roving tabindex: one tab stop, arrows move and select.
  tabsEl.addEventListener("keydown", (e) => {
    const tabs = [...tabsEl.children] as HTMLButtonElement[];
    const current = tabs.findIndex((b) => b.tabIndex === 0);
    let next: number;
    switch (e.key) {
      case "ArrowRight":
      case "ArrowDown":
        next = (current + 1) % tabs.length;
        break;
      case "ArrowLeft":
      case "ArrowUp":
        next = (current - 1 + tabs.length) % tabs.length;
        break;
      case "Home":
        next = 0;
        break;
      case "End":
        next = tabs.length - 1;
        break;
      default:
        return;
    }
    e.preventDefault();
    selectTab(tabs[next], true);
  });
}

function buildForm(focusField = false): void {
  const values = state.values[state.type];
  // Adopt anything typed into the static boot form before this script ran.
  for (const el of formEl.querySelectorAll<HTMLInputElement | HTMLTextAreaElement>('[id^="f-"]')) {
    const key = el.id.slice(2);
    if (el.value && values[key] === undefined) values[key] = el.value;
  }
  formEl.innerHTML = "";
  FIELDS[state.type].forEach((f, rowIndex) => {
    const row = document.createElement("div");
    row.className = f.kind === "checkbox" ? "field-row check" : "field-row";
    row.style.setProperty("--i", String(rowIndex)); // stagger the stamp-in

    const label = document.createElement("label");
    label.textContent = f.label;
    label.htmlFor = `f-${f.key}`;

    let input: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement;
    switch (f.kind) {
      case "textarea":
        input = document.createElement("textarea");
        input.rows = 3;
        break;
      case "select": {
        const sel = document.createElement("select");
        for (const [value, text] of f.options ?? []) {
          const opt = document.createElement("option");
          opt.value = value;
          opt.textContent = text;
          sel.append(opt);
        }
        input = sel;
        break;
      }
      default: {
        const inp = document.createElement("input");
        inp.type =
          { datetime: "datetime-local", checkbox: "checkbox", number: "number" }[
            f.kind as string
          ] ?? f.kind;
        if (f.step) inp.step = f.step;
        input = inp;
      }
    }
    input.id = `f-${f.key}`;
    if ("placeholder" in input && f.placeholder) input.placeholder = f.placeholder;

    const saved = values[f.key];
    if (f.kind === "checkbox") (input as HTMLInputElement).checked = saved === "true";
    else if (saved !== undefined) input.value = saved;

    const typed = f.kind !== "checkbox" && f.kind !== "select";
    input.addEventListener("input", () => {
      values[f.key] =
        f.kind === "checkbox" ? String((input as HTMLInputElement).checked) : input.value;
      regenerate(typed ? TYPING_DEBOUNCE : undefined);
    });

    if (f.kind === "checkbox") row.append(input, label);
    else row.append(label, input);
    formEl.append(row);
  });
  // preventScroll: on mobile the form sits below the preview ticket, and a
  // scrolling boot focus reads as the page jumping (Lighthouse flags it as CLS).
  if (focusField)
    (formEl.querySelector("input, textarea") as HTMLElement | null)?.focus({
      preventScroll: true,
    });
}

// --- generation -------------------------------------------------------------

function options(format: "png" | "svg"): GenerateOptions {
  return {
    text: lastPayload,
    ecLevel: state.ecLevel,
    eciPolicy: state.eciPolicy,
    format,
    dark: state.dark,
    light: state.light,
    quiet: state.quiet,
    ...(format === "png" ? { size: state.pngSize } : {}),
    ...(state.logo
      ? { logo: state.logo, logoModules: state.logoModules, logoScale: state.logoScale }
      : {}),
    ...(state.version > 0 ? { version: state.version } : {}),
    ...(state.mask >= 0 ? { mask: state.mask } : {}),
    ...(state.moduleShape !== "square" ? { moduleShape: state.moduleShape } : {}),
    ...(state.eyeFrameShape !== "square" ? { eyeFrameShape: state.eyeFrameShape } : {}),
    ...(state.eyeBallShape !== "square" ? { eyeBallShape: state.eyeBallShape } : {}),
    ...(state.eyeFrame ? { eyeFrame: state.eyeFrame } : {}),
    ...(state.eyeBall ? { eyeBall: state.eyeBall } : {}),
    ...(state.gradientKind !== "none"
      ? {
          gradient: {
            kind: state.gradientKind,
            from: state.gradientFrom,
            to: state.gradientTo,
            ...(state.gradientKind === "linear" ? { angle: state.gradientAngle } : {}),
          },
        }
      : {}),
  };
}

function showEmpty(message: string, isProblem = false): void {
  previewEl.removeAttribute("role");
  previewEl.removeAttribute("aria-label");
  previewEl.innerHTML = "";
  const p = document.createElement("p");
  p.className = isProblem ? "empty problem" : "empty";
  p.textContent = message;
  previewEl.append(p);
  specEl.textContent = "";
  warningsEl.textContent = "";
  payloadEl.textContent = "";
  lastResult = null;
  setDownloadsEnabled(false);
}

function setDownloadsEnabled(on: boolean): void {
  for (const id of ["#download", "#copy"]) ($(id) as HTMLButtonElement).disabled = !on;
}

// The format switch scopes both action verbs; the buttons stay put, only
// their accessible names follow the selection.
function exportFormat(): "png" | "svg" {
  return document.querySelector<HTMLInputElement>("#dl-format input:checked")?.value === "svg"
    ? "svg"
    : "png";
}

function syncExportLabels(): void {
  const fmt = exportFormat();
  $("#download").setAttribute("aria-label", fmt === "png" ? "Download PNG" : "Download SVG");
  // SVG lands as markup text: image/svg+xml is not paste-able in most apps.
  $("#copy").setAttribute("aria-label", fmt === "png" ? "Copy PNG image" : "Copy SVG markup");
}

// Discrete controls (radios, selects, sliders) coalesce on a short delay;
// typed fields wait longer so the code doesn't reprint on every keystroke.
const TYPING_DEBOUNCE = 350;
let timer: ReturnType<typeof setTimeout> | undefined;
function regenerate(delay = 120): void {
  clearTimeout(timer);
  timer = setTimeout(render, delay);
}

// Controls are live before the wasm module finishes loading; renders that
// arrive early are dropped and the post-init render picks up the latest state.
let wasmReady = false;

function render(): void {
  if (!wasmReady) return showEmpty("Warming up the printer…");
  const type = CONTENT_TYPES.find((t) => t.id === state.type)!;
  const payload = type.payload(state.values[state.type]);

  if (isError(payload)) return showEmpty(payload.error, true);
  if (payload === "") return showEmpty("Type something to print a code");

  lastPayload = payload;
  const result = qrgo.generate(options("svg"));
  if (isError(result)) return showEmpty(result.error, true);

  lastResult = result;
  previewEl.setAttribute("role", "img");
  previewEl.setAttribute("aria-label", `QR code, ${result.size} by ${result.size} modules, encoding: ${payload.length > 60 ? payload.slice(0, 60) + "…" : payload}`);
  previewEl.innerHTML = result.data as string;
  const svg = previewEl.querySelector("svg");
  if (svg) {
    svg.setAttribute("width", "100%");
    svg.setAttribute("height", "100%");
  }
  ticketEl.classList.remove("printed");
  void ticketEl.offsetWidth; // restart the re-print flick
  ticketEl.classList.add("printed");

  const bytes = new TextEncoder().encode(payload).length;
  specEl.textContent = `${result.size}×${result.size} modules · v${result.version} · EC ${state.ecLevel} · mask ${result.mask} · ${bytes} bytes`;
  warningsEl.textContent = (result.warnings ?? []).join(" · ");
  payloadEl.textContent = payload;

  // The budget moves with the code's size and EC level; keep the thumb pinned
  // to the right end while the span is "max", and re-derive an explicit span
  // (the browser clamps the value when the new max is smaller). Only odd
  // spans centre on the module grid, so with the slider stepping 1,3,5… the
  // max itself must be odd or the "max" stop becomes unreachable.
  logoModulesEl.max = String(result.maxLogoModules - ((result.maxLogoModules + 1) % 2));
  if (state.logoModules === 0) logoModulesEl.value = logoModulesEl.max;
  readLogoSpan();
  setDownloadsEnabled(true);
}

// The logo only ever spans a fraction of the code, so a phone photo's full
// resolution is pure overhead: every regeneration would decode it in wasm and
// the SVG renderer would embed it as a data URI. Shrink it once, up front.
async function downscaleLogo(file: File, maxDim = 512): Promise<Uint8Array> {
  // SVG bytes are already tiny and Go rasterizes them at its own target size;
  // pushing one through canvas would just bake it into a PNG.
  if (file.type === "image/svg+xml") {
    return new Uint8Array(await file.arrayBuffer());
  }
  try {
    const bmp = await createImageBitmap(file);
    const scale = Math.min(1, maxDim / Math.max(bmp.width, bmp.height));
    if (scale === 1) {
      bmp.close();
      return new Uint8Array(await file.arrayBuffer());
    }
    const canvas = document.createElement("canvas");
    canvas.width = Math.round(bmp.width * scale);
    canvas.height = Math.round(bmp.height * scale);
    canvas.getContext("2d")!.drawImage(bmp, 0, 0, canvas.width, canvas.height);
    bmp.close();
    const blob = await new Promise<Blob>((resolve, reject) =>
      canvas.toBlob(
        (b) => (b ? resolve(b) : reject(new Error("canvas.toBlob failed"))),
        "image/png",
      ),
    );
    return new Uint8Array(await blob.arrayBuffer());
  } catch {
    // Undecodable in the browser (or canvas unavailable): let Go try the raw bytes.
    return new Uint8Array(await file.arrayBuffer());
  }
}

// --- downloads --------------------------------------------------------------

function download(blob: Blob, filename: string): void {
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
  URL.revokeObjectURL(a.href);
}

function slug(): string {
  return `qr-${state.type}`;
}

// Brief "receipt" on a button after it did its job, then back to normal.
const receiptTimers = new WeakMap<HTMLButtonElement, ReturnType<typeof setTimeout>>();
function receipt(btn: HTMLButtonElement, message: string): void {
  const original = (btn.dataset.label ??= btn.textContent!);
  clearTimeout(receiptTimers.get(btn));
  btn.textContent = message;
  btn.classList.add("done");
  receiptTimers.set(
    btn,
    setTimeout(() => {
      btn.textContent = original;
      btn.classList.remove("done");
    }, 1200),
  );
}

// --- wiring -----------------------------------------------------------------

// Browsers (notably Firefox) restore form control values across reloads and
// bfcache navigations. State boots from hardcoded defaults, so it must read
// the DOM once wired, or restored controls and readouts disagree (e.g. L
// checked while the hint still describes M).
function syncFromDom(): void {
  const ec = document.querySelector<HTMLInputElement>("#ec-levels input:checked");
  if (ec) state.ecLevel = ec.value as State["ecLevel"];
  $("#ec-hint").textContent = EC_HINTS[state.ecLevel];

  const eci = document.querySelector<HTMLInputElement>("#eci-policy input:checked");
  if (eci) state.eciPolicy = eci.value as State["eciPolicy"];

  for (const key of ["dark", "light"] as const) {
    const swatch = $<HTMLButtonElement>(`#${key}`);
    state[key] = (swatch.dataset.value ?? state[key]).toLowerCase();
    $<HTMLInputElement>(`#${key}-hex`).value = state[key];
  }

  const quiet = $<HTMLInputElement>("#quiet");
  state.quiet = Number(quiet.value);
  $("#quiet-out").textContent = quiet.value;

  const pngSize = $<HTMLInputElement>("#png-size");
  state.pngSize = Number(pngSize.value);
  $("#png-size-out").textContent = pngSize.value;

  state.version = Number($<HTMLSelectElement>("#version").value);
  state.mask = Number($<HTMLSelectElement>("#mask").value);

  state.moduleShape = $<HTMLSelectElement>("#module-shape").value as State["moduleShape"];
  state.eyeFrameShape = $<HTMLSelectElement>("#eye-frame-shape").value as State["eyeFrameShape"];
  state.eyeBallShape = $<HTMLSelectElement>("#eye-ball-shape").value as State["eyeBallShape"];
  syncQrShapePreset(); // the QR shape select is derived from the three above
  state.gradientKind = $<HTMLSelectElement>("#gradient-kind").value as State["gradientKind"];

  // Eye colors: an empty hex field means "follow the module color".
  for (const [key, id] of [
    ["eyeFrame", "eye-frame"],
    ["eyeBall", "eye-ball"],
  ] as const) {
    const hex = $<HTMLInputElement>(`#${id}-hex`).value.trim().toLowerCase();
    state[key] = /^#[0-9a-f]{6}$/.test(hex) ? hex : "";
  }
  for (const [key, id] of [
    ["gradientFrom", "grad-from"],
    ["gradientTo", "grad-to"],
  ] as const) {
    const swatch = $<HTMLButtonElement>(`#${id}`);
    state[key] = (swatch.dataset.value ?? state[key]).toLowerCase();
    $<HTMLInputElement>(`#${id}-hex`).value = state[key];
  }
  const angle = $<HTMLInputElement>("#gradient-angle");
  state.gradientAngle = Number(angle.value);
  $("#gradient-angle-out").textContent = angle.value;
  syncGradientRows();

  readLogoSpan();
  state.logoScale = Number(logoScaleEl.value);
  logoScaleOut.textContent = logoScaleEl.value;
  syncExportLabels();
}

// The gradient color/angle rows only make sense while a gradient is active.
function syncGradientRows(): void {
  $("#gradient-colors").classList.toggle("hidden", state.gradientKind === "none");
  $("#gradient-angle-row").classList.toggle("hidden", state.gradientKind !== "linear");
}

function wireStyleControls(): void {
  for (const radio of document.querySelectorAll<HTMLInputElement>(
    '#ec-levels input[name="ecLevel"]',
  )) {
    radio.addEventListener("change", () => {
      setEcLevel(radio.value as State["ecLevel"]);
      regenerate();
    });
  }

  // Version and mask: selects constrain input to valid values; anything the
  // encoder still rejects (forced version too small) surfaces via the error path.
  const versionSel = $<HTMLSelectElement>("#version");
  const maskSel = $<HTMLSelectElement>("#mask");

  const opt = (value: string, label: string) => {
    const o = document.createElement("option");
    o.value = value;
    o.textContent = label;
    return o;
  };
  versionSel.append(opt("0", "Auto"));
  for (let v = 1; v <= 40; v++) {
    const modules = 17 + 4 * v;
    versionSel.append(opt(String(v), `v${v} · ${modules}×${modules}`));
  }
  maskSel.append(opt("-1", "Auto"));
  for (let m = 0; m <= 7; m++) maskSel.append(opt(String(m), `Pattern ${m}`));

  versionSel.addEventListener("change", () => {
    state.version = Number(versionSel.value);
    regenerate();
  });
  maskSel.addEventListener("change", () => {
    state.mask = Number(maskSel.value);
    regenerate();
  });

  for (const radio of document.querySelectorAll<HTMLInputElement>(
    '#eci-policy input[name="eciPolicy"]',
  )) {
    radio.addEventListener("change", () => {
      state.eciPolicy = radio.value as State["eciPolicy"];
      regenerate();
    });
  }

  // Swatch + hex sync both ways; the hex field only applies once valid.
  const wireSwatch = (key: "dark" | "light") => {
    const hex = $<HTMLInputElement>(`#${key}-hex`);
    const picker = createSwatchPicker({
      button: $<HTMLButtonElement>(`#${key}`),
      label: key === "dark" ? "Modules" : "Background",
      onPick: (value) => {
        state[key] = value;
        hex.value = value;
        regenerate();
      },
    });
    hex.addEventListener("input", () => {
      const normalized = hex.value.startsWith("#") ? hex.value : `#${hex.value}`;
      if (!/^#[0-9a-fA-F]{6}$/.test(normalized)) return;
      state[key] = normalized.toLowerCase();
      picker.value = state[key];
      regenerate(TYPING_DEBOUNCE);
    });
    hex.addEventListener("blur", () => (hex.value = state[key]));
  };
  wireSwatch("dark");
  wireSwatch("light");

  // Generic hex+swatch wiring for the styling colors. allowEmpty maps a
  // cleared field to "" (follow the module color).
  const wireColor = (
    id: string,
    label: string,
    get: () => string,
    set: (v: string) => void,
    allowEmpty = false,
  ) => {
    const hex = $<HTMLInputElement>(`#${id}-hex`);
    const picker = createSwatchPicker({
      button: $<HTMLButtonElement>(`#${id}`),
      label,
      onPick: (value) => {
        set(value);
        hex.value = value;
        regenerate();
      },
    });
    hex.addEventListener("input", () => {
      const raw = hex.value.trim();
      if (allowEmpty && raw === "") {
        set("");
        regenerate();
        return;
      }
      const normalized = raw.startsWith("#") ? raw : `#${raw}`;
      if (!/^#[0-9a-fA-F]{6}$/.test(normalized)) return;
      set(normalized.toLowerCase());
      picker.value = get();
      regenerate(TYPING_DEBOUNCE);
    });
    hex.addEventListener("blur", () => (hex.value = get()));
  };
  wireColor("eye-frame", "Eye frame", () => state.eyeFrame, (v) => (state.eyeFrame = v), true);
  wireColor("eye-ball", "Eye ball", () => state.eyeBall, (v) => (state.eyeBall = v), true);
  wireColor("grad-from", "Gradient start", () => state.gradientFrom, (v) => (state.gradientFrom = v));
  wireColor("grad-to", "Gradient end", () => state.gradientTo, (v) => (state.gradientTo = v));

  const qrShapeSel = $<HTMLSelectElement>("#qr-shape");
  const moduleShapeSel = $<HTMLSelectElement>("#module-shape");
  const frameShapeSel = $<HTMLSelectElement>("#eye-frame-shape");
  const ballShapeSel = $<HTMLSelectElement>("#eye-ball-shape");

  qrShapeSel.addEventListener("change", () => {
    const p = QR_SHAPE_PRESETS[qrShapeSel.value];
    if (!p) return;
    state.moduleShape = p.module;
    state.eyeFrameShape = p.frame;
    state.eyeBallShape = p.ball;
    moduleShapeSel.value = p.module;
    frameShapeSel.value = p.frame;
    ballShapeSel.value = p.ball;
    syncQrShapePreset(); // drops the stale Custom option
    regenerate();
  });

  moduleShapeSel.addEventListener("change", () => {
    state.moduleShape = moduleShapeSel.value as State["moduleShape"];
    syncQrShapePreset();
    regenerate();
  });
  frameShapeSel.addEventListener("change", () => {
    state.eyeFrameShape = frameShapeSel.value as State["eyeFrameShape"];
    syncQrShapePreset();
    regenerate();
  });
  ballShapeSel.addEventListener("change", () => {
    state.eyeBallShape = ballShapeSel.value as State["eyeBallShape"];
    syncQrShapePreset();
    regenerate();
  });

  const gradientKindSel = $<HTMLSelectElement>("#gradient-kind");
  gradientKindSel.addEventListener("change", () => {
    state.gradientKind = gradientKindSel.value as State["gradientKind"];
    syncGradientRows();
    regenerate();
  });
  const gradientAngle = $<HTMLInputElement>("#gradient-angle");
  gradientAngle.addEventListener("input", () => {
    state.gradientAngle = Number(gradientAngle.value);
    $("#gradient-angle-out").textContent = gradientAngle.value;
    regenerate();
  });

  const quiet = $<HTMLInputElement>("#quiet");
  quiet.addEventListener("input", () => {
    state.quiet = Number(quiet.value);
    $("#quiet-out").textContent = quiet.value;
    regenerate();
  });

  const pngSize = $<HTMLInputElement>("#png-size");
  pngSize.addEventListener("input", () => {
    state.pngSize = Number(pngSize.value);
    $("#png-size-out").textContent = pngSize.value;
  });

  const logoInput = $<HTMLInputElement>("#logo");
  const logoName = $("#logo-name");
  logoInput.addEventListener("change", async () => {
    const file = logoInput.files?.[0];
    if (!file) return;
    state.logo = await downscaleLogo(file);
    // A logo eats modules; High recovers the most, so switch to it quietly.
    setEcLevel("H");
    logoName.textContent = file.name;
    logoExtrasEl.classList.remove("hidden");
    logoSpanRowEl.classList.remove("hidden");
    logoScaleRowEl.classList.remove("hidden");
    regenerate();
  });

  logoModulesEl.addEventListener("input", () => {
    readLogoSpan();
    regenerate();
  });

  logoScaleEl.addEventListener("input", () => {
    state.logoScale = Number(logoScaleEl.value);
    logoScaleOut.textContent = logoScaleEl.value;
    regenerate();
  });

  $("#logo-clear").addEventListener("click", () => {
    state.logo = null;
    state.logoModules = 0;
    state.logoScale = 80;
    logoInput.value = "";
    logoName.textContent = "none";
    logoModulesEl.value = logoModulesEl.max;
    logoModulesOut.textContent = "max";
    logoScaleEl.value = "80";
    logoScaleOut.textContent = "80";
    logoExtrasEl.classList.add("hidden");
    logoSpanRowEl.classList.add("hidden");
    logoScaleRowEl.classList.add("hidden");
    regenerate();
  });

  const pngBlob = (): Blob | null => {
    if (!lastPayload) return null;
    const result = qrgo.generate(options("png"));
    if (isError(result)) {
      warningsEl.textContent = result.error;
      return null;
    }
    return new Blob([result.data as Uint8Array<ArrayBuffer>], { type: "image/png" });
  };

  const dlBtn = $<HTMLButtonElement>("#download");
  const cpBtn = $<HTMLButtonElement>("#copy");
  for (const radio of document.querySelectorAll<HTMLInputElement>("#dl-format input")) {
    radio.addEventListener("change", syncExportLabels);
  }

  dlBtn.addEventListener("click", () => {
    const blob =
      exportFormat() === "png"
        ? pngBlob()
        : lastResult && new Blob([lastResult.data as string], { type: "image/svg+xml" });
    if (!blob) return;
    download(blob, `${slug()}.${exportFormat()}`);
    receipt(dlBtn, "Saved");
  });

  cpBtn.addEventListener("click", async () => {
    try {
      if (exportFormat() === "png") {
        const blob = pngBlob();
        if (!blob) return;
        await navigator.clipboard.write([new ClipboardItem({ "image/png": blob })]);
      } else {
        if (!lastResult) return;
        await navigator.clipboard.writeText(lastResult.data as string);
      }
      receipt(cpBtn, "Copied");
    } catch {
      warningsEl.textContent = "Clipboard blocked by the browser; download instead";
    }
  });

  const copyBtn = $<HTMLButtonElement>("#payload-copy");
  copyBtn.addEventListener("click", async () => {
    if (!lastPayload) return;
    try {
      await navigator.clipboard.writeText(lastPayload);
      copyBtn.textContent = "Copied";
      setTimeout(() => (copyBtn.textContent = "Copy payload"), 1200);
    } catch {
      copyBtn.textContent = "Copy failed";
      setTimeout(() => (copyBtn.textContent = "Copy payload"), 1200);
    }
  });
}

// Opening a fold scrolls its header to the top of the viewport so the whole
// fold ends up visible instead of expanding below the page fold. The scroll is
// driven by hand, frame by frame, alongside the 280ms height animation: a
// single smooth scrollIntoView would be clamped to the document height at
// toggle time, before the fold has grown, and stop short. The header's
// document position never moves (growth happens below it), so the target is
// computed once; per-frame scrollTo re-clamps as the document grows.
function wireFoldScroll(): void {
  const easeOut = (t: number): number => 1 - (1 - t) ** 3;
  for (const details of document.querySelectorAll<HTMLDetailsElement>(
    "details.advanced, details.payload",
  )) {
    details.addEventListener("toggle", () => {
      if (!details.open) return;
      if (matchMedia("(prefers-reduced-motion: reduce)").matches) {
        details.scrollIntoView({ block: "start" }); // no transitions: doc already grown
        return;
      }
      const from = window.scrollY;
      const target = details.getBoundingClientRect().top + from - 16;
      const t0 = performance.now();
      const step = (): void => {
        const t = Math.min((performance.now() - t0) / 320, 1);
        window.scrollTo({ top: from + (target - from) * easeOut(t), behavior: "instant" });
        if (t < 1) requestAnimationFrame(step);
      };
      requestAnimationFrame(step);
    });
  }
}

// --- boot -------------------------------------------------------------------

drawWordmark(document.getElementById("wordmark") as unknown as SVGElement, "QR·GO");
buildTabs();
buildForm(true); // land ready to type
wireStyleControls();
wireFoldScroll();
syncFromDom();
setDownloadsEnabled(false);

// bfcache restores (back button) revive old control values after boot.
window.addEventListener("pageshow", (e) => {
  if (!e.persisted) return;
  syncFromDom();
  if (document.body.classList.contains("ready")) render();
});

try {
  await initQrgo();
} catch {
  showEmpty("The print engine failed to load. Refresh to try again.", true);
  throw new Error("qrgo wasm failed to load");
}
wasmReady = true;
document.body.classList.add("ready");
render();
