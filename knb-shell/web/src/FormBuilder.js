/**
 * FormBuilder - Composant de création de formulaires et pages web
 * Extrait de demo3.html pour intégration Shell MFE
 */

export const FormBuilder = {
    state: {
        elements: [],
        selectedId: null,
        isPreview: false,
        device: 'desktop' // desktop, tablet, mobile
    },

    // --- 1. CONFIGURATION DES ÉLÉMENTS ---
    getDefaultProps(type) {
        const base = { label: 'Libellé', placeholder: 'Saisir ici...', required: false };
        switch (type) {
            case 'text': return { ...base, type: 'text' };
            case 'textarea': return { ...base, rows: 3 };
            case 'select': return { ...base, options: 'Option 1, Option 2' };
            case 'button': return { label: 'Envoyer', style: 'primary', actions: [] };
            case 'card': return { title: 'Titre de la carte', content: 'Contenu ici...', border: true };
            default: return {};
        }
    },

    // --- 2. RENDU HTML ---
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

    // --- 3. GESTION DE L'INTERFACE ---
    mount(containerId) {
        const container = document.getElementById(containerId);
        if (!container) return;
        window.FormBuilder = this;

        container.innerHTML = `
            <div class="flex h-full bg-slate-50 font-sans overflow-hidden">
                <aside class="w-64 bg-white border-r border-slate-200 flex flex-col shadow-sm">
                    <div class="p-4 border-b font-bold text-slate-800">Composants Web</div>
                    <div class="p-4 grid grid-cols-2 gap-2 overflow-y-auto">
                        <div draggable="true" ondragstart="window.FormBuilder.handleDragStart(event, 'text')" class="p-3 bg-slate-100 rounded-lg text-xs cursor-move hover:bg-indigo-100 text-center">Texte</div>
                        <div draggable="true" ondragstart="window.FormBuilder.handleDragStart(event, 'button')" class="p-3 bg-slate-100 rounded-lg text-xs cursor-move hover:bg-indigo-100 text-center">Bouton</div>
                        <div draggable="true" ondragstart="window.FormBuilder.handleDragStart(event, 'card')" class="p-3 bg-slate-100 rounded-lg text-xs cursor-move hover:bg-indigo-100 text-center">Carte</div>
                    </div>
                </aside>

                <main class="flex-1 p-8 overflow-y-auto canvas-grid flex flex-col items-center">
                    <div id="fb-canvas" class="bg-white shadow-xl min-h-[600px] transition-all duration-300 p-8 rounded-2xl w-full max-w-2xl drop-zone"
                         ondragover="event.preventDefault()" ondrop="window.FormBuilder.handleDrop(event)">
                    </div>
                </main>

                <aside id="fb-inspector" class="w-80 bg-white border-l border-slate-200 p-4 shadow-sm overflow-y-auto">
                    <div id="fb-inspector-content">
                        <p class="text-center text-slate-400 mt-20 italic text-sm">Sélectionnez un composant</p>
                    </div>
                </aside>
            </div>
        `;
        this.render();
    },

    render() {
        const canvas = document.getElementById('fb-canvas');
        if (canvas) canvas.innerHTML = this.renderToHTML(this.state.elements);
    },

    // --- 4. ACTIONS ---
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
        this.render();
        this.updateInspector();
    },

    deleteElement(id) {
        this.state.elements = this.state.elements.filter(el => el.id !== id);
        this.state.selectedId = null;
        this.render();
        this.updateInspector();
    },

    updateInspector() {
        const panel = document.getElementById('fb-inspector-content');
        const el = this.state.elements.find(e => e.id === this.state.selectedId);
        if (!el) return;

        panel.innerHTML = `
            <h3 class="font-bold text-indigo-600 uppercase text-xs mb-4">Édition : ${el.type}</h3>
            <div class="space-y-4">
                <div>
                    <label class="text-[10px] font-bold text-gray-500 uppercase">Libellé</label>
                    <input type="text" value="${el.props.label || el.props.title || ''}" 
                           oninput="window.FormBuilder.updateProp('${el.type === 'card' ? 'title' : 'label'}', this.value)" 
                           class="w-full p-2 border rounded mt-1 text-sm">
                </div>
            </div>
        `;
    },

    updateProp(key, value) {
        const el = this.state.elements.find(e => e.id === this.state.selectedId);
        if (el) {
            el.props[key] = value;
            this.render();
        }
    }
};