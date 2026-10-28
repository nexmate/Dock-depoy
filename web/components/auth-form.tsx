"use client"

import * as React from "react"
import { useRouter } from "next/navigation"
import { useQueryClient } from "@tanstack/react-query"
import { Loader2Icon } from "lucide-react"

import { Brand } from "@/components/brand"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ApiError } from "@/lib/api"

type Props = {
  mode: "login" | "setup"
  onSubmit: (username: string, password: string) => Promise<void>
}

export function AuthForm({ mode, onSubmit }: Props) {
  const router = useRouter()
  const qc = useQueryClient()
  const [error, setError] = React.useState<string | null>(null)
  const [pending, setPending] = React.useState(false)
  const setup = mode === "setup"

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const username = String(form.get("username") ?? "").trim()
    const password = String(form.get("password") ?? "")
    if (setup && password !== form.get("confirm")) {
      setError("Passwords do not match")
      return
    }
    setPending(true)
    setError(null)
    try {
      await onSubmit(username, password)
      await qc.invalidateQueries({ queryKey: ["auth"] })
      router.replace("/stacks")
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong")
      setPending(false)
    }
  }

  return (
    <div className="flex min-h-svh items-center justify-center bg-muted/40 p-4">
      <div className="w-full max-w-sm space-y-6">
        <Brand className="justify-center" />
        <Card>
          <CardHeader>
            <CardTitle>{setup ? "Create admin account" : "Welcome back"}</CardTitle>
            <CardDescription>
              {setup
                ? "First run — choose the credentials you will sign in with."
                : "Sign in to manage your stacks."}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="grid gap-4">
              <div className="grid gap-2">
                <Label htmlFor="username">Username</Label>
                <Input id="username" name="username" autoComplete="username" required autoFocus />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="password">Password</Label>
                <Input
                  id="password"
                  name="password"
                  type="password"
                  autoComplete={setup ? "new-password" : "current-password"}
                  minLength={setup ? 8 : undefined}
                  required
                />
              </div>
              {setup && (
                <div className="grid gap-2">
                  <Label htmlFor="confirm">Confirm password</Label>
                  <Input id="confirm" name="confirm" type="password" autoComplete="new-password" required />
                </div>
              )}
              {error && <p className="text-sm text-destructive">{error}</p>}
              <Button type="submit" disabled={pending} className="w-full">
                {pending && <Loader2Icon className="animate-spin" />}
                {setup ? "Create account" : "Sign in"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
