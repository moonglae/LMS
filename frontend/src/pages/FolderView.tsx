import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Loader2, AlertCircle, ArrowLeft, Book, Sparkles, FolderOpen, LogOut, CheckCircle, XCircle } from 'lucide-react';
import { apiFetch } from '../api';
import type { Module } from '../types';
import { useAuthStore } from '../store/authStore';

interface ToastProps {
    text: string;
    type: 'success' | 'error';
}

export default function FolderView() {
    const { id } = useParams();
    const navigate = useNavigate();
    const user = useAuthStore((state) => state.user);

    const [modules, setModules] = useState<Module[]>([]);
    const [folderName, setFolderName] = useState<string>('Завантаження...');
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const [toast, setToast] = useState<ToastProps | null>(null);

    const showToast = (text: string, type: 'success' | 'error' = 'success') => {
        setToast({ text, type });
        setTimeout(() => setToast(null), 3000);
    };

    useEffect(() => {
        const fetchFolderData = async () => {
            try {
                const modulesData = await apiFetch(`/folders/${id}`);
                setModules(Array.isArray(modulesData) ? modulesData : []);

                const foldersData = await apiFetch('/folders');
                const currentFolder = foldersData.find((f: any) => f.id.toString() === id);
                if (currentFolder) {
                    setFolderName(currentFolder.name);
                } else {
                    setFolderName('Папка');
                }
            } catch (err: any) {
                setError(err.message || 'Не вдалося завантажити вміст папки.');
            } finally {
                setIsLoading(false);
            }
        };

        fetchFolderData();
    }, [id]);

    const handleRemoveFromFolder = async (moduleId: number) => {
        if (!window.confirm("Вилучити модуль з папки? Він все ще буде доступний у вкладці 'Всі модулі'.")) return;

        try {
            await apiFetch(`/folders/${id}/modules/${moduleId}`, {
                method: 'DELETE',
            });
            setModules(prev => prev.filter(m => m.id !== moduleId));
            showToast("Модуль успішно вилучено з папки!");
        } catch (err: any) {
            showToast("Помилка вилучення: " + err.message, 'error');
        }
    };

    if (isLoading) return <div className="flex justify-center mt-20"><Loader2 className="w-8 h-8 animate-spin text-primary" /></div>;

    if (error) {
        return (
            <div className="max-w-md mx-auto mt-20 bg-red-500/10 border border-red-500/20 rounded-3xl p-8 text-center">
                <AlertCircle className="w-16 h-16 text-red-500 mx-auto mb-4" />
                <h2 className="text-2xl font-bold mb-2 text-textMain">Помилка</h2>
                <p className="text-textMuted mb-6">{error}</p>
                <button onClick={() => navigate('/')} className="w-full bg-primary py-3 rounded-xl text-white font-semibold hover:bg-primaryHover transition-colors">
                    На головну
                </button>
            </div>
        );
    }

    return (
        <div className="space-y-8 relative">
            <div className="flex flex-col gap-4">
                <button onClick={() => navigate('/')} className="inline-flex items-center gap-2 text-textMuted hover:text-white transition-colors w-fit">
                    <ArrowLeft className="w-5 h-5" /> Назад до папок
                </button>

                <div className="flex items-center gap-3">
                    <div className="bg-primary/10 p-3 rounded-xl">
                        <FolderOpen className="w-8 h-8 text-primary" />
                    </div>
                    <div>
                        <h1 className="text-3xl font-bold text-textMain tracking-tight">{folderName}</h1>
                        <p className="text-textMuted mt-1">
                            {modules.length} {modules.length === 1 ? 'модуль' : modules.length > 1 && modules.length < 5 ? 'модулі' : 'модулів'}
                        </p>
                    </div>
                </div>
            </div>

            {modules.length === 0 ? (
                <div className="bg-surface border border-surfaceBorder rounded-2xl p-12 text-center">
                    <Book className="w-12 h-12 text-textMuted mx-auto mb-4 opacity-50" />
                    <h3 className="text-xl font-medium text-textMain">Папка порожня</h3>
                    <p className="text-textMuted mt-2">Поверніться на головну та додайте сюди модулі.</p>
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                    {modules.map((mod) => (
                        <div key={mod.id} className="group bg-surface border border-surfaceBorder rounded-2xl p-6 hover:border-primary/50 transition-all duration-300 flex flex-col h-full relative">
                            {user?.id === mod.created_by && mod.invite_code && (
                                <div className="absolute top-4 right-4 bg-mainBg border border-surfaceBorder text-xs text-textMuted px-2 py-1 rounded-lg font-mono">
                                    Код: {mod.invite_code}
                                </div>
                            )}

                            <h3 className="text-xl font-bold text-textMain mb-2 pr-20">{mod.title}</h3>
                            <p className="text-textMuted text-sm flex-grow mb-6">{mod.description}</p>

                            <div className="flex items-center flex-wrap gap-4 mt-auto pt-4 border-t border-surfaceBorder/50">
                                <button onClick={() => navigate(`/modules/${mod.id}/theory`)} className="text-blue-400 text-sm font-medium hover:text-blue-300 transition-colors">Теорія</button>
                                <button onClick={() => navigate(`/modules/${mod.id}/flashcards`)} className="text-primary text-sm font-medium hover:text-white transition-colors">Картки</button>
                                <button onClick={() => navigate(`/modules/${mod.id}/quiz`)} className="text-green-500 text-sm font-medium hover:text-green-400 transition-colors">Тест</button>

                                <button onClick={() => navigate('/practice/ai-test', { state: { topic: `${mod.description} (Граматика)`, theory: mod.theory } })} className="text-purple-500 text-sm font-medium hover:text-purple-400 transition-colors flex items-center gap-1">
                                    <Sparkles className="w-4 h-4" /> ШІ Граматика
                                </button>

                                <button
                                    onClick={() => handleRemoveFromFolder(mod.id)}
                                    className="text-red-400 text-sm font-medium hover:text-red-300 transition-colors flex items-center gap-1 ml-auto"
                                    title="Вилучити модуль з цієї папки"
                                >
                                    <LogOut className="w-4 h-4" /> З папки
                                </button>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {/* СПОВІЩЕННЯ */}
            {toast && (
                <div className="fixed bottom-8 right-8 z-50 animate-in fade-in slide-in-from-bottom-4 duration-300">
                    <div className={`px-6 py-4 rounded-2xl shadow-lg backdrop-blur-md flex items-center gap-3 border ${toast.type === 'success' ? 'bg-green-500/10 border-green-500/20 text-green-400' : 'bg-red-500/10 border-red-500/20 text-red-400'}`}>
                        {toast.type === 'success' ? <CheckCircle className="w-6 h-6" /> : <XCircle className="w-6 h-6" />}
                        <span className="font-medium text-lg">{toast.text}</span>
                    </div>
                </div>
            )}
        </div>
    );
}