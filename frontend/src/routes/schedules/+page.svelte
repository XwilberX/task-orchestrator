<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { schedulesService, type Schedule } from '$lib/services/schedules';
	import { definitionsService, type Definition } from '$lib/services/definitions';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { format, formatDistanceToNow } from 'date-fns';
	import { es } from 'date-fns/locale';
	import { Plus, Pause, Play, Trash2, CalendarClock } from '@lucide/svelte';

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

	// ─── Formulario inline ─────────────────────────────────────────────────────
	let newDefId = $state('');
	let newCron = $state('');
	let formError = $state('');
	let formLoading = $state(false);

	async function createSchedule(e: SubmitEvent) {
		e.preventDefault();
		formError = '';
		if (!newDefId) { formError = 'Selecciona una definición'; return; }
		if (!newCron.trim()) { formError = 'La expresión cron es requerida'; return; }
		formLoading = true;
		try {
			await schedulesService.create({ definition_id: newDefId, cron: newCron.trim() });
			queryClient.invalidateQueries({ queryKey: ['schedules'] });
			newDefId = '';
			newCron = '';
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

	const CRON_EXAMPLES = [
		{ label: 'Cada minuto', value: '* * * * *' },
		{ label: 'Cada hora', value: '0 * * * *' },
		{ label: 'Cada día 9am', value: '0 9 * * *' },
		{ label: 'Lunes 9am', value: '0 9 * * 1' },
	];
</script>

<div class="flex h-full flex-col">
	<!-- Header -->
	<div class="flex items-center justify-between border-b border-[#1f1f24] px-6 py-4">
		<div>
			<h1 class="text-lg font-semibold text-[#e7e4ec]">Schedules</h1>
			{#if schedulesData}
				<p class="font-mono text-xs text-[#75757c]">{schedulesData.length} programados</p>
			{/if}
		</div>
	</div>

	<div class="flex-1 overflow-auto">
		<!-- Tabla de schedules -->
		{#if schedules.isLoading}
			<div class="space-y-px p-4">
				{#each { length: 5 } as _}
					<Skeleton class="h-14 w-full rounded bg-[#19191d]" />
				{/each}
			</div>
		{:else if (schedulesData?.length ?? 0) === 0}
			<div class="flex flex-col items-center justify-center py-16 text-center">
				<CalendarClock size={32} class="mb-3 text-[#3d3b3e]" />
				<p class="text-sm text-[#75757c]">No hay schedules aún</p>
				<p class="mt-1 text-xs text-[#3d3b3e]">Crea uno con el formulario de abajo</p>
			</div>
		{:else}
			<table class="w-full text-sm">
				<thead class="sticky top-0 bg-[#0e0e10]">
					<tr class="border-b border-[#1f1f24]">
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Definición</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Cron</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Estado</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Último run</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Próximo run</th>
						<th class="px-4 py-2.5 text-left text-xs font-medium text-[#75757c]">Acciones</th>
					</tr>
				</thead>
				<tbody>
					{#each schedulesData ?? [] as sched}
						<tr class="border-b border-[#1f1f24] last:border-0 hover:bg-[#19191d] transition-colors">
							<td class="px-4 py-3 font-medium text-[#e7e4ec]">{defName(sched.definition_id)}</td>
							<td class="px-4 py-3">
								<code class="font-mono text-xs text-violet-300 bg-violet-500/10 px-2 py-0.5 rounded">
									{sched.cron}
								</code>
							</td>
							<td class="px-4 py-3">
								{#if sched.status === 'active'}
									<span class="inline-flex items-center gap-1.5 rounded-full border border-green-500/20 bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-400">
										<span class="h-1.5 w-1.5 rounded-full bg-green-400"></span>
										Activo
									</span>
								{:else}
									<span class="inline-flex items-center gap-1.5 rounded-full border border-zinc-600/20 bg-zinc-500/10 px-2 py-0.5 text-xs font-medium text-zinc-500">
										<span class="h-1.5 w-1.5 rounded-full bg-zinc-500"></span>
										Pausado
									</span>
								{/if}
							</td>
							<td class="px-4 py-3 text-xs text-[#75757c]">{relativeTime(sched.last_run_at)}</td>
							<td class="px-4 py-3 text-xs text-[#a09da1]">{fmt(sched.next_run_at)}</td>
							<td class="px-4 py-3">
								<div class="flex items-center gap-1">
									<button
										onclick={() => toggleSchedule(sched.id)}
										class="rounded p-1.5 text-[#75757c] transition-colors
											{sched.status === 'active'
												? 'hover:bg-amber-500/10 hover:text-amber-400'
												: 'hover:bg-green-500/10 hover:text-green-400'}"
										title={sched.status === 'active' ? 'Pausar' : 'Reanudar'}
									>
										{#if sched.status === 'active'}
											<Pause size={14} />
										{:else}
											<Play size={14} />
										{/if}
									</button>
									<button
										onclick={() => deleteSchedule(sched.id)}
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

		<!-- Formulario inline: agregar schedule -->
		<div class="border-t border-[#1f1f24] bg-[#19191d] px-6 py-5">
			<div class="flex items-center gap-2 mb-4">
				<Plus size={14} class="text-[#75757c]" />
				<h2 class="text-xs font-medium uppercase tracking-wider text-[#75757c]">Agregar schedule</h2>
			</div>

			<form onsubmit={createSchedule} class="flex flex-wrap items-end gap-3">
				<!-- Seleccionar definición -->
				<div class="flex-1 min-w-48">
					<label class="mb-1.5 block text-xs text-[#75757c]">Definición</label>
					<select
						bind:value={newDefId}
						class="w-full rounded-md border border-[#2b2c32] bg-[#0e0e10] px-3 py-2 text-sm text-[#e7e4ec] focus:border-violet-500/50 focus:outline-none"
					>
						<option value="">Seleccionar definición...</option>
						{#each defsData ?? [] as def}
							<option value={def.id}>{def.name}</option>
						{/each}
					</select>
				</div>

				<!-- Expresión cron -->
				<div class="flex-1 min-w-48">
					<label class="mb-1.5 block text-xs text-[#75757c]">Expresión cron</label>
					<input
						bind:value={newCron}
						placeholder="0 9 * * *"
						class="w-full rounded-md border border-[#2b2c32] bg-[#0e0e10] px-3 py-2 font-mono text-sm text-[#e7e4ec] placeholder-[#3d3b3e] focus:border-violet-500/50 focus:outline-none"
					/>
				</div>

				<button
					type="submit"
					disabled={formLoading}
					class="flex items-center gap-1.5 rounded-md bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-500 disabled:opacity-50 transition-colors"
				>
					<Plus size={14} />
					{formLoading ? 'Creando...' : 'Crear'}
				</button>
			</form>

			{#if formError}
				<p class="mt-2 text-xs text-red-400">{formError}</p>
			{/if}

			<!-- Ejemplos de cron -->
			<div class="mt-3 flex flex-wrap gap-2">
				<span class="text-xs text-[#3d3b3e]">Ejemplos:</span>
				{#each CRON_EXAMPLES as ex}
					<button
						type="button"
						onclick={() => newCron = ex.value}
						class="font-mono text-xs text-[#75757c] hover:text-violet-300 transition-colors"
					>
						{ex.value} <span class="font-sans text-[#3d3b3e]">({ex.label})</span>
					</button>
				{/each}
			</div>
		</div>
	</div>
</div>
