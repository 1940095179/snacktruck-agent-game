export default {
  async fetch(request, env) {
    const origin = env.ORIGIN_URL;
    if (!origin) {
      return new Response(JSON.stringify({ error: { code: 'CONFIG_ERROR', message: 'ORIGIN_URL is required' } }), {
        status: 500,
        headers: { 'content-type': 'application/json; charset=utf-8' }
      });
    }

    const target = new URL(request.url);
    const upstream = new URL(origin);
    upstream.pathname = target.pathname;
    upstream.search = target.search;

    const proxyReq = new Request(upstream.toString(), request);
    const resp = await fetch(proxyReq, {
      cf: { cacheTtl: 0, cacheEverything: false }
    });

    const headers = new Headers(resp.headers);
    headers.set('access-control-allow-origin', '*');
    headers.set('access-control-allow-methods', 'GET,POST,OPTIONS');
    headers.set('access-control-allow-headers', 'Content-Type');

    if (request.method === 'OPTIONS') {
      return new Response(null, { status: 204, headers });
    }

    return new Response(resp.body, {
      status: resp.status,
      headers
    });
  }
};
