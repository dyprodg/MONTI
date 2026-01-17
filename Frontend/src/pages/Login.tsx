import { useEffect } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useTheme } from '../contexts/ThemeContext'
import { useNavigate } from 'react-router-dom'

export const Login = () => {
  const { login, isAuthenticated } = useAuth()
  const { colors } = useTheme()
  const navigate = useNavigate()

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/')
    }
  }, [isAuthenticated, navigate])

  const handleLogin = async () => {
    await login()
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
      }}
    >
      <div
        style={{
          backgroundColor: colors.surface,
          padding: '48px',
          borderRadius: '12px',
          boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
          textAlign: 'center',
          maxWidth: '400px',
        }}
      >
        <h1
          style={{
            fontSize: '36px',
            fontWeight: '700',
            marginBottom: '16px',
            color: colors.text,
          }}
        >
          MONTI
        </h1>
        <p
          style={{
            color: colors.textSecondary,
            marginBottom: '32px',
            fontSize: '16px',
          }}
        >
          Live Call Center Monitoring
        </p>

        <button
          onClick={handleLogin}
          style={{
            backgroundColor: colors.primary,
            color: 'white',
            padding: '12px 32px',
            borderRadius: '8px',
            border: 'none',
            fontSize: '16px',
            fontWeight: '600',
            cursor: 'pointer',
            width: '100%',
            transition: 'background-color 0.2s',
          }}
          onMouseOver={(e) => {
            e.currentTarget.style.backgroundColor = colors.primaryHover
          }}
          onMouseOut={(e) => {
            e.currentTarget.style.backgroundColor = colors.primary
          }}
        >
          Sign in with SSO
        </button>

        <p
          style={{
            marginTop: '24px',
            fontSize: '12px',
            color: colors.textSecondary,
          }}
        >
          Development Mode
        </p>
      </div>
    </div>
  )
}
