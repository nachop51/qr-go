import { initQrgo, isError, type GenerateOptions, type GenerateResult } from "./qrgo";
import { CONTENT_TYPES, FIELDS, type Field, type Values } from "./forms";

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
  dark: string;
  light: string;
  quiet: number;
  pngSize: number;
  logo: Uint8Array | null;
  logoModules: number; // 0 = auto
}

const state: State = {
  type: "text",
  values: Object.fromEntries(Object.keys(FIELDS).map((k) => [k, {}])),
  ecLevel: "M",
  dark: "#1a1a1e",
  light: "#ffffff",
  quiet: 4,
  pngSize: 800,
  logo: null,
  logoModules: 0,
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
const logoModulesEl = $<HTMLInputElement>("#logo-modules");
const logoModulesOut = $("#logo-modules-out");

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
  formEl.innerHTML = "";
  const values = state.values[state.type];
  for (const f of FIELDS[state.type]) {
    const row = document.createElement("div");
    row.className = f.kind === "checkbox" ? "field-row check" : "field-row";

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

    input.addEventListener("input", () => {
      values[f.key] =
        f.kind === "checkbox" ? String((input as HTMLInputElement).checked) : input.value;
      regenerate();
    });

    if (f.kind === "checkbox") row.append(input, label);
    else row.append(label, input);
    formEl.append(row);
  }
  if (focusField) (formEl.querySelector("input, textarea") as HTMLElement | null)?.focus();
}

// --- generation -------------------------------------------------------------

function options(format: "png" | "svg"): GenerateOptions {
  return {
    text: lastPayload,
    ecLevel: state.ecLevel,
    format,
    dark: state.dark,
    light: state.light,
    quiet: state.quiet,
    ...(format === "png" ? { size: state.pngSize } : {}),
    ...(state.logo ? { logo: state.logo, logoModules: state.logoModules } : {}),
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
  ($("#dl-png") as HTMLButtonElement).disabled = !on;
  ($("#dl-svg") as HTMLButtonElement).disabled = !on;
}

let timer: ReturnType<typeof setTimeout> | undefined;
function regenerate(): void {
  clearTimeout(timer);
  timer = setTimeout(render, 120);
}

function render(): void {
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

  const version = (result.size - 17) / 4;
  const bytes = new TextEncoder().encode(payload).length;
  specEl.textContent = `${result.size}×${result.size} modules · v${version} · EC ${state.ecLevel} · ${bytes} bytes`;
  warningsEl.textContent = (result.warnings ?? []).join(" · ");
  payloadEl.textContent = payload;

  logoModulesEl.max = String(result.maxLogoModules);
  setDownloadsEnabled(true);
}

// The logo only ever spans a fraction of the code, so a phone photo's full
// resolution is pure overhead: every regeneration would decode it in wasm and
// the SVG renderer would embed it as a data URI. Shrink it once, up front.
async function downscaleLogo(file: File, maxDim = 512): Promise<Uint8Array> {
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

// --- wiring -----------------------------------------------------------------

function wireStyleControls(): void {
  for (const radio of document.querySelectorAll<HTMLInputElement>(
    '#ec-levels input[name="ecLevel"]',
  )) {
    radio.addEventListener("change", () => {
      state.ecLevel = radio.value as State["ecLevel"];
      $("#ec-hint").textContent = EC_HINTS[state.ecLevel];
      regenerate();
    });
  }

  const dark = $<HTMLInputElement>("#dark");
  const light = $<HTMLInputElement>("#light");
  dark.addEventListener("input", () => ((state.dark = dark.value), regenerate()));
  light.addEventListener("input", () => ((state.light = light.value), regenerate()));

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
  logoInput.addEventListener("change", async () => {
    const file = logoInput.files?.[0];
    if (!file) return;
    state.logo = await downscaleLogo(file);
    logoExtrasEl.classList.remove("hidden");
    regenerate();
  });

  logoModulesEl.addEventListener("input", () => {
    state.logoModules = Number(logoModulesEl.value);
    logoModulesOut.textContent = state.logoModules === 0 ? "auto" : `${state.logoModules} modules`;
    regenerate();
  });

  $("#logo-clear").addEventListener("click", () => {
    state.logo = null;
    state.logoModules = 0;
    logoInput.value = "";
    logoModulesEl.value = "0";
    logoModulesOut.textContent = "auto";
    logoExtrasEl.classList.add("hidden");
    regenerate();
  });

  $("#dl-png").addEventListener("click", () => {
    if (!lastPayload) return;
    const result = qrgo.generate(options("png"));
    if (isError(result)) {
      warningsEl.textContent = result.error;
      return;
    }
    download(
      new Blob([result.data as Uint8Array<ArrayBuffer>], { type: "image/png" }),
      `${slug()}.png`,
    );
  });

  $("#dl-svg").addEventListener("click", () => {
    if (!lastResult) return;
    download(new Blob([lastResult.data as string], { type: "image/svg+xml" }), `${slug()}.svg`);
  });
}

// --- boot -------------------------------------------------------------------

drawWordmark(document.getElementById("wordmark") as unknown as SVGElement, "QR·GO");
buildTabs();
buildForm(true); // land ready to type
wireStyleControls();
setDownloadsEnabled(false);

await initQrgo();
document.body.classList.add("ready");
render();
