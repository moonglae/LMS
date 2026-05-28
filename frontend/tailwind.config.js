/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Твоя палітра
        mainBg: '#050505',      // Майже глибокий чорний для фону
        surface: '#111111',     // Трохи світліший чорний для карток/панелей
        surfaceBorder: '#222222', // Темно-сірий для ледь помітних меж
        primary: '#3b82f6',     // Яскравий синій (Tailwind blue-500)
        primaryHover: '#2563eb',// Темніший синій для наведення
        textMain: '#f3f4f6',    // Світло-сірий для читабельного тексту
        textMuted: '#9ca3af',   // Приглушений сірий для другорядного тексту
      }
    },
  },
  plugins: [],
}