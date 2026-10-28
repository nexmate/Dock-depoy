"use client"

import * as React from "react"
import { toast } from "sonner"

import type { RunningJob } from "@/components/action-sheet"
import { api, ApiError, type StackAction } from "@/lib/api"

/** Starts stack actions and tracks the job shown in the ActionSheet. */
export function useStackAction() {
  const [job, setJob] = React.useState<RunningJob | null>(null)
  const run = React.useCallback(async (stack: string, action: StackAction) => {
    try {
      const { jobId } = await api.action(stack, action)
      setJob({ stack, action, jobId })
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Could not start action")
    }
  }, [])
  return { job, run, close: () => setJob(null) }
}
