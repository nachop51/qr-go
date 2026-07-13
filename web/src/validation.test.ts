import { describe, expect, test } from "bun:test";
import { coordinatesAreValid, MAX_BROWSER_PNG_EDGE, MAX_LOGO_BYTES } from "./validation";

describe("browser boundaries", () => {
  test("coordinate ranges are finite", () => {
    expect(coordinatesAreValid(-90, 180)).toBe(true);
    expect(coordinatesAreValid(Number.NaN, 0)).toBe(false);
    expect(coordinatesAreValid(0, 181)).toBe(false);
  });
  test("resource limits stay pinned", () => {
    expect(MAX_LOGO_BYTES).toBe(16 * 1024 * 1024);
    expect(MAX_BROWSER_PNG_EDGE).toBe(4096);
  });
});
