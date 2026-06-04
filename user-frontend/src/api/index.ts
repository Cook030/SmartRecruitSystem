import axios from 'axios';

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('candidate_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('candidate_token');
      localStorage.removeItem('candidate_user');
      window.location.href = '/login';
    }
    return Promise.reject(err);
  }
);

export const authApi = {
  register: (data: { username: string; password: string }) =>
    api.post('/candidate/register', data),
  login: (data: { username: string; password: string }) =>
    api.post('/candidate/login', data),
};

export const jobApi = {
  listPublic: (page = 1, pageSize = 10) =>
    api.get('/jobs', { params: { page, pageSize } }),
};

export const profileApi = {
  get: () => api.get('/candidate/profile'),
  update: (data: { name: string; phone: string; education: string; university: string; experience: string; skills: string }) =>
    api.put('/candidate/profile', data),
};

export const resumeApi = {
  upload: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return api.post('/candidate/resume/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },
  get: () => api.get('/candidate/resume'),
  delete: () => api.delete('/candidate/resume'),
};

export const applicationApi = {
  apply: (jobId: number) => api.post(`/candidate/apply/${jobId}`),
  list: (page = 1, pageSize = 10) =>
    api.get('/candidate/applications', { params: { page, pageSize } }),
  checkApplied: async (jobIds: number[]) => {
    if (jobIds.length === 0) return [];
    const res = await api.get('/candidate/applications', { params: { page: 1, pageSize: 1000 } });
    const appliedJobIds = new Set((res.data.data.applications || []).map((app: any) => app.jobId));
    return jobIds.filter(id => appliedJobIds.has(id));
  },
};

export const taskApi = {
  get: (id: number) =>
    api.get(`/tasks/${id}`),
  triggerResumeParse: (resumeId: number) =>
    api.post(`/candidate/resume/${resumeId}/parse`),
  getResumeParseResult: (resumeId: number) =>
    api.get(`/candidate/resume/${resumeId}/parse-result`),
  deleteResumeParseResult: (resumeId: number) =>
    api.delete(`/candidate/resume/${resumeId}/parse-result`),
};

export default api;
