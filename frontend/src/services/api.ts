import axios from 'axios';

const api = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
  headers: { 'Content-Type': 'application/json' },
});

export interface Job {
  ID: number;
  CreatedAt: string;
  platform: string;
  job_id: string;
  title: string;
  company: string;
  location: string;
  url: string;
  description: string;
  salary: string;
  posted_at: string;
  skills: string;
  status: string;
}

export interface Application {
  ID: number;
  CreatedAt: string;
  job_id: number;
  user_id: number;
  job: Job;
  status: string;
  resume_used: string;
  score: number;
  applied_at: string;
  notes: string;
  tailored_jd: string;
}

export interface SearchQuery {
  ID: number;
  query: string;
  location: string;
  platforms: string;
  active: boolean;
  auto_apply: boolean;
  max_applied: number;
}

export interface Analytics {
  TotalJobs: number;
  NewJobs: number;
  AppliedJobs: number;
  FailedApps: number;
  SuccessApps: number;
}

export interface SearchResult {
  total: number;
  jobs: Job[];
}

export interface TailorResponse {
  tailored_resume: string;
  match_score: number;
  missing_skills: string;
  notes: string;
  tailored_pdf?: string;
  latex_source?: string;
}

export const uploadLatex = async (latex: string) => {
  const { data } = await api.post('/resume/latex', { latex });
  return data;
};

export const getLatexSource = async () => {
  const { data } = await api.get<{ latex_source: string }>('/resume/latex');
  return data;
};

export const searchJobs = async (query: string, location: string) => {
  const { data } = await api.post<SearchResult>('/jobs/search', { query, location });
  return data;
};

export const listJobs = async (status?: string, platform?: string) => {
  const params = new URLSearchParams();
  if (status) params.append('status', status);
  if (platform) params.append('platform', platform);
  const { data } = await api.get<{ total: number; jobs: Job[] }>(`/jobs?${params}`);
  return data;
};

export const getJob = async (id: number) => {
  const { data } = await api.get<Job>(`/jobs/${id}`);
  return data;
};

export const getJobDetails = async (id: number) => {
  const { data } = await api.get<{ id: number; description: string }>(`/jobs/${id}/details`);
  return data;
};

export const updateJobStatus = async (id: number, status: string) => {
  await api.put(`/jobs/${id}/status`, { status });
};

export const applyToJob = async (jobId: number) => {
  const { data } = await api.post(`/applications/apply/${jobId}`);
  return data;
};

export const listApplications = async () => {
  const { data } = await api.get<{ total: number; applications: Application[] }>('/applications');
  return data;
};

export const uploadResume = async (file: File) => {
  const formData = new FormData();
  formData.append('resume', file);
  const { data } = await api.post('/resume/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
  return data;
};

export const tailorResume = async (jobId: number, resumeData: string, instructions?: string) => {
  const { data } = await api.post<TailorResponse>(`/resume/tailor/${jobId}`, {
    resume_data: resumeData,
    instructions: instructions || '',
  });
  return data;
};

export const createSearchQuery = async (query: string, location: string, platforms?: string, autoApply?: boolean) => {
  const { data } = await api.post<SearchQuery>('/search-queries', {
    query, location, platforms, auto_apply: autoApply,
  });
  return data;
};

export const listSearchQueries = async () => {
  const { data } = await api.get<{ total: number; queries: SearchQuery[] }>('/search-queries');
  return data;
};

export const getSettings = async () => {
  const { data } = await api.get('/settings');
  return data;
};

export const updateSettings = async (settings: Record<string, string>) => {
  const { data } = await api.put('/settings', {
    linkedin_email: settings.linkedin_email || '',
    linkedin_password: settings.linkedin_password || '',
    google_ai_key: settings.google_ai_key || '',
  });
  return data;
};

export const toggleAutoApply = async (enabled: boolean) => {
  const { data } = await api.post('/settings/auto-apply', { enabled });
  return data;
};

export const getAnalytics = async () => {
  const { data } = await api.get<Analytics>('/analytics');
  return data;
};

export default api;
