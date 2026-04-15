import { AreaChart, Area, XAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import { useCurrency } from '@/stores/app'

interface DataPoint {
  date: string
  revenue: number
  orders?: number
  transactions?: number
}

interface RevenueChartProps {
  data: DataPoint[]
  height?: number
  /** Custom formatter for tooltip value (e.g. for non-money metrics like Orders/MI/T). Defaults to money. */
  valueFormatter?: (v: number) => string
}

export function RevenueChart({ data, height = 200, valueFormatter }: RevenueChartProps) {
  const cs = useCurrency()
  const formatted = data.map((d) => ({
    ...d,
    label: new Date(d.date).toLocaleDateString('en', { month: 'short', day: 'numeric' }),
  }))

  const defaultMoneyFmt = (v: number) =>
    `${Number(v).toLocaleString('ru-KZ', { maximumFractionDigits: 0 })}${cs}`
  const fmt = valueFormatter ?? defaultMoneyFmt

  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={formatted} margin={{ top: 5, right: 0, left: 0, bottom: 0 }}>
        <defs>
          <linearGradient id="revGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#6ADEBF" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#6ADEBF" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="#F1F2F7" vertical={false} />
        <XAxis dataKey="label" tick={{ fontSize: 10, fill: '#A4A2B7' }} />
        <Tooltip
          content={({ active, payload, label }) => {
            if (!active || !payload || !payload.length) return null
            const p = payload[0].payload as DataPoint & { label?: string }
            const txn = p.transactions ?? p.orders
            return (
              <div
                style={{
                  background: '#1F2125',
                  color: '#FFF',
                  borderRadius: 12,
                  padding: '8px 12px',
                  fontSize: 12,
                  boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
                }}
              >
                <div style={{ fontSize: 10, opacity: 0.7, marginBottom: 2 }}>{label}</div>
                <div style={{ fontWeight: 700 }}>{fmt(p.revenue)}</div>
                {txn !== undefined && txn !== null && (
                  <div style={{ opacity: 0.8, marginTop: 2 }}>
                    {txn} {txn === 1 ? 'transaction' : 'transactions'}
                  </div>
                )}
              </div>
            )
          }}
        />
        <Area type="monotone" dataKey="revenue" stroke="#6ADEBF" fill="url(#revGradient)" strokeWidth={2} />
      </AreaChart>
    </ResponsiveContainer>
  )
}
