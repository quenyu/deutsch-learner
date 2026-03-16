<script lang="ts">
	import Badge from "$lib/components/ui/badge.svelte";
	import Card from "$lib/components/ui/card.svelte";
	import SaveButton from "$lib/features/catalog/save-button.svelte";
	import type { Resource } from "$lib/types";

	export let resource: Resource;
</script>

<Card className="space-y-4" data-testid="resource-card">
	<div class="flex items-start justify-between gap-3">
		<div class="space-y-1">
			<p class="text-xs uppercase tracking-[0.08em] text-muted">{resource.sourceName}</p>
			<h3 class="text-lg font-semibold leading-tight text-foreground">
				<a href={`/resources/${resource.slug}`} class="hover:text-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/70">
					{resource.title}
				</a>
			</h3>
		</div>
		<SaveButton resourceID={resource.id} />
	</div>

	<p class="text-sm leading-relaxed text-muted">{resource.summary}</p>

	<div class="flex flex-wrap gap-2">
		<Badge>{resource.cefrLevel}</Badge>
		<Badge variant="outline">{resource.format}</Badge>
		<Badge variant={resource.isFree ? "support" : "outline"}>
			{resource.isFree ? "Free" : "Paid"}
		</Badge>
	</div>

	<div class="flex flex-wrap gap-2">
		{#each resource.skillTags as skill}
			<Badge variant="outline" className="normal-case tracking-normal">{skill}</Badge>
		{/each}
		{#each resource.topicTags as topic}
			<Badge variant="outline" className="normal-case tracking-normal">{topic}</Badge>
		{/each}
	</div>

	<div class="pt-1">
		<a
			href={resource.externalUrl}
			target="_blank"
			rel="noopener noreferrer"
			class="inline-flex items-center gap-2 text-sm font-medium text-accent hover:text-accent/80"
		>
			Open external resource
		</a>
	</div>
</Card>
