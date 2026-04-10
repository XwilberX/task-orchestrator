<script lang="ts">
	import { goto } from '$app/navigation';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { definitionsService, type DefinitionPayload } from '$lib/services/definitions';
	import DefinitionForm from '$lib/components/DefinitionForm.svelte';
	import { ArrowLeft } from '@lucide/svelte';

	const queryClient = useQueryClient();
	let loading = $state(false);
	let error = $state('');

	async function handleSubmit(data: DefinitionPayload) {
		loading = true;
		error = '';
		try {
			await definitionsService.create(data);
			queryClient.invalidateQueries({ queryKey: ['definitions'] });
			goto('/definitions');
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Error al crear la definición';
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex h-full flex-col">
	<div class="flex items-center gap-3 border-b border-[#1f1f24] px-6 py-4">
		<a href="/definitions" class="rounded p-1 text-[#75757c] hover:bg-[#19191d] hover:text-[#e7e4ec] transition-colors">
			<ArrowLeft size={16} />
		</a>
		<h1 class="text-lg font-semibold text-[#e7e4ec]">Nueva definición</h1>
	</div>
	{#if error}
		<div class="mx-6 mt-4 rounded-md border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-400">
			{error}
		</div>
	{/if}
	<div class="flex-1 overflow-hidden">
		<DefinitionForm onsubmit={handleSubmit} {loading} />
	</div>
</div>
