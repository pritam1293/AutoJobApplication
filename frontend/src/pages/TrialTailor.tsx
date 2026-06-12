import { useState } from 'react';
import { trialTailor } from '../services/api';
import { FlaskConical, Loader } from 'lucide-react';

export default function TrialTailor() {
  const [jd, setJd] = useState('');
  const [title, setTitle] = useState('');
  const [company, setCompany] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{
    tailored_resume: string;
    match_score: number;
    missing_skills: string;
    notes: string;
    tailored_pdf?: string;
    latex_source?: string;
  } | null>(null);
  const [error, setError] = useState('');

  const handleTailor = async () => {
    if (!jd.trim()) return;
    setLoading(true);
    setError('');
    setResult(null);
    try {
      const data = await trialTailor(jd, title, company);
      setResult(data);
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } };
      setError(err?.response?.data?.error || 'Tailoring failed');
    }
    setLoading(false);
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-2 flex items-center gap-2">
        <FlaskConical className="w-6 h-6 text-yellow-400" /> Trial Tailor
      </h1>
      <p className="text-sm text-gray-400 mb-6">
        Test resume tailoring against any job description. Results here don't get saved.
      </p>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Input */}
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <div className="mb-4">
            <label className="block text-sm text-gray-400 mb-1">Job Title (optional)</label>
            <input
              className="w-full bg-gray-950 border border-gray-700 rounded-lg p-2 text-sm text-gray-200"
              placeholder="Software Engineer"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
          </div>
          <div className="mb-4">
            <label className="block text-sm text-gray-400 mb-1">Company (optional)</label>
            <input
              className="w-full bg-gray-950 border border-gray-700 rounded-lg p-2 text-sm text-gray-200"
              placeholder="Arista Networks"
              value={company}
              onChange={(e) => setCompany(e.target.value)}
            />
          </div>
          <div className="mb-4">
            <label className="block text-sm text-gray-400 mb-1">Job Description *</label>
            <textarea
              className="w-full h-72 bg-gray-950 border border-gray-700 rounded-lg p-3 text-sm text-gray-200 font-mono"
              placeholder="Paste the full job description here..."
              value={jd}
              onChange={(e) => setJd(e.target.value)}
            />
          </div>

          {error && (
            <div className="mb-4 text-red-400 text-sm">{error}</div>
          )}

          <button
            className="bg-yellow-600 hover:bg-yellow-500 text-white px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50 flex items-center gap-2"
            onClick={handleTailor}
            disabled={!jd.trim() || loading}
          >
            {loading ? <Loader className="w-4 h-4 animate-spin" /> : <FlaskConical className="w-4 h-4" />}
            {loading ? 'Tailoring...' : 'Trial Tailor'}
          </button>
        </div>

        {/* Result */}
        <div className="space-y-4">
          {result && (
            <>
              {/* Score card */}
              <div className="bg-gray-900 border border-gray-800 rounded-xl p-4">
                <div className="flex items-center gap-4">
                  <div className="text-3xl font-bold text-blue-400">{result.match_score?.toFixed(0) || '?'}%</div>
                  <div>
                    <div className="font-semibold">Match Score</div>
                    {result.missing_skills && (
                      <div className="text-xs text-gray-400 mt-1">
                        Missing: {result.missing_skills}
                      </div>
                    )}
                  </div>
                </div>
                {result.notes && (
                  <div className="mt-2 text-sm text-gray-400">{result.notes}</div>
                )}
                {result.tailored_pdf && (
                  <a
                    href={`http://localhost:8080/${result.tailored_pdf}`}
                    target="_blank"
                    className="mt-3 inline-block text-sm text-blue-400 hover:underline"
                  >
                    Download PDF
                  </a>
                )}
                {result.latex_source && (
                  <div className="mt-2 text-xs text-gray-500">
                    LaTeX source available (install texlive for PDF compilation)
                  </div>
                )}
              </div>

              {/* Tailored resume preview */}
              <div className="bg-gray-900 border border-gray-800 rounded-xl p-4">
                <h3 className="font-semibold mb-2">Tailored Resume</h3>
                <pre className="text-xs text-gray-300 whitespace-pre-wrap max-h-96 overflow-y-auto font-mono">
                  {result.tailored_resume}
                </pre>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
