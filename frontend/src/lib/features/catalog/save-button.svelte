<script lang="ts">
	import { cn } from "$lib/utils/cn";
	import { savedResources } from "$lib/stores/saved-resources";

	export let resourceID: string;
	export let className = "";

	$: isSaved = $savedResources.includes(resourceID);

	function toggle(event: MouseEvent) {
		event.preventDefault();
		event.stopPropagation();
		savedResources.toggle(resourceID);
	}
</script>

<button
	type="button"
	class={cn(
		"inline-flex h-8 items-center justify-center rounded-xl px-3 text-xs font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/80 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
		isSaved
			? "border border-border bg-surface-soft text-foreground hover:border-accent/70"
			: "bg-accent text-accent-foreground hover:bg-accent/90",
		className
	)}
	aria-pressed={isSaved}
	on:click={toggle}
>
	{isSaved ? "Saved" : "Save"}
</button>
