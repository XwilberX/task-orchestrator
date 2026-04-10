import { api } from './api';

export interface Task {
	id: string;
	definition_id?: string;
	definition?: string;
	runtime: string;
	code: string;
	args?: string[];
	packages?: string;
	input?: Record<string, unknown>;
	status: 'PENDING' | 'QUEUED' | 'RUNNING' | 'SUCCESS' | 'FAILED' | 'TIMEOUT' | 'CANCELLED';
	attempt: number;
	max_retries: number;
	timeout_seconds: number;
	memory_mb: number;
	exit_code?: number;
	created_at: string;
	started_at?: string;
	finished_at?: string;
}

export interface DispatchRequest {
	definition?: string;
	input?: Record<string, unknown>;
	runtime?: string;
	code?: string;
	args?: string[];
	packages?: string;
	timeout_seconds?: number;
	memory_mb?: number;
}

export interface TaskFilter {
	status?: string;
	runtime?: string;
	definition_id?: string;
	from?: string;
	to?: string;
}

export const tasksService = {
	list: (filter?: TaskFilter) => {
		const params = new URLSearchParams();
		if (filter?.status) params.set('status', filter.status);
		if (filter?.runtime) params.set('runtime', filter.runtime);
		if (filter?.definition_id) params.set('definition_id', filter.definition_id);
		if (filter?.from) params.set('from', filter.from);
		if (filter?.to) params.set('to', filter.to);
		const qs = params.toString();
		return api.get<Task[]>(`/api/v1/tasks${qs ? `?${qs}` : ''}`);
	},
	get: (id: string) => api.get<Task>(`/api/v1/tasks/${id}`),
	dispatch: (req: DispatchRequest) => api.post<Task>('/api/v1/tasks', req),
	cancel: (id: string) => api.delete<null>(`/api/v1/tasks/${id}`),
	logs: (id: string) => api.get<{ _msg: string; _time: string; stream: string }[]>(`/api/v1/tasks/${id}/logs`)
};
