import { useState } from 'react';
import { searchJobs } from '../services/api';
import { Search as SearchIcon, Loader2 } from 'lucide-react';

interface JobResult {
  platform: string;
  job_id: string;
  title: string;
  company: string;
  location: string;
  url: string;
  description?: string;
  salary?: string;
}

export default function Search() {
  const [query, setQuery] = useState('software engineer');
  const [location, setLocation] = useState('');
  const [jobs, setJobs] = useState<JobResult[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSearch = async () => {
    setLoading(true);
    setError('');
    try {
      const data = await searchJobs(query, location);
      setJobs(data.jobs || []);
      setTotal(data.total || 0);
    } catch (e) {
      setError((e as Error)?.message || 'Search failed. Check your credentials.');
      setJobs([]);
      setTotal(0);
    }
    setLoading(false);
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Search Jobs</h1>

      <div className="bg-gray-900 border border-gray-800 rounded-xl p-4 mb-6">
        <div className="flex flex-wrap gap-3">
          <input
            className="flex-1 min-w-[200px] bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
            placeholder="Job title, keywords..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          />
          <input
            className="flex-1 min-w-[150px] bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
            placeholder="Location"
            value={location}
            onChange={(e) => setLocation(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          />
          <button
            className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium flex items-center gap-2"
            onClick={handleSearch}
            disabled={loading}
          >
            {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <SearchIcon className="w-4 h-4" />}
            Search
          </button>
        </div>
      </div>

      {error && <div className="text-red-400 text-sm mb-4">{error}</div>}

      {total > 0 && (
        <div className="text-sm text-gray-400 mb-4">Found {total} job{total !== 1 ? 's' : ''}</div>
      )}

      <div className="space-y-3">
        {jobs.map((job, i) => (
          <div key={`${job.platform}-${job.job_id}-${i}`} className="bg-gray-900 border border-gray-800 rounded-xl p-4 hover:border-gray-700 transition-colors">
            <div className="flex items-start justify-between">
              <div>
                <h3 className="font-semibold">{job.title || '(no title)'}</h3>
                <p className="text-sm text-gray-400">{job.company || '(no company)'} &middot; {job.location || '(no location)'}</p>
              </div>
              <span className="text-xs px-2 py-1 rounded bg-blue-900 text-blue-300">
                {job.platform || 'LinkedIn'}
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
