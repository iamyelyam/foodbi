import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { Header } from '@/components/layout/Header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Snackbar } from '@/components/ui/snackbar'
import api from '@/lib/api'
import { useCurrency } from '@/stores/app'
import { useT } from '@/i18n'

export function EditInvoicePage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const t = useT()
  const cs = useCurrency()
  const [supplier, setSupplier] = useState('')
  const [totalAmount, setTotalAmount] = useState('')
  const [invoiceDate, setInvoiceDate] = useState('')
  const [invoiceNumber, setInvoiceNumber] = useState('')
  const [notes, setNotes] = useState('')
  const [showSuccess, setShowSuccess] = useState(false)

  const mutation = useMutation({
    mutationFn: (data: {
      supplier: string
      total_amount: string
      invoice_date: string
      invoice_number: string
      notes: string
    }) => api.put(`/files/${id}/invoice`, data),
    onSuccess: () => {
      setShowSuccess(true)
      setTimeout(() => navigate('/file-upload'), 1500)
    },
  })

  const handleSave = () => {
    mutation.mutate({
      supplier,
      total_amount: totalAmount,
      invoice_date: invoiceDate,
      invoice_number: invoiceNumber,
      notes,
    })
  }

  return (
    <div className="flex flex-col min-h-dvh bg-white">
      <Header title={t('editInvoice.title')} showBack />

      <div className="flex flex-col flex-1 px-4 pt-4 gap-4">
        <Input
          label={t('editInvoice.supplierLabel')}
          placeholder={t('editInvoice.supplierPh')}
          value={supplier}
          onChange={(e) => setSupplier(e.target.value)}
        />

        <Input
          label={t('editInvoice.totalAmountLabel')}
          placeholder={`0${cs}`}
          value={totalAmount}
          onChange={(e) => setTotalAmount(e.target.value)}
        />

        <Input
          label={t('editInvoice.invoiceDateLabel')}
          type="date"
          value={invoiceDate}
          onChange={(e) => setInvoiceDate(e.target.value)}
        />

        <Input
          label={t('editInvoice.invoiceNumberLabel')}
          placeholder={t('editInvoice.invoiceNumberPh')}
          value={invoiceNumber}
          onChange={(e) => setInvoiceNumber(e.target.value)}
        />

        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium text-gray">{t('editInvoice.notesLabel')}</label>
          <textarea
            className="min-h-[100px] w-full rounded-[12px] border border-bg-alt bg-white px-4 py-3 text-base text-dark placeholder:text-gray-light focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary resize-none"
            placeholder={t('editInvoice.notesPh')}
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
          />
        </div>

        {mutation.isError && (
          <p className="text-sm text-danger text-center">{t('editInvoice.saveFailed')}</p>
        )}

        <div className="mt-auto pb-8">
          <Button fullWidth onClick={handleSave} disabled={mutation.isPending || !supplier}>
            {mutation.isPending ? t('common.saving') : t('editInvoice.saveBtn')}
          </Button>
        </div>
      </div>

      <Snackbar
        isOpen={showSuccess}
        onClose={() => setShowSuccess(false)}
        message={t('editInvoice.savedSuccess')}
        type="success"
      />
    </div>
  )
}
