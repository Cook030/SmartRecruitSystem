import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Layout from '../components/Layout';
import Login from '../pages/Login';
import Register from '../pages/Register';
import JobList from '../pages/JobList';
import Profile from '../pages/Profile';
import Resume from '../pages/Resume';
import MyApplications from '../pages/MyApplications';

function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem('candidate_token');
  if (!token) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export default function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="/" element={<Layout />}>
          <Route index element={<JobList />} />
          <Route path="profile" element={<RequireAuth><Profile /></RequireAuth>} />
          <Route path="resume" element={<RequireAuth><Resume /></RequireAuth>} />
          <Route path="applications" element={<RequireAuth><MyApplications /></RequireAuth>} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
