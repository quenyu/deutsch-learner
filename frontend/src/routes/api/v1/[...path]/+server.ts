import { env } from "$env/dynamic/private";
import type { RequestHandler } from "./$types";

const defaultProxyTarget = "http://localhost:8080";

function getProxyTargetBase() {
	return (env.API_PROXY_TARGET || env.BACKEND_API_BASE_URL || defaultProxyTarget).replace(/\/+$/, "");
}

async function proxyRequest(event: Parameters<RequestHandler>[0]) {
	const { fetch, request, params, url } = event;
	const targetURL = `${getProxyTargetBase()}/api/v1/${params.path ?? ""}${url.search}`;

	const headers = new Headers(request.headers);
	headers.delete("host");
	headers.delete("connection");
	headers.delete("content-length");

	const init: RequestInit = {
		method: request.method,
		headers,
		redirect: "manual"
	};

	if (request.method !== "GET" && request.method !== "HEAD") {
		const body = await request.arrayBuffer();
		if (body.byteLength > 0) {
			init.body = body;
		}
	}

	try {
		const upstream = await fetch(targetURL, init);
		const responseHeaders = new Headers(upstream.headers);
		responseHeaders.delete("content-length");
		responseHeaders.delete("connection");
		responseHeaders.delete("transfer-encoding");

		return new Response(upstream.body, {
			status: upstream.status,
			headers: responseHeaders
		});
	} catch {
		return new Response(JSON.stringify({ message: "backend unavailable" }), {
			status: 502,
			headers: {
				"content-type": "application/json"
			}
		});
	}
}

export const GET: RequestHandler = proxyRequest;
export const POST: RequestHandler = proxyRequest;
export const PUT: RequestHandler = proxyRequest;
export const PATCH: RequestHandler = proxyRequest;
export const DELETE: RequestHandler = proxyRequest;
export const OPTIONS: RequestHandler = proxyRequest;
