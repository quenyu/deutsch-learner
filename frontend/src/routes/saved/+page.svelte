<script lang="ts">
	import Badge from "$lib/components/ui/badge.svelte";
	import ResourceCard from "$lib/components/resource/resource-card.svelte";
	import { savedResources } from "$lib/stores/saved-resources";
	import { onMount } from "svelte";
	import type { PageData } from "./$types";

	export let data: PageData;

	onMount(() => {
		if (!data.loadError) {
			savedResources.replace(data.resources.map((resource) => resource.id));
		}
	});

	$: savedItems = data.resources.filter((resource) => $savedResources.ids.includes(resource.id));
	$: inProgressItems = savedItems.filter((resource) => resource.progressStatus === "in_progress");
	$: completedItems = savedItems.filter((resource) => resource.progressStatus === "completed");
	$: readyToStartItems = savedItems.filter((resource) => resource.progressStatus !== "in_progress" && resource.progressStatus !== "completed");
	$: continueNext = inProgressItems[0] ?? readyToStartItems[0] ?? null;
</script>

<section class="space-y-6">
	<div class="space-y-2">
		<p class="text-xs uppercase tracking-[0.14em] text-accent">Personal Workspace</p>
		<h1 class="text-3xl font-semibold leading-tight text-foreground sm:text-4xl">Saved Resources</h1>
		<p class="max-w-2xl text-sm leading-relaxed text-muted sm:text-base">
			Keep your shortlist focused. These resources are ready for your next study session.
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
			<p>Saved resources and progress persist across reloads.</p>
		</div>

		{#if continueNext}
			<div class="rounded-2xl border border-border bg-surface p-5">
				<p class="text-xs uppercase tracking-[0.14em] text-accent">Next action</p>
				<div class="mt-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
					<div class="space-y-1">
						<h2 class="text-lg font-semibold">Continue this resource next</h2>
						<p class="text-sm text-muted">{continueNext.title}</p>
					</div>
					<a
						href={`/resources/${continueNext.slug}`}
						class="inline-flex rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-accent-foreground hover:bg-accent/90"
					>
						Open resource
					</a>
				</div>
			</div>
		{/if}

		<section class="space-y-3" data-testid="saved-in-progress">
			<div class="flex items-center justify-between">
				<h2 class="text-xl font-semibold">In progress</h2>
				<Badge>{inProgressItems.length}</Badge>
			</div>
			{#if inProgressItems.length === 0}
				<p class="rounded-xl border border-border bg-surface p-4 text-sm text-muted">
					Nothing in progress yet. Start a saved resource to build momentum.
				</p>
			{:else}
				<ul class="grid gap-4 sm:grid-cols-2">
					{#each inProgressItems as resource}
						<li>
							<ResourceCard {resource} />
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<section class="space-y-3" data-testid="saved-completed">
			<div class="flex items-center justify-between">
				<h2 class="text-xl font-semibold">Completed</h2>
				<Badge variant="support">{completedItems.length}</Badge>
			</div>
			{#if completedItems.length === 0}
				<p class="rounded-xl border border-border bg-surface p-4 text-sm text-muted">
					Completed resources will appear here for easy review.
				</p>
			{:else}
				<ul class="grid gap-4 sm:grid-cols-2">
					{#each completedItems as resource}
						<li>
							<ResourceCard {resource} />
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<section class="space-y-3" data-testid="saved-ready">
			<div class="flex items-center justify-between">
				<h2 class="text-xl font-semibold">Ready to start</h2>
				<Badge variant="outline">{readyToStartItems.length}</Badge>
			</div>
			{#if readyToStartItems.length === 0}
				<p class="rounded-xl border border-border bg-surface p-4 text-sm text-muted">
					All saved resources are either in progress or completed.
				</p>
			{:else}
				<ul class="grid gap-4 sm:grid-cols-2">
					{#each readyToStartItems as resource}
						<li>
							<ResourceCard {resource} />
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}
</section>
