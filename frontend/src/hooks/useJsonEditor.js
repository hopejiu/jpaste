import { useRef, useCallback } from 'react'

/**
 * Hook for dynamically loading and managing a JSONEditor instance.
 * Returns imperative methods so both full-page (JsonViewPage) and
 * toggle-on-demand (CurlViewPage) scenarios work.
 *
 * @returns {{ containerRef, updateJson, destroyEditor }}
 *   containerRef  - attach to the div that will hold the editor
 *   updateJson(data, mode?)  - set data; creates editor on first call, updates after
 *   destroyEditor() - destroys and cleans up the instance
 */
export function useJsonEditor() {
  const containerRef = useRef(null)
  const editorRef = useRef(null)

  const updateJson = useCallback(async (data, mode = null) => {
    const container = containerRef.current
    if (!container || !data) return

    // Update existing.
    if (editorRef.current) {
      editorRef.current.update(data)
      return
    }

    // Lazy-load and create.
    const [{ default: JSONEditor }] = await Promise.all([
      import('jsoneditor'),
      import('jsoneditor/dist/jsoneditor.css'),
    ])

    const savedMode = (() => {
      try { return localStorage.getItem('jpaste-json-mode') || 'tree' } catch { return 'tree' }
    })()

    const editor = new JSONEditor(container, {
      mode: mode || savedMode,
      modes: ['tree', 'code'],
      mainMenuBar: true,
      navigationBar: true,
      statusBar: true,
      search: true,
      history: true,
      indentation: 2,
      sortObjectKeys: false,
      limitDragging: false,
      onModeChange: (newMode) => {
        try { localStorage.setItem('jpaste-json-mode', newMode) } catch { /* ignore */ }
      },
    }, data)
    editorRef.current = editor
  }, [])

  const destroyEditor = useCallback(() => {
    if (editorRef.current) {
      editorRef.current.destroy()
      editorRef.current = null
    }
  }, [])

  return { containerRef, updateJson, destroyEditor }
}
