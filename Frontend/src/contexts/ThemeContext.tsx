import { createContext, useContext, useState, useEffect, ReactNode } from 'react'

export type Theme = 'light' | 'dark'

export interface ThemeColors {
  background: string
  surface: string
  surfaceHover: string
  text: string
  textSecondary: string
  border: string
  // Semantic colors (same in both themes)
  primary: string
  primaryHover: string
  error: string
  errorBg: string
  errorBorder: string
  // KPI highlight
  highlightBg: string
  highlightBorder: string
}

const themes: Record<Theme, ThemeColors> = {
  light: {
    background: '#f9fafb',
    surface: '#ffffff',
    surfaceHover: '#f3f4f6',
    text: '#111827',
    textSecondary: '#6b7280',
    border: '#e5e7eb',
    primary: '#3b82f6',
    primaryHover: '#2563eb',
    error: '#991b1b',
    errorBg: '#fef2f2',
    errorBorder: '#fecaca',
    highlightBg: '#f0f9ff',
    highlightBorder: '#bae6fd',
  },
  dark: {
    background: '#111827',
    surface: '#1f2937',
    surfaceHover: '#374151',
    text: '#f9fafb',
    textSecondary: '#9ca3af',
    border: '#374151',
    primary: '#3b82f6',
    primaryHover: '#2563eb',
    error: '#fca5a5',
    errorBg: '#450a0a',
    errorBorder: '#7f1d1d',
    highlightBg: '#1e3a5f',
    highlightBorder: '#2563eb',
  },
}

interface ThemeContextType {
  theme: Theme
  colors: ThemeColors
  toggleTheme: () => void
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

const STORAGE_KEY = 'monti-theme'

export const ThemeProvider = ({ children }: { children: ReactNode }) => {
  const [theme, setTheme] = useState<Theme>(() => {
    // Check localStorage first
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'light' || stored === 'dark') {
      return stored
    }
    // Respect system preference
    if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
      return 'dark'
    }
    return 'light'
  })

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, theme)
  }, [theme])

  const toggleTheme = () => {
    setTheme((prev) => (prev === 'light' ? 'dark' : 'light'))
  }

  const value: ThemeContextType = {
    theme,
    colors: themes[theme],
    toggleTheme,
  }

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export const useTheme = (): ThemeContextType => {
  const context = useContext(ThemeContext)
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider')
  }
  return context
}
