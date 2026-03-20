import { defineConfig } from 'vite';
import federation from "@originjs/vite-plugin-federation";

export default defineConfig({
  plugins: [
    federation({
      // Le Go remplacera ce placeholder par 'knb-shell' au runtime
      name: 'KNB_DYNAMIC_SERVICE_NAME',
      filename: 'remoteEntry.js',
      exposes: {
        // Forcez l'exposition d'un composant même minimal pour générer le fichier
        './App': './src/main.js',
      },      
      // On laisse remotes vide ici car nous allons utiliser 
      // la méthode de chargement dynamique (Runtime Discovery) 
      // via une API ou Consul.
      remotes: {
        dynamic_resolver: 'http://localhost/placeholder.js'
      },

      // MÊME SI LE SHELL EST EN VANILLA JS :
      // On déclare 'vue' en shared. Cela permet au Shell de fournir 
      // l'instance de Vue à tous les Remotes qui en ont besoin.
      // Note : Il faudra faire un 'npm install vue' dans le dossier web du shell.
      shared:['vue','pinia']
      // shared: {
      //   vue: {
      //     singleton: true, // Une seule instance pour tout le monde
      //     requiredVersion: '^3.3.0'
      //   }
      // }
    })
  ],
  build: {
    target: 'esnext',
    assetsInlineLimit: 0,
    minify: false,
    cssCodeSplit: false,
    rollupOptions: {
      output: {
        // Garantit que les fichiers ne changent pas de nom à chaque build
        // pour faciliter l'injection par le backend Go
        entryFileNames: `assets/[name].js`,
        chunkFileNames: `assets/[name].js`,
        assetFileNames: `assets/[name].[ext]`
      }
    }
  }
});