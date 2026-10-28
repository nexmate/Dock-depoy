import { Loader2Icon } from "lucide-react"

export function FullPageSpinner() {
  return (
    <div className="flex min-h-svh items-center justify-center">
      <Loader2Icon className="size-5 animate-spin text-muted-foreground" />
    </div>
  )
}
