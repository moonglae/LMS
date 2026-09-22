import { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useAuthStore } from './store/authStore';
import { Ban } from 'lucide-react';

import Layout from './components/Layout';
import Auth from './pages/Auth';
import Dashboard from './pages/Dashboard';
import Flashcards from './pages/Flashcards';
import Quiz from './pages/Quiz';
import Mistakes from './pages/Mistakes';
import Profile from './pages/Profile';
import ModuleEditor from './pages/ModuleEditor';
import TheoryView from './pages/TheoryView';
import AIChat from './pages/AIChat';
import AITest from './pages/AITest';
import AdminDashboard from './pages/AdminDashboard';
import ChatNotebook from './pages/ChatNotebook'; // <--- ДОДАНО ІМПОРТ ЗОШИТА

function App() {
  const { isAuthenticated, user, isCheckingAuth, checkAuth, logout } = useAuthStore();

  useEffect(() => {
    checkAuth();
  }, []);

  if (isCheckingAuth) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-mainBg text-primary font-medium text-xl animate-pulse">
        Перевірка доступу...
      </div>
    );
  }

  // ЖОРСТКИЙ БЛОК ДЛЯ ЗАБАНЕНИХ КОРИСТУВАЧІВ ІЗ ПРИЧИНОЮ
  if (isAuthenticated && user?.is_banned) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center bg-mainBg p-4">
        <div className="bg-surface border border-surfaceBorder p-8 md:p-10 rounded-3xl shadow-2xl max-w-md w-full flex flex-col items-center text-center">
          <div className="bg-red-500/10 p-4 rounded-full mb-6">
            <Ban className="w-16 h-16 text-red-500" />
          </div>
          <h1 className="text-3xl font-bold text-textMain mb-2">Акаунт заблоковано</h1>
          <p className="text-textMuted mb-6 text-lg">Ваш доступ до системи обмежено адміністратором.</p>

          {/* БЛОК З ПРИЧИНОЮ */}
          {user?.ban_reason && (
            <div className="w-full bg-red-500/10 border border-red-500/20 rounded-xl p-4 mb-8 text-left">
              <span className="block text-xs font-bold text-red-400 uppercase tracking-wider mb-1">
                Причина блокування:
              </span>
              <span className="text-red-300 font-medium">
                {user.ban_reason}
              </span>
            </div>
          )}

          <button
            onClick={() => {
              logout();
              window.location.href = '/login';
            }}
            className="w-full bg-red-500 hover:bg-red-600 text-white px-8 py-3.5 rounded-xl font-bold transition-colors"
          >
            Вийти з акаунту
          </button>
        </div>
      </div>
    );
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route
          path="/login"
          element={
            !isAuthenticated ? (
              <Auth />
            ) : user?.role === 'admin' ? (
              <Navigate to="/admin" replace />
            ) : (
              <Navigate to="/" replace />
            )
          }
        />

        <Route
          path="/"
          element={isAuthenticated ? <Layout /> : <Navigate to="/login" replace />}
        >
          <Route index element={<Dashboard />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="admin" element={<AdminDashboard />} />
          <Route path="modules/:id/flashcards" element={<Flashcards />} />
          <Route path="modules/:id/quiz" element={<Quiz />} />
          <Route path="modules/:id/theory" element={<TheoryView />} />
          <Route path="module/new" element={<ModuleEditor />} />
          <Route path="module/:id/edit" element={<ModuleEditor />} />
          <Route path="mistakes" element={<Mistakes />} />
          <Route path="quiz/mistakes" element={<Quiz />} />
          <Route path="quiz/:id" element={<Quiz />} />
          <Route path="profile" element={<Profile />} />

          <Route path="practice/chat" element={<AIChat />} />
          <Route path="practice/ai-test" element={<AITest />} />

          {/* НОВИЙ МАРШРУТ ЗОШИТА */}
          <Route path="notebook" element={<ChatNotebook />} />
        </Route>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;