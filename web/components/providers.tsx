"use client"

import * as React from "react"
import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query"

import { Toaster } from "@/components/ui/sonner"
import { TooltipProvider } from "@/components/ui/tooltip"
import { ApiError } from "@/lib/api"

function onError(error: unknown) {
  if (error instanceof ApiError && error.status === 401 && !location.pathname.startsWith("/login")) {
    location.assign("/login")
  }
}

export function Providers({ children }: { children: React.ReactNode }) {
  const [client] = React.useState(
    () =>
      new QueryClient({
        queryCache: new QueryCache({ onError }),
        mutationCache: new MutationCache({ onError }),
        defaultOptions: {
          queries: {
            retry: (count, err) => !(err instanceof ApiError && err.status < 500) && count < 2,
            refetchOnWindowFocus: true,
          },
        },
      })
  )
  return (
    <QueryClientProvider client={client}>
      <TooltipProvider>{children}</TooltipProvider>
      <Toaster richColors position="bottom-right" />
    </QueryClientProvider>
  )
}
