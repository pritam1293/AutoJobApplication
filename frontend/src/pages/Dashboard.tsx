import { useEffect, useState } from 'react';
import { getAnalytics } from '../services/api';
import { BarChart3, Briefcase, CheckCircle, XCircle, Clock } from 'lucide-react';

interface Analytics {
  TotalJobs: number;
  NewJobs: number;
  AppliedJobs: number;
  FailedApps: number;
  SuccessApps: number;
}

export default function Dashboard() {
  const [analytics, setAnalytics] = useState<Analytics | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getAnalytics().then(setAnalytics).finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="text-gray-400">Loading...</div>;

  const cards = [
    { label: 'Total Jobs', value: analytics?.TotalJobs ?? 0, icon: Briefcase, color: 'text-blue-400' },
    { label: 'New Jobs', value: analytics?.NewJobs ?? 0, icon: Clock, color: 'text-yellow-400' },
    { label: 'Applied', value: analytics?.AppliedJobs ?? 0, icon: CheckCircle, color: 'text-green-400' },
    { label: 'Successful', value: analytics?.SuccessApps ?? 0, icon: BarChart3, color: 'text-emerald-400' },
    { label: 'Failed', value: analytics?.FailedApps ?? 0, icon: XCircle, color: 'text-red-400' },
  ];

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-4">
        {cards.map(({ label, value, icon: Icon, color }) => (
          <div key={label} className="bg-gray-900 border border-gray-800 rounded-xl p-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm text-gray-400">{label}</span>
              <Icon className={`w-5 h-5 ${color}`} />
            </div>
            <span className="text-3xl font-bold">{value}</span>
          </div>
        ))}
      </div>

      <div className="mt-8 bg-gray-900 border border-gray-800 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Getting Started</h2>
        <ol className="space-y-2 text-sm text-gray-400 list-decimal list-inside">
          <li>Go to <strong>Settings</strong> to configure your LinkedIn/Indeed credentials and OpenAI API key</li>
          <li>Upload your resume in the <strong>Resume</strong> section</li>
          <li>Use <strong>Search Jobs</strong> to find software engineering positions</li>
          <li>Review and tailor your resume for specific jobs</li>
          <li>Use <strong>Apply</strong> to auto-submit applications</li>
        </ol>
      </div>
    </div>
  );
}
