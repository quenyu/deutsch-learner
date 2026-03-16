<script lang="ts">
	import { cn } from "$lib/utils/cn";

	export let variant: "default" | "outline" | "support" = "default";
	export let className = "";

	const variants = {
		default: "border border-accent/35 bg-accent/18 text-accent",
		outline: "border border-border bg-surface-soft/80 text-foreground",
		support: "border border-success/35 bg-success/18 text-success"
	};
</script>

<span
	class={cn(
		"inline-flex items-center rounded-full px-2.5 py-1 text-[11px] font-medium uppercase tracking-[0.08em]",
		variants[variant],
		className
	)}
	{...$$restProps}
>
	<slot />
</span>
