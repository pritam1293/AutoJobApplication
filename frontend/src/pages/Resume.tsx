import { useState } from 'react';
import { uploadResume } from '../services/api';
import { Upload, CheckCircle, AlertCircle } from 'lucide-react';

export default function Resume() {
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [result, setResult] = useState<{ message: string; pages: number; size: number } | null>(null);
  const [error, setError] = useState('');

  const handleUpload = async () => {
    if (!file) return;
    setUploading(true);
    setError('');
    try {
      const data = await uploadResume(file);
      setResult(data);
    } catch {
      setError('Upload failed. Make sure the file is a valid PDF.');
    }
    setUploading(false);
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Resume</h1>

      <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 max-w-xl">
        <div className="border-2 border-dashed border-gray-700 rounded-xl p-8 text-center">
          <Upload className="w-8 h-8 mx-auto mb-3 text-gray-500" />
          <p className="text-sm text-gray-400 mb-4">Upload your base resume (PDF)</p>
          <input
            type="file"
            accept=".pdf"
            className="block w-full text-sm text-gray-400 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:bg-blue-600 file:text-white hover:file:bg-blue-700"
            onChange={(e) => setFile(e.target.files?.[0] || null)}
          />
        </div>

        {file && (
          <div className="mt-4 text-sm text-gray-400">
            Selected: {file.name} ({(file.size / 1024).toFixed(1)} KB)
          </div>
        )}

        {error && (
          <div className="mt-4 flex items-center gap-2 text-red-400 text-sm">
            <AlertCircle className="w-4 h-4" /> {error}
          </div>
        )}

        {result && (
          <div className="mt-4 flex items-center gap-2 text-green-400 text-sm">
            <CheckCircle className="w-4 h-4" /> {result.message} ({result.pages} pages, {(result.size / 1024).toFixed(1)} KB)
          </div>
        )}

        <button
          className="mt-4 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
          onClick={handleUpload}
          disabled={!file || uploading}
        >
          {uploading ? 'Uploading...' : 'Upload Resume'}
        </button>
      </div>

      <div className="mt-8 bg-gray-900 border border-gray-800 rounded-xl p-6 max-w-xl">
        <h2 className="text-lg font-semibold mb-2">Resume Tailoring</h2>
        <p className="text-sm text-gray-400">
          After uploading your base resume, go to a job listing and use the "Tailor Resume" option
          to generate an ATS-optimized version matching the job description.
        </p>
      </div>
    </div>
  );
}
