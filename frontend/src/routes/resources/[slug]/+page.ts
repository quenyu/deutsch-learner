import { getProgressForResource, getResourceBySlug, listResources } from "$lib/api/resources";
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
	const progressResult = await getProgressForResource(fetch, resource.id);
	const resourceWithProgress = {
		...resource,
		progressStatus: progressResult.item?.status ?? "not_started"
	};

	const relatedResult = await listResources(fetch, {
		level: resourceWithProgress.cefrLevel,
		skill: "",
		topic: "",
		provider: "",
		type: "",
		query: "",
		free: null
	});

	const relatedResources = relatedResult.items
		.filter((candidate) => candidate.id !== resourceWithProgress.id)
		.slice(0, 3);

	return {
		resource: resourceWithProgress,
		relatedResources,
		loadError: detailResult.error ?? progressResult.error ?? relatedResult.error
	};
};
