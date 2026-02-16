import { useEffect, useState } from 'react'
import { applyMigration, listMigrationStatus } from '../api/migrations'
import { getErrorMessage } from '../api/errors'
import type { MigrationStatus } from '../types/api'
import { useAuth } from '../context/AuthContext'

export function MigrationsPage() {
  const auth = useAuth()
  const [items, setItems] = useState<MigrationStatus[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [busyMigration, setBusyMigration] = useState<string | null>(null)

  async function refresh() {
    if (!auth.token) return
    setError(null)
    setLoading(true)
    try {
      const res = await listMigrationStatus(auth.token)
      setItems(res.items)
    } catch (err) {
      setError(getErrorMessage(err, 'Failed to load migrations'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [auth.token])

  async function handleApply(name: string) {
    if (!auth.token) return
    setError(null)
    setBusyMigration(name)
    try {
      await applyMigration(auth.token, name)
      await refresh()
    } catch (err) {
      setError(getErrorMessage(err, 'Failed to apply migration'))
    } finally {
      setBusyMigration(null)
    }
  }

  return (
    <main className="page with-nav">
      <section className="panel">
        <h1>Migrations</h1>
        <p>Review migration status and mark unapplied files as applied.</p>
        {error && <div className="error">{error}</div>}
        {loading && <p>Loading migrations...</p>}
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Checksum</th>
              <th>Applied</th>
              <th>Applied At</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.name}>
                <td>{item.name}</td>
                <td className="mono">{item.checksum.slice(0, 16)}...</td>
                <td>{item.applied ? 'Yes' : 'No'}</td>
                <td>{item.applied_at ? new Date(item.applied_at).toLocaleString() : '-'}</td>
                <td>
                  {!item.applied ? (
                    <button disabled={busyMigration === item.name} onClick={() => void handleApply(item.name)}>
                      {busyMigration === item.name ? 'Applying...' : 'Mark Applied'}
                    </button>
                  ) : (
                    <span>-</span>
                  )}
                </td>
              </tr>
            ))}
            {items.length === 0 && (
              <tr>
                <td colSpan={5}>No migration files found.</td>
              </tr>
            )}
          </tbody>
        </table>
      </section>
    </main>
  )
}
