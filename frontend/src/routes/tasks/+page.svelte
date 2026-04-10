<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { tasksService, type Task, type TaskFilter } from '$lib/services/tasks';
	import TaskStatusBadge from '$lib/components/TaskStatusBadge.svelte';
	import RuntimeBadge from '$lib/components/RuntimeBadge.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { formatDistanceToNow } from 'date-fns';
	import { es } from 'date-fns/locale';
	import { Zap, Ban, ChevronLeft, ChevronRight, MoreVertical } from '@lucide/svelte';
	import DateRangePicker from '$lib/components/DateRangePicker.svelte';

	const queryClient = useQueryClient();

	// ─── Filtros ───────────────────────────────────────────────────────────────
	const STATUSES: (Task['status'] | 'ALL')[] = ['ALL','PENDING','QUEUED','RUNNING','SUCCESS','FAILED','TIMEOUT','CANCELLED'];
	const RUNTIMES = ['ANY','python','nodejs','go','java'];

	let selectedStatus = $state<string>('ALL');
	let selectedRuntime = $state<string>('ANY');
	let fromDate = $state('');
	let toDate = $state('');

	const hasFilters = $derived(
		selectedStatus !== 'ALL' || selectedRuntime !== 'ANY' || !!fromDate || !!toDate
	);

	function clearFilters() {
		selectedStatus = 'ALL';
		selectedRuntime = 'ANY';
		fromDate = '';
		toDate = '';
		currentPage = 1;
	}

	// ─── Query ─────────────────────────────────────────────────────────────────
	const filter = $derived<TaskFilter>({
		status: selectedStatus !== 'ALL' ? selectedStatus : undefined,
		runtime: selectedRuntime !== 'ANY' ? selectedRuntime : undefined,
		from: fromDate || undefined,
		to: toDate || undefined,
	});

	const tasks = createQuery(() => ({
		queryKey: ['tasks', filter],
		queryFn: () => tasksService.list(filter),
		refetchInterval: 10_000
	}));

	const tasksData = $derived(tasks.data as Task[] | undefined);

	// ─── Paginación ────────────────────────────────────────────────────────────
	let pageSize = $state(10);
	let currentPage = $state(1);

	$effect(() => { filter; pageSize; currentPage = 1; });

	const totalPages = $derived(Math.max(1, Math.ceil((tasksData?.length ?? 0) / pageSize)));
	const paginated = $derived((tasksData ?? []).slice((currentPage - 1) * pageSize, currentPage * pageSize));
	const start = $derived((currentPage - 1) * pageSize + 1);
	const end = $derived(Math.min(currentPage * pageSize, tasksData?.length ?? 0));

	// Páginas visibles (máx 7)
	const visiblePages = $derived(() => {
		const total = totalPages;
		if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
		if (currentPage <= 4) return Array.from({ length: 7 }, (_, i) => i + 1);
		if (currentPage >= total - 3) return Array.from({ length: 7 }, (_, i) => total - 6 + i);
		return Array.from({ length: 7 }, (_, i) => currentPage - 3 + i);
	});

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
		const ms = (finished ? new Date(finished) : new Date()).getTime() - new Date(started).getTime();
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
		<div class="flex items-center gap-3">
			<h1 class="text-lg font-semibold text-[#e7e4ec]">Tasks</h1>
			{#if tasksData}
				<span class="rounded-md border border-[#1f1f24] bg-[#19191d] px-2 py-0.5 font-mono text-xs text-[#75757c]">
					{tasksData.length} ejecuciones
				</span>
			{/if}
		</div>
		<a
			href="/definitions"
			class="flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-violet-500"
		>
			<Zap size={14} />
			Nueva tarea
		</a>
	</div>

	<!-- Filtros -->
	<div class="flex flex-wrap items-center gap-2 border-b border-[#1f1f24] px-6 py-3">
		<!-- Status dropdown -->
		<div class="flex items-center gap-1.5 rounded-lg border border-[#1f1f24] bg-[#19191d] px-3 py-1.5">
			<span class="text-xs text-[#75757c]">Estado:</span>
			<select
				bind:value={selectedStatus}
				class="bg-transparent font-mono text-xs text-[#e7e4ec] focus:outline-none cursor-pointer"
			>
				{#each STATUSES as s}
					<option value={s} class="bg-[#19191d]">{s === 'ALL' ? 'Todos' : s}</option>
				{/each}
			</select>
		</div>

		<!-- Runtime dropdown -->
		<div class="flex items-center gap-1.5 rounded-lg border border-[#1f1f24] bg-[#19191d] px-3 py-1.5">
			<span class="text-xs text-[#75757c]">Runtime:</span>
			<select
				bind:value={selectedRuntime}
				class="bg-transparent font-mono text-xs text-[#e7e4ec] focus:outline-none cursor-pointer"
			>
				{#each RUNTIMES as r}
					<option value={r} class="bg-[#19191d]">{r === 'ANY' ? 'Cualquiera' : r}</option>
				{/each}
			</select>
		</div>

		<!-- Rango fechas -->
		<DateRangePicker bind:fromDate bind:toDate />

		{#if hasFilters}
			<button
				onclick={clearFilters}
				class="text-xs text-[#75757c] underline-offset-2 hover:text-[#a09da1] transition-colors"
			>
				Limpiar
			</button>
		{/if}

		<!-- Page size selector -->
		<div class="ml-auto flex items-center gap-2">
			<span class="text-xs text-[#3d3b3e]">Filas:</span>
			<select
				bind:value={pageSize}
				class="rounded-lg border border-[#1f1f24] bg-[#19191d] px-2 py-1 font-mono text-xs text-[#a09da1] focus:outline-none cursor-pointer"
			>
				{#each [10, 25, 50] as size}
					<option value={size} class="bg-[#19191d]">{size}</option>
				{/each}
			</select>
		</div>
	</div>

	<!-- Tabla -->
	<div class="flex-1 overflow-auto">
		{#if tasks.isLoading}
			<div class="space-y-px p-4">
				{#each { length: 10 } as _}
					<Skeleton class="h-12 w-full rounded bg-[#19191d]" />
				{/each}
			</div>
		{:else if (tasksData?.length ?? 0) === 0}
			<div class="flex flex-col items-center justify-center py-24 text-center">
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
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-[#3d3b3e]">Estado</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-[#3d3b3e]">Task ID</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-[#3d3b3e]">Definición</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-[#3d3b3e]">Runtime</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-[#3d3b3e]">Duración</th>
						<th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-[#3d3b3e]">Iniciada</th>
						<th class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-[#3d3b3e]">Acciones</th>
					</tr>
				</thead>
				<tbody>
					{#each paginated as task}
						<tr
							class="border-b border-[#1f1f24] last:border-0 transition-colors hover:bg-[#19191d] cursor-pointer"
							onclick={() => window.location.href = `/tasks/${task.id}`}
						>
							<td class="px-6 py-3.5"><TaskStatusBadge status={task.status} /></td>
							<td class="px-6 py-3.5">
								<span class="font-mono text-xs text-[#75757c]">#{task.id.slice(0, 7)}</span>
							</td>
							<td class="px-6 py-3.5">
								{#if task.definition}
									<span class="text-sm text-[#e7e4ec]">{task.definition}</span>
								{:else}
									<span class="text-sm italic text-[#75757c]">ad-hoc</span>
								{/if}
							</td>
							<td class="px-6 py-3.5"><RuntimeBadge runtime={task.runtime} /></td>
							<td class="px-6 py-3.5">
								<span class="font-mono text-xs text-[#a09da1]">{duration(task.started_at, task.finished_at)}</span>
							</td>
							<td class="px-6 py-3.5">
								<span class="text-xs text-[#75757c]">{relativeTime(task.started_at)}</span>
							</td>
							<td class="px-6 py-3.5">
								<div class="flex items-center justify-end gap-1">
									{#if task.status === 'PENDING' || task.status === 'QUEUED'}
										<button
											onclick={(e) => cancelTask(task.id, e)}
											title="Cancelar"
											class="rounded-lg p-1.5 text-[#75757c] transition-colors hover:bg-red-500/10 hover:text-red-400"
										>
											<Ban size={13} />
										</button>
									{:else}
										<button
											onclick={(e) => { e.stopPropagation(); window.location.href = `/tasks/${task.id}`; }}
											title="Ver detalle"
											class="rounded-lg p-1.5 text-[#3d3b3e] transition-colors hover:bg-[#2b2c32] hover:text-[#75757c]"
										>
											<MoreVertical size={13} />
										</button>
									{/if}
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>

	<!-- Paginación -->
	{#if (tasksData?.length ?? 0) > 0}
		<div class="flex items-center justify-between border-t border-[#1f1f24] px-6 py-3">
			<p class="font-mono text-xs text-[#3d3b3e]">
				Mostrando <span class="text-[#75757c]">{start}–{end}</span>
				de <span class="text-[#75757c]">{tasksData?.length}</span> tareas
			</p>
			<div class="flex items-center gap-1">
				<button
					onclick={() => currentPage--}
					disabled={currentPage === 1}
					class="flex h-7 w-7 items-center justify-center rounded-lg border border-[#1f1f24] text-[#75757c] transition-colors hover:bg-[#19191d] hover:text-[#e7e4ec] disabled:cursor-not-allowed disabled:opacity-30"
				>
					<ChevronLeft size={13} />
				</button>
				{#each visiblePages() as p}
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
