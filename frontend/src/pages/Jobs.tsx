import { useCallback, useEffect, useState } from 'react';
import { listJobs, updateJobStatus, getJobDetails, applyToJob, tailorResume } from '../services/api';
import type { Job } from '../services/api';
import { Loader2 } from 'lucide-react';

export default function Jobs() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [filter, setFilter] = useState('');
  const [loading, setLoading] = useState(true);
  const [detailJobId, setDetailJobId] = useState<number | null>(null);
  const [detailText, setDetailText] = useState('');
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [applying, setApplying] = useState<number | null>(null);
  const [tailoring, setTailoring] = useState<number | null>(null);
  const [tailoredResult, setTailoredResult] = useState<Record<number, { tailored_resume: string; match_score: number; missing_skills: string; notes: string; tailored_pdf?: string }>>({});
  const [messages, setMessages] = useState<Record<number, { type: 'success' | 'error' | 'info'; text: string }>>({});

  const load = useCallback(async () => {
    setLoading(true);
    const data = await listJobs(filter || undefined);
    setJobs(data.jobs);
    setLoading(false);
  }, [filter]);

  useEffect(() => { load(); }, [load]);

  const setMsg = (id: number, type: 'success' | 'error' | 'info', text: string) => {
    setMessages(prev => ({ ...prev, [id]: { type, text } }));
    setTimeout(() => setMessages(prev => { const n = { ...prev }; delete n[id]; return n; }), 5000);
  };

  const handleStatusChange = async (id: number, status: string) => {
    await updateJobStatus(id, status);
    setMsg(id, 'info', `Marked as ${status}`);
    load();
  };

  const handleViewDetails = async (id: number) => {
    if (detailJobId === id) {
      setDetailJobId(null);
      setDetailText('');
      return;
    }
    setLoadingDetail(true);
    setDetailJobId(id);
    try {
      const data = await getJobDetails(id);
      setDetailText(data.description || '(no description)');
    } catch {
      setDetailText('Failed to load details.');
    }
    setLoadingDetail(false);
  };

  const handleApply = async (jobId: number) => {
    setApplying(jobId);
    try {
      const result = await applyToJob(jobId);
      if (result.status === 'success') {
        setMsg(jobId, 'success', `Applied successfully! ${result.message}`);
      } else {
        setMsg(jobId, 'error', `Apply ${result.status}: ${result.message}`);
      }
      load();
    } catch (e) {
      setMsg(jobId, 'error', `Apply failed: ${(e as Error).message}`);
    }
    setApplying(null);
  };

  const handleTailor = async (jobId: number) => {
    setTailoring(jobId);
    try {
      const result = await tailorResume(jobId, '');
      if (result.tailored_resume) {
        setTailoredResult(prev => ({ ...prev, [jobId]: result }));
        setMsg(jobId, 'success', `Match score: ${Math.round(result.match_score)}%. Missing: ${result.missing_skills || 'none'}`);
      } else {
        setMsg(jobId, 'error', 'No tailored resume returned. Upload a resume first.');
      }
    } catch (e) {
      setMsg(jobId, 'error', `Tailor failed: ${(e as Error).message}`);
    }
    setTailoring(null);
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Jobs</h1>
        <select
          className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        >
          <option value="">All Status</option>
          <option value="new">New</option>
          <option value="matched">Matched</option>
          <option value="applied">Applied</option>
          <option value="rejected">Rejected</option>
          <option value="skipped">Skipped</option>
        </select>
      </div>

      {loading ? (
        <div className="text-gray-400">Loading...</div>
      ) : jobs.length === 0 ? (
        <div className="text-gray-500 text-center py-12">No jobs found. Try searching first.</div>
      ) : (
        <div className="space-y-3">
          {jobs.map((job) => (
            <div key={job.ID} className="bg-gray-900 border border-gray-800 rounded-xl p-4">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <h3 className="font-semibold">{job.title}</h3>
                  <p className="text-sm text-gray-400">{job.company} &middot; {job.location}</p>
                  {job.salary && <p className="text-xs text-green-400 mt-1">{job.salary}</p>}
                </div>
                <div className="flex items-center gap-2">
                  <span className={`text-xs px-2 py-1 rounded ${
                    job.status === 'new' ? 'bg-blue-900 text-blue-300' :
                    job.status === 'applied' ? 'bg-green-900 text-green-300' :
                    job.status === 'rejected' ? 'bg-red-900 text-red-300' :
                    'bg-gray-700 text-gray-300'
                  }`}>
                    {job.status}
                  </span>
                  <span className={`text-xs px-2 py-1 rounded ${
                    job.platform === 'linkedin' ? 'bg-blue-900 text-blue-300' : 'bg-orange-900 text-orange-300'
                  }`}>
                    {job.platform}
                  </span>
                </div>
              </div>
              {detailJobId === job.ID && (
                <div className="mt-3 bg-gray-800 rounded-lg p-3 text-sm text-gray-300 max-h-60 overflow-y-auto">
                  {loadingDetail ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <pre className="whitespace-pre-wrap font-sans">{detailText}</pre>
                  )}
                </div>
              )}
              {tailoredResult[job.ID] && (
                <div className="mt-3 bg-gray-850 border border-purple-800 rounded-lg p-3">
                  <div className="flex items-center gap-3 mb-2 text-xs">
                    <span className="text-purple-300 font-semibold">Match: {Math.round(tailoredResult[job.ID].match_score)}%</span>
                    <span className="text-gray-400">Missing: {tailoredResult[job.ID].missing_skills || 'none'}</span>
                    {tailoredResult[job.ID].tailored_pdf && (
                      <a href={`http://localhost:8080/${tailoredResult[job.ID].tailored_pdf}`} target="_blank" rel="noopener noreferrer" className="text-blue-400 underline">Download PDF</a>
                    )}
                  </div>
                  {tailoredResult[job.ID].notes && (
                    <p className="text-xs text-gray-400 mb-2">{tailoredResult[job.ID].notes}</p>
                  )}
                  <pre className="text-sm text-gray-300 max-h-80 overflow-y-auto whitespace-pre-wrap font-sans">{tailoredResult[job.ID].tailored_resume}</pre>
                </div>
              )}
              <div className="mt-3 flex gap-2 flex-wrap">
                <button
                  className="text-xs px-2 py-1 rounded bg-blue-700 hover:bg-blue-600"
                  onClick={() => handleViewDetails(job.ID)}
                >
                  {detailJobId === job.ID ? 'Hide' : 'Details'}
                </button>
                <button
                  className="text-xs px-2 py-1 rounded bg-purple-700 hover:bg-purple-600 disabled:opacity-50 flex items-center gap-1"
                  onClick={() => handleTailor(job.ID)}
                  disabled={tailoring === job.ID}
                >
                  {tailoring === job.ID ? <Loader2 className="w-3 h-3 animate-spin" /> : null}
                  Tailor
                </button>
                <button
                  className="text-xs px-2 py-1 rounded bg-green-700 hover:bg-green-600 disabled:opacity-50 flex items-center gap-1"
                  onClick={() => handleApply(job.ID)}
                  disabled={applying === job.ID || job.status === 'applied'}
                >
                  {applying === job.ID ? <Loader2 className="w-3 h-3 animate-spin" /> : null}
                  {job.status === 'applied' ? 'Applied' : 'Apply'}
                </button>
                <button
                  className="text-xs px-2 py-1 rounded bg-red-700 hover:bg-red-600"
                  onClick={() => handleStatusChange(job.ID, 'skipped')}
                >
                  Skip
                </button>
              </div>
              {messages[job.ID] && (
                <div className={`mt-2 text-xs ${
                  messages[job.ID].type === 'success' ? 'text-green-400' :
                  messages[job.ID].type === 'error' ? 'text-red-400' :
                  'text-yellow-400'
                }`}>
                  {messages[job.ID].text}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
