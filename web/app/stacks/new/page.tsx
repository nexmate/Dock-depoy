"use client"

import * as React from "react"
import { useRouter } from "next/navigation"
import { useQueryClient } from "@tanstack/react-query"
import { Loader2Icon, RocketIcon } from "lucide-react"
import { toast } from "sonner"

import { StackFilesEditor, STARTER_COMPOSE } from "@/components/stack-files-editor"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { api, ApiError } from "@/lib/api"
import { keys } from "@/lib/queries"

const NAME_RE = /^[a-z0-9][a-z0-9_-]*$/

export default function NewStackPage() {
  const router = useRouter()
  const qc = useQueryClient()
  const [name, setName] = React.useState("")
  const [compose, setCompose] = React.useState(STARTER_COMPOSE)
  const [env, setEnv] = React.useState("")
  const [error, setError] = React.useState<string | null>(null)
  const [pending, setPending] = React.useState<null | "create" | "deploy">(null)
  const nameValid = NAME_RE.test(name) && name.length <= 64

  async function create(deploy: boolean) {
    setPending(deploy ? "deploy" : "create")
    setError(null)
    try {
      await api.createStack(name, compose, env)
      await qc.invalidateQueries({ queryKey: keys.stacks })
      toast.success(`Stack ${name} created`)
      router.push(`/stacks/${name}${deploy ? "?deploy=1" : ""}`)
    } catch (err) {
      setError(err instanceof ApiError ? [err.message, err.detail].filter(Boolean).join("\n") : String(err))
      setPending(null)
    }
  }

  return (
    <div className="mx-auto grid w-full max-w-5xl gap-6 p-4 md:p-6">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">New stack</h1>
        <p className="text-sm text-muted-foreground">
          A stack is a folder with a compose file. The name becomes the compose project name.
        </p>
      </div>
      <div className="grid max-w-sm gap-2">
        <Label htmlFor="name">Stack name</Label>
        <Input
          id="name"
          value={name}
          onChange={(e) => setName(e.target.value.toLowerCase())}
          placeholder="my-app"
          className="font-mono"
          autoFocus
          aria-invalid={name !== "" && !nameValid}
        />
        <p className="text-xs text-muted-foreground">
          Lowercase letters, digits, dashes and underscores.
        </p>
      </div>
      <StackFilesEditor
        compose={compose}
        env={env}
        onComposeChange={setCompose}
        onEnvChange={setEnv}
        error={error}
      />
      <div className="flex justify-end gap-2">
        <Button variant="outline" disabled={!nameValid || !!pending} onClick={() => create(false)}>
          {pending === "create" && <Loader2Icon className="animate-spin" />}
          Create
        </Button>
        <Button disabled={!nameValid || !!pending} onClick={() => create(true)}>
          {pending === "deploy" ? <Loader2Icon className="animate-spin" /> : <RocketIcon />}
          Create &amp; deploy
        </Button>
      </div>
    </div>
  )
}
