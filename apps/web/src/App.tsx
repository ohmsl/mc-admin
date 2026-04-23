import { useCallback, useEffect, useMemo, useState } from "react"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import { executeMinecraftAction, getServerStatus } from "@/lib/mc-api"
import type {
  ExecuteAction,
  ExecuteResponse,
  ServerStatus,
} from "@/lib/mc-api"
import { authWithUsersOrAdmin, isPocketBaseAdminSession, pb } from "@/lib/pocketbase"
import type { AuthRecord } from "@/lib/pocketbase"
import { toast } from "sonner"

type Role = "viewer" | "operator" | "owner"

type AuditLogRecord = {
  id: string
  created: string
  actor_email?: string
  actor_role?: string
  action: string
  outcome: "success" | "failed" | "denied"
  error?: string
  request_id?: string
}

const statusPollIntervalMs = 15000
const auditPollIntervalMs = 30000

function normalizeRole(raw: unknown): Role {
  if (raw === "owner") {
    return "owner"
  }

  if (raw === "operator") {
    return "operator"
  }

  return "viewer"
}

function canExecuteAction(role: Role, action: ExecuteAction): boolean {
  if (role === "owner") {
    return action !== "raw_command"
  }

  if (role === "operator") {
    return [
      "whitelist_add",
      "whitelist_remove",
      "kick",
      "say",
      "save_world",
    ].includes(action)
  }

  return false
}

function createRequestId() {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID()
  }

  return `req_${Date.now()}`
}

function formatTimestamp(raw: string) {
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) {
    return "Unknown"
  }

  return date.toLocaleString()
}

function formatError(error: unknown): string {
  if (error instanceof Error) {
    return error.message
  }

  return "Unexpected error"
}

function LoginScreen({
  isSubmitting,
  onSubmit,
}: {
  isSubmitting: boolean
  onSubmit: (email: string, password: string) => Promise<void>
}) {
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    await onSubmit(email, password)
  }

  return (
    <div className="mx-auto flex min-h-svh w-full max-w-5xl items-center px-6 py-10">
      <Card className="w-full">
        <CardHeader>
          <CardTitle>Minecraft Operations</CardTitle>
          <CardDescription>
            Sign in with a PocketBase users account or superuser account.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
            <div className="flex flex-col gap-2">
              <label className="text-muted-foreground" htmlFor="email">
                Email
              </label>
              <Input
                id="email"
                type="email"
                autoComplete="username"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                required
              />
            </div>

            <div className="flex flex-col gap-2">
              <label className="text-muted-foreground" htmlFor="password">
                Password
              </label>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                required
              />
            </div>

            <Button disabled={isSubmitting} type="submit">
              {isSubmitting ? "Signing in..." : "Sign in"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

function App() {
  const [authRecord, setAuthRecord] = useState<AuthRecord | null>(
    (pb.authStore.record as AuthRecord | null) ?? null
  )
  const [isAdminSession, setIsAdminSession] = useState<boolean>(
    isPocketBaseAdminSession()
  )
  const [isSigningIn, setIsSigningIn] = useState(false)

  useEffect(() => {
    const unsubscribe = pb.authStore.onChange((_: string, record: unknown) => {
      setAuthRecord((record as AuthRecord | null) ?? null)
      setIsAdminSession(isPocketBaseAdminSession())
    }, true)

    return () => {
      unsubscribe()
    }
  }, [])

  const handleLogin = useCallback(async (email: string, password: string) => {
    setIsSigningIn(true)
    try {
      await authWithUsersOrAdmin(email, password)
      toast.success("Signed in")
    } catch (error) {
      toast.error(formatError(error))
    } finally {
      setIsSigningIn(false)
    }
  }, [])

  const handleLogout = useCallback(() => {
    pb.authStore.clear()
    toast.success("Signed out")
  }, [])

  if (!authRecord) {
    return <LoginScreen isSubmitting={isSigningIn} onSubmit={handleLogin} />
  }

  return (
    <Dashboard
      authRecord={authRecord}
      isAdminSession={isAdminSession}
      onLogout={handleLogout}
    />
  )
}

function Dashboard({
  authRecord,
  isAdminSession,
  onLogout,
}: {
  authRecord: AuthRecord
  isAdminSession: boolean
  onLogout: () => void
}) {
  const role = useMemo(
    () => (isAdminSession ? "owner" : normalizeRole(authRecord.role)),
    [authRecord.role, isAdminSession]
  )

  const [status, setStatus] = useState<ServerStatus | null>(null)
  const [statusLoading, setStatusLoading] = useState(true)
  const [statusError, setStatusError] = useState<string | null>(null)

  const [auditLogs, setAuditLogs] = useState<AuditLogRecord[]>([])
  const [auditLoading, setAuditLoading] = useState(true)
  const [auditError, setAuditError] = useState<string | null>(null)

  const [pendingAction, setPendingAction] = useState<ExecuteAction | null>(null)

  const [whitelistPlayer, setWhitelistPlayer] = useState("")
  const [kickPlayer, setKickPlayer] = useState("")
  const [kickReason, setKickReason] = useState("")
  const [serverMessage, setServerMessage] = useState("")
  const [restartMessage, setRestartMessage] = useState(
    "Server restarting for maintenance."
  )

  const canRun = useCallback(
    (action: ExecuteAction) => canExecuteAction(role, action),
    [role]
  )

  const loadStatus = useCallback(async () => {
    try {
      const nextStatus = await getServerStatus()
      setStatus(nextStatus)
      setStatusError(null)
    } catch (error) {
      setStatusError(formatError(error))
    } finally {
      setStatusLoading(false)
    }
  }, [])

  const loadAuditLogs = useCallback(async () => {
    try {
      const result = await pb.collection("mc_audit_logs").getList<AuditLogRecord>(1, 20, {
        sort: "-created",
      })
      setAuditLogs(result.items)
      setAuditError(null)
    } catch (error) {
      setAuditError(formatError(error))
    } finally {
      setAuditLoading(false)
    }
  }, [])

  useEffect(() => {
    const bootstrapTimer = window.setTimeout(() => {
      void loadStatus()
      void loadAuditLogs()
    }, 0)

    const statusTimer = window.setInterval(() => {
      void loadStatus()
    }, statusPollIntervalMs)

    const auditTimer = window.setInterval(() => {
      void loadAuditLogs()
    }, auditPollIntervalMs)

    return () => {
      window.clearTimeout(bootstrapTimer)
      window.clearInterval(statusTimer)
      window.clearInterval(auditTimer)
    }
  }, [loadAuditLogs, loadStatus])

  const runAction = useCallback(
    async (action: ExecuteAction, payload: Record<string, unknown>) => {
      if (!canRun(action)) {
        toast.error("This role is not allowed to run that action")
        return
      }

      setPendingAction(action)

      try {
        const response: ExecuteResponse = await executeMinecraftAction({
          action,
          payload,
          requestId: createRequestId(),
        })

        toast.success(response.message)
        await Promise.all([loadStatus(), loadAuditLogs()])
      } catch (error) {
        toast.error(formatError(error))
      } finally {
        setPendingAction(null)
      }
    },
    [canRun, loadAuditLogs, loadStatus]
  )

  return (
    <div className="mx-auto flex min-h-svh w-full max-w-[1400px] flex-col gap-6 px-6 py-6 lg:px-10">
      <header className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex flex-col gap-2">
          <p className="text-muted-foreground">Homelab Minecraft Admin Panel</p>
          <h1 className="font-heading text-2xl font-semibold tracking-tight">
            Operations Console
          </h1>
        </div>

        <div className="flex items-center gap-2">
          {isAdminSession ? <Badge>PocketBase Admin</Badge> : null}
          <Badge variant="secondary">Role: {role}</Badge>
          <Badge variant="outline">{authRecord.email ?? authRecord.id}</Badge>
          <Button variant="outline" onClick={onLogout}>
            Sign out
          </Button>
        </div>
      </header>

      <div className="grid grid-cols-1 gap-6 xl:grid-cols-[1.1fr_1.6fr]">
        <div className="flex flex-col gap-6">
          <Card>
            <CardHeader>
              <CardTitle>Server Status</CardTitle>
              <CardDescription>
                Reachability, current player count, and live player list.
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              {statusLoading ? (
                <div className="flex flex-col gap-2">
                  <Skeleton className="h-8" />
                  <Skeleton className="h-8" />
                  <Skeleton className="h-24" />
                </div>
              ) : (
                <>
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">Server</span>
                    <Badge
                      variant={status?.serverReachable ? "default" : "destructive"}
                    >
                      {status?.serverReachable ? "Reachable" : "Unreachable"}
                    </Badge>
                  </div>

                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">Players online</span>
                    <span>
                      {status?.onlineCount ?? 0}
                      {status?.maxPlayers ? ` / ${status.maxPlayers}` : ""}
                    </span>
                  </div>

                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">Last check</span>
                    <span>{status ? formatTimestamp(status.checkedAt) : "Unknown"}</span>
                  </div>

                  {status?.degraded ? (
                    <Badge variant="outline">Partial status parse</Badge>
                  ) : null}

                  {statusError ? (
                    <Badge variant="destructive">{statusError}</Badge>
                  ) : null}

                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Online Players</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {status?.onlinePlayers.length ? (
                        status.onlinePlayers.map((player) => (
                          <TableRow key={player}>
                            <TableCell>{player}</TableCell>
                          </TableRow>
                        ))
                      ) : (
                        <TableRow>
                          <TableCell className="text-muted-foreground">
                            No players online.
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </>
              )}
            </CardContent>
            <CardFooter className="justify-between">
              <p className="text-muted-foreground">Auto-refresh every 15s</p>
              <Button variant="outline" onClick={() => void loadStatus()}>
                Refresh
              </Button>
            </CardFooter>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Audit History</CardTitle>
              <CardDescription>
                Recent moderation and server-control events.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {auditLoading ? (
                <div className="flex flex-col gap-2">
                  <Skeleton className="h-8" />
                  <Skeleton className="h-8" />
                  <Skeleton className="h-8" />
                </div>
              ) : auditError ? (
                <Badge variant="destructive">{auditError}</Badge>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Time</TableHead>
                      <TableHead>Action</TableHead>
                      <TableHead>Outcome</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {auditLogs.length ? (
                      auditLogs.map((log) => (
                        <TableRow key={log.id}>
                          <TableCell>{formatTimestamp(log.created)}</TableCell>
                          <TableCell>{log.action}</TableCell>
                          <TableCell>
                            <Badge
                              variant={
                                log.outcome === "success"
                                  ? "default"
                                  : log.outcome === "failed"
                                    ? "destructive"
                                    : "outline"
                              }
                            >
                              {log.outcome}
                            </Badge>
                          </TableCell>
                        </TableRow>
                      ))
                    ) : (
                      <TableRow>
                        <TableCell className="text-muted-foreground" colSpan={3}>
                          No audit entries yet.
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              )}
            </CardContent>
            <CardFooter>
              <Button variant="outline" onClick={() => void loadAuditLogs()}>
                Refresh
              </Button>
            </CardFooter>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Action Console</CardTitle>
            <CardDescription>
              Constrained admin actions mapped to vetted command handlers.
            </CardDescription>
          </CardHeader>

          <CardContent className="flex flex-col gap-6">
            <Tabs defaultValue="moderation">
              <TabsList variant="line">
                <TabsTrigger value="moderation">Moderation</TabsTrigger>
                <TabsTrigger value="maintenance">Maintenance</TabsTrigger>
                <TabsTrigger value="messaging">Messaging</TabsTrigger>
              </TabsList>

              <TabsContent className="flex flex-col gap-6" value="moderation">
                <div className="flex flex-col gap-3">
                  <p className="font-medium">Whitelist Management</p>
                  <div className="flex flex-col gap-3 md:flex-row md:items-end">
                    <div className="flex-1">
                      <label className="text-muted-foreground" htmlFor="whitelist-player">
                        Player name
                      </label>
                      <Input
                        id="whitelist-player"
                        placeholder="Steve"
                        value={whitelistPlayer}
                        onChange={(event) => setWhitelistPlayer(event.target.value)}
                      />
                    </div>
                    <div className="flex gap-2">
                      <Button
                        disabled={!canRun("whitelist_add") || pendingAction !== null}
                        onClick={() =>
                          void runAction("whitelist_add", { player: whitelistPlayer })
                        }
                        variant="outline"
                      >
                        Add
                      </Button>
                      <Button
                        disabled={!canRun("whitelist_remove") || pendingAction !== null}
                        onClick={() =>
                          void runAction("whitelist_remove", { player: whitelistPlayer })
                        }
                        variant="outline"
                      >
                        Remove
                      </Button>
                    </div>
                  </div>
                </div>

                <Separator />

                <div className="flex flex-col gap-3">
                  <p className="font-medium">Kick Player</p>
                  <div className="flex flex-col gap-3">
                    <div>
                      <label className="text-muted-foreground" htmlFor="kick-player">
                        Player name
                      </label>
                      <Input
                        id="kick-player"
                        placeholder="Alex"
                        value={kickPlayer}
                        onChange={(event) => setKickPlayer(event.target.value)}
                      />
                    </div>
                    <div>
                      <label className="text-muted-foreground" htmlFor="kick-reason">
                        Reason (optional)
                      </label>
                      <Input
                        id="kick-reason"
                        placeholder="Spamming chat"
                        value={kickReason}
                        onChange={(event) => setKickReason(event.target.value)}
                      />
                    </div>

                    <AlertDialog>
                      <AlertDialogTrigger
                        className="inline-flex"
                        render={
                          <Button
                            disabled={!canRun("kick") || pendingAction !== null}
                            variant="destructive"
                          />
                        }
                      >
                        Kick player
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Confirm kick</AlertDialogTitle>
                          <AlertDialogDescription>
                            This will remove the player from the server immediately.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancel</AlertDialogCancel>
                          <AlertDialogAction
                            onClick={() =>
                              void runAction("kick", {
                                player: kickPlayer,
                                reason: kickReason,
                              })
                            }
                            variant="destructive"
                          >
                            Confirm kick
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                </div>
              </TabsContent>

              <TabsContent className="flex flex-col gap-6" value="maintenance">
                <div className="flex flex-col gap-3">
                  <p className="font-medium">World Save</p>
                  <p className="text-muted-foreground">
                    Trigger an immediate save operation for loaded worlds.
                  </p>
                  <Button
                    disabled={!canRun("save_world") || pendingAction !== null}
                    onClick={() => void runAction("save_world", {})}
                    variant="outline"
                  >
                    Save world
                  </Button>
                </div>

                <Separator />

                <div className="flex flex-col gap-3">
                  <p className="font-medium">Restart Server</p>
                  <p className="text-muted-foreground">
                    Executes a save then stop command. Container restart policy starts
                    the server again.
                  </p>
                  <label className="text-muted-foreground" htmlFor="restart-message">
                    Announcement (optional)
                  </label>
                  <Input
                    id="restart-message"
                    value={restartMessage}
                    onChange={(event) => setRestartMessage(event.target.value)}
                  />

                  <AlertDialog>
                    <AlertDialogTrigger
                      className="inline-flex"
                      render={
                        <Button
                          disabled={!canRun("restart_server") || pendingAction !== null}
                          variant="destructive"
                        />
                      }
                    >
                      Restart server
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Confirm server restart</AlertDialogTitle>
                        <AlertDialogDescription>
                          Connected players will be disconnected while the host restarts
                          the process.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                          onClick={() =>
                            void runAction("restart_server", {
                              message: restartMessage,
                            })
                          }
                          variant="destructive"
                        >
                          Confirm restart
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </TabsContent>

              <TabsContent className="flex flex-col gap-4" value="messaging">
                <label className="text-muted-foreground" htmlFor="broadcast-message">
                  Broadcast message
                </label>
                <Textarea
                  id="broadcast-message"
                  placeholder="Maintenance starts in 10 minutes."
                  value={serverMessage}
                  onChange={(event) => setServerMessage(event.target.value)}
                />
                <Button
                  disabled={!canRun("say") || pendingAction !== null}
                  onClick={() => void runAction("say", { message: serverMessage })}
                >
                  Send message
                </Button>
              </TabsContent>
            </Tabs>
          </CardContent>

          <CardFooter>
            {pendingAction ? (
              <Badge variant="outline">Executing: {pendingAction}</Badge>
            ) : (
              <Badge variant="secondary">Ready</Badge>
            )}
          </CardFooter>
        </Card>
      </div>
    </div>
  )
}

export default App
