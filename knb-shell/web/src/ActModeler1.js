export const ActModeler = {
    state: {
        elements: [],
        selectedId: null,
        userData: {
            DATE_ACTUELLE: new Date().toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' }),
            VILLE: "Cotonou"
        },
        header: {
            republique: "RÉPUBLIQUE DU BÉNIN",
            ministere: "Ministère du Numérique et de la Digitalisation",
            devise: "Fraternité - Justice - Travail"
        },
        footer: {
            adresse: "Fait à {VILLE}, le {DATE_ACTUELLE}",
            contact: "contact@numerique.bj"
        },
        isPreview: false,
        isInspectorCollapsed: false
    },

    parseKeywords(text, data = {}) {
        if (typeof text !== 'string') return text;
        return text.replace(/\{(\w+)\}/g, (match, key) => data[key] !== undefined ? data[key] : match);
    },

    // --- LOGIQUE TABLEAUX DYNAMIQUES ---
    updateTableCell(row, col, value) {
        const el = this.state.elements.find(e => e.id === this.state.selectedId);
        if (el && el.type === 'table') {
            el.props.data[row][col] = value;
            this.render();
        }
    },

    resizeTable(type, value) {
        const el = this.state.elements.find(e => e.id === this.state.selectedId);
        const newVal = parseInt(value);
        if (!el || newVal < 1) return;

        if (type === 'rows') {
            while (el.props.data.length < newVal) el.props.data.push(new Array(el.props.cols).fill(''));
            el.props.data = el.props.data.slice(0, newVal);
            el.props.rows = newVal;
        } else {
            el.props.data = el.props.data.map(row => {
                const newRow = [...row];
                while (newRow.length < newVal) newRow.push('');
                return newRow.slice(0, newVal);
            });
            el.props.cols = newVal;
        }
        this.render();
        this.updateInspector();
    },

    // --- RENDU ---
    renderTable(p) {
        let html = `<table class="w-full border-collapse border border-slate-400 my-4 text-sm">`;
        for (let i = 0; i < p.rows; i++) {
            html += `<tr>`;
            for (let j = 0; j < p.cols; j++) {
                const val = this.parseKeywords(p.data[i][j] || "", this.state.userData);
                html += `<td class="border border-slate-400 p-2">${val}</td>`;
            }
            html += `</tr>`;
        }
        return html + `</table>`;
    },
    printPDF() {
        // Cibler le canevas (drop-zone-root)
        const canvas = document.getElementById('drop-zone-root');
        if (!canvas) {
            console.error("Canvas non trouvé");
            return;
        }

        const printWindow = window.open('', '_blank');
        const content = canvas.innerHTML;

        printWindow.document.write(`
            <html>
                <head>
                    <title>KNB Cloud - Impression</title>
                    <script src="https://cdn.tailwindcss.com"></script>
                    <style>
                        @media print { 
                            body { margin: 0; } 
                            .print-area { width: 210mm; min-height: 297mm; padding: 20mm; margin: auto; }
                        }
                        table { width: 100%; border-collapse: collapse; border: 1px solid black; }
                        td, th { border: 1px solid black; padding: 8px; }
                    </style>
                </head>
                <body onload="setTimeout(() => { window.print(); window.close(); }, 500)">
                    <div class="print-area font-serif">${content}</div>
                </body>
            </html>
        `);
        printWindow.document.close();
    },
    mount(container) {
        if (!container) return;
        window.ActModeler = this;

        container.innerHTML = `
        <div class="flex h-full w-full bg-slate-100 overflow-hidden font-sans">
            <aside class="w-16 flex flex-col items-center py-4 bg-[#1e293b] text-white gap-4 shadow-xl z-10">
                <button draggable="true" ondragstart="window.ActModeler.handleDragStart(event, 'admin_acte')" class="p-3 hover:bg-slate-700 rounded-xl">Actes</button>
                <button draggable="true" ondragstart="window.ActModeler.handleDragStart(event, 'table')" class="p-3 hover:bg-slate-700 rounded-xl">Table</button>
            </aside>

            <main class="flex-1 overflow-y-auto p-8 flex justify-center bg-slate-200/50">
                <div id="drop-zone-root" class="w-[210mm] min-h-[297mm] bg-white shadow-2xl p-12 drop-zone" 
                     ondragover="event.preventDefault()" ondrop="window.ActModeler.handleDrop(event, 'root')"></div>
            </main>

            <div class="relative flex h-full">
                <button onclick="window.ActModeler.toggleInspector()" id="toggle-inspector-btn"
                        class="absolute -left-[16px] top-12 z-[100] bg-white border border-slate-200 rounded-full p-1 shadow-lg hover:bg-indigo-50">
                    <svg id="collapse-icon" class="w-4 h-4 transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path d="M15 19l-7-7 7-7" stroke-width="2"/></svg>
                </button>
                <aside id="inspector-panel" class="bg-white border-l border-slate-200 transition-all duration-300 overflow-hidden" style="width: 320px;">
                    <div id="inspector-content" class="h-full overflow-y-auto w-[320px] pr-2"></div>
                </aside>
            </div>
        </div>`;

        // Raccourci clavier Ctrl+B
        window.addEventListener('keydown', (e) => {
            if (e.ctrlKey && e.key === 'b') { e.preventDefault(); this.toggleInspector(); }
        });

        this.render();
    },

    toggleInspector() {
        this.state.isInspectorCollapsed = !this.state.isInspectorCollapsed;
        const panel = document.getElementById('inspector-panel');
        const icon = document.getElementById('collapse-icon');
        panel.style.width = this.state.isInspectorCollapsed ? '0px' : '320px';
        icon.style.transform = this.state.isInspectorCollapsed ? 'rotate(180deg)' : 'rotate(0deg)';
    },

    updateInspector() {
        const content = document.getElementById('inspector-content');
        if (!content) return;
        
        let html = '';
        const el = this.state.elements.find(e => e.id === this.state.selectedId);

        if (el && el.type === 'table') {
            html = `<div class="p-4 space-y-4">
                <h3 class="font-bold text-xs uppercase">Configuration Tableau</h3>
                <div class="flex gap-2">
                    <input type="number" value="${el.props.rows}" onchange="window.ActModeler.resizeTable('rows', this.value)" class="w-full p-1 border text-xs" title="Lignes">
                    <input type="number" value="${el.props.cols}" onchange="window.ActModeler.resizeTable('cols', this.value)" class="w-full p-1 border text-xs" title="Colonnes">
                </div>
                <div class="space-y-1 overflow-x-auto">
                    ${el.props.data.map((row, r) => `
                        <div class="flex gap-1">
                            ${row.map((cell, c) => `
                                <input type="text" value="${cell}" oninput="window.ActModeler.updateTableCell(${r}, ${c}, this.value)" class="w-16 p-1 border text-[10px]">
                            `).join('')}
                        </div>
                    `).join('')}
                </div>
            </div>`;
        } else {
            html = `<div class="p-10 text-center text-slate-400 italic text-xs">Sélectionnez un élément</div>`;
        }

        content.innerHTML = html + this.renderGlobalActions();
    },

    renderGlobalActions() {
        if (this.state.elements.length === 0) return '';
        return `<div class="p-4 border-t mt-auto space-y-2">
            <button onclick="window.ActModeler.printPDF()" class="w-full py-2 bg-red-600 text-white text-[10px] font-bold uppercase rounded">Aperçu PDF</button>
        </div>`;
    },

    render() {
        const root = document.getElementById('drop-zone-root');
        if (root) root.innerHTML = this.renderToHTML(this.state.elements, this.state.userData);
    },

    renderToHTML(elements, userData) {
        return elements.map(el => {
            const isSelected = this.state.selectedId === el.id;
            let inner = el.type === 'table' ? this.renderTable(el.props) : `<div class="p-4">Acte</div>`;
            return `<div onclick="window.ActModeler.selectElement('${el.id}')" class="mb-4 ${isSelected ? 'ring-2 ring-indigo-500' : ''}">${inner}</div>`;
        }).join('');
    },

    selectElement(id) { this.state.selectedId = id; this.render(); this.updateInspector(); },
    handleDragStart(e, type) { e.dataTransfer.setData('type', type); },
    handleDrop(e) {
        e.preventDefault();
        const type = e.dataTransfer.getData('type');
        this.state.elements.push({ 
            id: 'el_' + Math.random().toString(36).substr(2, 9), 
            type, 
            props: this.getDefaultProps(type) 
        });
        this.render();
        this.updateInspector();
    },
    getDefaultProps(type) {
        return type === 'table' ? { rows: 2, cols: 2, data: [['', ''], ['', '']] } : { text: '' };
    }
};