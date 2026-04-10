<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { definitionsService, type Definition } from '$lib/services/definitions';
	import { tasksService } from '$lib/services/tasks';
	import RuntimeBadge from '$lib/components/RuntimeBadge.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { formatDistanceToNow } from 'date-fns';
	import { es } from 'date-fns/locale';
	import { Plus, Pencil, Trash2, Play, Clock, RefreshCw } from '@lucide/svelte';

	const queryClient = useQueryClient();

	const defs = createQuery(() => ({
		queryKey: ['definitions'],
		queryFn: definitionsService.list
	}));

	const defsData = $derived(defs.data as Definition[] | undefined);

	async function deleteDef(id: string, name: string) {
		if (!confirm(`¿Eliminar definición "${name}"?`)) return;
		await definitionsService.delete(id);
		queryClient.invalidateQueries({ queryKey: ['definitions'] });
	}

	async function dispatchDef(name: string) {
		await tasksService.dispatch({ definition: name });
		queryClient.invalidateQueries({ queryKey: ['tasks'] });
	}

	function relativeTime(date: string) {
		return formatDistanceToNow(new Date(date), { addSuffix: true, locale: es });
	}
</script>

<div class="flex h-full flex-col">
	<!-- Header -->
	<div class="flex items-center justify-between border-b border-[#1f1f24] px-6 py-4">
		<div>
			<h1 class="text-lg font-semibold text-[#e7e4ec]">Definiciones</h1>
			{#if defsData}
				<p class="font-mono text-xs text-[#75757c]">{defsData.length} definiciones</p>
			{/if}
		</div>
		<a
			href="/definitions/new"
			class="flex items-center gap-1.5 rounded-md bg-violet-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-violet-500 transition-colors"
		>
			<Plus size={14} />
			Nueva definición
		</a>
	</div>

	<!-- Tabla -->
	<div class="flex-1 overflow-auto">
		{#if defs.isLoading}
			<div class="space-y-px p-4">
				{#each { length: 6 } as _}
					<Skeleton class="h-14 w-full rounded bg-[#19191d]" />
				{/each}
			</div>
		{:else if (defsData?.length ?? 0) === 0}
			<div class="flex flex-col items-center justify-center py-24 text-center">
				<RefreshCw size={32} class="mb-3 text-[#3d3b3e]" />
				<p class="text-sm text-[#75757c]">No hay definiciones aún</p>
				<a href="/definitions/new" class="mt-2 text-xs text-[#d0bcff] hover:underline">
					Crear la primera →
				</a>
			</div>
		{:else}
			<table class="w-full text-sm">
				<thead class="sticky top-0 bg-[#0e0e10]">
					<tr class="border-b border-[#1f1f24]">
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Nombre</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Runtime</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Timeout</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Reintentos</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Creada</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Acciones</th>
					</tr>
				</thead>
				<tbody>
					{#each defsData ?? [] as def}
						<tr class="border-b border-[#1f1f24] last:border-0 hover:bg-[#19191d] transition-colors">
							<td class="px-4 py-3">
								<p class="font-medium text-[#e7e4ec]">{def.name}</p>
								{#if def.description}
									<p class="text-xs text-[#75757c] truncate max-w-xs">{def.description}</p>
								{/if}
							</td>
							<td class="px-4 py-3"><RuntimeBadge runtime={def.runtime} /></td>
							<td class="px-4 py-3">
								<span class="flex items-center gap-1 font-mono text-xs text-[#a09da1]">
									<Clock size={11} />
									{def.timeout_seconds}s
								</span>
							</td>
							<td class="px-4 py-3">
								<span class="font-mono text-xs text-[#a09da1]">{def.max_retries}</span>
							</td>
							<td class="px-4 py-3">
								<span class="text-xs text-[#75757c]">{relativeTime(def.created_at)}</span>
							</td>
							<td class="px-4 py-3">
								<div class="flex items-center gap-1">
									<button
										onclick={() => dispatchDef(def.name)}
										class="rounded p-1.5 text-[#75757c] hover:bg-green-500/10 hover:text-green-400 transition-colors"
										title="Disparar ahora"
									>
										<Play size={14} />
									</button>
									<a
										href="/definitions/{def.id}"
										class="rounded p-1.5 text-[#75757c] hover:bg-[#2b2c32] hover:text-[#e7e4ec] transition-colors"
										title="Editar"
									>
										<Pencil size={14} />
									</a>
									<button
										onclick={() => deleteDef(def.id, def.name)}
										class="rounded p-1.5 text-[#75757c] hover:bg-red-500/10 hover:text-red-400 transition-colors"
										title="Eliminar"
									>
										<Trash2 size={14} />
									</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>
</div>
