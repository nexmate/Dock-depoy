import Link from "next/link"
import { LayersIcon, PlusIcon } from "lucide-react"

import { Button } from "@/components/ui/button"

export default function StacksIndex() {
  return (
    <div className="flex flex-1 items-center justify-center p-8">
      <div className="max-w-sm text-center">
        <div className="mx-auto mb-4 flex size-12 items-center justify-center rounded-xl border bg-muted/50">
          <LayersIcon className="size-5 text-muted-foreground" />
        </div>
        <h1 className="text-lg font-semibold tracking-tight">Select a stack</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Pick a stack from the sidebar, or create a new one from a compose file.
        </p>
        <Button className="mt-5" nativeButton={false} render={<Link href="/stacks/new" />}>
          <PlusIcon />
          New stack
        </Button>
      </div>
    </div>
  )
}
