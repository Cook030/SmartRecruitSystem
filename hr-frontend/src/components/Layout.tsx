import { Outlet, Link, useNavigate, useLocation } from 'react-router-dom';
import AIFloatingButton from './AIFloatingButton';

export default function Layout() {
  const navigate = useNavigate();
  const location = useLocation();
  const userStr = localStorage.getItem('hr_user');
  const user = userStr ? JSON.parse(userStr) : null;

  const handleLogout = () => {
    localStorage.removeItem('hr_token');
    localStorage.removeItem('hr_user');
    navigate('/login');
  };

  const navStyle = {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '0 24px',
    height: 56,
    background: '#001529',
    color: '#fff',
  };

  const getLinkStyle = (path: string) => ({
    color: location.pathname === path ? '#ffffff' : '#ffffffa6',
    textDecoration: 'none',
    marginRight: 24,
    fontSize: 14,
    fontWeight: location.pathname === path ? 'bold' : 'normal',
    borderBottom: location.pathname === path ? '2px solid #ffffff' : '2px solid transparent',
  });

  return (
    <div>
      <nav style={navStyle}>
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <span style={{ fontSize: 18, fontWeight: 'bold', marginRight: 40 }}>智能招聘 - HR 管理端</span>
          <Link to="/dashboard" style={getLinkStyle('/dashboard')}>数据看板</Link>
          <Link to="/jobs" style={getLinkStyle('/jobs')}>岗位管理</Link>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <span style={{ fontSize: 14 }}>{user?.name || user?.username}</span>
          <button onClick={handleLogout}
            style={{ padding: '4px 12px', background: 'transparent', color: '#fff', border: '1px solid #ffffff40', borderRadius: 4, cursor: 'pointer' }}>
            退出
          </button>
        </div>
      </nav>
      <div style={{ padding: 24, maxWidth: 1200, margin: '0 auto' }}>
        <Outlet />
      </div>
      <AIFloatingButton />
    </div>
  );
}
