import { useState, useEffect, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Window } from '@wailsio/runtime'

import { Service as HistoryService } from '../../bindings/jpaste/internal/history'

import { log } from '../logger'

export default function JsonViewPage() {
  const [searchParams] = useSearchParams()
  const entryId = parseInt(searchParams.get('id'), 10)
  const [jsonData, setJsonData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const containerRef = useRef(null)
  const editorRef = useRef(null)
  const fetchedRef = useRef(false)
  const JSONEditorRef = useRef(null)

  log.info('JsonViewPage', 'render, id=', entryId, 'loading=', loading, 'hasData=', !!jsonData)

  useEffect(() => {
    if (!entryId) {
      setLoading(false)
      setError('缺少 entry ID 参数')
      return
    }
    if (fetchedRef.current) {
      log.info('JsonViewPage', 'skip duplicate fetch (StrictMode)')
      return
    }
    fetchedRef.current = true

    log.info('JsonViewPage', 'calling GetEntryContent, id=', entryId)
    HistoryService.GetEntryContent(entryId)
      .then((data) => {
        log.info('JsonViewPage', 'data received, len=', data?.length)
        if (!data) {
          setError('条目内容为空')
          setLoading(false)
        } else {
          try {
            setJsonData(JSON.parse(data))
          } catch (e) {
            setError('JSON 解析失败: ' + e.message)
          }
          setLoading(false)
        }
      })
      .catch((err) => {
        log.error('JsonViewPage', 'fetch error:', err)
        setError(err?.message || '获取数据失败')
        setLoading(false)
      })
  }, [entryId])

  useEffect(() => {
    if (!jsonData || !containerRef.current) {
      log.info('JsonViewPage', 'editor init skipped, container=', !!containerRef.current, 'data=', !!jsonData)
      return
    }
    if (editorRef.current) {
      log.info('JsonViewPage', 'editor already exists, updating data')
      editorRef.current.update(jsonData)
      return
    }

    log.info('JsonViewPage', 'loading jsoneditor dynamically')
    let cancelled = false
    Promise.all([
      import('jsoneditor'),
      import('jsoneditor/dist/jsoneditor.css'),
    ]).then(([{ default: JSONEditor }]) => {
      if (cancelled) return
      JSONEditorRef.current = JSONEditor
      log.info('JsonViewPage', 'creating JSONEditor instance')

      // 从 localStorage 读取上次使用的模式，默认为 tree
      const savedMode = (() => {
        try { return localStorage.getItem('jpaste-json-mode') || 'tree' } catch { return 'tree' }
      })()

      const editor = new JSONEditor(containerRef.current, {
        mode: savedMode,
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
      }, jsonData)

      editorRef.current = editor
      log.info('JsonViewPage', 'editor created OK, mode=', savedMode)
    })
    return () => { cancelled = true }
  }, [jsonData])

  useEffect(() => {
    return () => {
      if (editorRef.current) {
        log.info('JsonViewPage', 'destroying editor')
        editorRef.current.destroy()
        editorRef.current = null
      }
    }
  }, [])

  useEffect(() => {
    const handler = (e) => {
      if (e.key !== 'Escape') return
      const tag = document.activeElement?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || document.activeElement?.isContentEditable) return
      e.preventDefault()
      Window.Hide()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [])

  return (
    <div className="w-screen h-screen relative">
      {/* 隐藏 code 模式右上角的 "powered by ace" 链接 */}
      <style>{'.jsoneditor-poweredBy { display: none !important; }'}</style>
      <div ref={containerRef} className="w-full h-screen" />

      {loading && (
        <div className="absolute inset-0 flex items-center justify-center flex-col z-[1]" style={{ background: 'var(--color-background)' }}>
          <span className="text-sm" style={{ color: 'var(--color-muted, #94A3B8)', fontFamily: 'Inter, system-ui, sans-serif' }}>
            加载中...
          </span>
        </div>
      )}

      {error && !loading && (
        <div className="absolute inset-0 flex items-center justify-center flex-col z-[1]" style={{ background: 'var(--color-background)' }}>
          <div>
            <span className="text-sm" style={{ color: 'var(--color-destructive, #EF4444)', fontFamily: 'Inter, system-ui, sans-serif' }}>
              错误: {error}
            </span>
            <br />
            <span className="text-xs mt-2" style={{ color: 'var(--color-muted, #94A3B8)', fontFamily: 'Inter, system-ui, sans-serif' }}>
              Entry ID: {entryId || '(无)'}
            </span>
          </div>
        </div>
      )}
    </div>
  )
}
