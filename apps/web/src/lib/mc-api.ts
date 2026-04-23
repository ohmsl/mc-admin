import { pb } from "@/lib/pocketbase"

export type ServerStatus = {
  serverReachable: boolean
  onlinePlayers: string[]
  onlineCount: number
  maxPlayers: number | null
  checkedAt: string
  degraded: boolean
  message?: string
}

export type ExecuteAction =
  | "whitelist_add"
  | "whitelist_remove"
  | "kick"
  | "say"
  | "save_world"
  | "restart_server"
  | "raw_command"

export type ExecuteRequest = {
  action: ExecuteAction
  payload: Record<string, unknown>
  requestId?: string
}

export type ExecuteResponse = {
  ok: boolean
  action: ExecuteAction
  message: string
  auditLogId?: string
  executedAt: string
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = pb.authStore.token
  const response = await fetch(`${pb.baseUrl}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: token } : {}),
      ...(init?.headers ?? {}),
    },
  })

  const rawText = await response.text()
  const data = rawText ? (JSON.parse(rawText) as Record<string, unknown>) : {}

  if (!response.ok) {
    const message =
      typeof data.message === "string"
        ? data.message
        : `Request failed (${response.status})`
    throw new Error(message)
  }

  return data as T
}

export function getServerStatus(): Promise<ServerStatus> {
  return apiFetch<ServerStatus>("/api/mc/status", { method: "GET" })
}

export function executeMinecraftAction(
  request: ExecuteRequest
): Promise<ExecuteResponse> {
  return apiFetch<ExecuteResponse>("/api/mc/execute", {
    method: "POST",
    body: JSON.stringify(request),
  })
}
