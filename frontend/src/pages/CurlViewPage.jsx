import { useState, useEffect, useRef, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Window } from '@wailsio/runtime'
import { ArrowLeft, Send, Clipboard, Maximize2, Minimize2, Code, ChevronDown, ChevronRight } from 'lucide-react'

import { Service as HistoryService } from '../../bindings/jpaste/internal/history'
import { Service as CurlViewerService } from '../../bindings/jpaste/internal/curlviewer'

import { log } from '../logger'

const METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']
const HTTP_SCHEMES = ['http', 'https']

const CONVERT_LANGS = [
  { id: 'curl', label: 'cURL' },
  { id: 'Python', label: 'Python (requests)' },
  { id: 'JavaScript', label: 'JavaScript (fetch)' },
  { id: 'Node', label: 'Node.js (http)' },
  { id: 'NodeAxios', label: 'Node.js (axios)' },
  { id: 'Go', label: 'Go (net/http)' },
  { id: 'Rust', label: 'Rust' },
  { id: 'Php', label: 'PHP' },
  { id: 'Java', label: 'Java' },
  { id: 'CSharp', label: 'C#' },
  { id: 'PowerShell', label: 'PowerShell' },
  { id: 'Dart', label: 'Dart' },
  { id: 'Swift', label: 'Swift' },
  { id: 'Kotlin', label: 'Kotlin' },
  { id: 'Ruby', label: 'Ruby' },
  { id: 'Wget', label: 'wget' },
]

function stripHttpProtocol(value) {
  for (const s of HTTP_SCHEMES) {
    const prefix = s + '://'
    if (value.startsWith(prefix)) {
      return { scheme: s, host: value.slice(prefix.length) }
    }
  }
  return null
}

/**
 * Parse a URL into base and query parameters.
 */
function parseUrl(fullUrl) {
  try {
    const u = new URL(fullUrl)
    const params = []
    for (const [k, v] of u.searchParams.entries()) {
      params.push({ key: k, value: v })
    }
    return { base: u.origin + u.pathname, params }
  } catch {
    return { base: fullUrl, params: [] }
  }
}

/**
 * Build a full URL from scheme + host + query params.
 */
function buildUrl(host, scheme, params) {
  const filtered = params.filter(p => p.key.trim())
  const base = `${scheme}://${host}`
  if (filtered.length === 0) return base
  const qs = filtered.map(p => encodeURIComponent(p.key.trim()) + '=' + encodeURIComponent(p.value)).join('&')
  return base + '?' + qs
}

function statusBadgeClass(code) {
  if (code >= 200 && code < 300) return 'text-success'
  if (code >= 300 && code < 400) return 'text-favorite'
  if (code >= 400) return 'text-destructive'
  return ''
}

export default function CurlViewPage() {
  const [searchParams] = useSearchParams()
  const entryId = parseInt(searchParams.get('id'), 10)

  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const [method, setMethod] = useState('GET')
  const [urlBase, setUrlBase] = useState('')
  const [httpScheme, setHttpScheme] = useState('https')
  const [queryParams, setQueryParams] = useState([])
  const [headers, setHeaders] = useState([])
  const [body, setBody] = useState('')
  const [followRedirects, setFollowRedirects] = useState(false)
  const [timeout, setTimeout_] = useState(30)

  const [sendLoading, setSendLoading] = useState(false)
  const [response, setResponse] = useState(null)

  const [jsonMode, setJsonMode] = useState(false)
  const jsonContainerRef = useRef(null)
  const jsonEditorRef = useRef(null)

  const [respCollapsed, setRespCollapsed] = useState(false)
  const fetchedRef = useRef(false)

  // Code generation
  const [originalCurl, setOriginalCurl] = useState('')
  const [targetLang, setTargetLang] = useState('curl')
  const [convertedCode, setConvertedCode] = useState('')
  const curlconverterRef = useRef(null)

  useEffect(() => {
    if (!entryId) {
      setLoading(false)
      setError('缺少 entry ID 参数')
      return
    }
    if (fetchedRef.current) return
    fetchedRef.current = true

    HistoryService.GetEntryContent(entryId)
      .then(async (data) => {
        if (!data) {
          setError('条目内容为空')
          setLoading(false)
          return
        }
        try {
          const curlconverter = await import('curlconverter')
          curlconverterRef.current = curlconverter
          setOriginalCurl(data)
          const parsed = curlconverter.toJsonObject(data)
          applyParsed(parsed)
        } catch (e) {
          log.error('CurlViewPage', 'parse failed:', e)
          setError('curl 解析失败: ' + e.message)
        }
        setLoading(false)
      })
      .catch((err) => {
        log.error('CurlViewPage', 'fetch error:', err)
        setError(err?.message || '获取数据失败')
        setLoading(false)
      })
  }, [entryId])

  function applyParsed(parsed) {
    setMethod((parsed.method || 'GET').toUpperCase())
    const rawUrl = parsed.url || ''
    const { base, params } = parseUrl(rawUrl)
    const detected = stripHttpProtocol(base)
    if (detected) {
      setHttpScheme(detected.scheme)
      setUrlBase(detected.host)
    } else {
      setUrlBase(base)
    }
    setQueryParams(params.length > 0 ? params : [{ key: '', value: '' }])
    const hdrMap = parsed.headers || {}
    const hdrList = Object.keys(hdrMap).map(k => ({ key: k, value: hdrMap[k] }))
    setHeaders(hdrList.length > 0 ? hdrList : [{ key: '', value: '' }])
    setBody(parsed.data || '')
    setFollowRedirects(false)
  }

  const generateCurl = useCallback(() => {
    const fullUrl = buildUrl(urlBase, httpScheme, queryParams)
    let cmd = 'curl'
    if (method !== 'GET') {
      cmd += ' -X ' + method
    }
    for (const h of headers) {
      if (h.key.trim()) {
        cmd += ' -H ' + JSON.stringify(h.key + ': ' + h.value)
      }
    }
    if (body) {
      cmd += ' -d ' + JSON.stringify(body)
    }
    if (followRedirects) {
      cmd += ' -L'
    }
    cmd += ' ' + JSON.stringify(fullUrl)
    return cmd
  }, [method, urlBase, httpScheme, queryParams, headers, body, followRedirects])

  const handleSend = useCallback(async () => {
    const fullUrl = buildUrl(urlBase, httpScheme, queryParams)
    if (!fullUrl) return

    const hdrMap = {}
    for (const h of headers) {
      if (h.key.trim()) {
        hdrMap[h.key.trim()] = h.value
      }
    }

    setSendLoading(true)
    setResponse(null)
    setJsonMode(false)
    setRespCollapsed(false)

    try {
      const resp = await CurlViewerService.SendCurlRequest({
        method,
        url: fullUrl,
        headers: hdrMap,
        body: body || '',
        followRedirects,
        timeout,
      })
      setResponse(resp)
    } catch (err) {
      setResponse({
        statusCode: 0,
        statusText: '请求失败',
        headers: {},
        body: err?.message || '未知错误',
        durationMs: 0,
      })
    }
    setSendLoading(false)
  }, [method, urlBase, httpScheme, queryParams, headers, body, followRedirects, timeout])

  // Auto-generate preview when form fields change (for curl mode)
  useEffect(() => {
    if (targetLang === 'curl') {
      setConvertedCode(generateCurl())
    }
  }, [method, urlBase, httpScheme, queryParams, headers, body, followRedirects, targetLang, generateCurl])

  const handleLangChange = useCallback((lang) => {
    setTargetLang(lang)
    if (lang === 'curl' || !curlconverterRef.current) {
      setConvertedCode(generateCurl())
      return
    }
    try {
      const fn = curlconverterRef.current[`to${lang}`]
      if (fn) {
        const code = fn(originalCurl)
        setConvertedCode(code)
      } else {
        setConvertedCode(`// ${lang}: 不支持此转换目标`)
      }
    } catch (e) {
      setConvertedCode(`// ${lang} 转换失败: ${e.message}`)
    }
  }, [originalCurl, generateCurl])

  const handleCopyConverted = useCallback(() => {
    if (!convertedCode) return
    navigator.clipboard.writeText(convertedCode).catch(e => log.error('CurlViewPage', 'copy converted failed:', e))
  }, [convertedCode])

  useEffect(() => {
    if (!jsonMode || !response || !jsonContainerRef.current) return
    if (jsonEditorRef.current) {
      try {
        const data = JSON.parse(response.body)
        jsonEditorRef.current.update(data)
      } catch { /* ignore */ }
      return
    }

    let cancelled = false
    Promise.all([
      import('jsoneditor'),
      import('jsoneditor/dist/jsoneditor.css'),
    ]).then(([{ default: JSONEditor }]) => {
      if (cancelled) return
      try {
        const data = JSON.parse(response.body)
        const editor = new JSONEditor(jsonContainerRef.current, {
          mode: 'tree',
          modes: ['tree', 'code'],
          mainMenuBar: true,
          navigationBar: true,
          statusBar: true,
          search: true,
          history: true,
          indentation: 2,
        }, data)
        jsonEditorRef.current = editor
      } catch { /* JSON parse failed */ }
    })
    return () => { cancelled = true }
  }, [jsonMode, response])

  useEffect(() => {
    return () => {
      if (jsonEditorRef.current) {
        jsonEditorRef.current.destroy()
        jsonEditorRef.current = null
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

  const isBodyJson = (() => {
    if (!response?.body) return false
    try { JSON.parse(response.body); return true } catch { return false }
  })()

  const handleCopyResponse = useCallback(() => {
    if (!response?.body) return
    navigator.clipboard.writeText(response.body).catch(e => log.error('CurlViewPage', 'copy response failed:', e))
  }, [response])

  const handleUrlChange = useCallback((value) => {
    const detected = stripHttpProtocol(value)
    if (detected) {
      setHttpScheme(detected.scheme)
      setUrlBase(detected.host)
    } else {
      setUrlBase(value)
    }
  }, [])

  const handleToggleJson = useCallback(() => {
    if (jsonMode) {
      if (jsonEditorRef.current) {
        jsonEditorRef.current.destroy()
        jsonEditorRef.current = null
      }
    }
    setJsonMode(v => !v)
  }, [jsonMode])

  if (loading) {
    return (
      <div className="w-screen h-screen flex items-center justify-center bg-background">
        <span className="text-sm text-muted">加载中...</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="w-screen h-screen flex items-center justify-center bg-background">
        <div className="text-center">
          <p className="text-sm text-destructive">{error}</p>
          <p className="text-xs mt-2 text-muted">Entry ID: {entryId || '(无)'}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-screen overflow-hidden bg-background animate-[slideDown_200ms_ease-out]">
      {/* ── Title Bar ── */}
      <div className="flex items-center px-4 py-3 gap-3 border-b border-border flex-shrink-0 bg-surface">
        <button
          className="w-9 h-9 flex items-center justify-center border-none bg-transparent text-foreground cursor-pointer rounded-md transition-[background] duration-fast hover:bg-surface-hover"
          onClick={() => Window.Hide()}
          title="关闭"
        >
          <ArrowLeft size={20} />
        </button>
        <h2 className="text-xl font-semibold flex-1">HTTP 调试</h2>
        <span className="text-xs font-medium px-2.5 py-1 rounded-full bg-primary-alpha-08 text-primary">从 curl 命令解析</span>
      </div>

      {/* ── Scrollable Request Panel ── */}
      <div className={`overflow-y-auto ${response ? 'flex-shrink-0 max-h-[47vh]' : 'flex-1'}`}>
        <div className="px-4 pt-4 pb-3 space-y-3">

          {/* Method + URL — Card */}
          <div className="rounded-lg border border-border bg-surface p-3">
            <div className="flex items-center gap-2">
              <select
                value={method}
                onChange={e => setMethod(e.target.value)}
                className="h-9 text-xs font-mono font-semibold border border-border rounded-md px-2.5 bg-surface text-foreground cursor-pointer outline-none flex-shrink-0 transition-[border-color] duration-fast focus:border-ring"
              >
                {METHODS.map(m => <option key={m} value={m}>{m}</option>)}
              </select>
              <div className="flex items-center gap-0 flex-1">
                <select
                  value={httpScheme}
                  onChange={e => setHttpScheme(e.target.value)}
                  className="h-9 text-xs font-mono font-semibold border border-border rounded-l-md px-2 bg-surface text-foreground cursor-pointer outline-none transition-[border-color] duration-fast focus:border-ring"
                >
                  {HTTP_SCHEMES.map(s => <option key={s} value={s}>{s}</option>)}
                </select>
                <span className="h-9 flex items-center px-0.5 text-xs text-muted bg-surface border-t border-b border-border">://</span>
                <input
                  type="text"
                  value={urlBase}
                  onChange={e => handleUrlChange(e.target.value)}
                  placeholder="example.com/api"
                  className="flex-1 h-9 text-xs font-mono border border-border rounded-r-md px-3 bg-surface text-foreground outline-none transition-[border-color] duration-fast focus:border-ring"
                />
              </div>
            </div>
          </div>

          {/* Query Parameters — Card */}
          <SectionHeader title="Query Parameters" desc="URL 查询参数">
            <div className="p-3">
              <KVTable rows={queryParams} onChange={setQueryParams} keyPlaceholder="key" valuePlaceholder="value" />
            </div>
          </SectionHeader>

          {/* Headers — Card */}
          <SectionHeader title="Headers" desc="请求头">
            <div className="p-3">
              <KVTable rows={headers} onChange={setHeaders} keyPlaceholder="Header-Name" valuePlaceholder="Header-Value" />
            </div>
          </SectionHeader>

          {/* Body — Card */}
          <SectionHeader title="Body" desc="请求体（JSON / Form 数据等）">
            <div className="p-3">
              <textarea
                value={body}
                onChange={e => setBody(e.target.value)}
                placeholder="请求体（可选）"
                rows={4}
                className="w-full text-xs font-mono border border-border rounded-md p-3 bg-surface text-foreground outline-none resize-vertical transition-[border-color] duration-fast focus:border-ring"
              />
            </div>
          </SectionHeader>

          {/* Options + Send — Card */}
          <div className="rounded-lg border border-border bg-surface p-3">
            <div className="flex items-center justify-between flex-wrap gap-y-2">
              <div className="flex items-center gap-4">
                <label className="flex items-center gap-1.5 text-xs text-muted cursor-pointer select-none">
                  <input
                    type="checkbox"
                    checked={followRedirects}
                    onChange={e => setFollowRedirects(e.target.checked)}
                    className="cursor-pointer accent-primary"
                  />
                  跟随重定向
                </label>
                <label className="flex items-center gap-1.5 text-xs text-muted select-none">
                  超时:
                  <input
                    type="number"
                    value={timeout}
                    onChange={e => setTimeout_(Math.max(1, parseInt(e.target.value) || 30))}
                    className="w-14 h-7 text-xs font-mono border border-border rounded-md px-1.5 bg-surface text-foreground outline-none transition-[border-color] duration-fast focus:border-ring"
                    min="1"
                    max="300"
                  />
                  <span className="text-muted">s</span>
                </label>
              </div>
              <button
                onClick={handleSend}
                disabled={sendLoading}
                className="flex items-center gap-1.5 px-5 py-2 text-xs font-medium text-white border-none rounded-md cursor-pointer disabled:opacity-50 transition-all duration-fast bg-primary hover:opacity-90 active:scale-[0.97]"
              >
                <Send size={14} />
                {sendLoading ? '发送中...' : '发送请求'}
              </button>
            </div>
          </div>

          {/* ── Code Generation — Card (collapsed by default) ── */}
          <SectionHeader title="代码生成" desc={targetLang === 'curl' ? 'cURL' : (CONVERT_LANGS.find(l => l.id === targetLang)?.label || targetLang)} defaultOpen={false}>
            <div className="p-3">
              <div className="flex items-center gap-2 mb-2">
                <select
                  value={targetLang}
                  onChange={e => handleLangChange(e.target.value)}
                  className="h-7 text-xs font-mono border border-border rounded-md px-2 bg-surface text-foreground cursor-pointer outline-none transition-[border-color] duration-fast focus:border-ring"
                >
                  {CONVERT_LANGS.map(l => (
                    <option key={l.id} value={l.id}>{l.label}</option>
                  ))}
                </select>
                <button
                  onClick={handleCopyConverted}
                  disabled={!convertedCode}
                  className="flex items-center gap-1 px-3 py-1 text-xs font-medium border border-border rounded-md cursor-pointer disabled:opacity-40 transition-all duration-fast bg-surface text-foreground hover:bg-surface-hover active:scale-[0.97]"
                  title="复制代码"
                >
                  <Clipboard size={12} />
                  复制
                </button>
              </div>
              <pre className="m-0 w-full max-h-[160px] overflow-auto text-[11px] font-mono whitespace-pre-wrap break-all border border-border rounded-md p-2.5 bg-surface text-foreground select-text leading-relaxed">
                {convertedCode || <span className="text-muted">选择语言后自动生成代码</span>}
              </pre>
            </div>
          </SectionHeader>
        </div>
      </div>

      {/* ── Response Panel ── */}
      {response && (
        <div className={`flex flex-col border-t border-border bg-surface ${respCollapsed ? 'flex-shrink-0' : 'flex-1 min-h-0'}`}>
          {/* Status Bar */}
          <div className="flex items-center gap-2 px-4 py-2 border-b border-border bg-background/80 flex-shrink-0">
            {response.statusCode > 0 ? (
              <span className={`text-xs font-semibold px-2.5 py-0.5 rounded-md ${statusBadgeClass(response.statusCode)} bg-primary-alpha-08`}>
                {response.statusCode} {response.statusText}
              </span>
            ) : (
              <span className="text-xs font-semibold text-destructive px-2.5 py-0.5 rounded-md bg-primary-alpha-08">
                {response.statusText}
              </span>
            )}
            {response.durationMs > 0 && (
              <span className="text-xs text-muted font-mono flex items-center gap-1">⏱ {response.durationMs}ms</span>
            )}
            {response.body && (
              <span className="text-xs text-muted font-mono">{response.body.length} B</span>
            )}
            <div className="ml-auto flex items-center gap-1">
              <button
                onClick={handleCopyResponse}
                className="flex items-center gap-1 text-[11px] px-2 py-1 rounded-md border border-border cursor-pointer transition-all duration-fast bg-surface text-muted hover:bg-surface-hover"
                title="复制响应体"
              >
                <Clipboard size={11} />
                复制
              </button>
              {isBodyJson && (
                <button
                  onClick={handleToggleJson}
                  className="flex items-center gap-1 text-[11px] px-2 py-1 rounded-md border border-border cursor-pointer transition-all duration-fast text-primary hover:bg-primary-alpha-06"
                >
                  <Code size={11} />
                  {jsonMode ? '原始' : 'JSON'}
                </button>
              )}
              <button
                onClick={() => setRespCollapsed(v => !v)}
                className="w-7 h-7 flex items-center justify-center border-none bg-transparent text-muted cursor-pointer rounded transition-all duration-fast hover:bg-surface-hover"
                title={respCollapsed ? '展开' : '折叠'}
              >
                {respCollapsed ? <Maximize2 size={14} /> : <Minimize2 size={14} />}
              </button>
            </div>
          </div>

          {/* Response Content */}
          {!respCollapsed && (
            <div className="flex-1 overflow-auto p-3 select-text min-h-0 space-y-3">
              {/* Response Headers */}
              {Object.keys(response.headers).length > 0 && (
                <CollapsibleSection title="响应头" desc={`${Object.keys(response.headers).length} 项`}>
                  <div className="grid gap-x-4 gap-y-0.5 text-xs" style={{ gridTemplateColumns: 'auto 1fr' }}>
                    {Object.entries(response.headers).map(([k, v]) => (
                      <div key={k} className="contents">
                        <span className="font-mono whitespace-nowrap leading-5 text-muted">{k}</span>
                        <span className="font-mono leading-5 break-all text-foreground">{v}</span>
                      </div>
                    ))}
                  </div>
                </CollapsibleSection>
              )}

              {/* Body Content */}
              <CollapsibleSection
                title="响应体"
                desc={response.body ? `${response.body.length} bytes` : '空'}
              >
                {jsonMode ? (
                  <div ref={jsonContainerRef} className="w-full" style={{ minHeight: '200px' }} />
                ) : (
                  <pre className="m-0 text-xs font-mono whitespace-pre-wrap break-all select-text text-foreground leading-relaxed">
                    {response.body || <span className="text-muted">(空响应体)</span>}
                  </pre>
                )}
              </CollapsibleSection>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

// ─── Sub-components ───

function SectionHeader({ title, desc, defaultOpen = true, collapsible = true, children }) {
  const [open, setOpen] = useState(defaultOpen)
  return (
    <div className="rounded-lg border border-border bg-background/50 overflow-hidden">
      <div
        className={`flex items-center gap-1.5 px-4 py-2.5 border-b border-border transition-colors duration-fast ${collapsible ? 'cursor-pointer select-none hover:bg-surface-hover' : ''}`}
        onClick={() => collapsible && setOpen(!open)}
      >
        {collapsible && (open ? <ChevronDown size={14} className="flex-shrink-0 text-muted" /> : <ChevronRight size={14} className="flex-shrink-0 text-muted" />)}
        <span className="text-sm font-medium text-foreground">{title}</span>
        {desc && <span className="text-xs text-muted truncate">{desc}</span>}
      </div>
      {open && children}
    </div>
  )
}

function CollapsibleSection({ title, desc, defaultOpen = true, extra, children }) {
  const [open, setOpen] = useState(defaultOpen)
  return (
    <div>
      <div className="flex items-center gap-1.5 mb-1.5">
        <div
          className="flex items-center gap-1.5 cursor-pointer select-none text-foreground hover:text-primary transition-colors duration-fast min-w-0"
          onClick={() => setOpen(!open)}
        >
          {open ? <ChevronDown size={14} className="flex-shrink-0 text-muted" /> : <ChevronRight size={14} className="flex-shrink-0 text-muted" />}
          <span className="text-sm font-medium whitespace-nowrap">{title}</span>
          {desc && <span className="text-xs text-muted truncate">{desc}</span>}
        </div>
        {extra}
      </div>
      {open && children}
    </div>
  )
}

function KVTable({ rows, onChange, keyPlaceholder, valuePlaceholder }) {
  const ensureLast = (list) => {
    if (list.length === 0 || list[list.length - 1].key !== '' || list[list.length - 1].value !== '') {
      return [...list, { key: '', value: '' }]
    }
    return list
  }

  const updateRow = (idx, field, val) => {
    const next = rows.map((r, i) => i === idx ? { ...r, [field]: val } : r)
    onChange(ensureLast(next))
  }

  const removeRow = (idx) => {
    if (rows.length <= 1) return
    onChange(rows.filter((_, i) => i !== idx))
  }

  return (
    <div className="space-y-1">
      {rows.map((row, idx) => {
        const isEmpty = row.key === '' && row.value === ''
        const isLast = idx === rows.length - 1
        return (
          <div key={idx} className="flex items-center gap-1.5 group">
            <input
              type="text"
              value={row.key}
              onChange={e => updateRow(idx, 'key', e.target.value)}
              placeholder={keyPlaceholder}
              className="flex-1 h-8 text-xs font-mono border border-border rounded-md px-2.5 bg-surface text-foreground outline-none transition-[border-color] duration-fast focus:border-ring"
            />
            <span className="text-xs text-muted flex-shrink-0">:</span>
            <input
              type="text"
              value={row.value}
              onChange={e => updateRow(idx, 'value', e.target.value)}
              placeholder={valuePlaceholder}
              className="flex-1 h-8 text-xs font-mono border border-border rounded-md px-2.5 bg-surface text-foreground outline-none transition-[border-color] duration-fast focus:border-ring"
            />
            {isLast && isEmpty ? (
              <span className="w-7 h-7 flex items-center justify-center text-xs text-muted flex-shrink-0">
                +
              </span>
            ) : (
              <button
                onClick={() => removeRow(idx)}
                className="w-7 h-7 flex items-center justify-center border-none bg-transparent cursor-pointer flex-shrink-0 rounded transition-all duration-fast text-muted hover:text-destructive"
                title="删除此行"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
            )}
          </div>
        )
      })}
    </div>
  )
}
