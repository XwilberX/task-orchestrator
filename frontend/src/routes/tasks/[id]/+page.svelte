<script lang="ts">
	import { page } from '$app/state';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { tasksService, type Task } from '$lib/services/tasks';
	import TaskStatusBadge from '$lib/components/TaskStatusBadge.svelte';
	import RuntimeBadge from '$lib/components/RuntimeBadge.svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { onMount, onDestroy } from 'svelte';
	import { format, formatDistanceToNow } from 'date-fns';
	import { es } from 'date-fns/locale';
	import { ArrowLeft, Ban, Copy, RefreshCw, ClipboardCopy, Maximize2, Minimize2 } from '@lucide/svelte';
	import { PUBLIC_API_URL, PUBLIC_API_KEY } from '$env/static/public';

	const taskId = page.params.id ?? '';
	const queryClient = useQueryClient();

	// ─── Query de la tarea ─────────────────────────────────────────────────────
	const taskQuery = createQuery(() => ({
		queryKey: ['task', taskId],
		queryFn: () => tasksService.get(taskId),
		refetchInterval: (q) => {
			const status = (q.state.data as Task | undefined)?.status;
			if (!status || status === 'RUNNING' || status === 'PENDING' || status === 'QUEUED') return 3_000;
			return false;
		}
	}));

	const task = $derived(taskQuery.data as Task | undefined);
	const isTerminal = $derived(
		!!task && ['SUCCESS','FAILED','TIMEOUT','CANCELLED'].includes(task.status)
	);

	// ─── Logs ─────────────────────────────────────────────────────────────────
	interface LogLine {
		msg: string;
		stream: 'stdout' | 'stderr' | 'live';
		time: string; // ISO
	}

	let logs = $state<LogLine[]>([]);
	let logsLoading = $state(false);
	let logContainer = $state<HTMLDivElement | undefined>(undefined);
	let logContainerModal = $state<HTMLDivElement | undefined>(undefined);
	let autoScroll = $state(true);
	let logsExpanded = $state(false);

	let eventSource: EventSource | null = null;

	function scrollToBottom() {
		const el = logsExpanded ? logContainerModal : logContainer;
		if (autoScroll && el) el.scrollTop = el.scrollHeight;
	}

	// Hora local del cliente con milisegundos, ej: 13:42:07.391
	function fmtTime(isoTime: string): string {
		return new Date(isoTime).toLocaleTimeString(undefined, {
			hour12: false,
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit',
			fractionalSecondDigits: 3
		});
	}

	function copyLogs() {
		const text = logs.map(l => `[${l.stream}] ${l.msg}`).join('\n');
		navigator.clipboard.writeText(text);
	}

	$effect(() => {
		if (!task) return;

		if (task.status === 'RUNNING') {
			// Conectar SSE para logs en vivo
			eventSource?.close();
			logs = [];
			eventSource = new EventSource(
				`${PUBLIC_API_URL}/api/v1/tasks/${taskId}/stream?api_key=${PUBLIC_API_KEY}`
			);
			eventSource.onmessage = (e) => {
				if (e.data && e.data !== '{}') {
					logs.push({ msg: e.data, stream: 'live', time: new Date().toISOString() });
					setTimeout(scrollToBottom, 10);
				}
			};
			eventSource.addEventListener('done', () => {
				eventSource?.close();
				eventSource = null;
				queryClient.invalidateQueries({ queryKey: ['task', taskId] });
			});
		} else if (isTerminal && logs.length === 0) {
			// Cargar logs históricos
			logsLoading = true;
			tasksService.logs(taskId).then((entries) => {
				logs = entries.map((e) => ({
					msg: e._msg,
					stream: (e.stream === 'stderr' ? 'stderr' : 'stdout') as LogLine['stream'],
					time: e._time
				}));
				logsLoading = false;
				setTimeout(scrollToBottom, 10);
			}).catch(() => { logsLoading = false; });
		}
	});

	onDestroy(() => {
		eventSource?.close();
	});

	// ─── Cancelar ─────────────────────────────────────────────────────────────
	async function cancelTask() {
		if (!confirm('¿Cancelar esta tarea?')) return;
		await tasksService.cancel(taskId);
		queryClient.invalidateQueries({ queryKey: ['task', taskId] });
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

	function fmt(date?: string): string {
		if (!date) return '—';
		return format(new Date(date), "d MMM yyyy, HH:mm:ss", { locale: es });
	}

	function copyId() {
		navigator.clipboard.writeText(taskId);
	}

	const statusColor: Record<string, string> = {
		PENDING: 'text-zinc-400', QUEUED: 'text-amber-400',
		RUNNING: 'text-violet-300', SUCCESS: 'text-green-400',
		FAILED: 'text-red-400', TIMEOUT: 'text-orange-400', CANCELLED: 'text-zinc-500'
	};
</script>

<div class="flex h-full flex-col">
	<!-- Header -->
	<div class="flex items-center justify-between border-b border-[#1f1f24] px-6 py-4">
		<div class="flex items-center gap-3">
			<a href="/tasks" class="rounded p-1 text-[#75757c] hover:bg-[#19191d] hover:text-[#e7e4ec] transition-colors">
				<ArrowLeft size={16} />
			</a>
			<div>
				<div class="flex items-center gap-2">
					<h1 class="font-mono text-sm font-medium text-[#e7e4ec]">#{taskId.slice(0, 8)}</h1>
					<button onclick={copyId} class="text-[#75757c] hover:text-[#a09da1] transition-colors">
						<Copy size={12} />
					</button>
					{#if task}
						<TaskStatusBadge status={task.status} />
					{/if}
				</div>
				{#if task?.definition}
					<p class="text-xs text-[#75757c]">{task.definition}</p>
				{/if}
			</div>
		</div>
		<div class="flex items-center gap-2">
			{#if task?.status === 'PENDING' || task?.status === 'QUEUED'}
				<button
					onclick={cancelTask}
					class="flex items-center gap-1.5 rounded-md border border-red-500/30 px-3 py-1.5 text-xs text-red-400 hover:bg-red-500/10 transition-colors"
				>
					<Ban size={12} />
					Cancelar
				</button>
			{/if}
		</div>
	</div>

	{#if taskQuery.isLoading}
		<div class="p-6 space-y-4">
			<Skeleton class="h-6 w-48 bg-[#19191d]" />
			<Skeleton class="h-64 w-full bg-[#19191d]" />
		</div>
	{:else if task}
		<div class="flex-1 overflow-auto">
			<!-- Metadata bar -->
			<div class="grid grid-cols-2 gap-3 border-b border-[#1f1f24] px-6 py-4 sm:grid-cols-4 lg:grid-cols-6">
				<div>
					<p class="text-xs text-[#75757c]">Runtime</p>
					<div class="mt-1"><RuntimeBadge runtime={task.runtime} /></div>
				</div>
				<div>
					<p class="text-xs text-[#75757c]">Intento</p>
					<p class="mt-1 font-mono text-sm text-[#e7e4ec]">{task.attempt} / {task.max_retries}</p>
				</div>
				<div>
					<p class="text-xs text-[#75757c]">Duración</p>
					<p class="mt-1 font-mono text-sm {task.status === 'RUNNING' ? 'text-violet-300' : 'text-[#e7e4ec]'}">
						{duration(task.started_at, task.finished_at)}
					</p>
				</div>
				<div>
					<p class="text-xs text-[#75757c]">Memoria</p>
					<p class="mt-1 font-mono text-sm text-[#e7e4ec]">{task.memory_mb} MB</p>
				</div>
				<div>
					<p class="text-xs text-[#75757c]">Iniciada</p>
					<p class="mt-1 text-xs text-[#a09da1]">{fmt(task.started_at)}</p>
				</div>
				<div>
					<p class="text-xs text-[#75757c]">Finalizada</p>
					<p class="mt-1 text-xs text-[#a09da1]">{fmt(task.finished_at)}</p>
				</div>
			</div>

			<div class="grid grid-cols-1 gap-4 p-6 lg:grid-cols-5">
				<!-- Logs (60%) -->
				<div class="lg:col-span-3 flex flex-col gap-2">
					{@render logHeader()}
					<div
						bind:this={logContainer}
						class="h-96 overflow-auto rounded-lg border border-[#1f1f24] bg-[#080809] p-3"
						onscroll={() => {
							if (!logContainer) return;
							const atBottom = logContainer.scrollHeight - logContainer.scrollTop - logContainer.clientHeight < 50;
							autoScroll = atBottom;
						}}
					>
						{@render logLines()}
					</div>
				</div>

				<!-- Sidebar derecho (40%) -->
				<div class="lg:col-span-2 flex flex-col gap-4">
					<!-- Input JSON -->
					{#if task.input && Object.keys(task.input).length > 0}
						<div>
							<h2 class="mb-2 text-xs font-medium uppercase tracking-wider text-[#75757c]">Input</h2>
							<pre class="overflow-auto rounded-lg border border-[#1f1f24] bg-[#080809] p-3 font-mono text-xs text-[#a09da1]">{JSON.stringify(task.input, null, 2)}</pre>
						</div>
					{/if}

					<!-- Historial de reintentos -->
					<div>
						<h2 class="mb-2 text-xs font-medium uppercase tracking-wider text-[#75757c]">Reintentos</h2>
						<div class="rounded-lg border border-[#1f1f24] bg-[#19191d] overflow-hidden">
							<table class="w-full text-xs">
								<thead>
									<tr class="border-b border-[#1f1f24]">
										<th class="px-3 py-2 text-left font-medium text-[#75757c]">#</th>
										<th class="px-3 py-2 text-left font-medium text-[#75757c]">Estado</th>
										<th class="px-3 py-2 text-left font-medium text-[#75757c]">Iniciado</th>
										<th class="px-3 py-2 text-left font-medium text-[#75757c]">Exit</th>
									</tr>
								</thead>
								<tbody>
									{#each Array.from({length: task.attempt}, (_, i) => i + 1) as attempt}
										<tr class="border-b border-[#1f1f24] last:border-0">
											<td class="px-3 py-2 font-mono text-[#75757c]">{attempt}</td>
											<td class="px-3 py-2">
												{#if attempt === task.attempt}
													<TaskStatusBadge status={task.status} />
												{:else}
													<span class="text-[#75757c]">—</span>
												{/if}
											</td>
											<td class="px-3 py-2 text-[#75757c]">
												{attempt === task.attempt ? fmt(task.started_at) : '—'}
											</td>
											<td class="px-3 py-2 font-mono text-[#75757c]">
												{attempt === task.attempt && task.exit_code != null ? task.exit_code : '—'}
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</div>
				</div>
			</div>
		</div>
	{:else}
		<div class="flex flex-col items-center justify-center py-24 text-center">
			<p class="text-sm text-[#75757c]">Tarea no encontrada</p>
			<a href="/tasks" class="mt-2 text-xs text-[#d0bcff] hover:underline">← Volver a tareas</a>
		</div>
	{/if}
</div>

<!-- Modal de logs expandido -->
{#if logsExpanded && task}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-md p-6"
		onclick={(e) => { if (e.target === e.currentTarget) logsExpanded = false; }}
	>
		<div class="flex flex-col w-full max-w-6xl h-[90vh] rounded-xl border border-[#2a2730] bg-[#0f0e12]/95 shadow-2xl overflow-hidden">
			<!-- Header modal -->
			<div class="flex items-center justify-between px-5 py-3 border-b border-[#1f1f24] shrink-0">
				{@render logHeader(true)}
			</div>
			<!-- Cuerpo -->
			<div
				bind:this={logContainerModal}
				class="flex-1 overflow-auto p-4"
				onscroll={() => {
					if (!logContainerModal) return;
					const atBottom = logContainerModal.scrollHeight - logContainerModal.scrollTop - logContainerModal.clientHeight < 50;
					autoScroll = atBottom;
				}}
			>
				{@render logLines()}
			</div>
		</div>
	</div>
{/if}

{#snippet logHeader(insideModal = false)}
	<div class="flex w-full items-center justify-between">
		<h2 class="text-xs font-medium uppercase tracking-wider text-[#75757c]">
			{task?.status === 'RUNNING' ? 'Logs en vivo' : 'Logs'}
			{#if logs.length > 0}
				<span class="ml-2 text-[#3d3b3e] normal-case font-normal tracking-normal">{logs.length} líneas</span>
			{/if}
		</h2>
		<div class="flex items-center gap-3">
			{#if task?.status === 'RUNNING'}
				<span class="flex items-center gap-1 text-xs text-violet-300">
					<span class="h-1.5 w-1.5 animate-pulse rounded-full bg-violet-400"></span>
					LIVE
				</span>
			{/if}
			{#if logs.length > 0}
				<button onclick={copyLogs} class="flex items-center gap-1 text-xs text-[#75757c] hover:text-[#a09da1] transition-colors">
					<ClipboardCopy size={11} />
					Copiar
				</button>
			{/if}
			<label class="flex items-center gap-1 text-xs text-[#75757c] cursor-pointer">
				<input type="checkbox" bind:checked={autoScroll} class="h-3 w-3" />
				Auto-scroll
			</label>
			<button
				onclick={() => { logsExpanded = !logsExpanded; setTimeout(scrollToBottom, 50); }}
				class="text-[#75757c] hover:text-[#a09da1] transition-colors"
				title={logsExpanded ? 'Cerrar' : 'Expandir'}
			>
				{#if insideModal}
					<Minimize2 size={13} />
				{:else}
					<Maximize2 size={13} />
				{/if}
			</button>
		</div>
	</div>
{/snippet}

{#snippet logLines()}
	{#if logsLoading}
		<div class="flex items-center gap-2 text-[#75757c]">
			<RefreshCw size={12} class="animate-spin" />
			<span class="font-mono text-xs">Cargando logs...</span>
		</div>
	{:else if logs.length === 0}
		<p class="font-mono text-xs text-[#3d3b3e]">
			{task?.status === 'PENDING' || task?.status === 'QUEUED'
				? 'Esperando inicio...'
				: 'Sin output registrado'}
		</p>
	{:else}
		{#each logs as line, i}
			<div class="group flex gap-2 rounded px-1 py-[1px] hover:bg-[#0f0f12]">
				<span class="w-7 shrink-0 select-none text-right font-mono text-[10px] text-[#2e2d30] group-hover:text-[#3d3b3e] leading-relaxed pt-px">{i + 1}</span>
				<span class="shrink-0 font-mono text-[10px] text-[#3d3b3e] group-hover:text-[#504e52] leading-relaxed pt-px w-24 tabular-nums">
					{fmtTime(line.time)}
				</span>
				{#if line.stream === 'stderr'}
					<span class="shrink-0 self-start mt-[3px] rounded px-1 py-px font-mono text-[9px] uppercase leading-none bg-amber-950/60 text-amber-400/80">err</span>
				{:else if line.stream === 'stdout'}
					<span class="shrink-0 self-start mt-[3px] rounded px-1 py-px font-mono text-[9px] uppercase leading-none bg-transparent text-[#2e2d30] group-hover:text-[#3d3b3e]">out</span>
				{:else}
					<span class="shrink-0 w-6"></span>
				{/if}
				<span class="font-mono text-xs leading-relaxed break-all {line.stream === 'stderr' ? 'text-amber-300/70' : 'text-[#e7e4ec]'}">{line.msg}</span>
			</div>
		{/each}
	{/if}
{/snippet}
