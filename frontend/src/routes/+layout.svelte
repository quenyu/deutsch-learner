<script lang="ts">
	import "../app.css";
	import Badge from "$lib/components/ui/badge.svelte";
	import { savedResources } from "$lib/stores/saved-resources";
	import { page } from "$app/stores";

	$: path = $page.url.pathname;
</script>

<svelte:head>
	<title>Deutsch Learner</title>
	<meta
		name="description"
		content="Curated third-party German learning resources with calm guidance and progress-aware workflows."
	/>
</svelte:head>

<div class="min-h-screen">
	<header class="border-b border-border/80 bg-background/90 backdrop-blur">
		<div class="page-wrap flex items-center justify-between gap-3 pb-4 pt-4">
			<a href="/resources" class="text-lg font-semibold tracking-[0.02em] text-foreground">
				Deutsch Learner
			</a>

			<nav class="flex items-center gap-2 text-sm">
				<a
					href="/resources"
					class={`rounded-lg px-3 py-2 transition-colors ${
						path.startsWith("/resources")
							? "bg-surface-soft text-foreground"
							: "text-muted hover:bg-surface-soft hover:text-foreground"
					}`}
				>
					Catalog
				</a>
				<a
					href="/saved"
					class={`rounded-lg px-3 py-2 transition-colors ${
						path.startsWith("/saved")
							? "bg-surface-soft text-foreground"
							: "text-muted hover:bg-surface-soft hover:text-foreground"
					}`}
				>
					<span class="inline-flex items-center gap-2">
						Saved
						<Badge className="px-2 py-0.5 text-[10px]">{$savedResources.ids.length}</Badge>
					</span>
				</a>
			</nav>
		</div>
	</header>

	<main class="page-wrap">
		<slot />
	</main>
</div>
