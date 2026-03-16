import { env } from "$env/dynamic/public";
import type { CatalogFilters, CEFRLevel, Resource } from "$lib/types";

type ResourcesAPIResponse = {
	items?: Resource[];
	count?: number;
};

type APIErrorResponse = {
	message?: string;
};

type SaveResponse = {
	saved?: boolean;
	created?: boolean;
	removed?: boolean;
};

export type ListResourcesResult = {
	items: Resource[];
	count: number;
	error: string | null;
};

export type ResourceResult = {
	item: Resource | null;
	error: string | null;
	notFound: boolean;
};

export type SaveToggleResult = {
	ok: boolean;
	error: string | null;
	created?: boolean;
	removed?: boolean;
};

export const levelOptions: CEFRLevel[] = ["A1", "A2", "B1", "B2", "C1", "C2"];

const API_BASE_URL = env.PUBLIC_API_BASE_URL || "";
const DEFAULT_USER_ID = "11111111-1111-1111-1111-111111111111";
const USER_ID = (env.PUBLIC_USER_ID || DEFAULT_USER_ID).trim();

export async function listResources(
	loadFetch: typeof fetch,
	filters: CatalogFilters
): Promise<ListResourcesResult> {
	try {
		const query = new URLSearchParams();
		if (filters.level) {
			query.set("level", filters.level);
		}
		if (filters.skill) {
			query.set("skill", filters.skill);
		}
		if (filters.topic) {
			query.set("topic", filters.topic);
		}
		if (filters.query) {
			query.set("q", filters.query);
		}
		if (filters.free !== null) {
			query.set("free", String(filters.free));
		}

		const response = await loadFetch(apiURL(`/api/v1/resources${toQueryString(query)}`), {
			headers: buildHeaders({ includeUser: true })
		});

		if (!response.ok) {
			return {
				items: [],
				count: 0,
				error: await responseError(response, `Could not load resources (${response.status})`)
			};
		}

		const payload = (await response.json()) as ResourcesAPIResponse;
		const items = Array.isArray(payload.items) ? payload.items : [];
		const count = typeof payload.count === "number" ? payload.count : items.length;

		return {
			items,
			count,
			error: null
		};
	} catch {
		return {
			items: [],
			count: 0,
			error: "Could not load resources right now. Please try again."
		};
	}
}

export async function getResourceBySlug(loadFetch: typeof fetch, slug: string): Promise<ResourceResult> {
	try {
		const response = await loadFetch(apiURL(`/api/v1/resources/${slug}`), {
			headers: buildHeaders({ includeUser: true })
		});

		if (response.status === 404) {
			return {
				item: null,
				error: "Resource not found",
				notFound: true
			};
		}

		if (!response.ok) {
			return {
				item: null,
				error: await responseError(response, `Could not load resource (${response.status})`),
				notFound: false
			};
		}

		return {
			item: (await response.json()) as Resource,
			error: null,
			notFound: false
		};
	} catch {
		return {
			item: null,
			error: "Could not load resource right now. Please try again.",
			notFound: false
		};
	}
}

export async function listSavedResources(loadFetch: typeof fetch): Promise<ListResourcesResult> {
	if (!USER_ID) {
		return {
			items: [],
			count: 0,
			error: "Saved resources are unavailable because no user id is configured."
		};
	}

	try {
		const response = await loadFetch(apiURL("/api/v1/me/saved-resources"), {
			headers: buildHeaders({ includeUser: true })
		});

		if (!response.ok) {
			return {
				items: [],
				count: 0,
				error: await responseError(response, `Could not load saved resources (${response.status})`)
			};
		}

		const payload = (await response.json()) as ResourcesAPIResponse;
		const items = Array.isArray(payload.items) ? payload.items : [];
		const count = typeof payload.count === "number" ? payload.count : items.length;

		return {
			items,
			count,
			error: null
		};
	} catch {
		return {
			items: [],
			count: 0,
			error: "Could not load saved resources right now. Please try again."
		};
	}
}

export async function saveResource(loadFetch: typeof fetch, resourceID: string): Promise<SaveToggleResult> {
	if (!USER_ID) {
		return { ok: false, error: "No user id configured." };
	}

	try {
		const response = await loadFetch(apiURL("/api/v1/me/saved-resources"), {
			method: "POST",
			headers: buildHeaders({ includeUser: true, includeJSON: true }),
			body: JSON.stringify({ resourceId: resourceID })
		});

		if (!response.ok) {
			return {
				ok: false,
				error: await responseError(response, `Could not save resource (${response.status})`)
			};
		}

		const payload = (await response.json()) as SaveResponse;
		return {
			ok: true,
			error: null,
			created: Boolean(payload.created)
		};
	} catch {
		return {
			ok: false,
			error: "Could not save resource right now. Please try again."
		};
	}
}

export async function removeSavedResource(loadFetch: typeof fetch, resourceID: string): Promise<SaveToggleResult> {
	if (!USER_ID) {
		return { ok: false, error: "No user id configured." };
	}

	try {
		const response = await loadFetch(apiURL(`/api/v1/me/saved-resources/${resourceID}`), {
			method: "DELETE",
			headers: buildHeaders({ includeUser: true })
		});

		if (!response.ok) {
			return {
				ok: false,
				error: await responseError(response, `Could not remove saved resource (${response.status})`)
			};
		}

		const payload = (await response.json()) as SaveResponse;
		return {
			ok: true,
			error: null,
			removed: Boolean(payload.removed)
		};
	} catch {
		return {
			ok: false,
			error: "Could not remove saved resource right now. Please try again."
		};
	}
}

export function deriveFilterOptions(resources: Resource[]) {
	return {
		skills: uniqueValues(resources.flatMap((resource) => resource.skillTags)),
		topics: uniqueValues(resources.flatMap((resource) => resource.topicTags))
	};
}

export function getCurrentUserID() {
	return USER_ID;
}

function buildHeaders(options: { includeUser: boolean; includeJSON?: boolean }): HeadersInit {
	const headers: Record<string, string> = {
		Accept: "application/json"
	};

	if (options.includeJSON) {
		headers["Content-Type"] = "application/json";
	}

	if (options.includeUser && USER_ID) {
		headers["X-User-ID"] = USER_ID;
	}

	return headers;
}

async function responseError(response: Response, fallback: string): Promise<string> {
	try {
		const payload = (await response.json()) as APIErrorResponse;
		if (typeof payload.message === "string" && payload.message.trim() !== "") {
			return payload.message;
		}
	} catch {
		// ignore parsing error and use fallback
	}

	return fallback;
}

function toQueryString(params: URLSearchParams) {
	const query = params.toString();
	return query ? `?${query}` : "";
}

function apiURL(path: string) {
	return API_BASE_URL ? `${API_BASE_URL}${path}` : path;
}

function uniqueValues(values: string[]) {
	return Array.from(new Set(values)).sort((left, right) => left.localeCompare(right));
}
