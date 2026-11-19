import { describe, expect, it } from 'vitest'
import { exactRangeBounds } from '../exactRange'

const iso = (local: string) => new Date(local).toISOString()

describe('exactRangeBounds', () => {
  it('returns nothing for whole-day ranges', () => {
    expect(exactRangeBounds('2026-10-12', '2026-10-12', '', '')).toEqual({})
  })

  it('uses both times as local instants', () => {
    expect(exactRangeBounds('2026-10-12', '2026-10-12', '12:30', '13:30')).toEqual({
      from: iso('2026-10-12T12:30:00'),
      to: iso('2026-10-12T13:30:00')
    })
  })

  it('starts at midnight when only the end time is set', () => {
    expect(exactRangeBounds('2026-10-12', '2026-10-13', '', '06:00')).toEqual({
      from: iso('2026-10-12T00:00:00'),
      to: iso('2026-10-13T06:00:00')
    })
  })

  it('runs to the end of the end date when only the start time is set', () => {
    expect(exactRangeBounds('2026-10-12', '2026-10-12', '18:00', '')).toEqual({
      from: iso('2026-10-12T18:00:00'),
      to: iso('2026-10-13T00:00:00')
    })
  })

  it('returns nothing for an unparsable date', () => {
    expect(exactRangeBounds('', '2026-10-12', '12:00', '')).toEqual({})
  })
})
