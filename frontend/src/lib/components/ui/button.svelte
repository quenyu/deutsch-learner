<script lang="ts">
	import { cn } from "$lib/utils/cn";

	export let variant: "primary" | "outline" | "ghost" = "primary";
	export let size: "sm" | "md" | "lg" = "md";
	export let className = "";
	export let type: "button" | "submit" | "reset" = "button";
	export let disabled = false;

	const baseClass =
		"inline-flex items-center justify-center whitespace-nowrap rounded-xl font-semibold tracking-wide transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/80 focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50";
	const variants = {
		primary: "bg-accent text-accent-foreground hover:bg-accent/90",
		outline: "border border-border bg-surface-soft text-foreground hover:border-accent/70 hover:bg-surface",
		ghost: "text-muted hover:bg-surface-soft hover:text-foreground"
	};
	const sizes = {
		sm: "h-8 px-3 text-xs",
		md: "h-10 px-4 text-sm",
		lg: "h-11 px-5 text-sm"
	};
</script>

<button
	{type}
	{disabled}
	class={cn(baseClass, variants[variant], sizes[size], className)}
	on:click
	{...$$restProps}
>
	<slot />
</button>
