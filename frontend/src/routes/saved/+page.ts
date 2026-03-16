import { listSavedResources } from "$lib/api/resources";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ fetch }) => {
	const result = await listSavedResources(fetch);

	return {
		resources: result.items,
		totalCount: result.count,
		loadError: result.error
	};
};
