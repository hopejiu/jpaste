import { useRef, useEffect } from 'react'
import { Search, X, Settings, Code2 } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import SyncIndicator from './SyncIndicator'

export default function SearchBar({ search, onSearchChange, syncStatus, inputRef: externalRef, isRegex, onToggleRegex, styles }) {
  const internalRef = useRef(null)
  const inputRef = externalRef || internalRef
  const navigate = useNavigate()

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  return (
    <div style={styles.searchBox}>
      <Search size={18} style={styles.searchIcon} />
      <input
        ref={inputRef}
        style={styles.searchInput}
        type="text"
        placeholder={isRegex ? '正则搜索...' : '搜索剪贴板历史...'}
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        autoFocus
      />
      {search && (
        <button style={styles.clearBtn} onClick={() => { onSearchChange(''); inputRef.current?.focus() }}>
          <X size={16} />
        </button>
      )}
      <button
        style={isRegex ? { ...styles.regexBtn, ...styles.regexBtnActive } : styles.regexBtn}
        onClick={() => onToggleRegex(!isRegex)}
        title={isRegex ? '正则模式（点击关闭）' : '正则模式（点击开启）'}
      >
        <Code2 size={16} />
      </button>
      <SyncIndicator status={syncStatus} />
      <button style={styles.settingsBtn} onClick={() => navigate('/settings')} title="设置">
        <Settings size={20} />
      </button>
    </div>
  )
}
