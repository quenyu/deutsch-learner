import { getProfile, levelOptions, listSourceProviders, resourceTypeOptions } from "$lib/api/resources";
import type { PageLoad } from "./$types";

const skillOptions = ["grammar", "vocabulary", "listening", "reading", "speaking"];

export const load: PageLoad = async ({ fetch }) => {
	const [profileResult, providersResult] = await Promise.all([getProfile(fetch), listSourceProviders(fetch)]);

	return {
		profile: profileResult.item,
		levelOptions,
		resourceTypeOptions,
		skillOptions,
		sourceProviderOptions: providersResult.items,
		loadError: profileResult.error ?? providersResult.error
	};
};
