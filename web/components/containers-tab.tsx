"use client"

import { useQuery } from "@tanstack/react-query"
import { BoxIcon, ExternalLinkIcon, ScrollTextIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { api, type ContainerInfo } from "@/lib/api"
import { keys, POLL_MS } from "@/lib/queries"
import { cn } from "@/lib/utils"

export function useContainers(stack: string) {
  return useQuery({
    queryKey: keys.containers(stack),
    queryFn: () => api.containers(stack),
    refetchInterval: POLL_MS,
  })
}

function stateClass(state: string) {
  if (state === "running") return "border-emerald-500/40 text-emerald-600 dark:text-emerald-400"
  if (state === "restarting" || state === "paused") return "border-amber-500/40 text-amber-600 dark:text-amber-400"
  if (state === "dead") return "border-destructive/40 text-destructive"
  return "text-muted-foreground"
}

export function ContainersTab({
  stack,
  onShowLogs,
}: {
  stack: string
  onShowLogs: (c: ContainerInfo) => void
}) {
  const { data, isLoading, error } = useContainers(stack)

  if (isLoading)
    return (
      <div className="grid gap-3 md:grid-cols-2">
        {[0, 1].map((i) => (
          <Skeleton key={i} className="h-28 rounded-xl" />
        ))}
      </div>
    )
  if (error) return <p className="text-sm text-destructive">{error.message}</p>
  if (!data?.length)
    return (
      <div className="rounded-xl border border-dashed p-10 text-center">
        <BoxIcon className="mx-auto mb-3 size-6 text-muted-foreground" />
        <p className="text-sm font-medium">No containers</p>
        <p className="mt-1 text-sm text-muted-foreground">Deploy the stack to create its containers.</p>
      </div>
    )

  const host = typeof window === "undefined" ? "localhost" : window.location.hostname
  return (
    <div className="grid gap-3 md:grid-cols-2">
      {data.map((c) => (
        <Card key={c.id} size="sm" className="gap-0">
          <CardContent className="grid gap-3">
            <div className="flex items-start gap-3">
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium">{c.service || c.name}</p>
                <p className="truncate font-mono text-xs text-muted-foreground">{c.name}</p>
              </div>
              <Badge variant="outline" className={cn("capitalize", stateClass(c.state))}>
                {c.state}
              </Badge>
            </div>
            <div className="grid gap-1 text-xs">
              <p className="truncate">
                <span className="text-muted-foreground">Image </span>
                <span className="font-mono">{c.image}</span>
              </p>
              <p className="truncate text-muted-foreground">{c.status}</p>
            </div>
            <div className="flex flex-wrap items-center gap-1.5">
              {c.ports
                .filter((p) => p.public)
                .map((p) => (
                  <a
                    key={`${p.public}/${p.proto}`}
                    href={`http://${host}:${p.public}`}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-1 rounded-md border bg-muted/50 px-1.5 py-0.5 font-mono text-xs hover:bg-muted"
                  >
                    {p.public}→{p.private}
                    <ExternalLinkIcon className="size-3" />
                  </a>
                ))}
              <Button variant="ghost" size="sm" className="ml-auto" onClick={() => onShowLogs(c)}>
                <ScrollTextIcon />
                Logs
              </Button>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
