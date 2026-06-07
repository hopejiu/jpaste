import { useState, useEffect, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Window } from '@wailsio/runtime'
import { ChevronLeft, ChevronRight } from 'lucide-react'

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
  const [imageList, setImageList] = useState([])
  const [currentIdx, setCurrentIdx] = useState(-1)
  const [hoveredNav, setHoveredNav] = useState(null)

  const imgRef = useRef(null)
  const imgZoomRef = useRef({ scale: 1, tx: 0, ty: 0, dragging: false, lastX: 0, lastY: 0 })
  const [, setImgTick] = useState(0)
  const fetchedRef = useRef(false)

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
  }, [id])

  async function loadImage(entryId) {
    setLoading(true)
    return HistoryService.GetImageDataURL(entryId)
      .then((url) => {
        setImgUrl(url)
        setLoading(false)
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
      let w = img.naturalWidth + 60
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
      } catch { /* silent */ }
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

  return (
    <div className="w-screen h-screen flex items-center justify-center relative overflow-hidden select-none" style={{ background: 'var(--color-image-bg)' }}>
      {loading && <span className="text-sm" style={{ color: 'rgba(255,255,255,0.5)' }}>加载中...</span>}
      {error && !loading && <span className="text-sm text-destructive">{error}</span>}

      {imgUrl && !loading && (
        <>
          {imageList.length > 1 && currentIdx > 0 && (
            <button
              className="fixed top-1/2 -translate-y-1/2 left-4 w-12 h-12 flex items-center justify-center border-none text-white cursor-pointer rounded-full z-10 transition-opacity duration-150"
              style={{ background: 'rgba(255,255,255,0.1)', opacity: hoveredNav === 'left' ? 1 : 0.6 }}
              onClick={() => navigateImg(-1)}
              onMouseEnter={() => setHoveredNav('left')}
              onMouseLeave={() => setHoveredNav(null)}
            >
              <ChevronLeft size={28} />
            </button>
          )}
          {imageList.length > 1 && currentIdx < imageList.length - 1 && (
            <button
              className="fixed top-1/2 -translate-y-1/2 right-4 w-12 h-12 flex items-center justify-center border-none text-white cursor-pointer rounded-full z-10 transition-opacity duration-150"
              style={{ background: 'rgba(255,255,255,0.1)', opacity: hoveredNav === 'right' ? 1 : 0.6 }}
              onClick={() => navigateImg(1)}
              onMouseEnter={() => setHoveredNav('right')}
              onMouseLeave={() => setHoveredNav(null)}
            >
              <ChevronRight size={28} />
            </button>
          )}

          <img
            ref={imgRef}
            src={imgUrl}
            alt=""
            draggable={false}
            className="max-w-full max-h-screen object-contain select-none"
            style={{
              cursor: imgZoomRef.current.dragging ? 'grabbing' : (imgZoomRef.current.scale > 1 ? 'grab' : 'default'),
              transition: 'transform 0.1s ease-out',
              transform: `scale(${imgZoomRef.current.scale}) translate(${imgZoomRef.current.tx}px, ${imgZoomRef.current.ty}px)`,
            }}
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
