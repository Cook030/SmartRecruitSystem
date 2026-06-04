import { Outlet, Link, useNavigate, useLocation } from 'react-router-dom';

export default function Layout() {
  const navigate = useNavigate();
  const location = useLocation();
  const userStr = localStorage.getItem('candidate_user');
  const user = userStr ? JSON.parse(userStr) : null;
  const isLoggedIn = !!localStorage.getItem('candidate_token');

  const handleLogout = () => {
    localStorage.removeItem('candidate_token');
    localStorage.removeItem('candidate_user');
    navigate('/');
  };

  const navStyle = {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '0 24px',
    height: 56,
    background: '#1677ff',
    color: '#fff',
  };

  const getLinkStyle = (path: string) => ({
    color: location.pathname === path ? '#ffffff' : '#ffffffd9',
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
          <span style={{ fontSize: 18, fontWeight: 'bold', marginRight: 40 }}>智能招聘 - 候选人端</span>
          <Link to="/" style={getLinkStyle('/')}>岗位浏览</Link>
          {isLoggedIn && (
            <>
              <Link to="/profile" style={getLinkStyle('/profile')}>个人资料</Link>
              <Link to="/resume" style={getLinkStyle('/resume')}>简历管理</Link>
              <Link to="/applications" style={getLinkStyle('/applications')}>我的投递</Link>
            </>
          )}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          {isLoggedIn ? (
            <>
              <span style={{ fontSize: 14 }}>{user?.username}</span>
              <button onClick={handleLogout}
                style={{ padding: '4px 12px', background: 'transparent', color: '#fff', border: '1px solid #ffffff60', borderRadius: 4, cursor: 'pointer' }}>
                退出
              </button>
            </>
          ) : (
            <>
              <Link to="/login" style={getLinkStyle('/login')}>登录</Link>
              <Link to="/register" style={getLinkStyle('/register')}>注册</Link>
            </>
          )}
        </div>
      </nav>
      <div style={{ padding: 24, maxWidth: 1000, margin: '0 auto' }}>
        <Outlet />
      </div>
    </div>
  );
}
