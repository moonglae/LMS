import { useState, useEffect } from 'react';
import { Loader2, Users, BookOpen, BarChart3, User, Printer, Award, CheckCircle, AlertTriangle } from 'lucide-react';
import { apiFetch } from '../api';

// Повністю переписані стилі для ідеального друку (скидання темної теми)
const printStyles = `
    @media print {
        /* 1. Налаштування сторінки */
        @page { margin: 15mm; }
        
        /* 2. Ховаємо навігацію (хедер) та меню */
        header, nav, aside, .no-print { 
            display: none !important; 
        }
        
        /* 3. Примусово білий фон для всього */
        body, main, #root, .print-only, .bg-surface, .bg-mainBg { 
            background-color: white !important; 
            background: white !important;
        }

        /* 4. Розтягуємо звіт на весь аркуш */
        .print-only {
            width: 100% !important;
            display: block !important;
            border: none !important;
            box-shadow: none !important;
            padding: 0 !important;
            margin: 0 !important;
        }

        /* 5. Чорний текст для всіх елементів */
        h1, h2, h3, h4, p, span, div, li { 
            color: black !important; 
        }
        
        /* 6. Сірий колір для другорядного тексту */
        .text-textMuted { 
            color: #6b7280 !important; /* Сірий */
        }

        /* 7. Рамки для плашок замість темного фону */
        .print-box {
            border: 1px solid #d1d5db !important;
            box-shadow: none !important;
            background: transparent !important;
        }

        /* 8. Зберігаємо кольорові індикатори для оцінок (щоб не стали чорними) */
        .text-green-500 { color: #16a34a !important; }
        .text-yellow-500 { color: #ca8a04 !important; }
        .text-red-400 { color: #dc2626 !important; }
        .text-primary { color: #2563eb !important; }
    }
`;

interface Module { id: number; title: string; }
interface Student { id: number; first_name: string; last_name: string; email: string; }
interface ReportData { student_id: number; student_name: string; quizzes_taken: number; average_score: number; perfect_scores: number; mistakes_count: number; }

export default function TeacherPanel() {
    const [modules, setModules] = useState<Module[]>([]);
    const [students, setStudents] = useState<Student[]>([]);
    const [selectedModuleId, setSelectedModuleId] = useState<number | null>(null);
    const [selectedStudent, setSelectedStudent] = useState<Student | null>(null);
    const [report, setReport] = useState<ReportData | null>(null);
    const [mistakes, setMistakes] = useState<string[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [isDataLoading, setIsDataLoading] = useState(false);

    useEffect(() => { loadModules(); }, []);

    const loadModules = async () => {
        setIsLoading(true);
        try {
            const mods = await apiFetch('/modules');
            setModules(Array.isArray(mods) ? mods : []);
        } catch (err) { console.error(err); } finally { setIsLoading(false); }
    };

    const handleSelectModule = async (moduleId: number) => {
        setSelectedModuleId(moduleId);
        setStudents([]); setReport(null); setMistakes([]); setSelectedStudent(null);
        setIsDataLoading(true);
        try {
            const data = await apiFetch(`/modules/students?module_id=${moduleId}`);
            setStudents(Array.isArray(data) ? data : []);
        } finally { setIsDataLoading(false); }
    };

    const handleSelectStudent = async (student: Student) => {
        if (!selectedModuleId) return;
        setSelectedStudent(student);
        setIsDataLoading(true);
        try {
            const [reportData, mistakesData] = await Promise.all([
                apiFetch(`/analytics/student-report?module_id=${selectedModuleId}&student_id=${student.id}`),
                apiFetch(`/analytics/student-mistakes?module_id=${selectedModuleId}&student_id=${student.id}`)
            ]);
            setReport(reportData);
            setMistakes(Array.isArray(mistakesData) ? mistakesData : []);
        } finally { setIsDataLoading(false); }
    };

    if (isLoading) return <div className="flex justify-center p-20"><Loader2 className="animate-spin text-primary w-10 h-10" /></div>;

    const getScoreColor = (score: number) => {
        if (score >= 85) return 'text-green-500';
        if (score >= 60) return 'text-yellow-500';
        return 'text-red-400';
    };

    return (
        <div className="max-w-6xl mx-auto p-6 space-y-8 print:p-0 print:space-y-0">
            <style>{printStyles}</style>

            <div className="flex justify-between items-center no-print mb-8">
                <h1 className="text-3xl font-bold text-textMain">Аналітичний центр</h1>
                {report && (
                    <button onClick={() => window.print()} className="flex items-center gap-2 bg-primary hover:bg-primaryHover text-white px-5 py-2.5 rounded-xl text-sm font-medium transition-colors shadow-lg shadow-primary/20">
                        <Printer size={18} /> Друк звіту
                    </button>
                )}
            </div>

            <div className="grid lg:grid-cols-12 gap-6 print:block">
                {/* 1. Модулі (no-print) */}
                <div className="no-print lg:col-span-3 bg-surface p-6 rounded-3xl border border-surfaceBorder h-fit">
                    <h2 className="font-bold mb-4 flex items-center gap-2 text-lg"><BookOpen className="text-primary" /> Курси</h2>
                    <div className="space-y-2">
                        {modules.length === 0 ? <p className="text-sm text-textMuted">Немає курсів</p> : modules.map(m => (
                            <button
                                key={m.id}
                                onClick={() => handleSelectModule(m.id)}
                                className={`w-full text-left p-3 rounded-xl transition-all text-sm font-medium ${selectedModuleId === m.id ? 'bg-primary text-white shadow-md' : 'hover:bg-mainBg text-textMain'}`}
                            >
                                {m.title}
                            </button>
                        ))}
                    </div>
                </div>

                {/* 2. Студенти (no-print) */}
                <div className="no-print lg:col-span-3 bg-surface p-6 rounded-3xl border border-surfaceBorder h-fit max-h-[600px] overflow-y-auto custom-scrollbar">
                    <h2 className="font-bold mb-4 flex items-center gap-2 text-lg"><Users className="text-primary" /> Студенти</h2>
                    {!selectedModuleId ? (
                        <p className="text-sm text-textMuted">Оберіть курс ліворуч</p>
                    ) : (
                        <div className="space-y-2">
                            {students.length === 0 ? <p className="text-sm text-textMuted">Немає студентів</p> : students.map(s => (
                                <button
                                    key={s.id}
                                    onClick={() => handleSelectStudent(s)}
                                    className={`w-full text-left p-3 rounded-xl transition-all flex items-center gap-3 text-sm font-medium ${selectedStudent?.id === s.id ? 'bg-primary text-white shadow-md' : 'hover:bg-mainBg text-textMain'}`}
                                >
                                    <div className={`p-1.5 rounded-full ${selectedStudent?.id === s.id ? 'bg-white/20' : 'bg-primary/10 text-primary'}`}>
                                        <User size={16} />
                                    </div>
                                    {s.first_name} {s.last_name}
                                </button>
                            ))}
                        </div>
                    )}
                </div>

                {/* 3. Звіт (print-only) */}
                <div className="lg:col-span-6 bg-surface p-6 sm:p-8 rounded-3xl border border-surfaceBorder print-only">
                    {isDataLoading ? (
                        <div className="flex justify-center p-20 no-print"><Loader2 className="animate-spin text-primary w-10 h-10" /></div>
                    ) : report ? (
                        <div className="space-y-8">

                            {/* Офіційний заголовок тільки для друку */}
                            <div className="hidden print:block border-b border-gray-300 pb-4 mb-6">
                                <h1 className="text-2xl font-bold uppercase tracking-wider text-black">Індивідуальний звіт студента</h1>
                                <p className="text-gray-500 mt-1">Дата формування: {new Date().toLocaleDateString('uk-UA')}</p>
                            </div>

                            {/* Шапка студента */}
                            <div className="flex items-center gap-4 pb-6 border-b border-surfaceBorder print-box !border-x-0 !border-t-0">
                                <div className="w-14 h-14 bg-primary/10 rounded-full flex items-center justify-center shrink-0 print:border print:border-gray-300 print:bg-transparent">
                                    <User className="text-primary w-7 h-7" />
                                </div>
                                <div>
                                    <p className="text-xs text-textMuted uppercase tracking-wider font-semibold">Студент</p>
                                    <h3 className="text-2xl font-bold text-textMain">{report.student_name}</h3>
                                </div>
                            </div>

                            {/* Статистика */}
                            <div className="grid grid-cols-2 gap-4">
                                <StatBox label="Пройдено тестів" value={report.quizzes_taken} icon={<BarChart3 size={20} />} />
                                <StatBox
                                    label="Середній бал"
                                    value={`${report.average_score}%`}
                                    icon={<Award size={20} />}
                                    colorClass={getScoreColor(report.average_score)}
                                />
                                <StatBox label="Ідеальні спроби" value={report.perfect_scores} icon={<CheckCircle size={20} />} colorClass="text-primary" />
                                <StatBox label="Помилки" value={report.mistakes_count} icon={<AlertTriangle size={20} />} colorClass={report.mistakes_count > 0 ? "text-red-400" : "text-green-500"} />
                            </div>

                            {/* Помилки */}
                            {report.mistakes_count > 0 && (
                                <div className="pt-4">
                                    <h4 className="font-bold text-sm text-red-400 uppercase tracking-wider flex items-center gap-2 mb-4">
                                        <AlertTriangle size={16} /> Проблемні питання
                                    </h4>
                                    <ul className="space-y-2">
                                        {mistakes.map((text, i) => (
                                            <li key={i} className="text-sm bg-red-500/5 border border-red-500/10 p-4 rounded-xl text-textMain leading-relaxed print-box">
                                                {text}
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            )}

                            {report.mistakes_count === 0 && report.quizzes_taken > 0 && (
                                <div className="pt-4 flex items-center gap-3 text-green-500 bg-green-500/10 p-4 rounded-xl border border-green-500/20 print-box">
                                    <CheckCircle size={24} />
                                    <p className="font-medium">Відмінна робота! У студента немає жодної відкритої помилки.</p>
                                </div>
                            )}
                        </div>
                    ) : (
                        <div className="text-center text-textMuted py-20 flex flex-col items-center no-print">
                            <BarChart3 className="w-16 h-16 opacity-20 mb-4" />
                            <p>Оберіть студента для перегляду його прогресу</p>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}

// Компонент плашки зі статистикою
function StatBox({ label, value, icon, colorClass = "text-textMain" }: { label: string, value: any, icon: React.ReactNode, colorClass?: string }) {
    return (
        <div className="bg-mainBg border border-surfaceBorder p-5 rounded-2xl print-box flex flex-col justify-center">
            <div className="flex items-center gap-3 mb-2">
                <div className={`${colorClass}`}>{icon}</div>
                <p className="text-textMuted text-[11px] uppercase tracking-wider font-semibold">{label}</p>
            </div>
            <p className={`text-2xl font-bold ${colorClass === 'text-textMain' ? 'text-textMain' : colorClass}`}>
                {value}
            </p>
        </div>
    );
}