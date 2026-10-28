"use client"

import { AuthForm } from "@/components/auth-form"
import { api } from "@/lib/api"

export default function LoginPage() {
  return <AuthForm mode="login" onSubmit={api.login} />
}
