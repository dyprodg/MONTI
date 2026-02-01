import { createContext, useContext, useEffect, useState, ReactNode } from 'react'
import { authService, User } from '../services/auth'

interface AuthContextType {
  user: User | null
  loading: boolean
  isAuthenticated: boolean
  login: () => Promise<void>
  logout: () => Promise<void>
  getToken: () => Promise<string | null>
  refreshUser: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const initAuth = async () => {
      try {
        const currentUser = await authService.getUser()
        setUser(currentUser)
      } catch (error) {
        console.error('Auth init error:', error)
      } finally {
        setLoading(false)
      }
    }

    initAuth()
  }, [])

  const login = async () => {
    await authService.login()
  }

  const logout = async () => {
    await authService.logout()
    setUser(null)
  }

  const getToken = async () => {
    return await authService.getToken()
  }

  const refreshUser = async () => {
    try {
      const currentUser = await authService.getUser()
      setUser(currentUser)
    } catch (error) {
      console.error('Refresh user error:', error)
    }
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        isAuthenticated: user !== null,
        login,
        logout,
        getToken,
        refreshUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export const useAuth = () => {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return context
}
