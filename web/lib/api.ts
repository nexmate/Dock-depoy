export type StackStatus = "running" | "partial" | "stopped" | "inactive"

export type StackSummary = {
  name: string
  status: StackStatus
  containers: { running: number; total: number }
}

export type StackDetail = {
  name: string
  compose: string
  env: string
  status: StackStatus
}

export type ContainerInfo = {
  id: string
  name: string
  service: string
  state: string
  status: string
  image: string
  ports: { public: number; private: number; proto: string }[]
}

export type StackAction = "up" | "stop" | "restart" | "pull" | "down"

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public detail?: string
  ) {
    super(message)
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`/api${path}`, {
    method,
    credentials: "same-origin",
    headers: body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new ApiError(res.status, err.error ?? res.statusText, err.detail)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

const enc = encodeURIComponent

export const api = {
  authStatus: () =>
    request<{ setupDone: boolean; authenticated: boolean }>("GET", "/auth/status"),
  setup: (username: string, password: string) =>
    request<void>("POST", "/auth/setup", { username, password }),
  login: (username: string, password: string) =>
    request<void>("POST", "/auth/login", { username, password }),
  logout: () => request<void>("POST", "/auth/logout"),
  listStacks: () => request<StackSummary[]>("GET", "/stacks"),
  getStack: (name: string) => request<StackDetail>("GET", `/stacks/${enc(name)}`),
  createStack: (name: string, compose: string, env: string) =>
    request<StackDetail>("POST", "/stacks", { name, compose, env }),
  saveStack: (name: string, compose: string, env: string) =>
    request<void>("PUT", `/stacks/${enc(name)}`, { compose, env }),
  deleteStack: (name: string) => request<void>("DELETE", `/stacks/${enc(name)}`),
  action: (name: string, action: StackAction) =>
    request<{ jobId: string }>("POST", `/stacks/${enc(name)}/${action}`),
  containers: (name: string) =>
    request<ContainerInfo[]>("GET", `/stacks/${enc(name)}/containers`),
}

export const jobStreamUrl = (jobId: string) => `/api/jobs/${enc(jobId)}/stream`
export const logsStreamUrl = (containerId: string, tail = 200) =>
  `/api/containers/${enc(containerId)}/logs?tail=${tail}`
