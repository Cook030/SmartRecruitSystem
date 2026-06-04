import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { jobApi } from '../api';

interface ApplicationItem {
  id: number;
  jobId: number;
  candidateId: number;
  status: number;
  createdAt: string;
  created_at?: string;
  jobTitle: string;
  job_title?: string;
  candidateName: string;
  candidate_name?: string;
}

export default function Applications() {
  const { jobId } = useParams();
  const navigate = useNavigate();
  const [applications, setApplications] = useState<ApplicationItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);

  const load = async (p: number) => {
    setLoading(true);
    try {
      const res = await jobApi.getApplications(Number(jobId), p, 10);
      setApplications(res.data.data.applications || []);
      setTotal(res.data.data.total || 0);
    } catch {
      // handled
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(page); }, [page, jobId]);

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 16 }}>
        <button onClick={() => navigate('/jobs')} style={{ cursor: 'pointer' }}>&larr; 返回</button>
        <h2 style={{ margin: 0 }}>岗位投递列表</h2>
      </div>

      {loading ? <p>加载中...</p> : (
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr style={{ background: '#fafafa', textAlign: 'left' }}>
              <th style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>候选人</th>
              <th style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>投递时间</th>
              <th style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>状态</th>
              <th style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {applications.map((app) => (
              <tr key={app.id}>
                <td style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>{app.candidate_name || app.candidateName || '未填写'}</td>
                <td style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>{app.created_at || app.createdAt}</td>
                <td style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>{app.status === 0 ? '待处理' : '已查看'}</td>
                <td style={{ padding: 12, borderBottom: '1px solid #e8e8e8' }}>
                  <button onClick={() => navigate(`/applications/${app.id}/detail`)}
                    style={{ padding: '4px 12px', cursor: 'pointer' }}>查看详情</button>
                </td>
              </tr>
            ))}
            {applications.length === 0 && (
              <tr><td colSpan={4} style={{ padding: 24, textAlign: 'center', color: '#999' }}>暂无投递</td></tr>
            )}
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
