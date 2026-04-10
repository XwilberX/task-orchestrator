<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { definitionsService, type Definition } from '$lib/services/definitions';
	import { tasksService } from '$lib/services/tasks';
	import RuntimeBadge from '$lib/components/RuntimeBadge.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { formatDistanceToNow } from 'date-fns';
	import { es } from 'date-fns/locale';
	import { Plus, Pencil, Trash2, Play, Clock, Search, Code2, ChevronLeft, ChevronRight } from '@lucide/svelte';

	const queryClient = useQueryClient();

	const defs = createQuery(() => ({
		queryKey: ['definitions'],
		queryFn: definitionsService.list
	}));

	const defsData = $derived(defs.data as Definition[] | undefined);

	// ─── Search ────────────────────────────────────────────────────────────────
	let search = $state('');

	const filtered = $derived(
		(defsData ?? []).filter(d =>
			!search.trim() ||
			d.name.toLowerCase().includes(search.toLowerCase()) ||
			d.description?.toLowerCase().includes(search.toLowerCase()) ||
			d.runtime.toLowerCase().includes(search.toLowerCase())
		)
	);

	// ─── Stats ─────────────────────────────────────────────────────────────────
	const totalDefs = $derived(defsData?.length ?? 0);
	const uniqueRuntimes = $derived(new Set(defsData?.map(d => d.runtime) ?? []).size);
	const recentDefs = $derived(
		(defsData ?? []).filter(d => {
			const diff = Date.now() - new Date(d.created_at).getTime();
			return diff < 7 * 24 * 60 * 60 * 1000;
		}).length
	);

	// ─── Paginación ────────────────────────────────────────────────────────────
	const PAGE_SIZE = 10;
	let currentPage = $state(1);

	$effect(() => { search; currentPage = 1; });

	const totalPages = $derived(Math.max(1, Math.ceil(filtered.length / PAGE_SIZE)));
	const paginated = $derived(filtered.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE));
	const start = $derived((currentPage - 1) * PAGE_SIZE + 1);
	const end = $derived(Math.min(currentPage * PAGE_SIZE, filtered.length));

	// ─── Acciones ──────────────────────────────────────────────────────────────
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

	const RUNTIME_COLORS: Record<string, string> = {
		python: 'text-blue-400',
		nodejs: 'text-green-400',
		go: 'text-cyan-400',
		java: 'text-orange-400',
	};
</script>

<div class="flex h-full flex-col">

	<!-- Header -->
	<div class="border-b border-[#1f1f24] px-6 py-5">
		<div class="flex items-start justify-between gap-4">
			<div>
				<h1 class="text-xl font-semibold text-[#e7e4ec]">Definiciones</h1>
				<p class="mt-0.5 text-sm text-[#75757c]">Configura y gestiona tus templates de automatización</p>
			</div>
			<a
				href="/definitions/new"
				class="flex shrink-0 items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-violet-500"
			>
				<Plus size={15} />
				Nueva definición
			</a>
		</div>

		<!-- Stats -->
		<div class="mt-5 grid grid-cols-3 gap-4">
			<div class="rounded-xl border border-[#1f1f24] bg-[#19191d] px-5 py-4">
				<p class="text-xs text-[#75757c]">Total</p>
				<p class="mt-1 font-mono text-3xl font-semibold text-[#e7e4ec]">{totalDefs}</p>
				<p class="mt-0.5 text-xs text-[#3d3b3e]">definiciones registradas</p>
			</div>
			<div class="rounded-xl border border-[#1f1f24] bg-[#19191d] px-5 py-4">
				<p class="text-xs text-[#75757c]">Runtimes</p>
				<p class="mt-1 font-mono text-3xl font-semibold text-[#d0bcff]">{uniqueRuntimes}</p>
				<p class="mt-0.5 text-xs text-[#3d3b3e]">
					{[...new Set(defsData?.map(d => d.runtime) ?? [])].join(', ') || '—'}
				</p>
			</div>
			<div class="rounded-xl border border-[#1f1f24] bg-[#19191d] px-5 py-4">
				<p class="text-xs text-[#75757c]">Esta semana</p>
				<p class="mt-1 font-mono text-3xl font-semibold text-[#e7e4ec]">{recentDefs}</p>
				<p class="mt-0.5 text-xs text-[#3d3b3e]">creadas en los últimos 7 días</p>
			</div>
		</div>

		<!-- Search -->
		<div class="mt-4 relative">
			<Search size={14} class="absolute left-3 top-1/2 -translate-y-1/2 text-[#3d3b3e]" />
			<input
				bind:value={search}
				placeholder="Buscar por nombre, descripción o runtime..."
				class="w-full rounded-lg border border-[#1f1f24] bg-[#19191d] py-2 pl-9 pr-4 text-sm text-[#e7e4ec] placeholder-[#3d3b3e] focus:border-violet-500/50 focus:outline-none transition-colors"
			/>
		</div>
	</div>

	<!-- Tabla -->
	<div class="flex-1 overflow-auto">
		{#if defs.isLoading}
			<div class="space-y-2 p-6">
				{#each { length: 5 } as _}
					<Skeleton class="h-16 w-full rounded-xl bg-[#19191d]" />
				{/each}
			</div>
		{:else if (defsData?.length ?? 0) === 0}
			<div class="flex flex-col items-center justify-center py-24 text-center">
				<div class="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl border border-[#1f1f24] bg-[#19191d]">
					<Code2 size={24} class="text-[#3d3b3e]" />
				</div>
				<p class="text-sm font-medium text-[#75757c]">No hay definiciones aún</p>
				<a href="/definitions/new" class="mt-2 text-xs text-[#d0bcff] hover:underline">
					Crear la primera →
				</a>
			</div>
		{:else if filtered.length === 0}
			<div class="flex flex-col items-center justify-center py-16 text-center">
				<Search size={28} class="mb-3 text-[#3d3b3e]" />
				<p class="text-sm text-[#75757c]">Sin resultados para "{search}"</p>
				<button onclick={() => search = ''} class="mt-2 text-xs text-[#d0bcff] hover:underline">
					Limpiar búsqueda
				</button>
			</div>
		{:else}
			<table class="w-full text-sm">
				<thead class="sticky top-0 bg-[#0e0e10]">
					<tr class="border-b border-[#1f1f24]">
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-[#3d3b3e]">Nombre</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-[#3d3b3e]">Runtime</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-[#3d3b3e]">Paquetes</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-[#3d3b3e]">Timeout</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-[#3d3b3e]">Red</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-[#3d3b3e]">Creada</th>
						<th class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-[#3d3b3e]">Acciones</th>
					</tr>
				</thead>
				<tbody>
					{#each paginated as def}
						<tr class="border-b border-[#1f1f24] last:border-0 transition-colors hover:bg-[#19191d]">
							<td class="px-6 py-4">
								<p class="font-medium text-[#e7e4ec]">{def.name}</p>
								{#if def.description}
									<p class="mt-0.5 max-w-xs truncate text-xs text-[#75757c]">{def.description}</p>
								{/if}
							</td>
							<td class="px-6 py-4">
								<div class="flex items-center gap-2">
									<RuntimeBadge runtime={def.runtime} />
									{#if def.runtime_version}
										<span class="font-mono text-xs text-[#3d3b3e]">{def.runtime_version}</span>
									{/if}
								</div>
							</td>
							<td class="px-6 py-4">
								{#if def.packages}
									<span class="font-mono text-xs text-[#75757c] truncate max-w-[120px] block">{def.packages}</span>
								{:else}
									<span class="text-xs text-[#2e2d30]">—</span>
								{/if}
							</td>
							<td class="px-6 py-4">
								<span class="flex items-center gap-1 font-mono text-xs text-[#75757c]">
									<Clock size={11} />
									{def.timeout_seconds}s
								</span>
							</td>
							<td class="px-6 py-4">
								{#if def.network_enabled}
									<span class="rounded-full bg-green-500/10 px-2 py-0.5 text-xs text-green-400">sí</span>
								{:else}
									<span class="rounded-full bg-[#19191d] px-2 py-0.5 text-xs text-[#3d3b3e]">no</span>
								{/if}
							</td>
							<td class="px-6 py-4">
								<span class="text-xs text-[#75757c]">{relativeTime(def.created_at)}</span>
							</td>
							<td class="px-6 py-4">
								<div class="flex items-center justify-end gap-1">
									<button
										onclick={() => dispatchDef(def.name)}
										title="Ejecutar ahora"
										class="rounded-lg p-1.5 text-[#75757c] transition-colors hover:bg-green-500/10 hover:text-green-400"
									>
										<Play size={14} />
									</button>
									<a
										href="/definitions/{def.id}"
										title="Editar"
										class="rounded-lg p-1.5 text-[#75757c] transition-colors hover:bg-[#2b2c32] hover:text-[#e7e4ec]"
									>
										<Pencil size={14} />
									</a>
									<button
										onclick={() => deleteDef(def.id, def.name)}
										title="Eliminar"
										class="rounded-lg p-1.5 text-[#75757c] transition-colors hover:bg-red-500/10 hover:text-red-400"
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

	<!-- Paginación -->
	{#if filtered.length > 0}
		<div class="flex items-center justify-between border-t border-[#1f1f24] px-6 py-3">
			<p class="font-mono text-xs text-[#3d3b3e]">
				Mostrando <span class="text-[#75757c]">{start}–{end}</span> de <span class="text-[#75757c]">{filtered.length}</span> definiciones
			</p>
			<div class="flex items-center gap-1">
				<button
					onclick={() => currentPage--}
					disabled={currentPage === 1}
					class="flex h-7 w-7 items-center justify-center rounded-lg border border-[#1f1f24] text-[#75757c] transition-colors hover:bg-[#19191d] hover:text-[#e7e4ec] disabled:cursor-not-allowed disabled:opacity-30"
				>
					<ChevronLeft size={13} />
				</button>
				{#each Array.from({ length: totalPages }, (_, i) => i + 1) as p}
					<button
						onclick={() => currentPage = p}
						class="flex h-7 w-7 items-center justify-center rounded-lg font-mono text-xs transition-colors
							{p === currentPage
								? 'bg-violet-600 text-white'
								: 'border border-[#1f1f24] text-[#75757c] hover:bg-[#19191d] hover:text-[#e7e4ec]'}"
					>
						{p}
					</button>
				{/each}
				<button
					onclick={() => currentPage++}
					disabled={currentPage === totalPages}
					class="flex h-7 w-7 items-center justify-center rounded-lg border border-[#1f1f24] text-[#75757c] transition-colors hover:bg-[#19191d] hover:text-[#e7e4ec] disabled:cursor-not-allowed disabled:opacity-30"
				>
					<ChevronRight size={13} />
				</button>
			</div>
		</div>
	{/if}
</div>
