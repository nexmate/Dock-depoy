"use client"

import * as React from "react"
import { usePathname, useRouter } from "next/navigation"
import { QueryClient, QueryClientProvider, useQueryClient } from "@tanstack/react-query"

import { Toaster } from "@/components/ui/sonner"
import { TooltipProvider } from "@/components/ui/tooltip"
import { ApiError } from "@/lib/api"

const isUnauthorized = (e: unknown) => e instanceof ApiError && e.status === 401

/** Sends the user to /login whenever any query or mutation gets a 401. */
function UnauthorizedRedirect() {
  const qc = useQueryClient()
  const router = useRouter()
  const pathname = usePathname()
  React.useEffect(() => {
    const go = (err: unknown) => {
      if (isUnauthorized(err) && !pathname.startsWith("/login")) router.replace("/login")
    }
    const unsubQ = qc.getQueryCache().subscribe((e) => {
      if (e.type === "updated" && e.action.type === "error") go(e.action.error)
    })
    const unsubM = qc.getMutationCache().subscribe((e) => {
      if (e.type === "updated" && e.action.type === "error") go(e.action.error)
    })
    return () => {
      unsubQ()
      unsubM()
    }
  }, [qc, router, pathname])
  return null
}

export function Providers({ children }: { children: React.ReactNode }) {
  const [client] = React.useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            retry: (count, err) => !(err instanceof ApiError && err.status < 500) && count < 2,
          },
        },
      })
  )
  return (
    <QueryClientProvider client={client}>
      <UnauthorizedRedirect />
      <TooltipProvider>{children}</TooltipProvider>
      <Toaster richColors position="bottom-right" />
    </QueryClientProvider>
  )
}
