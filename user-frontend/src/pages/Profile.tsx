import { useState, useEffect } from 'react';
import { profileApi } from '../api';

export default function Profile() {
  const [name, setName] = useState('');
  const [phone, setPhone] = useState('');
  const [education, setEducation] = useState('');
  const [university, setUniversity] = useState('');
  const [experience, setExperience] = useState('');
  const [skills, setSkills] = useState('');
  const [isComplete, setIsComplete] = useState(false);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    profileApi.get().then((res) => {
      const p = res.data.data;
      if (p) {
        setName(p.name || '');
        setPhone(p.phone || '');
        setEducation(p.education || '');
        setUniversity(p.university || '');
        setExperience(p.experience || '');
        setSkills(p.skills || '');
        setIsComplete(p.isComplete || p.is_complete || false);
      }
    });
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setMessage('');
    setLoading(true);
    try {
      const res = await profileApi.update({ name, phone, education, university, experience, skills });
      const complete = res.data.data.isComplete || res.data.data.is_complete;
      setIsComplete(complete);
      setMessage(complete ? '资料已完善！' : '保存成功，但资料尚未完善（所有字段均需填写）');
    } catch (err: any) {
      setMessage(err.response?.data?.message || '保存失败');
    } finally {
      setLoading(false);
    }
  };

  const inputStyle = { width: '100%', padding: 8, borderRadius: 4, border: '1px solid #d9d9d9', boxSizing: 'border-box' as const, marginBottom: 16 };

  return (
    <div style={{ maxWidth: 600 }}>
      <h2 style={{ marginBottom: 8 }}>个人资料</h2>
      <p style={{ color: isComplete ? '#52c41a' : '#fa541c', marginBottom: 16 }}>
        {isComplete ? '资料已完善' : '资料未完善（投递前需补齐所有信息）'}
      </p>
      {message && <div style={{ padding: 8, marginBottom: 12, background: '#e6f7ff', borderRadius: 4 }}>{message}</div>}

      <form onSubmit={handleSubmit}>
        <label>姓名</label>
        <input value={name} onChange={(e) => setName(e.target.value)} style={inputStyle} />

        <label>联系电话</label>
        <input value={phone} onChange={(e) => setPhone(e.target.value)} style={inputStyle} />

        <label>最高学历</label>
        <select value={education} onChange={(e) => setEducation(e.target.value)} style={inputStyle}>
          <option value="">请选择</option>
          <option value="高中">高中</option>
          <option value="大专">大专</option>
          <option value="本科">本科</option>
          <option value="硕士">硕士</option>
          <option value="博士">博士</option>
        </select>

        <label>毕业院校</label>
        <input value={university} onChange={(e) => setUniversity(e.target.value)} style={inputStyle} />

        <label>工作/项目经历</label>
        <textarea value={experience} onChange={(e) => setExperience(e.target.value)}
          style={{ ...inputStyle, minHeight: 100 }}
          placeholder="请描述你的工作或项目经历" />

        <label>技能标签（逗号分隔）</label>
        <input value={skills} onChange={(e) => setSkills(e.target.value)} style={inputStyle}
          placeholder="例如: Go, React, MySQL, Docker" />

        <button type="submit" disabled={loading}
          style={{ padding: '8px 24px', background: '#1677ff', color: '#fff', border: 'none', borderRadius: 4, cursor: 'pointer' }}>
          {loading ? '保存中...' : '保存'}
        </button>
      </form>
    </div>
  );
}
