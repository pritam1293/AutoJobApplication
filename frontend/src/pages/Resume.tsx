import { useState, useEffect } from 'react';
import { uploadResume, uploadLatex, getLatexSource } from '../services/api';
import { Upload, CheckCircle, AlertCircle, FileCode } from 'lucide-react';

export default function Resume() {
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [result, setResult] = useState<{ message: string; pages: number; size: number } | null>(null);
  const [error, setError] = useState('');

  const [latexSource, setLatexSource] = useState('');
  const [savingLatex, setSavingLatex] = useState(false);
  const [latexSaved, setLatexSaved] = useState(false);
  const [latexError, setLatexError] = useState('');

  useEffect(() => {
    (async () => {
      try {
        const data = await getLatexSource();
        if (data.latex_source) setLatexSource(data.latex_source);
      } catch { }
    })();
  }, []);

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

  const handleSaveLatex = async () => {
    if (!latexSource.trim()) return;
    setSavingLatex(true);
    setLatexError('');
    setLatexSaved(false);
    try {
      await uploadLatex(latexSource);
      setLatexSaved(true);
      setTimeout(() => setLatexSaved(false), 5000);
    } catch {
      setLatexError('Failed to save LaTeX source.');
    }
    setSavingLatex(false);
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Resume</h1>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* PDF Upload */}
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
            <Upload className="w-5 h-5" /> PDF Resume
          </h2>
          <div className="border-2 border-dashed border-gray-700 rounded-xl p-8 text-center">
            <Upload className="w-8 h-8 mx-auto mb-3 text-gray-500" />
            <p className="text-sm text-gray-400 mb-4">Upload your base resume (PDF) for ATS parsing</p>
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

        {/* LaTeX Source */}
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
            <FileCode className="w-5 h-5" /> LaTeX Source (Overleaf)
          </h2>
          <p className="text-sm text-gray-400 mb-3">
            Paste your Overleaf LaTeX source here. When tailoring, only Projects and Technical Skills sections will be modified — all formatting stays identical.
          </p>
          <textarea
            className="w-full h-64 bg-gray-950 border border-gray-700 rounded-lg p-3 text-sm text-gray-200 font-mono"
            placeholder="\documentclass[letterpaper,11pt]{article}..."
            value={latexSource}
            onChange={(e) => setLatexSource(e.target.value)}
          />

          {latexError && (
            <div className="mt-3 flex items-center gap-2 text-red-400 text-sm">
              <AlertCircle className="w-4 h-4" /> {latexError}
            </div>
          )}

          {latexSaved && (
            <div className="mt-3 flex items-center gap-2 text-green-400 text-sm">
              <CheckCircle className="w-4 h-4" /> LaTeX source saved
            </div>
          )}

          <button
            className="mt-3 bg-green-700 hover:bg-green-600 text-white px-4 py-2 rounded-lg text-sm font-medium disabled:opacity-50"
            onClick={handleSaveLatex}
            disabled={!latexSource.trim() || savingLatex}
          >
            {savingLatex ? 'Saving...' : 'Save LaTeX Source'}
          </button>
        </div>
      </div>

      <div className="mt-8 bg-gray-900 border border-gray-800 rounded-xl p-6 max-w-2xl">
        <h2 className="text-lg font-semibold mb-2">How it works</h2>
        <ul className="text-sm text-gray-400 space-y-2 list-disc list-inside">
          <li>Upload your PDF resume for basic ATS parsing (required).</li>
          <li>Paste your LaTeX source (from Overleaf) for <strong>perfect formatting</strong> — the tailored PDF will look identical to your original.</li>
          <li>If LaTeX source is saved, the system compiles it with <code>pdflatex</code> after tailoring.</li>
          <li>If <code>pdflatex</code> is not installed, it falls back to the standard PDF generator.</li>
        </ul>
      </div>
    </div>
  );
}
