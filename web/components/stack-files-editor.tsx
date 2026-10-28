"use client"

import * as React from "react"
import dynamic from "next/dynamic"
import { AlertTriangleIcon, ChevronRightIcon } from "lucide-react"

import { Skeleton } from "@/components/ui/skeleton"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"

// CodeMirror touches the DOM on import, so keep it out of prerendering.
const CodeEditor = dynamic(() => import("@/components/code-editor").then((m) => m.CodeEditor), {
  ssr: false,
  loading: () => <Skeleton className="h-80 rounded-lg" />,
})

export const STARTER_COMPOSE = `services:
  app:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "8080:80"
`

/** compose.yaml editor + collapsible .env editor + validation error panel. */
export function StackFilesEditor({
  compose,
  env,
  onComposeChange,
  onEnvChange,
  error,
}: {
  compose: string
  env: string
  onComposeChange: (v: string) => void
  onEnvChange: (v: string) => void
  error?: string | null
}) {
  const [envOpen, setEnvOpen] = React.useState(env !== "")

  return (
    <div className="grid gap-4">
      {error && (
        <div className="flex gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
          <AlertTriangleIcon className="mt-0.5 size-4 shrink-0" />
          <pre className="min-w-0 font-mono text-xs break-words whitespace-pre-wrap">{error}</pre>
        </div>
      )}
      <div className="grid gap-2">
        <p className="font-mono text-xs text-muted-foreground">compose.yaml</p>
        <CodeEditor value={compose} onChange={onComposeChange} language="yaml" minHeight="360px" aria-label="compose.yaml" />
      </div>
      <Collapsible open={envOpen} onOpenChange={setEnvOpen} className="grid gap-2">
        <CollapsibleTrigger className="flex w-fit items-center gap-1 font-mono text-xs text-muted-foreground hover:text-foreground">
          <ChevronRightIcon className={`size-3.5 transition-transform ${envOpen ? "rotate-90" : ""}`} />
          .env
          {!envOpen && env === "" && <span className="font-sans">(optional)</span>}
        </CollapsibleTrigger>
        <CollapsibleContent>
          <CodeEditor value={env} onChange={onEnvChange} language="env" minHeight="120px" aria-label=".env" />
        </CollapsibleContent>
      </Collapsible>
    </div>
  )
}
