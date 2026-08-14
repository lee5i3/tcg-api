// Fully static SPA: no server rendering, no prerendered routes. The
// adapter-static `fallback: "index.html"` shell handles every URL and all
// data is fetched client-side from the REST API.
export const ssr = false;
export const prerender = false;
export const csr = true;
