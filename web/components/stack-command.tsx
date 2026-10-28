"use client"

import * as React from "react"
import { useRouter } from "next/navigation"
import { PlusIcon } from "lucide-react"

import { StatusDot } from "@/components/status-dot"
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command"
import { useStacks } from "@/lib/queries"

export function StackCommand({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const router = useRouter()
  const { data: stacks } = useStacks()

  React.useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key.toLowerCase() === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        onOpenChange(!open)
      }
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [open, onOpenChange])

  function go(href: string) {
    onOpenChange(false)
    router.push(href)
  }

  return (
    <CommandDialog open={open} onOpenChange={onOpenChange} title="Jump to stack">
      <CommandInput placeholder="Search stacks…" />
      <CommandList>
        <CommandEmpty>No stacks found.</CommandEmpty>
        <CommandGroup heading="Stacks">
          {(stacks ?? []).map((s) => (
            <CommandItem key={s.name} value={s.name} onSelect={() => go(`/stacks/${s.name}`)}>
              <StatusDot status={s.status} />
              {s.name}
            </CommandItem>
          ))}
        </CommandGroup>
        <CommandGroup heading="Actions">
          <CommandItem value="new stack create" onSelect={() => go("/stacks/new")}>
            <PlusIcon />
            New stack
          </CommandItem>
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  )
}
