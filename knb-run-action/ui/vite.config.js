import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import federation from "@originjs/vite-plugin-federation";

export default defineConfig({
  server: {
    cors: true // Autorise toutes les origines en dev
  },  
  plugins: [
    vue(),
    federation({
      // Placeholder qui sera remplacé par le Go au runtime (Option 1)
      name: 'KNB_DYNAMIC_SERVICE_NAME',
      filename: 'remoteEntry.js',
      exposes: {
        // C'est ce chemin que le Shell cherchera
        './Module': './src/main.js',
        './Config': './src/remote-config.js',
        './FormBuilder': './src/FormBuilder.js',
      },
      shared: ['vue','pinia']
    })
  ],
  build: {
    target: 'esnext',
    assetsInlineLimit: 0,
    minify: false, // Gardez-le à false pour le moment pour debug
    cssCodeSplit: false,
    rollupOptions: {
      output: {
        format: 'esm',
        entryFileNames: 'assets/[name].js',
        chunkFileNames: 'assets/[name].js',
        assetFileNames: 'assets/[name].[ext]'
      }
    }
  }});