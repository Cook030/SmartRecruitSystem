import { useCallback, useEffect, useMemo, useState } from 'react';
import { dashboardApi, taskApi } from '../api';

type TrendPoint = { date: string; count: number };
type HotJob = { job_id: number; title: string; location: string; salary: string; application_count: number };
type HotSkill = { skill: string; count: number };

type DashboardState = {
  overview: Record<string, number> | null;
  jobTrend: TrendPoint[];
  applicationTrend: TrendPoint[];
  hotJobs: HotJob[];
  hotSkills: HotSkill[];
  funnel: Record<string, number> | null;
  ai: Record<string, number> | null;
};

const initialState: DashboardState = {
  overview: null,
  jobTrend: [],
  applicationTrend: [],
  hotJobs: [],
  hotSkills: [],
  funnel: null,
  ai: null,
};

const cards = [
  ['total_jobs', '岗位总数'],
  ['online_jobs', '上架岗位'],
  ['offline_jobs', '下架岗位'],
  ['total_applications', '投递总数'],
  ['total_candidates', '候选人数量'],
  ['complete_profiles', '已完善资料'],
  ['total_resumes', '简历上传数量'],
  ['total_chats', 'AI 对话次数'],
];

export default function Dashboard() {
  const [range, setRange] = useState('week');
  const [data, setData] = useState<DashboardState>(initialState);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [refreshing, setRefreshing] = useState(false);
  const [taskMsg, setTaskMsg] = useState('');

  const fetchData = useCallback(() => {
    let mounted = true;
    setLoading(true);
    setError('');
    Promise.all([
      dashboardApi.overview(),
      dashboardApi.jobTrend({ range }),
      dashboardApi.applicationTrend({ range }),
      dashboardApi.hotJobs({ limit: 10, range }),
      dashboardApi.hotSkills({ limit: 10, range }),
      dashboardApi.candidateFunnel(),
      dashboardApi.aiMatchStats(),
    ])
      .then(([overview, jobTrend, applicationTrend, hotJobs, hotSkills, funnel, ai]) => {
        if (!mounted) return;
        setData({
          overview: overview.data.data,
          jobTrend: jobTrend.data.data?.points || [],
          applicationTrend: applicationTrend.data.data?.points || [],
          hotJobs: hotJobs.data.data?.jobs || [],
          hotSkills: hotSkills.data.data?.skills || [],
          funnel: funnel.data.data,
          ai: ai.data.data,
        });
      })
      .catch((err) => {
        if (!mounted) return;
        setError(err.response?.data?.message || err.message || '数据加载失败');
      })
      .finally(() => mounted && setLoading(false));
    return () => { mounted = false; };
  }, [range]);

  useEffect(() => { return fetchData(); }, [fetchData]);

  const handleRefresh = async () => {
    setRefreshing(true);
    setTaskMsg('');
    try {
      const res = await dashboardApi.refresh();
      const taskId = res.data?.data?.task_id;
      if (taskId) {
        setTaskMsg('统计刷新任务已提交，正在后台计算...');
        pollTask(taskId);
      } else {
        setTaskMsg('刷新请求已发送');
        setTimeout(() => setTaskMsg(''), 3000);
      }
    } catch (err: any) {
      setTaskMsg('刷新失败: ' + (err.response?.data?.message || err.message));
    } finally {
      setRefreshing(false);
    }
  };

  const pollTask = async (taskId: number) => {
    let attempts = 0;
    const maxAttempts = 30;
    const interval = setInterval(async () => {
      attempts++;
      if (attempts > maxAttempts) {
        clearInterval(interval);
        setTaskMsg('任务查询超时，数据可能仍在后台计算');
        return;
      }
      try {
        const res = await taskApi.get(taskId);
        const status = res.data?.data?.status;
        if (status === 'success') {
          clearInterval(interval);
          setTaskMsg('统计刷新完成');
          fetchData();
          setTimeout(() => setTaskMsg(''), 3000);
        } else if (status === 'failed') {
          clearInterval(interval);
          setTaskMsg('统计刷新失败: ' + (res.data?.data?.error_message || '未知错误'));
        }
      } catch {
        clearInterval(interval);
        setTaskMsg('任务查询失败');
      }
    }, 2000);
  };

  const maxTrend = useMemo(() => Math.max(1, ...data.jobTrend.map((i) => i.count), ...data.applicationTrend.map((i) => i.count)), [data]);
  const maxHotJob = useMemo(() => Math.max(1, ...data.hotJobs.map((i) => i.application_count)), [data.hotJobs]);
  const maxSkill = useMemo(() => Math.max(1, ...data.hotSkills.map((i) => i.count)), [data.hotSkills]);

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <div>
          <h2 style={{ margin: 0 }}>数据看板</h2>
          <p style={{ margin: '6px 0 0', color: '#667085' }}>岗位发布、投递、候选人转化与 AI 匹配统计</p>
        </div>
        <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
          <select value={range} onChange={(e) => setRange(e.target.value)} style={{ padding: '8px 12px', borderRadius: 8, border: '1px solid #d0d5dd' }}>
            <option value="week">最近 7 天</option>
            <option value="month">最近 30 天</option>
            <option value="year">最近 1 年</option>
          </select>
          <button onClick={handleRefresh} disabled={refreshing} style={{ padding: '8px 16px', borderRadius: 8, border: '1px solid #d0d5dd', background: refreshing ? '#f2f4f7' : '#fff', cursor: refreshing ? 'not-allowed' : 'pointer', fontSize: 14 }}>
            {refreshing ? '刷新中...' : '刷新统计'}
          </button>
        </div>
      </div>

      {taskMsg && <div style={{ padding: 12, marginBottom: 16, background: taskMsg.includes('失败') || taskMsg.includes('超时') ? '#fff1f0' : '#e6f7ff', color: taskMsg.includes('失败') || taskMsg.includes('超时') ? '#cf1322' : '#1677ff', borderRadius: 8 }}>{taskMsg}</div>}
      {error && <div style={{ padding: 12, marginBottom: 16, background: '#fff1f0', color: '#cf1322', borderRadius: 8 }}>{error}</div>}
      {loading && <div style={{ padding: 24, background: '#fff', borderRadius: 12 }}>数据加载中...</div>}

      {!loading && !error && (
        <>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16, marginBottom: 16 }}>
            {cards.map(([key, label]) => (
              <div key={key} style={cardStyle}>
                <div style={{ color: '#667085', fontSize: 13 }}>{label}</div>
                <div style={{ fontSize: 28, fontWeight: 700, marginTop: 8 }}>{data.overview?.[key] ?? 0}</div>
              </div>
            ))}
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 16 }}>
            <TrendPanel title="岗位发布趋势" items={data.jobTrend} max={maxTrend} color="#1677ff" />
            <TrendPanel title="投递趋势" items={data.applicationTrend} max={maxTrend} color="#52c41a" />
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 16 }}>
            <RankPanel title="热门岗位 Top 10" items={data.hotJobs.map((i) => ({ name: i.title, sub: `${i.location} · ${i.salary}`, value: i.application_count }))} max={maxHotJob} />
            <RankPanel title="热门技能 Top 10" items={data.hotSkills.map((i) => ({ name: i.skill, value: i.count }))} max={maxSkill} />
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
            <FunnelPanel data={data.funnel} />
            <AIPanel data={data.ai} />
          </div>
        </>
      )}
    </div>
  );
}

const cardStyle: React.CSSProperties = { padding: 18, background: '#fff', borderRadius: 12, boxShadow: '0 4px 16px rgba(16,24,40,.06)' };
const panelStyle: React.CSSProperties = { padding: 18, background: '#fff', borderRadius: 12, boxShadow: '0 4px 16px rgba(16,24,40,.06)' };

function TrendPanel({ title, items, max, color }: { title: string; items: TrendPoint[]; max: number; color: string }) {
  if (items.length === 0) {
    return (
      <div style={panelStyle}>
        <h3 style={{ marginTop: 0 }}>{title}</h3>
        <Empty hint={color === '#1677ff' ? '当前时间范围内暂无新发布的岗位，可切换时间范围查看' : '当前时间范围内暂无新的投递记录'} />
      </div>
    );
  }

  const chartH = 150;
  const padTop = 10;
  const padBottom = 24;
  const plotH = chartH - padTop - padBottom;
  const step = 100 / Math.max(1, items.length - 1);

  const points = items.map((item, i) => ({
    x: i * step,
    y: padTop + plotH - (item.count / max) * plotH,
    date: item.date,
    count: item.count,
  }));

  const linePath = points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ');
  const areaPath = `${linePath} L ${points[points.length - 1].x} ${padTop + plotH} L ${points[0].x} ${padTop + plotH} Z`;

  return (
    <div style={panelStyle}>
      <h3 style={{ marginTop: 0 }}>{title}</h3>
      <svg viewBox="0 0 100 ${chartH}" preserveAspectRatio="none" style={{ width: '100%', height: 180 }}>
        <defs>
          <linearGradient id={`grad-${color}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity="0.25" />
            <stop offset="100%" stopColor={color} stopOpacity="0.02" />
          </linearGradient>
        </defs>
        <path d={areaPath} fill={`url(#grad-${color})`} />
        <path d={linePath} fill="none" stroke={color} strokeWidth="0.8" strokeLinejoin="round" strokeLinecap="round" />
        {points.map((p, i) => (
          <circle key={i} cx={p.x} cy={p.y} r="1.2" fill={color} />
        ))}
      </svg>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 2 }}>
        {items.map((item, i) => (
          <span key={i} style={{ fontSize: 11, color: '#667085', flex: 1, textAlign: i === 0 ? 'left' : i === items.length - 1 ? 'right' : 'center' }}>
            {item.date.slice(5)}
          </span>
        ))}
      </div>
    </div>
  );
}

function RankPanel({ title, items, max }: { title: string; items: { name: string; sub?: string; value: number }[]; max: number }) {
  return <div style={panelStyle}><h3 style={{ marginTop: 0 }}>{title}</h3>{items.length === 0 ? <Empty hint={title.includes('技能') ? '候选人尚未填写技能信息，或暂无关联的候选人资料' : '岗位暂无投递记录'} /> : items.map((item) => <div key={item.name} style={{ marginBottom: 12 }}><div style={{ display: 'flex', justifyContent: 'space-between' }}><span>{item.name}<span style={{ color: '#98a2b3', marginLeft: 8 }}>{item.sub}</span></span><b>{item.value}</b></div><div style={{ height: 8, background: '#f2f4f7', borderRadius: 999, marginTop: 6 }}><div style={{ width: `${item.value / max * 100}%`, height: 8, background: '#1677ff', borderRadius: 999 }} /></div></div>)}</div>;
}

function FunnelPanel({ data }: { data: Record<string, number> | null }) {
  const items = [['registered', '注册'], ['profile_complete', '完善资料'], ['resume_uploaded', '上传简历'], ['applied', '完成投递']];
  const max = Math.max(1, ...(data ? items.map(([k]) => data[k] || 0) : [0]));
  return <div style={panelStyle}><h3 style={{ marginTop: 0 }}>候选人转化漏斗</h3>{items.map(([k, label]) => <div key={k} style={{ marginBottom: 12 }}><div style={{ display: 'flex', justifyContent: 'space-between' }}><span>{label}</span><b>{data?.[k] || 0}</b></div><div style={{ height: 18, background: '#f2f4f7', borderRadius: 999, marginTop: 6 }}><div style={{ width: `${((data?.[k] || 0) / max) * 100}%`, height: 18, background: '#722ed1', borderRadius: 999 }} /></div></div>)}</div>;
}

function AIPanel({ data }: { data: Record<string, number> | null }) {
  const hasData = data && (data.match_tasks || 0) > 0;
  const hint = '请在投递详情页点击AI匹配按钮触发匹配，统计数据将在此展示';
  if (!hasData) {
    return (
      <div style={panelStyle}>
        <h3 style={{ marginTop: 0 }}>AI 匹配统计</h3>
        <Empty hint={hint} />
      </div>
    );
  }
  return (
    <div style={panelStyle}>
      <h3 style={{ marginTop: 0 }}>AI 匹配统计</h3>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 12 }}>
        <Metric label="匹配任务数" value={data?.match_tasks || 0} />
        <Metric label="成功数量" value={data?.success_tasks || 0} />
        <Metric label="失败数量" value={data?.failed_tasks || 0} />
        <Metric label="平均匹配分" value={(data?.average_score || 0).toFixed(1)} />
      </div>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: number | string }) {
  return <div style={{ padding: 14, background: '#f8fafc', borderRadius: 10 }}><div style={{ color: '#667085', fontSize: 13 }}>{label}</div><div style={{ fontSize: 24, fontWeight: 700, marginTop: 6 }}>{value}</div></div>;
}

function Empty({ hint }: { hint?: string }) {
  return <div style={{ padding: 32, textAlign: 'center', color: '#98a2b3', background: '#f8fafc', borderRadius: 10 }}>
    <div style={{ fontSize: 14, marginBottom: 4 }}>暂无数据</div>
    {hint && <div style={{ fontSize: 12, color: '#b0b5bd' }}>{hint}</div>}
  </div>;
}
