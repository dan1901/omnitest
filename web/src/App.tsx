import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Layout from './components/Layout.tsx';
import TestListPage from './pages/TestListPage.tsx';
import TestRunPage from './pages/TestRunPage.tsx';
import AgentsPage from './pages/AgentsPage.tsx';

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<TestListPage />} />
          <Route path="/runs/:id" element={<TestRunPage />} />
          <Route path="/agents" element={<AgentsPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
