import { deriveFilterOptions, levelOptions, listResources } from "$lib/api/resources";
import type { CatalogFilters } from "$lib/types";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ url, fetch }) => {
	const freeParam = url.searchParams.get("free");

	const filters: CatalogFilters = {
		level: url.searchParams.get("level") ?? "",
		skill: url.searchParams.get("skill") ?? "",
		topic: url.searchParams.get("topic") ?? "",
		query: url.searchParams.get("q") ?? "",
		free: freeParam === "true" ? true : freeParam === "false" ? false : null
	};

	const result = await listResources(fetch, filters);
	const options = deriveFilterOptions(result.items);

	return {
		filters,
		options: {
			levels: levelOptions,
			skills: options.skills,
			topics: options.topics
		},
		resources: result.items,
		totalCount: result.count,
		loadError: result.error
	};
};
