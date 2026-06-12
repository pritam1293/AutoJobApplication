import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import Search from './pages/Search';
import Jobs from './pages/Jobs';
import Applications from './pages/Applications';
import Resume from './pages/Resume';
import TrialTailor from './pages/TrialTailor';
import Settings from './pages/Settings';
import Layout from './components/Layout';

export default function App() {
  return (
    <BrowserRouter>
      <Layout>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/search" element={<Search />} />
          <Route path="/jobs" element={<Jobs />} />
          <Route path="/applications" element={<Applications />} />
          <Route path="/resume" element={<Resume />} />
          <Route path="/trial-tailor" element={<TrialTailor />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </Layout>
    </BrowserRouter>
  );
}
