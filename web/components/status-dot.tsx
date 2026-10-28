import { Badge } from "@/components/ui/badge"
import type { StackStatus } from "@/lib/api"
import { cn } from "@/lib/utils"

const dot: Record<StackStatus, string> = {
  running: "bg-emerald-500",
  partial: "bg-amber-500",
  stopped: "bg-muted-foreground/40",
  inactive: "border border-muted-foreground/50 bg-transparent",
}

export const statusLabel: Record<StackStatus, string> = {
  running: "Running",
  partial: "Partially running",
  stopped: "Stopped",
  inactive: "Not deployed",
}

export function StatusDot({ status, className }: { status: StackStatus; className?: string }) {
  return (
    <span className={cn("relative flex size-2 shrink-0", className)} aria-label={statusLabel[status]}>
      {status === "running" && (
        <span className="absolute inline-flex size-full animate-ping rounded-full bg-emerald-500 opacity-40" />
      )}
      <span className={cn("relative inline-flex size-2 rounded-full", dot[status])} />
    </span>
  )
}

export function StatusBadge({ status }: { status: StackStatus }) {
  return (
    <Badge variant="outline" className="gap-1.5 font-normal">
      <StatusDot status={status} />
      {statusLabel[status]}
    </Badge>
  )
}
