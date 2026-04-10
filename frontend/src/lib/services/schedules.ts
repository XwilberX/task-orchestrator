import { api } from './api';

export interface Schedule {
	id: string;
	definition_id: string;
	cron: string;
	status: 'active' | 'paused';
	last_run_at?: string;
	next_run_at?: string;
	created_at: string;
}

export const schedulesService = {
	list: () => api.get<Schedule[]>('/api/v1/schedules'),
	get: (id: string) => api.get<Schedule>(`/api/v1/schedules/${id}`),
	create: (data: Pick<Schedule, 'definition_id' | 'cron'>) =>
		api.post<Schedule>('/api/v1/schedules', data),
	update: (id: string, data: Pick<Schedule, 'definition_id' | 'cron'>) =>
		api.put<Schedule>(`/api/v1/schedules/${id}`, data),
	delete: (id: string) => api.delete<null>(`/api/v1/schedules/${id}`),
	toggle: (id: string) => api.patch<Schedule>(`/api/v1/schedules/${id}/toggle`)
};
