import { browser } from "$app/environment";
import { writable } from "svelte/store";

const STORAGE_KEY = "deutsch.saved-resource-ids";

const store = writable<string[]>([]);

function readFromStorage(): string[] {
	if (!browser) {
		return [];
	}

	const raw = localStorage.getItem(STORAGE_KEY);
	if (!raw) {
		return [];
	}

	try {
		const parsed = JSON.parse(raw);
		if (!Array.isArray(parsed)) {
			return [];
		}

		return parsed.filter((item): item is string => typeof item === "string");
	} catch {
		return [];
	}
}

function persist(values: string[]) {
	if (!browser) {
		return;
	}
	localStorage.setItem(STORAGE_KEY, JSON.stringify(values));
}

function unique(values: string[]) {
	return Array.from(new Set(values));
}

export const savedResources = {
	subscribe: store.subscribe,
	hydrate() {
		store.set(readFromStorage());
	},
	toggle(resourceID: string) {
		store.update((current) => {
			const exists = current.includes(resourceID);
			const next = exists
				? current.filter((value) => value !== resourceID)
				: unique([...current, resourceID]);
			persist(next);
			return next;
		});
	},
	save(resourceID: string) {
		store.update((current) => {
			if (current.includes(resourceID)) {
				return current;
			}
			const next = unique([...current, resourceID]);
			persist(next);
			return next;
		});
	},
	remove(resourceID: string) {
		store.update((current) => {
			const next = current.filter((value) => value !== resourceID);
			persist(next);
			return next;
		});
	}
};

if (browser) {
	savedResources.hydrate();
}
