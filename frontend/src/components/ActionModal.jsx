import Modal from './Modal'

/**
 * ActionModal wraps Modal for action content.
 * Props: open, onClose, title, children
 */
export default function ActionModal({ open, onClose, title, children }) {
  return (
    <Modal open={open} onClose={onClose} title={title}>
      {children}
    </Modal>
  )
}

