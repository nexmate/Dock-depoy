"use client"

import * as React from "react"
import Link from "next/link"
import { useParams, useRouter, useSearchParams } from "next/navigation"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import {
  ArrowDownToLineIcon,
  CircleStopIcon,
  DownloadCloudIcon,
  Loader2Icon,
  MoreHorizontalIcon,
  RefreshCwIcon,
  RocketIcon,
  SaveIcon,
  Trash2Icon,
} from "lucide-react"
import { toast } from "sonner"

import { ActionSheet } from "@/components/action-sheet"
import { DeleteDialog, DownDialog } from "@/components/confirm-dialogs"
import { ContainersTab } from "@/components/containers-tab"
import { LogsView } from "@/components/logs-view"
import { StackFilesEditor } from "@/components/stack-files-editor"
import { StatusBadge } from "@/components/status-dot"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { api, ApiError } from "@/lib/api"
import { keys, useStacks } from "@/lib/queries"
import { useStackAction } from "@/lib/use-stack-action"

export default function StackPage() {
  return (
    <React.Suspense>
      <StackDetail />
    </React.Suspense>
  )
}

function StackDetail() {
  const { name } = useParams<{ name: string }>()
  const router = useRouter()
  const search = useSearchParams()
  const qc = useQueryClient()
  const { data: stack, error, isLoading } = useQuery({
    queryKey: keys.stack(name),
    queryFn: () => api.getStack(name),
  })
  const { data: stacks } = useStacks()
  const status = stacks?.find((s) => s.name === name)?.status ?? stack?.status ?? "inactive"

  const [compose, setCompose] = React.useState("")
  const [env, setEnv] = React.useState("")
  const [saved, setSaved] = React.useState<{ compose: string; env: string } | null>(null)
  const [saveError, setSaveError] = React.useState<string | null>(null)
  const [saving, setSaving] = React.useState(false)
  const [tab, setTab] = React.useState("compose")
  const [logContainer, setLogContainer] = React.useState<string | null>(null)
  const [downOpen, setDownOpen] = React.useState(false)
  const [deleteOpen, setDeleteOpen] = React.useState(false)
  const action = useStackAction()
  const dirty = saved !== null && (compose !== saved.compose || env !== saved.env)

  // Load files into the editor once per stack; refetches never overwrite edits.
  if (stack && saved === null) {
    setCompose(stack.compose)
    setEnv(stack.env)
    setSaved({ compose: stack.compose, env: stack.env })
  }

  // Arriving from "Create & deploy".
  const autoDeployed = React.useRef(false)
  React.useEffect(() => {
    if (search.get("deploy") === "1" && stack && !autoDeployed.current) {
      autoDeployed.current = true
      router.replace(`/stacks/${name}`)
      action.run(name, "up")
    }
  }, [search, stack, name, router, action])

  const save = React.useCallback(async () => {
    setSaving(true)
    setSaveError(null)
    try {
      await api.saveStack(name, compose, env)
      setSaved({ compose, env })
      qc.setQueryData(keys.stack(name), (old: typeof stack) => (old ? { ...old, compose, env } : old))
      toast.success("Saved")
      return true
    } catch (err) {
      setSaveError(err instanceof ApiError ? [err.message, err.detail].filter(Boolean).join("\n") : String(err))
      setTab("compose")
      return false
    } finally {
      setSaving(false)
    }
  }, [name, compose, env, qc])

  React.useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key.toLowerCase() === "s" && (e.ctrlKey || e.metaKey)) {
        e.preventDefault()
        if (dirty && !saving) save()
      }
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [dirty, saving, save])

  async function remove() {
    try {
      await api.deleteStack(name)
      await qc.invalidateQueries({ queryKey: keys.stacks })
      toast.success(`Stack ${name} deleted`)
      router.push("/stacks")
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Delete failed")
    }
  }

  if (error instanceof ApiError && error.status === 404)
    return (
      <div className="flex flex-1 items-center justify-center p-8 text-center">
        <div>
          <h1 className="text-lg font-semibold">Stack not found</h1>
          <p className="mt-1 text-sm text-muted-foreground">There is no stack named {name}.</p>
          <Button variant="outline" className="mt-4" nativeButton={false} render={<Link href="/stacks" />}>
            Back to stacks
          </Button>
        </div>
      </div>
    )

  const busy = !!action.job
  const run = (a: Parameters<typeof action.run>[1]) => action.run(name, a)

  return (
    <div className="mx-auto flex w-full max-w-6xl min-h-0 flex-1 flex-col gap-5 p-4 md:p-6">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-3">
        <div className="flex min-w-0 items-center gap-3">
          <h1 className="truncate font-mono text-xl font-semibold tracking-tight">{name}</h1>
          <StatusBadge status={status} />
        </div>
        <div className="ml-auto flex flex-wrap items-center gap-2">
          <Button onClick={() => run("up")} disabled={busy}>
            <RocketIcon />
            Deploy
          </Button>
          <Button variant="outline" onClick={() => run("restart")} disabled={busy || status === "inactive"}>
            <RefreshCwIcon />
            Restart
          </Button>
          <Button variant="outline" onClick={() => run("stop")} disabled={busy || status === "inactive" || status === "stopped"}>
            <CircleStopIcon />
            Stop
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger render={<Button variant="outline" size="icon" aria-label="More actions" />}>
              <MoreHorizontalIcon />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-48">
              <DropdownMenuItem onClick={() => run("pull")} disabled={busy}>
                <DownloadCloudIcon />
                Pull &amp; redeploy
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setDownOpen(true)} disabled={busy || status === "inactive"}>
                <ArrowDownToLineIcon />
                Down
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem variant="destructive" onClick={() => setDeleteOpen(true)}>
                <Trash2Icon />
                Delete stack
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <Tabs value={tab} onValueChange={(v) => setTab(String(v))} className="min-h-0 flex-1">
        <TabsList>
          <TabsTrigger value="compose">
            Compose
            {dirty && <span className="size-1.5 rounded-full bg-sidebar-primary" aria-label="unsaved changes" />}
          </TabsTrigger>
          <TabsTrigger value="containers">Containers</TabsTrigger>
          <TabsTrigger value="logs">Logs</TabsTrigger>
        </TabsList>

        <TabsContent value="compose" className="grid gap-4 pt-2">
          {isLoading || saved === null ? (
            <Skeleton className="h-96 rounded-lg" />
          ) : (
            <>
              <StackFilesEditor
                compose={compose}
                env={env}
                onComposeChange={setCompose}
                onEnvChange={setEnv}
                error={saveError}
              />
              <div className="flex items-center justify-end gap-2">
                {dirty && <span className="mr-auto text-xs text-muted-foreground">Unsaved changes · Ctrl+S to save</span>}
                <Button variant="outline" disabled={!dirty || saving} onClick={save}>
                  {saving ? <Loader2Icon className="animate-spin" /> : <SaveIcon />}
                  Save
                </Button>
                <Button
                  disabled={!dirty || saving || busy}
                  onClick={async () => {
                    if (await save()) run("up")
                  }}
                >
                  <RocketIcon />
                  Save &amp; deploy
                </Button>
              </div>
            </>
          )}
        </TabsContent>

        <TabsContent value="containers" className="pt-2">
          <ContainersTab
            stack={name}
            onShowLogs={(c) => {
              setLogContainer(c.id)
              setTab("logs")
            }}
          />
        </TabsContent>

        <TabsContent value="logs" className="flex min-h-0 flex-col pt-2">
          {tab === "logs" && <LogsView stack={name} selected={logContainer} onSelect={setLogContainer} />}
        </TabsContent>
      </Tabs>

      <ActionSheet job={action.job} onClose={action.close} />
      <DownDialog open={downOpen} onOpenChange={setDownOpen} stack={name} onConfirm={() => run("down")} />
      <DeleteDialog open={deleteOpen} onOpenChange={setDeleteOpen} stack={name} onConfirm={remove} />
    </div>
  )
}
