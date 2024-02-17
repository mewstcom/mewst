/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './public/*.html',
    './app/helpers/**/*.rb',
    './app/javascript/**/*.ts',
    './app/components/**/*.erb',
    './app/views/**/*.erb',
    './config/locales/*.yml',
  ],
  theme: {
    extend: {},
  },
  plugins: [require('daisyui')],
  daisyui: {
    themes: [
      {
        mewst: {
          'base-100': '#faf0dc',
          info: '#0284c7',
          neutral: '#ffffff',
          primary: '#f4a90e',
          secondary: '#5f0f40',
        },
      },
    ],
  },
};
