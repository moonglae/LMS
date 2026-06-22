import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useAuthStore } from './store/authStore';

import Layout from './components/Layout';
import Auth from './pages/Auth';
import Dashboard from './pages/Dashboard';
import Flashcards from './pages/Flashcards';
import Quiz from './pages/Quiz';
import Mistakes from './pages/Mistakes';
import Profile from './pages/Profile';
import ModuleEditor from './pages/ModuleEditor';
import TheoryView from './pages/TheoryView';
// Імпортуємо наш новий компонент чату
import AIChat from './pages/AIChat';

function App() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  return (
    <BrowserRouter>
      <Routes>
        {/* Відкритий маршрут для входу */}
        <Route
          path="/login"
          element={!isAuthenticated ? <Auth /> : <Navigate to="/" />}
        />

        {/* ЗАХИЩЕНІ МАРШРУТИ */}
        <Route
          path="/"
          element={isAuthenticated ? <Layout /> : <Navigate to="/login" />}
        >
          {/* Головна сторінка */}
          <Route index element={<Dashboard />} />
          <Route path="dashboard" element={<Dashboard />} />

          {/* Маршрути модулів */}
          <Route path="modules/:id/flashcards" element={<Flashcards />} />
          <Route path="modules/:id/quiz" element={<Quiz />} />

          {/* Маршрути редактора */}
          <Route path="module/new" element={<ModuleEditor />} />
          <Route path="module/:id/edit" element={<ModuleEditor />} />

          {/* Аналітика та тести - видалено (статистика у профілі) */}

          <Route path="mistakes" element={<Mistakes />} />
          <Route path="quiz/mistakes" element={<Quiz />} />
          <Route path="quiz/:id" element={<Quiz />} />

          <Route path="profile" element={<Profile />} />
          <Route path="/modules/:id/theory" element={<TheoryView />} />

          {/* Наш новий маршрут для ШІ Тренажера */}
          <Route path="practice/chat" element={<AIChat />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;