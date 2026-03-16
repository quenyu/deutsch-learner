import type { CatalogFilters, CEFRLevel, Resource } from "$lib/types";

export const resources: Resource[] = [
	{
		id: "c01f9204-5f38-47e4-b3ec-c580691ff44f",
		slug: "dw-nicos-weg-a1-overview",
		title: "Nicos Weg A1 Overview",
		summary: "Structured beginner-friendly entry into practical German for daily communication.",
		sourceName: "Deutsche Welle",
		sourceType: "course",
		externalUrl: "https://learngerman.dw.com/en/nicos-weg/c-36519687",
		cefrLevel: "A1",
		format: "course",
		durationMinutes: 40,
		isFree: true,
		priceCents: null,
		skillTags: ["listening", "speaking"],
		topicTags: ["daily-life", "introductions"],
		isSaved: false,
		providerSlug: "manual",
		providerName: "Manual Curation",
		ingestionOrigin: "manual",
		sourceKind: "course"
	},
	{
		id: "4c55f202-c4c5-43c8-98ac-3ccf490f4b7f",
		slug: "easy-german-street-interviews-a2",
		title: "Easy German Street Interviews",
		summary: "Natural spoken German with subtitles and context from real-world conversations.",
		sourceName: "Easy German",
		sourceType: "youtube",
		externalUrl: "https://www.youtube.com/@EasyGerman",
		cefrLevel: "A2",
		format: "video",
		durationMinutes: 18,
		isFree: true,
		priceCents: null,
		skillTags: ["listening", "vocabulary"],
		topicTags: ["culture", "daily-life"],
		isSaved: false,
		providerSlug: "manual",
		providerName: "Manual Curation",
		ingestionOrigin: "manual",
		sourceKind: "video"
	},
	{
		id: "ad7a4a03-3ee8-4ec1-a21a-b9ca3ee2676b",
		slug: "yourdailygerman-case-system-guide",
		title: "German Case System Guide",
		summary: "Clear explanation of nominative, accusative, dative, and genitive with examples.",
		sourceName: "Your Daily German",
		sourceType: "article",
		externalUrl: "https://yourdailygerman.com/german-case-system/",
		cefrLevel: "B1",
		format: "article",
		durationMinutes: 25,
		isFree: true,
		priceCents: null,
		skillTags: ["grammar", "reading"],
		topicTags: ["grammar", "cases"],
		isSaved: false,
		providerSlug: "manual",
		providerName: "Manual Curation",
		ingestionOrigin: "manual",
		sourceKind: "article"
	},
	{
		id: "c58d0f53-fc1f-41d4-a8a0-91f77f74895d",
		slug: "langsam-gesprochene-nachrichten-b2",
		title: "Langsam Gesprochene Nachrichten",
		summary: "Slow-paced news audio that helps intermediate learners build listening confidence.",
		sourceName: "Deutsche Welle",
		sourceType: "podcast",
		externalUrl: "https://www.dw.com/de/deutsch-lernen/nachrichten/s-8030",
		cefrLevel: "B2",
		format: "podcast",
		durationMinutes: 12,
		isFree: true,
		priceCents: null,
		skillTags: ["listening", "news"],
		topicTags: ["current-events"],
		isSaved: false,
		providerSlug: "manual",
		providerName: "Manual Curation",
		ingestionOrigin: "manual",
		sourceKind: "podcast"
	},
	{
		id: "2828d81f-45bb-4410-8653-df2a1d4e657d",
		slug: "goethe-online-live-group-course",
		title: "Goethe Online Live Group Course",
		summary: "Instructor-led classes with structured progression and collaborative exercises.",
		sourceName: "Goethe-Institut",
		sourceType: "course",
		externalUrl: "https://www.goethe.de/en/spr/kur/typ.html",
		cefrLevel: "B1",
		format: "course",
		durationMinutes: 90,
		isFree: false,
		priceCents: 24900,
		skillTags: ["speaking", "grammar"],
		topicTags: ["exam-prep", "conversation"],
		isSaved: false,
		providerSlug: "manual",
		providerName: "Manual Curation",
		ingestionOrigin: "manual",
		sourceKind: "course"
	},
	{
		id: "622671c2-1095-44fd-a4e1-325d18f82eb9",
		slug: "deutschtrainer-grammar-reference-a2",
		title: "Deutschtrainer Grammar Reference",
		summary: "Compact grammar explanations for common beginner and lower-intermediate mistakes.",
		sourceName: "DW Deutschtrainer",
		sourceType: "grammar_reference",
		externalUrl: "https://learngerman.dw.com/en/deutschtrainer/c-56169568",
		cefrLevel: "A2",
		format: "reference",
		durationMinutes: 15,
		isFree: true,
		priceCents: null,
		skillTags: ["grammar", "vocabulary"],
		topicTags: ["daily-life", "basics"],
		isSaved: false,
		providerSlug: "manual",
		providerName: "Manual Curation",
		ingestionOrigin: "manual",
		sourceKind: "grammar_reference"
	}
];

export const levelOptions: CEFRLevel[] = ["A1", "A2", "B1", "B2", "C1", "C2"];

export const skillOptions = uniqueValues(resources.flatMap((resource) => resource.skillTags));
export const topicOptions = uniqueValues(resources.flatMap((resource) => resource.topicTags));

export function filterResources(items: Resource[], filters: CatalogFilters) {
	const query = filters.query.trim().toLowerCase();

	return items.filter((item) => {
		if (filters.level && item.cefrLevel !== filters.level) {
			return false;
		}
		if (filters.skill && !item.skillTags.includes(filters.skill)) {
			return false;
		}
		if (filters.topic && !item.topicTags.includes(filters.topic)) {
			return false;
		}
		if (filters.provider && item.providerSlug !== filters.provider) {
			return false;
		}
		if (filters.type && item.sourceType !== filters.type) {
			return false;
		}
		if (filters.free !== null && item.isFree !== filters.free) {
			return false;
		}
		if (!query) {
			return true;
		}

		const haystack = `${item.title} ${item.summary} ${item.sourceName}`.toLowerCase();
		return haystack.includes(query);
	});
}

export function getResourceBySlug(slug: string) {
	return resources.find((resource) => resource.slug === slug) ?? null;
}

export function getRelatedResources(resourceID: string, limit = 3) {
	const target = resources.find((resource) => resource.id === resourceID);
	if (!target) {
		return [];
	}

	return resources
		.filter((candidate) => candidate.id !== target.id)
		.map((candidate) => ({
			resource: candidate,
			score: calculateAffinity(target, candidate)
		}))
		sort((left, right) => right.score - left.score)
		slice(0, limit)
		.map((row) => row.resource);
}

function calculateAffinity(a: Resource, b: Resource) {
	let score = 0;
	if (a.cefrLevel === b.cefrLevel) {
		score += 2;
	}
	score += overlapCount(a.skillTags, b.skillTags) * 2;
	score += overlapCount(a.topicTags, b.topicTags);
	return score;
}

function overlapCount(a: string[], b: string[]) {
	const right = new Set(b);
	return a.filter((item) => right.has(item)).length;
}

function uniqueValues(values: string[]) {
	return Array.from(new Set(values)).sort((left, right) => left.localeCompare(right));
}
