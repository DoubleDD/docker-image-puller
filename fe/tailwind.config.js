/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: "class", // 使用 class 模式来启用暗色模式
  content: [
    "./src/**/*.{html,ts}", // 指定 Angular 项目中 HTML 和 TS 文件的路径
  ],
  theme: {
    extend: {
      boxShadow: {
        "custom-light": "0 4px 6px rgba(255, 255, 255, 0.3)", // 白色阴影
        "custom-dark": "0 4px 6px rgba(0, 0, 0, 0.5)", // 黑色阴影
        "custom-blue": "0 4px 6px rgba(59, 130, 246, 0.5)", // 蓝色阴影
      },
      backdropFilter: {
        none: "none",
        custom: "40px", // 更强的模糊
        blur: "blur(50px)",
      },
      backgroundOpacity: {
        10: "0.1",
        15: "0.15",
        20: "0.2",
        30: "0.3",
        40: "0.4",
        50: "0.5",
        60: "0.6",
        70: "0.7",
        80: "0.8",
        90: "0.9",
        95: "0.95",
      },
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
  variants: {
    extend: {
      backdropFilter: ["responsive"],
      backgroundOpacity: ["responsive"],
    },
  },
  plugins: [require("tailwindcss-filters")],
};
