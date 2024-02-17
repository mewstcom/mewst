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
};
