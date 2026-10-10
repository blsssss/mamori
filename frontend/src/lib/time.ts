const pad = (n: number) => String(n).padStart(2, '0')

// Go marshals time.Time with up to nine fractional digits, Date.parse is only sure of three
export function parseTime(s: string): Date | null {
  const d = new Date(s.replace(/(\.\d{3})\d+/, '$1'))
  return Number.isNaN(d.getTime()) ? null : d
}

// local time as YYYY-MM-DD HH:MM:SS, the text itself when it is not a date
export function formatTime(s: string): string {
  const d = parseTime(s)
  if (!d) return s
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}
