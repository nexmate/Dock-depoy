export type SSEEvent = { event: string; data: string }

/** Parse a chunk of an SSE stream; `carry` is the unfinished tail of the previous chunk. */
export function parseSSE(
  chunk: string,
  carry: string
): { events: SSEEvent[]; carry: string } {
  const text = (carry + chunk).replace(/\r\n/g, "\n")
  const blocks = text.split("\n\n")
  const rest = blocks.pop() ?? ""
  const events: SSEEvent[] = []
  for (const block of blocks) {
    let event = "message"
    const data: string[] = []
    for (const line of block.split("\n")) {
      if (line.startsWith("event:")) event = line.slice(6).trim()
      else if (line.startsWith("data:")) data.push(line.slice(5).replace(/^ /, ""))
    }
    if (data.length) events.push({ event, data: data.join("\n") })
  }
  return { events, carry: rest }
}

/** Stream an SSE endpoint with fetch (cookies included); resolves when the stream ends. */
export async function streamSSE(
  url: string,
  onEvent: (e: SSEEvent) => void,
  signal: AbortSignal
): Promise<void> {
  const res = await fetch(url, { signal, credentials: "same-origin" })
  if (!res.ok || !res.body) throw new Error(`stream failed: ${res.status}`)
  const reader = res.body.pipeThrough(new TextDecoderStream()).getReader()
  let carry = ""
  for (;;) {
    const { value, done } = await reader.read()
    if (done) break
    const parsed = parseSSE(value, carry)
    carry = parsed.carry
    parsed.events.forEach(onEvent)
  }
}
