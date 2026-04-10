<script lang="ts">
	import type { Task } from '$lib/services/tasks';

	let { status }: { status: Task['status'] } = $props();

	const config: Record<Task['status'], { label: string; class: string; dot?: boolean }> = {
		SUCCESS:   { label: 'Success',   class: 'bg-green-500/10 text-green-400 border-green-500/20' },
		FAILED:    { label: 'Failed',    class: 'bg-red-500/10 text-red-400 border-red-500/20' },
		RUNNING:   { label: 'Running',   class: 'bg-violet-500/10 text-violet-300 border-violet-500/20', dot: true },
		QUEUED:    { label: 'Queued',    class: 'bg-amber-500/10 text-amber-400 border-amber-500/20' },
		PENDING:   { label: 'Pending',   class: 'bg-zinc-500/10 text-zinc-400 border-zinc-500/20' },
		TIMEOUT:   { label: 'Timeout',   class: 'bg-orange-500/10 text-orange-400 border-orange-500/20' },
		CANCELLED: { label: 'Cancelled', class: 'bg-zinc-500/10 text-zinc-500 border-zinc-600/20' }
	};

	const c = $derived(config[status] ?? config.PENDING);
</script>

<span class="inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium {c.class}">
	{#if c.dot}
		<span class="h-1.5 w-1.5 animate-pulse rounded-full bg-current"></span>
	{/if}
	{c.label}
</span>
