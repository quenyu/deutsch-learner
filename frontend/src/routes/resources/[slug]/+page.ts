import { getResourceBySlug, listResources } from "$lib/api/resources";
import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ params, fetch }) => {
	const detailResult = await getResourceBySlug(fetch, params.slug);
	if (detailResult.notFound) {
		throw error(404, "Resource not found");
	}
	if (!detailResult.item) {
		throw error(502, detailResult.error ?? "Could not load resource");
	}

	const resource = detailResult.item;

	const relatedResult = await listResources(fetch, {
		level: resource.cefrLevel,
		skill: "",
		topic: "",
		query: "",
		free: null
	});

	const relatedResources = relatedResult.items.filter((candidate) => candidate.id !== resource.id).slice(0, 3);

	return {
		resource,
		relatedResources,
		loadError: detailResult.error ?? relatedResult.error
	};
};
