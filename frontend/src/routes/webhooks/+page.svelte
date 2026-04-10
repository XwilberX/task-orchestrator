<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { webhooksService, type Webhook, type WebhookDelivery } from '$lib/services/webhooks';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { format, formatDistanceToNow } from 'date-fns';
	import { es } from 'date-fns/locale';
	import { Plus, Trash2, ChevronDown, ChevronRight, Webhook as WebhookIcon, CheckCircle2, XCircle } from '@lucide/svelte';

	const queryClient = useQueryClient();

	const webhooks = createQuery(() => ({
		queryKey: ['webhooks'],
		queryFn: webhooksService.list
	}));

	const webhooksData = $derived(webhooks.data as Webhook[] | undefined);

	// ─── Historial de entregas expandido ──────────────────────────────────────
	let expandedId = $state<string | null>(null);
	let deliveries = $state<Record<string, WebhookDelivery[]>>({});
	let deliveriesLoading = $state<Record<string, boolean>>({});

	async function toggleDeliveries(id: string) {
		if (expandedId === id) {
			expandedId = null;
			return;
		}
		expandedId = id;
		if (!deliveries[id]) {
			deliveriesLoading = { ...deliveriesLoading, [id]: true };
			try {
				const data = await webhooksService.deliveries(id);
				deliveries = { ...deliveries, [id]: data };
			} finally {
				deliveriesLoading = { ...deliveriesLoading, [id]: false };
			}
		}
	}

	// ─── Eliminar webhook ──────────────────────────────────────────────────────
	async function deleteWebhook(id: string) {
		if (!confirm('¿Eliminar este webhook?')) return;
		await webhooksService.delete(id);
		queryClient.invalidateQueries({ queryKey: ['webhooks'] });
		if (expandedId === id) expandedId = null;
	}

	// ─── Formulario inline ─────────────────────────────────────────────────────
	let newUrl = $state('');
	let formError = $state('');
	let formLoading = $state(false);

	async function createWebhook(e: SubmitEvent) {
		e.preventDefault();
		formError = '';
		const url = newUrl.trim();
		if (!url) { formError = 'La URL es requerida'; return; }
		if (!url.startsWith('http')) { formError = 'La URL debe comenzar con http:// o https://'; return; }
		formLoading = true;
		try {
			await webhooksService.create(url);
			queryClient.invalidateQueries({ queryKey: ['webhooks'] });
			newUrl = '';
		} catch (err: unknown) {
			formError = err instanceof Error ? err.message : 'Error al registrar el webhook';
		} finally {
			formLoading = false;
		}
	}

	// ─── Helpers ──────────────────────────────────────────────────────────────
	function relativeTime(date: string): string {
		return formatDistanceToNow(new Date(date), { addSuffix: true, locale: es });
	}

	function fmt(date: string): string {
		return format(new Date(date), "d MMM, HH:mm:ss", { locale: es });
	}

	function truncateUrl(url: string, max = 52): string {
		return url.length > max ? url.slice(0, max) + '…' : url;
	}

	function lastDeliveryStatus(wh: Webhook): { ok: boolean; code?: number } | null {
		const list = deliveries[wh.id];
		if (!list || list.length === 0) return null;
		const last = list[0];
		return { ok: last.success, code: last.status_code };
	}
</script>

<div class="flex h-full flex-col">
	<!-- Header -->
	<div class="flex items-center justify-between border-b border-[#1f1f24] px-6 py-4">
		<div>
			<h1 class="text-lg font-semibold text-[#e7e4ec]">Webhooks</h1>
			<p class="text-xs text-[#75757c]">Notificaciones en tiempo real cuando las tareas terminan</p>
		</div>
	</div>

	<div class="flex-1 overflow-auto">
		<!-- Tabla -->
		{#if webhooks.isLoading}
			<div class="space-y-px p-4">
				{#each { length: 4 } as _}
					<Skeleton class="h-14 w-full rounded bg-[#19191d]" />
				{/each}
			</div>
		{:else if (webhooksData?.length ?? 0) === 0}
			<div class="flex flex-col items-center justify-center py-16 text-center">
				<WebhookIcon size={32} class="mb-3 text-[#3d3b3e]" />
				<p class="text-sm text-[#75757c]">No hay webhooks registrados</p>
				<p class="mt-1 text-xs text-[#3d3b3e]">Agrega uno con el formulario de abajo</p>
			</div>
		{:else}
			<div class="divide-y divide-[#1f1f24]">
				{#each webhooksData ?? [] as wh}
					<div>
						<!-- Fila principal -->
						<div class="flex items-center gap-3 px-4 py-3 hover:bg-[#19191d] transition-colors">
							<button
								onclick={() => toggleDeliveries(wh.id)}
								class="text-[#75757c] hover:text-[#e7e4ec] transition-colors"
								title={expandedId === wh.id ? 'Colapsar' : 'Ver historial'}
							>
								{#if expandedId === wh.id}
									<ChevronDown size={14} />
								{:else}
									<ChevronRight size={14} />
								{/if}
							</button>

							<div class="flex-1 min-w-0">
								<p class="font-mono text-sm text-[#e7e4ec] truncate" title={wh.url}>
									{truncateUrl(wh.url)}
								</p>
								<p class="text-xs text-[#75757c]">Registrado {relativeTime(wh.created_at)}</p>
							</div>

							<div class="flex items-center gap-3">
								<button
									onclick={() => toggleDeliveries(wh.id)}
									class="text-xs text-[#75757c] hover:text-[#d0bcff] transition-colors"
								>
									Ver entregas
								</button>
								<button
									onclick={() => deleteWebhook(wh.id)}
									class="rounded p-1.5 text-[#75757c] hover:bg-red-500/10 hover:text-red-400 transition-colors"
									title="Eliminar"
								>
									<Trash2 size={14} />
								</button>
							</div>
						</div>

						<!-- Historial de entregas expandido -->
						{#if expandedId === wh.id}
							<div class="border-t border-[#1f1f24] bg-[#080809] px-8 py-4">
								<p class="mb-3 text-xs font-medium uppercase tracking-wider text-[#75757c]">
									Últimas entregas
								</p>

								{#if deliveriesLoading[wh.id]}
									<div class="space-y-2">
										{#each { length: 3 } as _}
											<Skeleton class="h-8 w-full rounded bg-[#19191d]" />
										{/each}
									</div>
								{:else if (deliveries[wh.id]?.length ?? 0) === 0}
									<p class="text-xs text-[#3d3b3e]">Sin entregas registradas aún</p>
								{:else}
									<table class="w-full text-xs">
										<thead>
											<tr class="border-b border-[#1f1f24]">
												<th class="pb-2 text-left font-medium text-[#75757c]">Estado</th>
												<th class="pb-2 text-left font-medium text-[#75757c]">HTTP</th>
												<th class="pb-2 text-left font-medium text-[#75757c]">Task ID</th>
												<th class="pb-2 text-left font-medium text-[#75757c]">Fecha</th>
												<th class="pb-2 text-left font-medium text-[#75757c]">Respuesta</th>
											</tr>
										</thead>
										<tbody class="divide-y divide-[#1f1f24]">
											{#each deliveries[wh.id] as delivery}
												<tr>
													<td class="py-2 pr-4">
														{#if delivery.success}
															<span class="flex items-center gap-1 text-green-400">
																<CheckCircle2 size={12} />
																OK
															</span>
														{:else}
															<span class="flex items-center gap-1 text-red-400">
																<XCircle size={12} />
																Error
															</span>
														{/if}
													</td>
													<td class="py-2 pr-4">
														<span class="font-mono {delivery.success ? 'text-green-400' : 'text-red-400'}">
															{delivery.status_code || '—'}
														</span>
													</td>
													<td class="py-2 pr-4">
														<a
															href="/tasks/{delivery.task_id}"
															class="font-mono text-[#75757c] hover:text-[#d0bcff] transition-colors"
														>
															#{delivery.task_id.slice(0, 8)}
														</a>
													</td>
													<td class="py-2 pr-4 text-[#75757c]">{fmt(delivery.attempt_at)}</td>
													<td class="py-2 max-w-xs truncate font-mono text-[#3d3b3e]" title={delivery.response}>
														{delivery.response || '—'}
													</td>
												</tr>
											{/each}
										</tbody>
									</table>
								{/if}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}

		<!-- Formulario inline -->
		<div class="border-t border-[#1f1f24] bg-[#19191d] px-6 py-5">
			<div class="flex items-center gap-2 mb-4">
				<Plus size={14} class="text-[#75757c]" />
				<h2 class="text-xs font-medium uppercase tracking-wider text-[#75757c]">Registrar webhook</h2>
			</div>

			<form onsubmit={createWebhook} class="flex items-end gap-3">
				<div class="flex-1">
					<label class="mb-1.5 block text-xs text-[#75757c]">URL del endpoint</label>
					<input
						bind:value={newUrl}
						placeholder="https://tu-servidor.com/webhook"
						class="w-full rounded-md border border-[#2b2c32] bg-[#0e0e10] px-3 py-2 font-mono text-sm text-[#e7e4ec] placeholder-[#3d3b3e] focus:border-violet-500/50 focus:outline-none"
					/>
				</div>
				<button
					type="submit"
					disabled={formLoading}
					class="flex items-center gap-1.5 rounded-md bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-500 disabled:opacity-50 transition-colors"
				>
					<Plus size={14} />
					{formLoading ? 'Registrando...' : 'Agregar'}
				</button>
			</form>

			{#if formError}
				<p class="mt-2 text-xs text-red-400">{formError}</p>
			{/if}

			<p class="mt-3 text-xs text-[#3d3b3e]">
				El payload se firma con HMAC-SHA256 en el header <code class="text-[#75757c]">X-Signature</code>.
				Se realizan hasta 3 intentos con 10s de backoff.
			</p>
		</div>
	</div>
</div>
