"use client"

import { useQuery } from "@tanstack/react-query"

import { api } from "@/lib/api"

export const POLL_MS = 3000

export const keys = {
  stacks: ["stacks"] as const,
  stack: (name: string) => ["stack", name] as const,
  containers: (name: string) => ["containers", name] as const,
}

export function useStacks() {
  return useQuery({ queryKey: keys.stacks, queryFn: api.listStacks, refetchInterval: POLL_MS })
}

export function useAuthStatus() {
  return useQuery({ queryKey: ["auth"], queryFn: api.authStatus, staleTime: Infinity })
}
