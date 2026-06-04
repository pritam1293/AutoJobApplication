import { useEffect, useState } from 'react';
import { listApplications } from '../services/api';
import type { Application } from '../services/api';

export default function Applications() {
  const [apps, setApps] = useState<Application[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    listApplications()
      .then((data) => setApps(data.applications))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="text-gray-400">Loading...</div>;

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Applications</h1>
      {apps.length === 0 ? (
        <div className="text-gray-500 text-center py-12">No applications yet.</div>
      ) : (
        <div className="space-y-3">
          {apps.map((app) => (
            <div key={app.ID} className="bg-gray-900 border border-gray-800 rounded-xl p-4">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="font-semibold">{app.Job?.Title || 'Unknown Job'}</h3>
                  <p className="text-sm text-gray-400">{app.Job?.Company}</p>
                </div>
                <div className="flex items-center gap-2">
                  {app.Score > 0 && (
                    <span className="text-xs text-yellow-400">Match: {Math.round(app.Score)}%</span>
                  )}
                  <span className={`text-xs px-2 py-1 rounded ${
                    app.Status === 'success' ? 'bg-green-900 text-green-300' :
                    app.Status === 'failed' ? 'bg-red-900 text-red-300' :
                    'bg-yellow-900 text-yellow-300'
                  }`}>
                    {app.Status}
                  </span>
                </div>
              </div>
              {app.Notes && <p className="text-xs text-gray-500 mt-2">{app.Notes}</p>}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
