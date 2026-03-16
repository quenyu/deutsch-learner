<script lang="ts">
	import { cn } from "$lib/utils/cn";

	export let className = "";
	export let type = "text";
</script>

<input
	{type}
	class={cn(
		"h-10 w-full rounded-xl border border-border bg-surface px-3 text-sm text-foreground placeholder:text-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/80 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
		className
	)}
	{...$$restProps}
/>
