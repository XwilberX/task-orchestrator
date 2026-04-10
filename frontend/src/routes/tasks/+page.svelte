<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { tasksService, type Task, type TaskFilter } from '$lib/services/tasks';
	import TaskStatusBadge from '$lib/components/TaskStatusBadge.svelte';
	import RuntimeBadge from '$lib/components/RuntimeBadge.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { formatDistanceToNow } from 'date-fns';
	import { es } from 'date-fns/locale';
	import { Filter, X, Ban } from '@lucide/svelte';

	const queryClient = useQueryClient();

	// ─── Filtros ───────────────────────────────────────────────────────────────
	const STATUSES: Task['status'][] = ['PENDING','QUEUED','RUNNING','SUCCESS','FAILED','TIMEOUT','CANCELLED'];
	const RUNTIMES = ['python','nodejs','typescript','go','java'];

	let selectedStatuses = $state<Set<string>>(new Set());
	let selectedRuntimes = $state<Set<string>>(new Set());
	let fromDate = $state('');
	let toDate = $state('');

	function toggleStatus(s: string) {
		const next = new Set(selectedStatuses);
		next.has(s) ? next.delete(s) : next.add(s);
		selectedStatuses = next;
		currentPage = 1;
	}

	function toggleRuntime(r: string) {
		const next = new Set(selectedRuntimes);
		next.has(r) ? next.delete(r) : next.add(r);
		selectedRuntimes = next;
		currentPage = 1;
	}

	function clearFilters() {
		selectedStatuses = new Set();
		selectedRuntimes = new Set();
		fromDate = '';
		toDate = '';
		currentPage = 1;
	}

	const hasFilters = $derived(
		selectedStatuses.size > 0 || selectedRuntimes.size > 0 || !!fromDate || !!toDate
	);

	// ─── Query ─────────────────────────────────────────────────────────────────
	const filter = $derived<TaskFilter>({
		status: [...selectedStatuses].join(',') || undefined,
		runtime: [...selectedRuntimes].join(',') || undefined,
		from: fromDate || undefined,
		to: toDate || undefined
	});

	const tasks = createQuery(() => ({
		queryKey: ['tasks', filter],
		queryFn: () => tasksService.list(filter),
		refetchInterval: 10_000
	}));

	const tasksData = $derived(tasks.data as Task[] | undefined);

	// ─── Paginación cliente ────────────────────────────────────────────────────
	const PAGE_SIZE = 20;
	let currentPage = $state(1);

	const totalPages = $derived(Math.max(1, Math.ceil((tasksData?.length ?? 0) / PAGE_SIZE)));
	const paginated = $derived((tasksData ?? []).slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE));

	// ─── Cancelar tarea ───────────────────────────────────────────────────────
	async function cancelTask(id: string, e: MouseEvent) {
		e.stopPropagation();
		if (!confirm('¿Cancelar esta tarea?')) return;
		await tasksService.cancel(id);
		queryClient.invalidateQueries({ queryKey: ['tasks'] });
	}

	// ─── Helpers ──────────────────────────────────────────────────────────────
	function duration(started?: string, finished?: string): string {
		if (!started) return '—';
		const s = new Date(started);
		const f = finished ? new Date(finished) : new Date();
		const ms = f.getTime() - s.getTime();
		if (ms < 1000) return `${ms}ms`;
		if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
		return `${Math.floor(ms / 60_000)}m ${Math.floor((ms % 60_000) / 1000)}s`;
	}

	function relativeTime(date?: string): string {
		if (!date) return '—';
		return formatDistanceToNow(new Date(date), { addSuffix: true, locale: es });
	}
</script>

<div class="flex h-full flex-col">
	<!-- Header -->
	<div class="flex items-center justify-between border-b border-[#1f1f24] px-6 py-4">
		<div>
			<h1 class="text-lg font-semibold text-[#e7e4ec]">Tareas</h1>
			{#if tasksData}
				<p class="font-mono text-xs text-[#75757c]">{tasksData.length} ejecuciones</p>
			{/if}
		</div>
	</div>

	<!-- Filtros -->
	<div class="flex flex-wrap items-center gap-3 border-b border-[#1f1f24] bg-[#0e0e10] px-6 py-3">
		<Filter size={14} class="text-[#75757c]" />

		<!-- Status multi-select -->
		<div class="flex flex-wrap gap-1.5">
			{#each STATUSES as status}
				<button
					onclick={() => toggleStatus(status)}
					class="rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors
						{selectedStatuses.has(status)
							? 'border-violet-500/50 bg-violet-500/10 text-violet-300'
							: 'border-[#1f1f24] bg-[#19191d] text-[#75757c] hover:border-[#3d3b3e] hover:text-[#a09da1]'}"
				>
					{status}
				</button>
			{/each}
		</div>

		<div class="h-4 w-px bg-[#1f1f24]"></div>

		<!-- Runtime multi-select -->
		<div class="flex flex-wrap gap-1.5">
			{#each RUNTIMES as runtime}
				<button
					onclick={() => toggleRuntime(runtime)}
					class="rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors
						{selectedRuntimes.has(runtime)
							? 'border-violet-500/50 bg-violet-500/10 text-violet-300'
							: 'border-[#1f1f24] bg-[#19191d] text-[#75757c] hover:border-[#3d3b3e] hover:text-[#a09da1]'}"
				>
					{runtime}
				</button>
			{/each}
		</div>

		<div class="h-4 w-px bg-[#1f1f24]"></div>

		<!-- Rango de fechas -->
		<div class="flex items-center gap-2">
			<input
				type="date"
				bind:value={fromDate}
				class="rounded-md border border-[#1f1f24] bg-[#19191d] px-2 py-1 font-mono text-xs text-[#a09da1] focus:border-violet-500/50 focus:outline-none"
				placeholder="Desde"
			/>
			<span class="text-xs text-[#75757c]">→</span>
			<input
				type="date"
				bind:value={toDate}
				class="rounded-md border border-[#1f1f24] bg-[#19191d] px-2 py-1 font-mono text-xs text-[#a09da1] focus:border-violet-500/50 focus:outline-none"
				placeholder="Hasta"
			/>
		</div>

		{#if hasFilters}
			<button
				onclick={clearFilters}
				class="flex items-center gap-1 rounded-md px-2 py-1 text-xs text-[#75757c] hover:text-[#e7e4ec] transition-colors"
			>
				<X size={12} />
				Limpiar
			</button>
		{/if}
	</div>

	<!-- Tabla -->
	<div class="flex-1 overflow-auto">
		{#if tasks.isLoading}
			<div class="space-y-px p-4">
				{#each { length: 12 } as _}
					<Skeleton class="h-11 w-full rounded bg-[#19191d]" />
				{/each}
			</div>
		{:else if (tasksData?.length ?? 0) === 0}
			<div class="flex flex-col items-center justify-center py-24 text-center">
				<Filter size={32} class="mb-3 text-[#3d3b3e]" />
				<p class="text-sm text-[#75757c]">
					{hasFilters ? 'No hay tareas con estos filtros' : 'No hay ejecuciones aún'}
				</p>
				{#if hasFilters}
					<button onclick={clearFilters} class="mt-2 text-xs text-[#d0bcff] hover:underline">
						Limpiar filtros
					</button>
				{/if}
			</div>
		{:else}
			<table class="w-full text-sm">
				<thead class="sticky top-0 bg-[#0e0e10]">
					<tr class="border-b border-[#1f1f24]">
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Estado</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Task ID</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Definición</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Runtime</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Intento</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Duración</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Iniciada</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]"></th>
					</tr>
				</thead>
				<tbody>
					{#each paginated as task}
						<tr
							class="border-b border-[#1f1f24] last:border-0 hover:bg-[#19191d] transition-colors cursor-pointer"
							onclick={() => { window.location.href = `/tasks/${task.id}`; }}
						>
							<td class="px-4 py-3"><TaskStatusBadge status={task.status} /></td>
							<td class="px-4 py-3">
								<span class="font-mono text-xs text-[#a09da1]">#{task.id.slice(0, 8)}</span>
							</td>
							<td class="px-4 py-3 text-[#e7e4ec]">
								{#if task.definition}
									{task.definition}
								{:else}
									<span class="italic text-[#75757c]">ad-hoc</span>
								{/if}
							</td>
							<td class="px-4 py-3"><RuntimeBadge runtime={task.runtime} /></td>
							<td class="px-4 py-3">
								<span class="font-mono text-xs text-[#75757c]">{task.attempt}/{task.max_retries}</span>
							</td>
							<td class="px-4 py-3">
								<span class="font-mono text-xs text-[#a09da1]">{duration(task.started_at, task.finished_at)}</span>
							</td>
							<td class="px-4 py-3">
								<span class="text-xs text-[#75757c]">{relativeTime(task.created_at)}</span>
							</td>
							<td class="px-4 py-3">
								{#if task.status === 'PENDING' || task.status === 'QUEUED'}
									<button
										onclick={(e) => cancelTask(task.id, e)}
										class="rounded p-1 text-[#75757c] hover:bg-red-500/10 hover:text-red-400 transition-colors"
										title="Cancelar tarea"
									>
										<Ban size={14} />
									</button>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>

	<!-- Paginación -->
	{#if (tasksData?.length ?? 0) > PAGE_SIZE}
		<div class="flex items-center justify-between border-t border-[#1f1f24] px-6 py-3">
			<p class="font-mono text-xs text-[#75757c]">
				{(currentPage - 1) * PAGE_SIZE + 1}–{Math.min(currentPage * PAGE_SIZE, tasksData?.length ?? 0)}
				de {tasksData?.length} tareas
			</p>
			<div class="flex items-center gap-2">
				<button
					onclick={() => currentPage--}
					disabled={currentPage === 1}
					class="rounded-md border border-[#1f1f24] px-3 py-1 text-xs text-[#a09da1] hover:bg-[#19191d] disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
				>
					← Anterior
				</button>
				<span class="font-mono text-xs text-[#75757c]">{currentPage} / {totalPages}</span>
				<button
					onclick={() => currentPage++}
					disabled={currentPage === totalPages}
					class="rounded-md border border-[#1f1f24] px-3 py-1 text-xs text-[#a09da1] hover:bg-[#19191d] disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
				>
					Siguiente →
				</button>
			</div>
		</div>
	{/if}
</div>
