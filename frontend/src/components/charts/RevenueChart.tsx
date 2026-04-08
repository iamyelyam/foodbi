import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import { useCurrency } from '@/stores/app'

interface DataPoint {
  date: string
  revenue: number
  orders?: number
}

interface RevenueChartProps {
  data: DataPoint[]
  height?: number
}

export function RevenueChart({ data, height = 200 }: RevenueChartProps) {
  const cs = useCurrency()
  const formatted = data.map((d) => ({
    ...d,
    label: new Date(d.date).toLocaleDateString('en', { month: 'short', day: 'numeric' }),
  }))

  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={formatted} margin={{ top: 5, right: 0, left: -20, bottom: 0 }}>
        <defs>
          <linearGradient id="revGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#6ADEBF" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#6ADEBF" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="#F1F2F7" />
        <XAxis dataKey="label" tick={{ fontSize: 10, fill: '#A4A2B7' }} />
        <YAxis tick={{ fontSize: 10, fill: '#A4A2B7' }} />
        <Tooltip
          contentStyle={{ borderRadius: 12, border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }}
          formatter={(value) => [`${Number(value).toFixed(2)}${cs}`, 'Revenue']}
        />
        <Area type="monotone" dataKey="revenue" stroke="#6ADEBF" fill="url(#revGradient)" strokeWidth={2} />
      </AreaChart>
    </ResponsiveContainer>
  )
}
