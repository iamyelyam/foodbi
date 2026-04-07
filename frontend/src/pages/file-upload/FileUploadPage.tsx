import { useRef } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { Upload, Camera, FileText, Image, File } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'

const mimeIcons: Record<string, typeof File> = {
  'application/pdf': FileText,
  'image/jpeg': Image,
  'image/png': Image,
}

export function FileUploadPage() {
  const queryClient = useQueryClient()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const cameraInputRef = useRef<HTMLInputElement>(null)

  const { data: files = [] } = useQuery({
    queryKey: ['files'],
    queryFn: () => api.get('/files').then((r) => r.data),
  })

  const uploadMutation = useMutation({
    mutationFn: (file: globalThis.File) => {
      const formData = new FormData()
      formData.append('file', file)
      return api.post('/files/upload', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['files'] }),
  })

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) uploadMutation.mutate(file)
  }

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  }

  return (
    <div className="flex flex-col min-h-dvh bg-bg">
      <Header title="File Upload" showBack showNotification />

      <main className="flex-1 px-4 pt-4 pb-20 space-y-4">
        {/* Upload actions */}
        <div className="grid grid-cols-2 gap-3">
          <button
            onClick={() => cameraInputRef.current?.click()}
            className="bg-white rounded-[16px] p-6 shadow-sm flex flex-col items-center gap-3"
          >
            <div className="w-14 h-14 rounded-full bg-primary-lighter flex items-center justify-center">
              <Camera className="h-7 w-7 text-primary" />
            </div>
            <span className="text-sm font-medium text-dark">Scan Invoice</span>
          </button>

          <button
            onClick={() => fileInputRef.current?.click()}
            className="bg-white rounded-[16px] p-6 shadow-sm flex flex-col items-center gap-3"
          >
            <div className="w-14 h-14 rounded-full bg-info-light flex items-center justify-center">
              <Upload className="h-7 w-7 text-info" />
            </div>
            <span className="text-sm font-medium text-dark">Upload File</span>
          </button>
        </div>

        <input ref={fileInputRef} type="file" className="hidden" accept=".pdf,.jpg,.jpeg,.png" onChange={handleFileSelect} />
        <input ref={cameraInputRef} type="file" className="hidden" accept="image/*" capture="environment" onChange={handleFileSelect} />

        {uploadMutation.isPending && (
          <div className="bg-primary-lighter rounded-[12px] p-4 text-center">
            <p className="text-sm font-medium text-primary">Uploading...</p>
          </div>
        )}

        {/* Files list */}
        <div>
          <h2 className="text-sm font-semibold text-dark mb-2">{files.length} files</h2>
          <div className="space-y-2">
            {files.map((f: any) => {
              const Icon = mimeIcons[f.mime_type] || File
              return (
                <div key={f.id} className="bg-white rounded-[12px] p-4 shadow-sm">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-[8px] bg-bg-alt flex items-center justify-center">
                      <Icon className="h-5 w-5 text-gray" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-dark truncate">{f.filename}</p>
                      <p className="text-xs text-gray mt-0.5">
                        {formatSize(f.size)} - {new Date(f.created_at).toLocaleDateString()}
                      </p>
                    </div>
                    <span className={cn(
                      'text-xs px-2 py-0.5 rounded-full font-medium',
                      f.status === 'processed' ? 'bg-success/10 text-success' :
                      f.status === 'processing' ? 'bg-warning/10 text-warning' : 'bg-bg-alt text-gray'
                    )}>
                      {f.status}
                    </span>
                  </div>
                </div>
              )
            })}

            {files.length === 0 && (
              <div className="text-center py-8">
                <p className="text-sm text-gray">No files uploaded yet</p>
              </div>
            )}
          </div>
        </div>
      </main>

      <Tabbar />
    </div>
  )
}
