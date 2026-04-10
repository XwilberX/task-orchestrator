import { api } from './api';

export interface MetricsSummary {
	tasks_today: number;
	tasks_failed: number;
	tasks_queued: number;
	tasks_running: number;
	avg_duration_seconds: number;
}

export const metricsService = {
	summary: () => api.get<MetricsSummary>('/api/v1/metrics/summary')
};
