import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { authApi } from '../api';

export default function Login() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const res = await authApi.login({ username, password });
      const { token, id, username: uname } = res.data.data;
      localStorage.setItem('candidate_token', token);
      localStorage.setItem('candidate_user', JSON.stringify({ id, username: uname }));
      navigate('/');
    } catch (err: any) {
      setError(err.response?.data?.message || '登录失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#f0f2f5' }}>
      <div style={{ background: '#fff', padding: 40, borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.1)', width: 400 }}>
        <h2 style={{ textAlign: 'center', marginBottom: 24 }}>候选人登录</h2>
        {error && <div style={{ color: 'red', marginBottom: 12 }}>{error}</div>}
        <form onSubmit={handleSubmit}>
          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'block', marginBottom: 4 }}>用户名</label>
            <input type="text" value={username} onChange={(e) => setUsername(e.target.value)}
              style={{ width: '100%', padding: 8, borderRadius: 4, border: '1px solid #d9d9d9', boxSizing: 'border-box' }} required />
          </div>
          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'block', marginBottom: 4 }}>密码</label>
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)}
              style={{ width: '100%', padding: 8, borderRadius: 4, border: '1px solid #d9d9d9', boxSizing: 'border-box' }} required />
          </div>
          <button type="submit" disabled={loading}
            style={{ width: '100%', padding: 10, background: '#1677ff', color: '#fff', border: 'none', borderRadius: 4, cursor: 'pointer', fontSize: 16 }}>
            {loading ? '登录中...' : '登录'}
          </button>
        </form>
        <div style={{ textAlign: 'center', marginTop: 16 }}>
          <Link to="/register">没有账号？去注册</Link>
        </div>
      </div>
    </div>
  );
}
