import { useRef } from 'react'
import { X, ImageIcon } from 'lucide-react'

interface CameraScannerProps {
  isOpen: boolean
  onClose: () => void
  onCapture: (file: File) => void
}

export function CameraScanner({ isOpen, onClose, onCapture }: CameraScannerProps) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  const galleryInputRef = useRef<HTMLInputElement>(null)

  if (!isOpen) return null

  const handleCapture = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      onCapture(file)
      onClose()
    }
  }

  return (
    <div className="fixed inset-0 z-50 bg-black flex flex-col">
      {/* Top bar */}
      <div className="flex items-center justify-between px-4 pt-safe-top h-14">
        <button onClick={onClose} className="w-10 h-10 flex items-center justify-center">
          <X className="h-6 w-6 text-white" />
        </button>
        <span className="text-white/70 text-sm font-medium">Scan Invoice</span>
        <div className="w-10" />
      </div>

      {/* Viewfinder area */}
      <div className="flex-1 flex flex-col items-center justify-center px-12">
        <div className="relative w-[280px] h-[380px]">
          {/* Dashed border viewfinder */}
          <div className="absolute inset-0 border-2 border-dashed border-white/70 rounded-[16px]" />

          {/* Corner accents */}
          <div className="absolute top-0 left-0 w-8 h-8 border-t-[3px] border-l-[3px] border-white rounded-tl-[16px]" />
          <div className="absolute top-0 right-0 w-8 h-8 border-t-[3px] border-r-[3px] border-white rounded-tr-[16px]" />
          <div className="absolute bottom-0 left-0 w-8 h-8 border-b-[3px] border-l-[3px] border-white rounded-bl-[16px]" />
          <div className="absolute bottom-0 right-0 w-8 h-8 border-b-[3px] border-r-[3px] border-white rounded-br-[16px]" />
        </div>

        <p className="text-white/70 text-sm mt-6">Position invoice within the frame</p>
      </div>

      {/* Bottom bar */}
      <div className="flex items-center justify-between px-10 pb-safe-bottom h-28">
        {/* Gallery picker */}
        <button
          onClick={() => galleryInputRef.current?.click()}
          className="w-12 h-12 rounded-[10px] border-2 border-white/40 flex items-center justify-center"
        >
          <ImageIcon className="h-6 w-6 text-white/80" />
        </button>

        {/* Shutter button — triggers OS camera via capture="environment" */}
        <button
          onClick={() => fileInputRef.current?.click()}
          className="w-[72px] h-[72px] rounded-full border-[4px] border-white flex items-center justify-center"
        >
          <div className="w-[60px] h-[60px] rounded-full bg-white" />
        </button>

        {/* Spacer for symmetry */}
        <div className="w-12 h-12" />
      </div>

      {/* Hidden file inputs */}
      <input
        ref={fileInputRef}
        type="file"
        className="hidden"
        accept="image/*"
        capture="environment"
        onChange={handleCapture}
      />
      <input
        ref={galleryInputRef}
        type="file"
        className="hidden"
        accept="image/*"
        onChange={handleCapture}
      />
    </div>
  )
}
