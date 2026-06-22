export interface User {
    id: number;
    email: string;
    first_name: string;
    last_name: string;
}

export interface Module {
    id: number;
    title: string;
    description: string;
    invite_code: string;
    student_count?: number;
    created_by: number; // <-- Додай цей рядок
}

export interface Flashcard {
    id: number;
    question: string;
    answer: string;
}

export interface QuizSubmitRequest {
    user_id: number;
    quiz_id: number;
    score: number;
    total_questions: number;
    answers: Array<{
        question_id: number;
        is_correct: boolean;
    }>;
}
