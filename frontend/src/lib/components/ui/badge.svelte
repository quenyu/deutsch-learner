<script lang="ts">
	import { cn } from "$lib/utils/cn";

	export let variant: "default" | "outline" | "support" = "default";
	export let className = "";

	const variants = {
		default: "bg-accent/15 text-accent border border-accent/25",
		outline: "bg-surface-soft text-foreground border border-border/80",
		support: "bg-success/20 text-success border border-success/30"
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
