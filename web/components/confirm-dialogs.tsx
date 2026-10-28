"use client"

import * as React from "react"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Input } from "@/components/ui/input"

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  stack: string
  onConfirm: () => void
}

export function DownDialog({ open, onOpenChange, stack, onConfirm }: Props) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Take {stack} down?</AlertDialogTitle>
          <AlertDialogDescription>
            Stops and removes the stack&apos;s containers and networks. Named volumes and the compose
            file are kept.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={() => {
              onOpenChange(false)
              onConfirm()
            }}
          >
            Down
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

export function DeleteDialog({ open, onOpenChange, stack, onConfirm }: Props) {
  const [typed, setTyped] = React.useState("")
  const change = (next: boolean) => {
    if (!next) setTyped("")
    onOpenChange(next)
  }
  return (
    <AlertDialog open={open} onOpenChange={change}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete {stack}?</AlertDialogTitle>
          <AlertDialogDescription>
            Runs <code className="font-mono">docker compose down</code> and permanently deletes the
            stack folder, including its compose file and .env. Type <b>{stack}</b> to confirm.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <Input
          value={typed}
          onChange={(e) => setTyped(e.target.value)}
          placeholder={stack}
          aria-label="Stack name"
          autoFocus
        />
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={typed !== stack}
            onClick={() => {
              change(false)
              onConfirm()
            }}
          >
            Delete stack
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
