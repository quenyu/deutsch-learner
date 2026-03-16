<script lang="ts">
	import { updateProfile } from "$lib/api/resources";
	import Button from "$lib/components/ui/button.svelte";
	import Card from "$lib/components/ui/card.svelte";
	import Input from "$lib/components/ui/input.svelte";
	import type { PageData } from "./$types";

	export let data: PageData;

	let displayName = data.profile?.displayName ?? "";
	let targetLevel = data.profile?.targetLevel ?? "";
	let learningGoals = data.profile?.learningGoals ?? "";
	let selectedResourceTypes = [...(data.profile?.preferredResourceTypes ?? [])];
	let selectedSkills = [...(data.profile?.preferredSkills ?? [])];
	let selectedProviders = [...(data.profile?.preferredSourceProviders ?? [])];

	let saveError: string | null = null;
	let saveMessage: string | null = null;
	let saving = false;

	function toggleSelection(list: string[], value: string) {
		return list.includes(value) ? list.filter((item) => item !== value) : [...list, value];
	}

	async function saveProfile() {
		saving = true;
		saveError = null;
		saveMessage = null;

		const result = await updateProfile(fetch, {
			displayName,
			targetLevel: targetLevel ? targetLevel : null,
			learningGoals,
			preferredResourceTypes: selectedResourceTypes,
			preferredSkills: selectedSkills,
			preferredSourceProviders: selectedProviders
		});
		saving = false;

		if (!result.ok || !result.item) {
			saveError = result.error ?? "Could not update profile.";
			return;
		}

		displayName = result.item.displayName;
		targetLevel = result.item.targetLevel ?? "";
		learningGoals = result.item.learningGoals;
		selectedResourceTypes = [...result.item.preferredResourceTypes];
		selectedSkills = [...result.item.preferredSkills];
		selectedProviders = [...result.item.preferredSourceProviders];
		saveMessage = "Profile saved. Catalog defaults will use these preferences when no URL filters are set.";
	}
</script>

<section class="space-y-6">
	<div class="space-y-2">
		<p class="text-xs uppercase tracking-[0.14em] text-accent">Personalization</p>
		<h1 class="text-3xl font-semibold leading-tight text-foreground sm:text-4xl">Learning Profile</h1>
		<p class="max-w-2xl text-sm leading-relaxed text-muted sm:text-base">
			Set stable preferences so catalog discovery starts from your level, goals, and trusted source mix.
		</p>
	</div>

	{#if data.loadError}
		<div class="rounded-xl border border-border bg-surface-soft/85 p-3 text-sm text-muted" role="status" aria-live="polite">
			{data.loadError}
		</div>
	{/if}

	<Card className="space-y-5">
		<form class="space-y-5" on:submit|preventDefault={saveProfile}>
			<div class="grid gap-4 sm:grid-cols-2">
				<label class="space-y-1 text-xs font-medium uppercase tracking-[0.08em] text-muted">
					<span>Display name</span>
					<Input bind:value={displayName} required />
				</label>

				<label class="space-y-1 text-xs font-medium uppercase tracking-[0.08em] text-muted">
					<span>Target level</span>
					<select
						bind:value={targetLevel}
						class="h-10 w-full rounded-xl border border-border bg-surface px-3 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/80"
					>
						<option value="">No preference</option>
						{#each data.levelOptions as option}
							<option value={option}>{option}</option>
						{/each}
					</select>
				</label>
			</div>

			<label class="space-y-1 text-xs font-medium uppercase tracking-[0.08em] text-muted">
				<span>Learning goals</span>
				<textarea
					bind:value={learningGoals}
					rows="3"
					class="w-full rounded-xl border border-border bg-surface px-3 py-2 text-sm text-foreground placeholder:text-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/80"
					placeholder="Example: Improve listening confidence for everyday conversations"
				></textarea>
			</label>

			<fieldset class="space-y-2">
				<legend class="text-xs font-medium uppercase tracking-[0.08em] text-muted">Preferred resource types</legend>
				<div class="grid gap-2 sm:grid-cols-2">
					{#each data.resourceTypeOptions as option}
						<label class="flex items-center gap-2 rounded-lg border border-border bg-surface-soft px-3 py-2 text-sm">
							<input
								type="checkbox"
								checked={selectedResourceTypes.includes(option)}
								on:change={() => (selectedResourceTypes = toggleSelection(selectedResourceTypes, option))}
							/>
							<span>{option}</span>
						</label>
					{/each}
				</div>
			</fieldset>

			<fieldset class="space-y-2">
				<legend class="text-xs font-medium uppercase tracking-[0.08em] text-muted">Preferred skills</legend>
				<div class="grid gap-2 sm:grid-cols-2">
					{#each data.skillOptions as option}
						<label class="flex items-center gap-2 rounded-lg border border-border bg-surface-soft px-3 py-2 text-sm">
							<input
								type="checkbox"
								checked={selectedSkills.includes(option)}
								on:change={() => (selectedSkills = toggleSelection(selectedSkills, option))}
							/>
							<span>{option}</span>
						</label>
					{/each}
				</div>
			</fieldset>

			<fieldset class="space-y-2">
				<legend class="text-xs font-medium uppercase tracking-[0.08em] text-muted">Preferred source providers</legend>
				<div class="grid gap-2 sm:grid-cols-2">
					{#each data.sourceProviderOptions as provider}
						<label class="flex items-center gap-2 rounded-lg border border-border bg-surface-soft px-3 py-2 text-sm">
							<input
								type="checkbox"
								checked={selectedProviders.includes(provider.slug)}
								on:change={() => (selectedProviders = toggleSelection(selectedProviders, provider.slug))}
							/>
							<span>{provider.name}</span>
						</label>
					{/each}
				</div>
			</fieldset>

			<div class="flex items-center gap-3">
				<Button type="submit" disabled={saving}>{saving ? "Saving..." : "Save profile"}</Button>
				{#if saveMessage}
					<p class="text-sm text-success" role="status" aria-live="polite">{saveMessage}</p>
				{/if}
			</div>
			{#if saveError}
				<p class="rounded-lg border border-border bg-surface-soft p-3 text-sm text-muted" role="alert">{saveError}</p>
			{/if}
		</form>
	</Card>
</section>
