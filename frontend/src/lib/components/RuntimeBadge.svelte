<script lang="ts">
	let { runtime }: { runtime: string } = $props();

	const config: Record<string, { label: string; class: string }> = {
		python:     { label: 'Python',     class: 'bg-blue-500/10 text-blue-400 border-blue-500/20' },
		nodejs:     { label: 'Node.js',    class: 'bg-green-500/10 text-green-400 border-green-500/20' },
		typescript: { label: 'TypeScript', class: 'bg-sky-500/10 text-sky-400 border-sky-500/20' },
		go:         { label: 'Go',         class: 'bg-cyan-500/10 text-cyan-400 border-cyan-500/20' },
		java:       { label: 'Java',       class: 'bg-orange-500/10 text-orange-400 border-orange-500/20' }
	};

	const c = $derived(config[runtime] ?? { label: runtime, class: 'bg-zinc-500/10 text-zinc-400 border-zinc-500/20' });
</script>

<span class="inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium {c.class}">
	{c.label}
</span>
