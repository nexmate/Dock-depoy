"use client"

import * as React from "react"
import { useQueryClient } from "@tanstack/react-query"
import { CheckCircle2Icon, Loader2Icon, XCircleIcon } from "lucide-react"
import { toast } from "sonner"

import { LogTerminal, useLineStream } from "@/components/log-terminal"
import { Badge } from "@/components/ui/badge"
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { jobStreamUrl, type StackAction } from "@/lib/api"
import { keys } from "@/lib/queries"

export type RunningJob = { stack: string; action: StackAction; jobId: string }

export const actionLabel: Record<StackAction, string> = {
  up: "Deploy",
  stop: "Stop",
  restart: "Restart",
  pull: "Update",
  down: "Down",
}

export function ActionSheet({ job, onClose }: { job: RunningJob | null; onClose: () => void }) {
  const qc = useQueryClient()
  const { lines, exitCode, error } = useLineStream(job ? jobStreamUrl(job.jobId) : null)
  const finished = exitCode !== null

  React.useEffect(() => {
    if (!job || exitCode === null) return
    const label = `${actionLabel[job.action]} ${job.stack}`
    if (exitCode === 0) toast.success(`${label} finished`)
    else toast.error(`${label} failed`, { description: `Exit code ${exitCode}` })
    qc.invalidateQueries({ queryKey: keys.stacks })
    qc.invalidateQueries({ queryKey: keys.stack(job.stack) })
    qc.invalidateQueries({ queryKey: keys.containers(job.stack) })
  }, [job, exitCode, qc])

  return (
    <Sheet open={!!job} onOpenChange={(open) => !open && onClose()}>
      <SheetContent side="bottom" className="mx-auto max-h-[70svh] max-w-5xl gap-0 rounded-t-xl">
        <SheetHeader className="flex-row items-center gap-3 pr-12">
          <div className="grid gap-0.5">
            <SheetTitle>
              {job ? `${actionLabel[job.action]} · ${job.stack}` : "Action"}
            </SheetTitle>
            <SheetDescription>Live output from docker compose</SheetDescription>
          </div>
          <div className="ml-auto">
            {!finished && !error && (
              <Badge variant="outline" className="gap-1.5">
                <Loader2Icon className="size-3 animate-spin" /> Running
              </Badge>
            )}
            {finished && exitCode === 0 && (
              <Badge variant="outline" className="gap-1.5 border-emerald-500/40 text-emerald-600 dark:text-emerald-400">
                <CheckCircle2Icon className="size-3" /> Succeeded
              </Badge>
            )}
            {((finished && exitCode !== 0) || error) && (
              <Badge variant="destructive" className="gap-1.5">
                <XCircleIcon className="size-3" /> {error ? "Stream error" : `Failed (${exitCode})`}
              </Badge>
            )}
          </div>
        </SheetHeader>
        <div className="px-4 pb-4">
          <LogTerminal lines={error ? [...lines, `error: ${error}`] : lines} className="h-[45svh]" />
        </div>
      </SheetContent>
    </Sheet>
  )
}
