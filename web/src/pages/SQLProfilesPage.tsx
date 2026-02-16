import { FormEvent, useEffect, useMemo, useState } from 'react'
import { createSQLProfile, deleteSQLProfile, listSQLProfiles, updateSQLProfile, type SQLProfileInput } from '../api/sqlProfiles'
import { getErrorMessage } from '../api/errors'
import type { SQLProfile } from '../types/api'
import { useAuth } from '../context/AuthContext'

type DBType = 'mysql' | 'mssql' | 'pgsql'

type ProfileForm = {
  name: string
  db_type: DBType
  host: string
  port: number
  username: string
  database: string
  commands: string
  use_ssl: boolean
}

const defaultForm: ProfileForm = {
  name: '',
  db_type: 'mysql',
  host: 'localhost',
  port: 3306,
  username: 'mcs',
  database: 'mcsdb',
  commands: 'SELECT 1',
  use_ssl: false
}

function mapProfileToForm(p: SQLProfile): ProfileForm {
  return {
    name: p.name,
    db_type: p.db_type,
    host: p.host,
    port: p.port,
    username: p.username,
    database: p.database,
    commands: p.commands,
    use_ssl: p.use_ssl
  }
}

export function SQLProfilesPage() {
  const auth = useAuth()
  const [items, setItems] = useState<SQLProfile[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [creating, setCreating] = useState(false)
  const [updating, setUpdating] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [createForm, setCreateForm] = useState<ProfileForm>(defaultForm)

  const [editingId, setEditingId] = useState<string | null>(null)
  const [editForm, setEditForm] = useState<ProfileForm | null>(null)

  async function refresh() {
    if (!auth.token) return
    setError(null)
    setLoading(true)
    try {
      const res = await listSQLProfiles(auth.token)
      setItems(res.items)
    } catch (err) {
      setError(getErrorMessage(err, 'Failed to load SQL profiles'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [auth.token])

  const editingProfile = useMemo(() => items.find((p) => p.id === editingId) || null, [items, editingId])

  async function handleCreate(e: FormEvent) {
    e.preventDefault()
    if (!auth.token) return
    setError(null)
    setCreating(true)
    try {
      await createSQLProfile(auth.token, createForm as SQLProfileInput)
      setCreateForm(defaultForm)
      await refresh()
    } catch (err) {
      setError(getErrorMessage(err, 'Failed to create profile'))
    } finally {
      setCreating(false)
    }
  }

  function beginEdit(item: SQLProfile) {
    setEditingId(item.id)
    setEditForm(mapProfileToForm(item))
  }

  function cancelEdit() {
    setEditingId(null)
    setEditForm(null)
  }

  async function handleUpdate(e: FormEvent) {
    e.preventDefault()
    if (!auth.token || !editingId || !editForm) return
    setError(null)
    setUpdating(true)
    try {
      await updateSQLProfile(auth.token, editingId, editForm as SQLProfileInput)
      cancelEdit()
      await refresh()
    } catch (err) {
      setError(getErrorMessage(err, 'Failed to update profile'))
    } finally {
      setUpdating(false)
    }
  }

  async function handleDelete(id: string) {
    if (!auth.token) return
    setError(null)
    setDeletingId(id)
    try {
      await deleteSQLProfile(auth.token, id)
      if (editingId === id) {
        cancelEdit()
      }
      await refresh()
    } catch (err) {
      setError(getErrorMessage(err, 'Failed to delete profile'))
    } finally {
      setDeletingId(null)
    }
  }

  return (
    <main className="page with-nav">
      <section className="panel">
        <h1>SQL Profiles</h1>
        <p>Create and manage SQL export profiles.</p>
        <form className="form-grid" onSubmit={handleCreate}>
          <label>
            Name
            <input value={createForm.name} onChange={(e) => setCreateForm((v) => ({ ...v, name: e.target.value }))} required />
          </label>
          <label>
            DB Type
            <select
              value={createForm.db_type}
              onChange={(e) => setCreateForm((v) => ({ ...v, db_type: e.target.value as DBType }))}
            >
              <option value="mysql">MySQL</option>
              <option value="mssql">MS SQL</option>
              <option value="pgsql">PostgreSQL</option>
            </select>
          </label>
          <label>
            Host
            <input value={createForm.host} onChange={(e) => setCreateForm((v) => ({ ...v, host: e.target.value }))} required />
          </label>
          <label>
            Port
            <input
              type="number"
              value={createForm.port}
              onChange={(e) => setCreateForm((v) => ({ ...v, port: Number(e.target.value) }))}
              required
            />
          </label>
          <label>
            Username
            <input value={createForm.username} onChange={(e) => setCreateForm((v) => ({ ...v, username: e.target.value }))} required />
          </label>
          <label>
            Database
            <input value={createForm.database} onChange={(e) => setCreateForm((v) => ({ ...v, database: e.target.value }))} required />
          </label>
          <label className="full">
            Commands
            <textarea
              value={createForm.commands}
              onChange={(e) => setCreateForm((v) => ({ ...v, commands: e.target.value }))}
              rows={4}
              required
            />
          </label>
          <label className="checkbox">
            <input
              type="checkbox"
              checked={createForm.use_ssl}
              onChange={(e) => setCreateForm((v) => ({ ...v, use_ssl: e.target.checked }))}
            />
            Use SSL
          </label>
          <button disabled={creating}>{creating ? 'Creating...' : 'Create Profile'}</button>
        </form>
        {error && <div className="error">{error}</div>}
        {loading && <p>Loading SQL profiles...</p>}
      </section>

      {editingProfile && editForm && (
        <section className="panel">
          <h2>Edit Profile: {editingProfile.name}</h2>
          <form className="form-grid" onSubmit={handleUpdate}>
            <label>
              Name
              <input value={editForm.name} onChange={(e) => setEditForm((v) => (v ? { ...v, name: e.target.value } : v))} required />
            </label>
            <label>
              DB Type
              <select
                value={editForm.db_type}
                onChange={(e) => setEditForm((v) => (v ? { ...v, db_type: e.target.value as DBType } : v))}
              >
                <option value="mysql">MySQL</option>
                <option value="mssql">MS SQL</option>
                <option value="pgsql">PostgreSQL</option>
              </select>
            </label>
            <label>
              Host
              <input value={editForm.host} onChange={(e) => setEditForm((v) => (v ? { ...v, host: e.target.value } : v))} required />
            </label>
            <label>
              Port
              <input
                type="number"
                value={editForm.port}
                onChange={(e) => setEditForm((v) => (v ? { ...v, port: Number(e.target.value) } : v))}
                required
              />
            </label>
            <label>
              Username
              <input
                value={editForm.username}
                onChange={(e) => setEditForm((v) => (v ? { ...v, username: e.target.value } : v))}
                required
              />
            </label>
            <label>
              Database
              <input
                value={editForm.database}
                onChange={(e) => setEditForm((v) => (v ? { ...v, database: e.target.value } : v))}
                required
              />
            </label>
            <label className="full">
              Commands
              <textarea
                value={editForm.commands}
                onChange={(e) => setEditForm((v) => (v ? { ...v, commands: e.target.value } : v))}
                rows={4}
                required
              />
            </label>
            <label className="checkbox">
              <input
                type="checkbox"
                checked={editForm.use_ssl}
                onChange={(e) => setEditForm((v) => (v ? { ...v, use_ssl: e.target.checked } : v))}
              />
              Use SSL
            </label>
            <div className="action-row">
              <button disabled={updating}>{updating ? 'Saving...' : 'Save Changes'}</button>
              <button type="button" className="secondary" disabled={updating} onClick={cancelEdit}>
                Cancel
              </button>
            </div>
          </form>
        </section>
      )}

      <section className="panel">
        <h2>Existing Profiles</h2>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Type</th>
              <th>Host</th>
              <th>Database</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id}>
                <td>{item.name}</td>
                <td>{item.db_type}</td>
                <td>
                  {item.host}:{item.port}
                </td>
                <td>{item.database}</td>
                <td className="action-cell">
                  <button className="secondary" disabled={deletingId === item.id} onClick={() => beginEdit(item)}>
                    Edit
                  </button>
                  <button className="danger" disabled={deletingId === item.id} onClick={() => void handleDelete(item.id)}>
                    {deletingId === item.id ? 'Deleting...' : 'Delete'}
                  </button>
                </td>
              </tr>
            ))}
            {items.length === 0 && (
              <tr>
                <td colSpan={5}>No profiles yet.</td>
              </tr>
            )}
          </tbody>
        </table>
      </section>
    </main>
  )
}
