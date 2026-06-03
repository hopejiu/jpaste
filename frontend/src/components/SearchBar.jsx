import { useRef, useEffect } from 'react'
import { Search, X, Settings } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import SyncIndicator from './SyncIndicator'

/**
 * Search bar with inline clear button, sync indicator, and settings link.
 *
 * Interface:
 *   search: string
 *   onSearchChange: (term: string) => void
 *   syncStatus: string
 *   styles: object  (searchBox, searchIcon, searchInput, clearBtn, settingsBtn)
 */
export default function SearchBar({ search, onSearchChange, syncStatus, styles }) {
  const inputRef = useRef(null)
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
        placeholder="搜索剪贴板历史..."
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        autoFocus
      />
      {search && (
        <button style={styles.clearBtn} onClick={() => { onSearchChange(''); inputRef.current?.focus() }}>
          <X size={16} />
        </button>
      )}
      <SyncIndicator status={syncStatus} />
      <button style={styles.settingsBtn} onClick={() => navigate('/settings')} title="设置">
        <Settings size={20} />
      </button>
    </div>
  )
}
