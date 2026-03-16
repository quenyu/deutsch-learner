import {
	deriveFilterOptions,
	getProfile,
	levelOptions,
	listProgress,
	listResources,
	listSourceProviders,
	resourceTypeOptions
} from "$lib/api/resources";
import type { CatalogFilters } from "$lib/types";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ url, fetch }) => {
	const freeParam = url.searchParams.get("free");

	const requestedFilters: CatalogFilters = {
		level: url.searchParams.get("level") ?? "",
		skill: url.searchParams.get("skill") ?? "",
		topic: url.searchParams.get("topic") ?? "",
		provider: url.searchParams.get("provider") ?? "",
		type: url.searchParams.get("type") ?? "",
		query: url.searchParams.get("q") ?? "",
		free: freeParam === "true" ? true : freeParam === "false" ? false : null
	};

	const profileResult = await getProfile(fetch);
	const hasExplicitFilters =
		url.searchParams.has("level") ||
		url.searchParams.has("skill") ||
		url.searchParams.has("topic") ||
		url.searchParams.has("provider") ||
		url.searchParams.has("type") ||
		url.searchParams.has("q") ||
		url.searchParams.has("free");

	const filters: CatalogFilters = { ...requestedFilters };
	let appliedProfileDefaults = false;

	if (!hasExplicitFilters && profileResult.item) {
		if (!filters.level && profileResult.item.targetLevel) {
			filters.level = profileResult.item.targetLevel;
			appliedProfileDefaults = true;
		}
		if (!filters.skill && profileResult.item.preferredSkills.length > 0) {
			filters.skill = profileResult.item.preferredSkills[0];
			appliedProfileDefaults = true;
		}
		if (!filters.provider && profileResult.item.preferredSourceProviders.length > 0) {
			filters.provider = profileResult.item.preferredSourceProviders[0];
			appliedProfileDefaults = true;
		}
		if (!filters.type && profileResult.item.preferredResourceTypes.length > 0) {
			filters.type = profileResult.item.preferredResourceTypes[0];
			appliedProfileDefaults = true;
		}
	}

	const [sourceProvidersResult, resourcesResult, progressResult] = await Promise.all([
		listSourceProviders(fetch),
		listResources(fetch, filters),
		listProgress(fetch)
	]);

	const progressByResourceID = new Map(progressResult.items.map((item) => [item.resourceId, item.status]));

	const resources = resourcesResult.items.map((resource) => ({
		...resource,
		progressStatus: progressByResourceID.get(resource.id) ?? "not_started"
	}));

	const options = deriveFilterOptions(resources);
	const providerOptions =
		sourceProvidersResult.items.length > 0
			? sourceProvidersResult.items.map((provider) => ({
					slug: provider.slug,
					name: provider.name
				}))
			: options.providers.map((slug) => ({ slug, name: slug }));

	return {
		filters,
		options: {
			levels: levelOptions,
			skills: options.skills,
			topics: options.topics,
			providers: providerOptions,
			types: resourceTypeOptions
		},
		resources,
		totalCount: resourcesResult.count,
		loadError:
			resourcesResult.error ??
			progressResult.error ??
			sourceProvidersResult.error ??
			profileResult.error,
		appliedProfileDefaults,
		profileSummary: profileResult.item
			? {
					displayName: profileResult.item.displayName
				}
			: null
	};
};
