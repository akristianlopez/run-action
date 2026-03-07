/**
 * ActModeler - Composant de conception d'actes administratifs
 * Intégrable dans un Shell MFE (Vite/Vue3/Vanilla)
 */

export const ActModeler = {
    // État interne du composant (sera synchronisé avec la DB)
    state: {
        elements: [],
        selectedId: null,
        userData: {}, // Données pour parseKeywords (ex: {NOM_AGENT: "Jean Dupont"})
        
        header: {
            republique: "RÉPUBLIQUE DU BÉNIN",
            ministere: "Ministère du Numérique et de la Digitalisation",
            logo: "", // URL ou base64
            devise: "Fraternité - Justice - Travail"
        },
        footer: {
            adresse: "Cotonou, Bénin",
            contact: "Tél: +229 00 00 00 00 | Email: contact@numerique.bj"
        },
        isPreview: false,
        isInspectorCollapsed: false,
        paperFormat: 'A4',
        formats: {
            'A3': { width: '297mm', height: '420mm', label: 'A3 (Grand Format)' },
            'A4': { width: '210mm', height: '297mm', label: 'A4 (Standard)' },
            'A5': { width: '148mm', height: '210mm', label: 'A5 (Note)' },
            'A6': { width: '105mm', height: '148mm', label: 'A6 (Fiche)' },
            'A7': { width: '74mm', height: '105mm', label: 'A7' },
            'A8': { width: '52mm', height: '74mm', label: 'A8' }
        }
    },

    updatePaperFormat(format) {
        this.state.paperFormat = format;
        const canvas = document.getElementById('drop-zone-root');
        if (canvas) {
            const dimensions = this.state.formats[format];
            canvas.style.width = dimensions.width;
            canvas.style.minHeight = dimensions.height;
        }
        this.render();
    },

    // --- 1. GESTION DES MOTS-CLÉS ---
    parseKeywords(text, data = {}) {
        if (typeof text !== 'string') return text;
        return text.replace(/\{(\w+)\}/g, (match, key) => {
            return data[key] !== undefined ? data[key] : match;
        });
    },

    // --- 2. USINE À ÉLÉMENTS (Factory) ---
    createTable(rows = 3, cols = 2) {
        return {
            id: 'el_' + Math.random().toString(36).substr(2, 9),
            type: 'table',
            props: {
                rows, cols,
                hasHeader: true,
                data: Array.from({ length: rows }, () => Array(cols).fill('Texte...'))
            }
        };
    },

    createChart(chartType = 'bar') {
        return {
            id: 'el_' + Math.random().toString(36).substr(2, 9),
            type: 'chart',
            props: {
                id: Math.random().toString(36).substr(2, 5),
                chartType: chartType,
                title: 'Statistiques',
                labels: ['Jan', 'Fév', 'Mar'],
                data: [12, 19, 3]
            }
        };
    },

    // --- 3. MOTEURS DE RENDU (Renderers) ---
    renderTable(props) {
        const { rows, cols, data, hasHeader } = props;
        let html = `<table class="w-full border-collapse border border-slate-400 my-4 text-sm">`;
        for (let i = 0; i < rows; i++) {
            html += `<tr>`;
            for (let j = 0; j < cols; j++) {
                const val = this.parseKeywords(data[i][j] || "", this.state.userData);
                if (i === 0 && hasHeader) {
                    html += `<th class="border border-slate-400 p-2 bg-slate-100 font-bold">${val}</th>`;
                } else {
                    html += `<td class="border border-slate-400 p-2">${val}</td>`;
                }
            }
            html += `</tr>`;
        }
        return html + `</table>`;
    },

    renderChart(props) {
        // On utilise un timeout pour laisser le DOM s'injecter avant d'appeler Chart.js
        setTimeout(() => this.initChart(props), 100);
        return `
            <div class="my-6 p-4 bg-white border rounded-lg">
                <canvas id="chart-${props.id}" style="max-height: 250px;"></canvas>
                <p class="text-[10px] text-center text-slate-400 mt-2 uppercase font-bold">${props.title}</p>
            </div>`;
    },

    initChart(p) {
        const ctx = document.getElementById(`chart-${p.id}`);
        if (!ctx || typeof Chart === 'undefined') return;
        new Chart(ctx, {
            type: p.chartType,
            data: {
                labels: p.labels,
                datasets: [{ label: p.title, data: p.data, backgroundColor: '#4f46e5' }]
            },
            options: { responsive: true, maintainAspectRatio: false }
        });
    },
    renderToHTML(elements, userData = {}) {
        this.state.userData = userData;
        
        // Rendu de l'en-tête global
        let fullHtml = this.renderHeader();

        // Rendu des éléments du corps
        fullHtml += elements.map(el => {
            const isSelected = this.state.selectedId === el.id;
            const p = el.props;
            let content = '';
            switch(el.type) {
                case 'acte_nature': 
                    content = `<div class="text-center font-black text-xl underline uppercase my-4">${p.documentType}</div>`; break;
                case 'acte_reference': 
                    content = `<div class="text-sm font-bold my-2 text-left text-slate-800">N° ${p.refNumber}</div>`; break;
                case 'article': 
                    content = `<div class="my-6"><h4 class="font-bold text-center underline mb-2 uppercase text-sm">${p.title}</h4><p class="text-justify text-sm">${this.parseKeywords(p.content, this.state.userData)}</p></div>`; break;
                case 'signataire': 
                    content = `<div class="mt-10 ml-auto w-1/2 text-center font-serif"><p class="font-bold underline uppercase text-xs mb-10">${p.role}${p.isInterim ? ' P.I.' : ''}</p><p class="font-black text-sm uppercase">${p.name}</p></div>`; break;
                case 'table': 
                    content = this.renderTable(p); break;
            }
            return `
                <div onclick="window.ActModeler.selectElement('${el.id}')" 
                    class="relative cursor-pointer transition-all mb-2 ${isSelected ? 'ring-2 ring-indigo-500 rounded-lg bg-indigo-50/30' : 'hover:bg-slate-50'}">
                    ${content}
                </div>`;
        }).join('');

        // Rendu du pied de page global
        fullHtml += this.renderFooter();

        return fullHtml;
    },
    // Spécifique aux modèles d'actes administratifs
    renderAdminActe(p) {
        const vus = (p.vus || []).map(v => `
            <li class="flex items-start mb-1 text-sm">
                <span class="font-bold mr-2 uppercase">Vu</span> 
                <span>${this.parseKeywords(v.text, this.state.userData)}</span>
            </li>`).join('');

        return `
            <div class="admin-acte-container p-8 bg-white shadow-sm font-serif">
                <div class="text-center font-bold mb-6 text-lg underline uppercase">${p.documentType || 'ARRETE'}</div>
                <div class="mb-6 text-sm italic">${this.parseKeywords(p.documentTitle, this.state.userData)}</div>
                <ul class="list-none pl-0 mb-4">${vus}</ul>
                <div class="text-center my-8">
                    <h3 class="font-black underline uppercase text-lg">${p.decideWord || 'DECIDE'} :</h3>
                </div>
            </div>`;
    },

    togglePreview() {
        this.state.isPreview = !this.state.isPreview;
        
        // Récupération des éléments du Shell pour masquage
        const sidebar = document.querySelector('aside'); 
        const toolBar = document.querySelector('aside.w-16'); 
        const inspector = document.getElementById('inspector-panel');
        const header = document.querySelector('header');

        if (this.state.isPreview) {
            // MODE APERÇU
            if (sidebar) sidebar.style.display = 'none';
            if (toolBar) toolBar.style.display = 'none';
            if (inspector) inspector.style.display = 'none';
            if (header) header.style.display = 'none';
            
            this.state.selectedId = null; // Désactive les bordures bleues
            this.injectCloseButton(); // On injecte le bouton pour revenir
        } else {
            // MODE ÉDITION
            if (sidebar) sidebar.style.display = 'flex';
            if (toolBar) toolBar.style.display = 'flex';
            if (inspector) inspector.style.display = 'block';
            if (header) header.style.display = 'flex';

            const closeBtn = document.getElementById('btn-close-preview');
            if (closeBtn) closeBtn.remove();
        }

        this.render();
        // On ne reconstruit l'inspecteur que si on sort de l'aperçu
        if (!this.state.isPreview) this.updateInspector();
    },

    // printPDF() {
    //     this.state.isPreview = !this.state.isPreview;
        
    //     // 1. Récupération des éléments du Shell
    //     const sidebar = document.querySelector('aside'); 
    //     const toolBar = document.querySelector('aside.w-16'); 
    //     const inspector = document.getElementById('inspector-panel');
    //     const header = document.querySelector('header');

    //     if (this.state.isPreview) {
    //         // MODE APERÇU : On cache tout
    //         if (sidebar) sidebar.style.display = 'none';
    //         if (toolBar) toolBar.style.display = 'none';
    //         if (inspector) inspector.style.display = 'none';
    //         if (header) header.style.display = 'none';
            
    //         this.state.selectedId = null;
    //         this.injectCloseButton();
    //     } else {
    //         // MODE ÉDITION : On réaffiche tout
    //         if (sidebar) sidebar.style.display = 'flex';
    //         if (toolBar) toolBar.style.display = 'flex';
    //         if (inspector) inspector.style.display = 'block';
    //         if (header) header.style.display = 'flex';

    //         // Suppression du bouton flottant
    //         const closeBtn = document.getElementById('btn-close-preview');
    //         if (closeBtn) closeBtn.remove();
            
    //         // On force la reconstruction de l'inspecteur pour retrouver nos boutons
    //         this.updateInspector();
    //     }

    //     // 2. Mise à jour du canvas
    //     this.render();   
    // },


    injectCloseButton() {
        // Éviter les doublons
        if (document.getElementById('btn-close-preview')) return;

        const closeBtn = document.createElement('button');
        closeBtn.id = 'btn-close-preview';
        closeBtn.type = 'button'; // Indispensable pour éviter tout comportement par défaut
        
        closeBtn.innerHTML = `
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
            <span>Quitter l'aperçu</span>
        `;
        
        // Style flottant très visible (Rouge pour indiquer une sortie)
        closeBtn.className = `
            fixed top-6 right-6 z-[9999] 
            flex items-center gap-2 
            bg-red-600 text-white px-6 py-3 
            rounded-full shadow-2xl hover:bg-red-700 
            transition-all duration-300 font-bold text-sm
        `;

        // Gestion du clic avec blocage des événements parents
        closeBtn.onclick = (e) => {
            if (e) {
                e.preventDefault();
                e.stopPropagation(); // BLOQUE le déclenchement d'autres fonctions (comme l'impression)
            }
            this.togglePreview();
        };

        document.body.appendChild(closeBtn);
    },

    mount(container) {
        if (!container) return;
        window.ActModeler = this;
        // À ajouter au début de mount(container)
        const style = document.createElement('style');
        style.innerHTML = `
            #inspector-content::-webkit-scrollbar {
                width: 6px;
            }
            #inspector-content::-webkit-scrollbar-thumb {
                background-color: #e2e8f0;
                border-radius: 10px;
            }
            #inspector-content::-webkit-scrollbar-track {
                background: transparent;
            }
        `;
        document.head.appendChild(style);        
        // 1. Définition de la structure HTML (Layout 3 colonnes)
        container.innerHTML = `
        <div class="flex h-full w-full bg-slate-100 overflow-hidden font-sans">
            <aside class="w-16 flex flex-col items-center py-4 bg-[#1e293b] text-white gap-4 shadow-xl z-10">
                <div class="p-2 bg-indigo-600 rounded-lg mb-4">
                     <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path d="M12 4v16m8-8H4" stroke-width="2" stroke-linecap="round"/></svg>
                </div>
                <button draggable="true" ondragstart="ActModeler.handleDragStart(event, 'admin_acte')" class="p-3 hover:bg-slate-700 rounded-xl transition-all group relative" title="Acte Administratif">
                    <svg class="w-6 h-6 text-slate-400 group-hover:text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" stroke-width="2"/></svg>
                </button>
                <button draggable="true" ondragstart="ActModeler.handleDragStart(event, 'table')" class="p-3 hover:bg-slate-700 rounded-xl transition-all group relative" title="Tableau">
                    <svg class="w-6 h-6 text-slate-400 group-hover:text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path d="M3 10h18M3 14h18m-9-4v8m-7 0h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" stroke-width="2"/></svg>
                </button>
                <button draggable="true" ondragstart="ActModeler.handleDragStart(event, 'chart')" class="p-3 hover:bg-slate-700 rounded-xl transition-all group relative" title="Graphique">
                    <svg class="w-6 h-6 text-slate-400 group-hover:text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" stroke-width="2"/></svg>
                </button>
                <button draggable="true" ondragstart="window.ActModeler.handleDragStart(event, 'acte_nature')" class="p-3 hover:bg-slate-700 rounded-xl" title="Nature">📄</button>
                <button draggable="true" ondragstart="window.ActModeler.handleDragStart(event, 'reglementaire')" class="p-3 hover:bg-slate-700 rounded-xl" title="Visas">⚖️</button>
                <button draggable="true" ondragstart="window.ActModeler.handleDragStart(event, 'article')" class="p-3 hover:bg-slate-700 rounded-xl" title="Article">§</button>
                <button draggable="true" ondragstart="window.ActModeler.handleDragStart(event, 'signataire')" class="p-3 hover:bg-slate-700 rounded-xl" title="Signataire">✍️</button>                
            </aside>

            <main class="flex-1 overflow-y-auto p-8 canvas-grid flex justify-center bg-slate-200/50">
                <div id="drop-zone-root" 
                     class="w-[210mm] min-h-[297mm] bg-white shadow-2xl p-12 transition-all drop-zone"
                     ondragover="event.preventDefault()"
                     ondrop="ActModeler.handleDrop(event, 'root')">
                     </div>
            </main>
            <div class="flex h-full bg-slate-100 overflow-hidden font-sans text-slate-900">
                <div class="relative flex h-full">
                    <button onclick="window.ActModeler.toggleInspector()" 
                            id="toggle-inspector-btn"
                            class="absolute -left-[10px] top-5 z-[100] bg-white border border-slate-200 rounded-full p-1.5 shadow-xl hover:bg-indigo-50 text-slate-600 transition-all">
                        <svg id="collapse-icon" class="w-4 h-4 transition-transform duration-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path d="M15 19l-7-7 7-7" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>
                    </button>

                    <aside id="inspector-panel" 
                        class="bg-white border-l border-slate-200 shadow-xl transition-all duration-300 ease-in-out overflow-hidden"
                        style="width: 320px;">
                        <div id="inspector-content" class="h-full overflow-y-auto w-[320px] pr-2">
                            </div>
                    </aside>
                </div>
            </div>

        </div>
        `;

        // 2. Initialisation des événements
        this.render();
        // À la fin de mount(container)
        window.addEventListener('keydown', (e) => {
            // Ctrl + B (comme dans VS Code) ou Alt + P
            if ((e.ctrlKey && e.key === 'b') || (e.altKey && e.key === 'p')) {
                e.preventDefault();
                this.toggleInspector();
            }
        });
    },


    toggleInspector() {
        this.state.isInspectorCollapsed = !this.state.isInspectorCollapsed;
        const panel = document.getElementById('inspector-panel');
        const icon = document.getElementById('collapse-icon');
        const btn = document.getElementById('toggle-inspector-btn');

        if (this.state.isInspectorCollapsed) {
            panel.style.width = '5px';
            panel.style.borderLeftWidth = '5px';
            icon.style.transform = 'rotate(180deg)';
            
            // En mode rétracté, on remet le bouton contre le bord
            btn.style.left = '-14px'; 
        } else {
            panel.style.width = '320px';
            panel.style.borderLeftWidth = '1px';
            icon.style.transform = 'rotate(0deg)';
            
            // En mode ouvert, on le décale pour libérer l'accès à l'ascenseur
            btn.style.left = '-10px';//-18
        }
    },
    // --- LOGIQUE DE DRAG & DROP ---
    handleDragStart(e, type) {
        e.dataTransfer.setData('type', type);
        e.dataTransfer.effectAllowed = 'copy';
    },

    handleDrop(e, containerId) {
        e.preventDefault();
        const type = e.dataTransfer.getData('type');
        
        const newEl = {
            id: 'el_' + Math.random().toString(36).substr(2, 9),
            type: type,
            props: this.getDefaultProps(type)
        };

        this.state.elements.push(newEl);
        this.render();
    },

    getDefaultProps(type) {
        switch(type) {
            case 'acte_nature': return { documentType: 'ARRETE' };
            case 'acte_reference': return { refNumber: '2026/001/MND/DC/SGM' };
            case 'article': return { title: 'Article 1er', content: 'Le présent article dispose que...' };
            case 'signataire': return { role: 'Le Ministre', name: 'Atouba Christian LOPEZ', isInterim: false };
            case 'reglementaire': return { vus: [{ id: 1, text: "la Constitution" }] };
            case 'table': return { rows: 2, cols: 2, data: [['', ''], ['', '']] };
            default: return {};
        }
    },
    
    render() {
        const root = document.getElementById('drop-zone-root');
        if (root) {
            root.innerHTML = this.renderToHTML(this.state.elements, this.state.userData);
        }
        // Appel de Lucide si présent pour rafraîchir les icônes
        if (window.lucide) lucide.createIcons();
    },

    printPDF() {
        const canvas = document.getElementById('drop-zone-root');
        if (!canvas) return;

        const format = this.state.paperFormat;
        const dim = this.state.formats[format];
        const printWindow = window.open('', '_blank');

        printWindow.document.write(`
            <html>
                <head>
                    <title>Impression ${format}</title>
                    <script src="https://cdn.tailwindcss.com"></script>
                    <style>
                        @page { size: ${format}; margin: 0; }
                        @media print {
                            body { margin: 0; }
                            .print-area { 
                                width: ${dim.width}; 
                                min-height: ${dim.height}; 
                                padding: 20mm; margin: auto; 
                            }
                        }
                        table { width: 100%; border-collapse: collapse; border: 1px solid black; }
                        td { border: 1px solid black; padding: 5px; }
                    </style>
                </head>
                <body onload="setTimeout(() => { window.print(); window.close(); }, 500)">
                    <div class="print-area font-serif bg-white">${canvas.innerHTML}</div>
                </body>
            </html>
        `);
        printWindow.document.close();
    },
    
    selectElement(id) {
        this.state.selectedId = id;
        this.render(); // Pour mettre à jour la bordure de sélection
        this.updateInspector();
    },
    
    // updateInspector() {
    //     const panel = document.getElementById('inspector-panel');
    //     if (!panel) return;

    //     const el = this.state.elements.find(e => e.id === this.state.selectedId);
    //     if (!el) {
    //         panel.innerHTML = '<div class="text-center py-10 text-slate-400 italic text-sm">Sélectionnez un élément.</div>';
    //         return;
    //     }
    //     let fields = `<h3 class="font-bold text-xs uppercase text-indigo-600 mb-4">Propriétés : ${el.type}</h3>`;
    
    //     if (el.type === 'acte_nature') {
    //         fields += `<input type="text" value="${el.props.documentType}" oninput="window.ActModeler.uDeep('documentType', this.value)" class="w-full p-2 border rounded text-xs font-black uppercase">`;
    //     } 
    //     else if (el.type === 'acte_reference') {
    //         fields += `<input type="text" value="${el.props.refNumber}" oninput="window.ActModeler.uDeep('refNumber', this.value)" class="w-full p-2 border rounded text-xs">`;
    //     }
    //     else if (el.type === 'article') {
    //         fields += `
    //             <input type="text" value="${el.props.title}" oninput="window.ActModeler.uDeep('title', this.value)" class="w-full p-2 border rounded text-xs font-bold mb-2">
    //             <textarea oninput="window.ActModeler.uDeep('content', this.value)" class="w-full p-2 border rounded text-xs w-full" rows="5">${el.props.content}</textarea>`;
    //     }
    //     else if (el.type === 'signataire') {
    //         fields += `
    //             <input type="text" value="${el.props.role}" oninput="window.ActModeler.uDeep('role', this.value)" class="w-full p-2 border rounded text-xs mb-2" placeholder="Rôle">
    //             <input type="text" value="${el.props.name}" oninput="window.ActModeler.uDeep('name', this.value)" class="w-full p-2 border rounded text-xs font-bold" placeholder="Nom">
    //             <div class="mt-2 flex items-center gap-2">
    //                 <input type="checkbox" ${el.props.isInterim ? 'checked' : ''} onchange="window.ActModeler.uDeep('isInterim', this.checked)">
    //                 <span class="text-[10px]">Par intérim (P.I.)</span>
    //             </div>`;
    //     }else if (el.type === 'table') {
    //         // Dans updateInspector(), ajouter ce cas pour le type 'table'
    //         let tableEditor = `
    //             <div class="space-y-4">
    //                 <div class="grid grid-cols-2 gap-2">
    //                     <div>
    //                         <label class="text-[9px] font-bold text-slate-500 uppercase">Lignes</label>
    //                         <input type="number" value="${el.props.rows}" 
    //                             onchange="window.ActModeler.resizeTable('rows', this.value)" 
    //                             class="w-full p-2 border rounded text-xs">
    //                     </div>
    //                     <div>
    //                         <label class="text-[9px] font-bold text-slate-500 uppercase">Colonnes</label>
    //                         <input type="number" value="${el.props.cols}" 
    //                             onchange="window.ActModeler.resizeTable('cols', this.value)" 
    //                             class="w-full p-2 border rounded text-xs">
    //                     </div>
    //                 </div>
                    
    //                 <div class="pt-4 border-t">
    //                     <label class="text-[9px] font-bold text-slate-500 uppercase block mb-2">Contenu des cellules</label>
    //                     <div class="max-h-60 overflow-y-auto border rounded bg-slate-50 p-2">
    //         `;

    //         // Générer un input pour chaque cellule
    //         for (let r = 0; r < el.props.rows; r++) {
    //             tableEditor += `<div class="flex gap-1 mb-1">`;
    //             for (let c = 0; c < el.props.cols; c++) {
    //                 tableEditor += `
    //                     <input type="text" value="${el.props.data[r][c] || ''}" 
    //                         oninput="window.ActModeler.updateTableCell(${r}, ${c}, this.value)"
    //                         placeholder="R${r+1}C${c+1}"
    //                         class="w-full p-1 border rounded text-[10px]">`;
    //             }
    //             tableEditor += `</div>`;
    //         }

    //         tableEditor += `</div></div></div>`;
    //         fields += tableEditor;

    //     }

    //     // 2. Injection du contenu + Boutons Globaux (toujours présents)
    //     contentContainer.innerHTML = fields + this.renderGlobalActions();
    // },
        // Modifier le contenu d'une cellule spécifique
updateInspector() {
    const contentContainer = document.getElementById('inspector-content');
    if (!contentContainer) return;

    // Récupération de l'élément sélectionné
    const el = this.state.elements.find(e => e.id === this.state.selectedId);
    let fields = '';

    // Gestion des zones Globales (Header/Footer)
    if (this.state.selectedId === 'GLOBAL_HEADER') {
        fields = this.renderHeaderInspector(); // On peut isoler pour plus de clarté
    }else if (this.state.selectedId === 'GLOBAL_FOOTER') {
        fields = this.renderFooterInspector();
    } else if (el) {
        fields = `<h3 class="font-bold text-xs uppercase text-indigo-600 mb-4">Propriétés : ${el.type}</h3>`;
        
        switch(el.type) {
            case 'acte_nature':
                fields += `
                    <label class="text-[9px] font-bold text-slate-500 uppercase">Nature de l'acte</label>
                    <input type="text" value="${el.props.documentType}" oninput="window.ActModeler.uDeep('documentType', this.value)" class="w-full p-2 border rounded text-xs font-black uppercase">`;
                break;
            case 'acte_reference':
                fields += `
                    <label class="text-[9px] font-bold text-slate-500 uppercase">Référence</label>
                    <input type="text" value="${el.props.refNumber}" oninput="window.ActModeler.uDeep('refNumber', this.value)" class="w-full p-2 border rounded text-xs">`;
                break;
            case 'article':
                fields += `
                    <label class="text-[9px] font-bold text-slate-500 uppercase">Titre</label>
                    <input type="text" value="${el.props.title}" oninput="window.ActModeler.uDeep('title', this.value)" class="w-full p-2 border rounded text-xs font-bold mb-3">
                    <label class="text-[9px] font-bold text-slate-500 uppercase">Contenu</label>
                    <textarea oninput="window.ActModeler.uDeep('content', this.value)" class="w-full p-2 border rounded text-xs" rows="6">${el.props.content}</textarea>`;
                break;
            case 'signataire':
                fields += `
                    <label class="text-[9px] font-bold text-slate-500 uppercase">Titre/Rôle</label>
                    <input type="text" value="${el.props.role}" oninput="window.ActModeler.uDeep('role', this.value)" class="w-full p-2 border rounded text-xs mb-3">
                    <label class="text-[9px] font-bold text-slate-500 uppercase">Nom complet</label>
                    <input type="text" value="${el.props.name}" oninput="window.ActModeler.uDeep('name', this.value)" class="w-full p-2 border rounded text-xs font-bold">
                    <div class="flex items-center gap-2 mt-3">
                        <input type="checkbox" ${el.props.isInterim ? 'checked' : ''} onchange="window.ActModeler.uDeep('isInterim', this.checked)">
                        <label class="text-xs">Mention P.I. (Par intérim)</label>
                    </div>`;
                break;
            case 'table':
                // ... ton code existant pour le tableau (lignes, colonnes)
                break;
        }
    } else {
        // fields = `<div class="p-10 text-center text-slate-400 italic text-xs">Sélectionnez un élément pour modifier ses propriétés.</div>`;
    if (!el && !['GLOBAL_HEADER', 'GLOBAL_FOOTER'].includes(this.state.selectedId)) {
                // --- CONFIGURATION DU PAPIER (Si rien n'est sélectionné) ---
                fields = `
                    <div class="p-4 space-y-4">
                        <h3 class="font-bold text-xs uppercase text-indigo-600">Document</h3>
                        <div>
                            <label class="text-[9px] font-bold text-slate-500 uppercase">Format du Papier</label>
                            <select onchange="window.ActModeler.updatePaperFormat(this.value)" 
                                    class="w-full p-2 border rounded text-xs bg-white focus:ring-2 focus:ring-indigo-500">
                                ${Object.keys(this.state.formats).map(f => `
                                    <option value="${f}" ${this.state.paperFormat === f ? 'selected' : ''}>
                                        ${this.state.formats[f].label}
                                    </option>
                                `).join('')}
                            </select>
                        </div>
                    </div>`;
            } else {
                // ... (Ici tes codes existants pour l'édition des composants Article, Signataire, etc.)
                fields = `<div class="p-4 text-xs italic text-slate-400">Édition de l'élément : ${el?.type || this.state.selectedId}</div>`;
            }
              
        }
        content.innerHTML = fields + this.renderGlobalActions();  
    // // Injection finale avec les boutons d'action (Aperçu PDF, etc.)
    // contentContainer.innerHTML = `
    //     <div class="flex flex-col h-full">
    //         <div class="flex-1 overflow-y-auto">${fields}</div>
    //         ${this.renderGlobalActions()}
    //     </div>
    // `;
},


        updateTableCell(row, col, value) {
        const el = this.state.elements.find(e => e.id === this.state.selectedId);
        if (el && el.type === 'table') {
            el.props.data[row][col] = value;
            this.render(); // Mise à jour du canvas en temps réel
        }
    },

    // Redimensionner le tableau sans perdre les données existantes
    resizeTable(type, value) {
        const el = this.state.elements.find(e => e.id === this.state.selectedId);
        const newVal = parseInt(value);
        if (!el || newVal < 1) return;

        if (type === 'rows') {
            while (el.props.data.length < newVal) {
                el.props.data.push(new Array(el.props.cols).fill(''));
            }
            el.props.data = el.props.data.slice(0, newVal);
            el.props.rows = newVal;
        } else {
            el.props.data = el.props.data.map(row => {
                while (row.length < newVal) row.push('');
                return row.slice(0, newVal);
            });
            el.props.cols = newVal;
        }
        
        this.render();
        this.updateInspector(); // On rafraîchit l'inspecteur pour voir les nouveaux champs
    },

    uDeep(key, value) {
        const el = this.state.elements.find(e => e.id === this.state.selectedId);
        if (el) {
            el.props[key] = value;
            this.render(); // Rafraîchit le document A4
        }
    },
    renderHeader() {
        const h = this.state.header;
        const isSelected = this.state.selectedId === 'GLOBAL_HEADER';
        return `
            <div onclick="window.ActModeler.selectGlobal('header')" 
                 class="p-4 border-b-2 border-slate-900 mb-8 cursor-pointer transition-all ${isSelected ? 'ring-2 ring-indigo-500 bg-indigo-50' : 'hover:bg-slate-50'}">
                <div class="flex justify-between items-start text-[10px] font-bold">
                    <div class="text-left uppercase">
                        ${h.republique}<br>
                        ----------<br>
                        ${h.ministere}
                    </div>
                    <div class="text-right italic">
                        ${h.devise}
                    </div>
                </div>
            </div>`;
    },

    renderFooter() {
        const f = this.state.footer;
        const isSelected = this.state.selectedId === 'GLOBAL_FOOTER';
        
        // On parse les mots-clés pour permettre l'insertion de la date ou du lieu
        const adresseParsed = this.parseKeywords(f.adresse, this.state.userData);
        const contactParsed = this.parseKeywords(f.contact, this.state.userData);

        return `
            <div onclick="window.ActModeler.selectGlobal('footer')" 
                class="mt-auto pt-4 border-t border-slate-300 text-[10px] text-center text-slate-500 cursor-pointer transition-all 
                ${isSelected ? 'ring-2 ring-indigo-500 bg-indigo-50 rounded' : 'hover:bg-slate-50'}">
                <p class="font-medium">${adresseParsed}</p>
                <p>${contactParsed}</p>
            </div>`;
    },  
    selectGlobal(type) {
        this.state.selectedId = type === 'header' ? 'GLOBAL_HEADER' : 'GLOBAL_FOOTER';
        this.render();
        this.updateInspector();
    },
    // Dans updateInspector()
    updateInspector() {
        const panel = document.getElementById('inspector-panel');
        if (!panel) return;

        // Cas 1 : Header Global
        if (this.state.selectedId === 'GLOBAL_HEADER') {
            panel.innerHTML = `
                <h3 class="font-black text-xs uppercase mb-4 text-indigo-600">Édition de l'En-tête</h3>
                <div class="space-y-3">
                    <div>
                        <label class="text-[9px] font-bold text-slate-500 uppercase">République</label>
                        <input type="text" value="${this.state.header.republique}" 
                            oninput="window.ActModeler.updateGlobal('header', 'republique', this.value)" 
                            class="w-full p-2 border rounded text-xs uppercase font-bold">
                    </div>
                    <div>
                        <label class="text-[9px] font-bold text-slate-500 uppercase">Ministère / Direction</label>
                        <textarea oninput="window.ActModeler.updateGlobal('header', 'ministere', this.value)" 
                                class="w-full p-2 border rounded text-xs" rows="3">${this.state.header.ministere}</textarea>
                    </div>
                    <div>
                        <label class="text-[9px] font-bold text-slate-500 uppercase">Devise nationale</label>
                        <input type="text" value="${this.state.header.devise}" 
                            oninput="window.ActModeler.updateGlobal('header', 'devise', this.value)" 
                            class="w-full p-2 border rounded text-xs italic">
                    </div>
                                        <div class="pt-6 border-t border-slate-100 space-y-2">
                        <button onclick="window.ActModeler.exportModel()" 
                                class="w-full py-2 bg-slate-800 text-white text-[10px] font-bold uppercase rounded hover:bg-slate-900 transition-all">
                            💾 Exporter JSON (Local)
                        </button>
                        <button onclick="window.ActModeler.saveToCloud()" 
                                class="w-full py-2 bg-indigo-600 text-white text-[10px] font-bold uppercase rounded hover:bg-indigo-700 transition-all">
                            ☁️ Sauvegarder (Cloud)
                        </button>
                        <button onclick="window.ActModeler.printPDF()" 
                                class="w-full py-2 bg-red-500 text-white text-[10px] font-bold uppercase rounded hover:bg-red-600 transition-all">
                            📄 Aperçu PDF
                        </button>
                    </div>

                </div>`;
            return; // On s'arrête ici si c'est le header
        }

        // Cas 2 : Footer Global (C'est ce qui manquait)
        if (this.state.selectedId === 'GLOBAL_FOOTER') {
            panel.innerHTML = `
                <h3 class="font-black text-xs uppercase mb-4 text-indigo-600">Édition du Pied de page</h3>
                <div class="space-y-4">
                    <div class="p-2 bg-blue-50 border border-blue-100 rounded text-[9px] text-blue-700">
                        <strong>Astuce :</strong> Utilisez <code>{DATE_ACTUELLE}</code> et <code>{VILLE}</code>.
                    </div>
                    <div>
                        <label class="text-[9px] font-bold text-slate-500 uppercase">Ligne 1</label>
                        <textarea oninput="window.ActModeler.updateGlobal('footer', 'adresse', this.value)" 
                                class="w-full p-2 border rounded text-xs" rows="2">${this.state.footer.adresse}</textarea>
                    </div>
                    <div>
                        <label class="text-[9px] font-bold text-slate-500 uppercase">Ligne 2</label>
                        <textarea oninput="window.ActModeler.updateGlobal('footer', 'contact', this.value)" 
                                class="w-full p-2 border rounded text-xs" rows="2">${this.state.footer.contact}</textarea>
                    </div>

                    <div class="pt-6 border-t border-slate-100 space-y-2">
                        <button onclick="window.ActModeler.exportModel()" 
                                class="w-full py-2 bg-slate-800 text-white text-[10px] font-bold uppercase rounded hover:bg-slate-900 transition-all">
                            💾 Exporter JSON (Local)
                        </button>
                        <button onclick="window.ActModeler.saveToCloud()" 
                                class="w-full py-2 bg-indigo-600 text-white text-[10px] font-bold uppercase rounded hover:bg-indigo-700 transition-all">
                            ☁️ Sauvegarder (Cloud)
                        </button>
                        <button onclick="window.ActModeler.printPDF()" 
                                class="w-full py-2 bg-red-500 text-white text-[10px] font-bold uppercase rounded hover:bg-red-600 transition-all">
                            📄 Aperçu PDF
                        </button>
                    </div>
                </div>`;
            return;
        }

        // Cas 3 : Éléments standards (votre code actuel pour les éléments du canvas)
        const el = this.state.elements.find(e => e.id === this.state.selectedId);
        if (!el) {
            panel.innerHTML = '<div class="text-center py-10 text-slate-400 italic text-sm">Sélectionnez un élément (En-tête, Corps ou Pied de page).</div>';
            return;
        }

        let fields = `<h3 class="font-black text-xs uppercase tracking-tighter text-slate-800 mb-6">Propriétés : ${el.type}</h3>`;

        // Génération dynamique des champs selon le type
        if (el.type === 'admin_acte') {
            fields += `
                <div class="space-y-4">
                    <div>
                        <label class="block text-[10px] font-bold text-slate-500 uppercase">Type d'acte</label>
                        <input type="text" value="${el.props.documentType}" oninput="window.ActModeler.uDeep('documentType', this.value)" 
                               class="w-full p-2 bg-slate-50 border rounded text-xs font-bold uppercase">
                    </div>
                    <div>
                        <label class="block text-[10px] font-bold text-slate-500 uppercase">Mot-clé (DÉCIDE/ARRÊTE)</label>
                        <input type="text" value="${el.props.decideWord}" oninput="window.ActModeler.uDeep('decideWord', this.value)" 
                               class="w-full p-2 bg-slate-50 border rounded text-xs font-bold uppercase">
                    </div>
                    <div>
                        <label class="block text-[10px] font-bold text-slate-500 uppercase">Titre du document</label>
                        <textarea oninput="window.ActModeler.uDeep('documentTitle', this.value)" 
                                  class="w-full p-2 bg-slate-50 border rounded text-xs leading-snug" rows="3">${el.props.documentTitle}</textarea>
                    </div>
                </div>
            `;
        } else if (el.type === 'table') {
            fields += `
                <div class="space-y-4">
                    <div class="grid grid-cols-2 gap-2">
                        <div>
                            <label class="block text-[10px] font-bold text-slate-500 uppercase">Lignes</label>
                            <input type="number" value="${el.props.rows}" onchange="window.ActModeler.uDeep('rows', parseInt(this.value))" class="w-full p-2 border rounded text-xs">
                        </div>
                        <div>
                            <label class="block text-[10px] font-bold text-slate-500 uppercase">Colonnes</label>
                            <input type="number" value="${el.props.cols}" onchange="window.ActModeler.uDeep('cols', parseInt(this.value))" class="w-full p-2 border rounded text-xs">
                        </div>
                    </div>
                    <p class="text-[9px] text-slate-400 italic">Note: Le contenu du tableau se gère via les variables de la base de données.</p>
                </div>
            `;
        }

        panel.innerHTML = fields;
    },
    updateGlobal(section, key, value) {
        this.state[section][key] = value;
        this.render();
    },
    // Export Local (JSON)
    exportModel() {
        const data = {
            header: this.state.header,
            footer: this.state.footer,
            elements: this.state.elements,
            version: "1.0"
        };
        const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = `modele_acte_${Date.now()}.json`;
        link.click();
    },

    // Sauvegarde vers le Backend Go (Microservice)
    async saveToCloud() {
        const token = window.userToken; // Récupéré depuis le Shell/Keycloak
        const data = {
            header: this.state.header,
            footer: this.state.footer,
            elements: this.state.elements
        };

        try {
            const response = await fetch('/api/templates/save', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify(data)
            });

            if (response.ok) {
                alert("✅ Modèle sauvegardé avec succès sur KNB Cloud");
            } else {
                throw new Error("Erreur lors de la sauvegarde");
            }
        } catch (err) {
            console.error(err);
            alert("❌ Échec de la sauvegarde : " + err.message);
        }
    },
    renderGlobalActions() {
        const hasElements = this.state.elements.length > 0;
        const opacity = hasElements ? 'opacity-100' : 'opacity-50 cursor-not-allowed';
        const disabled = hasElements ? '' : 'disabled';
        return `
            <div class="mt-auto pt-6 border-t border-slate-200 space-y-2 bg-slate-50 p-4 sticky bottom-0">
                <p class="text-[9px] font-bold text-slate-400 uppercase mb-2">Actions Globales</p>
                <button onclick="${hasElements ? 'window.ActModeler.exportModel()' : ''}" ${disabled}
                        class="w-full py-2 bg-slate-800 text-white text-[10px] font-bold uppercase rounded hover:bg-slate-900 transition-all ${opacity}">
                    💾 Exporter JSON (Local)
                </button>
                <button onclick="${hasElements ? 'window.ActModeler.saveToCloud()' : ''}" ${disabled}
                        class="w-full py-2 bg-indigo-600 text-white text-[10px] font-bold uppercase rounded hover:bg-indigo-700 transition-all ${opacity}">
                    ☁️ Sauvegarder (Cloud)
                </button>
                <button onclick="${hasElements ? 'window.ActModeler.printPDF()' : ''}" ${disabled}
                        class="w-full py-2 bg-red-500 text-white text-[10px] font-bold uppercase rounded hover:bg-red-600 transition-all ${opacity}">
                    📄 Aperçu PDF
                </button>
                ${!hasElements ? '<p class="text-[8px] text-orange-500 italic text-center">Ajoutez un élément pour activer les actions</p>' : ''}
            </div>
        `;
    },
    handleDrop(e, containerId) {
    e.preventDefault();
    const type = e.dataTransfer.getData('type');
    
    const newEl = {
        id: 'el_' + Math.random().toString(36).substr(2, 9),
        type: type,
        props: this.getDefaultProps(type)
    };

    this.state.elements.push(newEl);
    this.render();
    this.updateInspector(); // Crucial pour activer les boutons immédiatement
} 
};