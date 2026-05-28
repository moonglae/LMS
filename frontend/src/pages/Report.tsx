import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Loader2, ArrowLeft, BarChart3, Printer, BookOpen, Calendar, Award, CheckCircle, AlertTriangle } from 'lucide-react';
import { apiFetch } from '../api';

interface StudentReport {
    student_id: number;
    student_name: string;
    attempts: number;
    average_score: number;
    registration_date: string;
}

interface DetailedStudentReport {
    student_id: number;
    student_name: string;
    quizzes_taken: number;
    average_score: number;
    perfect_scores: number;
    mistakes_count: number;
}

export default function Report() {
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const moduleId = searchParams.get('module_id');

    const [moduleTitle, setModuleTitle] = useState<string>('');
    const [students, setStudents] = useState<StudentReport[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [selectedView, setSelectedView] = useState<string>('all');
    const [detailedReport, setDetailedReport] = useState<DetailedStudentReport | null>(null);
    const [isDetailLoading, setIsDetailLoading] = useState(false);

    useEffect(() => {
        const fetchAll = async () => {
            try {
                // Завантажуємо звіт та список модулів (щоб дістати назву)
                const [reportData, modulesData] = await Promise.all([
                    apiFetch(`/analytics/module-report?module_id=${moduleId}`),
                    apiFetch('/modules')
                ]);

                setStudents(Array.isArray(reportData) ? reportData : []);

                // Знаходимо назву поточного модуля
                const mods = Array.isArray(modulesData) ? modulesData : [];
                const currentMod = mods.find(m => m.id === Number(moduleId));
                if (currentMod) setModuleTitle(currentMod.title);

            } catch (err) {
                console.error("Помилка завантаження звіту", err);
            } finally {
                setIsLoading(false);
            }
        };
        fetchAll();
    }, [moduleId]);

    useEffect(() => {
        if (selectedView === 'all') return;
        const fetchDetailed = async () => {
            setIsDetailLoading(true);
            try {
                const data = await apiFetch(`/analytics/student-report?student_id=${selectedView}&module_id=${moduleId}`);
                setDetailedReport(data);
            } finally {
                setIsDetailLoading(false);
            }
        };
        fetchDetailed();
    }, [selectedView, moduleId]);

    if (isLoading) return <div className="flex justify-center mt-20"><Loader2 className="w-8 h-8 animate-spin text-primary" /></div>;

    // Колір для оцінки
    const getScoreColor = (score: number) => {
        if (score >= 85) return 'text-green-500 bg-green-500/10 border-green-500/20';
        if (score >= 60) return 'text-yellow-500 bg-yellow-500/10 border-yellow-500/20';
        return 'text-red-400 bg-red-500/10 border-red-500/20';
    };

    const currentDate = new Date().toLocaleDateString('uk-UA', { day: '2-digit', month: 'long', year: 'numeric' });

    return (
        <div className="max-w-5xl mx-auto p-4 mt-8 print:mt-0 print:p-0">
            {/* ПАНЕЛЬ КЕРУВАННЯ (Не друкується) */}
            <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-8 print:hidden">
                <button
                    onClick={() => navigate('/teacher')}
                    className="flex items-center gap-2 px-4 py-2 bg-surface border border-surfaceBorder rounded-xl hover:bg-mainBg transition-colors text-sm font-medium"
                >
                    <ArrowLeft size={16} /> До панелі викладача
                </button>

                <div className="flex items-center gap-3 w-full sm:w-auto">
                    <select
                        value={selectedView}
                        onChange={e => setSelectedView(e.target.value)}
                        className="bg-surface border border-surfaceBorder px-4 py-2 rounded-xl text-sm outline-none focus:border-primary flex-1 sm:w-48"
                    >
                        <option value="all">Загальний звіт (Всі)</option>
                        {students.map(s => <option key={s.student_id} value={s.student_id}>{s.student_name}</option>)}
                    </select>
                    <button
                        onClick={() => window.print()}
                        className="flex items-center gap-2 bg-primary hover:bg-primaryHover text-white px-5 py-2 rounded-xl font-medium transition-colors"
                    >
                        <Printer size={16} /> Друк
                    </button>
                </div>
            </div>

            {/* ОФІЦІЙНИЙ ШАПКА ЗВІТУ (Адаптовано для друку) */}
            <div className="bg-surface border border-surfaceBorder p-8 rounded-3xl print:border-none print:shadow-none print:p-0 mb-6">
                <div className="border-b border-surfaceBorder pb-6 mb-6 flex items-start justify-between">
                    <div>
                        <div className="flex items-center gap-3 mb-2">
                            <BookOpen className="text-primary w-8 h-8 print:text-black" />
                            <h1 className="text-3xl font-bold text-textMain print:text-black">Аналітичний звіт</h1>
                        </div>
                        <h2 className="text-xl font-medium text-textMuted print:text-gray-700">
                            Курс: <span className="text-textMain font-bold print:text-black">{moduleTitle || 'Завантаження...'}</span>
                        </h2>
                    </div>
                    <div className="text-right text-sm text-textMuted flex flex-col items-end gap-1 print:text-gray-600">
                        <div className="flex items-center gap-1"><Calendar size={14} /> {currentDate}</div>
                        <p>Тип звіту: {selectedView === 'all' ? 'Груповий' : 'Індивідуальний'}</p>
                    </div>
                </div>

                {/* КОНТЕНТ ЗВІТУ */}
                {selectedView === 'all' ? (
                    <div>
                        <table className="w-full text-left border-collapse">
                            <thead>
                                <tr className="border-b border-surfaceBorder text-textMuted text-sm uppercase tracking-wider">
                                    <th className="p-4 font-semibold print:text-black">ПІБ Студента</th>
                                    <th className="p-4 font-semibold text-center print:text-black">Дата реєстрації</th>
                                    <th className="p-4 font-semibold text-center print:text-black">К-сть спроб</th>
                                    <th className="p-4 font-semibold text-right print:text-black">Середній бал</th>
                                </tr>
                            </thead>
                            <tbody>
                                {students.length === 0 ? (
                                    <tr>
                                        <td colSpan={4} className="text-center p-8 text-textMuted">Немає даних для відображення</td>
                                    </tr>
                                ) : students.map((s, index) => (
                                    <tr key={s.student_id} className={`border-b border-surfaceBorder/50 hover:bg-mainBg/50 transition-colors ${index % 2 === 0 ? 'print:bg-gray-50' : 'print:bg-white'}`}>
                                        <td className="p-4 font-medium text-textMain print:text-black">{s.student_name}</td>
                                        <td className="p-4 text-center text-textMuted print:text-gray-700">{s.registration_date}</td>
                                        <td className="p-4 text-center font-medium print:text-black">{s.attempts}</td>
                                        <td className="p-4 text-right">
                                            <span className={`px-3 py-1 rounded-lg text-sm font-bold border print:border-gray-300 print:text-black print:bg-transparent ${getScoreColor(s.average_score)}`}>
                                                {s.average_score}%
                                            </span>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                ) : isDetailLoading ? (
                    <div className="flex justify-center py-12"><Loader2 className="w-8 h-8 animate-spin text-primary" /></div>
                ) : detailedReport ? (
                    <div className="space-y-8">
                        <div className="bg-mainBg border border-surfaceBorder p-6 rounded-2xl print:border-gray-300 print:bg-transparent">
                            <p className="text-sm text-textMuted uppercase tracking-wide mb-1">Студент</p>
                            <p className="text-2xl font-bold text-textMain print:text-black">{detailedReport.student_name}</p>
                        </div>

                        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
                            <StatBox label="Пройдено тестів" value={detailedReport.quizzes_taken} icon={<BarChart3 size={20} />} />
                            <StatBox
                                label="Середній бал"
                                value={`${detailedReport.average_score}%`}
                                icon={<Award size={20} />}
                                colorClass={detailedReport.average_score >= 60 ? 'text-green-500' : 'text-yellow-500'}
                            />
                            <StatBox label="Ідеальні спроби" value={detailedReport.perfect_scores} icon={<CheckCircle size={20} />} />
                            <StatBox label="Кількість помилок" value={detailedReport.mistakes_count} icon={<AlertTriangle size={20} />} colorClass="text-red-400" />
                        </div>
                    </div>
                ) : null}
            </div>
        </div>
    );
}

// Допоміжний компонент для детального звіту
function StatBox({ label, value, icon, colorClass = "text-primary" }: { label: string, value: any, icon: React.ReactNode, colorClass?: string }) {
    return (
        <div className="bg-mainBg border border-surfaceBorder p-5 rounded-2xl print:border-gray-300 print:bg-transparent flex flex-col items-center sm:items-start text-center sm:text-left">
            <div className={`${colorClass} mb-2`}>{icon}</div>
            <p className="text-textMuted text-xs uppercase tracking-wider mb-1 print:text-gray-600">{label}</p>
            <p className="text-2xl font-bold text-textMain print:text-black">{value}</p>
        </div>
    );
}