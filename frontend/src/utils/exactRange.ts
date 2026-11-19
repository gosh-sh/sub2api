/**
 * Turn a date range with optional HH:MM times into RFC3339 `from` / `to` bounds.
 *
 * Dates and times are read in the browser's local timezone. An empty start time means the start
 * of the start date; an empty end time means the end of the end date. The range is half-open:
 * `to` itself is not included. With neither time set nothing is returned, and callers keep
 * sending whole-day `start_date` / `end_date`.
 */
export function exactRangeBounds(
  startDate: string,
  endDate: string,
  startTime: string,
  endTime: string
): { from?: string; to?: string } {
  if (!startTime && !endTime) return {}
  const from = new Date(`${startDate}T${startTime || '00:00'}:00`)
  const to = new Date(`${endDate}T${endTime || '00:00'}:00`)
  if (!endTime) to.setDate(to.getDate() + 1)
  if (Number.isNaN(from.getTime()) || Number.isNaN(to.getTime())) return {}
  return { from: from.toISOString(), to: to.toISOString() }
}
