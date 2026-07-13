const root = `${import.meta.dir}/../dist`;

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
    return new Response(file);
  },
});
