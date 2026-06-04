import axios from 'axios';

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('hr_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('hr_token');
      localStorage.removeItem('hr_user');
      window.location.href = '/login';
    }
    return Promise.reject(err);
  }
);

export const authApi = {
  register: (data: { username: string; password: string; name: string }) =>
    api.post('/hr/register', data),
  login: (data: { username: string; password: string }) =>
    api.post('/hr/login', data),
};

export const jobApi = {
  create: (data: { title: string; description: string; requirement: string; salary: string; location: string }) =>
    api.post('/hr/jobs', data),
  update: (id: number, data: { title: string; description: string; requirement: string; salary: string; location: string }) =>
    api.put(`/hr/jobs/${id}`, data),
  offline: (id: number) =>
    api.put(`/hr/jobs/${id}/offline`),
  list: (page = 1, pageSize = 10) =>
    api.get('/hr/jobs', { params: { page, pageSize } }),
  listPublic: (page = 1, pageSize = 10) =>
    api.get('/jobs', { params: { page, pageSize } }),
  getApplications: (jobId: number, page = 1, pageSize = 10) =>
    api.get(`/hr/jobs/${jobId}/applications`, { params: { page, pageSize } }),
  getApplicationDetail: (appId: number) =>
    api.get(`/hr/applications/${appId}/detail`),
};

export const chatApi = {
  send: (content: string) =>
    api.post('/hr/chat', { content }),
  history: () =>
    api.get('/hr/chat/history'),
};

export const dashboardApi = {
  overview: () =>
    api.get('/hr/dashboard/overview'),
  jobTrend: (params?: { startDate?: string; endDate?: string; range?: string; jobId?: number }) =>
    api.get('/hr/dashboard/job-trend', { params }),
  applicationTrend: (params?: { startDate?: string; endDate?: string; range?: string; jobId?: number }) =>
    api.get('/hr/dashboard/application-trend', { params }),
  hotJobs: (params?: { limit?: number; range?: string }) =>
    api.get('/hr/dashboard/hot-jobs', { params }),
  hotSkills: (params?: { limit?: number; range?: string }) =>
    api.get('/hr/dashboard/hot-skills', { params }),
  candidateFunnel: () =>
    api.get('/hr/dashboard/candidate-funnel'),
  aiMatchStats: () =>
    api.get('/hr/dashboard/ai-match-stats'),
  refresh: () =>
    api.post('/hr/dashboard/refresh'),
};

export const taskApi = {
  get: (id: number) =>
    api.get(`/tasks/${id}`),
  triggerAIMatch: (applicationId: number) =>
    api.post(`/hr/applications/${applicationId}/match`),
  triggerResumeParse: (resumeId: number) =>
    api.post(`/candidate/resume/${resumeId}/parse`),
};

export default api;
