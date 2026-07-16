const root = `${import.meta.dir}/../dist`;

// Apply the site-wide (`/*`) headers from _headers so smoke tests run under
// the same CSP as production; a font or script the CSP would block in prod
// must also be blocked here.
const siteHeaders: Record<string, string> = {};
{
  const text = await Bun.file(`${root}/_headers`).text();
  let inGlobal = false;
  for (const line of text.split("\n")) {
    if (/^\S/.test(line)) {
      inGlobal = line.trim() === "/*";
      continue;
    }
    if (!inGlobal) continue;
    const m = line.match(/^\s+([\w-]+):\s*(.+)$/);
    if (m) siteHeaders[m[1]] = m[2];
  }
}

Bun.serve({
  hostname: "0.0.0.0",
  port: Number(process.env.PORT ?? 4173),
  async fetch(request) {
    const url = new URL(request.url);
    let pathname: string;
    try {
      pathname = decodeURIComponent(url.pathname);
    } catch {
      return new Response("Bad request", { status: 400 });
    }
    if (pathname === "/") pathname = "/index.html";
    if (pathname.includes("\0") || pathname.split("/").includes("..")) {
      return new Response("Bad request", { status: 400 });
    }

    const file = Bun.file(`${root}${pathname}`);
    if (!(await file.exists())) return new Response("Not found", { status: 404 });
    return new Response(file, { headers: siteHeaders });
  },
});
