import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { jobApi } from '../api';

export default function JobForm() {
  const navigate = useNavigate();
  const { id } = useParams();
  const isEdit = !!id;

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [requirement, setRequirement] = useState('');
  const [salary, setSalary] = useState('');
  const [location, setLocation] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isEdit) {
      jobApi.list(1, 100).then((res) => {
        const jobs = res.data.data.jobs || [];
        const job = jobs.find((j: any) => j.id === Number(id));
        if (job) {
          setTitle(job.title);
          setDescription(job.description);
          setRequirement(job.requirement);
          setSalary(job.salary);
          setLocation(job.location);
        }
      });
    }
  }, [id, isEdit]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const data = { title, description, requirement, salary, location };
      if (isEdit) {
        await jobApi.update(Number(id), data);
      } else {
        await jobApi.create(data);
      }
      navigate('/jobs');
    } catch (err: any) {
      setError(err.response?.data?.message || '操作失败');
    } finally {
      setLoading(false);
    }
  };

  const inputStyle = { width: '100%', padding: 8, borderRadius: 4, border: '1px solid #d9d9d9', boxSizing: 'border-box' as const, marginBottom: 16 };

  return (
    <div style={{ maxWidth: 600 }}>
      <h2>{isEdit ? '编辑岗位' : '新增岗位'}</h2>
      {error && <div style={{ color: 'red', marginBottom: 12 }}>{error}</div>}
      <form onSubmit={handleSubmit}>
        <label>岗位名称 *</label>
        <input value={title} onChange={(e) => setTitle(e.target.value)} style={inputStyle} required />

        <label>岗位描述</label>
        <textarea value={description} onChange={(e) => setDescription(e.target.value)}
          style={{ ...inputStyle, minHeight: 80 }} />

        <label>岗位要求</label>
        <textarea value={requirement} onChange={(e) => setRequirement(e.target.value)}
          style={{ ...inputStyle, minHeight: 80 }} />

        <label>薪资范围</label>
        <input value={salary} onChange={(e) => setSalary(e.target.value)} style={inputStyle} />

        <label>工作地点</label>
        <input value={location} onChange={(e) => setLocation(e.target.value)} style={inputStyle} />

        <div style={{ display: 'flex', gap: 12 }}>
          <button type="submit" disabled={loading}
            style={{ padding: '8px 24px', background: '#1677ff', color: '#fff', border: 'none', borderRadius: 4, cursor: 'pointer' }}>
            {loading ? '提交中...' : '提交'}
          </button>
          <button type="button" onClick={() => navigate('/jobs')}
            style={{ padding: '8px 24px', border: '1px solid #d9d9d9', borderRadius: 4, cursor: 'pointer', background: '#fff' }}>
            取消
          </button>
        </div>
      </form>
    </div>
  );
}
