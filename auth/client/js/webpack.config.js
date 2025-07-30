const path = require('path');

module.exports = (env, argv) => {
    const app = env.app;
    return {
        entry: `./src/App.js`,
        output: {
            filename: 'bundle.js',
            path: path.resolve(__dirname, `./apps/${app}/dist`),
        },
        mode: 'development', // or 'production'
        module: {
            rules: [
                {
                    test: /\.js$/,
                    exclude: /node_modules/,
                    use: {
                        loader: 'babel-loader',
                    },
                },
            ],
        },
        resolve: {
            modules: [path.resolve(__dirname, 'node_modules')],
        },
    }
};
