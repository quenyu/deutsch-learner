import { env } from "$env/dynamic/public";
import type {
	CatalogFilters,
	CEFRLevel,
	ProgressStatus,
	Resource,
	ResourceProgress,
	ResourceType,
	SourceProvider,
	UserProfile
} from "$lib/types";

type ResourcesAPIResponse = {
	items?: Resource[];
	count?: number;
};

type SourceProvidersAPIResponse = {
	items?: SourceProvider[];
	count?: number;
};

type ProgressAPIResponse = {
	items?: ResourceProgress[];
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

type ProfileAPIResponse = {
	userId: string;
	displayName: string;
	targetLevel?: string | null;
	learningGoals?: string;
	preferredResourceTypes?: string[];
	preferredSkills?: string[];
	preferredSourceProviders?: string[];
	updatedAt?: string;
};

type ProgressUpdateResponse = ResourceProgress;

export type ListResourcesResult = {
	items: Resource[];
	count: number;
	error: string | null;
};

export type SourceProvidersResult = {
	items: SourceProvider[];
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

export type ProfileResult = {
	item: UserProfile | null;
	error: string | null;
};

export type ProfileUpdateResult = {
	ok: boolean;
	item: UserProfile | null;
	error: string | null;
};

export type ProgressListResult = {
	items: ResourceProgress[];
	count: number;
	error: string | null;
};

export type ProgressResult = {
	item: ResourceProgress | null;
	error: string | null;
};

export type ProgressUpdateResult = {
	ok: boolean;
	item: ResourceProgress | null;
	error: string | null;
};

export type ProfileUpdateInput = {
	displayName: string;
	targetLevel: CEFRLevel | null;
	learningGoals: string;
	preferredResourceTypes: string[];
	preferredSkills: string[];
	preferredSourceProviders: string[];
};

export const levelOptions: CEFRLevel[] = ["A1", "A2", "B1", "B2", "C1", "C2"];
export const resourceTypeOptions: ResourceType[] = [
	"youtube",
	"article",
	"playlist",
	"course",
	"podcast",
	"grammar_reference",
	"exercise"
];

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
		if (filters.provider) {
			query.set("provider", filters.provider);
		}
		if (filters.type) {
			query.set("type", filters.type);
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

export async function listSourceProviders(loadFetch: typeof fetch): Promise<SourceProvidersResult> {
	try {
		const response = await loadFetch(apiURL("/api/v1/source-providers"), {
			headers: buildHeaders({ includeUser: false })
		});
		if (!response.ok) {
			return {
				items: [],
				count: 0,
				error: await responseError(response, `Could not load source providers (${response.status})`)
			};
		}

		const payload = (await response.json()) as SourceProvidersAPIResponse;
		const items = Array.isArray(payload.items) ? payload.items : [];
		const count = typeof payload.count === "number" ? payload.count : items.length;
		return { items, count, error: null };
	} catch {
		return {
			items: [],
			count: 0,
			error: "Could not load source providers right now. Please try again."
		};
	}
}

export async function getProfile(loadFetch: typeof fetch): Promise<ProfileResult> {
	if (!USER_ID) {
		return {
			item: null,
			error: "Profile is unavailable because no user id is configured."
		};
	}

	try {
		const response = await loadFetch(apiURL("/api/v1/me/profile"), {
			headers: buildHeaders({ includeUser: true })
		});

		if (!response.ok) {
			return {
				item: null,
				error: await responseError(response, `Could not load profile (${response.status})`)
			};
		}

		const payload = (await response.json()) as ProfileAPIResponse;
		return {
			item: normalizeProfile(payload),
			error: null
		};
	} catch {
		return {
			item: null,
			error: "Could not load profile right now. Please try again."
		};
	}
}

export async function updateProfile(
	loadFetch: typeof fetch,
	input: ProfileUpdateInput
): Promise<ProfileUpdateResult> {
	if (!USER_ID) {
		return { ok: false, item: null, error: "No user id configured." };
	}

	try {
		const response = await loadFetch(apiURL("/api/v1/me/profile"), {
			method: "PUT",
			headers: buildHeaders({ includeUser: true, includeJSON: true }),
			body: JSON.stringify({
				displayName: input.displayName,
				targetLevel: input.targetLevel,
				learningGoals: input.learningGoals,
				preferredResourceTypes: input.preferredResourceTypes,
				preferredSkills: input.preferredSkills,
				preferredSourceProviders: input.preferredSourceProviders
			})
		});

		if (!response.ok) {
			return {
				ok: false,
				item: null,
				error: await responseError(response, `Could not update profile (${response.status})`)
			};
		}

		const payload = (await response.json()) as ProfileAPIResponse;
		return {
			ok: true,
			item: normalizeProfile(payload),
			error: null
		};
	} catch {
		return {
			ok: false,
			item: null,
			error: "Could not update profile right now. Please try again."
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

export async function listProgress(loadFetch: typeof fetch): Promise<ProgressListResult> {
	if (!USER_ID) {
		return {
			items: [],
			count: 0,
			error: "Progress is unavailable because no user id is configured."
		};
	}

	try {
		const response = await loadFetch(apiURL("/api/v1/me/progress"), {
			headers: buildHeaders({ includeUser: true })
		});

		if (!response.ok) {
			return {
				items: [],
				count: 0,
				error: await responseError(response, `Could not load progress (${response.status})`)
			};
		}

		const payload = (await response.json()) as ProgressAPIResponse;
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
			error: "Could not load progress right now. Please try again."
		};
	}
}

export async function getProgressForResource(loadFetch: typeof fetch, resourceID: string): Promise<ProgressResult> {
	if (!USER_ID) {
		return {
			item: null,
			error: "Progress is unavailable because no user id is configured."
		};
	}

	try {
		const response = await loadFetch(apiURL(`/api/v1/me/progress/${resourceID}`), {
			headers: buildHeaders({ includeUser: true })
		});

		if (!response.ok) {
			return {
				item: null,
				error: await responseError(response, `Could not load progress (${response.status})`)
			};
		}

		return {
			item: (await response.json()) as ResourceProgress,
			error: null
		};
	} catch {
		return {
			item: null,
			error: "Could not load progress right now. Please try again."
		};
	}
}

export async function updateResourceProgress(
	loadFetch: typeof fetch,
	resourceID: string,
	status: ProgressStatus
): Promise<ProgressUpdateResult> {
	if (!USER_ID) {
		return { ok: false, item: null, error: "No user id configured." };
	}

	try {
		const response = await loadFetch(apiURL(`/api/v1/me/progress/${resourceID}`), {
			method: "PUT",
			headers: buildHeaders({ includeUser: true, includeJSON: true }),
			body: JSON.stringify({ status })
		});

		if (!response.ok) {
			return {
				ok: false,
				item: null,
				error: await responseError(response, `Could not update progress (${response.status})`)
			};
		}

		return {
			ok: true,
			item: (await response.json()) as ProgressUpdateResponse,
			error: null
		};
	} catch {
		return {
			ok: false,
			item: null,
			error: "Could not update progress right now. Please try again."
		};
	}
}

export function deriveFilterOptions(resources: Resource[]) {
	return {
		skills: uniqueValues(resources.flatMap((resource) => resource.skillTags)),
		topics: uniqueValues(resources.flatMap((resource) => resource.topicTags)),
		providers: uniqueValues(resources.map((resource) => resource.providerSlug)),
		types: uniqueValues(resources.map((resource) => resource.sourceType))
	};
}

export function getCurrentUserID() {
	return USER_ID;
}

function normalizeProfile(payload: ProfileAPIResponse): UserProfile {
	const level = typeof payload.targetLevel === "string" && payload.targetLevel.trim() !== "" ? payload.targetLevel : null;
	return {
		userId: payload.userId,
		displayName: payload.displayName,
		targetLevel: level as CEFRLevel | null,
		learningGoals: payload.learningGoals ?? "",
		preferredResourceTypes: Array.isArray(payload.preferredResourceTypes) ? payload.preferredResourceTypes : [],
		preferredSkills: Array.isArray(payload.preferredSkills) ? payload.preferredSkills : [],
		preferredSourceProviders: Array.isArray(payload.preferredSourceProviders)
			? payload.preferredSourceProviders
			: [],
		updatedAt: payload.updatedAt ?? new Date().toISOString()
	};
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
	return Array.from(new Set(values.filter((value) => value.trim() !== ""))).sort((left, right) =>
		left.localeCompare(right)
	);
}
