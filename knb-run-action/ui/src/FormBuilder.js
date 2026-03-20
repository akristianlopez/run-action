/**
 * FormBuilder - Composant de création de formulaires et pages web
 * Extrait de demo3.html pour intégration Shell MFE
 */
import Swal from 'sweetalert2';
// Définir un style commun pour votre application
const AppAlert = Swal.mixin({
    customClass: {
        popup: 'rounded-[2rem] shadow-2xl font-sans border border-slate-100',
        confirmButton: 'rounded-xl px-6 py-3 text-xs uppercase tracking-widest font-bold ml-2',
        cancelButton: 'rounded-xl px-6 py-3 text-xs uppercase tracking-widest font-bold'
    },
    buttonsStyling: true // Permet d'utiliser les couleurs définies ci-dessous
});

export const FormBuilder = {
    state: {
        elements: [], // Arbre des composants
        history: [], // Pile pour stocker les états précédents
        selectedId: null,
        context: { role: null, knowledge: null, lang: 'fr' },
        draggedType: null,
        leftCollapsed: false,
        rightCollapsed: false,
        activeTab: 'inputs',
        isPreview: false,
        inspectorTab: 'general',
    },
    saveHistory() {
        // On garde une copie profonde (Deep Copy) pour éviter les problèmes de référence
        const snapshot = JSON.parse(JSON.stringify(this.state.elements));
        this.state.history.push(snapshot);

        // Optionnel : Limiter l'historique aux 20 dernières actions pour la mémoire
        if (this.state.history.length > 20) {
            this.state.history.shift();
        }
    },
    undo() {
        if (this.state.history.length > 0) {
            // On récupère le dernier état sauvegardé
            const previousState = this.state.history.pop();
            this.state.elements = previousState;
            
            // On désélectionne pour éviter les erreurs sur un ID qui n'existe plus
            this.state.selectedId = null;
            
            console.log("⏪ Undo effectué");
            this.render();
        } else {
            // Petit toast SweetAlert2 pour informer l'utilisateur
            Swal.fire({
                toast: true,
                position: 'bottom-end',
                title: 'Rien à annuler',
                icon: 'info',
                showConfirmButton: false,
                timer: 1500
            });
        }
    },

    async mount(containerId, context) {
        console.log("📥 FormBuilder reçu du Shell :", context);

        // 1. On initialise l'état avec des valeurs par défaut au cas où
        this.state.context = {
            role: context.role || 'Utilisateur',
            knowledge: context.knowledge || 'Inconnu',
            lang: context.lang || 'fr'
        };

        // 2. Vérification de sécurité
        if (this.state.context.knowledge !== 'expert') {
            const host = document.getElementById(containerId);
            if(host) host.innerHTML = `<div class="p-10 text-red-500">Niveau expert requis.</div>`;
            return;
        }

        // 3. IMPORTANT : On lie l'instance à window pour les événements HTML
        window.FormBuilder = this;

        // 4. On monte l'interface
        const host = document.getElementById(containerId);
        if (host) {
            // On génère le layout
            host.innerHTML = this.renderLayout();
            // On lance le rendu des éléments vides au départ
            this.render();
        }
        window.addEventListener('keydown', (e) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 'z') {
                e.preventDefault();
                this.undo();
            }
        }); 
    },

    renderLayout() {
        const { role, lang, knowledge } = this.state.context;
        const isSaveDisabled = !this.state.isDataFound ? 'opacity-50 cursor-not-allowed' : 'hover:bg-black';
        const leftWidth = this.state.leftCollapsed ? 'w-20' : 'w-72';
        const rightWidth = this.state.rightCollapsed ? 'w-12' : 'w-80';
        const isPreview = this.state.isPreview;
        const btnPreviewLabel = isPreview ? '📝 Mode Édition' : '👁️ Aperçu';
        const btnPreviewClass = isPreview ? 'bg-amber-500 hover:bg-amber-600' : 'bg-slate-800 hover:bg-black';

        return `
            <style>
                #fb-canvas-root .group { opacity: 1 !important; visibility: visible !important; }
                #fb-canvas-root input, #fb-canvas-root button { display: block !important; opacity: 1 !important; }
            </style>
        
            <div class="flex h-full bg-slate-50 font-sans overflow-hidden border-t">
                <aside class="${leftWidth} bg-white border-r border-slate-200 flex flex-col shadow-sm z-20 transition-all duration-300 relative">
                    <button onclick="window.FormBuilder.toggleLeft()" class="absolute -right-3 top-7 bg-white border border-slate-200 rounded-full p-1 hover:bg-indigo-50 z-30 shadow-sm">
                        ${this.state.leftCollapsed ? '➡️' : '⬅️'}
                    </button>
                                        
                    <div class="p-4 border-b h-12 bg-slate-50/50 flex justify-between items-center ${this.state.leftCollapsed ? 'hidden' : ''}">
                        <span class="font-black text-[10px] uppercase text-slate-500 leading-none">Outils : ${role} (${lang})</span>
                    </div>

                    <div class="flex border-b text-[10px] font-bold bg-slate-50/30">
                        <button onclick="window.FormBuilder.switchTab('inputs')" class="flex-1 py-3 border-b-2 transition-all ${this.state.activeTab === 'inputs' ? 'border-indigo-500 text-indigo-600 bg-white' : 'border-transparent text-slate-400'}">
                            ${this.state.leftCollapsed ? '📝' : 'Saisie'}
                        </button>
                        <button onclick="window.FormBuilder.switchTab('content')" class="flex-1 py-3 border-b-2 transition-all ${this.state.activeTab === 'content' ? 'border-emerald-500 text-emerald-600 bg-white' : 'border-transparent text-slate-400'}">
                            ${this.state.leftCollapsed ? '🖱️' : 'Contenu'}
                        </button>
                        <button onclick="window.FormBuilder.switchTab('struct')" class="flex-1 py-3 border-b-2 transition-all ${this.state.activeTab === 'struct' ? 'border-amber-500 text-amber-600 bg-white' : 'border-transparent text-slate-400'}">
                            ${this.state.leftCollapsed ? '📦' : 'Structure'}
                        </button>
                    </div>


                    <div class="flex-1 overflow-y-auto">
                        ${this.renderDraggableList()}
                    </div>

                </aside>

                <main class="flex-1 flex flex-col min-w-0 overflow-hidden">
                    <div class="h-12 bg-white border-b flex items-center px-6 z-10 shrink-0 gap-4">
                        
                        <div class="flex-1 max-w-md relative flex items-center">
                            <span class="absolute left-3 text-slate-400 text-xs">🔍</span>
                            <input type="text" 
                                placeholder="Rechercher par knowledge (ex: ${knowledge})..." 
                                id="fb-search-input"
                                class="w-full pl-9 pr-10 py-1.5 bg-slate-100 border-transparent rounded-lg text-xs focus:bg-white focus:ring-2 focus:ring-indigo-500 transition-all"
                                onkeyup="if(event.key === 'Enter') window.FormBuilder.fetchConfig()">
                            <button onclick="window.FormBuilder.fetchConfig()" class="absolute right-2 text-[10px] bg-white px-2 py-0.5 rounded border shadow-sm hover:text-indigo-600">
                                ${this.state.searchLoading ? '⏳' : 'Entrée'}
                            </button>
                        </div>

                        <div class="h-6 w-px bg-slate-200 mx-2"></div>

                        <div class="flex items-center gap-2">
                            <button id="btn-undo" 
                                    onclick="window.FormBuilder.undo()" 
                                    disabled
                                    class="flex items-center gap-2 px-3 py-1.5 border border-slate-200 text-slate-600 rounded-lg text-[10px] font-bold opacity-50 cursor-not-allowed transition-all">
                                <span>↩️</span> ANNULER
                            </button>

                            <div class="h-6 w-px bg-slate-200 mx-1"></div>
                            <button onclick="window.FormBuilder.clearCanvas()" 
                                    class="flex items-center gap-2 px-3 py-1.5 border border-red-200 text-red-600 rounded-lg text-[10px] font-bold hover:bg-red-50 transition-all">
                                <span>🗑️</span> VIDER
                            </button>

                            <div class="h-6 w-px bg-slate-200 mx-1"></div>
                        
                            <button id="btn-save-config" 
                                    onclick="window.FormBuilder.saveConfig()"
                                    ${!this.state.isDataFound ? 'disabled' : ''}
                                    class="flex items-center gap-2 px-4 py-1.5 bg-indigo-600 text-white rounded-lg text-[10px] font-bold shadow-sm transition-all ${isSaveDisabled}">
                                <span>💾</span> ENREGISTRER
                            </button>
                            
                            <button onclick="window.FormBuilder.togglePreview()" 
                                    class="px-4 py-1.5 ${isPreview ? 'bg-amber-500' : 'bg-slate-800'} text-white rounded-lg text-[10px] font-bold shadow-sm hover:opacity-90 transition-all">
                                ${isPreview ? '📝 ÉDITION' : '👁️ APERÇU'}
                            </button>
                        </div>
                    </div>

                <div class="flex-1 overflow-y-auto p-8 ${isPreview ? 'bg-white' : 'canvas-grid'}">
                    <div class="max-w-5xl mx-auto">
                        <div id="fb-canvas-root" 
                            class="${isPreview ? '' : 'drop-zone border-2 border-transparent shadow-2xl'} bg-white min-h-[85vh] w-full p-12 rounded-[2.5rem] transition-all"
                            ondragover="window.FormBuilder.hDragOver(event)" 
                            ondrop="window.FormBuilder.hDrop(event, 'root')">
                        </div>
                    </div>
                </div>                    

                </main>

                <aside class="${rightWidth} bg-white border-l border-slate-200 shadow-sm transition-all duration-300 relative flex flex-col shrink-0">
                    <button onclick="window.FormBuilder.toggleRight()" class="absolute -left-3 top-7 bg-white border border-slate-200 rounded-full p-1 hover:bg-indigo-50 z-30 shadow-sm">
                        ${this.state.rightCollapsed ? '⬅️' : '➡️'}
                    </button>

                    <div class="h-full overflow-y-auto ${this.state.rightCollapsed ? 'hidden' : 'block'}">
                        <div id="fb-inspector-content" class="p-6">
                            <div class="text-center text-slate-400 mt-20 italic text-xs">Sélectionnez un élément</div>
                        </div>
                    </div>
                    
                    <div class="${this.state.rightCollapsed ? 'flex' : 'hidden'} flex-col items-center pt-20 space-y-4 text-slate-300">
                        <span class="text-xl">⚙️</span>
                        <span class="rotate-90 origin-center whitespace-nowrap font-bold text-[10px] uppercase tracking-widest">Propriétés</span>
                    </div>
                </aside>
            </div>
        `;
    },

    clearCanvas() {
        AppAlert.fire({
            title: 'Vider le projet ?',
            text: "Tous les composants présents sur le canevas seront définitivement supprimés.",
            icon: 'warning',
            showCancelButton: true,
            confirmButtonColor: '#4f46e5', // Indigo
            cancelButtonColor: '#ef4444',  // Red
            confirmButtonText: 'Oui, tout effacer',
            cancelButtonText: 'Annuler',
        }).then((result) => {
            if (result.isConfirmed) {
                this.state.elements = [];
                this.state.selectedId = null;
                
                this.render();
                this.updateInspector();

                // Petit feedback de confirmation
                AppAlert.fire({
                    title: 'Nettoyé !',
                    icon: 'success',
                    timer: 1000,
                    showConfirmButton: false
                });
            }
        });
    },

    async fetchConfig() {
        const input = document.getElementById('fb-search-input');
        const query = input ? input.value : '';
        if (!query) return;

        this.state.searchLoading = true;
        this.refreshUI(); // Pour afficher le loader

        try {
            const { role, knowledge, lang } = this.state.context;
            
            // Construction de l'URL avec les paramètres requis
            const params = new URLSearchParams({
                q: query,
                role: role,
                knowledge: knowledge,
                lang: lang
            });

            const response = await fetch(`/organization/run?${params.toString()}`);
            
            if (response.ok) {
                const data = await response.json();
                
                // Si on trouve une donnée, on active le bouton
                this.state.isDataFound = true;
                console.log("✅ Données récupérées :", data);
                
                // Optionnel : On peut charger automatiquement les éléments si data.elements existe
                if (data.elements) {
                    this.state.elements = data.elements;
                }
            } else {
                this.state.isDataFound = false;
                console.warn("❓ Aucune donnée correspondante");
            }
        } catch (error) {
            console.error("🚨 Erreur Fetch :", error);
            this.state.isDataFound = false;
        } finally {
            this.state.searchLoading = false;
            this.refreshUI();
        }
    },

    async saveConfig() {
        if (!this.state.isDataFound) return;

        try {
            const response = await fetch('/organization/run', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    elements: this.state.elements,
                    context: this.state.context
                })
            });

            if (response.ok) {
                alert("Configuration enregistrée avec succès !");
            }
        } catch (error) {
            alert("Erreur lors de l'enregistrement.");
        }
    },

    renderDraggableList() {
        const tab = this.state.activeTab;
        const items = {
            inputs: [
                {t:'text', i:'📝', l:'Texte'}, {t:'number', i:'🔢', l:'Nombre'},
                {t:'date', i:'📅', l:'Date'}, {t:'select', i:'🔽', l:'Liste'},
                {t:'email', i:'📧', l:'Email'},{t:'password', i:'🔑', l:'Mot de passe'},
                {t:'textarea', i:'📋', l:'Zone de texte'},{t:'checkbox', i:'✅', l:'Case à cocher'},
                {t:'radio', i:'🔘', l:'Bouton Radio'},{t:'range', i:'🎚️', l:'Curseur / Range'},
                {t: 'maskedit', i: '🎭', l: 'Champ Masqué' }
            ],
            content: [
                {t:'button', i:'🖱️', l:'Bouton'}, {t:'title', i:'🅰️', l:'Titre'},
                {t:'image', i:'🖼️', l:'Image'}, {t:'action-button', i:'🖱️', l:'Bouton Action'},
                {t:'submit', i:'🚀', l:'Bouton Envoyer'}, {t:'paragraph', i:'📄', l:'Paragraphe RTF'},
                {t:'icon', i:'✨', l:'Icône Lucide'}, {t:'divider', i:'➖', l:'Séparateur'},
                {t:'link', i:'🔗', l:'Lien hypertexte'}
            ],
            struct: [
                {t:'container', i:'📦', l:'Card'}, {t:'tabs',i:'🗂️',l:'tabs'},
                {t:'wizard',i:'🪄',l:'wazard'},{t:'accordion',i:'🪗',l:'accordion'},
                {t:'grid', i:'🏁', l:'Grille'},
                {t:'section', i:'📑', l:'Section / Div'}, {t:'chart', i:'📊', l:'Graphique / Data'},
                {t:'code', i:'💻', l:'Bloc Code (Ace)'}, {t:'table', i:'📅', l:'Tableau de données' },
                {t:'spacer', i:'↔️', l:'Espace vide'}
            ]
        };

        return items[tab].map(item => `
            <div draggable="true" ondragstart="window.FormBuilder.hDragStart(event, '${item.t}')" 
                class="flex items-center ${this.state.leftCollapsed ? 'justify-center' : 'gap-3 p-3'} bg-white border border-slate-100 rounded-xl cursor-move hover:border-indigo-400 hover:shadow-md transition-all group"
                title="${item.l}">
                <div class="text-lg">${item.i}</div>
                <span class="font-bold text-slate-700 text-xs ${this.state.leftCollapsed ? 'hidden' : 'block'}">${item.l}</span>
            </div>
        `).join('');
    },

    switchTab(tabName) {
        this.state.activeTab = tabName;
        const host = document.getElementById('content-area'); // ou l'ID de votre container principal
        if (host) {
            host.innerHTML = this.renderLayout();
            this.render(); // Re-afficher le canvas
        }
    }, 

    togglePreview() {
        this.state.isPreview = !this.state.isPreview;
        // On ferme les volets pour un aperçu plein écran
        if (this.state.isPreview) {
            this.state.leftCollapsed = true;
            this.state.rightCollapsed = true;
        } else {
            this.state.leftCollapsed = false;
            this.state.rightCollapsed = false;
        }
        
        // Rafraîchir l'interface
        const host = document.getElementById('content-area');
        if (host) {
            host.innerHTML = this.renderLayout();
            this.render();
        }
    },

    toggleLeft() {
        this.state.leftCollapsed = !this.state.leftCollapsed;
        this.refreshUI();
    },

    toggleRight() {
        this.state.rightCollapsed = !this.state.rightCollapsed;
        this.refreshUI();
    },

    switchTab(tab) {
        this.state.activeTab = tab;
        this.refreshUI();
    },
    moveElement(id, direction) {
        const moveInArray = (arr) => {
            const index = arr.findIndex(el => el.id === id);
            if (index !== -1) {
                const newIndex = index + direction;
                // Vérification des limites
                if (newIndex >= 0 && newIndex < arr.length) {
                    // Échange (Swap)
                    [arr[index], arr[newIndex]] = [arr[newIndex], arr[index]];
                    return true;
                }
            }
            // Si non trouvé au premier niveau, chercher dans les enfants (récursivité)
            for (const item of arr) {
                if (item.children && moveInArray(item.children)) return true;
            }
            return false;
        };

        if (moveInArray(this.state.elements)) {
            console.log(`↕️ Élément déplacé vers ${direction > 0 ? 'le bas' : 'le haut'}`);
            this.render();
        }
    },

    refreshUI() {
        const host = document.getElementById('content-area');
        if (host) {
            host.innerHTML = this.renderLayout();
            this.render(); // Redessine le contenu du canvas
        }
    },

    createDraggable(type, icon, label) {
        return `
            <div draggable="true" 
                ondragstart="window.FormBuilder.hDragStart(event, '${type}')" 
                class="flex items-center gap-3 p-3 bg-white border border-slate-200 rounded-xl text-xs cursor-move hover:border-indigo-400 hover:shadow-md transition-all group">
                <div class="w-8 h-8 flex items-center justify-center bg-slate-50 rounded-lg group-hover:bg-indigo-50 text-lg">
                    ${icon}
                </div>
                <span class="font-bold text-slate-700">${label}</span>
            </div>
        `;
    },

    hDragOver(e) {
        e.preventDefault();
        e.stopPropagation();
        const target = e.target.closest('.drop-zone');
        if (target) target.classList.add('border-indigo-500', 'bg-indigo-50/50');
    },

    hDragLeave(e) {
        const target = e.target.closest('.drop-zone');
        if (target) target.classList.remove('border-indigo-500', 'bg-indigo-50/50');
    },

    hDragStart(e, type) {
        this.state.draggedType = type;
        // On stocke le type dans le dataTransfer pour le récupérer au drop
        e.dataTransfer.setData('application/json', JSON.stringify({ type }));
        e.dataTransfer.effectAllowed = 'move';
    },

    createNewElement(type) {
        const id = 'el_' + Math.random().toString(36).substr(2, 9);
        const base = { id, type, props:this.getDefaultProps(type)  }; //{ label: `Nouveau ${type}` }
        // const base =this.getDefaultProps(type)
        if (type === 'tabs') {
            return {
                ...base,
                props: { activeTab: 0 },
                zones: [
                    { id: id + '_z0', label: 'Onglet 1', children: [] },
                    { id: id + '_z1', label: 'Onglet 2', children: [] }
                ]
            };
        }
        
        if (['container', 'grid', 'accordion'].includes(type)) {
            return { ...base, children: [] };
        }

        return base;
    },


    hDrop(e, targetId, zoneId = null) {
        // 1. ARRÊTER LA PROPAGATION IMMÉDIATEMENT
        e.preventDefault();
        e.stopPropagation(); 

        this.saveHistory();
        
        // Récupération sécurisée du type
        let type;
        try {
            const data = JSON.parse(e.dataTransfer.getData('application/json'));
            type = data.type;
        } catch (err) {
            type = e.dataTransfer.getData('type'); 
        }

        if (!type) return;

        const newElement = this.createNewElement(type);

        // 2. LOGIQUE D'INSERTION
        const addToTarget = (elements) => {
            for (let el of elements) {
                // Cas 1 : Cible trouvée (Conteneur simple ou Racine d'un complexe)
                if (el.id === targetId && !zoneId) {
                    if (!el.children) el.children = [];
                    el.children.push(newElement);
                    return true;
                }
                // Cas 2 : Cible trouvée (Zone spécifique dans Tabs/Wizard)
                if (el.id === targetId && zoneId && el.zones) {
                    const zone = el.zones.find(z => z.id === zoneId);
                    if (zone) {
                        if (!zone.children) zone.children = [];
                        zone.children.push(newElement);
                        return true;
                    }
                }
                
                // 3. RÉCURSIVITÉ (On continue de chercher si pas trouvé)
                if (el.children && addToTarget(el.children)) return true;
                if (el.zones) {
                    for (let z of el.zones) {
                        if (z.children && addToTarget(z.children)) return true;
                    }
                }
            }
            return false;
        };

        if (targetId === 'root') {
            this.state.elements.push(newElement);
        } else {
            addToTarget(this.state.elements);
        }

        this.render();
    },

    findElementById(elements, id) {
        for (const el of elements) {
            if (el.id === id) return el;
            if (el.children) {
                const found = this.findElementById(el.children, id);
                if (found) return found;
            }
            if (el.zones) {
                for (const zone of el.zones) {
                    const found = this.findElementById(zone.children, id);
                    if (found) return found;
                }
            }
        }
        return null;
    },

    render() {
        const canvas = document.getElementById('fb-canvas-root');
        if (canvas) {
            // --- MISE À JOUR DYNAMIQUE DU BOUTON UNDO ---
            const btnUndo = document.getElementById('btn-undo'); // Assurez-vous d'ajouter cet ID au bouton
            if (btnUndo) {
                const hasHistory = this.state.history.length > 0;
                btnUndo.disabled = !hasHistory;
                
                // On gère aussi l'aspect visuel (opacité/curseur)
                if (hasHistory) {
                    btnUndo.classList.remove('opacity-50', 'cursor-not-allowed');
                    btnUndo.classList.add('hover:bg-slate-50', 'cursor-pointer');
                } else {
                    btnUndo.classList.add('opacity-50', 'cursor-not-allowed');
                    btnUndo.classList.remove('hover:bg-slate-50', 'cursor-pointer');
                }
            }
            console.log("🎨 Rendu effectué, historique :", this.state.history.length);

            // IMPORTANT : Utilisez innerHTML et non innerText
            const html = this.renderElements(this.state.elements);
            canvas.innerHTML = html; 
            
                console.log("DOM injecté dans #fb-canvas-root");
        } else {
            // Si vous voyez ce message en console, c'est que l'ID dans renderLayout 
            // ne correspond pas à celui ici.
            console.error("Cible de rendu introuvable !");
        }
        // if (canvas) canvas.innerHTML = this.renderElements(this.state.elements);
    },

    // renderElements(elements) {
    //     if (!elements || !Array.isArray(elements)) return '';

    //     const self = this;
    //     const isPreview = self.state.isPreview;

    //     return elements.map((el, index) => {
    //         if (!el) return '';
            
    //         let innerContent = '';
    //         const forceVisible = `display: block !important; opacity: 1 !important; visibility: visible !important;`;
    //         const inputStyle = `display: block !important; width: 100%; min-height: 38px; border: 1px solid #cbd5e1; background-color: white !important; color: #0f172a !important;`;

    //         const isFirst = index === 0;
    //         const isLast = index === elements.length - 1;

    //         // --- BARRE D'OUTILS (HAUT / BAS) ---
    //         const moveButtons = isPreview ? '' : `
    //             <div class="absolute -top-3 left-4 flex flex-row items-center gap-2 opacity-0 group-hover:opacity-100 transition-all duration-200 z-30" 
    //                 style="display: flex !important; flex-direction: row !important; background: transparent !important;">
                    
    //                 <button onclick="event.stopPropagation(); window.FormBuilder.moveElement('${el.id}', -1)"
    //                         ${isFirst ? 'style="display:none;"' : ''} 
    //                         class="flex items-center justify-center transition-transform hover:scale-125"
    //                         style="width:18px; height:18px; background: transparent !important; border: none !important; cursor: pointer; color: #10b981 !important; font-size: 12px; font-weight: bold;">
    //                     ▲
    //                 </button>

    //                 <button onclick="event.stopPropagation(); window.FormBuilder.moveElement('${el.id}', 1)"
    //                         ${isLast ? 'style="display:none;"' : ''} 
    //                         class="flex items-center justify-center transition-transform hover:scale-125"
    //                         style="width:18px; height:18px; background: transparent !important; border: none !important; cursor: pointer; color: #10b981 !important; font-size: 12px; font-weight: bold;">
    //                     ▼
    //                 </button>
    //             </div>
    //         `;
                    
    //         // --- BOUTON FERMETURE (Aligné sur la même ligne) ---
    //         const deleteButton = isPreview ? '' : `
    //             <button class="absolute -top-2.5 -right-2 bg-red-500 text-white shadow-sm z-30 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-700" 
    //                     style="width:18px; height:18px; border-radius:4px; font-size:8px; cursor: pointer;"
    //                     onclick="event.stopPropagation(); window.FormBuilder.deleteElement('${el.id}')">
    //                 ✕
    //             </button>
    //         `;
    //             // 1. STRUCTURE (Containers & Grids)
    //         if (el.type === 'container' || el.type === 'grid' || el.type === 'section') {
    //             const containerClass = isPreview 
    //                 ? 'p-4 rounded-xl' 
    //                 : 'p-4 border-2 border-dashed border-slate-300 bg-slate-50/30 rounded-xl drop-zone min-h-[100px]';
                
    //             const childrenHtml = (el.children && el.children.length > 0) 
    //                 ? self.renderElements(el.children) 
    //                 : (isPreview ? '' : '<div class="text-[10px] text-slate-300 italic text-center py-4">Zone de dépôt</div>');

    //             innerContent = `
    //                 <div class="${containerClass}" style="${forceVisible}"
    //                     ${isPreview ? '' : `ondragover="window.FormBuilder.hDragOver(event)" ondrop="window.FormBuilder.hDrop(event, '${el.id}')"`}>
    //                     ${childrenHtml}
    //                 </div>`;

    //         // 2. SAISIE TEXTE / EMAIL / NUMBER / PASSWORD
    //         } else if (['text', 'number', 'email', 'password', 'date'].includes(el.type)){
    //             innerContent = `
    //                 <div class="space-y-1 w-full" style="${forceVisible}">
    //                     <label class="block text-[10px] font-bold text-slate-500 uppercase tracking-tight">${el.props?.label || el.type} ${el.props?.required ? '<span class="text-red-500 ml-1">*</span>' : ''}</label>
    //                     <input type="${el.type}" placeholder="..." class="w-full p-2 border border-slate-300 rounded-lg bg-white" style="${inputStyle}" disabled>
    //                 </div>`;
    //         }  else if (el.type === 'button' || el.type === 'submit') {
    //             innerContent = `
    //                 <div style="${forceVisible}">
    //                     <button class="w-full py-2.5 bg-indigo-600 text-white rounded-xl font-bold shadow-md hover:bg-indigo-700"
    //                             style="display: block !important; width: 100%; background-color: #4f46e5 !important; color: white !important; border: none;">
    //                         ${el.props?.label || 'Action'}
    //                     </button>
    //                 </div>`;

    //         // 4. TITRES & TEXTES
    //         } else if (el.type === 'title') {
    //             innerContent = `<h2 class="font-bold text-slate-800" style="${forceVisible} font-size: 1.25rem;">${el.props?.label || 'Nouveau Titre'}</h2>`;
    //         } else if (el.type === 'paragraph') {
    //             innerContent = `<p class="text-sm text-slate-600" style="${forceVisible}">${el.props?.label || 'Texte du paragraphe...'}</p>`;

    //         // 5. SELECT / LISTES
    //         } else if (el.type === 'select') {
    //             innerContent = `
    //                 <div class="space-y-1 w-full" style="${forceVisible}">
    //                     <label class="block text-[10px] font-bold text-slate-500 uppercase tracking-tight">${el.props?.label || 'Liste'} ${el.props?.required ? '<span class="text-red-500 ml-1">*</span>' : ''}</label>
    //                     <select class="w-full p-2 border border-slate-300 rounded-lg bg-white" style="${inputStyle}" disabled>
    //                         <option>${el.props?.placeholder || 'Sélectionnez...'}</option>
    //                     </select>
    //                 </div>`;

    //         // 6. CHECKBOX / RADIO
    //         } else if (el.type === 'checkbox' || el.type === 'radio') {
    //             innerContent = `
    //                 <div class="flex items-center gap-3" style="${forceVisible}">
    //                     <input type="${el.type}" class="w-4 h-4 text-indigo-600 border-slate-300 rounded" style="opacity:1!important; visibility:visible!important;" disabled>
    //                     <label class="block text-[10px] font-bold text-slate-500 uppercase tracking-tight">${el.props?.label || 'Option'} ${el.props?.required ? '<span class="text-red-500 ml-1">*</span>' : ''}</label>
    //                 </div>`;

    //         // 7. CODE / SYSTÈME
    //         } else if (el.type === 'code') {
    //             innerContent = `
    //                 <div class="p-3 bg-slate-900 rounded-lg font-mono text-[10px] text-indigo-300 border border-slate-800" style="${forceVisible}">
    //                     // Bloc de code dynamique
    //                 </div>`;

    //         } else if (el.type === 'tabs') {
    //             const activeIdx = el.props?.activeTab || 0;
    //             const zones = el.zones || [{ id: 'z0', label: 'Tab 1', children: [] }];

    //             innerContent = `
    //                 <div class="border rounded-xl bg-white shadow-sm overflow-hidden" style="${forceVisible}">
    //                     <div class="flex bg-slate-50 border-b">
    //                         ${zones.map((z, i) => `
    //                             <button onclick="event.stopPropagation(); window.FormBuilder.setActiveTab('${el.id}', ${i})"
    //                                     class="px-4 py-2 text-[10px] font-bold border-r transition-colors ${i === activeIdx ? 'bg-white text-indigo-600 border-b-2 border-b-indigo-500' : 'text-slate-400'}">
    //                                 ${z.label}
    //                             </button>
    //                         `).join('')}
    //                     </div>
    //                     <div class="p-4 min-h-[120px] drop-zone"
    //                         ondragover="window.FormBuilder.hDragOver(event)"
    //                         ondrop="window.FormBuilder.hDrop(event, '${el.id}', '${zones[activeIdx].id}')">
    //                         ${self.renderElements(zones[activeIdx].children)}
    //                         ${zones[activeIdx].children.length === 0 ? '<div class="text-[10px] text-slate-300 italic text-center py-8">Déposez dans cet onglet</div>' : ''}
    //                     </div>
    //                 </div>`;

    //         // B. ACCORDION / COLLAPSE
    //         } else if (el.type === 'accordion' || el.type === 'collapse') {
    //             const label = el.props?.label || "Section dépliante";
    //             innerContent = `
    //                 <div class="border border-slate-200 rounded-lg overflow-hidden bg-white" style="${forceVisible}">
    //                     <div class="flex items-center justify-between px-4 py-3 bg-slate-50 border-b border-slate-200">
    //                         <span class="text-[10px] font-bold text-slate-700 uppercase">${label}</span>
    //                         <span class="text-slate-400">▼</span>
    //                     </div>
    //                     <div class="p-4 min-h-[80px] drop-zone"
    //                         ondragover="window.FormBuilder.hDragOver(event)" 
    //                         ondrop="window.FormBuilder.hDrop(event, '${el.id}')">
    //                         ${el.children?.length > 0 ? self.renderElements(el.children) : '<div class="text-[10px] text-slate-300 italic text-center py-2">Contenu accordéon</div>'}
    //                     </div>
    //                 </div>`;

    //         // C. WIZARD (ÉTAPE PAR ÉTAPE)
    //         } else if (el.type === 'wizard') {
    //             const currentStep = el.props?.currentStep || 1;
    //             innerContent = `
    //                 <div class="bg-white border border-slate-200 rounded-2xl p-6 shadow-sm" style="${forceVisible}">
    //                     <div class="flex items-center justify-center gap-4 mb-8">
    //                         <div class="flex items-center gap-2">
    //                             <span class="w-6 h-6 rounded-full bg-indigo-600 text-white text-[10px] flex items-center justify-center font-bold">1</span>
    //                             <div class="h-1 w-12 bg-slate-100 rounded"></div>
    //                             <span class="w-6 h-6 rounded-full bg-slate-100 text-slate-400 text-[10px] flex items-center justify-center">2</span>
    //                         </div>
    //                     </div>
    //                     <div class="border-2 border-dashed border-indigo-100 rounded-xl p-4 min-h-[120px] drop-zone"
    //                         ondragover="window.FormBuilder.hDragOver(event)" 
    //                         ondrop="window.FormBuilder.hDrop(event, '${el.id}')">
    //                         ${el.children?.length > 0 ? self.renderElements(el.children) : '<div class="text-[10px] text-indigo-300 italic text-center py-6 font-medium">Contenu de l\'étape ${currentStep}</div>'}
    //                     </div>
    //                     <div class="flex justify-between mt-6">
    //                         <button class="px-4 py-2 rounded-lg bg-slate-100 text-slate-400 text-[10px] font-bold" disabled>PRÉCÉDENT</button>
    //                         <button class="px-4 py-2 rounded-lg bg-indigo-600 text-white text-[10px] font-bold shadow-md">SUIVANT</button>
    //                     </div>
    //                 </div>`;
    //         } else if (el.type === 'maskedit') {
    //             innerContent = `
    //                 <div class="space-y-1 w-full" style="${forceVisible}">
    //                     <label class="block text-[10px] font-bold text-slate-500 uppercase tracking-tight">${el.props?.label || 'Champ Masqué'}</label>
    //                     <div class="relative">
    //                         <input type="text" placeholder="${el.props?.mask || el.props?.placeholder}" 
    //                             class="w-full p-2 border border-slate-300 rounded-lg bg-white italic text-slate-400" 
    //                             style="${inputStyle}" disabled>
    //                         <span class="absolute right-2 top-2 opacity-30 text-xs">🎭</span>
    //                     </div>
    //                 </div>`;
    //         } else  {
    //             innerContent = `<div class="p-2 bg-amber-50 text-amber-600 text-[10px] border border-amber-200 rounded" style="${forceVisible}">Composant: ${el.type}</div>`;
    //         }

    //         const isSelected = self.state.selectedId === el.id;
    //         const wrapperClass = isPreview 
    //             ? 'mb-4' 
    //             : `relative group mb-6 p-2 border-2 rounded-xl transition-all cursor-pointer 
    //             ${isSelected ? 'border-indigo-500 bg-indigo-50/50' : 'border-slate-200 border-dashed hover:border-indigo-300'}`;

    //         return `
    //             <div class="${wrapperClass}" style="${forceVisible} position: relative;" 
    //                 ${isPreview ? '' : `onclick="event.stopPropagation(); window.FormBuilder.select('${el.id}')"`}>
    //                 ${deleteButton}
    //                 ${moveButtons}
    //                 ${innerContent}
    //             </div>`;
    //     }).join('');
    // },
renderElements(elements) {
    if (!elements || !Array.isArray(elements)) return '';

    const self = this;
    const isPreview = self.state.isPreview;
    let onClickAttr = '';

    // Définition des variantes de style pour les boutons
    const STYLE_VARIANTS = {
        primaire: { bg: '#4f46e5', text: 'white' },
        succès:   { bg: '#10b981', text: 'white' },
        danger:   { bg: '#ef4444', text: 'white' },
        warning:  { bg: '#f59e0b', text: 'white' }, // Orange ambré
        info:     { bg: '#0ea5e9', text: 'white' }, // Bleu ciel
        light:    { bg: '#f8fafc', text: '#475569', border: '#e2e8f0' }, // Gris très clair
        dark:     { bg: '#1e293b', text: 'white' }, // Ardoise foncée
        link:     { bg: 'transparent', text: '#4f46e5', decoration: 'underline' } // Style lien
    };
    return elements.map((el, index) => {
        if (!el) return '';
        
        let innerContent = '';
        const forceVisible = `display: block !important; opacity: 1 !important; visibility: visible !important;`;
        
        // Styles dynamiques issus des props
        const alignment = el.props?.align || 'left';
        const color = el.props?.color || 'inherit';
        const isRequired = el.props?.required;
        // Correspondance des classes Flexbox / Text-align
        const justifyClass = alignment === 'center' ? 'justify-center' : alignment === 'right' ? 'justify-end' : 'justify-start';
        const textAlign = `text-align: ${alignment};`;

        const inputStyle = `display: block !important; width: 100%; min-height: 38px; border: 1px solid #cbd5e1; background-color: white !important; color: #0f172a !important; text-align: ${alignment};`;

        const isFirst = index === 0;
        const isLast = index === elements.length - 1;

        // --- BARRE D'OUTILS ---
        const moveButtons = isPreview ? '' : `
            <div class="absolute -top-3 left-4 flex flex-row items-center gap-2 opacity-0 group-hover:opacity-100 transition-all duration-200 z-30" 
                style="display: flex !important; flex-direction: row !important; background: transparent !important;">
                <button onclick="event.stopPropagation(); window.FormBuilder.moveElement('${el.id}', -1)"
                        ${isFirst ? 'style="display:none;"' : ''} 
                        class="flex items-center justify-center transition-transform hover:scale-125"
                        style="width:18px; height:18px; background: transparent !important; border: none !important; cursor: pointer; color: #10b981 !important; font-size: 12px; font-weight: bold;">▲</button>
                <button onclick="event.stopPropagation(); window.FormBuilder.moveElement('${el.id}', 1)"
                        ${isLast ? 'style="display:none;"' : ''} 
                        class="flex items-center justify-center transition-transform hover:scale-125"
                        style="width:18px; height:18px; background: transparent !important; border: none !important; cursor: pointer; color: #10b981 !important; font-size: 12px; font-weight: bold;">▼</button>
            </div>`;

        const deleteButton = isPreview ? '' : `
            <button class="absolute -top-2.5 -right-2 bg-red-500 text-white shadow-sm z-30 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-700" 
                    style="width:18px; height:18px; border-radius:4px; font-size:8px; cursor: pointer;"
                    onclick="event.stopPropagation(); window.FormBuilder.deleteElement('${el.id}')">✕</button>`;

        // --- LOGIQUE DE RENDU PAR TYPE ---

        // 1. STRUCTURE (Containers)
        if (el.type === 'container' || el.type === 'grid' || el.type === 'section') {
            const containerClass = isPreview ? 'p-4 rounded-xl' : 'p-4 border-2 border-dashed border-slate-300 bg-slate-50/30 rounded-xl drop-zone min-h-[100px]';
            const childrenHtml = (el.children && el.children.length > 0) ? self.renderElements(el.children) : (isPreview ? '' : '<div class="text-[10px] text-slate-300 italic text-center py-4">Zone de dépôt</div>');
            
            innerContent = `
                <div class="${containerClass}" style="${forceVisible} border-style: ${el.props?.showBorder === false ? 'none' : 'dashed'};"
                    ${isPreview ? '' : `ondragover="window.FormBuilder.hDragOver(event)" ondrop="window.FormBuilder.hDrop(event, '${el.id}')"`}>
                    ${el.props?.title ? `<h4 class="text-xs font-bold mb-2 uppercase text-slate-400">${el.props.title}</h4>` : ''}
                    ${childrenHtml}
                </div>`;

        // 2. SAISIE CLASSIQUE
        } else if (['text', 'number', 'email', 'password', 'date','textarea'].includes(el.type)){
            const minL = el.props?.minlength || 0;
            const maxL = el.props?.maxlength || '';
            
            innerContent = `
                <div class="space-y-1 w-full" style="${forceVisible}">
                    <div class="flex justify-between items-end">
                        <label class="block text-[10px] font-bold text-slate-500 uppercase tracking-tight">
                            ${el.props?.label || el.type} 
                            ${el.props?.required ? '<span class="text-red-500 ml-1">*</span>' : ''}
                        </label>
                        ${maxL ? `<span class="text-[8px] text-slate-300 font-mono">max: ${maxL}</span>` : ''}
                    </div>
                    <input type="${el.type}" 
                        placeholder="${el.props?.placeholder || ''}" 
                        minlength="${minL}"
                        maxlength="${maxL}"
                        class="w-full p-2 border border-slate-300 rounded-lg bg-white" 
                        style="${inputStyle}" 
                        disabled>
                </div>`;
        // 3. BOUTONS
        } else if (el.type === 'button' || el.type === 'submit' || el.type === 'action-button') {
            const variantKey = el.props?.variant || 'primaire';
            const theme = STYLE_VARIANTS[variantKey] || STYLE_VARIANTS.primaire;
            
            // Calcul dynamique du style inline
            const shadow = variantKey === 'link' ? 'none' : '0 4px 6px -1px rgb(0 0 0 / 0.1)';
            const border = theme.border ? `1px solid ${theme.border}` : 'none';
            const decoration = theme.decoration ? `text-decoration: ${theme.decoration};` : '';
 
            innerContent = `
                <div style="${forceVisible}">
                    <button onclick="${onClickAttr}"
                            class="w-full py-2.5 rounded-xl font-bold shadow-md transition-all active:scale-95"
                            style="background-color: ${theme.bg} !important; 
                            color: ${theme.text} !important; border: none; cursor: pointer;">
                        ${el.props?.label || 'Bouton'}
                    </button>
                </div>`;

            innerContent = `
                <div style="${forceVisible}">
                    <button class="w-full py-2.5 rounded-xl font-bold transition-all hover:opacity-90 active:scale-95"
                            style="display: block !important; width: 100%; 
                                background-color: ${theme.bg} !important; 
                                color: ${theme.text} !important; 
                                border: ${border}; 
                                box-shadow: ${shadow};
                                ${decoration}
                                text-align: center; cursor: pointer;">
                        ${el.props?.label || 'Action'}
                    </button>
                </div>`;
        // 4. CONTENU TEXTUEL
        } else if (el.type === 'title') {
            innerContent = `<h2 class="font-bold" style="${forceVisible} font-size: 1.25rem; text-align: ${alignment}; color: ${color};">${el.props?.label || 'Nouveau Titre'}</h2>`;
        } else if (el.type === 'paragraph') {
            innerContent = `<p class="text-sm" style="${forceVisible} text-align: ${alignment}; color: ${color};">${el.props?.label || el.props?.content || 'Texte du paragraphe...'}</p>`;

        // 5. LISTES
        } else if (el.type === 'select') {
            innerContent = `
                <div class="space-y-1 w-full" style="${forceVisible}">
                    <label class="block text-[10px] font-bold text-slate-500 uppercase tracking-tight">${el.props?.label || 'Liste'} ${isRequired ? '<span class="text-red-500 ml-1">*</span>' : ''}</label>
                    <select class="w-full p-2 border border-slate-300 rounded-lg bg-white" style="${inputStyle}" disabled>
                        <option>${el.props?.placeholder || 'Sélectionnez...'}</option>
                    </select>
                </div>`;

        // 6. COMPLEXES (Tabs, Accordion, Wizard)
        } else if (el.type === 'tabs') {
            const activeIdx = el.props?.activeTab || 0;
            const zones = el.zones || [{ id: 'z0', label: 'Tab 1', children: [] }];
            innerContent = `
                <div class="border rounded-xl bg-white shadow-sm overflow-hidden" style="${forceVisible}">
                    <div class="flex bg-slate-50 border-b">
                        ${zones.map((z, i) => `<button onclick="event.stopPropagation(); window.FormBuilder.setActiveTab('${el.id}', ${i})" class="px-4 py-2 text-[10px] font-bold border-r transition-colors ${i === activeIdx ? 'bg-white text-indigo-600 border-b-2 border-b-indigo-500' : 'text-slate-400'}">${z.label}</button>`).join('')}
                    </div>
                    <div class="p-4 min-h-[120px] drop-zone" ondragover="window.FormBuilder.hDragOver(event)" ondrop="window.FormBuilder.hDrop(event, '${el.id}', '${zones[activeIdx].id}')">
                        ${self.renderElements(zones[activeIdx].children)}
                    </div>
                </div>`;

        } else if (el.type === 'accordion') {
            innerContent = `
                <div class="border border-slate-200 rounded-lg overflow-hidden bg-white" style="${forceVisible}">
                    <div class="flex items-center justify-between px-4 py-3 bg-slate-50 border-b border-slate-200">
                        <span class="text-[10px] font-bold text-slate-700 uppercase" style="color: ${color}">${el.props?.label || "Section"}</span>
                        <span class="text-slate-400">${el.props?.isOpen ? '▲' : '▼'}</span>
                    </div>
                    <div class="p-4 min-h-[80px] drop-zone" ondragover="window.FormBuilder.hDragOver(event)" ondrop="window.FormBuilder.hDrop(event, '${el.id}')">
                        ${self.renderElements(el.children || [])}
                    </div>
                </div>`;

        } else if (el.type === 'maskedit') {
            const minL = el.props?.minlength || 0;
            const maxL = el.props?.maxlength || '';

            innerContent = `
                <div class="space-y-1 w-full" style="${forceVisible}">
                    <label class="block text-[10px] font-bold text-slate-500 uppercase tracking-tight">${el.props?.label || 'Masque'} ${isRequired ? '<span class="text-red-500 ml-1">*</span>' : ''}</label>
                    <div class="relative">
                        <input type="text" placeholder="${el.props?.mask || el.props?.placeholder}" 
                        minlength="${minL}"
                        maxlength="${maxL}"
                        
                        class="w-full p-2 border border-slate-300 rounded-lg bg-white italic text-slate-400" style="${inputStyle}" disabled>
                        <span class="absolute right-2 top-2 opacity-30 text-xs">🎭</span>
                    </div>
                </div>`;
        } else {
            innerContent = `<div class="p-2 bg-amber-50 text-amber-600 text-[10px] border border-amber-200 rounded" style="${forceVisible}">Composant: ${el.type}</div>`;
        }

        const isSelected = self.state.selectedId === el.id;
        const wrapperClass = isPreview ? 'mb-4' : `relative group mb-6 p-2 border-2 rounded-xl transition-all cursor-pointer ${isSelected ? 'border-indigo-500 bg-indigo-50/50' : 'border-slate-200 border-dashed hover:border-indigo-300'}`;

        return `<div class="${wrapperClass}" style="${forceVisible} position: relative;" ${isPreview ? '' : `onclick="event.stopPropagation(); window.FormBuilder.select('${el.id}')"`}>${deleteButton}${moveButtons}${innerContent}</div>`;
    }).join('');
},
    

    getDefaultProps(type) {
        const base = { 
            label: 'Nouveau champ', 
            name: 'field_' + Math.random().toString(36).substr(2, 5), // ID technique pour l'export
            required: false 
        };

        switch (type) {
            // --- INPUTS ---
            case 'text':
            case 'email':
            case 'password':
            case 'date':
                return { ...base, placeholder: 'Saisir ici...', defaultValue: '',required: false,regex: '', errorMessage: '',minlength: 0,maxlength: 255  };
            
            case 'number':
                return { ...base, placeholder: '0', min: 0, max: 100, step: 1,required: false,regex: '', errorMessage: '',minlength: 0,maxlength: 255 };

            case 'textarea':
                return { ...base, placeholder: 'Saisir le texte...', rows: 3,required: false,regex: '', errorMessage: '',minlength: 0,maxlength: 255 };
            case 'maskedit':
                return { 
                    ...base, 
                    placeholder: '(__) __ __ __ __', 
                    mask: '(99) 99 99 99 99', // Format standard (9 = chiffre, A = lettre)
                    defaultValue: '',required: false,regex: '', errorMessage: '' ,minlength: 0,maxlength: 255
                };
            case 'select':
            case 'radio':
            case 'checkbox':
                return { ...base, options: [{label: 'Option 1', value: 'opt1'}, {label: 'Option 2', value: 'opt2'}] };
            case 'range':
                return { ...base, min: 0, max: 100, step: 1, defaultValue: 50 };

            // --- CONTENT ---
            case 'title':
                return { label: 'Titre de section', level: 'h2', align: 'left', color: '#0f172a',minlength: 0,maxlength: 255 };

            case 'paragraph':
                return { content: 'Éditez ce texte pour décrire votre formulaire...', align: 'left',minlength: 0,maxlength: 255 };

            case 'button':
            case 'submit':
            case 'action-button':
                return { label: 'Action', variant: 'primary', size: 'md', 
                        actionType: 'submit',
                        label: 'Action', 
                        variant: 'primaire', 
                        actionConfig: {
                            script: '',
                            url: '',
                            target: '_self'
                        }
                     };

            case 'image':
                return { src: 'https://via.placeholder.com/150', alt: 'Image', width: '100%', height: 'auto' };

            case 'divider':
                return { spacing: 'my-4', borderStyle: 'solid' };

            // --- STRUCTURE ---
            case 'container':
                return { title: 'Titre de la carte', showBorder: true, padding: 'p-4', shadow: 'shadow-sm' };

            case 'tabs':
                return { activeTab: 0, variant: 'line' };

            case 'wizard':
                return { currentStep: 0, linear: true, finishLabel: 'Terminer' };

            case 'accordion':
                return { label: 'Section pliable', isOpen: true };

            case 'grid':
                return { columns: 2, gap: 4 };

            case 'table':
                return { headers: ['Nom', 'Email', 'Rôle'], pageSize: 5 };

            default:
                return { ...base };
        }
    }, 

    template() {
        return `
            <div class="fb-container flex h-screen bg-slate-50">
                <aside class="w-64 border-r bg-white">...</aside>
                <main id="fb-canvas" class="flex-1 p-10 canvas-grid">...</main>
            </div>
        `;
    },

    renderToHTML(elements) {
        return elements.map(el => {
            const isSelected = this.state.selectedId === el.id;
            let content = '';

            switch (el.type) {
                case 'text':
                    content = `<div class="space-y-1">
                        <label class="block text-sm font-medium text-gray-700">${el.props.label}</label>
                        <input type="text" placeholder="${el.props.placeholder}" class="w-full p-2 border rounded-md bg-gray-50">
                    </div>`;
                    break;
                case 'button':
                    content = `<button class="px-4 py-2 bg-indigo-600 text-white rounded-md shadow-sm w-full font-bold">
                        ${el.props.label}
                    </button>`;
                    break;
                case 'card':
                    content = `<div class="p-4 border rounded-xl shadow-sm bg-white">
                        <h4 class="font-bold border-b pb-2 mb-2">${el.props.title}</h4>
                        <p class="text-sm text-gray-600">${el.props.content}</p>
                    </div>`;
                    break;
                // Ajouter les autres types ici...
            }

            return `
                <div onclick="window.FormBuilder.selectElement('${el.id}')" 
                     class="group relative p-2 mb-2 cursor-pointer transition-all ${isSelected ? 'ring-2 ring-indigo-500 bg-indigo-50' : 'hover:border-indigo-200 border-2 border-transparent border-dashed'}">
                    ${content}
                    ${isSelected ? `<button onclick="window.FormBuilder.deleteElement('${el.id}')" class="absolute -top-2 -right-2 bg-red-500 text-white rounded-full p-1 opacity-0 group-hover:opacity-100 transition-opacity">✕</button>` : ''}
                </div>`;
        }).join('');
    },

    handleDragStart(e, type) { e.dataTransfer.setData('type', type); },
    
    handleDrop(e) {
        e.preventDefault();
        const type = e.dataTransfer.getData('type');
        const newEl = {
            id: 'fb_' + Math.random().toString(36).substr(2, 9),
            type,
            props: this.getDefaultProps(type)
        };
        this.state.elements.push(newEl);
        this.selectElement(newEl.id);
    },

    selectElement(id) {
        this.state.selectedId = id;
        console.log("Élément sélectionné :", id);
        this.render();
        this.updateInspector();
    },
    select(id) {
        this.state.selectedId = id;
        console.log("Élément sélectionné :", id);
        this.render();
        this.updateInspector();
    },

    deleteElement(id) {
        this.saveHistory();

        const removeFromLevel = (elements) => {
            // 1. Filtrer le niveau actuel
            const initialLength = elements.length;
            const filtered = elements.filter(el => el.id !== id);
            
            // Si la taille a changé, on a trouvé et supprimé l'élément à ce niveau
            if (filtered.length !== initialLength) {
                return filtered;
            }

            // 2. Sinon, chercher récursivement dans les enfants ou les zones
            return elements.map(el => {
                // Chercher dans les conteneurs simples (children)
                if (el.children && el.children.length > 0) {
                    return { ...el, children: removeFromLevel(el.children) };
                }
                // Chercher dans les conteneurs complexes (zones pour Tabs/Wizard)
                if (el.zones && el.zones.length > 0) {
                    return {
                        ...el,
                        zones: el.zones.map(zone => ({
                            ...zone,
                            children: removeFromLevel(zone.children)
                        }))
                    };
                }
                return el;
            });
        };

        // Appliquer la fonction récursive à l'état global
        this.state.elements = removeFromLevel(this.state.elements);
        
        // Nettoyage de la sélection
        if (this.state.selectedId === id) {
            this.state.selectedId = null;
        }

        this.render();
        this.updateInspector();
    },

    // updateInspector() {
    //     const panel = document.getElementById('fb-inspector-content');
    //     const MASK_PRESETS = [
    //         { label: '📞 Mobile (FR)', mask: '06 99 99 99 99' },
    //         { label: '📮 Code Postal', mask: '99999' },
    //         { label: '💳 CB', mask: '9999 9999 9999 9999' },
    //         { label: '📅 Date', mask: '99/99/9999' },
    //         { label: '🏦 IBAN', mask: 'FR99 9999 9999 9999 9999 999' }
    //     ];

    //     if (!panel) return;

    //     if (!this.state.selectedId) {
    //         panel.innerHTML = `
    //             <div class="text-center text-slate-400 mt-20 italic text-xs px-10">
    //                 <div class="text-4xl mb-4 opacity-20">⚙️</div>
    //                 Sélectionnez un élément pour modifier ses propriétés
    //             </div>`;
    //         return;
    //     }

    //     const el = this.findElementById(this.state.elements, this.state.selectedId);
    //     if (!el) return;

    //     let propsHtml = '';
        
    //     // Génération dynamique des champs selon les props de l'élément
    //     Object.keys(el.props).forEach(key => {
    //         const val = el.props[key];
    //         const label = key.charAt(0).toUpperCase() + key.slice(1);
            
    //         let inputHtml = '';
            
    //         // Choisir le type d'input selon la valeur ou la clé
    //         if (typeof val === 'boolean') {
    //             inputHtml = `
    //                 <label class="relative inline-flex items-center cursor-pointer mt-1">
    //                     <input type="checkbox" ${val ? 'checked' : ''} onchange="window.FormBuilder.updateProp('${key}', this.checked)" class="sr-only peer">
    //                     <div class="w-9 h-5 bg-slate-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-600"></div>
    //                 </label>`;
    //         } else if (key === 'options') {
    //             // Pour les listes (select/radio), on gère une zone de texte simplifiée
    //             inputHtml = `<textarea oninput="window.FormBuilder.updateProp('${key}', this.value)" class="w-full p-2 border rounded mt-1 text-xs font-mono" rows="3">${val}</textarea>
    //                         <p class="text-[9px] text-slate-400 mt-1">Séparez les options par une virgule</p>`;
    //         } else if (key === 'color') {
    //             inputHtml = `<input type="color" value="${val}" oninput="window.FormBuilder.updateProp('${key}', this.value)" class="w-full h-8 p-1 border rounded mt-1 cursor-pointer">`;
    //         } else if (key === 'mask') {
    //             const presetsHtml = MASK_PRESETS.map(p => `
    //                     <button onclick="window.FormBuilder.updateProp('mask', '${p.mask}')" 
    //                             class="px-2 py-1 bg-slate-100 hover:bg-indigo-100 text-slate-600 hover:text-indigo-600 rounded text-[8px] font-bold transition-colors">
    //                         ${p.label}
    //                     </button>
    //                 `).join('');

    //             inputHtml = `
    //                 <div class="flex flex-col gap-2 mt-1">
    //                     <input type="text" value="${val}" oninput="window.FormBuilder.updateProp('${key}', this.value)" 
    //                         class="w-full p-2 border border-indigo-200 rounded text-xs font-mono bg-indigo-50/30 shadow-inner focus:ring-2 focus:ring-indigo-400 outline-none"
    //                         placeholder="ex: 99/99/9999">
                        
    //                     <div class="text-[9px] text-slate-400 font-semibold uppercase tracking-wider">Modèles rapides</div>
    //                     <div class="flex flex-wrap gap-1">
    //                         ${presetsHtml}
    //                     </div>
                        
    //                     <div class="flex gap-2 mt-2 p-2 bg-slate-50 rounded border border-slate-100">
    //                         <span class="text-[8px] text-slate-500"><strong>9</strong>: Chiffre | <strong>A</strong>: Lettre | <strong>*</strong>: Tous</span>
    //                     </div>
    //                 </div>`;
    //     }else {
    //             // Par défaut : Texte ou Nombre
    //             const inputType = typeof val === 'number' ? 'number' : 'text';
    //             inputHtml = `<input type="${inputType}" value="${val}" oninput="window.FormBuilder.updateProp('${key}', this.value)" class="w-full p-2 border border-slate-200 rounded mt-1 text-xs focus:ring-1 focus:ring-indigo-500 outline-none">`;
    //         }

    //         propsHtml += `
    //             <div class="pb-3 border-b border-slate-50">
    //                 <label class="text-[10px] font-bold text-slate-500 uppercase tracking-tight">${label}</label>
    //                 ${inputHtml}
    //             </div>`;
    //     });

    //     panel.innerHTML = `
    //         <div class="flex items-center justify-between mb-6">
    //             <h3 class="font-bold text-indigo-600 uppercase text-[11px] tracking-widest">Configuration</h3>
    //             <span class="px-2 py-1 bg-indigo-50 text-indigo-600 rounded text-[9px] font-bold">${el.type}</span>
    //         </div>
    //         <div class="space-y-4">
    //             ${propsHtml}
    //         </div>
    //     `;
    // },    

    updateInspector() {
        const panel = document.getElementById('fb-inspector-content');
        if (!panel || !this.state.selectedId) {
            this.renderEmptyInspector(panel);
            return;
        }

        const el = this.findElementById(this.state.elements, this.state.selectedId);
        if (!el) return;

        const currentTab = this.state.inspectorTab;

        // --- RENDER DES ONGLETS DE L'INSPECTEUR ---
        const tabsHtml = `
            <div class="flex border-b border-slate-200 mb-6 bg-slate-50/50 -mx-4 px-4">
                ${['general', 'style', 'validation'].map(t => `
                    <button onclick="window.FormBuilder.setInspectorTab('${t}')" 
                            class="flex-1 py-3 text-[10px] font-bold uppercase tracking-wider transition-all
                            ${currentTab === t ? 'text-indigo-600 border-b-2 border-indigo-600 bg-white' : 'text-slate-400 hover:text-slate-600'}">
                        ${t === 'general' ? '⚙️' : t === 'style' ? '🎨' : '🛡️'} ${t}
                    </button>
                `).join('')}
            </div>
        `;

        // --- FILTRAGE DES PROPRIÉTÉS PAR ONGLET ---
        let filteredProps = {};
        if (currentTab === 'general') {
            filteredProps = this.filterProps(el.props, ['label', 'placeholder', 'name', 'defaultValue', 'options', 'mask', 'actionType', 'script']);
        } else if (currentTab === 'style') {
            filteredProps = this.filterProps(el.props, ['color', 'align', 'size', 'variant', 'padding', 'shadow', 'border', 'rows']);
        } else if (currentTab === 'validation') {
            filteredProps = this.filterProps(el.props, ['required', 'min', 'max', 'step', 'errorMessage', 'regex','minlength','maxlength']);
        }

        panel.innerHTML = `
            <div class="flex items-center justify-between mb-4">
                <h3 class="font-bold text-slate-800 uppercase text-[11px]">${el.type} #${el.id.slice(-4)}</h3>
                <button onclick="window.FormBuilder.deleteElement('${el.id}')" class="text-red-400 hover:text-red-600 text-[10px]">Supprimer</button>
            </div>
            ${tabsHtml}
            <div class="space-y-4 animate-in fade-in duration-300">
                ${this.generateInspectorFields(el, filteredProps)}
            </div>
        `;
    },
    // Changer d'onglet dans l'inspecteur
    setInspectorTab(tab) {
        this.state.inspectorTab = tab;
        this.updateInspector();
    },

    // Filtrer les propriétés pour l'onglet actif
    filterProps(props, allowedKeys) {
        return Object.keys(props)
            .filter(key => allowedKeys.includes(key))
            .reduce((obj, key) => {
                obj[key] = props[key];
                return obj;
            }, {});
    },

    // Générer les inputs (ton ancienne logique d'updateInspector déplacée ici)
    generateInspectorFields(el, props) {
        const REGEX_PRESETS = [
            { label: '📧 Email', pattern: '^[\\w-\\.]+@([\\w-]+\\.)+[\\w-]{2,4}$' },
            { label: '🌐 URL', pattern: '^(https?:\\/\\/)?([\\da-z\\.-]+)\\.([a-z\\.]{2,6})([\\/\\w \\.-]*)*\\/?$' },
            { label: '📱 Mobile (FR)', pattern: '^(06|07)[0-9]{8}$' },
            { label: '🔢 Chiffres uniquement', pattern: '^[0-9]*$' },
            { label: '🔠 Lettres uniquement', pattern: '^[a-zA-Z\\s]*$' }
        ];
        const MASK_PRESETS = [
            { label: '📞 Mobile (FR)', mask: '06 99 99 99 99' },
            { label: '📮 Code Postal', mask: '99999' },
            { label: '💳 CB', mask: '9999 9999 9999 9999' },
            { label: '📅 Date', mask: '99/99/9999' },
            { label: '🏦 IBAN', mask: 'FR99 9999 9999 9999 9999 999' }
        ];
        const STYLE_VARIANTS = [
            { label: 'Primaire', variant: 'primaire', class: 'bg-indigo-600', text: 'text-white' },
            { label: 'Succès', variant: 'succès', class: 'bg-emerald-500', text: 'text-white' },
            { label: 'Danger', variant: 'danger', class: 'bg-rose-500', text: 'text-white' },
            { label: 'Warning', variant: 'warning', class: 'bg-amber-500', text: 'text-white' },
            { label: 'Info', variant: 'info', class: 'bg-sky-500', text: 'text-white' },
            { label: 'Light', variant: 'light', class: 'bg-slate-100', text: 'text-slate-600' },
            { label: 'Dark', variant: 'dark', class: 'bg-slate-900', text: 'text-white' },
            { label: 'Link', variant: 'link', class: 'bg-transparent', text: 'text-indigo-600' }
        ];    
        const STANDARD_ACTIONS = [
            { label: '🚀 Soumettre le formulaire', value: 'submit', hint: 'Valide et envoie les données.' },
            { label: '🔄 Réinitialiser', value: 'reset', hint: 'Efface tous les champs.' },
            { label: '⬅️ Page précédente', value: 'prev_step', hint: 'Pour les Wizards.' },
            { label: '➡️ Page suivante', value: 'next_step', hint: 'Pour les Wizards.' },
            { label: '💻 Script Personnalisé', value: 'custom', hint: 'Écrivez votre propre code JS.' }
        ];

        let propsHtml = '';
        Object.keys(props).forEach(key => {
                const val = props[key];
                const label = key.charAt(0).toUpperCase() + key.slice(1);
                
                let inputHtml = '';
                
                // Choisir le type d'input selon la valeur ou la clé
                if (typeof val === 'boolean') {
                    inputHtml = `
                        <label class="relative inline-flex items-center cursor-pointer mt-1">
                            <input type="checkbox" ${val ? 'checked' : ''} onchange="window.FormBuilder.updateProp('${key}', this.checked)" class="sr-only peer">
                            <div class="w-9 h-5 bg-slate-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-600"></div>
                        </label>`;
                } else if (key === 'options') {
                    // Pour les listes (select/radio), on gère une zone de texte simplifiée
                    inputHtml = `<textarea oninput="window.FormBuilder.updateProp('${key}', this.value)" class="w-full p-2 border rounded mt-1 text-xs font-mono" rows="3">${val}</textarea>
                                <p class="text-[9px] text-slate-400 mt-1">Séparez les options par une virgule</p>`;
                } else if (key === 'mask') {
                    const presetsHtml = MASK_PRESETS.map(p => `
                            <button onclick="window.FormBuilder.updateProp('mask', '${p.mask}')" 
                                    class="px-2 py-1 bg-slate-100 hover:bg-indigo-100 text-slate-600 hover:text-indigo-600 rounded text-[8px] font-bold transition-colors">
                                ${p.label}
                            </button>
                        `).join('');

                    inputHtml = `
                        <div class="flex flex-col gap-2 mt-1">
                            <input type="text" value="${val}" oninput="window.FormBuilder.updateProp('${key}', this.value)" 
                                class="w-full p-2 border border-indigo-200 rounded text-xs font-mono bg-indigo-50/30 shadow-inner focus:ring-2 focus:ring-indigo-400 outline-none"
                                placeholder="ex: 99/99/9999">
                            
                            <div class="text-[9px] text-slate-400 font-semibold uppercase tracking-wider">Modèles rapides</div>
                            <div class="flex flex-wrap gap-1">
                                ${presetsHtml}
                            </div>
                            
                            <div class="flex gap-2 mt-2 p-2 bg-slate-50 rounded border border-slate-100">
                                <span class="text-[8px] text-slate-500"><strong>9</strong>: Chiffre | <strong>A</strong>: Lettre | <strong>*</strong>: Tous</span>
                            </div>
                        </div>`;
                }else if (key === 'regex') {
                    const presetsHtml = REGEX_PRESETS.map(p => `
                        <button onclick="window.FormBuilder.updateProp('regex', '${p.pattern.replace(/\\/g, '\\\\')}')" 
                                class="px-2 py-1 bg-slate-100 hover:bg-emerald-100 text-slate-600 hover:text-emerald-700 rounded text-[8px] font-bold transition-colors">
                            ${p.label}
                        </button>
                    `).join('');

                    inputHtml = `
                        <div class="flex flex-col gap-2 mt-1">
                            <input type="text" value="${val || ''}" oninput="window.FormBuilder.updateProp('${key}', this.value)" 
                                class="w-full p-2 border border-emerald-200 rounded text-[10px] font-mono bg-emerald-50/20 outline-none focus:ring-1 focus:ring-emerald-400"
                                placeholder="^([a-z0-9]+)$">
                            
                            <div class="text-[9px] text-slate-400 font-bold uppercase tracking-wider">Modèles de validation</div>
                            <div class="flex flex-wrap gap-1">
                                ${presetsHtml}
                            </div>
                        </div>`;

                } else if (key === 'errorMessage') {
                    inputHtml = `
                        <input type="text" value="${val || ''}" oninput="window.FormBuilder.updateProp('${key}', this.value)" 
                            placeholder="Ex: Format d'email invalide"
                            class="w-full p-2 border border-slate-200 rounded mt-1 text-xs italic text-red-500 bg-red-50/20">`;
                }else if (key === 'align') {
                    const options = [
                        { v: 'left', i: 'Aligné à gauche', icon: 'format_align_left' },
                        { v: 'center', i: 'Centré', icon: 'format_align_center' },
                        { v: 'right', i: 'Aligné à droite', icon: 'format_align_right' }
                    ];
                    inputHtml = `
                        <div class="flex gap-1 mt-1 bg-slate-100 p-1 rounded-lg">
                            ${options.map(opt => `
                                <button onclick="window.FormBuilder.updateProp('align', '${opt.v}')" 
                                        class="flex-1 py-1 rounded ${val === opt.v ? 'bg-white shadow-sm text-indigo-600' : 'text-slate-400 hover:text-slate-600'}">
                                    ${opt.v === 'left' ? '⬅️' : opt.v === 'center' ? '↔️' : '➡️'}
                                </button>
                            `).join('')}
                        </div>`;

                } else if (key === 'variant') {

                    // Dans ton onglet 'style' de l'inspecteur :
                    inputHtml = `
                        <div class="grid grid-cols-2 gap-2 mt-1">
                            ${STYLE_VARIANTS.map(v => `
                                <button onclick="window.FormBuilder.updateProp('variant', '${v.variant}')" 
                                        class="px-2 py-2 rounded border text-[9px] font-bold uppercase tracking-tighter transition-all flex items-center
                                        ${val === v.variant ? 'border-indigo-600 ring-2 ring-indigo-100 bg-indigo-50/30' : 'border-slate-200 bg-white hover:bg-slate-50'}">
                                    <span class="inline-block w-2.5 h-2.5 rounded-sm ${v.class} mr-2 border border-black/5"></span>
                                    ${v.label}
                                </button>
                            `).join('')}
                        </div>`;                

                } else if (key === 'color') {
                    inputHtml = `
                        <div class="flex items-center gap-2 mt-1">
                            <input type="color" value="${val || '#475569'}" oninput="window.FormBuilder.updateProp('color', this.value)" 
                                class="w-10 h-8 border-none cursor-pointer bg-transparent">
                            <input type="text" value="${val || '#475569'}" oninput="window.FormBuilder.updateProp('color', this.value)" 
                                class="flex-1 p-1.5 border rounded text-[10px] font-mono outline-none">
                        </div>`;
                }else if (key === 'options') {
                    const options = val || [];
                    inputHtml = `
                        <div class="space-y-2 mt-2">
                            ${options.map((opt, i) => `
                                <div class="flex items-center gap-1">
                                    <input type="text" value="${opt.label}" 
                                        oninput="window.FormBuilder.updateOption(${i}, 'label', this.value)"
                                        class="flex-1 p-1.5 border rounded text-[10px]" placeholder="Label">
                                    <input type="text" value="${opt.value}" 
                                        oninput="window.FormBuilder.updateOption(${i}, 'value', this.value)"
                                        class="w-16 p-1.5 border rounded text-[10px] bg-slate-50 font-mono" placeholder="Val">
                                    <button onclick="window.FormBuilder.removeOption(${i})" class="text-red-400 hover:text-red-600 px-1">✕</button>
                                </div>
                            `).join('')}
                            <button onclick="window.FormBuilder.addOption()" 
                                    class="w-full py-1.5 mt-2 border-2 border-dashed border-slate-200 rounded text-[9px] font-bold text-slate-400 hover:border-indigo-300 hover:text-indigo-500 transition-all">
                                + AJOUTER UNE OPTION
                            </button>
                        </div>`;
                } else if (key === 'minlength' || key === 'maxlength') {
                    inputHtml = `
                        <div class="flex flex-col gap-1">
                            <label class="text-[9px] font-semibold text-slate-400 uppercase">${key === 'minlength' ? 'Longueur Min' : 'Longueur Max'}</label>
                            <input type="number" value="${val}" 
                                oninput="window.FormBuilder.updateProp('${key}', parseInt(this.value) || 0)" 
                                class="w-full p-2 border border-slate-200 rounded text-xs outline-none focus:border-indigo-400"
                                min="0">
                        </div>`;
                }else if (key === 'script' && el.props.actionType === 'custom') {
                    inputHtml = `
                        <div class="mt-4">
                            <label class="text-[9px] font-bold text-amber-600 uppercase">Script à stocker</label>
                            <textarea 
                                oninput="window.FormBuilder.updateNestedProp('actionConfig', 'script', this.value)" 
                                class="w-full p-3 bg-slate-900 text-emerald-400 font-mono text-[10px] rounded-lg border border-slate-700 mt-1"
                                rows="6" spellcheck="false" 
                                placeholder="// Le Renderer exécutera ce code">${el.props.actionConfig?.script || ''}</textarea>
                        </div>`;
                } else if (key === 'url' && el.props.actionType === 'link') {
                    inputHtml = `
                        <input type="text" 
                            value="${el.props.actionConfig?.url || ''}" 
                            oninput="window.FormBuilder.updateNestedProp('actionConfig', 'url', this.value)"
                            placeholder="https://..." 
                            class="w-full p-2 border border-slate-200 rounded mt-1 text-xs">`;
                }else if (key === 'actionType') {
                    inputHtml = `
                        <select onchange="window.FormBuilder.updateProp('actionType', this.value)" 
                                class="w-full p-2 border border-slate-200 rounded mt-1 text-xs bg-white">
                            ${STANDARD_ACTIONS.map(a => `
                                <option value="${a.value}" ${val === a.value ? 'selected' : ''}>${a.label}</option>
                            `).join('')}
                        </select>
                        <p class="text-[9px] text-slate-400 mt-1 italic">
                            ${STANDARD_ACTIONS.find(a => a.value === val)?.hint || ''}
                        </p>
                    `;
                }else {
                    // Par défaut : Texte ou Nombre
                    const inputType = typeof val === 'number' ? 'number' : 'text';
                    inputHtml = `<input type="${inputType}" value="${val}" oninput="window.FormBuilder.updateProp('${key}', this.value)" class="w-full p-2 border border-slate-200 rounded mt-1 text-xs focus:ring-1 focus:ring-indigo-500 outline-none">`;
                }

                propsHtml += `
                    <div class="pb-3 border-b border-slate-50">
                        <label class="text-[10px] font-bold text-slate-500 uppercase tracking-tight">${label}</label>
                        ${inputHtml}
                    </div>`;
            });
            return `
                <div class="flex items-center justify-between mb-6">
                    <h3 class="font-bold text-indigo-600 uppercase text-[11px] tracking-widest">Configuration</h3>
                    <span class="px-2 py-1 bg-indigo-50 text-indigo-600 rounded text-[9px] font-bold">${el.type}</span>
                </div>
                <div class="space-y-4">
                    ${propsHtml}
                </div>
            `
    },

    updateProp(key, value) {
        if (!this.state.selectedId) return;
        
        const el = this.findElementById(this.state.elements, this.state.selectedId);
        if (el) {
            // Optionnel : Sauvegarder dans l'historique pour Ctrl+Z
            // this.saveHistory(); 
            
            el.props[key] = value;
            
            // On rafraîchit le canvas pour voir le changement de label/placeholder
            this.render();
        }
    },
    addOption() {
        const el = this.findElementById(this.state.elements, this.state.selectedId);
        if (el && el.props.options) {
            this.saveHistory();
            el.props.options.push({ label: 'Nouvelle Option', value: 'val' });
            this.render();
            this.updateInspector();
        }
    },

    updateOption(index, key, value) {
        const el = this.findElementById(this.state.elements, this.state.selectedId);
        if (el && el.props.options[index]) {
            el.props.options[index][key] = value;
            this.render(); // Pour mettre à jour le select dans le canvas
        }
    },

    removeOption(index) {
        const el = this.findElementById(this.state.elements, this.state.selectedId);
        if (el && el.props.options.length > 1) {
            this.saveHistory();
            el.props.options.splice(index, 1);
            this.render();
            this.updateInspector();
        }
    },
    updateNestedProp(parentKey, childKey, value) {
        if (!this.state.selectedId) return;
        
        const el = this.findElementById(this.state.elements, this.state.selectedId);
        if (el) {
            if (!el.props[parentKey]) el.props[parentKey] = {};
            el.props[parentKey][childKey] = value;
            
            // Pas besoin de render() ici car cela n'affecte pas le visuel du bouton
            // mais on peut le faire pour forcer la sauvegarde de l'historique
        }
    }

};