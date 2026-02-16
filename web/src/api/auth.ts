import { request } from './client'
import type { LoginResponse, SessionView } from '../types/api'

export function login(username: string, password: string) {
  return request<LoginResponse>('/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password })
  })
}

export function me(token: string) {
  return request('/v1/auth/me', { method: 'GET' }, token)
}

export function logout(token: string) {
  return request<void>('/v1/auth/logout', { method: 'POST' }, token)
}

export function changePassword(token: string, currentPassword: string, newPassword: string) {
  return request<void>(
    '/v1/auth/change-password',
    {
      method: 'POST',
      body: JSON.stringify({ current_password: currentPassword, new_password: newPassword })
    },
    token
  )
}

export function listSessions(token: string) {
  return request<{ items: SessionView[] }>('/v1/system/sessions', { method: 'GET' }, token)
}

export function revokeSession(token: string, sessionId: string) {
  return request<void>(`/v1/system/sessions/${encodeURIComponent(sessionId)}`, { method: 'DELETE' }, token)
}
