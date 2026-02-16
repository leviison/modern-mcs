import { useEffect, useState } from 'react'
import { listSessions, revokeSession } from '../api/auth'
import { getErrorMessage } from '../api/errors'
import type { SessionView } from '../types/api'
import { useAuth } from '../context/AuthContext'

export function SessionsPage() {
  const auth = useAuth()
  const [items, setItems] = useState<SessionView[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [busySessionId, setBusySessionId] = useState<string | null>(null)

  async function refresh() {
    if (!auth.token) return
    setError(null)
    setLoading(true)
    try {
      const res = await listSessions(auth.token)
      setItems(res.items)
    } catch (err) {
      setError(getErrorMessage(err, 'Failed to load sessions'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [auth.token])

  async function handleRevoke(sessionId: string) {
    if (!auth.token) return
    setError(null)
    setBusySessionId(sessionId)
    try {
      await revokeSession(auth.token, sessionId)
      await refresh()
    } catch (err) {
      setError(getErrorMessage(err, 'Failed to revoke session'))
    } finally {
      setBusySessionId(null)
    }
  }

  return (
    <main className="page with-nav">
      <section className="panel">
        <h1>Sessions</h1>
        <p>Active sessions (admin view).</p>
        {error && <div className="error">{error}</div>}
        {loading && <p>Loading sessions...</p>}
        <table>
          <thead>
            <tr>
              <th>Session ID</th>
              <th>User</th>
              <th>Created</th>
              <th>Expires</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id}>
                <td>{item.id}</td>
                <td>{item.username}</td>
                <td>{new Date(item.created_at).toLocaleString()}</td>
                <td>{new Date(item.expires_at).toLocaleString()}</td>
                <td>
                  <button className="danger" disabled={busySessionId === item.id} onClick={() => void handleRevoke(item.id)}>
                    {busySessionId === item.id ? 'Revoking...' : 'Revoke'}
                  </button>
                </td>
              </tr>
            ))}
            {items.length === 0 && (
              <tr>
                <td colSpan={5}>No active sessions.</td>
              </tr>
            )}
          </tbody>
        </table>
      </section>
    </main>
  )
}
