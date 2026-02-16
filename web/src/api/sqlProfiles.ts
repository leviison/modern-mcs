import { request } from './client'
import type { SQLProfile } from '../types/api'

export type SQLProfileInput = {
  name: string
  db_type: 'mysql' | 'mssql' | 'pgsql'
  host: string
  port: number
  username: string
  database: string
  commands: string
  use_ssl: boolean
}

export function listSQLProfiles(token: string) {
  return request<{ items: SQLProfile[] }>('/v1/sql-profiles', { method: 'GET' }, token)
}

export function createSQLProfile(token: string, payload: SQLProfileInput) {
  return request<SQLProfile>(
    '/v1/sql-profiles',
    {
      method: 'POST',
      body: JSON.stringify(payload)
    },
    token
  )
}

export function updateSQLProfile(token: string, id: string, payload: SQLProfileInput) {
  return request<SQLProfile>(
    `/v1/sql-profiles/${encodeURIComponent(id)}`,
    {
      method: 'PUT',
      body: JSON.stringify(payload)
    },
    token
  )
}

export function deleteSQLProfile(token: string, id: string) {
  return request<void>(`/v1/sql-profiles/${encodeURIComponent(id)}`, { method: 'DELETE' }, token)
}
