<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { definitionsService, type Definition, type DefinitionPayload } from '$lib/services/definitions';
	import DefinitionForm from '$lib/components/DefinitionForm.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { ArrowLeft } from '@lucide/svelte';

	const id = page.params.id ?? '';
	const queryClient = useQueryClient();

	const defQuery = createQuery(() => ({
		queryKey: ['definition', id],
		queryFn: () => definitionsService.get(id)
	}));

	const def = $derived(defQuery.data as Definition | undefined);

	let loading = $state(false);
	let error = $state('');

	async function handleSubmit(data: DefinitionPayload) {
		loading = true;
		error = '';
		try {
			await definitionsService.update(id, data);
			queryClient.invalidateQueries({ queryKey: ['definitions'] });
			queryClient.invalidateQueries({ queryKey: ['definition', id] });
			goto('/definitions');
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'Error al actualizar';
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
		<h1 class="text-lg font-semibold text-[#e7e4ec]">
			{def ? `Editar: ${def.name}` : 'Editar definición'}
		</h1>
	</div>

	{#if error}
		<div class="mx-6 mt-4 rounded-md border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-400">
			{error}
		</div>
	{/if}

	{#if defQuery.isLoading}
		<div class="p-6 space-y-3">
			<Skeleton class="h-8 w-48 bg-[#19191d]" />
			<Skeleton class="h-96 w-full bg-[#19191d]" />
		</div>
	{:else if def}
		<div class="flex-1 overflow-hidden">
			<DefinitionForm
				initial={{
					name: def.name,
					description: def.description,
					runtime: def.runtime,
					runtime_version: def.runtime_version,
					code: def.code,
					packages: def.packages,
					timeout_seconds: def.timeout_seconds,
					max_retries: def.max_retries,
					backoff_multiplier: def.backoff_multiplier,
					max_concurrent: def.max_concurrent,
					memory_mb: def.memory_mb,
					cpu_shares: def.cpu_shares,
					network_enabled: def.network_enabled
				}}
				onsubmit={handleSubmit}
				{loading}
			/>
		</div>
	{:else}
		<div class="flex flex-col items-center justify-center py-24">
			<p class="text-sm text-[#75757c]">Definición no encontrada</p>
			<a href="/definitions" class="mt-2 text-xs text-[#d0bcff] hover:underline">← Volver</a>
		</div>
	{/if}
</div>
