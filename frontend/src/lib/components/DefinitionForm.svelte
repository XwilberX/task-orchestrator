<script lang="ts">
	import CodeEditor from './CodeEditor.svelte';
	import type { DefinitionPayload } from '$lib/services/definitions';

	let {
		initial = {},
		onsubmit,
		loading = false
	}: {
		initial?: Partial<DefinitionPayload>;
		onsubmit: (data: DefinitionPayload) => void;
		loading?: boolean;
	} = $props();

	const RUNTIMES = ['python','nodejs','typescript','go','java'] as const;

	const PLACEHOLDERS: Record<string, string> = {
		python: `import sys\nprint("Hello from task!", sys.argv)`,
		nodejs: `console.log("Hello from task!", process.argv);`,
		typescript: `const args: string[] = process.argv.slice(2);\nconsole.log("Hello!", args);`,
		go: `package main\nimport "fmt"\nfunc main() { fmt.Println("Hello from task!") }`,
		java: `public class Main {\n    public static void main(String[] args) {\n        System.out.println("Hello from task!");\n    }\n}`
	};

	// Form state
	let name = $state(initial.name ?? '');
	let description = $state(initial.description ?? '');
	let runtime = $state<typeof RUNTIMES[number]>(initial.runtime ?? 'python');
	let code = $state(initial.code ?? PLACEHOLDERS.python);
	let packages = $state(initial.packages ?? '');
	let timeoutSeconds = $state(initial.timeout_seconds ?? 60);
	let maxRetries = $state(initial.max_retries ?? 3);
	let backoffMultiplier = $state(initial.backoff_multiplier ?? 5);
	let maxConcurrent = $state(initial.max_concurrent ?? 1);
	let memoryMb = $state(initial.memory_mb ?? 256);
	let cpuShares = $state(initial.cpu_shares ?? 512);
	let networkEnabled = $state(initial.network_enabled ?? false);

	// Errors
	let errors = $state<Record<string, string>>({});

	function handleRuntimeChange() {
		if (!initial.code) {
			code = PLACEHOLDERS[runtime];
		}
	}

	function validate(): boolean {
		const e: Record<string, string> = {};
		if (!name.trim()) e.name = 'El nombre es requerido';
		if (!code.trim()) e.code = 'El código es requerido';
		errors = e;
		return Object.keys(e).length === 0;
	}

	function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!validate()) return;
		onsubmit({
			name: name.trim(),
			description: description.trim(),
			runtime,
			code,
			packages: packages.trim(),
			timeout_seconds: timeoutSeconds,
			max_retries: maxRetries,
			backoff_multiplier: backoffMultiplier,
			max_concurrent: maxConcurrent,
			memory_mb: memoryMb,
			cpu_shares: cpuShares,
			network_enabled: networkEnabled
		});
	}

	const fieldClass = 'w-full rounded-md border border-[#1f1f24] bg-[#19191d] px-3 py-2 text-sm text-[#e7e4ec] placeholder-[#3d3b3e] focus:border-violet-500/50 focus:outline-none transition-colors';
	const labelClass = 'mb-1.5 block text-xs font-medium text-[#75757c]';
	const errorClass = 'mt-1 text-xs text-red-400';
</script>

<form onsubmit={handleSubmit} class="flex h-full flex-col">
	<div class="flex flex-1 overflow-hidden">

		<!-- Panel izquierdo: campos del formulario -->
		<div class="w-80 shrink-0 overflow-auto border-r border-[#1f1f24] p-6 space-y-4">

			<div>
				<label class={labelClass} for="name">Nombre <span class="text-red-400">*</span></label>
				<input
					id="name"
					bind:value={name}
					class={fieldClass}
					placeholder="send-report-email"
				/>
				{#if errors.name}<p class={errorClass}>{errors.name}</p>{/if}
			</div>

			<div>
				<label class={labelClass} for="description">Descripción</label>
				<input
					id="description"
					bind:value={description}
					class={fieldClass}
					placeholder="Descripción opcional..."
				/>
			</div>

			<div>
				<label class={labelClass} for="runtime">Runtime</label>
				<select
					id="runtime"
					bind:value={runtime}
					onchange={handleRuntimeChange}
					class="{fieldClass} cursor-pointer"
				>
					{#each RUNTIMES as rt}
						<option value={rt}>{rt}</option>
					{/each}
				</select>
			</div>

			<div>
				<label class={labelClass} for="packages">Paquetes</label>
				<input
					id="packages"
					bind:value={packages}
					class={fieldClass}
					placeholder="requests pandas numpy"
				/>
				<p class="mt-1 text-xs text-[#3d3b3e]">Separados por espacios</p>
			</div>

			<div class="h-px bg-[#1f1f24]"></div>

			<div class="grid grid-cols-2 gap-3">
				<div>
					<label class={labelClass} for="timeout">Timeout (s)</label>
					<input id="timeout" type="number" bind:value={timeoutSeconds} min="1" max="3600" class={fieldClass} />
				</div>
				<div>
					<label class={labelClass} for="retries">Max reintentos</label>
					<input id="retries" type="number" bind:value={maxRetries} min="0" max="10" class={fieldClass} />
				</div>
				<div>
					<label class={labelClass} for="backoff">Backoff multiplier</label>
					<input id="backoff" type="number" bind:value={backoffMultiplier} min="1" class={fieldClass} />
				</div>
				<div>
					<label class={labelClass} for="concurrent">Max concurrente</label>
					<input id="concurrent" type="number" bind:value={maxConcurrent} min="1" class={fieldClass} />
				</div>
				<div>
					<label class={labelClass} for="memory">Memoria (MB)</label>
					<input id="memory" type="number" bind:value={memoryMb} min="64" class={fieldClass} />
				</div>
				<div>
					<label class={labelClass} for="cpu">CPU shares</label>
					<input id="cpu" type="number" bind:value={cpuShares} min="0" class={fieldClass} />
				</div>
			</div>

			<div class="flex items-center justify-between rounded-md border border-[#1f1f24] bg-[#19191d] px-3 py-2.5">
				<div>
					<p class="text-sm text-[#e7e4ec]">Red habilitada</p>
					<p class="text-xs text-[#75757c]">Permite acceso a internet</p>
				</div>
				<button
					type="button"
					onclick={() => networkEnabled = !networkEnabled}
					class="relative h-5 w-9 rounded-full transition-colors {networkEnabled ? 'bg-violet-600' : 'bg-[#3d3b3e]'}"
				>
					<span class="absolute top-0.5 h-4 w-4 rounded-full bg-white shadow transition-transform {networkEnabled ? 'translate-x-4' : 'translate-x-0.5'}"></span>
				</button>
			</div>

		</div>

		<!-- Panel derecho: Monaco Editor -->
		<div class="flex flex-1 flex-col overflow-hidden p-6 gap-3">
			<div class="flex items-center justify-between">
				<label class={labelClass}>Código <span class="text-red-400">*</span></label>
				<span class="font-mono text-xs text-[#75757c]">{runtime}</span>
			</div>
			<div class="flex-1 min-h-0">
				<CodeEditor bind:value={code} language={runtime} height="100%" />
			</div>
			{#if errors.code}<p class={errorClass}>{errors.code}</p>{/if}
		</div>
	</div>

	<!-- Footer -->
	<div class="flex items-center justify-end gap-3 border-t border-[#1f1f24] px-6 py-4">
		<a href="/definitions" class="text-sm text-[#75757c] hover:text-[#e7e4ec] transition-colors">
			Cancelar
		</a>
		<button
			type="submit"
			disabled={loading}
			class="flex items-center gap-1.5 rounded-md bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
		>
			{loading ? 'Guardando...' : 'Guardar definición'}
		</button>
	</div>
</form>
