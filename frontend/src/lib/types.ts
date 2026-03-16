export type CEFRLevel = "A1" | "A2" | "B1" | "B2" | "C1" | "C2";

export interface Resource {
	id: string;
	slug: string;
	title: string;
	summary: string;
	sourceName: string;
	sourceType: "youtube" | "article" | "playlist" | "course" | "podcast" | "grammar_reference" | "exercise";
	externalUrl: string;
	cefrLevel: CEFRLevel;
	format: string;
	durationMinutes: number;
	isFree: boolean;
	priceCents: number | null;
	skillTags: string[];
	topicTags: string[];
}

export interface CatalogFilters {
	level: string;
	skill: string;
	topic: string;
	query: string;
	free: boolean | null;
}
