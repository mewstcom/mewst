/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './app/components/**/*.erb',
    './app/components/**/*.rb',
    './app/helpers/**/*.rb',
    './app/javascript/**/*.ts',
    './app/views/**/*.erb',
    './config/locales/*.yml',
    './public/*.html',
  ],
  theme: {
    extend: {},
  },
  plugins: [require('daisyui')],
  daisyui: {
    themes: [
      {
        mewst: {
          'base-100': '#f6f2eb',
          'base-300': '#ffffff',
          error: '#ef4444',
          info: '#0284c7',
          neutral: '#737373',
          primary: '#f4a90e',
          secondary: '#5f0f40',
          success: '#22c55e',
        },
      },
    ],
  },
};
