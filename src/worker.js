/**
 * Cloudflare Workers entrypoint for myjournal-app.
 * Proxies application and health check traffic to the upstream
 * application service, attaching edge deployment diagnostic headers.
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

		// Resolve upstream application target.
		const upstreamRaw = env.UPSTREAM_URL;
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

		// Forward client host and protocol information while preserving
		// the canonical host expected by RequestAuthority and canonicalGoogleLogin.
		let canonicalHost = url.host;
		if (env.APP_BASE_URL) {
			try {
				canonicalHost = new URL(env.APP_BASE_URL).host;
			} catch (_) {
				canonicalHost = url.host;
			}
		}

		const headers = new Headers(request.headers);
		headers.set("Host", canonicalHost);
		headers.set("X-Forwarded-Host", url.host);
		headers.set("X-Forwarded-Proto", url.protocol.replace(":", ""));

		const init = {
			method: request.method,
			headers,
			redirect: "manual"
		};

		if (request.method !== "GET" && request.method !== "HEAD") {
			init.body = request.body;
		}

		try {
			const response = await fetch(targetUrl.toString(), init);

			// For /healthz, preserve the upstream JSON response contract while attaching edge deployment headers.
			if (url.pathname === "/healthz") {
				const healthHeaders = new Headers(response.headers);
				healthHeaders.set("X-Deploy-Target", env.DEPLOY_TARGET || "cloudflare");
				healthHeaders.set("X-Cache-Storage", env.CACHE_STORAGE || "lru");
				healthHeaders.set("X-Base-URL", env.APP_BASE_URL || "https://myjournal.ink");
				return new Response(response.body, {
					status: response.status,
					statusText: response.statusText,
					headers: healthHeaders
				});
			}

			return response;
		} catch (err) {
			return new Response(`Upstream connection failed: ${err.message}`, {
				status: 502,
				headers: { "Content-Type": "text/plain; charset=utf-8" }
			});
		}
	}
};
