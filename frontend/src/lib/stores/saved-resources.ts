import { browser } from "$app/environment";
import { listSavedResources, removeSavedResource, saveResource } from "$lib/api/resources";
import type { Resource } from "$lib/types";
import { get, writable } from "svelte/store";

type SavedState = {
	ids: string[];
	pending: Record<string, true>;
	error: string | null;
	isLoaded: boolean;
};

const store = writable<SavedState>({
	ids: [],
	pending: {},
	error: null,
	isLoaded: false
});

function unique(values: string[]) {
	return Array.from(new Set(values));
}

function removePending(state: SavedState, resourceID: string) {
	const { [resourceID]: _, ...pending } = state.pending;
	return pending;
}

export const savedResources = {
	subscribe: store.subscribe,
	async load(loadFetch: typeof fetch = fetch) {
		const result = await listSavedResources(loadFetch);
		if (result.error) {
			store.update((state) => ({
				...state,
				error: result.error,
				isLoaded: true
			}));
			return;
		}

		store.set({
			ids: unique(result.items.map((resource) => resource.id)),
			pending: {},
			error: null,
			isLoaded: true
		});
	},
	replace(ids: string[]) {
		store.update((state) => ({
			...state,
			ids: unique(ids),
			error: null,
			isLoaded: true
		}));
	},
	mergeFromResources(resources: Resource[]) {
		const savedIDs = resources.filter((resource) => resource.isSaved).map((resource) => resource.id);
		if (savedIDs.length === 0) {
			return;
		}

		store.update((state) => ({
			...state,
			ids: unique([...state.ids, ...savedIDs]),
			isLoaded: true
		}));
	},
	async toggle(resourceID: string, loadFetch: typeof fetch = fetch) {
		const previous = get(store);
		if (previous.pending[resourceID]) {
			return;
		}

		const wasSaved = previous.ids.includes(resourceID);
		const optimisticIDs = wasSaved
			? previous.ids.filter((value) => value !== resourceID)
			: unique([...previous.ids, resourceID]);

		store.set({
			...previous,
			ids: optimisticIDs,
			pending: {
				...previous.pending,
				[resourceID]: true
			},
			error: null,
			isLoaded: true
		});

		const result = wasSaved
			? await removeSavedResource(loadFetch, resourceID)
			: await saveResource(loadFetch, resourceID);

		if (!result.ok) {
			store.update((state) => ({
				...state,
				ids: previous.ids,
				pending: removePending(state, resourceID),
				error: result.error ?? "Could not update saved resources.",
				isLoaded: true
			}));
			return;
		}

		store.update((state) => ({
			...state,
			pending: removePending(state, resourceID),
			error: null,
			isLoaded: true
		}));
	}
};

if (browser) {
	void savedResources.load();
}
