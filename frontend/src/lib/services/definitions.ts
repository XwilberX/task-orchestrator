import { api } from './api';

export interface Definition {
	id: string;
	name: string;
	description?: string;
	runtime: 'python' | 'nodejs' | 'go' | 'java';
	runtime_version?: string;
	code: string;
	packages?: string;
	timeout_seconds: number;
	max_retries: number;
	backoff_multiplier: number;
	max_concurrent: number;
	memory_mb: number;
	cpu_shares: number;
	network_enabled: boolean;
	created_at: string;
	updated_at: string;
}

export type DefinitionPayload = Omit<Definition, 'id' | 'created_at' | 'updated_at'>;

export interface RuntimeMeta {
	label: string;
	image: string;
	suffix: string;
	versions: string[];
}

export const definitionsService = {
	list: () => api.get<Definition[]>('/api/v1/definitions'),
	get: (id: string) => api.get<Definition>(`/api/v1/definitions/${id}`),
	create: (data: DefinitionPayload) => api.post<Definition>('/api/v1/definitions', data),
	update: (id: string, data: DefinitionPayload) => api.put<Definition>(`/api/v1/definitions/${id}`, data),
	delete: (id: string) => api.delete<null>(`/api/v1/definitions/${id}`),
	runtimes: () => api.get<Record<string, RuntimeMeta>>('/api/v1/runtimes')
};
