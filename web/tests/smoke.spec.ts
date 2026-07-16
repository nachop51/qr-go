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

test("pixel font, wordmark, and icons survive the production CSP", async ({ page }) => {
  await page.goto("/");
  await page.waitForFunction(() => document.body.classList.contains("ready"));

  // The font must actually load; under the dist server the production CSP is
  // enforced, so a data:-inlined font (blocked by font-src 'self') fails here.
  await page.waitForFunction(() => document.fonts.check('16px "Departure Mono"'));

  // Every wordmark pixel must be visible once the stamp stagger is over.
  // Guards the WebKit bug where batches of delayed per-rect animations stayed
  // stuck in their opacity:0 fill state (garbled logo on iOS).
  await page.waitForTimeout(1500);
  const wordmark = await page.evaluate(() => {
    const rects = [...document.querySelectorAll("#wordmark rect")];
    return {
      total: rects.length,
      hidden: rects.filter((r) => getComputedStyle(r).opacity !== "1").length,
    };
  });
  expect(wordmark.total).toBeGreaterThan(0);
  expect(wordmark.hidden).toBe(0);

  // Icon links (incl. the PNG fallbacks Safari needs) must resolve.
  const iconHrefs = await page.evaluate(() =>
    [...document.querySelectorAll('link[rel="icon"], link[rel="apple-touch-icon"]')].map(
      (l) => (l as HTMLLinkElement).href,
    ),
  );
  expect(iconHrefs.length).toBeGreaterThanOrEqual(3);
  for (const href of iconHrefs) {
    const res = await page.request.get(href);
    expect(res.status(), href).toBe(200);
  }
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
