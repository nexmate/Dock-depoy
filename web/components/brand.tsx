import { ContainerIcon } from "lucide-react"

import { cn } from "@/lib/utils"

export function Brand({ className }: { className?: string }) {
  return (
    <div className={cn("flex items-center gap-2.5", className)}>
      <div className="flex size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground shadow-sm">
        <ContainerIcon className="size-4" />
      </div>
      <div className="grid leading-tight">
        <span className="text-sm font-semibold tracking-tight">Docker-depoy</span>
        <span className="text-xs text-muted-foreground">Compose stacks</span>
      </div>
    </div>
  )
}
