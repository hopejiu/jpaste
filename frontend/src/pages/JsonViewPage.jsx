import { useState, useEffect, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Window } from '@wailsio/runtime'
import { useJsonEditor } from '../hooks/useJsonEditor'
import { Service as HistoryService } from '../../bindings/jpaste/internal/history'
import { log } from '../logger'

export default function JsonViewPage() {
  const [searchParams] = useSearchParams()
  const entryId = parseInt(searchParams.get('id'), 10)
  const [jsonData, setJsonData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const fetchedRef = useRef(false)
  const { containerRef, updateJson, destroyEditor } = useJsonEditor()

  // Capture-phase Escape handler: fires before jsoneditor's internal handlers
  // to reliably close the window (jsoneditor may stopPropagation on Escape).
  useEffect(() => {
    const handler = (e) => {
      if (e.key === 'Escape') { e.preventDefault(); Window.Close() }
    }
    document.addEventListener('keydown', handler, true)
    return () => document.removeEventListener('keydown', handler, true)
  }, [])

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
    if (!jsonData) return
    log.info('JsonViewPage', 'updating/creating editor')
    updateJson(jsonData)
  }, [jsonData])

  useEffect(() => {
    return () => destroyEditor()
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
