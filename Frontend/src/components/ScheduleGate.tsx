import { useState, useEffect, ReactNode } from 'react'
import { useTheme } from '../contexts/ThemeContext'
import { ThemeToggle } from './ThemeToggle'

const SCHEDULE = {
  startHour: 14,
  endHour: 16,
  timezone: 'Europe/Berlin',
  days: [1, 2, 3, 4, 5], // Mon-Fri
} as const

function getBerlinTime(): Date {
  return new Date(new Date().toLocaleString('en-US', { timeZone: SCHEDULE.timezone }))
}

function isWithinWindow(): boolean {
  const now = getBerlinTime()
  const day = now.getDay()
  const hour = now.getHours()
  const minutes = now.getMinutes()
  const timeDecimal = hour + minutes / 60

  return SCHEDULE.days.includes(day) && timeDecimal >= SCHEDULE.startHour && timeDecimal < SCHEDULE.endHour
}

function getNextSessionText(): string {
  const now = getBerlinTime()
  const day = now.getDay()
  const hour = now.getHours()

  // If it's a weekday and before the window, session is today
  if (SCHEDULE.days.includes(day) && hour < SCHEDULE.startHour) {
    return 'Today at 2:00 PM CET'
  }

  // Otherwise, find the next weekday
  let daysUntil = 1
  let nextDay = (day + 1) % 7
  while (!SCHEDULE.days.includes(nextDay)) {
    daysUntil++
    nextDay = (nextDay + 1) % 7
  }

  const dayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']

  if (daysUntil === 1) {
    return `Tomorrow (${dayNames[nextDay]}) at 2:00 PM CET`
  }
  return `${dayNames[nextDay]} at 2:00 PM CET`
}

export const ScheduleGate = ({ children }: { children: ReactNode }) => {
  const [isOnline, setIsOnline] = useState(isWithinWindow)
  const { colors } = useTheme()

  useEffect(() => {
    const interval = setInterval(() => {
      setIsOnline(isWithinWindow())
    }, 30_000)
    return () => clearInterval(interval)
  }, [])

  if (isOnline) {
    return <>{children}</>
  }

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '100vh',
        backgroundColor: colors.background,
        padding: '24px',
      }}
    >
      <div style={{ position: 'absolute', top: '16px', right: '16px' }}>
        <ThemeToggle />
      </div>

      <div
        style={{
          backgroundColor: colors.surface,
          padding: '48px',
          borderRadius: '12px',
          boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
          textAlign: 'center',
          maxWidth: '520px',
          width: '100%',
        }}
      >
        <h1
          style={{
            fontSize: '36px',
            fontWeight: '700',
            marginBottom: '8px',
            color: colors.text,
          }}
        >
          MONTI
        </h1>
        <p
          style={{
            color: colors.textSecondary,
            marginBottom: '32px',
            fontSize: '14px',
          }}
        >
          Live Call Center Monitoring
        </p>

        <div
          style={{
            width: '48px',
            height: '48px',
            borderRadius: '50%',
            backgroundColor: colors.highlightBg,
            border: `2px solid ${colors.highlightBorder}`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            margin: '0 auto 24px',
          }}
        >
          <svg
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            stroke={colors.primary}
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <circle cx="12" cy="12" r="10" />
            <polyline points="12 6 12 12 16 14" />
          </svg>
        </div>

        <p
          style={{
            color: colors.text,
            fontSize: '15px',
            lineHeight: '1.6',
            marginBottom: '24px',
          }}
        >
          Due to simulation costs and infrequent access, I decided to keep costs low. The simulation
          runs automatically Monday to Friday, 2:00 PM – 4:00 PM (CET).
        </p>

        <div
          style={{
            backgroundColor: colors.highlightBg,
            border: `1px solid ${colors.highlightBorder}`,
            borderRadius: '8px',
            padding: '12px 16px',
            marginBottom: '32px',
          }}
        >
          <p style={{ color: colors.textSecondary, fontSize: '12px', marginBottom: '4px' }}>
            Next session
          </p>
          <p style={{ color: colors.text, fontSize: '16px', fontWeight: '600' }}>
            {getNextSessionText()}
          </p>
        </div>

        <div
          style={{
            display: 'flex',
            gap: '16px',
            justifyContent: 'center',
            flexWrap: 'wrap',
          }}
        >
          <a
            href="https://www.linkedin.com/in/dennisdiepolder"
            target="_blank"
            rel="noopener noreferrer"
            style={{
              color: colors.primary,
              fontSize: '14px',
              textDecoration: 'none',
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill={colors.primary}>
              <path d="M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433a2.062 2.062 0 0 1-2.063-2.065 2.064 2.064 0 1 1 2.063 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z" />
            </svg>
            LinkedIn
          </a>
          <a
            href="https://github.com/ddiepolder/MONTI"
            target="_blank"
            rel="noopener noreferrer"
            style={{
              color: colors.primary,
              fontSize: '14px',
              textDecoration: 'none',
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill={colors.primary}>
              <path d="M12 0C5.374 0 0 5.373 0 12c0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23A11.509 11.509 0 0 1 12 5.803c1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576C20.566 21.797 24 17.3 24 12c0-6.627-5.373-12-12-12z" />
            </svg>
            GitHub
          </a>
        </div>
      </div>
    </div>
  )
}
