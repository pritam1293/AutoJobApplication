import { NavLink } from 'react-router-dom';
import { Briefcase, Search, FileText, Settings, BarChart3, Upload, FlaskConical } from 'lucide-react';

const navItems = [
  { to: '/', label: 'Dashboard', icon: BarChart3 },
  { to: '/search', label: 'Search Jobs', icon: Search },
  { to: '/jobs', label: 'Jobs', icon: Briefcase },
  { to: '/applications', label: 'Applications', icon: FileText },
  { to: '/resume', label: 'Resume', icon: Upload },
  { to: '/trial-tailor', label: 'Trial Tailor', icon: FlaskConical },
  { to: '/settings', label: 'Settings', icon: Settings },
];

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-screen bg-gray-950 text-gray-100">
      <aside className="w-64 border-r border-gray-800 p-4 flex flex-col">
        <div className="flex items-center gap-2 mb-8 px-2">
          <Briefcase className="w-6 h-6 text-blue-400" />
          <span className="font-bold text-lg">JobHaunt</span>
        </div>
        <nav className="flex-1 space-y-1">
          {navItems.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/'}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${
                  isActive
                    ? 'bg-blue-600 text-white'
                    : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800'
                }`
              }
            >
              <Icon className="w-4 h-4" />
              {label}
            </NavLink>
          ))}
        </nav>
        <div className="text-xs text-gray-600 px-2">
          JobHaunt v0.1.0
        </div>
      </aside>
      <main className="flex-1 overflow-auto p-6">
        {children}
      </main>
    </div>
  );
}
