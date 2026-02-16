import { createContext, useCallback, useContext, useMemo, useState } from 'react'
import type { ReactNode } from 'react'

type AuthState = {
  token: string | null
  sessionId: string | null
  username: string | null
  roles: string[]
}

type AuthContextValue = AuthState & {
  isAuthenticated: boolean
  setAuth: (input: { token: string; sessionId: string; username: string; roles: string[] }) => void
  clearAuth: () => void
}

const STORAGE_KEY = 'modern-mcs-auth'

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

function loadInitialState(): AuthState {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return { token: null, sessionId: null, username: null, roles: [] }
    const parsed = JSON.parse(raw)
    return {
      token: parsed.token || null,
      sessionId: parsed.sessionId || null,
      username: parsed.username || null,
      roles: Array.isArray(parsed.roles) ? parsed.roles : []
    }
  } catch {
    return { token: null, sessionId: null, username: null, roles: [] }
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>(loadInitialState)

  const setAuth = useCallback((input: { token: string; sessionId: string; username: string; roles: string[] }) => {
    const next: AuthState = {
      token: input.token,
      sessionId: input.sessionId,
      username: input.username,
      roles: input.roles
    }
    setState(next)
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
  }, [])

  const clearAuth = useCallback(() => {
    setState({ token: null, sessionId: null, username: null, roles: [] })
    localStorage.removeItem(STORAGE_KEY)
  }, [])

  const value = useMemo(
    () => ({
      ...state,
      isAuthenticated: Boolean(state.token),
      setAuth,
      clearAuth
    }),
    [state, setAuth, clearAuth]
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
