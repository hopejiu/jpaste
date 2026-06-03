import { useState, useRef } from 'react'
import { XCircle } from 'lucide-react'

/**
 * Full-screen image preview overlay with zoom (wheel) and pan (drag).
 *
 * Interface:
 *   imagePreview: { url: string, loading: boolean } | null
 *   onClose: () => void
 */
export default function ImagePreview({ imagePreview, onClose }) {
  const imgZoomRef = useRef({ scale: 1, tx: 0, ty: 0, dragging: false, lastX: 0, lastY: 0 })
  const [, setImgTick] = useState(0)

  if (!imagePreview) return null

  const styles = {
    overlay: {
      position: 'fixed', inset: 0, zIndex: 1000,
      background: 'rgba(0,0,0,0.85)', display: 'flex',
      alignItems: 'center', justifyContent: 'center',
      cursor: 'zoom-out',
    },
    closeBtn: {
      position: 'absolute', top: 16, right: 16, zIndex: 10,
      background: 'none', border: 'none', color: '#fff',
      cursor: 'pointer', padding: 4,
    },
    loading: { color: '#fff', fontSize: 14 },
    img: (dragging) => ({
      maxWidth: '90vw', maxHeight: '90vh', objectFit: 'contain',
      borderRadius: 8, cursor: dragging ? 'grabbing' : 'grab',
      userSelect: 'none',
    }),
  }

  const handleClose = () => {
    const s = imgZoomRef.current
    s.scale = 1; s.tx = 0; s.ty = 0
    onClose()
  }

  return (
    <div style={styles.overlay} onClick={handleClose}>
      <button style={styles.closeBtn} onClick={handleClose}>
        <XCircle size={24} />
      </button>
      {imagePreview.loading ? (
        <div style={styles.loading}>加载中...</div>
      ) : (
        <img
          src={imagePreview.url}
          alt="clipboard preview"
          style={{
            ...styles.img(imgZoomRef.current.dragging),
            transform: `scale(${imgZoomRef.current.scale}) translate(${imgZoomRef.current.tx}px, ${imgZoomRef.current.ty}px)`,
          }}
          onClick={(e) => e.stopPropagation()}
          onWheel={(e) => {
            e.preventDefault()
            const s = imgZoomRef.current
            const delta = e.deltaY > 0 ? -0.1 : 0.1
            s.scale = Math.max(0.5, Math.min(5, s.scale + delta))
            if (s.scale <= 1) { s.tx = 0; s.ty = 0 }
            setImgTick(t => t + 1)
          }}
          onMouseDown={(e) => {
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
      )}
    </div>
  )
}
