<script lang="ts">
	import ResourceCard from "$lib/components/resource/resource-card.svelte";
	import CatalogFilters from "$lib/features/catalog/catalog-filters.svelte";
	import { savedResources } from "$lib/stores/saved-resources";
	import { onMount } from "svelte";
	import type { PageData } from "./$types";

	export let data: PageData;

	onMount(() => {
		savedResources.mergeFromResources(data.resources);
	});
</script>

<section class="space-y-6">
	<div class="space-y-3">
		<p class="text-xs uppercase tracking-[0.14em] text-accent">Curated Discovery</p>
		<h1 class="text-3xl font-semibold leading-tight text-foreground sm:text-4xl">Resource Catalog</h1>
		<p class="max-w-2xl text-sm leading-relaxed text-muted sm:text-base">
			Discover trusted external German learning resources, filtered by level, skill, and topic. Save what
			matters and build calm momentum over time.
		</p>
	</div>

	{#if data.loadError}
		<div
			class="rounded-xl border border-border bg-surface-soft/85 p-3 text-sm text-muted"
			role="status"
			aria-live="polite"
		>
			{data.loadError}
		</div>
	{/if}

	<CatalogFilters
		levelOptions={data.options.levels}
		skillOptions={data.options.skills}
		topicOptions={data.options.topics}
		level={data.filters.level}
		skill={data.filters.skill}
		topic={data.filters.topic}
		query={data.filters.query}
		free={data.filters.free}
	/>

	<div class="flex items-center justify-between text-sm text-muted">
		<p>{data.totalCount ?? data.resources.length} resources</p>
		<p>Outbound links are reviewed for quality and relevance.</p>
	</div>

	{#if data.resources.length === 0}
		<div class="rounded-2xl border border-border bg-surface p-6 text-center">
			<h2 class="text-lg font-semibold">No resources match these filters</h2>
			<p class="mt-2 text-sm text-muted">Try broadening your level, topic, or search query.</p>
			<a
				href="/resources"
				class="mt-4 inline-flex rounded-xl border border-border bg-surface-soft px-4 py-2 text-sm font-medium text-foreground hover:border-accent/60"
			>
				Reset filters
			</a>
		</div>
	{:else}
		<ul class="grid gap-4 sm:grid-cols-2">
			{#each data.resources as resource}
				<li>
					<ResourceCard {resource} />
				</li>
			{/each}
		</ul>
	{/if}
</section>
