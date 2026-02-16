import { Link, useNavigate } from 'react-router-dom'
import { logout } from '../api/auth'
import { useAuth } from '../context/AuthContext'

export function NavBar() {
  const auth = useAuth()
  const navigate = useNavigate()

  async function handleLogout() {
    if (auth.token) {
      try {
        await logout(auth.token)
      } catch {
        // ignore and clear local state regardless
      }
    }
    auth.clearAuth()
    navigate('/login')
  }

  return (
    <header className="topbar">
      <div className="brand">modern-mcs admin</div>
      <nav className="navlinks">
        <Link to="/sql-profiles">SQL Profiles</Link>
        <Link to="/sessions">Sessions</Link>
        <Link to="/migrations">Migrations</Link>
        <Link to="/change-password">Change Password</Link>
      </nav>
      <div className="userinfo">
        <span>{auth.username}</span>
        <button onClick={handleLogout}>Logout</button>
      </div>
    </header>
  )
}
