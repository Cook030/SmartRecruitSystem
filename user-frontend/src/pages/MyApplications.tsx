import { useState, useEffect } from 'react';
import { applicationApi } from '../api';

interface AppItem {
  id: number;
  jobId: number;
  candidateId: number;
  status: number;
  createdAt: string;
  created_at?: string;
  jobTitle: string;
  job_title?: string;
  jobLocation: string;
  job_location?: string;
  jobSalary: string;
  job_salary?: string;
}

export default function MyApplications() {
  const [apps, setApps] = useState<AppItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);

  const load = async (p: number) => {
    setLoading(true);
    try {
      const res = await applicationApi.list(p, 10);
      setApps(res.data.data.applications || []);
      setTotal(res.data.data.total || 0);
    } catch {
      // handled
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(page); }, [page]);

  return (
    <div>
      <h2 style={{ marginBottom: 16 }}>我的投递</h2>

      {loading ? <p>加载中...</p> : (
        <div style={{ display: 'grid', gap: 12 }}>
          {apps.map((app) => (
            <div key={app.id} style={{ padding: 16, background: '#fff', borderRadius: 8, boxShadow: '0 1px 3px rgba(0,0,0,0.1)' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                  <h3 style={{ margin: '0 0 4px 0' }}>{app.job_title || app.jobTitle}</h3>
                  <div style={{ color: '#666', fontSize: 14 }}>
                    <span>{app.job_location || app.jobLocation}</span>
                    <span style={{ marginLeft: 16, color: '#fa541c' }}>{app.job_salary || app.jobSalary}</span>
                  </div>
                </div>
                <div style={{ textAlign: 'right' }}>
                  <div style={{ color: app.status === 0 ? '#fa8c16' : '#52c41a', fontWeight: 'bold' }}>
                    {app.status === 0 ? '待处理' : '已查看'}
                  </div>
                  <div style={{ fontSize: 12, color: '#999', marginTop: 4 }}>
                    {app.created_at || app.createdAt}
                  </div>
                </div>
              </div>
            </div>
          ))}
          {apps.length === 0 && <p style={{ textAlign: 'center', color: '#999' }}>暂无投递记录</p>}
        </div>
      )}

      {total > 10 && (
        <div style={{ marginTop: 16, textAlign: 'center' }}>
          <button disabled={page <= 1} onClick={() => setPage(page - 1)} style={{ marginRight: 8 }}>上一页</button>
          <span>第 {page} 页 / 共 {Math.ceil(total / 10)} 页</span>
          <button disabled={page >= Math.ceil(total / 10)} onClick={() => setPage(page + 1)} style={{ marginLeft: 8 }}>下一页</button>
        </div>
      )}
    </div>
  );
}
