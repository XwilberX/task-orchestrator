<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { schedulesService, type Schedule } from '$lib/services/schedules';
	import { definitionsService, type Definition } from '$lib/services/definitions';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { format, formatDistanceToNow } from 'date-fns';
	import { es } from 'date-fns/locale';
	import { Plus, Pause, Play, Trash2, CalendarClock, Activity, Clock, X } from '@lucide/svelte';

	const queryClient = useQueryClient();

	const schedules = createQuery(() => ({
		queryKey: ['schedules'],
		queryFn: schedulesService.list,
		refetchInterval: 30_000
	}));

	const defs = createQuery(() => ({
		queryKey: ['definitions'],
		queryFn: definitionsService.list
	}));

	const schedulesData = $derived(schedules.data as Schedule[] | undefined);
	const defsData = $derived(defs.data as Definition[] | undefined);

	const activeCount = $derived(schedulesData?.filter(s => s.status === 'active').length ?? 0);
	const nextSchedule = $derived(
		schedulesData
			?.filter(s => s.status === 'active' && s.next_run_at)
			.sort((a, b) => new Date(a.next_run_at!).getTime() - new Date(b.next_run_at!).getTime())[0]
	);

	// ─── Formulario ────────────────────────────────────────────────────────────
	let showForm = $state(false);
	let newDefId = $state('');
	let newCron = $state('');
	let formError = $state('');
	let formLoading = $state(false);

	async function createSchedule(e: SubmitEvent) {
		e.preventDefault();
		formError = '';
		if (!newDefId) { formError = 'Selecciona una definición'; return; }
		if (!newCron.trim()) { formError = 'Expresión cron requerida'; return; }
		formLoading = true;
		try {
			await schedulesService.create({ definition_id: newDefId, cron: newCron.trim() });
			queryClient.invalidateQueries({ queryKey: ['schedules'] });
			newDefId = '';
			newCron = '';
			showForm = false;
		} catch (err: unknown) {
			formError = err instanceof Error ? err.message : 'Error al crear el schedule';
		} finally {
			formLoading = false;
		}
	}

	async function toggleSchedule(id: string) {
		await schedulesService.toggle(id);
		queryClient.invalidateQueries({ queryKey: ['schedules'] });
	}

	async function deleteSchedule(id: string) {
		if (!confirm('¿Eliminar este schedule?')) return;
		await schedulesService.delete(id);
		queryClient.invalidateQueries({ queryKey: ['schedules'] });
	}

	function fmt(date?: string): string {
		if (!date) return '—';
		return format(new Date(date), "d MMM, HH:mm", { locale: es });
	}

	function relativeTime(date?: string): string {
		if (!date) return '—';
		return formatDistanceToNow(new Date(date), { addSuffix: true, locale: es });
	}

	function defName(id: string): string {
		return defsData?.find(d => d.id === id)?.name ?? id.slice(0, 8);
	}

	const CRON_PRESETS = [
		{ label: 'Cada minuto',   value: '* * * * *' },
		{ label: 'Cada 5 min',    value: '*/5 * * * *' },
		{ label: 'Cada hora',     value: '0 * * * *' },
		{ label: 'Cada día 9am',  value: '0 9 * * *' },
		{ label: 'Lunes 9am',     value: '0 9 * * 1' },
		{ label: 'Cada domingo',  value: '0 0 * * 0' },
	];
</script>

<div class="flex h-full flex-col">

	<!-- Header -->
	<div class="border-b border-[#1f1f24] px-6 py-5">
		<div class="flex items-start justify-between">
			<div>
				<h1 class="text-xl font-semibold text-[#e7e4ec]">Schedules</h1>
				<p class="mt-0.5 text-sm text-[#75757c]">Automatiza tus tareas con expresiones cron</p>
			</div>
			<button
				onclick={() => showForm = !showForm}
				class="flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-violet-500"
			>
				<Plus size={15} />
				Nuevo schedule
			</button>
		</div>

		<!-- Stats -->
		<div class="mt-5 grid grid-cols-3 gap-4">
			<div class="rounded-xl border border-[#1f1f24] bg-[#19191d] px-5 py-4">
				<p class="text-xs text-[#75757c]">Activos</p>
				<p class="mt-1 font-mono text-3xl font-semibold text-[#e7e4ec]">{activeCount}</p>
				<p class="mt-0.5 text-xs text-[#3d3b3e]">de {schedulesData?.length ?? 0} en total</p>
			</div>
			<div class="rounded-xl border border-[#1f1f24] bg-[#19191d] px-5 py-4">
				<p class="text-xs text-[#75757c]">Próxima ejecución</p>
				{#if nextSchedule?.next_run_at}
					<p class="mt-1 font-mono text-xl font-semibold text-[#d0bcff]">
						{format(new Date(nextSchedule.next_run_at), 'HH:mm')}
					</p>
					<p class="mt-0.5 text-xs text-[#75757c]">{defName(nextSchedule.definition_id)}</p>
				{:else}
					<p class="mt-1 font-mono text-xl font-semibold text-[#3d3b3e]">—</p>
					<p class="mt-0.5 text-xs text-[#3d3b3e]">sin schedules activos</p>
				{/if}
			</div>
			<div class="rounded-xl border border-[#1f1f24] bg-[#19191d] px-5 py-4">
				<p class="text-xs text-[#75757c]">Pausados</p>
				<p class="mt-1 font-mono text-3xl font-semibold text-[#e7e4ec]">
					{(schedulesData?.length ?? 0) - activeCount}
				</p>
				<p class="mt-0.5 text-xs text-[#3d3b3e]">schedules en pausa</p>
			</div>
		</div>
	</div>

	<!-- Form modal inline -->
	{#if showForm}
		<div class="border-b border-[#1f1f24] bg-[#0f0f12] px-6 py-5">
			<div class="flex items-center justify-between mb-4">
				<h2 class="text-sm font-medium text-[#e7e4ec]">Nuevo schedule</h2>
				<button onclick={() => showForm = false} class="text-[#75757c] hover:text-[#a09da1] transition-colors">
					<X size={16} />
				</button>
			</div>
			<form onsubmit={createSchedule} class="flex flex-wrap items-end gap-3">
				<div class="min-w-52 flex-1">
					<label class="mb-1.5 block text-xs text-[#75757c]">Definición</label>
					<select
						bind:value={newDefId}
						class="w-full rounded-lg border border-[#2b2c32] bg-[#19191d] px-3 py-2 text-sm text-[#e7e4ec] focus:border-violet-500/50 focus:outline-none"
					>
						<option value="">Seleccionar...</option>
						{#each defsData ?? [] as def}
							<option value={def.id}>{def.name}</option>
						{/each}
					</select>
				</div>
				<div class="min-w-52 flex-1">
					<label class="mb-1.5 block text-xs text-[#75757c]">Expresión cron</label>
					<input
						bind:value={newCron}
						placeholder="0 9 * * *"
						class="w-full rounded-lg border border-[#2b2c32] bg-[#19191d] px-3 py-2 font-mono text-sm text-[#e7e4ec] placeholder-[#3d3b3e] focus:border-violet-500/50 focus:outline-none"
					/>
				</div>
				<button
					type="submit"
					disabled={formLoading}
					class="rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-500 disabled:opacity-50 transition-colors"
				>
					{formLoading ? 'Creando...' : 'Crear'}
				</button>
			</form>
			{#if formError}
				<p class="mt-2 text-xs text-red-400">{formError}</p>
			{/if}
			<!-- Presets -->
			<div class="mt-3 flex flex-wrap items-center gap-2">
				<span class="text-xs text-[#3d3b3e]">Atajos:</span>
				{#each CRON_PRESETS as p}
					<button
						type="button"
						onclick={() => newCron = p.value}
						class="rounded-md border border-[#1f1f24] bg-[#19191d] px-2 py-0.5 font-mono text-xs text-[#75757c] hover:border-violet-500/30 hover:text-violet-300 transition-colors"
					>
						{p.value}
						<span class="font-sans text-[#3d3b3e] ml-1">· {p.label}</span>
					</button>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Tabla -->
	<div class="flex-1 overflow-auto">
		{#if schedules.isLoading}
			<div class="space-y-px p-6">
				{#each { length: 4 } as _}
					<Skeleton class="h-16 w-full rounded-xl bg-[#19191d]" />
				{/each}
			</div>
		{:else if (schedulesData?.length ?? 0) === 0}
			<div class="flex flex-col items-center justify-center py-24 text-center">
				<div class="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl border border-[#1f1f24] bg-[#19191d]">
					<CalendarClock size={24} class="text-[#3d3b3e]" />
				</div>
				<p class="text-sm font-medium text-[#75757c]">No hay schedules configurados</p>
				<p class="mt-1 text-xs text-[#3d3b3e]">Crea uno con el botón de arriba</p>
			</div>
		{:else}
			<div class="p-6 space-y-3">
				{#each schedulesData ?? [] as sched}
					<div class="flex items-center gap-4 rounded-xl border border-[#1f1f24] bg-[#19191d] px-5 py-4 transition-colors hover:border-[#2b2c32]">

						<!-- Estado indicator -->
						<div class="shrink-0">
							{#if sched.status === 'active'}
								<span class="flex h-2 w-2 rounded-full bg-green-400 ring-4 ring-green-400/20"></span>
							{:else}
								<span class="flex h-2 w-2 rounded-full bg-zinc-600"></span>
							{/if}
						</div>

						<!-- Nombre definición -->
						<div class="min-w-0 flex-1">
							<p class="truncate text-sm font-medium text-[#e7e4ec]">{defName(sched.definition_id)}</p>
							<p class="text-xs text-[#3d3b3e]">#{sched.id.slice(0, 8)}</p>
						</div>

						<!-- Cron -->
						<div class="shrink-0 hidden sm:block">
							<code class="rounded-md bg-violet-500/10 px-2.5 py-1 font-mono text-xs text-violet-300">
								{sched.cron}
							</code>
						</div>

						<!-- Status badge -->
						<div class="shrink-0 hidden md:block">
							{#if sched.status === 'active'}
								<span class="rounded-full bg-green-500/10 px-2.5 py-0.5 text-xs font-medium text-green-400">Activo</span>
							{:else}
								<span class="rounded-full bg-zinc-500/10 px-2.5 py-0.5 text-xs font-medium text-zinc-500">Pausado</span>
							{/if}
						</div>

						<!-- Último run -->
						<div class="shrink-0 hidden lg:block text-right min-w-28">
							<p class="text-xs text-[#3d3b3e]">Último</p>
							<p class="text-xs text-[#75757c]">{relativeTime(sched.last_run_at)}</p>
						</div>

						<!-- Próximo run -->
						<div class="shrink-0 text-right min-w-28">
							<p class="text-xs text-[#3d3b3e]">Próximo</p>
							<p class="font-mono text-xs text-[#a09da1]">{fmt(sched.next_run_at)}</p>
						</div>

						<!-- Acciones -->
						<div class="flex shrink-0 items-center gap-1">
							<button
								onclick={() => toggleSchedule(sched.id)}
								title={sched.status === 'active' ? 'Pausar' : 'Reanudar'}
								class="rounded-lg p-2 text-[#75757c] transition-colors
									{sched.status === 'active'
										? 'hover:bg-amber-500/10 hover:text-amber-400'
										: 'hover:bg-green-500/10 hover:text-green-400'}"
							>
								{#if sched.status === 'active'}
									<Pause size={14} />
								{:else}
									<Play size={14} />
								{/if}
							</button>
							<button
								onclick={() => deleteSchedule(sched.id)}
								title="Eliminar"
								class="rounded-lg p-2 text-[#75757c] transition-colors hover:bg-red-500/10 hover:text-red-400"
							>
								<Trash2 size={14} />
							</button>
						</div>
					</div>
				{/each}
			</div>
		{/if}

		<!-- Info cards -->
		{#if !schedules.isLoading}
			<div class="grid grid-cols-1 gap-4 px-6 pb-6 sm:grid-cols-2">
				<div class="rounded-xl border border-[#1f1f24] bg-[#19191d] p-5">
					<div class="mb-3 flex items-center gap-2">
						<Clock size={14} class="text-violet-400" />
						<h3 class="text-xs font-medium text-[#a09da1] uppercase tracking-wider">Cron helper</h3>
					</div>
					<div class="space-y-1.5">
						{#each CRON_PRESETS as p}
							<div class="flex items-center justify-between">
								<span class="text-xs text-[#75757c]">{p.label}</span>
								<code class="font-mono text-xs text-violet-300/70">{p.value}</code>
							</div>
						{/each}
					</div>
					<p class="mt-3 text-xs text-[#3d3b3e]">Formato: minuto hora día mes día-semana</p>
				</div>
				<div class="rounded-xl border border-[#1f1f24] bg-[#19191d] p-5">
					<div class="mb-3 flex items-center gap-2">
						<Activity size={14} class="text-violet-400" />
						<h3 class="text-xs font-medium text-[#a09da1] uppercase tracking-wider">Estado del sistema</h3>
					</div>
					<div class="space-y-3">
						<div class="flex items-center justify-between">
							<span class="text-xs text-[#75757c]">Schedules activos</span>
							<span class="font-mono text-xs text-green-400">{activeCount}</span>
						</div>
						<div class="flex items-center justify-between">
							<span class="text-xs text-[#75757c]">Total schedules</span>
							<span class="font-mono text-xs text-[#a09da1]">{schedulesData?.length ?? 0}</span>
						</div>
						<div class="flex items-center justify-between">
							<span class="text-xs text-[#75757c]">Próxima ejecución</span>
							<span class="font-mono text-xs text-[#d0bcff]">
								{nextSchedule?.next_run_at ? fmt(nextSchedule.next_run_at) : '—'}
							</span>
						</div>
					</div>
				</div>
			</div>
		{/if}
	</div>
</div>
