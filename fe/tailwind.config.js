/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./src/**/*.{html,ts}", // 指定 Angular 项目中 HTML 和 TS 文件的路径
  ],
  theme: {
    extend: {
      animation: {
        "fade-in": "fade-in 1s ease-in-out",
        "scale-in": "scale-in 1s ease-in-out",
      },
      keyframes: {
        "fade-in": {
          "0%": { opacity: "0" },
          "100%": { opacity: "1" },
        },
        "scale-in": {
          "0%": { transform: "scale(0)" },
          "100%": { transform: "scale(1)" },
        },
      },
    },
  },
  plugins: [],
};
