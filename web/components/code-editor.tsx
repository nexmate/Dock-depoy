"use client"

import * as React from "react"
import CodeMirror, { EditorView } from "@uiw/react-codemirror"
import { yaml } from "@codemirror/lang-yaml"
import { StreamLanguage } from "@codemirror/language"
import { properties } from "@codemirror/legacy-modes/mode/properties"
import { useTheme } from "next-themes"

import { cn } from "@/lib/utils"

const languages = {
  yaml: () => yaml(),
  env: () => StreamLanguage.define(properties),
}

const baseTheme = EditorView.theme({
  "&": { fontSize: "13px", backgroundColor: "transparent" },
  ".cm-content": { fontFamily: "var(--font-mono), ui-monospace, monospace", padding: "12px 0" },
  ".cm-gutters": { backgroundColor: "transparent", border: "none" },
  "&.cm-focused": { outline: "none" },
})

type Props = {
  value: string
  onChange: (value: string) => void
  language: keyof typeof languages
  className?: string
  minHeight?: string
  "aria-label"?: string
}

export function CodeEditor({ value, onChange, language, className, minHeight = "320px", ...rest }: Props) {
  const { resolvedTheme } = useTheme()
  const extensions = React.useMemo(() => [languages[language](), baseTheme], [language])
  return (
    <div className={cn("overflow-hidden rounded-lg border bg-muted/30 focus-within:ring-2 focus-within:ring-ring/40", className)}>
      <CodeMirror
        value={value}
        onChange={onChange}
        extensions={extensions}
        theme={resolvedTheme === "dark" ? "dark" : "light"}
        minHeight={minHeight}
        basicSetup={{ foldGutter: true, highlightActiveLine: true, tabSize: 2 }}
        aria-label={rest["aria-label"]}
      />
    </div>
  )
}
