import { useState, useEffect, useRef, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { ArrowLeft, Send, Plug, PlugZap, Copy, ChevronDown, ChevronRight, Trash2 } from 'lucide-react'

import { useEscapeHide } from '../hooks/useEscapeHide'
import { Window, Events } from '@wailsio/runtime'
import { EVENTS } from '../events'
import { Service as HistoryService } from '../../bindings/jpaste/internal/history'
import { log } from '../logger'

const WS_SCHEMES = ['ws', 'wss']

function stripWsProtocol(value) {
  for (const s of WS_SCHEMES) {
    const prefix = s + '://'
    if (value.startsWith(prefix)) {
      return { scheme: s, host: value.slice(prefix.length) }
    }
  }
  return null
}

function isJson(str) {
  if (!str) return false
  try { JSON.parse(str); return true } catch { return false }
}

function formatJson(str) {
  try { return JSON.stringify(JSON.parse(str), null, 2) } catch { return str }
}

function formatTime(d) {
  const h = String(d.getHours()).padStart(2, '0')
  const m = String(d.getMinutes()).padStart(2, '0')
  const s = String(d.getSeconds()).padStart(2, '0')
  return `${h}:${m}:${s}`
}

export default function WsViewPage() {
  useEscapeHide()
  const [searchParams] = useSearchParams()
  const entryId = parseInt(searchParams.get('id'), 10)

  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const [wsUrl, setWsUrl] = useState('')
  const [wsScheme, setWsScheme] = useState('wss')
  const [connected, setConnected] = useState(false)
  const [connecting, setConnecting] = useState(false)
  const [messages, setMessages] = useState([])
  const [inputText, setInputText] = useState('')
  const [expandedJson, setExpandedJson] = useState({})

  const wsRef = useRef(null)
  const msgListRef = useRef(null)
  const fetchedRef = useRef(false)
  const msgCountRef = useRef(0)

  useEffect(() => {
    if (!entryId) {
      setLoading(false)
      setError('缺少 entry ID 参数')
      return
    }
    if (fetchedRef.current) return
    fetchedRef.current = true

    HistoryService.GetEntryContent(entryId)
      .then((data) => {
        if (!data) {
          setError('条目内容为空')
        } else {
          const raw = data.trim()
          const detected = stripWsProtocol(raw)
          if (detected) {
            setWsScheme(detected.scheme)
            setWsUrl(detected.host)
          } else {
            setWsUrl(raw)
          }
        }
        setLoading(false)
      })
      .catch((err) => {
        log.error('WsViewPage', 'fetch error:', err)
        setError(err?.message || '获取数据失败')
        setLoading(false)
      })
  }, [entryId])

  const addMessage = useCallback((type, text) => {
    setMessages(prev => {
      const idx = msgCountRef.current++
      return [...prev, { type, text, time: new Date(), id: idx }]
    })
  }, [])

  const handleWsUrlChange = useCallback((value) => {
    const detected = stripWsProtocol(value)
    if (detected) {
      setWsScheme(detected.scheme)
      setWsUrl(detected.host)
    } else {
      setWsUrl(value)
    }
  }, [])

  const handleConnect = useCallback(() => {
    if (!wsUrl || wsRef.current) return
    const fullUrl = `${wsScheme}://${wsUrl}`
    setConnecting(true)
    addMessage('system', `正在连接 ${fullUrl}...`)

    try {
      const ws = new WebSocket(fullUrl)
      ws.onopen = () => {
        setConnected(true)
        setConnecting(false)
        addMessage('system', '已连接')
      }
      ws.onmessage = (event) => {
        addMessage('received', event.data)
      }
      ws.onerror = () => {
        addMessage('system', '连接错误')
        setConnected(false)
        setConnecting(false)
        wsRef.current = null
      }
      ws.onclose = (event) => {
        addMessage('system', `连接已关闭 (code=${event.code})`)
        setConnected(false)
        setConnecting(false)
        wsRef.current = null
      }
      wsRef.current = ws
    } catch (e) {
      addMessage('system', '创建连接失败: ' + e.message)
      setConnecting(false)
    }
  }, [wsUrl, wsScheme, addMessage])

  const handleDisconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
      setConnected(false)
      setConnecting(false)
      addMessage('system', '已断开连接')
    }
  }, [addMessage])

  const handleSend = useCallback(() => {
    if (!wsRef.current || !inputText.trim()) return
    const text = inputText.trim()
    wsRef.current.send(text)
    addMessage('sent', text)
    setInputText('')
  }, [inputText, addMessage])

  const handleKeyDown = useCallback((e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }, [handleSend])

  useEffect(() => {
    if (msgListRef.current) {
      msgListRef.current.scrollTop = msgListRef.current.scrollHeight
    }
  }, [messages])

  useEffect(() => {
    return () => {
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
    }
  }, [])

  // Disconnect WebSocket when the window is hidden (Escape, back button, focus loss).
  useEffect(() => {
    const unsub = Events.On(EVENTS.WINDOW_HIDING, () => {
      if (wsRef.current) {
        wsRef.current.close(1001, 'window hidden')
        wsRef.current = null
        setConnected(false)
        setConnecting(false)
      }
    })
    return () => unsub()
  }, [])

  const handleCopyMessage = useCallback((text) => {
    navigator.clipboard.writeText(text).catch(e => log.error('WsViewPage', 'copy failed:', e))
  }, [])

  const toggleJsonExpand = useCallback((id) => {
    setExpandedJson(prev => ({ ...prev, [id]: !prev[id] }))
  }, [])

  const clearMessages = useCallback(() => {
    setMessages([])
    msgCountRef.current = 0
  }, [])

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
          onClick={() => Window.Close()}
          title="关闭"
        >
          <ArrowLeft size={20} />
        </button>
        <h2 className="text-xl font-semibold flex-1">WS 调试</h2>
        {connected && (
          <span className="flex items-center gap-1.5 text-xs font-medium px-2.5 py-1 rounded-full bg-primary-alpha-08 text-success">
            <span className="w-1.5 h-1.5 rounded-full bg-success" />
            已连接
          </span>
        )}
      </div>

      {/* ── Connection Bar ── */}
      <div className="flex items-center gap-2 px-4 py-3 border-b border-border bg-surface flex-shrink-0">
        <select
          value={wsScheme}
          onChange={e => setWsScheme(e.target.value)}
          disabled={connected || connecting}
          className="h-9 text-xs font-mono font-semibold border border-border rounded-md px-2 bg-surface text-foreground cursor-pointer outline-none flex-shrink-0 transition-[border-color] duration-fast focus:border-ring disabled:opacity-40"
        >
          {WS_SCHEMES.map(s => <option key={s} value={s}>{s}</option>)}
        </select>
        <span className="h-9 flex items-center text-xs text-muted font-mono flex-shrink-0">://</span>
        <input
          type="text"
          value={wsUrl}
          onChange={e => handleWsUrlChange(e.target.value)}
          placeholder="example.com/ws"
          className="flex-1 h-9 text-xs font-mono border border-border rounded-md px-3 bg-background text-foreground outline-none transition-[border-color] duration-fast focus:border-ring disabled:opacity-40"
          disabled={connected || connecting}
        />
        {connected ? (
          <button
            onClick={handleDisconnect}
            className="flex items-center gap-1.5 px-4 py-2 text-xs font-medium text-white border-none rounded-md cursor-pointer transition-all duration-fast bg-destructive hover:opacity-90 active:scale-[0.97]"
          >
            <PlugZap size={14} />
            断开
          </button>
        ) : (
          <button
            onClick={handleConnect}
            disabled={connecting || !wsUrl}
            className="flex items-center gap-1.5 px-4 py-2 text-xs font-medium text-white border-none rounded-md cursor-pointer disabled:opacity-50 transition-all duration-fast bg-primary hover:opacity-90 active:scale-[0.97]"
          >
            <Plug size={14} />
            {connecting ? '连接中...' : '连接'}
          </button>
        )}
      </div>

      {/* ── Messages Area ── */}
      <div
        ref={msgListRef}
        className="flex-1 overflow-auto"
        style={{ background: 'var(--color-background)' }}
      >
        {messages.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full gap-3 px-4">
            <div className="w-14 h-14 rounded-full bg-surface border border-border flex items-center justify-center">
              <Plug size={22} className="text-muted" />
            </div>
            <span className="text-sm text-muted">连接 WebSocket 后开始调试</span>
            <span className="text-xs text-muted/60">输入地址并点击「连接」按钮</span>
          </div>
        ) : (
          <div className="px-3 py-3 space-y-2">
            {messages.map((msg) => {
              const isSystem = msg.type === 'system'
              const isSent = msg.type === 'sent'
              const isReceived = msg.type === 'received'
              const isMsgJson = isJson(msg.text)
              const isExpanded = !!expandedJson[msg.id]

              if (isSystem) {
                return (
                  <div key={msg.id} className="flex justify-center py-1">
                    <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[11px] bg-surface border border-border text-muted/70">
                      <span className="opacity-50">{formatTime(msg.time)}</span>
                      {msg.text}
                    </span>
                  </div>
                )
              }

              return (
                <div
                  key={msg.id}
                  className={`flex ${isSent ? 'justify-end' : 'justify-start'} px-1`}
                >
                  <div
                    className={`max-w-[80%] min-w-0 overflow-hidden ${
                      isSent
                        ? 'rounded-2xl rounded-br-md bg-primary text-white'
                        : 'rounded-2xl rounded-bl-md bg-surface border border-border shadow-card'
                    }`}
                  >
                    {/* Message Header */}
                    <div className={`flex items-center gap-1.5 px-3.5 pt-2 pb-0.5 ${
                      isSent ? 'text-white/50' : 'text-muted'
                    }`}>
                      <span className="text-[10px] font-medium leading-none">
                        {formatTime(msg.time)}
                      </span>
                      <span className={`text-[10px] ml-auto ${isSent ? 'text-white/40' : 'text-muted/50'}`}>
                        {isSent ? '已发送' : '已接收'}
                      </span>
                    </div>

                    {/* Message Body */}
                    <div className={`px-3.5 pb-1.5 ${!isSent && 'border-l-[3px] border-primary/30 ml-0 pl-2.5'}`}>
                      {isExpanded ? (
                        <pre className="m-0 text-[11px] font-mono whitespace-pre-wrap break-all leading-relaxed select-text"
                          style={{ color: isSent ? 'rgba(255,255,255,0.9)' : 'var(--color-foreground)' }}
                        >
                          {formatJson(msg.text)}
                        </pre>
                      ) : (
                        <span className="text-xs font-mono whitespace-pre-wrap break-all leading-relaxed select-text block"
                          style={{ color: isSent ? '#fff' : 'var(--color-foreground)' }}
                        >
                          {msg.text}
                        </span>
                      )}
                    </div>

                    {/* Action Buttons */}
                    <div className={`flex items-center gap-1 px-3 pb-2 ${isSent ? 'justify-end' : 'justify-start'}`}>
                      <button
                        onClick={() => handleCopyMessage(msg.text)}
                        className={`flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-md cursor-pointer transition-all duration-fast leading-none font-medium ${
                          isSent
                            ? 'text-white/60 hover:bg-white/10'
                            : 'text-muted hover:bg-surface-hover'
                        }`}
                        title="复制消息"
                      >
                        <Copy size={10} />
                        复制
                      </button>
                      {isMsgJson && (
                        <button
                          onClick={() => toggleJsonExpand(msg.id)}
                          className={`flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-md cursor-pointer transition-all duration-fast leading-none font-medium ${
                            isSent
                              ? 'text-white/60 hover:bg-white/10'
                              : 'text-primary hover:bg-primary-alpha-06'
                          }`}
                        >
                          {isExpanded ? <ChevronDown size={10} /> : <ChevronRight size={10} />}
                          {isExpanded ? '折叠' : 'JSON'}
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* ── Input Bar ── */}
      <div className="flex items-center gap-2 px-4 py-3 border-t border-border bg-surface flex-shrink-0">
        <input
          type="text"
          value={inputText}
          onChange={e => setInputText(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={connected ? '输入消息，Enter 发送' : '请先连接'}
          disabled={!connected}
          className="flex-1 h-9 text-xs font-mono border border-border rounded-md px-3 bg-background text-foreground outline-none transition-[border-color] duration-fast focus:border-ring disabled:opacity-40"
        />
        <div className="flex items-center gap-1.5">
          {messages.length > 0 && (
            <button
              onClick={clearMessages}
              className="flex items-center justify-center w-9 h-9 border border-border rounded-md cursor-pointer transition-all duration-fast bg-surface text-muted hover:bg-surface-hover active:scale-[0.97]"
              title="清空消息"
            >
              <Trash2 size={14} />
            </button>
          )}
          <button
            onClick={handleSend}
            disabled={!connected || !inputText.trim()}
            className="flex items-center gap-1.5 px-4 py-2 text-xs font-medium text-white border-none rounded-md cursor-pointer disabled:opacity-50 transition-all duration-fast bg-primary hover:opacity-90 active:scale-[0.97]"
          >
            <Send size={14} />
            发送
          </button>
        </div>
      </div>
    </div>
  )
}
