import { request } from './client'
import type { MigrationFile, MigrationStatus } from '../types/api'

export function listMigrationFiles(token: string) {
  return request<{ items: MigrationFile[] }>('/v1/system/migrations', { method: 'GET' }, token)
}

export function listMigrationStatus(token: string) {
  return request<{ items: MigrationStatus[] }>('/v1/system/migrations/status', { method: 'GET' }, token)
}

export function applyMigration(token: string, name: string) {
  return request<{ status: string; name: string }>(
    `/v1/system/migrations/${encodeURIComponent(name)}/apply`,
    { method: 'POST' },
    token
  )
}
