import { useState, useEffect, useRef, useCallback } from 'react';
import { resumeApi, taskApi } from '../api';

const keyframes = `
@keyframes pulseGlow {
  0% { box-shadow: 0 0 8px rgba(22,119,255,0.3); }
  50% { box-shadow: 0 0 24px rgba(22,119,255,0.7), 0 0 48px rgba(22,119,255,0.3); }
  100% { box-shadow: 0 0 8px rgba(22,119,255,0.3); }
}
@keyframes particleFloat {
  0% { transform: translate(0, 0) scale(1); opacity: 0.8; }
  50% { transform: translate(var(--dx), var(--dy)) scale(0.6); opacity: 0.4; }
  100% { transform: translate(0, 0) scale(1); opacity: 0.8; }
}
@keyframes dotBlink {
  0%, 20% { opacity: 0.2; }
  50% { opacity: 1; }
  80%, 100% { opacity: 0.2; }
}
`;

function AIThinkingAnimation() {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const animFrameRef = useRef<number>(0);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const dpr = window.devicePixelRatio || 1;
    const w = 280;
    const h = 120;
    canvas.width = w * dpr;
    canvas.height = h * dpr;
    canvas.style.width = w + 'px';
    canvas.style.height = h + 'px';
    ctx.scale(dpr, dpr);

    const particles: { x: number; y: number; vx: number; vy: number; r: number; phase: number }[] = [];
    for (let i = 0; i < 40; i++) {
      particles.push({
        x: w / 2 + (Math.random() - 0.5) * 60,
        y: h / 2 + (Math.random() - 0.5) * 40,
        vx: (Math.random() - 0.5) * 1.2,
        vy: (Math.random() - 0.5) * 1.2,
        r: Math.random() * 2.5 + 1,
        phase: Math.random() * Math.PI * 2,
      });
    }

    const cx = w / 2;
    const cy = h / 2;

    const draw = (t: number) => {
      ctx.clearRect(0, 0, w, h);

      const pulseR = 20 + Math.sin(t / 600) * 8;
      const grad = ctx.createRadialGradient(cx, cy, 0, cx, cy, pulseR * 2);
      grad.addColorStop(0, 'rgba(22,119,255,0.25)');
      grad.addColorStop(0.5, 'rgba(22,119,255,0.08)');
      grad.addColorStop(1, 'rgba(22,119,255,0)');
      ctx.fillStyle = grad;
      ctx.beginPath();
      ctx.arc(cx, cy, pulseR * 2, 0, Math.PI * 2);
      ctx.fill();

      ctx.strokeStyle = `rgba(22,119,255,${0.3 + Math.sin(t / 400) * 0.15})`;
      ctx.lineWidth = 1.5;
      ctx.beginPath();
      ctx.arc(cx, cy, pulseR, 0, Math.PI * 2);
      ctx.stroke();

      for (const p of particles) {
        const angle = Math.atan2(cy - p.y, cx - p.x);
        const dist = Math.sqrt((p.x - cx) ** 2 + (p.y - cy) ** 2);
        const attract = 0.02 + Math.sin(t / 800 + p.phase) * 0.01;
        const spread = Math.sin(t / 1200 + p.phase * 2) * 0.008;

        p.vx += Math.cos(angle) * attract - Math.cos(angle) * spread * dist * 0.01;
        p.vy += Math.sin(angle) * attract - Math.sin(angle) * spread * dist * 0.01;
        p.vx *= 0.98;
        p.vy *= 0.98;
        p.x += p.vx;
        p.y += p.vy;

        if (p.x < 0 || p.x > w) p.vx *= -1;
        if (p.y < 0 || p.y > h) p.vy *= -1;

        const alpha = 0.4 + Math.sin(t / 300 + p.phase) * 0.3;
        ctx.fillStyle = `rgba(22,119,255,${alpha})`;
        ctx.beginPath();
        ctx.arc(p.x, p.y, p.r, 0, Math.PI * 2);
        ctx.fill();
      }

      for (let i = 0; i < particles.length; i++) {
        for (let j = i + 1; j < particles.length; j++) {
          const dx = particles[i].x - particles[j].x;
          const dy = particles[i].y - particles[j].y;
          const d = Math.sqrt(dx * dx + dy * dy);
          if (d < 50) {
            ctx.strokeStyle = `rgba(22,119,255,${0.12 * (1 - d / 50)})`;
            ctx.lineWidth = 0.5;
            ctx.beginPath();
            ctx.moveTo(particles[i].x, particles[i].y);
            ctx.lineTo(particles[j].x, particles[j].y);
            ctx.stroke();
          }
        }
      }

      animFrameRef.current = requestAnimationFrame(() => draw(performance.now()));
    };

    animFrameRef.current = requestAnimationFrame(() => draw(performance.now()));
    return () => cancelAnimationFrame(animFrameRef.current);
  }, []);

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      padding: '20px 0',
    }}>
      <canvas ref={canvasRef} style={{ display: 'block' }} />
      <div style={{ marginTop: 8, color: '#1677ff', fontSize: 14, fontWeight: 500, letterSpacing: 1 }}>
        AI 正在分析简历
        <span style={{ animation: 'dotBlink 1.4s infinite', animationDelay: '0s' }}>.</span>
        <span style={{ animation: 'dotBlink 1.4s infinite', animationDelay: '0.2s' }}>.</span>
        <span style={{ animation: 'dotBlink 1.4s infinite', animationDelay: '0.4s' }}>.</span>
      </div>
      <style>{keyframes}</style>
    </div>
  );
}

export default function Resume() {
  const [resume, setResume] = useState<any>(null);
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [message, setMessage] = useState('');
  const [parsing, setParsing] = useState(false);
  const [parseMsg, setParseMsg] = useState('');
  const [parseResult, setParseResult] = useState<any>(null);
  const [collapsed, setCollapsed] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const loadResume = useCallback(async () => {
    try {
      const res = await resumeApi.get();
      setResume(res.data.data);
    } catch {
      setResume(null);
    }
  }, []);

  const loadParseResult = useCallback(async (resumeId: number) => {
    try {
      const res = await taskApi.getResumeParseResult(resumeId);
      const data = res.data?.data;
      if (data) {
        setParseResult({
          education: data.education,
          experience: data.experience,
          projects: data.projects,
          skills: data.skills,
          raw: data.raw_result,
        });
      }
    } catch {
      setParseResult(null);
    }
  }, []);

  useEffect(() => {
    loadResume();
  }, [loadResume]);

  useEffect(() => {
    if (resume?.id) {
      loadParseResult(resume.id);
    }
  }, [resume?.id, loadParseResult]);

  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  const handleUpload = async () => {
    if (!file) return;
    setMessage('');
    setUploading(true);
    try {
      await resumeApi.upload(file);
      setMessage('上传成功！');
      setFile(null);
      setParseResult(null);
      loadResume();
    } catch (err: any) {
      setMessage(err.response?.data?.message || '上传失败');
    } finally {
      setUploading(false);
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (!f) return;
    const ext = f.name.split('.').pop()?.toLowerCase();
    if (!['pdf', 'doc', 'docx'].includes(ext || '')) {
      setMessage('仅支持 PDF、DOC、DOCX 格式');
      return;
    }
    setFile(f);
    setMessage('');
  };

  const handleDeleteResume = async () => {
    if (!window.confirm('确定要删除当前简历吗？删除后需要重新上传才能投递岗位。')) return;
    setDeleting(true);
    try {
      await resumeApi.delete();
      setResume(null);
      setParseResult(null);
      setMessage('简历已删除');
    } catch (err: any) {
      setMessage(err.response?.data?.message || '删除失败');
    } finally {
      setDeleting(false);
    }
  };

  const handleParseResume = async () => {
    if (!resume?.id) return;
    setParsing(true);
    setParseMsg('');
    setCollapsed(false);
    try {
      const res = await taskApi.triggerResumeParse(resume.id);
      const taskId = res.data?.data?.task_id;
      if (taskId) {
        setParseMsg('简历解析任务已提交，正在分析中...');
        pollParseTask(taskId);
      } else {
        setParseMsg('解析请求已发送');
        setTimeout(() => setParseMsg(''), 3000);
      }
    } catch (err: any) {
      setParseMsg('解析失败: ' + (err.response?.data?.message || err.message));
      setParsing(false);
    }
  };

  const pollParseTask = (taskId: number) => {
    let attempts = 0;
    const maxAttempts = 60;
    if (pollRef.current) clearInterval(pollRef.current);
    pollRef.current = setInterval(async () => {
      attempts++;
      if (attempts > maxAttempts) {
        clearInterval(pollRef.current!);
        setParseMsg('任务查询超时，解析可能仍在后台处理');
        setParsing(false);
        return;
      }
      try {
        const res = await taskApi.get(taskId);
        const status = res.data?.data?.status;
        if (status === 'success') {
          clearInterval(pollRef.current!);
          setParseMsg('简历解析完成');
          setParsing(false);
          if (resume?.id) {
            loadParseResult(resume.id);
          }
          setTimeout(() => setParseMsg(''), 3000);
        } else if (status === 'failed') {
          clearInterval(pollRef.current!);
          setParseMsg('简历解析失败: ' + (res.data?.data?.error_message || '未知错误'));
          setParsing(false);
        }
      } catch {
        clearInterval(pollRef.current!);
        setParseMsg('任务查询失败');
        setParsing(false);
      }
    }, 2000);
  };

  const handleDeleteParseResult = async () => {
    if (!resume?.id) return;
    if (!window.confirm('确定要删除解析结果吗？')) return;
    try {
      await taskApi.deleteResumeParseResult(resume.id);
      setParseResult(null);
      setCollapsed(false);
    } catch {
      setMessage('删除解析结果失败');
    }
  };

  return (
    <div style={{ maxWidth: 600 }}>
      <h2 style={{ marginBottom: 16 }}>简历管理</h2>

      {resume && (
        <div style={{
          padding: 16,
          background: '#f6ffed',
          border: '1px solid #b7eb8f',
          borderRadius: 8,
          marginBottom: 24,
          animation: parsing ? 'pulseGlow 2s ease-in-out infinite' : 'none',
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
            <h3 style={{ margin: 0 }}>当前简历</h3>
            <div style={{ display: 'flex', gap: 8 }}>
              <button
                onClick={handleParseResume}
                disabled={parsing}
                style={{
                  padding: '6px 14px',
                  background: parsing ? '#f2f4f7' : '#1677ff',
                  color: parsing ? '#999' : '#fff',
                  border: 'none',
                  borderRadius: 4,
                  cursor: parsing ? 'not-allowed' : 'pointer',
                  fontSize: 13,
                }}
              >
                {parsing ? '解析中...' : 'AI 解析简历'}
              </button>
              <button
                onClick={handleDeleteResume}
                disabled={deleting}
                style={{
                  padding: '6px 14px',
                  background: deleting ? '#f2f4f7' : '#ff4d4f',
                  color: deleting ? '#999' : '#fff',
                  border: 'none',
                  borderRadius: 4,
                  cursor: deleting ? 'not-allowed' : 'pointer',
                  fontSize: 13,
                }}
              >
                {deleting ? '删除中...' : '删除简历'}
              </button>
            </div>
          </div>
          <p>文件名：{resume.file_name || resume.fileName}</p>
          <p>类型：{resume.file_type || resume.fileType}</p>
          <p>上传时间：{resume.created_at || resume.createdAt}</p>
          {resume.url && (
            <a href={resume.url} target="_blank" rel="noopener noreferrer"
              style={{ color: '#1677ff' }}>查看/下载简历</a>
          )}
        </div>
      )}

      {parsing && <AIThinkingAnimation />}

      {parseMsg && !parsing && (
        <div style={{
          padding: '10px 14px',
          marginBottom: 16,
          borderRadius: 6,
          background: parseMsg.includes('失败') || parseMsg.includes('超时') ? '#fff1f0' : '#e6f7ff',
          color: parseMsg.includes('失败') || parseMsg.includes('超时') ? '#cf1322' : '#1677ff',
          fontSize: 14,
        }}>
          {parseMsg}
        </div>
      )}

      {parseResult && !parsing && (
        <div style={{
          padding: 16,
          background: '#fafafa',
          borderRadius: 8,
          marginBottom: 24,
          border: '1px solid #e8e8e8',
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: collapsed ? 0 : 12 }}>
            <h3 style={{ margin: 0, fontSize: 15 }}>AI 解析结果</h3>
            <div style={{ display: 'flex', gap: 6 }}>
              <button
                onClick={() => setCollapsed(!collapsed)}
                style={{
                  padding: '3px 10px',
                  background: '#fff',
                  border: '1px solid #d9d9d9',
                  borderRadius: 4,
                  cursor: 'pointer',
                  fontSize: 12,
                  color: '#666',
                }}
              >
                {collapsed ? '展开' : '收起'}
              </button>
              <button
                onClick={handleDeleteParseResult}
                style={{
                  padding: '3px 10px',
                  background: '#fff',
                  border: '1px solid #ffccc7',
                  borderRadius: 4,
                  cursor: 'pointer',
                  fontSize: 12,
                  color: '#ff4d4f',
                }}
              >
                删除
              </button>
            </div>
          </div>

          {!collapsed && (
            <>
              {parseResult.education && (
                <div style={{ marginBottom: 8 }}>
                  <span style={{ fontWeight: 'bold', display: 'inline-block', width: 80, color: '#666' }}>教育经历：</span>
                  <span style={{ whiteSpace: 'pre-wrap' }}>{parseResult.education}</span>
                </div>
              )}
              {parseResult.experience && (
                <div style={{ marginBottom: 8 }}>
                  <span style={{ fontWeight: 'bold', display: 'inline-block', width: 80, color: '#666' }}>工作经历：</span>
                  <span style={{ whiteSpace: 'pre-wrap' }}>{parseResult.experience}</span>
                </div>
              )}
              {parseResult.projects && (
                <div style={{ marginBottom: 8 }}>
                  <span style={{ fontWeight: 'bold', display: 'inline-block', width: 80, color: '#666' }}>项目经历：</span>
                  <span style={{ whiteSpace: 'pre-wrap' }}>{parseResult.projects}</span>
                </div>
              )}
              {parseResult.skills && (
                <div style={{ marginBottom: 8 }}>
                  <span style={{ fontWeight: 'bold', display: 'inline-block', width: 80, color: '#666' }}>技能标签：</span>
                  <div style={{ display: 'inline-flex', flexWrap: 'wrap', gap: 6 }}>
                    {parseResult.skills.split(',').filter(Boolean).map((skill: string, i: number) => (
                      <span key={i} style={{ padding: '2px 10px', background: '#e6f7ff', border: '1px solid #91d5ff', borderRadius: 4, fontSize: 13 }}>{skill.trim()}</span>
                    ))}
                  </div>
                </div>
              )}
              {parseResult.raw && !parseResult.education && (
                <div style={{ padding: '10px 14px', background: '#fff', borderRadius: 6, fontSize: 13, color: '#666', whiteSpace: 'pre-wrap' }}>
                  {parseResult.raw}
                </div>
              )}
            </>
          )}
        </div>
      )}

      <div style={{ padding: 16, background: '#fff', borderRadius: 8, border: '1px solid #d9d9d9' }}>
        <h3 style={{ margin: '0 0 12px 0' }}>{resume ? '更新简历' : '上传简历'}</h3>
        <p style={{ color: '#666', fontSize: 14, marginBottom: 12 }}>支持格式：PDF、DOC、DOCX</p>

        {message && <div style={{ padding: 8, marginBottom: 12, background: '#e6f7ff', borderRadius: 4 }}>{message}</div>}

        <input type="file" accept=".pdf,.doc,.docx" onChange={handleFileChange}
          style={{ marginBottom: 12, display: 'block' }} />

        {file && (
          <div style={{ marginBottom: 12, color: '#666' }}>
            已选择：{file.name} ({(file.size / 1024).toFixed(1)} KB)
          </div>
        )}

        <button onClick={handleUpload} disabled={!file || uploading}
          style={{ padding: '8px 24px', background: '#1677ff', color: '#fff', border: 'none', borderRadius: 4, cursor: 'pointer' }}>
          {uploading ? '上传中...' : '上传'}
        </button>
      </div>
    </div>
  );
}
