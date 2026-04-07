import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'

interface DataPoint {
  date: string
  revenue: number
  cost: number
  profit: number
}

interface ProfitChartProps {
  data: DataPoint[]
  height?: number
}

export function ProfitChart({ data, height = 200 }: ProfitChartProps) {
  const formatted = data.map((d) => ({
    ...d,
    label: new Date(d.date).toLocaleDateString('en', { month: 'short', day: 'numeric' }),
  }))

  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={formatted} margin={{ top: 5, right: 0, left: -20, bottom: 0 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="#F1F2F7" />
        <XAxis dataKey="label" tick={{ fontSize: 10, fill: '#A4A2B7' }} />
        <YAxis tick={{ fontSize: 10, fill: '#A4A2B7' }} />
        <Tooltip
          contentStyle={{ borderRadius: 12, border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }}
          formatter={(value) => `$${Number(value).toFixed(2)}`}
        />
        <Legend wrapperStyle={{ fontSize: 11 }} />
        <Bar dataKey="revenue" fill="#6ADEBF" radius={[4, 4, 0, 0]} name="Revenue" />
        <Bar dataKey="cost" fill="#EF8F00" radius={[4, 4, 0, 0]} name="Cost" />
      </BarChart>
    </ResponsiveContainer>
  )
}
