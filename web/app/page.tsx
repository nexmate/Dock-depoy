"use client"

import * as React from "react"
import { useRouter } from "next/navigation"

import { FullPageSpinner } from "@/components/full-page-spinner"
import { useAuthStatus } from "@/lib/queries"

export default function Home() {
  const router = useRouter()
  const { data } = useAuthStatus()
  React.useEffect(() => {
    if (!data) return
    router.replace(!data.setupDone ? "/setup" : data.authenticated ? "/stacks" : "/login")
  }, [data, router])
  return <FullPageSpinner />
}
