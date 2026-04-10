import { PUBLIC_API_URL, PUBLIC_API_KEY } from '$env/static/public';

export interface ApiResponse<T> {
	success: boolean;
	data: T;
	message?: string;
	error?: string;
}

export class ApiError extends Error {
	constructor(
		public status: number,
		message: string,
		public detail?: string
	) {
		super(message);
	}
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(`${PUBLIC_API_URL}${path}`, {
		...init,
		headers: {
			'Content-Type': 'application/json',
			'X-API-Key': PUBLIC_API_KEY,
			...init?.headers
		}
	});

	const body: ApiResponse<T> = await res.json();

	if (!body.success) {
		throw new ApiError(res.status, body.message ?? 'Error desconocido', body.error);
	}

	return body.data;
}

export const api = {
	get: <T>(path: string) => request<T>(path),
	post: <T>(path: string, data: unknown) =>
		request<T>(path, { method: 'POST', body: JSON.stringify(data) }),
	put: <T>(path: string, data: unknown) =>
		request<T>(path, { method: 'PUT', body: JSON.stringify(data) }),
	patch: <T>(path: string, data?: unknown) =>
		request<T>(path, { method: 'PATCH', body: data ? JSON.stringify(data) : undefined }),
	delete: <T>(path: string) => request<T>(path, { method: 'DELETE' })
};
