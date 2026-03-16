<script lang="ts">
	import Button from "$lib/components/ui/button.svelte";
	import Card from "$lib/components/ui/card.svelte";
	import Input from "$lib/components/ui/input.svelte";

	export let levelOptions: string[] = [];
	export let skillOptions: string[] = [];
	export let topicOptions: string[] = [];
	export let providerOptions: { slug: string; name: string }[] = [];
	export let typeOptions: string[] = [];
	export let level = "";
	export let skill = "";
	export let topic = "";
	export let provider = "";
	export let type = "";
	export let query = "";
	export let free: boolean | null = null;
</script>

<Card className="space-y-4">
	<form method="GET" action="/resources" class="space-y-4">
		<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-6">
			<label class="space-y-1 text-xs font-medium uppercase tracking-[0.08em] text-muted">
				<span>Search</span>
				<Input name="q" value={query} placeholder="Topic, source, or title" />
			</label>

			<label class="space-y-1 text-xs font-medium uppercase tracking-[0.08em] text-muted">
				<span>CEFR level</span>
				<select
					name="level"
					value={level}
					class="h-10 w-full rounded-xl border border-border bg-surface px-3 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/80"
				>
					<option value="">All levels</option>
					{#each levelOptions as option}
						<option value={option}>{option}</option>
					{/each}
				</select>
			</label>

			<label class="space-y-1 text-xs font-medium uppercase tracking-[0.08em] text-muted">
				<span>Skill</span>
				<select
					name="skill"
					value={skill}
					class="h-10 w-full rounded-xl border border-border bg-surface px-3 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/80"
				>
					<option value="">All skills</option>
					{#each skillOptions as option}
						<option value={option}>{option}</option>
					{/each}
				</select>
			</label>

			<label class="space-y-1 text-xs font-medium uppercase tracking-[0.08em] text-muted">
				<span>Topic</span>
				<select
					name="topic"
					value={topic}
					class="h-10 w-full rounded-xl border border-border bg-surface px-3 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/80"
				>
					<option value="">All topics</option>
					{#each topicOptions as option}
						<option value={option}>{option}</option>
					{/each}
				</select>
			</label>

			<label class="space-y-1 text-xs font-medium uppercase tracking-[0.08em] text-muted">
				<span>Provider</span>
				<select
					name="provider"
					value={provider}
					class="h-10 w-full rounded-xl border border-border bg-surface px-3 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/80"
				>
					<option value="">All providers</option>
					{#each providerOptions as option}
						<option value={option.slug}>{option.name}</option>
					{/each}
				</select>
			</label>

			<label class="space-y-1 text-xs font-medium uppercase tracking-[0.08em] text-muted">
				<span>Type</span>
				<select
					name="type"
					value={type}
					class="h-10 w-full rounded-xl border border-border bg-surface px-3 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/80"
				>
					<option value="">All types</option>
					{#each typeOptions as option}
						<option value={option}>{option}</option>
					{/each}
				</select>
			</label>
		</div>

		<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
			<label class="space-y-1 text-xs font-medium uppercase tracking-[0.08em] text-muted">
				<span>Price filter</span>
				<select
					name="free"
					value={free === null ? "" : String(free)}
					class="h-10 min-w-44 rounded-xl border border-border bg-surface px-3 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/80"
				>
					<option value="">Free + paid</option>
					<option value="true">Free only</option>
					<option value="false">Paid only</option>
				</select>
			</label>
			<div class="flex gap-2">
				<Button type="submit" size="sm">Apply filters</Button>
				<a
					href="/resources"
					class="inline-flex h-8 items-center justify-center rounded-xl px-3 text-xs font-semibold text-muted transition-colors hover:bg-surface-soft hover:text-foreground"
				>
					Clear
				</a>
			</div>
		</div>
	</form>
</Card>
