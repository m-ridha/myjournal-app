// Cloudflare Workers fetch entrypoint for myjournal-app.
export default {
	async fetch(request, env, ctx) {
		const url = new URL(request.url);
		if (url.pathname === "/healthz") {
			return new Response("OK", {
				status: 200,
				headers: {
					"Content-Type": "text/plain; charset=utf-8",
					"X-Deploy-Target": env.DEPLOY_TARGET || "cloudflare",
					"X-Cache-Storage": env.CACHE_STORAGE || "lru",
					"X-Base-URL": env.APP_BASE_URL || "https://myjournal.ink"
				}
			});
		}
		return new Response(JSON.stringify({
			status: "ok",
			target: env.DEPLOY_TARGET || "cloudflare",
			cache: env.CACHE_STORAGE || "lru",
			base_url: env.APP_BASE_URL || "https://myjournal.ink",
			url: request.url
		}), {
			status: 200,
			headers: { "Content-Type": "application/json" }
		});
	}
};
