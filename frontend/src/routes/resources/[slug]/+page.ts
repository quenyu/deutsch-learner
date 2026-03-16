import { getRelatedResources, getResourceBySlug } from "$lib/mock/resources";
import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params }) => {
	const resource = getResourceBySlug(params.slug);
	if (!resource) {
		throw error(404, "Resource not found");
	}

	return {
		resource,
		relatedResources: getRelatedResources(resource.id, 3)
	};
};
