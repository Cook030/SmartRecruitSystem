import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { jobApi } from '../api';

interface Job {
  id: number;
  title: string;
  description: string;
  requirement: string;
  salary: string;
  location: string;
  status: number;
  createdAt: string;
  created_at?: string;
}

export default function JobList() {
  const navigate = useNavigate();
  const [jobs, setJobs] = useState<Job[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);

  const loadJobs = async (p: number) => {
    setLoading(true);
    try {
      const res = await jobApi.list(p, 10);
      setJobs(res.data.data.jobs || []);
      setTotal(res.data.data.total || 0);
    } catch {
      // handled by interceptor
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadJobs(page); }, [page]);

  const handleOffline = async (id: number) => {
    if (!confirm('确认下架此岗位？')) return;
    try {
      await jobApi.offline(id);
      loadJobs(page);
    } catch (err: any) {
      alert(err.response?.data?.message || '操作失败');
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>我的岗位</h2>
        <button onClick={() => navigate('/jobs/new')}
          style={{ padding: '8px 20px', background: '#1677ff', color: '#fff', border: 'none', borderRadius: 4, cursor: 'pointer' }}>
          新增岗位
        </button>
      </div>

      {loading ? <p>加载中...</p> : (
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr style={{ background: '#fafafa', textAlign: 'left' }}>
              <th style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>岗位名称</th>
              <th style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>薪资</th>
              <th style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>地点</th>
              <th style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>状态</th>
              <th style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>创建时间</th>
              <th style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {jobs.map((job) => (
              <tr key={job.id}>
                <td style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>{job.title}</td>
                <td style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>{job.salary}</td>
                <td style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>{job.location}</td>
                <td style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>
                  <span style={{ color: job.status === 1 ? '#52c41a' : '#999' }}>{job.status === 1 ? '上架' : '下架'}</span>
                </td>
                <td style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>{job.created_at || job.createdAt}</td>
                <td style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>
                  <button onClick={() => navigate(`/jobs/edit/${job.id}`)}
                    style={{
                      marginRight: 8,
                      padding: '6px 16px',
                      background: '#e6f4ff',
                      color: '#1677ff',
                      border: '1px solid #91caff',
                      borderRadius: 6,
                      cursor: 'pointer',
                      fontSize: 13,
                      fontWeight: 500,
                      transition: 'all 0.2s'
                    }}
                    onMouseOver={(e) => {
                      e.currentTarget.style.background = '#bae7ff';
                      e.currentTarget.style.borderColor = '#69b1ff';
                    }}
                    onMouseOut={(e) => {
                      e.currentTarget.style.background = '#e6f4ff';
                      e.currentTarget.style.borderColor = '#91caff';
                    }}>
                    编辑
                  </button>
                  {job.status === 1 && (
                    <button onClick={() => handleOffline(job.id)}
                      style={{
                        marginRight: 8,
                        padding: '6px 16px',
                        background: '#fff1f0',
                        color: '#ff4d4f',
                        border: '1px solid #ffccc7',
                        borderRadius: 6,
                        cursor: 'pointer',
                        fontSize: 13,
                        fontWeight: 500,
                        transition: 'all 0.2s'
                      }}
                      onMouseOver={(e) => {
                        e.currentTarget.style.background = '#ffccc7';
                        e.currentTarget.style.borderColor = '#ff7875';
                      }}
                      onMouseOut={(e) => {
                        e.currentTarget.style.background = '#fff1f0';
                        e.currentTarget.style.borderColor = '#ffccc7';
                      }}>
                      下架
                    </button>
                  )}
                  <button onClick={() => navigate(`/jobs/${job.id}/applications`)}
                    style={{
                      padding: '6px 16px',
                      background: '#f6ffed',
                      color: '#52c41a',
                      border: '1px solid #b7eb8f',
                      borderRadius: 6,
                      cursor: 'pointer',
                      fontSize: 13,
                      fontWeight: 500,
                      transition: 'all 0.2s'
                    }}
                    onMouseOver={(e) => {
                      e.currentTarget.style.background = '#d9f7be';
                      e.currentTarget.style.borderColor = '#95de64';
                    }}
                    onMouseOut={(e) => {
                      e.currentTarget.style.background = '#f6ffed';
                      e.currentTarget.style.borderColor = '#b7eb8f';
                    }}>
                    查看投递
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
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
