<script lang="ts">
	import ResourceCard from "$lib/components/resource/resource-card.svelte";
	import { resources } from "$lib/mock/resources";
	import { savedResources } from "$lib/stores/saved-resources";

	$: savedItems = resources.filter((resource) => $savedResources.includes(resource.id));
</script>

<section class="space-y-6">
	<div class="space-y-2">
		<p class="text-xs uppercase tracking-[0.14em] text-accent">Personal Workspace</p>
		<h1 class="text-3xl font-semibold leading-tight text-foreground sm:text-4xl">Saved Resources</h1>
		<p class="max-w-2xl text-sm leading-relaxed text-muted sm:text-base">
			Keep your shortlist focused. These resources are ready for your next study session.
		</p>
	</div>

	{#if savedItems.length === 0}
		<div class="rounded-2xl border border-border bg-surface p-6 text-center">
			<h2 class="text-lg font-semibold">Nothing saved yet</h2>
			<p class="mt-2 text-sm text-muted">
				Save resources from the catalog to build a personalized learning queue.
			</p>
			<a
				href="/resources"
				class="mt-4 inline-flex rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-accent-foreground hover:bg-accent/90"
			>
				Open catalog
			</a>
		</div>
	{:else}
		<div class="flex items-center justify-between text-sm text-muted">
			<p>{savedItems.length} saved</p>
			<p>Saved state is local for this scaffold.</p>
		</div>

		<ul class="grid gap-4 sm:grid-cols-2">
			{#each savedItems as resource}
				<li>
					<ResourceCard {resource} />
				</li>
			{/each}
		</ul>
	{/if}
</section>
