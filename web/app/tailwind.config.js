/** @type {import('tailwindcss').Config} */
module.exports = {
    // content: [],
    // content: ["/css/**/*.{html,js}"],
    // content: ["/templates/*"],

    // * By default Tailwind resolves non-absolute content paths relative to the current working directory
    //   ('relative' to where tailwind is executed), as of tailwind v3.4.3 (behaviour might change in next major according to tailwind docs)
    //   see: https://tailwindcss.com/docs/content-configuration#using-relative-paths
    // *  To resolve paths relative to the tailwind.config.js file, use the object notation
    //    and for `content` configuration set the `relative` property to true (content: {relative: true})
    content: {
        relative: true, // indicates relative to 'tailwind.config.js' and not 'current working directory'
        files: ["./../../pkg/router/routes/templates/**/*.{html,tmpl}"],
    },
    //
    //
    safelist: [
        // 'bg-red-500',
        // 'text-3xl',
        // 'lg:text-4xl',
        // {
        //   pattern: /([a-zA-Z]+)-./, // all of tailwind
        // },
    ],
    darkMode: 'class',
    theme: {
        extend: {},
    },
    plugins: [],
}
