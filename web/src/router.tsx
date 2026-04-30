import { createBrowserRouter, Navigate } from 'react-router-dom';
import MainLayout from './layouts/MainLayout';
import ClientExposuresPage from './pages/client/Exposures';
import ClientNodesPage from './pages/client/Nodes';
import StatusPage from './pages/client/Status';
import LogsPage from './pages/client/Logs';

export const router = createBrowserRouter([
  {
    path: '/',
    element: <MainLayout />,
    children: [
      { index: true, element: <Navigate to="/ports" replace /> },
      { path: 'ports', element: <ClientExposuresPage /> },
      { path: 'nodes', element: <ClientNodesPage /> },
      { path: 'status', element: <StatusPage /> },
      { path: 'logs', element: <LogsPage /> },
      { path: '*', element: <Navigate to="/ports" replace /> },
    ],
  },
]);
