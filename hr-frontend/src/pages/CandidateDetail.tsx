import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { jobApi, taskApi } from '../api';

export default function CandidateDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [matching, setMatching] = useState(false);
  const [matchMsg, setMatchMsg] = useState('');
  const [matchResult, setMatchResult] = useState<any>(null);

  const fetchData = useCallback(() => {
    if (!id) return;
    jobApi.getApplicationDetail(Number(id)).then((res) => {
      setData(res.data.data);
    }).finally(() => setLoading(false));
  }, [id]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleAIMatch = async () => {
    if (!id) return;
    setMatching(true);
    setMatchMsg('');
    try {
      const res = await taskApi.triggerAIMatch(Number(id));
      const taskId = res.data?.data?.task_id;
      if (taskId) {
        setMatchMsg('AI 匹配任务已提交，正在分析中...');
        pollMatchTask(taskId);
      } else {
        setMatchMsg('匹配请求已发送');
        setTimeout(() => setMatchMsg(''), 3000);
      }
    } catch (err: any) {
      setMatchMsg('匹配失败: ' + (err.response?.data?.message || err.message));
    } finally {
      setMatching(false);
    }
  };

  const pollMatchTask = (taskId: number) => {
    let attempts = 0;
    const maxAttempts = 30;
    const interval = setInterval(async () => {
      attempts++;
      if (attempts > maxAttempts) {
        clearInterval(interval);
        setMatchMsg('任务查询超时，AI 匹配可能仍在后台处理');
        return;
      }
      try {
        const res = await taskApi.get(taskId);
        const status = res.data?.data?.status;
        if (status === 'success') {
          clearInterval(interval);
          setMatchMsg('AI 匹配分析完成');
          const result = res.data?.data?.result;
          if (result) {
            try {
              const parsed = typeof result === 'string' ? JSON.parse(result) : result;
              setMatchResult(parsed);
            } catch {
              setMatchResult({ raw: result });
            }
          }
          setTimeout(() => setMatchMsg(''), 3000);
        } else if (status === 'failed') {
          clearInterval(interval);
          setMatchMsg('AI 匹配失败: ' + (res.data?.data?.error_message || '未知错误'));
        }
      } catch {
        clearInterval(interval);
        setMatchMsg('任务查询失败');
      }
    }, 2000);
  };

  if (loading) return <p>加载中...</p>;
  if (!data) return <p>数据不存在</p>;

  const { application, profile, resume } = data;

  const sectionStyle: React.CSSProperties = { marginBottom: 24, padding: 16, background: '#fafafa', borderRadius: 8 };
  const labelStyle: React.CSSProperties = { fontWeight: 'bold', display: 'inline-block', width: 100, color: '#666' };

  return (
    <div style={{ maxWidth: 700 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 16 }}>
        <button onClick={() => navigate(-1)} style={{ cursor: 'pointer' }}>&larr; 返回</button>
        <h2 style={{ margin: 0 }}>候选人详情</h2>
      </div>

      <div style={sectionStyle}>
        <h3>投递信息</h3>
        <p><span style={labelStyle}>岗位：</span>{application?.job_title || application?.jobTitle}</p>
        <p><span style={labelStyle}>投递时间：</span>{application?.created_at || application?.createdAt}</p>
        <p><span style={labelStyle}>状态：</span>{application?.status === 0 ? '待处理' : '已查看'}</p>
      </div>

      <div style={sectionStyle}>
        <h3>基础信息</h3>
        <p><span style={labelStyle}>姓名：</span>{profile?.name || '未填写'}</p>
        <p><span style={labelStyle}>电话：</span>{profile?.phone || '未填写'}</p>
        <p><span style={labelStyle}>学历：</span>{profile?.education || '未填写'}</p>
        <p><span style={labelStyle}>院校：</span>{profile?.university || '未填写'}</p>
      </div>

      <div style={sectionStyle}>
        <h3>工作/项目经历</h3>
        <p style={{ whiteSpace: 'pre-wrap' }}>{profile?.experience || '未填写'}</p>
      </div>

      <div style={sectionStyle}>
        <h3>技能标签</h3>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
          {(profile?.skills || '').split(',').filter(Boolean).map((skill: string, i: number) => (
            <span key={i} style={{ padding: '4px 12px', background: '#e6f7ff', border: '1px solid #91d5ff', borderRadius: 4 }}>{skill.trim()}</span>
          ))}
          {!profile?.skills && <span style={{ color: '#999' }}>未填写</span>}
        </div>
      </div>

      <div style={sectionStyle}>
        <h3>简历文件</h3>
        {resume?.url ? (
          <div>
            <p><span style={labelStyle}>文件名：</span>{resume.file_name || resume.fileName}</p>
            <p><span style={labelStyle}>文件类型：</span>{resume.file_type || resume.fileType}</p>
            <a href={resume.url} target="_blank" rel="noopener noreferrer"
              style={{ color: '#1677ff' }}>下载/查看简历</a>
          </div>
        ) : (
          <p style={{ color: '#999' }}>未上传简历</p>
        )}
      </div>

      <div style={sectionStyle}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h3 style={{ margin: 0 }}>AI 岗位匹配</h3>
          <button
            onClick={handleAIMatch}
            disabled={matching}
            style={{
              padding: '8px 16px',
              background: matching ? '#f2f4f7' : '#1677ff',
              color: matching ? '#999' : '#fff',
              border: 'none',
              borderRadius: 4,
              cursor: matching ? 'not-allowed' : 'pointer',
              fontSize: 14,
            }}
          >
            {matching ? '分析中...' : 'AI 匹配分析'}
          </button>
        </div>

        {matchMsg && (
          <div style={{
            marginTop: 12,
            padding: '10px 14px',
            borderRadius: 6,
            background: matchMsg.includes('失败') || matchMsg.includes('超时') ? '#fff1f0' : '#e6f7ff',
            color: matchMsg.includes('失败') || matchMsg.includes('超时') ? '#cf1322' : '#1677ff',
            fontSize: 14,
          }}>
            {matchMsg}
          </div>
        )}

        {matchResult && (
          <div style={{ marginTop: 16 }}>
            {matchResult.match_score !== undefined && (
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12, padding: '12px 16px', background: '#fff', borderRadius: 8 }}>
                <span style={{ fontSize: 15, color: '#666' }}>匹配分</span>
                <span style={{
                  fontSize: 28,
                  fontWeight: 700,
                  color: matchResult.match_score >= 80 ? '#52c41a' : matchResult.match_score >= 60 ? '#faad14' : '#ff4d4f',
                }}>
                  {matchResult.match_score}
                </span>
              </div>
            )}

            {matchResult.match_reason && (
              <div style={{ marginBottom: 12 }}>
                <h4 style={{ margin: '8px 0', color: '#333' }}>匹配理由</h4>
                <p style={{ whiteSpace: 'pre-wrap', margin: 0, padding: '10px 14px', background: '#fff', borderRadius: 6, fontSize: 14, lineHeight: 1.6 }}>{matchResult.match_reason}</p>
              </div>
            )}

            {matchResult.strengths && (
              <div style={{ marginBottom: 12 }}>
                <h4 style={{ margin: '8px 0', color: '#52c41a' }}>优势</h4>
                <p style={{ whiteSpace: 'pre-wrap', margin: 0, padding: '10px 14px', background: '#f6ffed', borderRadius: 6, fontSize: 14, lineHeight: 1.6 }}>{matchResult.strengths}</p>
              </div>
            )}

            {matchResult.weaknesses && (
              <div style={{ marginBottom: 12 }}>
                <h4 style={{ margin: '8px 0', color: '#faad14' }}>不足</h4>
                <p style={{ whiteSpace: 'pre-wrap', margin: 0, padding: '10px 14px', background: '#fffbe6', borderRadius: 6, fontSize: 14, lineHeight: 1.6 }}>{matchResult.weaknesses}</p>
              </div>
            )}

            {matchResult.interview_tips && (
              <div>
                <h4 style={{ margin: '8px 0', color: '#1677ff' }}>面试建议</h4>
                <p style={{ whiteSpace: 'pre-wrap', margin: 0, padding: '10px 14px', background: '#e6f7ff', borderRadius: 6, fontSize: 14, lineHeight: 1.6 }}>{matchResult.interview_tips}</p>
              </div>
            )}

            {matchResult.raw && !matchResult.match_reason && (
              <div style={{ marginTop: 8, padding: '10px 14px', background: '#fff', borderRadius: 6, fontSize: 13, color: '#666', whiteSpace: 'pre-wrap' }}>
                {matchResult.raw}
              </div>
            )}
          </div>
        )}

        {!matchResult && !matchMsg && (
          <p style={{ color: '#999', marginTop: 12, fontSize: 14 }}>点击"AI 匹配分析"按钮，系统将分析候选人与岗位的匹配度</p>
        )}
      </div>
    </div>
  );
}
