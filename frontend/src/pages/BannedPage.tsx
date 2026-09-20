import { Ban } from 'lucide-react';
import { useAuthStore } from '../store/authStore';

export default function BannedPage() {
    const logout = useAuthStore((state) => state.logout);
    const user = useAuthStore((state) => state.user);

    return (
        <div className="min-h-screen flex flex-col items-center justify-center bg-mainBg text-center p-4">
            <div className="bg-surface border border-surfaceBorder p-10 rounded-3xl shadow-xl max-w-lg w-full flex flex-col items-center">
                <div className="bg-red-500/10 p-4 rounded-full mb-6">
                    <Ban className="w-16 h-16 text-red-500" />
                </div>
                <h1 className="text-3xl font-bold text-textMain mb-2">Акаунт заблоковано</h1>
                <p className="text-textMuted mb-6">
                    Ваш доступ до системи було обмежено адміністратором.
                </p>

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
                    className="w-full bg-red-500 hover:bg-red-600 text-white px-8 py-3 rounded-xl font-bold transition-colors"
                >
                    Вийти з акаунту
                </button>
            </div>
        </div>
    );
}