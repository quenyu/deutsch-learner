import { listProgress, listSavedResources } from "$lib/api/resources";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ fetch }) => {
	const savedResult = await listSavedResources(fetch);
	const progressResult = await listProgress(fetch);
	const progressByResourceID = new Map(progressResult.items.map((item) => [item.resourceId, item.status]));

	const resources = savedResult.items.map((resource) => ({
		...resource,
		progressStatus: progressByResourceID.get(resource.id) ?? "not_started"
	}));

	return {
		resources,
		totalCount: savedResult.count,
		loadError: savedResult.error ?? progressResult.error
	};
};
