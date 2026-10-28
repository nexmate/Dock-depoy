"use client"

import { AuthForm } from "@/components/auth-form"
import { api } from "@/lib/api"

export default function SetupPage() {
  return <AuthForm mode="setup" onSubmit={api.setup} />
}
