import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ConnectionStatus } from './ConnectionStatus'
import { ConnectionState } from '../types'

describe('ConnectionStatus', () => {
  it('should render "Connected" when state is OPEN', () => {
    render(<ConnectionStatus state={ConnectionState.OPEN} />)
    expect(screen.getByText('Connected')).toBeInTheDocument()
  })

  it('should render "Connecting..." when state is CONNECTING', () => {
    render(<ConnectionStatus state={ConnectionState.CONNECTING} />)
    expect(screen.getByText('Connecting...')).toBeInTheDocument()
  })

  it('should render "Error" when state is ERROR', () => {
    render(<ConnectionStatus state={ConnectionState.ERROR} />)
    expect(screen.getByText('Error')).toBeInTheDocument()
  })

  it('should render "Disconnected" when state is CLOSED', () => {
    render(<ConnectionStatus state={ConnectionState.CLOSED} />)
    expect(screen.getByText('Disconnected')).toBeInTheDocument()
  })

  it('should have correct color for OPEN state', () => {
    const { container } = render(<ConnectionStatus state={ConnectionState.OPEN} />)
    const indicator = container.querySelector('div[style*="width: 8px"]') as HTMLElement
    expect(indicator?.style.backgroundColor).toBe('rgb(34, 197, 94)')
  })

  it('should have correct color for ERROR state', () => {
    const { container } = render(<ConnectionStatus state={ConnectionState.ERROR} />)
    const indicator = container.querySelector('div[style*="width: 8px"]') as HTMLElement
    expect(indicator?.style.backgroundColor).toBe('rgb(239, 68, 68)')
  })
})
