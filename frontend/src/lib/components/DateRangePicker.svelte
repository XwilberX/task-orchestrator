<script lang="ts">
	import { CalendarDate, getLocalTimeZone, today } from '@internationalized/date';
	import type { DateRange } from 'bits-ui';
	import { RangeCalendar } from '$lib/components/ui/range-calendar';
	import * as Popover from '$lib/components/ui/popover';
	import { CalendarDays, X } from '@lucide/svelte';
	import { format } from 'date-fns';
	import { es } from 'date-fns/locale';

	let {
		fromDate = $bindable(''),
		toDate = $bindable(''),
	}: {
		fromDate?: string;
		toDate?: string;
	} = $props();

	let open = $state(false);

	// Convierte string ISO a CalendarDate
	function toCalendarDate(iso: string): CalendarDate | undefined {
		if (!iso) return undefined;
		const d = new Date(iso);
		return new CalendarDate(d.getFullYear(), d.getMonth() + 1, d.getDate());
	}

	// Convierte CalendarDate a string YYYY-MM-DD
	function toISO(d: CalendarDate | undefined): string {
		if (!d) return '';
		return `${d.year}-${String(d.month).padStart(2, '0')}-${String(d.day).padStart(2, '0')}`;
	}

	let value = $state<DateRange>({
		start: toCalendarDate(fromDate),
		end: toCalendarDate(toDate),
	});

	$effect(() => {
		fromDate = toISO(value.start as CalendarDate | undefined);
		toDate = toISO(value.end as CalendarDate | undefined);
	});

	function clear(e: MouseEvent) {
		e.stopPropagation();
		value = { start: undefined, end: undefined };
		fromDate = '';
		toDate = '';
	}

	const label = $derived(() => {
		if (!value.start && !value.end) return null;
		const fmt = (d: CalendarDate) =>
			format(new Date(d.year, d.month - 1, d.day), 'd MMM', { locale: es });
		if (value.start && value.end)
			return `${fmt(value.start as CalendarDate)} → ${fmt(value.end as CalendarDate)}`;
		if (value.start) return `${fmt(value.start as CalendarDate)} → ...`;
		return null;
	});
</script>

<Popover.Root bind:open>
	<Popover.Trigger>
		<button
			class="flex items-center gap-2 rounded-lg border border-[#1f1f24] bg-[#19191d] px-3 py-1.5 text-xs transition-colors hover:border-[#2b2c32]
				{label() ? 'text-[#e7e4ec]' : 'text-[#75757c]'}"
		>
			<CalendarDays size={13} class={label() ? 'text-violet-400' : 'text-[#3d3b3e]'} />
			<span class="font-mono">{label() ?? 'Rango de fechas'}</span>
			{#if label()}
				<span
					role="button"
					tabindex="0"
					onclick={clear}
					onkeydown={(e) => e.key === 'Enter' && clear(e as unknown as MouseEvent)}
					class="ml-1 text-[#3d3b3e] hover:text-[#a09da1] transition-colors"
				>
					<X size={11} />
				</span>
			{/if}
		</button>
	</Popover.Trigger>
	<Popover.Content class="w-auto p-0 border-[#2b2c32] bg-[#19191d] shadow-2xl" align="start">
		<div class="date-picker-popover">
		<RangeCalendar
			bind:value
			locale="es-MX"
			class="rounded-xl border-0"
		/>
		{#if value.start || value.end}
			<div class="border-t border-[#2b2c32] px-3 py-2 flex justify-end gap-2">
				<button
					onclick={() => { clear(new MouseEvent('click')); open = false; }}
					class="text-xs text-[#75757c] hover:text-[#a09da1] transition-colors"
				>
					Limpiar
				</button>
				<button
					onclick={() => open = false}
					class="rounded-md bg-violet-600 px-3 py-1 text-xs text-white hover:bg-violet-500 transition-colors"
				>
					Aplicar
				</button>
			</div>
		{/if}
		</div>
	</Popover.Content>
</Popover.Root>
