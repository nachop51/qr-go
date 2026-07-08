// Typed bridge to the Go wasm module. cmd/wasm/main.go sets a `qrgo` global
// with generate() and the content payload helpers.
import "./vendor/wasm_exec.js";
// Bun's file loader copies the wasm binary into the bundle and returns its URL.
// @ts-expect-error resolved by Bun at build time
import wasmPath from "../qr.wasm" with { type: "file" };

export interface GenerateOptions {
  text: string;
  ecLevel?: "L" | "M" | "Q" | "H";
  eciPolicy?: "auto" | "disabled";
  format?: "png" | "svg";
  dark?: string;
  light?: string;
  quiet?: number;
  size?: number;
  moduleSize?: number;
  logo?: Uint8Array;
  logoModules?: number; // omit or 0 = max the EC level allows
  version?: number; // 1-40; omit = auto
  mask?: number; // 0-7; omit = auto
}

export interface GenerateResult {
  data: Uint8Array | string;
  size: number;
  version: number;
  mask: number;
  maxLogoModules: number;
  warnings: string[];
  error?: undefined;
}

export interface QrgoError {
  error: string;
}

type ContentResult = string | QrgoError;

interface QrgoAPI {
  generate(opts: GenerateOptions): GenerateResult | QrgoError;
  content: {
    wifi(o: { ssid: string; pass?: string; auth?: string; hidden?: boolean }): ContentResult;
    vcard(o: Record<string, string>): ContentResult;
    event(o: Record<string, string>): ContentResult;
    url(o: { url: string }): ContentResult;
    tel(o: { number: string }): ContentResult;
    sms(o: { number: string; message?: string }): ContentResult;
    geo(o: { lat: number; lng: number }): ContentResult;
    email(o: { to: string; subject?: string; body?: string }): ContentResult;
  };
}

declare global {
  // Declared by vendor/wasm_exec.js (copied from the Go toolchain by `make wasm`).
  var Go: new () => { importObject: WebAssembly.Imports; run(i: WebAssembly.Instance): Promise<void> };
  var qrgo: QrgoAPI;
}

export async function initQrgo(): Promise<QrgoAPI> {
  const go = new Go();
  const source = fetch(wasmPath as string);
  let instance: WebAssembly.Instance;
  try {
    ({ instance } = await WebAssembly.instantiateStreaming(source, go.importObject));
  } catch {
    // Fallback for servers that don't send application/wasm.
    const bytes = await (await fetch(wasmPath as string)).arrayBuffer();
    ({ instance } = await WebAssembly.instantiate(bytes, go.importObject));
  }
  void go.run(instance); // never resolves: the Go side blocks forever on purpose
  return globalThis.qrgo;
}

export function isError(r: ContentResult | GenerateResult | QrgoError): r is QrgoError {
  return typeof r === "object" && r !== null && "error" in r && r.error !== undefined;
}
