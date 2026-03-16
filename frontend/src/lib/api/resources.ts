import { env } from "$env/dynamic/public";
import type { CatalogFilters, CEFRLevel, Resource } from "$lib/types";

type ResourcesAPIResponse = {
	items?: Resource[];
	count?: number;
};

export type ListResourcesResult = {
	items: Resource[];
	count: number;
	error: string | null;
};

export const levelOptions: CEFRLevel[] = ["A1", "A2", "B1", "B2", "C1", "C2"];

const API_BASE_URL = env.PUBLIC_API_BASE_URL || "http://localhost:8080";

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

		const url = `${API_BASE_URL}/api/v1/resources${query.toString() ? `?${query.toString()}` : ""}`;
		const response = await loadFetch(url, {
			headers: {
				Accept: "application/json"
			}
		});

		if (!response.ok) {
			return {
				items: [],
				count: 0,
				error: `Could not load resources (${response.status})`
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

export function deriveFilterOptions(resources: Resource[]) {
	return {
		skills: uniqueValues(resources.flatMap((resource) => resource.skillTags)),
		topics: uniqueValues(resources.flatMap((resource) => resource.topicTags))
	};
}

function uniqueValues(values: string[]) {
	return Array.from(new Set(values)).sort((left, right) => left.localeCompare(right));
}
