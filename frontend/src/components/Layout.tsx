// src/components/Layout.tsx
import { Outlet, Link, useNavigate } from 'react-router-dom';
import {
    LogOut,
    BookOpen,
    User,
    AlertTriangle,
} from 'lucide-react';
import { useAuthStore } from '../store/authStore';

export default function Layout() {
    const logout = useAuthStore((state) => state.logout);
    const navigate = useNavigate();

    const handleLogout = () => {
        logout();
        navigate('/login');
    };

    return (
        <div className="min-h-screen bg-mainBg flex flex-col">
            <header className="bg-surface border-b border-surfaceBorder sticky top-0 z-10">
                <div className="max-w-7xl mx-auto px-4 h-16 flex items-center justify-between">
                    <Link to="/" className="flex items-center gap-2 cursor-pointer">
                        <BookOpen className="w-6 h-6 text-primary" />
                        <span className="text-xl font-bold text-textMain tracking-wide">
                            LMS<span className="text-primary">.system</span>
                        </span>
                    </Link>

                    <div className="flex items-center gap-4">
                        <Link
                            to="/mistakes"
                            className="flex items-center gap-2 text-textMuted hover:text-textMain transition-colors"
                        >
                            <AlertTriangle className="w-5 h-5" />
                            <span className="text-sm font-medium hidden sm:block">
                                Помилки
                            </span>
                        </Link>

                        <Link
                            to="/profile"
                            className="flex items-center gap-2 text-textMuted hover:text-textMain transition-colors"
                        >
                            <User className="w-5 h-5" />
                            <span className="text-sm font-medium hidden sm:block">
                                Профіль
                            </span>
                        </Link>

                        <div className="w-px h-6 bg-surfaceBorder mx-2" />

                        <button
                            onClick={handleLogout}
                            className="flex items-center gap-2 text-red-400 hover:text-red-300 transition-colors"
                        >
                            <LogOut className="w-5 h-5" />
                            <span className="text-sm font-medium hidden sm:block">
                                Вийти
                            </span>
                        </button>
                    </div>
                </div >
            </header >

            <main className="flex-1 max-w-7xl mx-auto w-full p-4 sm:p-6 lg:p-8">
                <Outlet />
            </main>
        </div >
    );
}
