<script lang="ts">
	import { updateResourceProgress } from "$lib/api/resources";
	import type { ProgressStatus } from "$lib/types";
	import { cn } from "$lib/utils/cn";
	import { createEventDispatcher } from "svelte";

	export let resourceID: string;
	export let initialStatus: ProgressStatus = "not_started";
	export let className = "";

	const dispatch = createEventDispatcher<{ change: { status: ProgressStatus } }>();

	const options: { value: ProgressStatus; label: string }[] = [
		{ value: "not_started", label: "Not started" },
		{ value: "in_progress", label: "In progress" },
		{ value: "completed", label: "Completed" }
	];

	let currentStatus: ProgressStatus = initialStatus;
	let previousInitialStatus = initialStatus;
	let pending = false;
	let error: string | null = null;

	$: if (initialStatus !== previousInitialStatus) {
		previousInitialStatus = initialStatus;
		currentStatus = initialStatus;
		error = null;
	}

	async function setStatus(status: ProgressStatus) {
		if (pending || status === currentStatus) {
			return;
		}

		const previousStatus = currentStatus;
		currentStatus = status;
		error = null;
		pending = true;

		const result = await updateResourceProgress(fetch, resourceID, status);
		pending = false;

		if (!result.ok || !result.item) {
			currentStatus = previousStatus;
			error = result.error ?? "Could not update progress.";
			return;
		}

		currentStatus = result.item.status;
		previousInitialStatus = currentStatus;
		dispatch("change", { status: currentStatus });
	}

	function helperText(status: ProgressStatus) {
		switch (status) {
			case "completed":
				return "Completed. Revisit when you want a quick refresh.";
			case "in_progress":
				return "In progress. Continue this resource in your next study session.";
			default:
				return "Not started yet. Mark as in progress when you begin.";
		}
	}
</script>

<section class={cn("space-y-3", className)}>
	<p class="text-xs uppercase tracking-[0.08em] text-muted">Learning progress</p>

	<div
		class="grid grid-cols-3 gap-2 rounded-xl border border-border bg-surface-soft/80 p-1"
		role="group"
		aria-label="Learning progress status"
	>
		{#each options as option}
			<button
				type="button"
				class={cn(
					"rounded-lg px-3 py-2 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/80 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
					currentStatus === option.value
						? "bg-accent text-accent-foreground"
						: "text-muted hover:bg-surface hover:text-foreground"
				)}
				aria-pressed={currentStatus === option.value}
				aria-busy={pending}
				disabled={pending}
				on:click={() => setStatus(option.value)}
				data-testid={`progress-${option.value}`}
			>
				{option.label}
			</button>
		{/each}
	</div>

	<p class="text-xs text-muted">{helperText(currentStatus)}</p>

	{#if error}
		<p class="rounded-lg border border-border bg-surface p-2 text-xs text-muted" role="status" aria-live="polite">
			{error}
		</p>
	{/if}
</section>
