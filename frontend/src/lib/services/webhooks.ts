import { api } from './api';

export interface Webhook {
	id: string;
	url: string;
	created_at: string;
}

export interface WebhookDelivery {
	id: string;
	webhook_id: string;
	task_id: string;
	status_code: number;
	success: boolean;
	response: string;
	attempt_at: string;
}

export const webhooksService = {
	list: () => api.get<Webhook[]>('/api/v1/webhooks'),
	create: (url: string) => api.post<Webhook>('/api/v1/webhooks', { url }),
	delete: (id: string) => api.delete<null>(`/api/v1/webhooks/${id}`),
	deliveries: (id: string) => api.get<WebhookDelivery[]>(`/api/v1/webhooks/${id}/logs`)
};
