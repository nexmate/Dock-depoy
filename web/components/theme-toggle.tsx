"use client"

import { MoonIcon, SunIcon } from "lucide-react"
import { useTheme } from "next-themes"

import { SidebarMenuButton } from "@/components/ui/sidebar"

export function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme()
  const dark = resolvedTheme === "dark"
  return (
    <SidebarMenuButton onClick={() => setTheme(dark ? "light" : "dark")} tooltip="Toggle theme (D)">
      <SunIcon className="dark:hidden" />
      <MoonIcon className="hidden dark:block" />
      <span>{dark ? "Dark" : "Light"} mode</span>
    </SidebarMenuButton>
  )
}
