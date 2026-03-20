import Keycloak from 'keycloak-js';
import './style.css'
import { __federation_method_getRemote, __federation_method_setRemote } from '__federation__';
import { ActModeler } from './ActModeler.js';

function initFederationGlobalScope() {
  if (!window.__federation_shared__) {
    // Initialise l'objet racine utilisé par le plugin pour stocker les versions des libs
    window.__federation_shared__ = window.__federation_shared__ || {};
    
    // Optionnel : Vous pouvez pré-enregistrer des dépendances si nécessaire
    // mais le plugin le fait généralement via les imports asynchrones.
    console.log("Federation shared scope initialized manually.");
  }
}

initFederationGlobalScope(); 

const keycloak = new Keycloak({
    url: 'https://auth.wosa.local', // URL de votre serveur Keycloak
    realm: 'knb-cloud',
    clientId: 'knb-client',//'knb-web'
});


/**
 * Point d'entrée principal de l'application
 */
async function bootstrap() {
    console.log("🚀 Initialisation du Shell KNB...");
    const statusText = document.getElementById('loading-status');
    const loadingScreen = document.getElementById('loading-screen');

    try {
        // 2. Initialisation de l'authentification
        // 'login-required' force la redirection vers Keycloak si l'utilisateur n'est pas connecté
        const authenticated = await keycloak.init({ 
            onLoad: 'login-required',
            checkLoginIframe: false ,
            flow: 'implicit',
        });

        if (!authenticated) {
            console.warn("User not authenticated");
            return;
        }

        console.log("✅ Authentifié en tant que :", keycloak.tokenParsed.preferred_username);
        
        // // On stocke le token globalement pour que les MFEs puissent l'utiliser pour leurs appels API
        window.userToken = keycloak.token;

        // 3. Récupération de la liste des micro-frontends depuis le backend Go
        const response = await fetch('/api/discovery', {
            headers: {
                'Authorization': `Bearer ${keycloak.token}`
            }
        });
        const remotes = await response.json();

        if (!remotes || remotes.length === 0) {
            console.warn("⚠️ Aucun micro-frontend trouvé dans Consul.");
            renderEmptyState();
            return;
        }
    
        // 4. Enregistrement de chaque service trouvé dans le moteur de fédération
        for (let service of remotes) {
            // Configuration de la fédération pour chaque service
            service.mfeConfig = null; // Initialisation de la config du MFE
            __federation_method_setRemote(service.name, {
                url: () => Promise.resolve(`${service.url}/remoteEntry.js`),
                format: 'esm',
                from: 'vite'
            });

            try {
                // Tentative de récupération de la config via le point d'entrée ./Config exposé par le MFE
                const configModule = await __federation_method_getRemote(service.name, './Config');
                const mfeConfig = await configModule.getConfig(service.name);
                
                // On stocke la config dans l'objet service pour que renderShell puisse l'utiliser
                service.mfeConfig = mfeConfig; 
            } catch (e) {
                console.warn(`Le service ${service.name} ne possède pas de configuration de menu.`);
            }
        }
        // 5. Construction de l'interface utilisateur
        renderShell(remotes);
        if (loadingScreen) {
            loadingScreen.classList.add('opacity-0', 'pointer-events-none');
            setTimeout(() => loadingScreen.remove(), 500);
        }
    } catch (error) {
        console.error("❌ Erreur lors de l'initialisation du Shell:", error);
        showNotification({
        title: "Erreur de connexion",
        message: "Impossible d'initialiser la session sécurisée ou de contacter le service d'authentification.",
        type: "error",
        technicalDetails: error.message
         });
        // renderErrorModal(error);
    }
}

function getIconSvg(iconName) {
    // Si vide ou 'default', on met une icône générique (petit point)
    const name = (!iconName || iconName === 'default') ? 'circle' : iconName.toLowerCase();
    
    // Dictionnaire enrichi avec les icônes classiques
    const icons = {
        // Base & Structure
        'circle': '<circle cx="12" cy="12" r="3"></circle>',
        'dashboard': '<rect x="3" y="3" width="7" height="7"></rect><rect x="14" y="3" width="7" height="7"></rect><rect x="14" y="14" width="7" height="7"></rect><rect x="3" y="14" width="7" height="7"></rect>',
        'home': '<path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path><polyline points="9 22 9 12 15 12 15 22"></polyline>',
        
        // Organisation & Utilisateurs
        'organization': '<rect x="2" y="7" width="20" height="14" rx="2" ry="2"></rect><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"></path>',
        'users': '<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path>',
        'user': '<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle>',
        
        // Contenu & Data
        'folder': '<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>',
        'file': '<path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"></path><polyline points="13 2 13 9 20 9"></polyline>',
        'database': '<ellipse cx="12" cy="5" rx="9" ry="3"></ellipse><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"></path><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"></path>',
        'chart': '<line x1="18" y1="20" x2="18" y2="10"></line><line x1="12" y1="20" x2="12" y2="4"></line><line x1="6" y1="20" x2="6" y2="14"></line>',
        
        // Actions & Outils
        'settings': '<circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>',
        'search': '<circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line>',
        'mail': '<path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"></path><polyline points="22,6 12,13 2,6"></polyline>',
        'bell': '<path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path><path d="M13.73 21a2 2 0 0 1-3.46 0"></path>',
        
        // Communication & Sécurité
        'shield': '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>',
        'key': '<path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3m-3-3l-2.5-2.5"></path>',
        'lock': '<rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path>',
        
        // Divers
        'help': '<circle cx="12" cy="12" r="10"></circle><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path><line x1="12" y1="17" x2="12.01" y2="17"></line>',
        'calendar': '<rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect><line x1="16" y1="2" x2="16" y2="6"></line><line x1="8" y1="2" x2="8" y2="6"></line><line x1="3" y1="10" x2="21" y2="10"></line>',
        'layers': '<polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline>',

        'book': '<path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"></path><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"></path>',
        'brain': '<path d="M9.5 2A5 5 0 0 1 12 11a5 5 0 0 1 2.5-9 5 5 0 1 1 5 8 4.5 4.5 0 0 1-3 8.5h-9A4.5 4.5 0 0 1 4.5 10 5 5 0 0 1 9.5 2z"></path><path d="M12 11v10"></path>',
        'lightbulb': '<path d="M15 14c.2-1 .7-1.7 1.5-2.5 1-.9 1.5-2.2 1.5-3.5A5 5 0 0 0 8 8c0 1.3.5 2.6 1.5 3.5.8.8 1.3 1.5 1.5 2.5"></path><path d="M9 18h6"></path><path d="M10 22h4"></path>',
        'graduation': '<path d="M22 10L12 5 2 10l10 5 10-5z"></path><path d="M6 12v5c0 2 2 3 6 3s6-1 6-3v-5"></path>',
        'wiki': '<path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"></path><path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"></path>',    
        
        // Développement & Logique Métier
        'actions': '<polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline>', // Un signal (Activity/Action)
        'screens': '<rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect><line x1="8" y1="21" x2="16" y2="21"></line><line x1="12" y1="17" x2="12" y2="21"></line>', // Un moniteur (Ecran/UI)
        'rules': '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline><line x1="16" y1="13" x2="8" y2="13"></line><line x1="16" y1="17" x2="8" y2="17"></line><line x1="10" y1="9" x2="8" y2="9"></line>', // Document avec lignes (Règles métier)
        'builder': '<path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"></path>', // Une clé à molette (Builder/Outils)
        'bug': '<rect x="8" y="7" width="8" height="10" rx="4"></rect><path d="M6 7l2 3"></path><path d="M18 7l-2 3"></path><path d="M6 17l2-3"></path><path d="M18 17l-2-3"></path><path d="M7 11h10"></path><path d="M7 15h10"></path><path d="M12 20v-3"></path><path d="M12 7V4"></path>', // Un insecte (Bug/Ticket) 

        // Workflow & Gouvernance
        'process': '<path d="M22 12h-4l-3 9L9 3l-3 9H2"></path>', // Variante plus "système" (Pulse/Process)
        // OU alternative pour Process (étapes) :
        'steps': '<circle cx="12" cy="12" r="3"></circle><path d="M3 12h6m6 0h6"></path><circle cx="3" cy="12" r="1"></circle><circle cx="21" cy="12" r="1"></circle>',

        'roles': '<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M19 8l2 2 4-4"></path>', // Utilisateur avec une coche (Rôles/Permissions)

        'labels': '<path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"></path><line x1="7" y1="7" x2="7.01" y2="7"></line>', // Étiquette (Libellés/Tags)        
    };

    return `
        <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" viewBox="0 0 24 24">
            ${icons[name] || icons['circle']}
        </svg>`;
}

// Fermer les flyouts si on clique n'importe où ailleurs
document.addEventListener('click', () => {
    document.querySelectorAll('.flyout-active').forEach(el => el.classList.remove('flyout-active'));
});
/**
 * Génère l'interface du Shell (Header, Menu, Logout)
 */
function renderShell(remotes) {
    const app = document.querySelector('#app');
    const userName = keycloak.tokenParsed?.given_name || "Utilisateur";
    const userLang = keycloak.tokenParsed?.lan || "fr";

    // Génération du menu (toujours récursive)
    const allMenusHtml = remotes.map(service => {
        if (service.mfeConfig && service.mfeConfig.menu && service.mfeConfig.menu[userLang] 
            && service.mfeConfig.menu[userLang].menu) {
            return `
                <div class="mb-4">
                    <div class="px-4 py-2 text-[10px] font-bold text-slate-400 uppercase tracking-widest service-title">${service.name}</div>
                    ${buildMenu(service.mfeConfig.menu[userLang].menu, service.name)}
                </div>`;
        }
        return '';
    }).join('');

    app.innerHTML = `
<style>
    /* --- 1. CONFIGURATION DE BASE (SIDEBAR) --- */
    #sidebar {
        width: 256px;
        background: white;
        z-index: 40;
        transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        border-right: 1px solid #e2e8f0;
    }

    /* --- 2. ÉTAT MINI (96px) --- */
    #sidebar.mini {
        width: 56px;
        overflow: visible !important;
    }

    #sidebar.mini nav {
        overflow: visible !important;
        padding-top: 10px !important;
    }

    /* Cache les éléments inutiles en mode mini */
    #sidebar.mini > nav .sidebar-label, 
    #sidebar.mini .service-title,
    #sidebar.mini .service-title-container,
    #sidebar.mini > nav > div > .menu-container > button .chevron {
        display: none !important;
        height: 0; margin: 0; padding: 0; overflow: hidden;
    }

    /* --- 3. ITEMS DE MENU (BOUTONS) --- */
    .menu-container {
        position: relative;
        overflow: visible !important;
        margin-bottom: 2px;
    }

    /* Style commun des boutons (Flex est la clé ici) */
    #sidebar .menu-container button,
    #sidebar.mini .submenu-container button {
        display: flex !important;
        width: 100% !important;
        align-items: center !important;
        text-align: left !important;
        padding: 8px 12px !important;
        transition: background 0.2s;
    }

    /* Ajustement hauteur en mode mini */
    #sidebar.mini > nav > div > .menu-container > button {
        height: 48px !important;
        justify-content: center !important;
        padding: 0 !important;
    }

    /* Icônes Géantes Premier Niveau */
    #sidebar.mini > nav > div > .menu-container > button svg:not(.chevron) {
        width: 32px !important;
        height: 32px !important;
        min-width: 32px !important;
        transition: transform 0.2s;
    }

    #sidebar.mini .menu-container:hover svg:not(.chevron) {
        transform: scale(1.1);
        color: #4f46e5;
    }

    /* --- 4. FLYOUT & TOOLTIP --- */
    
    /* Flyout (Le volet qui surgit) */
    #sidebar.mini .menu-container.flyout-active > .submenu-container {
        display: block !important;
        position: fixed;
        left: 56px;
        background: white;
        border: 1px solid #e2e8f0;
        box-shadow: 10px 10px 20px rgba(0,0,0,0.1);
        border-radius: 0 8px 8px 0;
        min-width: 220px;
        z-index: 1000;
        padding: 8px 4px;
        animation: flyoutIn 0.2s ease-out;
    }

    /* Réaffichage du texte dans le Flyout */
    #sidebar.mini .submenu-container .sidebar-label {
        display: inline-block !important;
        opacity: 1 !important;
        color: #334155 !important;
        font-size: 13px !important;
        margin-left: 10px;
    }

    /* Tooltip au survol simple */
    #sidebar.mini .menu-container:hover::after {
        content: attr(data-label);
        position: absolute;
        left: 100%; top: 50%; transform: translateY(-50%);
        margin-left: 15px;
        background: #1e293b; color: white;
        padding: 6px 12px; border-radius: 6px; font-size: 12px;
        white-space: nowrap; z-index: 9999;
        box-shadow: 4px 4px 15px rgba(0,0,0,0.3);
    }

    /* --- 5. LE CHEVRON (Indicateur sous-menu) --- */
    .chevron {
        width: 12px !important;
        height: 12px !important;
        min-width: 12px !important;
        /* LE MARGIN-LEFT: AUTO EST ICI : Il pousse le chevron tout à droite */
        margin-left: auto !important; 
        color: #94a3b8 !important;
        stroke-width: 2.5px !important;
        transition: transform 0.2s;
        flex-shrink: 0;
    }

    .chevron.rotate-90 {
        transform: rotate(90deg);
    }

    /* Icônes normales dans les sous-menus */
    #sidebar.mini .submenu-container svg:not(.chevron) {
        width: 18px !important;
        height: 18px !important;
    }

    /* --- 6. ANIMATIONS --- */
    @keyframes flyoutIn {
        from { opacity: 0; transform: translateX(-10px); }
        to { opacity: 1; transform: translateX(0); }
    }




    /* 1. RÉAFFICHAGE DU TEXTE (Ciblage précis du span uniquement) */
    #sidebar.mini .submenu-container span.sidebar-label,
    #sidebar.mini .flyout-active span.sidebar-label {
        display: inline-block !important; /* inline-block est plus stable que block ici */
        visibility: visible !important;
        opacity: 1 !important;
        width: auto !important;
        height: auto !important;
        margin-left: 10px !important;
        color: #334155 !important;
        font-size: 13px !important;
    }

    /* 2. VERROUILLAGE DE SÉCURITÉ (Pour empêcher les chevrons de grossir) */
    /* Cette règle s'assure que même si le texte réapparaît, le chevron reste petit */
    #sidebar.mini .submenu-container .chevron,
    #sidebar.mini .flyout-active .chevron {
        width: 12px !important;
        height: 12px !important;
        min-width: 12px !important;
        max-width: 12px !important;
        margin-left: auto !important; /* Garde le chevron à droite */
        display: block !important;
    }

    /* 3. VERROUILLAGE DES ICONES DANS LE FLYOUT */
    #sidebar.mini .submenu-container svg:not(.chevron) {
        width: 18px !important;
        height: 18px !important;
        min-width: 18px !important;
    }

    /* 4. ALIGNEMENT DU BOUTON DANS LE FLYOUT */
    #sidebar.mini .submenu-container button {
        display: flex !important;
        flex-direction: row !important;
        align-items: center !important;
        justify-content: flex-start !important;
        padding: 8px 12px !important;
        height: auto !important; /* Important : laisser la hauteur s'adapter au contenu */
    }
    #sidebar.mini .menu-container > button.mainmenu-button .sidebar-label {
        display: none !important;
    }
    #sidebar.mini .menu-container.flyout-active::after,
    #sidebar.mini .menu-container.flyout-active::before {
        display: none !important;
        opacity: 0 !important;
        visibility: hidden !important;
    }        
    #sidebar.mini .menu-container.flyout-active > .submenu-container {
        z-index: 10000 !important; /* Supérieur au 9999 de l'info-bulle */
    }     

/* --- LIGNE DE GUIDAGE GÉNÉRALE (Normal & Flyout) --- */
    /* On cible les sous-menus qui sont ENFANTS d'un autre sous-menu */
    .submenu-container .submenu-container {
        border-left: 2px solid #cbd5e1; /* slate-300 */
        transition: border-color 0.2s ease;
        display: none; /* Caché par défaut, géré par .hidden en JS */
    }

    /* --- AJUSTEMENTS SPÉCIFIQUES AU MODE NORMAL --- */
    #sidebar:not(.mini) .submenu-container {
        border-left: 2px solid #cbd5e1;
        margin-left: 1.75rem; 
        padding-left: 0.5rem;
    }

    /* --- AJUSTEMENTS SPÉCIFIQUES AU MODE MINI (Flyout) --- */
    /* Dans le Flyout, on réduit les marges car l'espace est précieux */
    #sidebar.mini .submenu-container .submenu-container {
        margin-left: 1.25rem !important; /* Moins de décalage qu'en mode large */
        padding-left: 0.25rem !important;
        border-left: 1.5px solid #6b828bff !important; /* Ligne un peu plus fine et claire #e2e8f0*/
    }

    /* L'effet de survol qui illumine la ligne (marche aussi en mini) */
    .menu-container:hover > .submenu-container {
        border-left-color: #3f8d68ff !important; /* indigo-500 #6366f1*/
    }

    /* --- ÉTAT DE VISIBILITÉ --- */
    .submenu-container:not(.hidden) {
        display: block !important;
    }    
    /* ... tes styles existants ... */
    .btn-header-tool {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 12px;
        background: rgba(79, 70, 229, 0.2); /* Indigo transparent */
        border: 1px solid rgba(79, 70, 229, 0.4);
        border-radius: 6px;
        font-size: 10px;
        font-weight: 800;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        transition: all 0.2s;
    }
    .btn-header-tool:hover {
        background: rgba(79, 70, 229, 0.4);
        border-color: #6366f1;
    }
</style>
        <div class="flex flex-col h-screen overflow-hidden bg-slate-50">
            <header class="h-12 bg-[#1e293b] text-white flex items-center justify-between px-4 z-50 shrink-0">
                <div class="flex items-center gap-4">
                    <button id="toggle-sidebar" class="p-1 hover:bg-slate-700 rounded transition-colors">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/></svg>
                    </button>
                    <div class="font-bold tracking-wider text-indigo-400">KNB <span class="text-white font-light text-xs ml-1">Cloud</span></div>
                </div>

                <div class="flex items-center gap-4">
                    <button id="open-act-builder" class="btn-header-tool text-indigo-300 hover:text-white">
                        ${getIconSvg('builder')}
                        <span>Concepteur d'Actes</span>
                    </button>
                
                    <span class="text-[11px] font-medium opacity-80 uppercase tracking-tight">👤 ${userName}</span>
                    <button id="logout-btn" class="text-red-400 hover:text-red-300 p-1 transition-colors">
                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" /></svg>
                    </button>
                </div>
            </header>

            <div class="flex flex-1 overflow-hidden relative">
                <aside id="sidebar" class="w-64 bg-white border-r border-slate-200 flex flex-col transition-all duration-300 ease-in-out">
                    <nav class="flex-1 overflow-y-auto py-4">
                        ${allMenusHtml}
                    </nav>
                </aside>

                <main id="content-area" class="flex-1 overflow-y-auto p-6 bg-slate-50">
                    <div class="max-w-5xl mx-auto">
                        <div class="bg-white p-10 rounded-2xl shadow-sm border border-slate-200">
                            <h2 class="text-2xl font-bold text-slate-800 mb-2">Bienvenue, ${userName}</h2>
                            <p class="text-slate-500">Sélectionnez un module dans le menu latéral pour commencer.</p>
                        </div>
                    </div>
                </main>
            </div>
        </div>
    `;

    document.getElementById('toggle-sidebar').addEventListener('click', () => {
        const sidebar = document.getElementById('sidebar');
        const isEnteringMini = !sidebar.classList.contains('mini'); // On va ajouter la classe mini

        if (isEnteringMini) {
            // --- NETTOYAGE COMPLET ---
            // 1. On retire la classe active des conteneurs
            document.querySelectorAll('.flyout-active').forEach(el => el.classList.remove('flyout-active'));

            // 2. On cache tous les sous-menus (submenu-container)
            document.querySelectorAll('.submenu-container').forEach(sub => {
                sub.classList.add('hidden');
                // Crucial : on nettoie les styles inline (fixed, top, left)
                sub.style.position = '';
                sub.style.top = '';
                sub.style.left = '';
            });

            // 3. On remet tous les chevrons à 0°
            document.querySelectorAll('.chevron').forEach(chev => chev.classList.remove('rotate-90'));
        }

        // Enfin, on bascule la classe sur la sidebar
        sidebar.classList.toggle('mini');
    });
    document.getElementById('logout-btn').addEventListener('click', () => keycloak.logout());
// Gestion du clic sur le Concepteur d'Actes
    document.getElementById('open-act-builder').addEventListener('click', () => {
        const contentArea = document.getElementById('content-area');
        
        // 1. Nettoyage de la zone
        contentArea.innerHTML = `
            <div class="flex flex-col h-full bg-white rounded-xl shadow-lg border border-slate-200 overflow-hidden">
                <div id="act-builder-root" class="flex-1 overflow-hidden">
                    <div class="p-10 text-center">
                        <div class="animate-pulse text-indigo-600 mb-4">Initialisation du moteur de conception...</div>
                    </div>
                </div>
            </div>
        `;

        // 2. Appel de la fonction de montage de ton ActModeler
        // Ici, on suppose que ActModeler a une méthode init ou mount
        if (typeof ActModeler.mount === 'function') {
            ActModeler.mount(document.getElementById('act-builder-root'));
        } else {
            // Si tu n'as pas encore fait de fonction mount, on simule le chargement
            const container = document.getElementById('act-builder-root');
            const demoElements = [
                { type: 'admin_acte', props: { documentType: 'ARRETE', decideWord: 'DECIDE', vus: [] } }
            ];
            container.innerHTML = ActModeler.renderToHTML(demoElements, { NOM_AGENT: userName });
        }
        
        // Mettre à jour le Breadcrumb
        const breadcrumb = document.getElementById('breadcrumb');
        if (breadcrumb) breadcrumb.innerText = "Système / Concepteur de Modèles";
    });    
}

/**
 * Charge dynamiquement un composant depuis un Remote
 */
async function loadMFE(serviceName) {
    // Force l'existence du scope de partage attendu par le plugin
    if (!window.__federation_shared__) window.__federation_shared__ = {};
    if (!window.__federation_shared__.default) window.__federation_shared__.default = {};

    const contentArea = document.querySelector('#content-area');
    contentArea.innerHTML = `
        <div style="display: flex; align-items: center; gap: 10px; color: #64748b;">
            <div class="spinner"></div> Chargement de ${serviceName}...
        </div>`;

    try {
        const container = await __federation_method_getRemote(serviceName, './Module');
        contentArea.innerHTML = '';
        
        if (container && typeof container.mount === 'function') {
            // On passe l'instance Keycloak ou le token au MFE s'il en a besoin
            container.mount(contentArea, { token: keycloak.token });
        } else {
            contentArea.innerHTML = `
                <div style="padding: 20px; background: #fff7ed; border: 1px solid #ffedd5; color: #9a3412; border-radius: 8px;">
                    Le module <strong>${serviceName}</strong> est chargé mais n'expose pas de fonction 'mount' compatible.
                </div>`;
        }
    } catch (err) {
        showNotification({
                title: `Erreur lors du chargement de ${serviceName}:`,
                message: "Échec de l'initialisation du cluster KNB.",
                type: "error",
                technicalDetails: err.message
        });        
        // console.error(`Erreur lors du chargement de ${serviceName}:`, err);
        // contentArea.innerHTML = `<p style="color:red;">Erreur lors du chargement du module distant.</p>`;
    }
}

function renderEmptyState() {
    const app = document.querySelector('#app');//'#app'
    
    app.innerHTML = `
        <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-sm">
            <div class="bg-white rounded-2xl shadow-xl max-w-sm w-full overflow-hidden  border border-slate-100">
                <div class="p-8 text-center">
                    <div class="mx-auto flex items-center justify-center h-16 w-16 rounded-full mb-4">
                        <svg class="h-10 w-10 text-orange-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                        </svg>
                    </div>
                    
                    <!-- <h3 class="text-lg font-semibold text-slate-800 mb-2">Catalogue vide</h3> --!>
                    <p class="text-slate-500 leading-relaxed mb-6">
                        Aucun service n'est actuellement disponible.
                    </p>
                    
                    <button onclick="window.location.reload()" 
                            class="inline-flex items-center px-6 py-2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-medium rounded-lg transition-colors text-sm">
                        <svg class="w-4 h-4 mr-2 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path d="M4 4v5h5M20 20v-5h-5M4 13a8.1 8.1 0 0015.5 2m.5-5a8.1 8.1 0 00-15.5-2" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>
                        Actualiser
                    </button>
                    <button id="logout-btn-2" 
                            class="inline-flex items-center px-6 py-2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-medium rounded-lg transition-colors text-sm">
                        <svg class="w-4 h-4 mr-2 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                        </svg>                        
                        Déconnexion
                    </button>
                </div>
            </div>
        </div>

        
    `;
 document.getElementById('logout-btn-2').onclick = () => keycloak.logout();   
}

/**
 * Affiche une notification plein écran (Modal) pour les erreurs ou les états vides
 * @param {Object} options - { title, message, type, technicalDetails }
 */
function showNotification({ title, message, type = 'info', technicalDetails = null }) {
    // const app = document.querySelector('#app');
    const app = document.querySelector('#content-area');
    // Configuration des styles selon le type
    const configs = {
        error: {
            bg: 'bg-red-50',
            iconColor: 'text-red-600',
            btnClass: 'bg-red-600 hover:bg-red-700 shadow-red-200',
            icon: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>`
        },
        info: {
            bg: 'bg-blue-50',
            iconColor: 'text-blue-600',
            btnClass: 'bg-slate-800 hover:bg-slate-900 shadow-slate-200',
            icon: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>`
        },
        empty: {
            bg: 'bg-slate-50',
            iconColor: 'text-indigo-500',
            btnClass: 'bg-indigo-600 hover:bg-indigo-700 shadow-indigo-200',
            // Icône simplifiée rappelant votre logo "Knowledge Bits"
            icon: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />`
        }
    };

    const config = configs[type] || configs.info;

    app.innerHTML = `
        <div class="fixed inset-0 z-[110] flex items-center justify-center p-4 bg-slate-900/80 backdrop-blur-md">
            <div class="bg-white rounded-3xl shadow-2xl max-w-md w-full overflow-hidden border border-white/20 transform transition-all animate-in zoom-in duration-300">
                <div class="p-8 text-center">
                    <div class="mx-auto flex items-center justify-center h-20 w-20 rounded-2xl ${config.bg} ${config.iconColor} mb-6">
                        <svg class="h-10 w-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            ${config.icon}
                        </svg>
                    </div>
                    
                    <h3 class="text-2xl font-bold text-slate-900 mb-3">${title}</h3>
                    <p class="text-slate-600 leading-relaxed mb-6">${message}</p>
                    
                    ${technicalDetails ? `
                    <div class="text-left mb-6">
                        <details class="group">
                            <summary class="list-none cursor-pointer flex items-center text-xs text-slate-400 font-mono italic">
                                <span>[+] Détails du processus</span>
                            </summary>
                            <div class="mt-2 p-3 bg-slate-900 rounded-lg text-[10px] font-mono text-green-400 break-all overflow-auto max-h-32">
                                ${technicalDetails}
                            </div>
                        </details>
                    </div>` : ''}

                    <button onclick="window.location.reload()" 
                            class="w-full py-4 px-6 ${config.btnClass} text-white font-bold rounded-2xl shadow-lg transition-all active:scale-95 flex items-center justify-center gap-2">
                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h5M20 20v-5h-5M4 13a8.1 8.1 0 0015.5 2m.5-5a8.1 8.1 0 00-15.5-2"/>
                        </svg>
                        Redémarrer le Shell
                    </button>
                </div>
            </div>
        </div>
    `;
}

// async function loadFeature(service, role, path, label) {
//     const featureName = element.getAttribute('data-feature'); // ex: "form-builder"
//     const requiredKnowledge = element.getAttribute('data-knowledge'); // ex: "expert"
    
//     if (featureName === 'form-builder') {
//         const context = {
//             role: keycloak.tokenParsed.realm_access.roles.includes('admin') ? 'admin' : 'user',
//             knowledge: requiredKnowledge
//         };
        
//         // On lance le chargement
//         mountRemoteFeature(mfeUrl, 'mfe_builder', './FormBuilder', context);
//     }
// }
// // main.js

// main.js

// async function mountRemoteFeature(remoteUrl, remoteName, exposedModule, context) {
//     // 1. Vérification immédiate du DOM pour éviter l'erreur 'innerHTML of null'
//     const container = document.getElementById('content-area');
//     if (!container) {
//         console.error("🚨 Erreur : L'élément #content-area est introuvable dans le DOM.");
//         return;
//     }

//     try {
//         // Nettoyage et affichage d'un état de chargement
//         container.innerHTML = '<div class="p-10 text-slate-500 italic">Chargement du module...</div>';

//         // 2. Enregistrement du remote
//         // remoteName doit correspondre au 'name' défini dans le MFE (ex: 'mfe_builder')
//         await __federation_method_setRemote(remoteName, `${remoteUrl}`);
        
//         // 3. Récupération du module
//         const module = await __federation_method_getRemote(remoteName, exposedModule);
        
//         // 4. Résolution du composant (Gestion des différents formats d'export)
//         const component = module.FormBuilder || module.default || module;

//         if (component && typeof component.mount === 'function') {
//             container.innerHTML = ''; // On vide le loader
            
//             // 5. Montage avec injection du contexte (role et knowledge)
//             await component.mount('content-area', context);
//             console.log(`✅ Module ${exposedModule} chargé avec succès.`);
//         } else {
//             throw new Error("Le module ne possède pas de méthode 'mount' valide.");
//         }
//     } catch (error) {
//         console.error("🚨 Erreur lors du chargement dynamique :", error);
//         container.innerHTML = `
//             <div class="p-10 text-red-600 bg-red-50 rounded-xl border border-red-200">
//                 <p class="font-bold">Erreur de chargement</p>
//                 <p class="text-xs mt-2">${error.message}</p>
//             </div>`;
//     }
// }

/**
 * Charge un module MFE de manière totalement dynamique via ESM.
 * Cette version contourne les limitations de window[remoteName] et gère le cache.
 */
async function mountRemoteFeature(remoteUrl, remoteName, exposedModule, context) {
    const container = document.getElementById('content-area');
    if (!container) {
        console.error("🚨 Élément #content-area introuvable dans le DOM.");
        return;
    }

    try {
        console.log(`📡 Initialisation ESM : ${remoteName} depuis ${remoteUrl}`);
        
        // 1. Importation dynamique native
        // Le timestamp (?t=...) force le navigateur à ignorer le cache du remoteEntry.js
        const remoteEntry = await import(/* @vite-ignore */ `${remoteUrl}?t=${Date.now()}`);

        // 2. Initialisation du Scope Partagé
        // Indispensable pour que le MFE puisse utiliser les bibliothèques du Shell (Vue, etc.)
        if (remoteEntry.init) {
            await remoteEntry.init(__federation_shared__);
        } else {
            console.warn(`⚠️ Le module ${remoteName} ne possède pas de méthode init.`);
        }

        // 3. Récupération de la factory du module exposé (ex: "./FormBuilder")
        const factory = await remoteEntry.get(exposedModule);
        const moduleContent = await factory();

        // 4. Debug & Résolution du composant
        console.log("📦 Contenu du module résolu :", moduleContent);

        // On cherche le point d'entrée 'mount' de manière flexible :
        // - Soit dans un export nommé (moduleContent.FormBuilder)
        // - Soit dans l'export par défaut (moduleContent.default)
        // - Soit le module lui-même
        const component = moduleContent.FormBuilder || 
                          moduleContent.default || 
                          moduleContent;

        // 5. Validation et Montage
        if (component && typeof component.mount === 'function') {
            // Nettoyage de la zone d'affichage
            container.innerHTML = ''; 
            
            console.log(`🚀 Montage de ${exposedModule} avec le contexte :`, context);
            
            // Appel de la méthode mount du MFE
            await component.mount('content-area', context);
            
            console.log(`✅ ${remoteName} monté avec succès sur #content-area`);
        } else {
            throw new Error(`Le module ${exposedModule} n'expose pas de méthode mount() valide.`);
        }

    } catch (error) {
        console.error("🚨 Détails de l'erreur de chargement :", error);
        
        // Affichage de l'erreur à l'utilisateur dans l'interface
        container.innerHTML = `
            <div class="p-6 bg-red-50 border-l-4 border-red-500 text-red-700 shadow-sm">
                <div class="flex items-center mb-2">
                    <span class="font-bold uppercase text-xs tracking-wider">Erreur de service : ${remoteName}</span>
                </div>
                <p class="text-sm font-medium">${error.message}</p>
                <div class="mt-4 text-[10px] font-mono bg-white/50 p-2 rounded">
                    Module recherché : ${exposedModule}<br>
                    Source : ${remoteUrl}
                </div>
            </div>`;
    }
}

function loadFeature(remoteUrl, remoteName, exposedModule, context) {
    const context_new = {
            role: 'admin',
            knowledge: 'expert', // Ce paramètre sera vérifié par le FormBuilder,
            lang: keycloak.tokenParsed?.lan || "fr"
        }; //cnt:  context 

    // Paramètres dynamiques
    const url = `/${remoteUrl}/ui/remoteEntry.js`; // L'URL complète du remoteEntry.js du MFE
    const name = remoteUrl; // Le nom interne du MFE
    const module = "./FormBuilder"; // Le chemin exposé

    mountRemoteFeature(url, name, module, context_new);
}


function buildMenu(navItems, serviceName, level = 0) {
    return navItems.map(item => {
        const hasChildren = item.children && item.children.length > 0;
        // const paddingLeft = level === 0 ? 'px-4' : 'pl-6 pr-4';
        const paddingLeft = level === 0 ? 'px-2' : `pl-${2 + (level * 2)}`;
        const aHeaderClass = level === 0?'mainmenu-button':'';
        const iconHtml = getIconSvg(item.icon);

        // NOUVEAU: Si l'élément a des enfants, il s'ouvre. Sinon (c'est une page), il lance loadFeature()
        // item.path est récupéré depuis Consul/Discovery (ex: "FormComposer")
        const clickAction = hasChildren 
            ? `handleMenuClick(this, event)` 
            : `loadFeature('${serviceName}','${item.role}', '${item.knb}', '${item.label.replace(/'/g, "\\'")}'); event.stopPropagation();`;

        // CHAQUE ITEM DOIT ÊTRE DANS CETTE DIV
        return `
            <div class="menu-container w-full" data-label="${item.label}">
                <button class="${aHeaderClass}" onclick="${clickAction}" 
                        class="flex items-center w-full ${paddingLeft} py-2 text-sm ...">
                    <span class="flex items-center gap-3">
                        ${iconHtml}
                        <span class="sidebar-label">${item.label}</span>
                    </span>
                    ${hasChildren ? `
                        <svg class="chevron ml-auto w-3.5 h-3.5 transition-transform duration-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path d="M9 5l7 7-7 7" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                            <path d="M9 5l7 7-7 7" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>` : ''
                        
                    }
                </button>
                ${hasChildren ? `
                    <div class="submenu-container hidden bg-slate-50/30">
                        ${buildMenu(item.children, serviceName, level + 1)}
                    </div>` : ''}
            </div>`;
    }).join('');
}

function handleMenuClick(element, event) {
    const sidebar = document.getElementById('sidebar');
    const container = element.parentElement; // Le .menu-container actuel
    const submenu = element.nextElementSibling;
    const isInsideFlyout = element.closest('.submenu-container');

    // 1. ARRÊTER LA PROPAGATION (Vital pour les menus imbriqués)
    event.stopPropagation();

    // 2. LOGIQUE D'ACCORDÉON UNIQUE (Niveau par Niveau)
    // On remonte au parent qui contient TOUTE la liste actuelle (nav ou submenu-container)
    const listContainer = container.parentElement; 
    
    // On cherche tous les .menu-container qui sont au même niveau de profondeur
    // On utilise Array.from pour ne filtrer que les enfants directs
    const siblingMenus = Array.from(listContainer.children).filter(child => 
        child.classList.contains('menu-container') && child !== container
    );

    siblingMenus.forEach(sibling => {
        // A. Désactiver l'état Flyout (si mode mini)
        sibling.classList.remove('flyout-active');
        
        // B. Cacher le sous-menu
        const siblingSub = sibling.querySelector('.submenu-container');
        if (siblingSub) {
            siblingSub.classList.add('hidden');
            // Nettoyage des styles de positionnement (au cas où)
            siblingSub.style.position = '';
            siblingSub.style.top = '';
        }

        // C. Remettre le chevron à sa position initiale
        const siblingChevron = sibling.querySelector('.chevron');
        if (siblingChevron) siblingChevron.classList.remove('rotate-90');
    });

    // 3. OUVERTURE DU MENU CLIQUÉ
    if (sidebar.classList.contains('mini') && !isInsideFlyout) {
        // MODE MINI : Flyout (Position fixe)
        if (submenu) {
            const rect = element.getBoundingClientRect();
            submenu.style.position = 'fixed';
            submenu.style.top = `${rect.top}px`;
            submenu.style.left = `${sidebar.offsetWidth}px`;
            container.classList.toggle('flyout-active');
        }
    } else {
        // MODE NORMAL OU ACCORDÉON DANS FLYOUT
        if (submenu) {
            // Nettoyage de sécurité des styles inline
            submenu.style.position = '';
            submenu.style.top = '';
            
            const isClosing = !submenu.classList.contains('hidden');
            submenu.classList.toggle('hidden');
            
            const chevron = element.querySelector('.chevron');
            if (chevron) {
                isClosing ? chevron.classList.remove('rotate-90') : chevron.classList.add('rotate-90');
            }
        }
    }
}
window.handleMenuClick = handleMenuClick;
window.loadFeature=loadFeature;

bootstrap();