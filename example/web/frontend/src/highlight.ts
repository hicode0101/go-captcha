import hljs from 'highlight.js/lib/core'
import go from 'highlight.js/lib/languages/go'
import 'highlight.js/styles/github-dark.css'

hljs.registerLanguage('go', go)

export function highlightedGo(code: string): string {
  return hljs.highlight(code, { language: 'go' }).value
}
