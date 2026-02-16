export type LoginResponse = {
  token: string
  session_id: string
  user: {
    id: string
    username: string
    roles: string[]
  }
  expires_at: string
}

export type SQLProfile = {
  id: string
  name: string
  db_type: 'mysql' | 'mssql' | 'pgsql'
  host: string
  port: number
  username: string
  database: string
  commands: string
  use_ssl: boolean
  created_at: string
  modified_at: string
}

export type SessionView = {
  id: string
  user_id: string
  username: string
  roles: string[]
  created_at: string
  expires_at: string
}

export type MigrationFile = {
  name: string
  checksum: string
}

export type MigrationStatus = {
  name: string
  checksum: string
  applied: boolean
  applied_at?: string
}
