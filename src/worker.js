/**
 * Cloudflare Workers entrypoint for myjournal-app.
 * Routes edge health checks directly and proxies application traffic
 * to the upstream application service.
 *
 * @module worker
 */

/**
 * Handle incoming HTTP requests at the edge.
 *
 * @param {Request} request - The incoming HTTP request.
 * @param {Record<string, string>} env - Bound environment variables and secrets.
 * @param {ExecutionContext} ctx - Execution context for background tasks.
 * @returns {Promise<Response>} The HTTP response.
 */
export default {
	async fetch(request, env, ctx) {
		const url = new URL(request.url);

		// Handle health checks directly at the edge.
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

		// Resolve upstream application target.
		const upstreamRaw = env.UPSTREAM_URL || "https://myjournal-app-production.up.railway.app";
		if (!upstreamRaw) {
			return new Response("Upstream application not configured", {
				status: 502,
				headers: { "Content-Type": "text/plain; charset=utf-8" }
			});
		}

		let upstreamUrl;
		try {
			upstreamUrl = new URL(upstreamRaw);
		} catch (err) {
			return new Response("Invalid upstream application URL", {
				status: 502,
				headers: { "Content-Type": "text/plain; charset=utf-8" }
			});
		}

		// Rewrite URL to upstream origin while preserving path and query.
		const targetUrl = new URL(request.url);
		targetUrl.protocol = upstreamUrl.protocol;
		targetUrl.hostname = upstreamUrl.hostname;
		targetUrl.port = upstreamUrl.port;

		// Forward client host and protocol information.
		const headers = new Headers(request.headers);
		headers.set("X-Forwarded-Host", url.host);
		headers.set("X-Forwarded-Proto", url.protocol.replace(":", ""));
		headers.delete("host");

		const init = {
			method: request.method,
			headers,
			redirect: "manual"
		};

		if (request.method !== "GET" && request.method !== "HEAD") {
			init.body = request.body;
		}

		try {
			return await fetch(targetUrl.toString(), init);
		} catch (err) {
			return new Response(`Upstream connection failed: ${err.message}`, {
				status: 502,
				headers: { "Content-Type": "text/plain; charset=utf-8" }
			});
		}
	}
};
