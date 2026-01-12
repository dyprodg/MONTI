import { useEffect } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useNavigate } from 'react-router-dom'

export const Login = () => {
  const { login, isAuthenticated } = useAuth()
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
        backgroundColor: '#f9fafb',
      }}
    >
      <div
        style={{
          backgroundColor: 'white',
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
            color: '#111827',
          }}
        >
          MONTI
        </h1>
        <p
          style={{
            color: '#6b7280',
            marginBottom: '32px',
            fontSize: '16px',
          }}
        >
          Live Call Center Monitoring
        </p>

        <button
          onClick={handleLogin}
          style={{
            backgroundColor: '#3b82f6',
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
            e.currentTarget.style.backgroundColor = '#2563eb'
          }}
          onMouseOut={(e) => {
            e.currentTarget.style.backgroundColor = '#3b82f6'
          }}
        >
          Sign in with SSO
        </button>

        <p
          style={{
            marginTop: '24px',
            fontSize: '12px',
            color: '#9ca3af',
          }}
        >
          Development Mode
        </p>
      </div>
    </div>
  )
}
