/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './public/*.html',
    './app/helpers/**/*.rb',
    './app/javascript/**/*.ts',
    './app/components/**/*.erb',
    './app/views/**/*.erb',
  ],
  theme: {
    extend: {},
  },
  plugins: [require('daisyui')],
  daisyui: {
    themes: [
      {
        mewst: {
          primary: '#F4A90E',
          secondary: '#5F0F40',
          'base-100': '#ffffff',
        },
      },
    ],
  },
};
