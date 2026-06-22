import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Book, Loader2, AlertCircle, KeyRound, Plus, Search, Bot } from 'lucide-react';
import { apiFetch } from '../api';
import type { Module } from '../types';

export default function Dashboard() {
    const navigate = useNavigate();

    const [modules, setModules] = useState<Module[]>([]);
    const [userId, setUserId] = useState<number | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [inviteCode, setInviteCode] = useState('');
    const [enrollMsg, setEnrollMsg] = useState<{ text: string; type: 'success' | 'error' } | null>(null);

    const [searchQuery, setSearchQuery] = useState('');

    const fetchDashboardData = async () => {
        try {
            setError(null);
            const [modulesData, userData] = await Promise.all([
                apiFetch('/modules'),
                apiFetch('/me'),
            ]);
            setModules(Array.isArray(modulesData) ? modulesData : []);
            setUserId(userData.id);
        } catch (err: any) {
            setError(err.message || 'Не вдалося завантажити дані.');
        } finally {
            setIsLoading(false);
        }
    };

    useEffect(() => {
        fetchDashboardData();
    }, []);

    const handleEnroll = async (e: React.FormEvent) => {
        e.preventDefault();
        setEnrollMsg(null);
        try {
            const data = await apiFetch('/modules/enroll', {
                method: 'POST',
                body: JSON.stringify({ invite_code: inviteCode }),
            });
            setEnrollMsg({ text: data.message || "Успішне приєднання", type: 'success' });
            setInviteCode('');
            const newModules = await apiFetch('/modules');
            setModules(Array.isArray(newModules) ? newModules : []);
        } catch (err: any) {
            setEnrollMsg({ text: err.message || 'Помилка приєднання.', type: 'error' });
        }
    };

    if (isLoading) return <div className="flex justify-center mt-20"><Loader2 className="w-8 h-8 animate-spin text-primary" /></div>;

    const filteredModules = modules.filter(mod =>
        mod.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
        (mod.description && mod.description.toLowerCase().includes(searchQuery.toLowerCase()))
    );

    return (
        <div className="space-y-8">
            <div className="flex flex-col sm:flex-row justify-between items-start sm:items-end gap-4">
                <div>
                    <h1 className="text-3xl font-bold text-textMain tracking-tight">Навчальні модулі</h1>
                    <p className="text-textMuted mt-2">
                        Створюйте власні модулі або приєднуйтесь до інших
                    </p>
                </div>
                <div className="flex gap-3">
                    {/* Нова кнопка для чату з ШІ */}
                    <button
                        onClick={() => navigate('/practice/chat')}
                        className="inline-flex shrink-0 items-center gap-2 bg-purple-600 hover:bg-purple-700 text-white px-5 py-2.5 rounded-xl font-medium transition-colors shadow-lg shadow-purple-500/20"
                    >
                        <Bot className="w-5 h-5" /> ШІ Тренажер
                    </button>

                    <button
                        onClick={() => navigate('/module/new')}
                        className="inline-flex shrink-0 items-center gap-2 bg-primary hover:bg-primaryHover text-white px-5 py-2.5 rounded-xl font-medium transition-colors"
                    >
                        <Plus className="w-5 h-5" /> Новий модуль
                    </button>
                </div>
            </div>

            {error && (
                <div className="bg-red-500/10 border border-red-500/20 rounded-2xl p-6 flex items-start gap-4">
                    <AlertCircle className="w-6 h-6 text-red-400 shrink-0" />
                    <p className="text-red-300/80">{error}</p>
                </div>
            )}

            <div className="space-y-4">
                <div className="bg-surface border border-surfaceBorder rounded-2xl p-6 flex flex-col md:flex-row gap-4 items-center justify-between">
                    <div className="flex items-center gap-3">
                        <div className="bg-primary/10 p-3 rounded-xl"><KeyRound className="w-6 h-6 text-primary" /></div>
                        <div>
                            <h3 className="text-lg font-bold text-textMain">Маєте код доступу?</h3>
                            <p className="text-textMuted text-sm">Приєднайтеся до курсу викладача або друга</p>
                        </div>
                    </div>
                    <form onSubmit={handleEnroll} className="flex w-full md:w-auto gap-2">
                        <input
                            type="text"
                            placeholder="Напр. CRS-12345"
                            value={inviteCode}
                            onChange={(e) => setInviteCode(e.target.value)}
                            className="w-full md:w-64 bg-mainBg border border-surfaceBorder text-textMain rounded-xl py-2 px-4 outline-none focus:border-primary"
                        />
                        <button type="submit" className="bg-primary hover:bg-primaryHover text-white px-4 py-2 rounded-xl font-medium">Додати</button>
                    </form>
                </div>
                {enrollMsg && (
                    <div className={`p-4 rounded-xl border ${enrollMsg.type === 'success' ? 'bg-green-500/10 text-green-400 border-green-500/20' : 'bg-red-500/10 text-red-400 border-red-500/20'}`}>
                        {enrollMsg.text}
                    </div>
                )}
            </div>

            {modules.length > 0 && (
                <div className="relative max-w-md">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-textMuted w-5 h-5" />
                    <input
                        type="text"
                        placeholder="Пошук модулів..."
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="w-full bg-surface border border-surfaceBorder text-textMain rounded-xl py-3 pl-10 pr-4 outline-none focus:border-primary transition-colors"
                    />
                </div>
            )}

            {modules.length === 0 ? (
                <div className="bg-surface border border-surfaceBorder rounded-2xl p-12 text-center">
                    <Book className="w-12 h-12 text-textMuted mx-auto mb-4 opacity-50" />
                    <h3 className="text-xl font-medium text-textMain">Модулів поки немає</h3>
                    <p className="text-textMuted mt-2">Створіть свій перший модуль, щоб почати навчання!</p>
                </div>
            ) : filteredModules.length === 0 ? (
                <div className="bg-surface border border-surfaceBorder rounded-2xl p-12 text-center">
                    <Search className="w-12 h-12 text-textMuted mx-auto mb-4 opacity-50" />
                    <h3 className="text-xl font-medium text-textMain">Нічого не знайдено</h3>
                    <p className="text-textMuted mt-2">За запитом "{searchQuery}" не знайдено жодного модуля.</p>
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                    {filteredModules.map((mod) => (
                        <div key={mod.id} className="group bg-surface border border-surfaceBorder rounded-2xl p-6 hover:border-primary/50 transition-all duration-300 flex flex-col h-full relative">
                            {userId === mod.created_by && mod.invite_code && (
                                <div className="absolute top-4 right-4 bg-mainBg border border-surfaceBorder text-xs text-textMuted px-2 py-1 rounded-lg font-mono">
                                    Код: {mod.invite_code}
                                </div>
                            )}

                            <h3 className="text-xl font-bold text-textMain mb-2 pr-20">{mod.title}</h3>
                            <p className="text-textMuted text-sm flex-grow mb-6">{mod.description}</p>

                            <div className="flex items-center flex-wrap gap-4 mt-auto pt-4 border-t border-surfaceBorder/50">
                                <button onClick={() => navigate(`/modules/${mod.id}/theory`)} className="text-blue-400 text-sm font-medium hover:text-blue-300 transition-colors">
                                    Теорія
                                </button>
                                <button onClick={() => navigate(`/modules/${mod.id}/flashcards`)} className="text-primary text-sm font-medium hover:text-white transition-colors">
                                    Картки
                                </button>
                                <button onClick={() => navigate(`/modules/${mod.id}/quiz`)} className="text-green-500 text-sm font-medium hover:text-white transition-colors">
                                    Тест
                                </button>
                                {userId === mod.created_by && (
                                    <button onClick={() => navigate(`/module/${mod.id}/edit`)} className="text-yellow-500 text-sm font-medium hover:text-yellow-400 transition-colors ml-auto">
                                        Редагувати
                                    </button>
                                )}
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}