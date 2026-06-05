import { useState, useEffect, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Window } from '@wailsio/runtime'
import { ChevronLeft, ChevronRight } from 'lucide-react'

// Wails auto-generated bindings.
// eslint-disable-next-line import/no-unresolved
import { Service as HistoryService } from '../../bindings/jpaste/internal/history'

import { log } from '../logger'

export default function ImageViewPage() {
  const [searchParams] = useSearchParams()
  const id = parseInt(searchParams.get('id'), 10)
  const tagMask = parseInt(searchParams.get('tag') || '0', 10)
  const search = searchParams.get('search') || ''

  const [imgUrl, setImgUrl] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [imageList, setImageList] = useState([]) // ordered entry IDs
  const [currentIdx, setCurrentIdx] = useState(-1)

  const imgRef = useRef(null)
  const imgZoomRef = useRef({ scale: 1, tx: 0, ty: 0, dragging: false, lastX: 0, lastY: 0 })
  const [, setImgTick] = useState(0)
  const fetchedRef = useRef(false)

  // Fetch image list for navigation, then load the current image.
  useEffect(() => {
    if (!id) { setLoading(false); setError('缺少 id 参数'); return }
    if (fetchedRef.current) return
    fetchedRef.current = true

    HistoryService.GetImageList(tagMask, search)
      .then((ids) => {
        const list = ids || []
        setImageList(list)
        const idx = list.indexOf(id)
        setCurrentIdx(idx >= 0 ? idx : 0)
        return loadImage(id)
      })
      .catch((err) => {
        log.error('ImageViewPage', 'list error:', err)
        return loadImage(id)
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id])

  async function loadImage(entryId) {
    setLoading(true)
    return HistoryService.GetImageDataURL(entryId)
      .then((url) => {
        setImgUrl(url)
        setLoading(false)
        // Auto-size window after image loads.
        autoSizeWindow(url)
      })
      .catch((err) => {
        log.error('ImageViewPage', 'load error:', err)
        setError('加载图片失败')
        setLoading(false)
      })
  }

  function autoSizeWindow(url) {
    const img = new Image()
    img.onload = () => {
      const screenW = window.screen.availWidth
      const screenH = window.screen.availHeight
      const maxW = Math.min(screenW * 0.9, 1600)
      const maxH = Math.min(screenH * 0.9, 1000)
      let w = img.naturalWidth + 60  // padding
      let h = img.naturalHeight + 60
      if (w > maxW || h > maxH) {
        const ratio = Math.min(maxW / w, maxH / h)
        w = Math.round(w * ratio)
        h = Math.round(h * ratio)
      }
      const minW = Math.min(w, 480)
      const minH = Math.min(h, 320)
      try {
        Window.SetSize(w, h)
        Window.SetMinSize(minW, minH)
      } catch { /* not all Wails runtimes support this */ }
    }
    img.src = url
  }

  function navigateImg(delta) {
    if (imageList.length === 0) return
    const newIdx = currentIdx + delta
    if (newIdx < 0 || newIdx >= imageList.length) return
    setCurrentIdx(newIdx)
    resetZoom()
    loadImage(imageList[newIdx])
  }

  function resetZoom() {
    const s = imgZoomRef.current
    s.scale = 1; s.tx = 0; s.ty = 0; s.dragging = false
    setImgTick(t => t + 1)
  }

  // Keyboard: Esc close, ← → navigate
  useEffect(() => {
    const handler = (e) => {
      if (e.key === 'Escape') { e.preventDefault(); Window.Hide(); return }
      if (e.key === 'ArrowLeft') { e.preventDefault(); navigateImg(-1); return }
      if (e.key === 'ArrowRight') { e.preventDefault(); navigateImg(1); return }
      if (e.key === '0' || e.key === 'Home') { e.preventDefault(); resetZoom(); return }
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [currentIdx, imageList])

  const styles = {
    container: {
      width: '100%', height: '100vh', background: 'var(--color-image-bg)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      position: 'relative', overflow: 'hidden',
    },
    img: (dragging) => ({
      maxWidth: '100%', maxHeight: '100vh', objectFit: 'contain',
      cursor: dragging ? 'grabbing' : (imgZoomRef.current.scale > 1 ? 'grab' : 'default'),
      userSelect: 'none', transition: 'transform 0.1s ease-out',
    }),
    navBtn: {
      position: 'fixed', top: '50%', transform: 'translateY(-50%)',
      width: 48, height: 48, display: 'flex', alignItems: 'center', justifyContent: 'center',
      border: 'none', background: 'rgba(255,255,255,0.1)', color: '#fff',
      cursor: 'pointer', borderRadius: '50%', zIndex: 10,
      opacity: 0.6, transition: 'opacity 0.15s',
    },
    loading: { color: 'rgba(255,255,255,0.5)', fontSize: 14 },
    error: { color: '#EF4444', fontSize: 14 },
  }

  return (
    <div style={styles.container}>
      {loading && <div style={styles.loading}>加载中...</div>}
      {error && !loading && <div style={styles.error}>{error}</div>}

      {imgUrl && !loading && (
        <>
          {imageList.length > 1 && currentIdx > 0 && (
            <button
              style={{ ...styles.navBtn, left: 16 }}
              onClick={() => navigateImg(-1)}
              onMouseEnter={e => e.currentTarget.style.opacity = '1'}
              onMouseLeave={e => e.currentTarget.style.opacity = '0.6'}
            >
              <ChevronLeft size={28} />
            </button>
          )}
          {imageList.length > 1 && currentIdx < imageList.length - 1 && (
            <button
              style={{ ...styles.navBtn, right: 16 }}
              onClick={() => navigateImg(1)}
              onMouseEnter={e => e.currentTarget.style.opacity = '1'}
              onMouseLeave={e => e.currentTarget.style.opacity = '0.6'}
            >
              <ChevronRight size={28} />
            </button>
          )}

          <img
            ref={imgRef}
            src={imgUrl}
            alt=""
            style={{
              ...styles.img(imgZoomRef.current.dragging),
              transform: `scale(${imgZoomRef.current.scale}) translate(${imgZoomRef.current.tx}px, ${imgZoomRef.current.ty}px)`,
            }}
            draggable={false}
            onClick={() => { resetZoom() }}
            onWheel={(e) => {
              e.preventDefault()
              const s = imgZoomRef.current
              const delta = e.deltaY > 0 ? -0.15 : 0.15
              s.scale = Math.max(0.3, Math.min(10, s.scale + delta))
              if (s.scale <= 1) { s.tx = 0; s.ty = 0 }
              setImgTick(t => t + 1)
            }}
            onMouseDown={(e) => {
              if (imgZoomRef.current.scale <= 1) return
              e.preventDefault()
              const s = imgZoomRef.current
              s.dragging = true; s.lastX = e.clientX; s.lastY = e.clientY
              setImgTick(t => t + 1)
            }}
            onMouseMove={(e) => {
              const s = imgZoomRef.current
              if (!s.dragging || s.scale <= 1) return
              s.tx += (e.clientX - s.lastX) / s.scale
              s.ty += (e.clientY - s.lastY) / s.scale
              s.lastX = e.clientX; s.lastY = e.clientY
              setImgTick(t => t + 1)
            }}
            onMouseUp={() => { imgZoomRef.current.dragging = false; setImgTick(t => t + 1) }}
            onMouseLeave={() => { imgZoomRef.current.dragging = false; setImgTick(t => t + 1) }}
          />
        </>
      )}
    </div>
  )
}
