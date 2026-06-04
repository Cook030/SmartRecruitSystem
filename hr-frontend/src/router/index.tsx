import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Layout from '../components/Layout';
import Login from '../pages/Login';
import Register from '../pages/Register';
import Dashboard from '../pages/Dashboard';
import JobList from '../pages/JobList';
import JobForm from '../pages/JobForm';
import Applications from '../pages/Applications';
import CandidateDetail from '../pages/CandidateDetail';

function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem('hr_token');
  if (!token) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export default function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="/" element={<RequireAuth><Layout /></RequireAuth>}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="jobs" element={<JobList />} />
          <Route path="jobs/new" element={<JobForm />} />
          <Route path="jobs/edit/:id" element={<JobForm />} />
          <Route path="jobs/:jobId/applications" element={<Applications />} />
          <Route path="applications/:id/detail" element={<CandidateDetail />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
