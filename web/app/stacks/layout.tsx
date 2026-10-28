"use client"

import * as React from "react"
import { useRouter } from "next/navigation"
import { SearchIcon } from "lucide-react"

import { AppSidebar } from "@/components/app-sidebar"
import { FullPageSpinner } from "@/components/full-page-spinner"
import { StackCommand } from "@/components/stack-command"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar"
import { useAuthStatus } from "@/lib/queries"

export default function StacksLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const { data: auth } = useAuthStatus()
  const [commandOpen, setCommandOpen] = React.useState(false)
  const allowed = auth?.setupDone && auth.authenticated

  React.useEffect(() => {
    if (auth && !allowed) router.replace(auth.setupDone ? "/login" : "/setup")
  }, [auth, allowed, router])

  if (!allowed) return <FullPageSpinner />

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset className="min-w-0">
        <header className="flex h-12 shrink-0 items-center gap-2 border-b px-3">
          <SidebarTrigger />
          <Separator orientation="vertical" className="mx-1 h-4" />
          <Button
            variant="outline"
            size="sm"
            onClick={() => setCommandOpen(true)}
            className="ml-auto w-48 justify-start gap-2 text-muted-foreground sm:w-64"
          >
            <SearchIcon />
            <span className="flex-1 text-left">Jump to stack…</span>
            <kbd className="pointer-events-none rounded border bg-muted px-1.5 font-mono text-[10px]">
              Ctrl K
            </kbd>
          </Button>
        </header>
        <div className="flex min-h-0 flex-1 flex-col">{children}</div>
      </SidebarInset>
      <StackCommand open={commandOpen} onOpenChange={setCommandOpen} />
    </SidebarProvider>
  )
}
