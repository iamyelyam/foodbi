import { useRef, useState, useCallback } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ListItemSkeleton } from '@/components/ui/skeleton'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { PermissionModal } from '@/components/ui/permission-modal'
import { ProgressBar } from '@/components/ui/progress-bar'
import { CameraScanner } from '@/components/camera/CameraScanner'
import { Upload, Camera, FileText, Image, File, Share2, Trash2, Eye } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useUnreadNotificationCount } from '@/hooks/useApi'
import api from '@/lib/api'
import { useT, useI18nStore } from '@/i18n'

const mimeIcons: Record<string, typeof File> = {
  'application/pdf': FileText,
  'image/jpeg': Image,
  'image/png': Image,
}

function getPermission(key: string): boolean {
  return localStorage.getItem(key) === 'granted'
}

function setPermission(key: string) {
  localStorage.setItem(key, 'granted')
}

const ALLOWED_TYPES = ['application/pdf', 'image/jpeg', 'image/png']
const MAX_FILE_SIZE = 10 * 1024 * 1024 // 10 MB

export function FileUploadPage() {
  const queryClient = useQueryClient()
  const t = useT()
  const locale = useI18nStore((s) => s.locale)
  const { data: unreadCount = 0 } = useUnreadNotificationCount()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [uploadError, setUploadError] = useState('')

  // Permission modals
  const [cameraPermissionOpen, setCameraPermissionOpen] = useState(false)
  const [filePermissionOpen, setFilePermissionOpen] = useState(false)

  // Camera scanner
  const [cameraScannerOpen, setCameraScannerOpen] = useState(false)

  // Upload progress
  const [uploadProgress, setUploadProgress] = useState(0)

  // Action sheet
  const [selectedFile, setSelectedFile] = useState<any>(null)
  const [actionSheetOpen, setActionSheetOpen] = useState(false)
  const longPressTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  function validateFile(file: globalThis.File): string | null {
    if (!ALLOWED_TYPES.includes(file.type)) {
      return t('fileUpload.invalidType')
    }
    if (file.size > MAX_FILE_SIZE) {
      return t('fileUpload.fileTooBig')
    }
    return null
  }

  const { data: files = [], isLoading: filesLoading } = useQuery({
    queryKey: ['files'],
    queryFn: () => api.get('/files').then((r) => r.data),
  })

  const uploadMutation = useMutation({
    mutationFn: (file: globalThis.File) => {
      const formData = new FormData()
      formData.append('file', file)
      setUploadProgress(0)
      return api.post('/files/upload', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
        onUploadProgress: (progressEvent) => {
          const total = progressEvent.total ?? 0
          if (total > 0) {
            setUploadProgress(Math.round((progressEvent.loaded * 100) / total))
          }
        },
      })
    },
    onSuccess: () => {
      setUploadProgress(100)
      queryClient.invalidateQueries({ queryKey: ['files'] })
    },
    onError: () => {
      setUploadProgress(0)
    },
  })

  const submitFile = (file: globalThis.File) => {
    const error = validateFile(file)
    if (error) {
      setUploadError(error)
      return
    }
    setUploadError('')
    uploadMutation.mutate(file)
  }

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) submitFile(file)
  }

  const handleCameraCapture = useCallback(
    (file: globalThis.File) => {
      submitFile(file)
    },
    []
  )

  const handleScanInvoice = () => {
    if (getPermission('camera_permission')) {
      setCameraScannerOpen(true)
    } else {
      setCameraPermissionOpen(true)
    }
  }

  const handleUploadFile = () => {
    if (getPermission('file_permission')) {
      fileInputRef.current?.click()
    } else {
      setFilePermissionOpen(true)
    }
  }

  const handleCameraAllow = () => {
    setPermission('camera_permission')
    setCameraPermissionOpen(false)
    setCameraScannerOpen(true)
  }

  const handleFileAllow = () => {
    setPermission('file_permission')
    setFilePermissionOpen(false)
    fileInputRef.current?.click()
  }

  // Long-press handlers for file action sheet
  const handlePointerDown = (f: any) => {
    longPressTimerRef.current = setTimeout(() => {
      setSelectedFile(f)
      setActionSheetOpen(true)
    }, 500)
  }

  const handlePointerUp = () => {
    if (longPressTimerRef.current) {
      clearTimeout(longPressTimerRef.current)
      longPressTimerRef.current = null
    }
  }

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/files/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['files'] })
      setActionSheetOpen(false)
      setSelectedFile(null)
    },
  })

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  }

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title={t('fileUpload.pageTitle')} showBack showNotification badgeCount={unreadCount} />

      <main className="flex-1 px-4 pt-4 pb-28 space-y-4">
        {/* Upload actions */}
        <div className="grid grid-cols-2 gap-3">
          <button
            onClick={handleScanInvoice}
            className="bg-white rounded-[16px] p-6 shadow-sm flex flex-col items-center gap-3"
          >
            <div className="w-14 h-14 rounded-full bg-primary-lighter flex items-center justify-center">
              <Camera className="h-7 w-7 text-primary" />
            </div>
            <span className="text-sm font-medium text-dark">{t('fileUpload.scanInvoice')}</span>
          </button>

          <button
            onClick={handleUploadFile}
            className="bg-white rounded-[16px] p-6 shadow-sm flex flex-col items-center gap-3"
          >
            <div className="w-14 h-14 rounded-full bg-info-light flex items-center justify-center">
              <Upload className="h-7 w-7 text-info" />
            </div>
            <span className="text-sm font-medium text-dark">{t('fileUpload.uploadFile')}</span>
          </button>
        </div>

        <input
          ref={fileInputRef}
          type="file"
          className="hidden"
          accept=".pdf,.jpg,.jpeg,.png"
          onChange={handleFileSelect}
        />

        {/* Upload error */}
        {uploadError && (
          <div className="bg-danger/10 rounded-[12px] p-3 text-center">
            <p className="text-sm font-medium text-danger">{uploadError}</p>
          </div>
        )}

        {/* Upload progress */}
        {uploadMutation.isPending && (
          <div className="bg-white rounded-[12px] p-4 shadow-sm">
            <div className="flex items-center justify-between mb-2">
              <p className="text-sm font-medium text-dark">{t('fileUpload.uploading')}</p>
              <span className="text-xs text-gray">{uploadProgress}%</span>
            </div>
            <ProgressBar value={uploadProgress} className="h-1.5" />
          </div>
        )}

        {/* Files list */}
        <div>
          <h2 className="text-sm font-semibold text-dark mb-2">{t('fileUpload.filesCount', { count: files.length })}</h2>
          {filesLoading ? (
            <div className="space-y-2">
              <ListItemSkeleton />
              <ListItemSkeleton />
              <ListItemSkeleton />
            </div>
          ) : (
          <div className="space-y-2">
            {files.map((f: any) => {
              const Icon = mimeIcons[f.mime_type] || File
              return (
                <div
                  key={f.id}
                  className="bg-white rounded-[12px] p-4 shadow-sm select-none"
                  onPointerDown={() => handlePointerDown(f)}
                  onPointerUp={handlePointerUp}
                  onPointerCancel={handlePointerUp}
                  onContextMenu={(e) => e.preventDefault()}
                >
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-[8px] bg-bg-alt flex items-center justify-center">
                      <Icon className="h-5 w-5 text-gray" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-dark truncate">{f.filename}</p>
                      <p className="text-xs text-gray mt-0.5">
                        {formatSize(f.size)} - {new Date(f.created_at).toLocaleDateString(locale)}
                      </p>
                    </div>
                    <span
                      className={cn(
                        'text-xs px-2 py-0.5 rounded-full font-medium',
                        f.status === 'processed'
                          ? 'bg-success/10 text-success'
                          : f.status === 'processing'
                            ? 'bg-warning/10 text-warning'
                            : 'bg-bg-alt text-gray'
                      )}
                    >
                      {f.status}
                    </span>
                  </div>
                </div>
              )
            })}

            {files.length === 0 && (
              <div className="text-center py-12">
                <Upload className="h-12 w-12 text-gray-light mx-auto mb-3" />
                <p className="text-sm text-gray">{t('fileUpload.noFilesYet')}</p>
              </div>
            )}
          </div>
          )}
        </div>
      </main>

      <Tabbar />

      {/* Permission modals */}
      <PermissionModal
        isOpen={cameraPermissionOpen}
        type="camera"
        onAllow={handleCameraAllow}
        onDeny={() => setCameraPermissionOpen(false)}
      />
      <PermissionModal
        isOpen={filePermissionOpen}
        type="files"
        onAllow={handleFileAllow}
        onDeny={() => setFilePermissionOpen(false)}
      />

      {/* Camera scanner overlay */}
      <CameraScanner
        isOpen={cameraScannerOpen}
        onClose={() => setCameraScannerOpen(false)}
        onCapture={handleCameraCapture}
      />

      {/* File action sheet */}
      <BottomSheet
        isOpen={actionSheetOpen}
        onClose={() => setActionSheetOpen(false)}
        title={selectedFile?.filename}
      >
        <div className="space-y-1">
          <button
            onClick={() => { window.open(`/api/v1/files/${selectedFile.id}`, '_blank'); setActionSheetOpen(false) }}
            className="w-full flex items-center gap-3 px-2 py-3 rounded-[12px] hover:bg-bg-alt transition-colors"
          >
            <Eye className="h-5 w-5 text-gray" />
            <span className="text-sm font-medium text-dark">{t('fileUpload.viewAction')}</span>
          </button>
          <button
            onClick={() => { navigator.share({ title: selectedFile.filename, url: `/api/v1/files/${selectedFile.id}` }).catch(() => { navigator.clipboard.writeText(window.location.origin + `/api/v1/files/${selectedFile.id}`) }); setActionSheetOpen(false) }}
            className="w-full flex items-center gap-3 px-2 py-3 rounded-[12px] hover:bg-bg-alt transition-colors"
          >
            <Share2 className="h-5 w-5 text-gray" />
            <span className="text-sm font-medium text-dark">{t('fileUpload.shareAction')}</span>
          </button>
          <button
            onClick={() => deleteMutation.mutate(selectedFile.id)}
            className="w-full flex items-center gap-3 px-2 py-3 rounded-[12px] hover:bg-bg-alt transition-colors"
          >
            <Trash2 className="h-5 w-5 text-danger" />
            <span className="text-sm font-medium text-danger">{t('fileUpload.deleteAction')}</span>
          </button>
        </div>
      </BottomSheet>
    </div>
  )
}
