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

  const load = useCallback(async () => {
    setLoading(true);
    const data = await listJobs(filter || undefined);
    setJobs(data.jobs);
    setLoading(false);
  }, [filter]);

  useEffect(() => { load(); }, [load]);

  const handleStatusChange = async (id: number, status: string) => {
    await updateJobStatus(id, status);
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
      alert(`Apply result: ${result.status} - ${result.message}`);
      load();
    } catch (e) {
      alert(`Apply failed: ${(e as Error).message}`);
    }
    setApplying(null);
  };

  const handleTailor = async (jobId: number) => {
    setTailoring(jobId);
    try {
      const result = await tailorResume(jobId, '');
      if (result.tailored_resume) {
        alert(`Tailored resume ready!\nMatch score: ${result.match_score}%\nMissing skills: ${result.missing_skills}`);
      } else {
        alert('No tailored resume returned. Upload a resume first.');
      }
    } catch (e) {
      alert(`Tailor failed: ${(e as Error).message}`);
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
                  <h3 className="font-semibold">{job.Title}</h3>
                  <p className="text-sm text-gray-400">{job.Company} &middot; {job.Location}</p>
                  {job.Salary && <p className="text-xs text-green-400 mt-1">{job.Salary}</p>}
                </div>
                <div className="flex items-center gap-2">
                  <span className={`text-xs px-2 py-1 rounded ${
                    job.Status === 'new' ? 'bg-blue-900 text-blue-300' :
                    job.Status === 'applied' ? 'bg-green-900 text-green-300' :
                    job.Status === 'rejected' ? 'bg-red-900 text-red-300' :
                    'bg-gray-700 text-gray-300'
                  }`}>
                    {job.Status}
                  </span>
                  <span className={`text-xs px-2 py-1 rounded ${
                    job.Platform === 'linkedin' ? 'bg-blue-900 text-blue-300' : 'bg-orange-900 text-orange-300'
                  }`}>
                    {job.Platform}
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
                  disabled={applying === job.ID || job.Status === 'applied'}
                >
                  {applying === job.ID ? <Loader2 className="w-3 h-3 animate-spin" /> : null}
                  {job.Status === 'applied' ? 'Applied' : 'Apply'}
                </button>
                <button
                  className="text-xs px-2 py-1 rounded bg-red-700 hover:bg-red-600"
                  onClick={() => handleStatusChange(job.ID, 'skipped')}
                >
                  Skip
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
