<script lang="ts">
	import { cn } from "$lib/utils/cn";

	export let className = "";
</script>

<article
	class={cn(
		"rounded-2xl border border-border/80 bg-surface/90 p-4 shadow-card backdrop-blur sm:p-5",
		className
	)}
	{...$$restProps}
>
	<slot />
</article>
