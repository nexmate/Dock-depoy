"use client"

import { ScrollTextIcon } from "lucide-react"

import { useContainers } from "@/components/containers-tab"
import { LogTerminal, useLineStream } from "@/components/log-terminal"
import { Button } from "@/components/ui/button"
import { logsStreamUrl } from "@/lib/api"

export function LogsView({
  stack,
  selected,
  onSelect,
}: {
  stack: string
  selected: string | null
  onSelect: (id: string) => void
}) {
  const { data: containers } = useContainers(stack)
  const current = containers?.find((c) => c.id === selected) ?? containers?.[0]
  const { lines, error } = useLineStream(current ? logsStreamUrl(current.id) : null)

  if (containers && containers.length === 0)
    return (
      <div className="rounded-xl border border-dashed p-10 text-center">
        <ScrollTextIcon className="mx-auto mb-3 size-6 text-muted-foreground" />
        <p className="text-sm text-muted-foreground">No containers to show logs for.</p>
      </div>
    )

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      <div className="flex flex-wrap gap-1.5">
        {containers?.map((c) => (
          <Button
            key={c.id}
            size="sm"
            variant={c.id === current?.id ? "secondary" : "ghost"}
            onClick={() => onSelect(c.id)}
            className="font-mono text-xs"
          >
            {c.service || c.name}
          </Button>
        ))}
      </div>
      <LogTerminal
        lines={error ? [...lines, `error: ${error}`] : lines}
        empty={current ? "No log output yet…" : "Loading…"}
        className="h-[60svh]"
      />
    </div>
  )
}
