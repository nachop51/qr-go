import { expect, test } from "@playwright/test";

test("WASM previews, exports, and accepts every advertised logo format", async ({ page }) => {
  await page.goto("/");
  await page.waitForFunction(() => typeof globalThis.qrgo?.generate === "function");

  const results = await page.evaluate(async () => {
    const raster = async (type: string) => {
      const canvas = document.createElement("canvas");
      canvas.width = canvas.height = 8;
      canvas.getContext("2d")!.fillRect(0, 0, 8, 8);
      const blob = await new Promise<Blob>((resolve) => canvas.toBlob((b) => resolve(b!), type));
      return new Uint8Array(await blob.arrayBuffer());
    };
    const logos = [
      await raster("image/png"),
      await raster("image/jpeg"),
      await raster("image/webp"),
      Uint8Array.from(atob("R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw=="), (c) => c.charCodeAt(0)),
      new TextEncoder().encode(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 8 8"><rect width="8" height="8"/></svg>`),
    ];
    const generated = logos.map((logo) => globalThis.qrgo.generate({ text: "smoke", format: "svg", logo }));
    const png = globalThis.qrgo.generate({ text: "smoke", format: "png", size: 256 });
    const malformed = globalThis.qrgo.generate({ text: "smoke", format: "svg", logo: new Uint8Array([1, 2, 3]) });
    return {
      allLogos: generated.every((r) => !("error" in r) && typeof r.data === "string"),
      png: !("error" in png) && png.data instanceof Uint8Array && png.data[0] === 0x89,
      malformed: "error" in malformed,
    };
  });
  expect(results).toEqual({ allLogos: true, png: true, malformed: true });

  await page.locator("textarea").first().fill("browser preview");
  await expect(page.locator("#preview img")).toBeVisible();
  await expect(page.locator("#download")).toBeEnabled();

  const pngDownload = page.waitForEvent("download");
  await page.locator("#download").click();
  expect((await pngDownload).suggestedFilename()).toMatch(/\.png$/);

  await page.locator('#dl-format input[value="svg"]').check();
  const svgDownload = page.waitForEvent("download");
  await page.locator("#download").click();
  expect((await svgDownload).suggestedFilename()).toMatch(/\.svg$/);
});

test("preview stays contained inside its ticket regardless of render element", async ({ page }) => {
  await page.goto("/");
  await page.waitForFunction(() => typeof globalThis.qrgo?.generate === "function");

  await page.locator("textarea").first().fill("containment check");
  const rendered = page.locator("#preview img, #preview svg").first();
  await expect(rendered).toBeVisible();

  const container = await page.locator("#preview").boundingBox();
  const content = await rendered.boundingBox();
  expect(container).not.toBeNull();
  expect(content).not.toBeNull();
  expect(content!.width).toBeLessThanOrEqual(container!.width + 1);
  expect(content!.height).toBeLessThanOrEqual(container!.height + 1);
  expect(content!.x).toBeGreaterThanOrEqual(container!.x - 1);
  expect(content!.y).toBeGreaterThanOrEqual(container!.y - 1);

  // The page itself must never scroll horizontally because of the preview.
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
  );
  expect(overflow).toBeLessThanOrEqual(0);
});
