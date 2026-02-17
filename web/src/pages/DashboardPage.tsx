import { Link } from 'react-router-dom'

export function DashboardPage() {
  return (
    <main className="page with-nav">
      <section className="panel">
        <h1>Dashboard</h1>
        <p>Choose an admin area.</p>
        <div className="dashboard-grid">
          <Link className="dashboard-card" to="/sql-profiles">
            <h2>SQL Profiles</h2>
            <p>Create and manage export/query profiles.</p>
          </Link>
          <Link className="dashboard-card" to="/sessions">
            <h2>Sessions</h2>
            <p>Review active sessions and revoke as needed.</p>
          </Link>
          <Link className="dashboard-card" to="/migrations">
            <h2>Migrations</h2>
            <p>Inspect migration status and mark applied files.</p>
          </Link>
          <Link className="dashboard-card" to="/change-password">
            <h2>Change Password</h2>
            <p>Update your account password.</p>
          </Link>
        </div>
      </section>
    </main>
  )
}
