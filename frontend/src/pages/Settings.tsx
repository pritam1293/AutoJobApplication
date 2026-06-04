import { useEffect, useState } from 'react';
import { getSettings, updateSettings } from '../services/api';
import { Save, AlertCircle, CheckCircle } from 'lucide-react';

export default function Settings() {
  const [form, setForm] = useState({
    linkedin_email: '',
    linkedin_password: '',
    google_ai_key: '',
  });
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  useEffect(() => {
    getSettings().then((data) => {
      setForm((prev) => ({
        ...prev,
        linkedin_email: data.linkedin_email || '',
      }));
    });
  }, []);

  const handleSave = async () => {
    setSaving(true);
    setMessage(null);
    try {
      const cleanForm = Object.fromEntries(
        Object.entries(form).filter(([_, v]) => v !== '')
      );
      await updateSettings(cleanForm);
      setMessage({ type: 'success', text: 'Settings saved successfully' });
    } catch {
      setMessage({ type: 'error', text: 'Failed to save settings' });
    }
    setSaving(false);
  };

  const update = (key: string, value: string) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Settings</h1>

      <div className="space-y-6 max-w-xl">
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">LinkedIn</h2>
          <div className="space-y-3">
            <input
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
              placeholder="LinkedIn Email"
              value={form.linkedin_email}
              onChange={(e) => update('linkedin_email', e.target.value)}
            />
            <input
              type="password"
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
              placeholder="LinkedIn Password"
              value={form.linkedin_password}
              onChange={(e) => update('linkedin_password', e.target.value)}
            />
          </div>
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Google AI Studio</h2>
          <p className="text-xs text-gray-500 mb-2">Get your API key from <a href="https://aistudio.google.com/apikey" target="_blank" rel="noopener noreferrer" className="text-blue-400 underline">aistudio.google.com</a></p>
          <input
            className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
            placeholder="Google AI API Key"
            value={form.google_ai_key}
            onChange={(e) => update('google_ai_key', e.target.value)}
          />
        </div>

        {message && (
          <div className={`flex items-center gap-2 text-sm ${
            message.type === 'success' ? 'text-green-400' : 'text-red-400'
          }`}>
            {message.type === 'success' ? <CheckCircle className="w-4 h-4" /> : <AlertCircle className="w-4 h-4" />}
            {message.text}
          </div>
        )}

        <button
          className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium flex items-center gap-2 disabled:opacity-50"
          onClick={handleSave}
          disabled={saving}
        >
          <Save className="w-4 h-4" />
          {saving ? 'Saving...' : 'Save Settings'}
        </button>
      </div>
    </div>
  );
}
