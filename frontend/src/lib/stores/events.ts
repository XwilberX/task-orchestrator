import { PUBLIC_API_URL, PUBLIC_API_KEY } from '$env/static/public';
import type { QueryClient } from '@tanstack/svelte-query';

export function connectEventSource(queryClient: QueryClient) {
	const url = `${PUBLIC_API_URL}/api/v1/events?api_key=${PUBLIC_API_KEY}`;
	const es = new EventSource(url);

	es.onmessage = (e) => {
		try {
			const event = JSON.parse(e.data) as { task_id: string; status: string };
			// Invalidar queries relevantes para que la UI se actualice
			queryClient.invalidateQueries({ queryKey: ['tasks'] });
			queryClient.invalidateQueries({ queryKey: ['task', event.task_id] });
			queryClient.invalidateQueries({ queryKey: ['metrics'] });
		} catch {
			// ignorar eventos mal formados
		}
	};

	es.onerror = () => {
		// Reconnect automático del browser — no hacer nada
	};

	return () => es.close();
}
