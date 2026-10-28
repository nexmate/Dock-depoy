"use client"

import * as React from "react"
import Link from "next/link"
import { usePathname, useRouter } from "next/navigation"
import { AlertTriangleIcon, LogOutIcon, PlusIcon, SearchIcon } from "lucide-react"

import { Brand } from "@/components/brand"
import { StatusDot, statusLabel } from "@/components/status-dot"
import { ThemeToggle } from "@/components/theme-toggle"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInput,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSkeleton,
} from "@/components/ui/sidebar"
import { api } from "@/lib/api"
import { useStacks } from "@/lib/queries"

export function AppSidebar() {
  const pathname = usePathname()
  const router = useRouter()
  const [filter, setFilter] = React.useState("")
  const { data: stacks, isLoading, error } = useStacks()
  const visible = (stacks ?? []).filter((s) => s.name.includes(filter.trim().toLowerCase()))

  async function logout() {
    await api.logout().catch(() => {})
    router.replace("/login")
  }

  return (
    <Sidebar variant="inset">
      <SidebarHeader className="gap-3">
        <Link href="/stacks" className="px-1 pt-1">
          <Brand />
        </Link>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              render={<Link href="/stacks/new" />}
              className="bg-sidebar-primary text-sidebar-primary-foreground shadow-sm hover:bg-sidebar-primary/90 hover:text-sidebar-primary-foreground active:bg-sidebar-primary/90 active:text-sidebar-primary-foreground"
            >
              <PlusIcon />
              <span>New stack</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
        <div className="relative">
          <SearchIcon className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <SidebarInput
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder="Filter stacks"
            className="pl-7"
            aria-label="Filter stacks"
          />
        </div>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Stacks{stacks ? ` · ${stacks.length}` : ""}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {isLoading &&
                Array.from({ length: 4 }).map((_, i) => (
                  <SidebarMenuItem key={i}>
                    <SidebarMenuSkeleton showIcon />
                  </SidebarMenuItem>
                ))}
              {error && (
                <div className="flex items-start gap-2 rounded-md border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive">
                  <AlertTriangleIcon className="mt-0.5 size-3.5 shrink-0" />
                  <span>{error.message}</span>
                </div>
              )}
              {stacks && visible.length === 0 && (
                <p className="px-2 py-1.5 text-xs text-muted-foreground">
                  {stacks.length === 0 ? "No stacks yet" : "No matches"}
                </p>
              )}
              {visible.map((s) => {
                const href = `/stacks/${s.name}`
                return (
                  <SidebarMenuItem key={s.name}>
                    <SidebarMenuButton
                      render={<Link href={href} />}
                      isActive={pathname === href}
                      title={statusLabel[s.status]}
                    >
                      <StatusDot status={s.status} className="mx-1" />
                      <span className="truncate">{s.name}</span>
                    </SidebarMenuButton>
                    {s.containers.total > 0 && (
                      <SidebarMenuBadge className="font-mono text-[10px] text-muted-foreground">
                        {s.containers.running}/{s.containers.total}
                      </SidebarMenuBadge>
                    )}
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <ThemeToggle />
          </SidebarMenuItem>
          <SidebarMenuItem>
            <SidebarMenuButton onClick={logout}>
              <LogOutIcon />
              <span>Sign out</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  )
}
