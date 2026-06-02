import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search, Settings, X, Copy, ClipboardPaste, ExternalLink, Trash2 } from 'lucide-react'
import { useClipboard } from '../context/ClipboardContext'
import { Service as FileService } from '../../bindings/jpaste/internal/fileop'
import { getById } from '../actions'
import { useActionDetection } from '../hooks/useActionDetection'
import { useKeyboardNavigation } from '../hooks/useKeyboardNavigation'
import { useContextMenu } from '../hooks/useContextMenu'
import { formatTime, previewContent } from '../utils/format'
import SyncIndicator from '../components/SyncIndicator'
import ActionButtons from '../components/ActionButtons'
import ActionModal from '../components/ActionModal'
import { styles } from './MainPage.styles'

export default function MainPage() {
  const { entries, search, setSearch, useEntry, deleteEntry, settings, syncStatus } = useClipboard()
  const [focusedIdx, setFocusedIdx] = useState(-1)
  const [modal, setModal] = useState(null)
  const inputRef = useRef(null)
  const listRef = useRef(null)
  const navigate = useNavigate()

  const { ctxMenu, showCtxMenu, hideCtxMenu } = useContextMenu()
  const closeModal = useCallback(() => setModal(null), [])

  const { detectedMap, observeItem } = useActionDetection(entries, settings.action_config, listRef)

  const handleKeyDown = useKeyboardNavigation({
    entries, focusedIdx, settings, useEntry, setSearch, setFocusedIdx, inputRef, modal, closeModal,
  })

  // Auto-focus search + scroll to top on mount.
  useEffect(() => {
    listRef.current?.scrollTo(0, 0)
    inputRef.current?.focus()
  }, [])

  // Re-focus search + scroll to top on window shown.
  useEffect(() => {
    const handler = () => {
      listRef.current?.scrollTo(0, 0)
      inputRef.current?.focus()
    }
    window.addEventListener('focus', handler)
    return () => window.removeEventListener('focus', handler)
  }, [])

  // Scroll focused item into view.
  useEffect(() => {
    if (focusedIdx >= 0 && listRef.current) {
      const item = listRef.current.children[focusedIdx]
      item?.scrollIntoView({ block: 'nearest' })
    }
  }, [focusedIdx])

  const handleOpenEditor = async (id) => {
    hideCtxMenu()
    try {
      await FileService.OpenInEditor(id)
    } catch (err) {
      console.error('Failed to open in editor:', err)
    }
  }

  const handleDelete = (entry) => {
    hideCtxMenu()
    deleteEntry(entry.id)
  }

  const handleActionClick = useCallback((actionId, entry) => {
    setModal({ actionId, entry })
  }, [])

  const modalAction = modal ? getById(modal.actionId) : null

  return (
    <div style={styles.container} onKeyDown={handleKeyDown} tabIndex={0}>
      {/* Header */}
      <div style={styles.header}>
        <div style={styles.searchBox}>
          <Search size={18} style={styles.searchIcon} />
          <input
            ref={inputRef}
            style={styles.searchInput}
            type="text"
            placeholder="搜索剪贴板历史..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              setFocusedIdx(-1)
            }}
            autoFocus
          />
          {search && (
            <button style={styles.clearBtn} onClick={() => { setSearch(''); inputRef.current?.focus() }}>
              <X size={16} />
            </button>
          )}
        </div>
        <SyncIndicator status={syncStatus.status} />
        <button style={styles.settingsBtn} onClick={() => navigate('/settings')} title="设置">
          <Settings size={20} />
        </button>
      </div>

      {/* List */}
      <div style={styles.list} ref={listRef}>
        {entries.length === 0 ? (
          <div style={styles.empty}>
            <p style={styles.emptyTitle}>{search ? '无匹配记录' : '暂无剪贴板历史'}</p>
            <p style={styles.emptyDesc}>
              {search ? '换个关键词试试' : '复制文本即可开始。jPaste 在后台监听剪贴板。'}
            </p>
          </div>
        ) : (
          entries.map((entry, idx) => {
            const isFocused = idx === focusedIdx
            const shortcut = idx < 9 ? `Ctrl+${idx + 1}` : null
            const time = formatTime(entry.updated_at)
            const detectedActions = detectedMap[entry.id]
            return (
              <div
                key={entry.id}
                ref={(el) => observeItem(el, entry.id, entry.content)}
                style={{ ...styles.item, ...(isFocused ? styles.itemFocused : {}) }}
                onMouseEnter={() => setFocusedIdx(idx)}
                onClick={() => useEntry(entry.id, settings.default_action)}
                onContextMenu={(e) => showCtxMenu(e, entry)}
              >
                {shortcut && <div style={styles.shortcut}>{idx + 1}</div>}

                <div style={styles.itemContent}>
                  <div style={styles.itemText}>{previewContent(entry.content)}</div>
                  <div style={styles.itemMeta}>
                    <span style={styles.itemTime}>
                      <span style={styles.itemRel}>{time.rel}</span>
                      <span style={styles.itemAbs}>{time.abs}</span>
                    </span>
                    <ActionButtons
                      actionIds={detectedActions}
                      onClick={(actionId) => handleActionClick(actionId, entry)}
                    />
                    <button
                      style={styles.actionBtn}
                      onClick={(e) => { e.stopPropagation(); useEntry(entry.id, 'copy') }}
                      title="复制"
                    >
                      <Copy size={14} />
                    </button>
                    <button
                      style={styles.actionBtn}
                      onClick={(e) => { e.stopPropagation(); useEntry(entry.id, 'paste') }}
                      title="粘贴"
                    >
                      <ClipboardPaste size={14} />
                    </button>
                  </div>
                </div>
              </div>
            )
          })
        )}
      </div>

      {/* Footer */}
      <div style={styles.footer}>
        <span style={styles.footerText}>Alt+V 切换 · Ctrl+1-9 选择 · 右键更多操作</span>
      </div>

      {/* Action Modal */}
      <ActionModal
        open={!!modal}
        onClose={closeModal}
        title={modalAction?.label || ''}
        wide={modal?.actionId === 'json'}
      >
        {modalAction && (
          <modalAction.Component
            content={modal.entry.content}
            entryId={modal.entry.id}
            onClose={closeModal}
          />
        )}
      </ActionModal>

      {/* Context Menu */}
      {ctxMenu && (
        <div style={{ ...styles.ctxOverlay, left: ctxMenu.x, top: ctxMenu.y }}>
          <div style={styles.ctxItem} onClick={() => handleOpenEditor(ctxMenu.entry.id)}>
            <ExternalLink size={14} />
            <span>在编辑器中打开</span>
          </div>
          <div style={styles.ctxItemDanger} onClick={() => handleDelete(ctxMenu.entry)}>
            <Trash2 size={14} />
            <span>删除</span>
          </div>
        </div>
      )}
    </div>
  )
}
