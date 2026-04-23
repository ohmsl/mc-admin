import PocketBase from "pocketbase"

const fallbackBaseUrl = "http://127.0.0.1:8090"

const baseUrl =
  import.meta.env.VITE_POCKETBASE_URL ??
  (typeof window !== "undefined" ? window.location.origin : fallbackBaseUrl)

export const pb = new PocketBase(baseUrl)
pb.autoCancellation(false)

export type AuthRecord = {
  id: string
  email?: string
  role?: string
  [key: string]: unknown
}

type LegacyAdminAuthResponse = {
  token: string
  admin?: AuthRecord
  record?: AuthRecord
}

function decodeJwtPayload(token: string): Record<string, unknown> | null {
  const parts = token.split(".")
  if (parts.length < 2) {
    return null
  }

  try {
    const normalized = parts[1].replace(/-/g, "+").replace(/_/g, "/")
    const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=")
    const raw = atob(padded)
    return JSON.parse(raw) as Record<string, unknown>
  } catch {
    return null
  }
}

export function isPocketBaseAdminSession(): boolean {
  if (pb.authStore.isSuperuser) {
    return true
  }

  const payload = decodeJwtPayload(pb.authStore.token)
  return payload?.type === "admin"
}

export async function authWithUsersOrAdmin(
  email: string,
  password: string
): Promise<void> {
  try {
    await pb.collection("users").authWithPassword(email, password)
    return
  } catch {
    // Continue through admin/superuser fallbacks.
  }

  try {
    await pb.collection("_superusers").authWithPassword(email, password)
    return
  } catch {
    // Continue to legacy admin endpoint for PocketBase <= 0.22.
  }

  const response = await pb.send<LegacyAdminAuthResponse>(
    "/api/admins/auth-with-password",
    {
      method: "POST",
      body: {
        identity: email,
        password,
      },
    }
  )

  pb.authStore.save(
    response.token,
    (response.admin ?? response.record ?? null) as any
  )
}
