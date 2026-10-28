"use client"

import * as React from "react"

import { streamSSE } from "@/lib/sse"
import { cn } from "@/lib/utils"

/** Terminal-style output panel that sticks to the bottom while new lines arrive. */
export function LogTerminal({
  lines,
  className,
  empty = "Waiting for output…",
}: {
  lines: string[]
  className?: string
  empty?: string
}) {
  const ref = React.useRef<HTMLDivElement>(null)
  const stick = React.useRef(true)

  React.useEffect(() => {
    const el = ref.current
    if (el && stick.current) el.scrollTop = el.scrollHeight
  }, [lines])

  return (
    <div
      ref={ref}
      onScroll={(e) => {
        const el = e.currentTarget
        stick.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24
      }}
      className={cn(
        "overflow-auto rounded-lg border border-zinc-800 bg-zinc-950 p-3 font-mono text-xs leading-relaxed text-zinc-200",
        className
      )}
    >
      {lines.length === 0 ? (
        <span className="text-zinc-500">{empty}</span>
      ) : (
        lines.map((l, i) => (
          <div
            key={i}
            className={cn(
              "break-all whitespace-pre-wrap",
              l.startsWith("$ ") && "text-sky-400",
              /\berror\b/i.test(l) && "text-red-400"
            )}
          >
            {l || " "}
          </div>
        ))
      )}
    </div>
  )
}

/** Subscribe to an SSE line stream; restarts when url changes. */
export function useLineStream(url: string | null, max = 5000) {
  const [lines, setLines] = React.useState<string[]>([])
  const [exitCode, setExitCode] = React.useState<number | null>(null)
  const [error, setError] = React.useState<string | null>(null)
  const [prevUrl, setPrevUrl] = React.useState(url)

  // Reset output when switching streams (state adjusted during render, not in an effect).
  if (url !== prevUrl) {
    setPrevUrl(url)
    setLines([])
    setExitCode(null)
    setError(null)
  }

  React.useEffect(() => {
    if (!url) return
    const ctrl = new AbortController()
    let buffer: string[] = []
    let frame = 0
    const flush = () => {
      frame = 0
      const chunk = buffer
      buffer = []
      setLines((prev) => {
        const next = prev.concat(chunk)
        return next.length > max ? next.slice(next.length - max) : next
      })
    }
    streamSSE(
      url,
      (e) => {
        if (e.event === "line") {
          buffer.push(JSON.parse(e.data))
          if (!frame) frame = requestAnimationFrame(flush)
        } else if (e.event === "done") {
          flush()
          setExitCode(JSON.parse(e.data).exitCode)
        } else if (e.event === "error") {
          setError(JSON.parse(e.data))
        }
      },
      ctrl.signal
    ).catch((err) => {
      if (!ctrl.signal.aborted) setError(String(err.message ?? err))
    })
    return () => {
      ctrl.abort()
      if (frame) cancelAnimationFrame(frame)
    }
  }, [url, max])

  return { lines, exitCode, error }
}
