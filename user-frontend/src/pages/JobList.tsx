import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { jobApi, applicationApi } from '../api';

interface Job {
  id: number;
  title: string;
  description: string;
  requirement: string;
  salary: string;
  location: string;
  status: number;
  created_at?: string;
  createdAt?: string;
}

export default function JobList() {
  const navigate = useNavigate();
  const [jobs, setJobs] = useState<Job[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [appliedJobIds, setAppliedJobIds] = useState<Set<number>>(new Set());

  const loadAppliedJobs = async () => {
    const token = localStorage.getItem('candidate_token');
    if (!token) {
      setAppliedJobIds(new Set());
      return;
    }
    try {
      const res = await applicationApi.list(1, 1000);
      const applications = res.data.data.applications || [];
      const appliedIds = new Set<number>(applications.map((app: any) => Number(app.job_id)));
      setAppliedJobIds(appliedIds);
    } catch (error) {
      console.error('Failed to load applied jobs:', error);
      setAppliedJobIds(new Set());
    }
  };

  const loadJobs = async (p: number) => {
    setLoading(true);
    try {
      const res = await jobApi.listPublic(p, 10);
      const jobList = res.data.data.jobs || [];
      setJobs(jobList);
      setTotal(res.data.data.total || 0);
      await loadAppliedJobs();
    } catch (error) {
      console.error('Failed to load jobs:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadJobs(page); }, [page]);

  const isLoggedIn = !!localStorage.getItem('candidate_token');

  const handleApply = async (jobId: number) => {
    if (!isLoggedIn) {
      if (confirm('请先登录后再投递，是否前往登录？')) {
        navigate('/login');
      }
      return;
    }
    try {
      await applicationApi.apply(jobId);
      alert('投递成功！');
      await loadAppliedJobs();
    } catch (err: any) {
      alert(err.response?.data?.message || '投递失败');
    }
  };

  return (
    <div>
      <h2 style={{ marginBottom: 16 }}>公开岗位列表</h2>

      {loading ? <p>加载中...</p> : (
        <div style={{ display: 'grid', gap: 16 }}>
          {jobs.map((job) => (
            <div key={job.id} style={{ padding: 20, background: '#fff', borderRadius: 8, boxShadow: '0 1px 3px rgba(0,0,0,0.1)' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div>
                  <h3 style={{ margin: '0 0 8px 0', fontSize: 18 }}>{job.title}</h3>
                  <div style={{ display: 'flex', gap: 16, color: '#666', fontSize: 14, marginBottom: 8 }}>
                    <span>{job.location || '未标注'}</span>
                    <span style={{ color: '#fa541c', fontWeight: 'bold' }}>{job.salary || '面议'}</span>
                  </div>
                  {job.description && <p style={{ color: '#666', fontSize: 14, marginBottom: 8 }}>{job.description.slice(0, 150)}{job.description.length > 150 ? '...' : ''}</p>}
                  {job.requirement && (
                    <div style={{ fontSize: 13, color: '#999' }}>
                      <strong>要求：</strong>{job.requirement.slice(0, 100)}{job.requirement.length > 100 ? '...' : ''}
                    </div>
                  )}
                </div>
                {appliedJobIds.has(job.id) ? (
                  <button disabled
                    style={{ padding: '8px 20px', background: '#d9d9d9', color: '#999', border: 'none', borderRadius: 4, cursor: 'not-allowed', whiteSpace: 'nowrap' }}>
                    已投递
                  </button>
                ) : (
                  <button onClick={() => handleApply(job.id)}
                    style={{ padding: '8px 20px', background: '#1677ff', color: '#fff', border: 'none', borderRadius: 4, cursor: 'pointer', whiteSpace: 'nowrap' }}>
                    投递
                  </button>
                )}
              </div>
            </div>
          ))}
          {jobs.length === 0 && <p style={{ textAlign: 'center', color: '#999' }}>暂无岗位</p>}
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
