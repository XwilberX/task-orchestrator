<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { metricsService } from '$lib/services/metrics';
	import { tasksService } from '$lib/services/tasks';
	import TaskStatusBadge from '$lib/components/TaskStatusBadge.svelte';
	import RuntimeBadge from '$lib/components/RuntimeBadge.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { formatDistanceToNow, format } from 'date-fns';
	import { es } from 'date-fns/locale';
	import { Activity, CheckCircle2, XCircle, Clock, Timer, ChevronLeft, ChevronRight } from '@lucide/svelte';
	import type { MetricsSummary } from '$lib/services/metrics';
	import type { Task } from '$lib/services/tasks';

	const metrics = createQuery(() => ({
		queryKey: ['metrics'],
		queryFn: metricsService.summary,
		refetchInterval: 15_000
	}));

	const recentTasks = createQuery(() => ({
		queryKey: ['tasks', 'recent'],
		queryFn: () => tasksService.list(),
		refetchInterval: 10_000
	}));

	// Helpers tipados para evitar casteos en el template
	const metricsData = $derived(metrics.data as MetricsSummary | undefined);
	const tasksData = $derived(recentTasks.data as Task[] | undefined);

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

	const today = format(new Date(), "d MMM yyyy", { locale: es });

	// ─── Paginación ────────────────────────────────────────────────────────────
	const PAGE_SIZE = 10;
	let currentPage = $state(1);
	const totalPages = $derived(Math.max(1, Math.ceil((tasksData?.length ?? 0) / PAGE_SIZE)));
	const paginated = $derived((tasksData ?? []).slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE));
	const start = $derived((currentPage - 1) * PAGE_SIZE + 1);
	const end = $derived(Math.min(currentPage * PAGE_SIZE, tasksData?.length ?? 0));
	const visiblePages = $derived(() => {
		const total = totalPages;
		if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
		if (currentPage <= 4) return Array.from({ length: 7 }, (_, i) => i + 1);
		if (currentPage >= total - 3) return Array.from({ length: 7 }, (_, i) => total - 6 + i);
		return Array.from({ length: 7 }, (_, i) => currentPage - 3 + i);
	});
</script>

<div class="flex h-full flex-col">
	<!-- Header -->
	<div class="flex items-center justify-between border-b border-[#1f1f24] px-6 py-4">
		<div>
			<h1 class="text-lg font-semibold text-[#e7e4ec]">Dashboard</h1>
			<p class="font-mono text-xs text-[#75757c]">{today}</p>
		</div>
	</div>

	<div class="flex-1 overflow-auto px-6 py-6 space-y-6">

		<!-- Metric cards -->
		<div class="grid grid-cols-2 gap-3 sm:grid-cols-4 xl:grid-cols-5">
			<div class="rounded-lg border border-[#1f1f24] bg-[#19191d] p-4">
				<div class="flex items-center justify-between mb-3">
					<span class="text-xs font-medium text-[#75757c] uppercase tracking-wider">Hoy</span>
					<Activity size={14} class="text-[#75757c]" />
				</div>
				{#if metrics.isLoading}
					<Skeleton class="h-8 w-16 bg-[#2b2c32]" />
				{:else}
					<p class="font-mono text-3xl font-semibold text-[#e7e4ec]">{metricsData?.tasks_today ?? 0}</p>
				{/if}
				<p class="mt-1 text-xs text-[#75757c]">tareas despachadas</p>
			</div>

			<div class="rounded-lg border border-[#1f1f24] bg-[#19191d] p-4">
				<div class="flex items-center justify-between mb-3">
					<span class="text-xs font-medium text-[#75757c] uppercase tracking-wider">Fallidas</span>
					<XCircle size={14} class="text-red-400" />
				</div>
				{#if metrics.isLoading}
					<Skeleton class="h-8 w-12 bg-[#2b2c32]" />
				{:else}
					<p class="font-mono text-3xl font-semibold text-red-400">{metricsData?.tasks_failed ?? 0}</p>
				{/if}
				<p class="mt-1 text-xs text-[#75757c]">con error hoy</p>
			</div>

			<div class="rounded-lg border border-[#1f1f24] bg-[#19191d] p-4">
				<div class="flex items-center justify-between mb-3">
					<span class="text-xs font-medium text-[#75757c] uppercase tracking-wider">En cola</span>
					<Clock size={14} class="text-amber-400" />
				</div>
				{#if metrics.isLoading}
					<Skeleton class="h-8 w-10 bg-[#2b2c32]" />
				{:else}
					<p class="font-mono text-3xl font-semibold text-amber-400">{metricsData?.tasks_queued ?? 0}</p>
				{/if}
				<p class="mt-1 text-xs text-[#75757c]">esperando slot</p>
			</div>

			<div class="rounded-lg border border-[#1f1f24] bg-[#19191d] p-4">
				<div class="flex items-center justify-between mb-3">
					<span class="text-xs font-medium text-[#75757c] uppercase tracking-wider">Corriendo</span>
					<span class="h-2 w-2 animate-pulse rounded-full bg-violet-400"></span>
				</div>
				{#if metrics.isLoading}
					<Skeleton class="h-8 w-10 bg-[#2b2c32]" />
				{:else}
					<p class="font-mono text-3xl font-semibold text-violet-300">{metricsData?.tasks_running ?? 0}</p>
				{/if}
				<p class="mt-1 text-xs text-[#75757c]">activas ahora</p>
			</div>

			<div class="rounded-lg border border-[#1f1f24] bg-[#19191d] p-4">
				<div class="flex items-center justify-between mb-3">
					<span class="text-xs font-medium text-[#75757c] uppercase tracking-wider">Avg</span>
					<Timer size={14} class="text-[#75757c]" />
				</div>
				{#if metrics.isLoading}
					<Skeleton class="h-8 w-16 bg-[#2b2c32]" />
				{:else}
					<p class="font-mono text-3xl font-semibold text-[#e7e4ec]">
						{(metricsData?.avg_duration_seconds ?? 0).toFixed(1)}<span class="text-lg text-[#75757c]">s</span>
					</p>
				{/if}
				<p class="mt-1 text-xs text-[#75757c]">duración promedio</p>
			</div>
		</div>

		<!-- Recent tasks table -->
		<div class="rounded-lg border border-[#1f1f24] bg-[#19191d]">
			<div class="flex items-center justify-between border-b border-[#1f1f24] px-4 py-3">
				<div class="flex items-center gap-2">
					<h2 class="text-sm font-medium text-[#e7e4ec]">Ejecuciones recientes</h2>
					{#if tasksData}
						<span class="font-mono text-xs text-[#3d3b3e]">{tasksData.length} total</span>
					{/if}
				</div>
				<a href="/tasks" class="text-xs text-[#d0bcff] hover:underline">Ver todas →</a>
			</div>

			{#if recentTasks.isLoading}
				<div class="space-y-px p-2">
					{#each { length: 8 } as _}
						<Skeleton class="h-10 w-full rounded bg-[#2b2c32]" />
					{/each}
				</div>
			{:else if (tasksData?.length ?? 0) === 0}
				<div class="flex flex-col items-center justify-center py-16 text-center">
					<CheckCircle2 size={32} class="mb-3 text-[#3d3b3e]" />
					<p class="text-sm text-[#75757c]">No hay ejecuciones aún</p>
					<p class="mt-1 text-xs text-[#3d3b3e]">Despacha una tarea para empezar</p>
				</div>
			{:else}
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-[#1f1f24]">
							<th class="px-4 py-2 text-left text-xs font-medium text-[#75757c]">Estado</th>
							<th class="px-4 py-2 text-left text-xs font-medium text-[#75757c]">Task ID</th>
							<th class="px-4 py-2 text-left text-xs font-medium text-[#75757c]">Definición</th>
							<th class="px-4 py-2 text-left text-xs font-medium text-[#75757c]">Runtime</th>
							<th class="px-4 py-2 text-left text-xs font-medium text-[#75757c]">Duración</th>
							<th class="px-4 py-2 text-left text-xs font-medium text-[#75757c]">Iniciada</th>
						</tr>
					</thead>
					<tbody>
						{#each paginated as task}
							<tr
								class="border-b border-[#1f1f24] last:border-0 hover:bg-[#1f1f24] transition-colors cursor-pointer"
								onclick={() => { window.location.href = `/tasks/${task.id}`; }}
							>
								<td class="px-4 py-2.5"><TaskStatusBadge status={task.status} /></td>
								<td class="px-4 py-2.5">
									<span class="font-mono text-xs text-[#a09da1]">#{task.id.slice(0, 8)}</span>
								</td>
								<td class="px-4 py-2.5 text-[#e7e4ec]">
									{#if task.definition}
										{task.definition}
									{:else}
										<span class="italic text-[#75757c]">ad-hoc</span>
									{/if}
								</td>
								<td class="px-4 py-2.5"><RuntimeBadge runtime={task.runtime} /></td>
								<td class="px-4 py-2.5">
									<span class="font-mono text-xs text-[#a09da1]">{duration(task.started_at, task.finished_at)}</span>
								</td>
								<td class="px-4 py-2.5">
									<span class="text-xs text-[#75757c]">{relativeTime(task.created_at)}</span>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}

			<!-- Paginación -->
			{#if (tasksData?.length ?? 0) > PAGE_SIZE}
				<div class="flex items-center justify-between border-t border-[#1f1f24] px-4 py-3">
					<p class="font-mono text-xs text-[#3d3b3e]">
						<span class="text-[#75757c]">{start}–{end}</span> de <span class="text-[#75757c]">{tasksData?.length}</span>
					</p>
					<div class="flex items-center gap-1">
						<button
							onclick={() => currentPage--}
							disabled={currentPage === 1}
							class="flex h-6 w-6 items-center justify-center rounded border border-[#1f1f24] text-[#75757c] transition-colors hover:bg-[#2b2c32] hover:text-[#e7e4ec] disabled:cursor-not-allowed disabled:opacity-30"
						>
							<ChevronLeft size={12} />
						</button>
						{#each visiblePages() as p}
							<button
								onclick={() => currentPage = p}
								class="flex h-6 w-6 items-center justify-center rounded font-mono text-[11px] transition-colors
									{p === currentPage
										? 'bg-violet-600 text-white'
										: 'border border-[#1f1f24] text-[#75757c] hover:bg-[#2b2c32] hover:text-[#e7e4ec]'}"
							>
								{p}
							</button>
						{/each}
						<button
							onclick={() => currentPage++}
							disabled={currentPage === totalPages}
							class="flex h-6 w-6 items-center justify-center rounded border border-[#1f1f24] text-[#75757c] transition-colors hover:bg-[#2b2c32] hover:text-[#e7e4ec] disabled:cursor-not-allowed disabled:opacity-30"
						>
							<ChevronRight size={12} />
						</button>
					</div>
				</div>
			{/if}
		</div>

	</div>
</div>
