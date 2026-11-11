let csrfToken = '';

export function setCSRFToken(token: string | undefined | null) {
  csrfToken = token ?? '';
}

export class ApiError extends Error {
  status: number;
  fields: Record<string, string>;

  constructor(message: string, status: number, fields: Record<string, string> = {}) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.fields = fields;
  }
}

type RequestOptions = Omit<RequestInit, 'body'> & { body?: unknown };

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { body, headers: rawHeaders, ...rest } = options;
  const headers = new Headers(rawHeaders);
  const method = (options.method ?? 'GET').toUpperCase();

  if (method !== 'GET' && method !== 'HEAD' && csrfToken) {
    headers.set('X-CSRF-Token', csrfToken);
  }

  let payload: BodyInit | undefined;
  if (body instanceof FormData) {
    payload = body;
  } else if (body !== undefined) {
    headers.set('Content-Type', 'application/json');
    payload = JSON.stringify(body);
  }

  const response = await fetch(`/api/v1${path}`, {
    ...rest,
    method,
    headers,
    body: payload,
    credentials: 'include',
  });

  const contentType = response.headers.get('Content-Type') ?? '';
  const data = contentType.includes('application/json')
    ? await response.json()
    : await response.text();

  if (!response.ok) {
    const message =
      typeof data === 'object' && data && 'message' in data
        ? String((data as { message: unknown }).message)
        : 'Não foi possível concluir a operação.';
    if (
      (response.status === 401 && !path.startsWith('/auth/')) ||
      (response.status === 403 && message.toLowerCase().includes('desativada'))
    ) {
      window.dispatchEvent(new Event('auth:expired'));
    }
    const fields =
      typeof data === 'object' && data && 'errors' in data
        ? ((data as { errors: Record<string, string> }).errors ?? {})
        : {};
    throw new ApiError(message, response.status, fields);
  }

  return data as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) => request<T>(path, { method: 'POST', body }),
  put: <T>(path: string, body?: unknown) => request<T>(path, { method: 'PUT', body }),
  patch: <T>(path: string, body?: unknown) => request<T>(path, { method: 'PATCH', body }),
  delete: <T>(path: string, body?: unknown) => request<T>(path, { method: 'DELETE', body }),
  upload: <T>(path: string, body: FormData) => request<T>(path, { method: 'POST', body }),
};
