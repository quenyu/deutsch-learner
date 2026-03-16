export type CEFRLevel = "A1" | "A2" | "B1" | "B2" | "C1" | "C2";
export type ProgressStatus = "not_started" | "in_progress" | "completed";
export type ProviderSlug = "manual" | "youtube" | "stepik" | "enrichment";
export type ResourceType = "youtube" | "article" | "playlist" | "course" | "podcast" | "grammar_reference" | "exercise";
export type IngestionOrigin = "manual" | "imported";

export interface Resource {
	id: string;
	slug: string;
	title: string;
	summary: string;
	sourceName: string;
	sourceType: ResourceType;
	externalUrl: string;
	cefrLevel: CEFRLevel;
	format: string;
	durationMinutes: number;
	isFree: boolean;
	priceCents: number | null;
	skillTags: string[];
	topicTags: string[];
	isSaved: boolean;
	providerSlug: ProviderSlug | string;
	providerName: string;
	ingestionOrigin: IngestionOrigin | string;
	sourceKind: string;
	lastSyncedAt?: string;
	progressStatus?: ProgressStatus;
}

export interface ResourceProgress {
	userId: string;
	resourceId: string;
	status: ProgressStatus;
	progressPercent: number;
	lastStudiedAt?: string;
	updatedAt?: string;
}

export interface CatalogFilters {
	level: string;
	skill: string;
	topic: string;
	provider: string;
	type: string;
	query: string;
	free: boolean | null;
}

export interface SourceProvider {
	id: string;
	slug: ProviderSlug | string;
	name: string;
	description: string;
	isEnabled: boolean;
}

export interface UserProfile {
	userId: string;
	displayName: string;
	targetLevel: CEFRLevel | null;
	learningGoals: string;
	preferredResourceTypes: string[];
	preferredSkills: string[];
	preferredSourceProviders: string[];
	updatedAt: string;
}
