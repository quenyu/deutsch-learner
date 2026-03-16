<script lang="ts">
	import Badge from "$lib/components/ui/badge.svelte";
	import Card from "$lib/components/ui/card.svelte";
	import SaveButton from "$lib/features/catalog/save-button.svelte";
	import ProgressControl from "$lib/features/progress/progress-control.svelte";
	import { savedResources } from "$lib/stores/saved-resources";
	import type { ProgressStatus } from "$lib/types";
	import { onMount } from "svelte";
	import type { PageData } from "./$types";

	export let data: PageData;
	let progressStatus: ProgressStatus = data.resource.progressStatus ?? "not_started";

	onMount(() => {
		savedResources.mergeFromResources([data.resource, ...data.relatedResources]);
	});

	function formatPrice(priceCents: number | null) {
		if (priceCents === null) {
			return "Free";
		}

		return new Intl.NumberFormat("en-US", {
			style: "currency",
			currency: "USD",
			maximumFractionDigits: 0
		}).format(priceCents / 100);
	}

	function progressLabel(status: ProgressStatus) {
		switch (status) {
			case "in_progress":
				return "In progress";
			case "completed":
				return "Completed";
			default:
				return "Not started";
		}
	}

	function progressVariant(status: ProgressStatus): "default" | "outline" | "support" {
		switch (status) {
			case "in_progress":
				return "default";
			case "completed":
				return "support";
			default:
				return "outline";
		}
	}

	function ingestionLabel(origin: string) {
		return origin === "imported" ? "Imported" : "Curated";
	}

	function sourceKindLabel(kind: string) {
		return kind.replaceAll("_", " ");
	}

	function handleProgressChange(event: CustomEvent<{ status: ProgressStatus }>) {
		progressStatus = event.detail.status;
	}
</script>

<section class="space-y-6">
	{#if data.loadError}
		<div
			class="rounded-xl border border-border bg-surface-soft/85 p-3 text-sm text-muted"
			role="status"
			aria-live="polite"
		>
			{data.loadError}
		</div>
	{/if}

	<a href="/resources" class="inline-flex items-center gap-2 text-sm text-muted hover:text-foreground">
		Back to catalog
	</a>

	<Card className="space-y-6 p-6">
		<div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
			<div class="space-y-2">
				<p class="text-xs uppercase tracking-[0.14em] text-accent">{data.resource.sourceName}</p>
				<h1 class="text-2xl font-semibold leading-tight text-foreground sm:text-3xl">{data.resource.title}</h1>
				<p class="max-w-3xl text-sm leading-relaxed text-muted sm:text-base">{data.resource.summary}</p>
			</div>
			<div class="flex shrink-0 items-start gap-3">
				<SaveButton resourceID={data.resource.id} className="shrink-0" />
				<Badge variant={progressVariant(progressStatus)}>{progressLabel(progressStatus)}</Badge>
			</div>
		</div>

		<div class="flex flex-wrap gap-2">
			<Badge>{data.resource.cefrLevel}</Badge>
			<Badge variant="outline">{data.resource.format}</Badge>
			<Badge variant="outline">{data.resource.providerSlug}</Badge>
			<Badge variant={data.resource.ingestionOrigin === "imported" ? "default" : "outline"}>
				{ingestionLabel(data.resource.ingestionOrigin)}
			</Badge>
			<Badge variant="outline">{sourceKindLabel(data.resource.sourceKind)}</Badge>
			<Badge variant={data.resource.isFree ? "support" : "outline"}>
				{data.resource.isFree ? "Free" : "Paid"}
			</Badge>
		</div>

		<dl class="grid gap-4 rounded-xl border border-border bg-surface-soft/80 p-4 text-sm sm:grid-cols-2">
			<div>
				<dt class="text-xs uppercase tracking-[0.08em] text-muted">Duration</dt>
				<dd class="mt-1 font-medium">{data.resource.durationMinutes} minutes</dd>
			</div>
			<div>
				<dt class="text-xs uppercase tracking-[0.08em] text-muted">Price</dt>
				<dd class="mt-1 font-medium">{formatPrice(data.resource.priceCents)}</dd>
			</div>
			<div>
				<dt class="text-xs uppercase tracking-[0.08em] text-muted">Skills</dt>
				<dd class="mt-1">{data.resource.skillTags.join(", ")}</dd>
			</div>
			<div>
				<dt class="text-xs uppercase tracking-[0.08em] text-muted">Topics</dt>
				<dd class="mt-1">{data.resource.topicTags.join(", ")}</dd>
			</div>
			<div>
				<dt class="text-xs uppercase tracking-[0.08em] text-muted">Provider</dt>
				<dd class="mt-1">{data.resource.providerName}</dd>
			</div>
			<div>
				<dt class="text-xs uppercase tracking-[0.08em] text-muted">Last synced</dt>
				<dd class="mt-1">{data.resource.lastSyncedAt ? new Date(data.resource.lastSyncedAt).toLocaleDateString() : "Not available"}</dd>
			</div>
		</dl>

		<ProgressControl
			resourceID={data.resource.id}
			initialStatus={progressStatus}
			on:change={handleProgressChange}
		/>

		<a
			href={data.resource.externalUrl}
			target="_blank"
			rel="noopener noreferrer"
			class="inline-flex w-full items-center justify-center rounded-xl bg-accent px-4 py-3 text-sm font-semibold text-accent-foreground transition hover:bg-accent/90 sm:w-auto"
		>
			Visit external resource
		</a>
		<p class="text-xs text-muted">You will continue on the original provider site. We never host lesson content.</p>
	</Card>

	<section class="space-y-3">
		<div class="flex items-end justify-between">
			<h2 class="text-xl font-semibold">Related resources</h2>
			<a href="/resources" class="text-sm text-muted hover:text-foreground">Browse all</a>
		</div>

		{#if data.relatedResources.length === 0}
			<p class="rounded-xl border border-border bg-surface p-4 text-sm text-muted">No related resources yet.</p>
		{:else}
			<ul class="grid gap-3 sm:grid-cols-3">
				{#each data.relatedResources as related}
					<li>
						<Card className="h-full space-y-3">
							<p class="text-xs uppercase tracking-[0.08em] text-muted">{related.sourceName}</p>
							<h3 class="text-base font-semibold">
								<a href={`/resources/${related.slug}`} class="hover:text-accent">
									{related.title}
								</a>
							</h3>
							<p class="text-sm text-muted">{related.summary}</p>
						</Card>
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</section>
