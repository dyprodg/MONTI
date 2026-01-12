import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { TimeDisplay } from './TimeDisplay'
import { TimeMessage } from '../types'

describe('TimeDisplay', () => {
  it('should render "Waiting for data..." when no data', () => {
    render(<TimeDisplay data={null} />)
    expect(screen.getByText('Waiting for data...')).toBeInTheDocument()
  })

  it('should render time data when provided', () => {
    const mockData: TimeMessage = {
      timestamp: '2026-01-12T10:00:00Z',
      serverTime: 1736676000,
    }

    render(<TimeDisplay data={mockData} />)

    expect(screen.getByText('Server Time')).toBeInTheDocument()
    expect(screen.getByText(/Unix:/)).toBeInTheDocument()
    expect(screen.getByText(/1736676000/)).toBeInTheDocument()
  })

  it('should format timestamp correctly', () => {
    const mockData: TimeMessage = {
      timestamp: '2026-01-12T15:30:45Z',
      serverTime: 1736695845,
    }

    render(<TimeDisplay data={mockData} />)

    // Unix timestamp should be displayed
    expect(screen.getByText('Unix: 1736695845')).toBeInTheDocument()
  })

  it('should handle invalid timestamp gracefully', () => {
    const mockData: TimeMessage = {
      timestamp: 'invalid-date',
      serverTime: 0,
    }

    render(<TimeDisplay data={mockData} />)

    // Should still render without crashing
    expect(screen.getByText('Server Time')).toBeInTheDocument()
  })
})
