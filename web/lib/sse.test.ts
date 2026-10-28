import { describe, expect, it } from "vitest"

import { parseSSE } from "./sse"

describe("parseSSE", () => {
  it("parses multiple events in one chunk", () => {
    const { events, carry } = parseSSE(
      'event: line\ndata: "a"\n\nevent: done\ndata: {"exitCode":0}\n\n',
      ""
    )
    expect(events).toEqual([
      { event: "line", data: '"a"' },
      { event: "done", data: '{"exitCode":0}' },
    ])
    expect(carry).toBe("")
  })

  it("carries partial events across chunks", () => {
    const first = parseSSE('event: line\ndata: "hel', "")
    expect(first.events).toEqual([])
    const second = parseSSE('lo"\n\n', first.carry)
    expect(second.events).toEqual([{ event: "line", data: '"hello"' }])
  })

  it("defaults event name to message and handles CRLF", () => {
    const { events } = parseSSE("data: x\r\n\r\n", "")
    expect(events).toEqual([{ event: "message", data: "x" }])
  })

  it("joins multi-line data", () => {
    const { events } = parseSSE("data: a\ndata: b\n\n", "")
    expect(events).toEqual([{ event: "message", data: "a\nb" }])
  })
})
