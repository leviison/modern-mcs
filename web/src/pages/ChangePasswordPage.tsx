import { FormEvent, useState } from 'react'
import { changePassword } from '../api/auth'
import { getErrorMessage } from '../api/errors'
import { useAuth } from '../context/AuthContext'

export function ChangePasswordPage() {
  const auth = useAuth()
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!auth.token) return
    setError(null)
    setMessage(null)

    if (newPassword !== confirmPassword) {
      setError('New password and confirmation do not match')
      return
    }

    setLoading(true)
    try {
      await changePassword(auth.token, currentPassword, newPassword)
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      setMessage('Password updated successfully')
    } catch (err) {
      setError(getErrorMessage(err, 'Failed to update password'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="page with-nav">
      <section className="panel narrow">
        <h1>Change Password</h1>
        <p>Password policy: 12-128 chars with upper/lower/digit/special.</p>
        <form className="form-grid" onSubmit={handleSubmit}>
          <label>
            Current password
            <input type="password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} required />
          </label>
          <label>
            New password
            <input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} required />
          </label>
          <label>
            Confirm new password
            <input type="password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} required />
          </label>
          {error && <div className="error">{error}</div>}
          {message && <div className="success">{message}</div>}
          <button disabled={loading}>{loading ? 'Updating...' : 'Update password'}</button>
        </form>
      </section>
    </main>
  )
}
