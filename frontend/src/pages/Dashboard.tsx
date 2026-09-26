import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Book, Loader2, AlertCircle, KeyRound, Plus, Search, Bot, Sparkles, Folder, FolderPlus, Trash2, CheckCircle, XCircle } from 'lucide-react';
import { apiFetch } from '../api';
import type { Module } from '../types';
import { useAuthStore } from '../store/authStore';

// Тип для папок
interface FolderType {
    id: number;
    name: string;
    module_count: number;
    created_at: string;
}

// Тип для сповіщення
interface ToastProps {
    text: string;
    type: 'success' | 'error';
}

export default function Dashboard() {
    const navigate = useNavigate();
    const user = useAuthStore((state) => state.user);

    const [activeTab, setActiveTab] = useState<'modules' | 'folders'>('modules');

    const [modules, setModules] = useState<Module[]>([]);
    const [folders, setFolders] = useState<FolderType[]>([]);
    const [userId, setUserId] = useState<number | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const [inviteCode, setInviteCode] = useState('');
    const [enrollMsg, setEnrollMsg] = useState<{ text: string; type: 'success' | 'error' } | null>(null);

    const [newFolderName, setNewFolderName] = useState('');
    const [isCreatingFolder, setIsCreatingFolder] = useState(false);

    const [isFolderModalOpen, setIsFolderModalOpen] = useState(false);
    const [moduleToAdd, setModuleToAdd] = useState<number | null>(null);

    // Універсальний стейт для красивих сповіщень
    const [toast, setToast] = useState<ToastProps | null>(null);
    const [searchQuery, setSearchQuery] = useState('');

    // Функція для показу сповіщення
    const showToast = (text: string, type: 'success' | 'error' = 'success') => {
        setToast({ text, type });
        setTimeout(() => setToast(null), 3000);
    };

    useEffect(() => {
        if (user?.role === 'admin') {
            navigate('/admin', { replace: true });
        }
    }, [user, navigate]);

    const fetchDashboardData = async () => {
        try {
            setError(null);
            const [modulesData, userData, foldersData] = await Promise.all([
                apiFetch('/modules'),
                apiFetch('/me'),
                apiFetch('/folders').catch(() => []),
            ]);
            setModules(Array.isArray(modulesData) ? modulesData : []);
            setUserId(userData.id);
            setFolders(Array.isArray(foldersData) ? foldersData : []);
        } catch (err: any) {
            setError(err.message || 'Не вдалося завантажити дані.');
        } finally {
            setIsLoading(false);
        }
    };

    useEffect(() => {
        if (user?.role !== 'admin') {
            fetchDashboardData();
        }
    }, [user]);

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

    const handleCreateFolder = async () => {
        if (!newFolderName.trim()) return;

        setIsCreatingFolder(true);
        try {
            await apiFetch('/folders', {
                method: 'POST',
                body: JSON.stringify({ name: newFolderName }),
            });
            setNewFolderName('');
            const newFolders = await apiFetch('/folders');
            setFolders(Array.isArray(newFolders) ? newFolders : []);
            showToast("Папку успішно створено!");
        } catch (err: any) {
            showToast(err.message || 'Помилка створення папки', 'error');
        } finally {
            setIsCreatingFolder(false);
        }
    };

    const handleDeleteFolder = async (e: React.MouseEvent, folderId: number) => {
        e.stopPropagation();
        if (!window.confirm("Видалити папку? Модулі всередині залишаться цілими.")) return;

        try {
            await apiFetch(`/folders/${folderId}`, { method: 'DELETE' });
            setFolders(folders.filter(f => f.id !== folderId));
            showToast("Папку успішно видалено!");
        } catch (err: any) {
            showToast("Помилка видалення: " + err.message, 'error');
        }
    };

    const handleAddModuleToFolder = async (folderId: number) => {
        if (!moduleToAdd) return;

        try {
            await apiFetch(`/folders/${folderId}/modules`, {
                method: 'POST',
                body: JSON.stringify({ module_id: moduleToAdd }),
            });

            const newFolders = await apiFetch('/folders');
            setFolders(Array.isArray(newFolders) ? newFolders : []);

            setIsFolderModalOpen(false);
            setModuleToAdd(null);
            showToast("Модуль успішно додано в папку!");
        } catch (err: any) {
            showToast("Помилка: " + err.message, 'error');
        }
    };

    if (user?.role === 'admin') return null;
    if (isLoading) return <div className="flex justify-center mt-20"><Loader2 className="w-8 h-8 animate-spin text-primary" /></div>;

    const filteredModules = modules.filter(mod =>
        mod.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
        (mod.description && mod.description.toLowerCase().includes(searchQuery.toLowerCase()))
    );

    const filteredFolders = folders.filter(f =>
        f.name.toLowerCase().includes(searchQuery.toLowerCase())
    );

    return (
        <div className="space-y-8 relative">
            <div className="flex flex-col sm:flex-row justify-between items-start sm:items-end gap-4">
                <div>
                    <h1 className="text-3xl font-bold text-textMain tracking-tight">Моє навчання</h1>
                    <p className="text-textMuted mt-2">Організуйте свої матеріали по папках або створюйте нові модулі</p>
                </div>
                <div className="flex gap-3">
                    <button onClick={() => navigate('/practice/chat')} className="inline-flex shrink-0 items-center gap-2 bg-purple-600 hover:bg-purple-700 text-white px-5 py-2.5 rounded-xl font-medium transition-colors shadow-lg shadow-purple-500/20">
                        <Bot className="w-5 h-5" /> ШІ Тренажер
                    </button>
                    <button onClick={() => navigate('/module/new')} className="inline-flex shrink-0 items-center gap-2 bg-primary hover:bg-primaryHover text-white px-5 py-2.5 rounded-xl font-medium transition-colors">
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

            <div className="flex gap-4 border-b border-surfaceBorder">
                <button onClick={() => setActiveTab('modules')} className={`pb-3 px-2 font-medium text-lg border-b-2 transition-colors ${activeTab === 'modules' ? 'border-primary text-primary' : 'border-transparent text-textMuted hover:text-textMain'}`}>
                    Всі модулі ({modules.length})
                </button>
                <button onClick={() => setActiveTab('folders')} className={`pb-3 px-2 font-medium text-lg border-b-2 transition-colors ${activeTab === 'folders' ? 'border-primary text-primary' : 'border-transparent text-textMuted hover:text-textMain'}`}>
                    Мої папки ({folders.length})
                </button>
            </div>

            <div className="relative max-w-md">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-textMuted w-5 h-5" />
                <input
                    type="text"
                    placeholder={activeTab === 'modules' ? "Пошук модулів..." : "Пошук папок..."}
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="w-full bg-surface border border-surfaceBorder text-textMain rounded-xl py-3 pl-10 pr-4 outline-none focus:border-primary transition-colors"
                />
            </div>

            {/* ВКЛАДКА МОДУЛІВ */}
            {activeTab === 'modules' && (
                <div className="space-y-6">
                    <div className="bg-surface border border-surfaceBorder rounded-2xl p-6 flex flex-col md:flex-row gap-4 items-center justify-between">
                        <div className="flex items-center gap-3">
                            <div className="bg-primary/10 p-3 rounded-xl"><KeyRound className="w-6 h-6 text-primary" /></div>
                            <div>
                                <h3 className="text-lg font-bold text-textMain">Маєте код доступу?</h3>
                                <p className="text-textMuted text-sm">Приєднайтеся до курсу викладача або друга</p>
                            </div>
                        </div>
                        <form onSubmit={handleEnroll} className="flex w-full md:w-auto gap-2">
                            <input type="text" placeholder="Напр. CRS-12345" value={inviteCode} onChange={(e) => setInviteCode(e.target.value)} className="w-full md:w-64 bg-mainBg border border-surfaceBorder text-textMain rounded-xl py-2 px-4 outline-none focus:border-primary" />
                            <button type="submit" className="bg-primary hover:bg-primaryHover text-white px-4 py-2 rounded-xl font-medium">Додати</button>
                        </form>
                    </div>
                    {enrollMsg && (
                        <div className={`p-4 rounded-xl border ${enrollMsg.type === 'success' ? 'bg-green-500/10 text-green-400 border-green-500/20' : 'bg-red-500/10 text-red-400 border-red-500/20'}`}>
                            {enrollMsg.text}
                        </div>
                    )}

                    {modules.length === 0 ? (
                        <div className="bg-surface border border-surfaceBorder rounded-2xl p-12 text-center">
                            <Book className="w-12 h-12 text-textMuted mx-auto mb-4 opacity-50" />
                            <h3 className="text-xl font-medium text-textMain">Модулів поки немає</h3>
                        </div>
                    ) : filteredModules.length === 0 ? (
                        <div className="bg-surface border border-surfaceBorder rounded-2xl p-12 text-center">
                            <Search className="w-12 h-12 text-textMuted mx-auto mb-4 opacity-50" />
                            <h3 className="text-xl font-medium text-textMain">Нічого не знайдено</h3>
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
                                        <button onClick={() => navigate(`/modules/${mod.id}/theory`)} className="text-blue-400 text-sm font-medium hover:text-blue-300 transition-colors">Теорія</button>
                                        <button onClick={() => navigate(`/modules/${mod.id}/flashcards`)} className="text-primary text-sm font-medium hover:text-white transition-colors">Картки</button>
                                        <button onClick={() => navigate(`/modules/${mod.id}/quiz`)} className="text-green-500 text-sm font-medium hover:text-green-400 transition-colors">Тест</button>

                                        <button onClick={() => navigate('/practice/ai-test', { state: { topic: `${mod.description} (Граматика)`, theory: mod.theory } })} className="text-purple-500 text-sm font-medium hover:text-purple-400 transition-colors flex items-center gap-1">
                                            <Sparkles className="w-4 h-4" /> ШІ Граматика
                                        </button>

                                        <button onClick={() => { setModuleToAdd(mod.id); setIsFolderModalOpen(true); }} className="text-orange-400 text-sm font-medium hover:text-orange-300 transition-colors flex items-center gap-1">
                                            <FolderPlus className="w-4 h-4" /> В папку
                                        </button>

                                        {userId === mod.created_by && (
                                            <button onClick={() => navigate(`/module/${mod.id}/edit`)} className="text-yellow-500 text-sm font-medium hover:text-yellow-400 transition-colors ml-auto">Редагувати</button>
                                        )}
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            )}

            {/* ВКЛАДКА ПАПОК */}
            {activeTab === 'folders' && (
                <div className="space-y-6">
                    <div className="bg-surface border border-surfaceBorder rounded-2xl p-6 flex flex-col md:flex-row gap-4 items-center justify-between">
                        <div className="flex items-center gap-3">
                            <div className="bg-primary/10 p-3 rounded-xl"><FolderPlus className="w-6 h-6 text-primary" /></div>
                            <div>
                                <h3 className="text-lg font-bold text-textMain">Створити нову папку</h3>
                                <p className="text-textMuted text-sm">Групуйте модулі за тематикою або рівнем</p>
                            </div>
                        </div>
                        <form onSubmit={(e) => { e.preventDefault(); handleCreateFolder(); }} className="flex w-full md:w-auto gap-2">
                            <input type="text" placeholder="Назва папки" value={newFolderName} onChange={(e) => setNewFolderName(e.target.value)} className="w-full md:w-64 bg-mainBg border border-surfaceBorder text-textMain rounded-xl py-2 px-4 outline-none focus:border-primary" />
                            <button type="submit" disabled={isCreatingFolder || !newFolderName.trim()} className="bg-primary hover:bg-primaryHover text-white px-4 py-2 rounded-xl font-medium disabled:opacity-50">
                                {isCreatingFolder ? <Loader2 className="w-5 h-5 animate-spin" /> : 'Створити'}
                            </button>
                        </form>
                    </div>

                    {folders.length === 0 ? (
                        <div className="bg-surface border border-surfaceBorder rounded-2xl p-12 text-center">
                            <Folder className="w-12 h-12 text-textMuted mx-auto mb-4 opacity-50" />
                            <h3 className="text-xl font-medium text-textMain">Папок поки немає</h3>
                        </div>
                    ) : filteredFolders.length === 0 ? (
                        <div className="bg-surface border border-surfaceBorder rounded-2xl p-12 text-center">
                            <Search className="w-12 h-12 text-textMuted mx-auto mb-4 opacity-50" />
                            <h3 className="text-xl font-medium text-textMain">Нічого не знайдено</h3>
                        </div>
                    ) : (
                        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
                            {filteredFolders.map((folder) => (
                                <div key={folder.id} onClick={() => navigate(`/folders/${folder.id}`)} className="bg-surface border border-surfaceBorder rounded-2xl p-5 hover:border-primary/50 cursor-pointer transition-all flex items-center justify-between group">
                                    <div className="flex items-center gap-3">
                                        <Folder className="w-8 h-8 text-primary/80 group-hover:text-primary transition-colors" />
                                        <div>
                                            <h3 className="font-bold text-textMain">{folder.name}</h3>
                                            <p className="text-xs text-textMuted">{folder.module_count} модулів</p>
                                        </div>
                                    </div>
                                    <button onClick={(e) => handleDeleteFolder(e, folder.id)} className="text-textMuted hover:text-red-500 opacity-0 group-hover:opacity-100 transition-all p-2" title="Видалити папку">
                                        <Trash2 className="w-5 h-5" />
                                    </button>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            )}

            {/* МОДАЛКА */}
            {isFolderModalOpen && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
                    <div className="bg-surface border border-surfaceBorder rounded-3xl p-6 w-full max-w-md shadow-xl">
                        <h2 className="text-xl font-bold text-textMain mb-4">Оберіть папку</h2>
                        {folders.length === 0 ? (
                            <p className="text-textMuted mb-6">У вас ще немає папок.</p>
                        ) : (
                            <div className="space-y-2 mb-6 max-h-60 overflow-y-auto pr-2 custom-scrollbar">
                                {folders.map(folder => (
                                    <button key={folder.id} onClick={() => handleAddModuleToFolder(folder.id)} className="w-full text-left p-3 rounded-xl border border-surfaceBorder hover:border-primary hover:bg-primary/10 transition-colors flex justify-between items-center">
                                        <span className="font-medium text-textMain">{folder.name}</span>
                                        <span className="text-xs text-textMuted bg-mainBg px-2 py-1 rounded-lg">{folder.module_count} мод.</span>
                                    </button>
                                ))}
                            </div>
                        )}
                        <div className="flex justify-end">
                            <button onClick={() => { setIsFolderModalOpen(false); setModuleToAdd(null); }} className="px-5 py-2 rounded-xl border border-surfaceBorder text-textMain hover:bg-surfaceBorder transition-colors font-medium">Скасувати</button>
                        </div>
                    </div>
                </div>
            )}

            {/* СПОВІЩЕННЯ (Універсальне: зелене/червоне) */}
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